package attest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/afelin/curbpack/internal/gitutil"
	"github.com/afelin/curbpack/internal/paths"
	"github.com/afelin/curbpack/internal/tty"
)

// Capsule is the Git Notes compliance capsule (Merkle + OCC).
type Capsule struct {
	SchemaVersion   string            `json:"schema_version"`
	Timestamp       string            `json:"timestamp"`
	CommitSHA       string            `json:"commit_sha"`
	StateHash       string            `json:"state_hash"`
	ParentStateHash string            `json:"parent_state_hash,omitempty"`
	OCCParent       string            `json:"expected_parent_commit_sha"`
	Signer          string            `json:"signer"`
	SSHSignature    string            `json:"ssh_signature,omitempty"`
	UserTouch       string            `json:"user_touch"`
	HPURLFragment   string            `json:"hpurl_fragment"`
	Evidence        map[string]string `json:"evidence,omitempty"`
}

// Options for attest.
type Options struct {
	RepoRoot   string
	AllowDirty bool
	// Optional digests to bind (CycloneDX / VEX). Empty strings omitted.
	SBOMDigest string
	VEXDigest  string
}

// ComputeStateHash returns sha256 hex of the length-prefixed field stream.
// Sole authority for capsule state_hash. Field order is frozen: commit,
// parentHash, sbomDigest, vexDigest. No wall-clock / UnixNano.
// Length-prefixing avoids pipe-delimiter ambiguity across field boundaries.
func ComputeStateHash(commit, parentHash, sbomDigest, vexDigest string) string {
	h := sha256.New()
	writeLenPrefixed(h, commit)
	writeLenPrefixed(h, parentHash)
	writeLenPrefixed(h, sbomDigest)
	writeLenPrefixed(h, vexDigest)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// writeLenPrefixed streams a netstring-style "%d:" + bytes field into h.
func writeLenPrefixed(h hash.Hash, s string) {
	_, _ = fmt.Fprintf(h, "%d:", len(s))
	_, _ = io.WriteString(h, s)
}

// Run creates a Git Notes capsule with Merkle parent link and best-effort SSH-agent sign.
func Run(opts Options) (Capsule, error) {
	root := opts.RepoRoot
	var err error
	if root == "" {
		root, err = gitutil.RepoRoot("")
		if err != nil {
			return Capsule{}, err
		}
	}

	if !opts.AllowDirty && gitutil.IsDirty(root) {
		return Capsule{}, fmt.Errorf("OCC conflict: working directory has uncommitted files")
	}

	commit, err := gitutil.HeadSHA(root)
	if err != nil {
		return Capsule{}, err
	}

	sbomDigest := opts.SBOMDigest
	vexDigest := opts.VEXDigest
	sbomRel := filepath.ToSlash(filepath.Join(paths.EvidenceRel, "sbom.cdx.json"))
	vexRel := filepath.ToSlash(filepath.Join(paths.EvidenceRel, "vex-pending.json"))
	if sbomDigest == "" {
		sbomDigest, sbomRel = digestEvidence(root, "sbom.cdx.json")
	}
	if vexDigest == "" {
		vexDigest, vexRel = digestEvidence(root, "vex-pending.json")
	}

	parentHash := gitutil.ParentNoteHash(root, commit)
	stateHash := ComputeStateHash(commit, parentHash, sbomDigest, vexDigest)

	signer := "local-unsigned"
	userTouch := "not-verified"
	sshSig := ""

	if sig, who, verified := trySSHAgentSign(root, stateHash); verified {
		sshSig = sig
		signer = who
		userTouch = "ssh-agent-signed"
	} else if who != "" {
		// Agent present but sign failed — honest unsigned, never fake "verified".
		signer = "local-unsigned"
		userTouch = "not-verified"
		sshSig = ""
		tty.PrintStatus("SSH-agent attest", false, "agent present but sign failed — unsigned capsule (not verified)")
	} else {
		tty.PrintStatus("SSH-agent attest", false, "no agent / sign failed — writing unsigned capsule")
	}

	evidence := map[string]string{}
	if sbomDigest != "" {
		evidence["sbom_digest"] = sbomDigest
		evidence["sbom_path"] = sbomRel
	}
	if vexDigest != "" {
		evidence["vex_digest"] = vexDigest
		evidence["vex_path"] = vexRel
	}
	evidence["local_pointer"] = paths.EvidenceRel + "/"

	cap := Capsule{
		SchemaVersion:   "v3.33-OCC",
		Timestamp:       time.Now().UTC().Format(time.RFC3339), // display-only; not in state_hash
		CommitSHA:       commit,
		StateHash:       stateHash,
		ParentStateHash: parentHash,
		OCCParent:       commit,
		Signer:          signer,
		SSHSignature:    sshSig,
		UserTouch:       userTouch,
		HPURLFragment:   fmt.Sprintf("#?h=%s&p=%s&s=%s", url.QueryEscape(stateHash), url.QueryEscape(commit), url.QueryEscape(truncate(sshSig, 32))),
		Evidence:        evidence,
	}

	body, err := json.MarshalIndent(cap, "", "  ")
	if err != nil {
		return Capsule{}, err
	}
	if err := gitutil.NotesAdd(root, commit, string(body)); err != nil {
		return Capsule{}, fmt.Errorf("git notes write: %w", err)
	}

	// Local evidence pointer for HPURL verify (write-new curbpack path).
	evidenceDir := paths.EvidenceDir(root)
	_ = os.MkdirAll(evidenceDir, 0o755)
	pointer := map[string]any{
		"state_hash":    stateHash,
		"commit_sha":    commit,
		"hpurl":         cap.HPURLFragment,
		"sbom_digest":   sbomDigest,
		"vex_digest":    vexDigest,
		"note":          "Client-side HPURL verify compares fragment h= to state_hash. Not a certification.",
		"evidence_root": paths.EvidenceRel + "/",
	}
	pb, _ := json.MarshalIndent(pointer, "", "  ")
	_ = os.WriteFile(filepath.Join(evidenceDir, "hpurl-pointer.json"), append(pb, '\n'), 0o644)

	tty.PrintStatus("Git Notes capsule", true, "refs/notes/curbpack @ "+truncate(commit, 12))
	tty.PrintStatus("HPURL fragment", true, cap.HPURLFragment)
	tty.PrintStatus("Evidence pointer", true, paths.EvidenceRel+"/hpurl-pointer.json")
	return cap, nil
}

// digestEvidence dual-reads new then legacy evidence files and returns sha256 hex
// plus the slash-relative path that was hashed (empty digest → default write-new rel).
func digestEvidence(root, name string) (digest, rel string) {
	abs := paths.ResolveUnderEvidence(root, name)
	rel = filepath.ToSlash(filepath.Join(paths.EvidenceRel, name))
	legacyAbs := filepath.Join(root, filepath.FromSlash(paths.LegacyEvidenceRel), name)
	if filepath.Clean(abs) == filepath.Clean(legacyAbs) {
		rel = filepath.ToSlash(filepath.Join(paths.LegacyEvidenceRel, name))
	}
	return fileDigest(abs), rel
}

func fileDigest(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func truncate(s string, n int) string {
	if len(s) <= n || n <= 0 {
		if s == "" {
			return "unsigned"
		}
		return s
	}
	return s[:n]
}

// HPURLParts are the fragment query fields used by proof/index.html verify.
type HPURLParts struct {
	StateHash string
	Commit    string
	SigHint   string
}

// ParseHPURLFragment parses an HPURL fragment per spec §3–4 shape:
// take the substring after the first '?' in the fragment, then query-parse.
// Accepts "#?h=&p=&s=", "#label?h=…", or bare "?h=…" / "h=…".
// Keys: h (aliases hash, state_hash), p (aliases commit, parent), s (aliases sig, signature).
// Returns ok=false on malformed input; never panics. Does not implement Ed25519 $/@/! product attest.
func ParseHPURLFragment(frag string) (HPURLParts, bool) {
	frag = strings.TrimSpace(frag)
	if frag == "" {
		return HPURLParts{}, false
	}
	frag = strings.TrimPrefix(frag, "#")
	query := frag
	if i := strings.IndexByte(frag, '?'); i >= 0 {
		query = frag[i+1:]
	} else if !strings.Contains(frag, "=") {
		return HPURLParts{}, false
	}
	parts := HPURLParts{}
	for _, kv := range strings.Split(query, "&") {
		if kv == "" {
			continue
		}
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		if u, err := url.QueryUnescape(v); err == nil {
			v = u
		}
		switch k {
		case "h", "hash", "state_hash":
			parts.StateHash = v
		case "p", "commit", "parent":
			parts.Commit = v
		case "s", "sig", "signature":
			parts.SigHint = v
		}
	}
	if parts.StateHash == "" {
		return parts, false
	}
	return parts, true
}

// command builds an *exec.Cmd (overridable in tests via fake PATH tooling).
var command = exec.Command

// trySSHAgentSign signs via ssh-keygen -Y when SSH_AUTH_SOCK is set.
// Returns verified=true only when a real signature file was produced.
// agent-bind placeholders are never treated as verified signatures.
//
// OpenSSH expects -f to be a key file (public key matching an agent-held private
// key is enough). Passing the payload path as -f always fails.
func trySSHAgentSign(repoRoot, payload string) (sig string, identity string, verified bool) {
	if strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")) == "" {
		return "", "", false
	}
	list := command("ssh-add", "-L")
	out, err := list.Output()
	if err != nil || len(out) == 0 {
		return "", "", false
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	first := lines[0]
	who := identityFromSSHAddLine(first)

	tmpPub, err := os.CreateTemp("", "curbpack-attest-*.pub")
	if err != nil {
		return "", who, false
	}
	defer os.Remove(tmpPub.Name())
	if _, err := tmpPub.WriteString(first + "\n"); err != nil {
		_ = tmpPub.Close()
		return "", who, false
	}
	_ = tmpPub.Close()

	tmpIn, err := os.CreateTemp("", "curbpack-attest-*.txt")
	if err != nil {
		return "", who, false
	}
	defer os.Remove(tmpIn.Name())
	if _, err := tmpIn.WriteString(payload); err != nil {
		_ = tmpIn.Close()
		return "", who, false
	}
	_ = tmpIn.Close()

	tmpOut := tmpIn.Name() + ".sig"
	defer os.Remove(tmpOut)

	// -f = public key (agent resolves private); final arg = data to sign.
	cmd := command("ssh-keygen", "-Y", "sign", "-f", tmpPub.Name(), "-n", "curbpack@attest", tmpIn.Name())
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return "", who, false
	}
	sigBytes, err := os.ReadFile(tmpOut)
	if err != nil || len(sigBytes) == 0 {
		return "", who, false
	}
	sigStr := strings.TrimSpace(string(sigBytes))
	if strings.HasPrefix(sigStr, "agent-bind:") {
		// Never treat synthetic bind tokens as cryptographic signatures.
		return "", who, false
	}
	return sigStr, who, true
}

// identityFromSSHAddLine builds a human-readable SSH-agent identity from one
// ssh-add -L line ("type key [comment…]"). Multi-word comments are preserved;
// comment-free keys (type + material only) use unnamed-key.
func identityFromSSHAddLine(line string) string {
	parts := strings.Fields(strings.TrimSpace(line))
	switch {
	case len(parts) >= 3:
		return "SSH-Agent:" + strings.Join(parts[2:], " ")
	case len(parts) == 2:
		return "SSH-Agent:unnamed-key"
	default:
		return "SSH-Agent"
	}
}

// View prints the capsule for HEAD.
func View(repoRoot string) error {
	if repoRoot == "" {
		var err error
		repoRoot, err = gitutil.RepoRoot("")
		if err != nil {
			return err
		}
	}
	commit, _ := gitutil.HeadSHA(repoRoot)
	body, err := gitutil.NotesShow(repoRoot, commit)
	if err != nil {
		fmt.Printf("%s\n", tty.C(tty.Yellow, "No verified compliance records for commit: "+commit))
		return nil
	}
	var cap Capsule
	if json.Unmarshal([]byte(body), &cap) != nil {
		fmt.Println(body)
		return nil
	}
	fmt.Printf("%s\n", tty.C(tty.Bold+tty.Green, "COMPLIANCE CAPSULE (Git Notes)"))
	fmt.Println("====================================================================")
	fmt.Printf("  Signer:          %s\n", cap.Signer)
	fmt.Printf("  User presence:   %s\n", cap.UserTouch)
	if cap.UserTouch != "ssh-agent-signed" || cap.SSHSignature == "" || strings.HasPrefix(cap.SSHSignature, "agent-bind:") {
		fmt.Printf("  Signature:       %s\n", tty.C(tty.Yellow, "UNSIGNED — not cryptographically verified"))
	} else {
		fmt.Printf("  Signature:       %s\n", tty.C(tty.Yellow, "ssh-agent signature blob present (not independently verified)"))
	}
	fmt.Printf("  Commit bound:    %s\n", cap.CommitSHA)
	fmt.Printf("  State hash:      %s\n", cap.StateHash)
	fmt.Printf("  Parent hash:     %s\n", cap.ParentStateHash)
	fmt.Printf("  HPURL fragment:  %s\n", cap.HPURLFragment)
	if len(cap.Evidence) > 0 {
		fmt.Printf("  Evidence:        %v\n", cap.Evidence)
	}
	fmt.Println("====================================================================")
	fmt.Println(tty.C(tty.Dim, "Not a certification — evidence for human review. Unsigned ≠ verified."))
	return nil
}
