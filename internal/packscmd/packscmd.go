package packscmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/afelin/cyberready/internal/config"
	"github.com/afelin/cyberready/internal/gitutil"
	"github.com/afelin/cyberready/internal/packs"
	"github.com/afelin/cyberready/internal/tty"
)

// List prints embedded packs and watchlist summary.
func List() error {
	all, err := packs.LoadEmbedded()
	if err != nil {
		return err
	}
	fmt.Println(tty.C(tty.Bold+tty.Cyan, "Embedded regulation packs"))
	for _, p := range all {
		fmt.Printf("  %-22s %s  (%d rules) — %s\n", p.ID, p.Version, len(p.Rules), p.Name)
	}
	wl, err := packs.LoadWatchlist()
	if err != nil {
		return err
	}
	fmt.Printf("\nWatchlist v%d updated %s — %d informational entries\n", wl.Version, wl.Updated, len(wl.Entries))
	fmt.Println(tty.C(tty.Dim, wl.Note))
	return nil
}

// UpdateStub documents / optionally fetches a pack update channel.
// Network fetch is disabled unless BOTH CYBERREADY_PACKS_URL and
// CYBERREADY_PACKS_SHA256 (hex) are set — fail-closed integrity pin.
func UpdateStub() error {
	url := strings.TrimSpace(os.Getenv("CYBERREADY_PACKS_URL"))
	pin := strings.ToLower(strings.TrimSpace(os.Getenv("CYBERREADY_PACKS_SHA256")))
	dest := strings.TrimSpace(os.Getenv("CYBERREADY_PACKS_DIR"))
	if dest == "" {
		dest = filepath.Join(".github", "cyberready", "packs")
	}
	if url == "" {
		fmt.Println(`packs update

CyberReady embeds packs in the binary. Network pack updates are OFF by default.

  1. Air-gap import (preferred):
       cyberready packs import ./path/to/packs-bundle

  2. Or set CYBERREADY_PACKS_DIR to a directory containing:
       cra-baseline/pack.json
       medtech-iec62304/pack.json
       house-policy/pack.json
       _watchlist.json

  3. Optional online channel (requires integrity pin):
       CYBERREADY_PACKS_URL=https://… \
       CYBERREADY_PACKS_SHA256=<sha256-hex> \
       cyberready packs update

Without CYBERREADY_PACKS_SHA256, network update is refused.
Watchlist refreshes are informational only and never fail validate.`)
		return nil
	}
	if pin == "" || len(pin) != 64 {
		return fmt.Errorf("CYBERREADY_PACKS_URL set but CYBERREADY_PACKS_SHA256 missing or not 64 hex chars — refusing network update (fail closed)")
	}
	if _, err := hex.DecodeString(pin); err != nil {
		return fmt.Errorf("CYBERREADY_PACKS_SHA256 is not valid hex: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("packs update fetch failed (offline?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("packs update HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20)) // 32 MiB cap
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != pin {
		return fmt.Errorf("packs update checksum mismatch: expected %s got %s — refusing write", pin, actual)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	out := filepath.Join(dest, "bundle.json")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return err
	}
	tty.PrintStatus("Packs update", true, "sha256 ok → wrote "+out+" — extract pack.json files manually or use packs import")
	return nil
}

// ImportAirGap copies pack.json files from a local directory into CYBERREADY_PACKS_DIR / dest.
// Import-only honesty: ValidatePack, require assurance_class, refuse claim-adjacent theater copy.
func ImportAirGap(src string) error {
	if src == "" {
		return fmt.Errorf("usage: cyberready packs import <directory>")
	}
	dest := strings.TrimSpace(os.Getenv("CYBERREADY_PACKS_DIR"))
	if dest == "" {
		dest = filepath.Join(".github", "cyberready", "packs")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	copied := 0
	for _, e := range entries {
		name := e.Name()
		if name == "_watchlist.json" {
			in, err := os.ReadFile(filepath.Join(src, name))
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dest, name), in, 0o644); err != nil {
				return err
			}
			copied++
			continue
		}
		if !e.IsDir() {
			continue
		}
		inPath := filepath.Join(src, name, "pack.json")
		data, err := os.ReadFile(inPath)
		if err != nil {
			continue
		}
		var p packs.Pack
		if err := json.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("packs import %s: invalid pack.json: %w", name, err)
		}
		if err := packs.ValidatePack(p); err != nil {
			return fmt.Errorf("packs import %s: %w", name, err)
		}
		if strings.TrimSpace(p.AssuranceClass) == "" {
			return fmt.Errorf("packs import %s: assurance_class required (import honesty)", name)
		}
		if hit := claimAdjacentHit(p.Name, p.Description); hit != "" {
			return fmt.Errorf("packs import %s: claim-adjacent %s refused", name, hit)
		}
		outDir := filepath.Join(dest, name)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outDir, "pack.json"), data, 0o644); err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		digest := hex.EncodeToString(sum[:])
		if err := os.WriteFile(filepath.Join(outDir, ".cyberready-pack.sha256"), []byte(digest+"\n"), 0o644); err != nil {
			return err
		}
		copied++
	}
	tty.PrintStatus("Air-gap import", true, fmt.Sprintf("%d items → %s", copied, dest))
	fmt.Println("Set CYBERREADY_PACKS_DIR=" + dest + " to use imported packs.")
	return nil
}

// claimAdjacentHit returns a short label when name/description looks like certification theater.
// Aligned with claim-safety DENY spirit; import-only fail-closed.
func claimAdjacentHit(name, description string) string {
	blob := strings.ToLower(strings.TrimSpace(name) + " " + strings.TrimSpace(description))
	needles := []string{
		"we are certified",
		"product is certified",
		"officially certified",
		"cyberready certifies",
		"notified-body approved",
		"notified body approved",
		"conformity assessment complete",
		"conformity assessment passed",
		"ce marking issued",
		"is ce-marked",
		"has been ce-marked",
		"we are cra compliant",
		"cra compliant",
		"certified conformity",
	}
	for _, n := range needles {
		if strings.Contains(blob, n) {
			return "copy (" + n + ")"
		}
	}
	return ""
}

// ExportGraph writes .github/cyberready/graph/policy-graph.json for active packs.
func ExportGraph(args []string) error {
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return fmt.Errorf("export-graph requires a git repository: %w", err)
	}
	packIDs := []string{}
	out := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--packs":
			if i+1 < len(args) {
				packIDs = append(packIDs, strings.Split(args[i+1], ",")...)
				i++
			}
		case "--out":
			if i+1 < len(args) {
				out = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") && out == "" {
				out = args[i]
			}
		}
	}
	if len(packIDs) == 0 {
		if cfg, err := config.Load(root); err == nil && cfg != nil && len(cfg.Packs) > 0 {
			packIDs = cfg.Packs
		} else {
			packIDs = []string{"house-policy"}
		}
	}
	path, err := packs.ExportPolicyGraph(root, packIDs, out)
	if err != nil {
		return err
	}
	tty.PrintStatus("policy-graph", true, path)
	fmt.Println(tty.C(tty.Dim, "Local RKG export — not a conformity assessment."))
	return nil
}

// Doctor reports expired/superseded/pin-skew pack issues.
func Doctor() error {
	f, err := packs.DoctorPacks()
	if err != nil {
		return err
	}
	fmt.Println(tty.C(tty.Bold+tty.Cyan, "Packs doctor (validity / supersession / pin skew)"))
	if len(f.Expired) == 0 && len(f.Superseded) == 0 && len(f.PinSkew) == 0 && len(f.UnknownBase) == 0 {
		tty.PrintStatus("packs doctor", true, "no expired/superseded/pin-skew issues")
		return nil
	}
	for _, e := range f.Expired {
		tty.PrintStatus("expired", false, e)
	}
	for _, e := range f.Superseded {
		tty.PrintStatus("superseded", false, e)
	}
	for _, e := range f.PinSkew {
		tty.PrintStatus("pin skew", false, e)
	}
	for _, e := range f.UnknownBase {
		tty.PrintStatus("extends", false, e)
	}
	fmt.Println(tty.C(tty.Dim, "Refresh via checksummed packs update/import — no unpinned law crawl."))
	return nil
}
