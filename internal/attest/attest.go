package attest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/afelin/cyberready/internal/gitutil"
	"github.com/afelin/cyberready/internal/tty"
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

// StateSeed builds the reproducible hash input (no wall-clock / UnixNano).
func StateSeed(commit, parentHash, sbomDigest, vexDigest string) string {
	return fmt.Sprintf("%s|%s|sbom=%s|vex=%s", commit, parentHash, sbomDigest, vexDigest)
}

// ComputeStateHash returns sha256 hex of the reproducible seed.
func ComputeStateHash(commit, parentHash, sbomDigest, vexDigest string) string {
	seed := StateSeed(commit, parentHash, sbomDigest, vexDigest)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(seed)))
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
	if sbomDigest == "" {
		sbomDigest = fileDigest(filepath.Join(root, ".github", "cyberready", "evidence", "sbom.cdx.json"))
	}
	if vexDigest == "" {
		vexDigest = fileDigest(filepath.Join(root, ".github", "cyberready", "evidence", "vex-pending.json"))
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
		evidence["sbom_path"] = ".github/cyberready/evidence/sbom.cdx.json"
	}
	if vexDigest != "" {
		evidence["vex_digest"] = vexDigest
		evidence["vex_path"] = ".github/cyberready/evidence/vex-pending.json"
	}
	evidence["local_pointer"] = ".github/cyberready/evidence/"

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
		HPURLFragment:   fmt.Sprintf("#?h=%s&p=%s&s=%s", stateHash, commit, truncate(sshSig, 32)),
		Evidence:        evidence,
	}

	body, err := json.MarshalIndent(cap, "", "  ")
	if err != nil {
		return Capsule{}, err
	}
	if err := gitutil.NotesAdd(root, commit, string(body)); err != nil {
		return Capsule{}, fmt.Errorf("git notes write: %w", err)
	}

	// Local evidence pointer for HPURL verify
	_ = os.MkdirAll(filepath.Join(root, ".github", "cyberready", "evidence"), 0o755)
	pointer := map[string]any{
		"state_hash":    stateHash,
		"commit_sha":    commit,
		"hpurl":         cap.HPURLFragment,
		"sbom_digest":   sbomDigest,
		"vex_digest":    vexDigest,
		"note":          "Client-side HPURL verify compares fragment h= to state_hash. Not a certification.",
		"evidence_root": ".github/cyberready/evidence/",
	}
	pb, _ := json.MarshalIndent(pointer, "", "  ")
	_ = os.WriteFile(filepath.Join(root, ".github", "cyberready", "evidence", "hpurl-pointer.json"), append(pb, '\n'), 0o644)

	tty.PrintStatus("Git Notes capsule", true, "refs/notes/cyberready @ "+truncate(commit, 12))
	tty.PrintStatus("HPURL fragment", true, cap.HPURLFragment)
	tty.PrintStatus("Evidence pointer", true, ".github/cyberready/evidence/hpurl-pointer.json")
	return cap, nil
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

// ParseHPURLFragment parses "#?h=&p=&s=" (also accepts without leading #).
// Returns ok=false on malformed input; never panics.
func ParseHPURLFragment(frag string) (HPURLParts, bool) {
	frag = strings.TrimSpace(frag)
	if frag == "" {
		return HPURLParts{}, false
	}
	frag = strings.TrimPrefix(frag, "#")
	frag = strings.TrimPrefix(frag, "?")
	parts := HPURLParts{}
	for _, kv := range strings.Split(frag, "&") {
		if kv == "" {
			continue
		}
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		switch k {
		case "h":
			parts.StateHash = v
		case "p":
			parts.Commit = v
		case "s":
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
	parts := strings.Fields(first)
	who := "SSH-Agent"
	if len(parts) >= 3 {
		who = "SSH-Agent:" + parts[len(parts)-1]
	}

	tmpPub, err := os.CreateTemp("", "cyberready-attest-*.pub")
	if err != nil {
		return "", who, false
	}
	defer os.Remove(tmpPub.Name())
	if _, err := tmpPub.WriteString(first + "\n"); err != nil {
		_ = tmpPub.Close()
		return "", who, false
	}
	_ = tmpPub.Close()

	tmpIn, err := os.CreateTemp("", "cyberready-attest-*.txt")
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
	cmd := command("ssh-keygen", "-Y", "sign", "-f", tmpPub.Name(), "-n", "cyberready@attest", tmpIn.Name())
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
