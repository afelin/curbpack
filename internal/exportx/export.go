package exportx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/afelin/cyberready/internal/ir"
	"github.com/afelin/cyberready/internal/packs"
	"github.com/afelin/cyberready/internal/sbom"
	"github.com/afelin/cyberready/internal/validate"
)

// SARIFDocument is SARIF 2.1.0 subset.
type SARIFDocument struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SARIFRun `json:"runs"`
}

type SARIFRun struct {
	Tool        SARIFTool         `json:"tool"`
	Results     []SARIFResult     `json:"results"`
	Invocations []SARIFInvocation `json:"invocations,omitempty"`
}

type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

type SARIFDriver struct {
	Name            string       `json:"name"`
	InformationURI  string       `json:"informationUri,omitempty"`
	Rules           []SARIFRule  `json:"rules"`
}

type SARIFRule struct {
	ID               string `json:"id"`
	ShortDescription struct {
		Text string `json:"text"`
	} `json:"shortDescription"`
}

type SARIFResult struct {
	RuleID     string          `json:"ruleId"`
	Level      string          `json:"level"`
	Message    SARIFMessage    `json:"message"`
	Locations  []SARIFLocation `json:"locations,omitempty"`
	Properties map[string]any  `json:"properties,omitempty"`
}

type SARIFMessage struct {
	Text string `json:"text"`
}

type SARIFLocation struct {
	PhysicalLocation struct {
		ArtifactLocation struct {
			URI string `json:"uri"`
		} `json:"artifactLocation"`
		Region struct {
			StartLine int `json:"startLine"`
		} `json:"region"`
	} `json:"physicalLocation"`
}

type SARIFInvocation struct {
	ExecutionSuccessful bool           `json:"executionSuccessful"`
	Properties          map[string]any `json:"properties,omitempty"`
}

// WriteSARIF runs validate (or uses payload) and writes SARIF under outPath.
func WriteSARIF(root string, packIDs []string, outPath string) (string, int, error) {
	res, err := validate.Run(validate.Options{RepoRoot: root, PackIDs: packIDs, Quiet: true})
	if err != nil {
		return "", 0, err
	}
	doc := FromGateFailures(res.Payload)
	if outPath == "" {
		outPath = filepath.Join(root, ".github", "cyberready", "cache", "cyberready.sarif")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", 0, err
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", 0, err
	}
	if err := os.WriteFile(outPath, append(b, '\n'), 0o644); err != nil {
		return "", 0, err
	}
	n := 0
	if len(doc.Runs) > 0 {
		n = len(doc.Runs[0].Results)
	}
	return outPath, n, nil
}

// FromGateFailures maps GateFailure IR to SARIF (ruleId == gate_id).
func FromGateFailures(payload ir.GateFailurePayload) SARIFDocument {
	rulesByID := map[string]SARIFRule{}
	results := make([]SARIFResult, 0, len(payload.Failures))
	for _, f := range payload.Failures {
		sev := strings.ToLower(f.Severity)
		level := "warning"
		if sev == "high" || sev == "critical" {
			level = "error"
		}
		if _, ok := rulesByID[f.GateID]; !ok {
			r := SARIFRule{ID: f.GateID}
			r.ShortDescription.Text = f.SanitizedDescription
			rulesByID[f.GateID] = r
		}
		res := SARIFResult{
			RuleID:  f.GateID,
			Level:   level,
			Message: SARIFMessage{Text: f.SanitizedDescription},
			Properties: map[string]any{
				"assurance_class":       "structural_draft",
				"certification_claimed": false,
				"instrument_panel":      true,
				"note":                  "Structural evidence for human review — not a conformity assessment.",
			},
		}
		file := strings.TrimSpace(f.ASTCoordinates.TargetFile)
		if file != "" {
			var loc SARIFLocation
			loc.PhysicalLocation.ArtifactLocation.URI = filepath.ToSlash(file)
			loc.PhysicalLocation.Region.StartLine = 1
			res.Locations = []SARIFLocation{loc}
		}
		results = append(results, res)
	}
	ruleIDs := make([]string, 0, len(rulesByID))
	for id := range rulesByID {
		ruleIDs = append(ruleIDs, id)
	}
	sort.Strings(ruleIDs)
	rules := make([]SARIFRule, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		rules = append(rules, rulesByID[id])
	}
	return SARIFDocument{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []SARIFRun{{
			Tool: SARIFTool{Driver: SARIFDriver{
				Name:           "cyberready",
				InformationURI: "https://github.com/afelin/cyberready",
				Rules:          rules,
			}},
			Results: results,
			Invocations: []SARIFInvocation{{
				ExecutionSuccessful: true,
				Properties: map[string]any{
					"assurance_class":       "structural_draft",
					"certification_claimed": false,
					"instrument_panel":      true,
					"note":                  "Structural evidence for human review — not a conformity assessment.",
				},
			}},
		}},
	}
}

// WatchlistJoinFinding is an informational SBOM ∩ watchlist hit.
type WatchlistJoinFinding struct {
	WatchlistID string `json:"watchlist_id"`
	Package     string `json:"package"`
	Version     string `json:"version"`
	PURL        string `json:"purl,omitempty"`
	Ecosystem   string `json:"ecosystem"`
	Reason      string `json:"reason"`
	Note        string `json:"note"`
}

// WatchlistJoinReport is informational only — does not fail gates.
type WatchlistJoinReport struct {
	SchemaVersion string                 `json:"schema_version"`
	Status        string                 `json:"status"`
	Findings      []WatchlistJoinFinding `json:"findings"`
	Note          string                 `json:"note"`
}

// WriteWatchlistJoin joins CycloneDX components with watchlist (informational).
func WriteWatchlistJoin(root, outPath string) (string, error) {
	wl, err := packs.LoadWatchlist()
	if err != nil {
		return "", err
	}
	pkgs, source, err := sbom.CollectPackages(root)
	report := WatchlistJoinReport{
		SchemaVersion: "1",
		Status:        "ok",
		Note:          "Informational watchlist∩SBOM join — does not fail validate. Not a vulnerability determination.",
	}
	if err != nil {
		if sbom.IsUnavailable(err) {
			report.Status = "unavailable"
			report.Note = "No supported lockfile/manifest for join (" + err.Error() + ")"
		} else {
			return "", err
		}
	} else {
		_ = source
		for _, e := range wl.Entries {
			for _, p := range pkgs {
				if !packageMatch(e, p) {
					continue
				}
				if len(e.Versions) > 0 && !versionMatch(e.Versions, p.Version) {
					continue
				}
				purl := e.PURL
				if purl == "" && e.Ecosystem == "npm" {
					purl = "pkg:npm/" + p.Name + "@" + p.Version
				}
				report.Findings = append(report.Findings, WatchlistJoinFinding{
					WatchlistID: e.ID,
					Package:     p.Name,
					Version:     p.Version,
					PURL:        purl,
					Ecosystem:   e.Ecosystem,
					Reason:      e.Reason,
					Note:        "Informational advisory row for human review.",
				})
			}
		}
		sort.Slice(report.Findings, func(i, j int) bool {
			return report.Findings[i].WatchlistID+report.Findings[i].Package < report.Findings[j].WatchlistID+report.Findings[j].Package
		})
	}
	if outPath == "" {
		outPath = filepath.Join(root, ".github", "cyberready", "cache", "watchlist-sbom-join.json")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(outPath, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	return outPath, nil
}

func packageMatch(e packs.WatchlistEntry, p sbom.Package) bool {
	eco := strings.ToLower(e.Ecosystem)
	pe := strings.ToLower(p.Ecosystem)
	if pe == "" {
		pe = "npm"
	}
	switch eco {
	case "maven":
		return false // not in npm/go SBOM collectors yet
	case "golang", "go":
		if pe != "golang" && pe != "go" {
			return false
		}
	case "npm", "":
		if pe != "npm" {
			return false
		}
	}
	return strings.EqualFold(e.Package, p.Name)
}

func versionMatch(banned []string, ver string) bool {
	ver = strings.TrimSpace(ver)
	for _, b := range banned {
		if strings.TrimSpace(b) == ver {
			return true
		}
	}
	return false
}