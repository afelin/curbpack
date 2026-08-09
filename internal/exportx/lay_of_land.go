package exportx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afelin/cyberready/internal/instrument"
	"github.com/afelin/cyberready/internal/packs"
	"github.com/afelin/cyberready/internal/sbom"
)

const layOfLandCovenant = "Instrument panel · not a security program · not conformity assessment"

// LayOfLandReport is the shareable map export (Markdown + JSON).
type LayOfLandReport struct {
	SchemaVersion string           `json:"schema_version"`
	Covenant      string           `json:"covenant"`
	Note          string           `json:"note"`
	DepsCount     int              `json:"deps_count"`
	DepsFP        string           `json:"deps_fp"`
	DepsSample    []instrument.Dep `json:"deps_sample,omitempty"`
	SecretHits    int              `json:"secret_hits"`
	WatchlistHits int              `json:"watchlist_hits"`
	WatchlistNote string           `json:"watchlist_note"`
	BuyerPointer  string           `json:"buyer_pointer"`
}

// WriteLayOfLand emits lay-of-land.md + .json (covenant, deps, secrets, watchlist∩SBOM).
func WriteLayOfLand(root, outPath string) (string, error) {
	snap := instrument.Compute(root)
	wlHits, wlNote := informationalWatchlistCount(root)
	sample := snap.Deps
	if len(sample) > 40 {
		sample = sample[:40]
	}
	report := LayOfLandReport{
		SchemaVersion: "1",
		Covenant:      layOfLandCovenant,
		Note:          "Shareable instrument map for humans. Not a vulnerability determination. Not CE / not notified-body.",
		DepsCount:     len(snap.Deps),
		DepsFP:        snap.DepsFP,
		DepsSample:    sample,
		SecretHits:    snap.SecretHits,
		WatchlistHits: wlHits,
		WatchlistNote: wlNote,
		BuyerPointer:  "cyberready export --buyer-questions",
	}

	mdPath, jsonPath := layOfLandPaths(root, outPath)
	if err := os.MkdirAll(filepath.Dir(mdPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(mdPath, []byte(formatLayOfLandMarkdown(report)), 0o644); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(jsonPath, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	// Also refresh informational watchlist join beside the map when possible.
	_, _ = WriteWatchlistJoin(root, "")
	return mdPath, nil
}

func layOfLandPaths(root, outPath string) (mdPath, jsonPath string) {
	if outPath == "" {
		base := filepath.Join(root, ".github", "cyberready", "cache", "lay-of-land")
		return base + ".md", base + ".json"
	}
	ext := strings.ToLower(filepath.Ext(outPath))
	stem := strings.TrimSuffix(outPath, ext)
	switch ext {
	case ".md", ".markdown":
		return outPath, stem + ".json"
	case ".json":
		return stem + ".md", outPath
	default:
		return outPath + ".md", outPath + ".json"
	}
}

func formatLayOfLandMarkdown(r LayOfLandReport) string {
	var b strings.Builder
	b.WriteString("# Lay of the land (instrument panel)\n\n")
	fmt.Fprintf(&b, "> %s\n\n", r.Covenant)
	b.WriteString(r.Note + "\n\n")
	fmt.Fprintf(&b, "- **Dependencies:** %d (fp `%s`)\n", r.DepsCount, r.DepsFP)
	fmt.Fprintf(&b, "- **Secret-hit count (high-signal scan):** %d\n", r.SecretHits)
	fmt.Fprintf(&b, "- **Watchlist∩SBOM (informational):** %d — %s\n", r.WatchlistHits, r.WatchlistNote)
	fmt.Fprintf(&b, "- **Buyer checklist:** `%s`\n\n", r.BuyerPointer)
	if len(r.DepsSample) > 0 {
		b.WriteString("## Dependencies (sample)\n\n")
		b.WriteString("| eco | name | version |\n|---|---|---|\n")
		for _, d := range r.DepsSample {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", mdCell(d.Eco), mdCell(d.Name), mdCell(d.Version))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func informationalWatchlistCount(root string) (int, string) {
	wl, err := packs.LoadWatchlist()
	if err != nil {
		return 0, "watchlist unavailable"
	}
	pkgs, _, err := sbom.CollectPackages(root)
	if err != nil {
		if sbom.IsUnavailable(err) {
			return 0, "no supported lockfile/manifest for join"
		}
		return 0, "SBOM collect error"
	}
	n := 0
	for _, e := range wl.Entries {
		for _, p := range pkgs {
			if !packageMatch(e, p) {
				continue
			}
			if len(e.Versions) > 0 && !versionMatch(e.Versions, p.Version) {
				continue
			}
			n++
		}
	}
	return n, "Informational advisory rows for human review — not a CVE product"
}
