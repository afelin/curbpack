package packs

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed data/cra-baseline/pack.json data/medtech-iec62304/pack.json data/house-policy/pack.json data/_watchlist.json
var embedded embed.FS

// Supported check kinds (engine stays industry-agnostic).
var supportedChecks = map[string]struct{}{
	"annex_file":       {},
	"file_present":     {},
	"anti_placeholder": {},
	"npm_dep_ban":      {},
	"manifest_dep_ban": {},
	"text_forbid":      {},
	"import_reach":     {},
}

// Rule is a single pack gate definition (JSON-eval, no OPA).
type Rule struct {
	ID               string     `json:"id"`
	Severity         string     `json:"severity"`
	Type             string     `json:"type"`
	Check            string     `json:"check"`
	Path             string     `json:"path,omitempty"`
	Paths            []string   `json:"paths,omitempty"`
	MinBytes         int        `json:"min_bytes,omitempty"`
	MinWords         int        `json:"min_words,omitempty"`
	RequireHeaders   []string   `json:"require_headers,omitempty"`
	BindRepoToken    bool       `json:"bind_repo_token,omitempty"`
	RequireTreePaths []string   `json:"require_tree_paths,omitempty"`
	Package          string     `json:"package,omitempty"`
	BannedVersions   []string   `json:"banned_versions,omitempty"`
	Pattern          string     `json:"pattern,omitempty"`
	Description      string     `json:"description"`
	Remediation      string     `json:"remediation"`
	Expected         string     `json:"expected"`
	Citations        []Citation `json:"citations,omitempty"`
}

// Citation links a pack/rule/watchlist entry to a regulatory instrument (informational).
type Citation struct {
	Framework     string `json:"framework,omitempty"`
	Instrument    string `json:"instrument,omitempty"`
	Article       string `json:"article,omitempty"`
	Annex         string `json:"annex,omitempty"`
	URL           string `json:"url,omitempty"`
	EffectiveFrom string `json:"effective_from,omitempty"`
	EffectiveTo   string `json:"effective_to,omitempty"`
}

// Validity is an optional pack-level effective window (YYYY-MM-DD or RFC3339 date).
type Validity struct {
	EffectiveFrom string `json:"effective_from,omitempty"`
	EffectiveTo   string `json:"effective_to,omitempty"`
}

// Pack is an embedded regulation / house-policy pack.
type Pack struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Description    string          `json:"description"`
	AssuranceClass string          `json:"assurance_class,omitempty"`
	Extends        string          `json:"extends,omitempty"`
	Overlays       []string        `json:"overlays,omitempty"`
	Overlay        json.RawMessage `json:"overlay,omitempty"` // optional RFC 7386 merge-patch on pack object
	Jurisdiction   string          `json:"jurisdiction,omitempty"`
	Validity       *Validity       `json:"validity,omitempty"`
	Supersedes     string          `json:"supersedes,omitempty"`
	SupersededBy   string          `json:"superseded_by,omitempty"`
	Citations      []Citation      `json:"citations,omitempty"`
	Rules          []Rule          `json:"rules"`
}

// Watchlist is informational only.
type Watchlist struct {
	Version int              `json:"version"`
	Updated string           `json:"updated"`
	Note    string           `json:"note"`
	Entries []WatchlistEntry `json:"entries"`
}

// WatchlistEntry is one informational advisory row.
type WatchlistEntry struct {
	ID        string     `json:"id"`
	Ecosystem string     `json:"ecosystem"`
	Package   string     `json:"package"`
	Versions  []string   `json:"versions"`
	Reason    string     `json:"reason"`
	Refs      []string   `json:"refs"`
	Citations []Citation `json:"citations,omitempty"`
	PURL      string     `json:"purl,omitempty"`
}

var builtinIDs = []string{"cra-baseline", "medtech-iec62304", "house-policy"}

// LoadEmbedded returns all built-in packs (schema-validated).
func LoadEmbedded() ([]Pack, error) {
	out := make([]Pack, 0, len(builtinIDs))
	for _, id := range builtinIDs {
		p, err := LoadPack(id)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// LoadPack loads one pack by id from embed, or from CYBERREADY_PACKS_DIR override.
func LoadPack(id string) (Pack, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Pack{}, fmt.Errorf("empty pack id")
	}
	if dir := envPacksDir(); dir != "" {
		packPath := filepath.Join(dir, id, "pack.json")
		if _, statErr := os.Stat(packPath); statErr == nil {
			return loadPackFromDir(dir, id) // prefer override; surface validation errors
		}
	}
	return loadPackEmbeddedOnly(id)
}

func envPacksDir() string {
	return strings.TrimSpace(os.Getenv("CYBERREADY_PACKS_DIR"))
}

func loadPackFromDir(dir, id string) (Pack, error) {
	data, err := os.ReadFile(filepath.Join(dir, id, "pack.json"))
	if err != nil {
		return Pack{}, err
	}
	return parseAndValidate(id, data)
}

func loadPackEmbeddedOnly(id string) (Pack, error) {
	data, err := embedded.ReadFile("data/" + id + "/pack.json")
	if err != nil {
		return Pack{}, fmt.Errorf("pack %q not found: %w", id, err)
	}
	return parseAndValidate(id, data)
}

func parseAndValidate(id string, data []byte) (Pack, error) {
	var p Pack
	if err := json.Unmarshal(data, &p); err != nil {
		return Pack{}, fmt.Errorf("pack %q JSON: %w", id, err)
	}
	if err := ValidatePack(p); err != nil {
		return Pack{}, err
	}
	return p, nil
}

// ValidatePack enforces the pack load schema (generic fields only — no industry branches).
func ValidatePack(p Pack) error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("pack schema: missing id")
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("pack %q schema: missing name", p.ID)
	}
	if strings.TrimSpace(p.Version) == "" {
		return fmt.Errorf("pack %q schema: missing version", p.ID)
	}
	if len(p.Rules) == 0 {
		return fmt.Errorf("pack %q schema: rules must be non-empty", p.ID)
	}
	if err := validateCitations(p.Citations, fmt.Sprintf("pack %q", p.ID)); err != nil {
		return err
	}
	if p.Validity != nil {
		if err := validateDateWindow(p.Validity.EffectiveFrom, p.Validity.EffectiveTo); err != nil {
			return fmt.Errorf("pack %q validity: %w", p.ID, err)
		}
	}
	seen := map[string]struct{}{}
	for i, r := range p.Rules {
		if strings.TrimSpace(r.ID) == "" {
			return fmt.Errorf("pack %q schema: rules[%d] missing id", p.ID, i)
		}
		if _, ok := seen[r.ID]; ok {
			return fmt.Errorf("pack %q schema: duplicate rule id %q", p.ID, r.ID)
		}
		seen[r.ID] = struct{}{}
		if strings.TrimSpace(r.Check) == "" {
			return fmt.Errorf("pack %q rule %q: missing check", p.ID, r.ID)
		}
		if _, ok := supportedChecks[r.Check]; !ok {
			return fmt.Errorf("pack %q rule %q: unsupported check %q", p.ID, r.ID, r.Check)
		}
		if strings.TrimSpace(r.Severity) == "" {
			return fmt.Errorf("pack %q rule %q: missing severity", p.ID, r.ID)
		}
		if strings.TrimSpace(r.Description) == "" {
			return fmt.Errorf("pack %q rule %q: missing description", p.ID, r.ID)
		}
		if err := validateCitations(r.Citations, fmt.Sprintf("pack %q rule %q", p.ID, r.ID)); err != nil {
			return err
		}
		for _, tp := range r.RequireTreePaths {
			if err := ValidateRelPath(tp); err != nil {
				return fmt.Errorf("pack %q rule %q: require_tree_paths: %w", p.ID, r.ID, err)
			}
		}
		switch r.Check {
		case "annex_file", "file_present":
			if strings.TrimSpace(r.Path) == "" {
				return fmt.Errorf("pack %q rule %q: path required for %s", p.ID, r.ID, r.Check)
			}
			if err := ValidateRelPath(r.Path); err != nil {
				return fmt.Errorf("pack %q rule %q: %w", p.ID, r.ID, err)
			}
		case "anti_placeholder", "text_forbid":
			if len(r.Paths) == 0 {
				return fmt.Errorf("pack %q rule %q: paths required for %s", p.ID, r.ID, r.Check)
			}
			for _, path := range r.Paths {
				if err := ValidateRelPath(path); err != nil {
					return fmt.Errorf("pack %q rule %q: %w", p.ID, r.ID, err)
				}
			}
			if r.Check == "text_forbid" {
				if strings.TrimSpace(r.Pattern) == "" {
					return fmt.Errorf("pack %q rule %q: pattern required for text_forbid", p.ID, r.ID)
				}
				if err := ValidateRegexPattern(r.Pattern); err != nil {
					return fmt.Errorf("pack %q rule %q: %w", p.ID, r.ID, err)
				}
			}
		case "npm_dep_ban", "manifest_dep_ban":
			if strings.TrimSpace(r.Package) == "" {
				return fmt.Errorf("pack %q rule %q: package required for %s", p.ID, r.ID, r.Check)
			}
			if len(r.BannedVersions) == 0 {
				return fmt.Errorf("pack %q rule %q: banned_versions required", p.ID, r.ID)
			}
		}
	}
	return nil
}

func validateCitations(cs []Citation, ctx string) error {
	for i, c := range cs {
		if err := validateDateWindow(c.EffectiveFrom, c.EffectiveTo); err != nil {
			return fmt.Errorf("%s citations[%d]: %w", ctx, i, err)
		}
	}
	return nil
}

// validateDateWindow accepts empty, YYYY-MM-DD, or RFC3339; refuses inverted windows.
func validateDateWindow(from, to string) error {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" && to == "" {
		return nil
	}
	parse := func(s string) (time.Time, error) {
		if s == "" {
			return time.Time{}, nil
		}
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t, nil
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t, nil
		}
		return time.Time{}, fmt.Errorf("invalid date %q (want YYYY-MM-DD or RFC3339)", s)
	}
	ft, err := parse(from)
	if err != nil {
		return err
	}
	tt, err := parse(to)
	if err != nil {
		return err
	}
	if !ft.IsZero() && !tt.IsZero() && tt.Before(ft) {
		return fmt.Errorf("effective_to before effective_from")
	}
	return nil
}

// ValidateRelPath refuses absolute paths, traversal, and .git/** (pack path jail).
// Used by ValidatePack and scaffold writers before joining under a repo root.
func ValidateRelPath(rel string) error {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return fmt.Errorf("empty path")
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "/") {
		return fmt.Errorf("absolute path refused")
	}
	clean := filepath.ToSlash(filepath.Clean(rel))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("path traversal refused")
	}
	if clean == ".git" || strings.HasPrefix(clean, ".git/") {
		return fmt.Errorf("path under .git refused")
	}
	return nil
}

// MaxRegexPatternLen is the hard cap on pack text_forbid patterns (ReDoS hygiene).
const MaxRegexPatternLen = 256

// MaxRegexMatchBytes caps file bytes scanned by text_forbid Match.
const MaxRegexMatchBytes = 2 << 20 // 2 MiB

// ValidateRegexPattern rejects oversized / pathological patterns at pack load.
func ValidateRegexPattern(pattern string) error {
	if utf8.RuneCountInString(pattern) > MaxRegexPatternLen {
		return fmt.Errorf("pattern exceeds %d runes (ReDoS limit)", MaxRegexPatternLen)
	}
	if nested := countNestedQuantifiers(pattern); nested > 3 {
		return fmt.Errorf("pattern too nested (%d overlapping quantifiers; ReDoS limit)", nested)
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}
	return nil
}

// countNestedQuantifiers approximates risky *+?{n,} nesting (heuristic, fail-closed).
func countNestedQuantifiers(pattern string) int {
	depth := 0
	max := 0
	inClass := false
	escaped := false
	for _, r := range pattern {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '[' {
			inClass = true
			continue
		}
		if r == ']' && inClass {
			inClass = false
			continue
		}
		if inClass {
			continue
		}
		if r == '(' {
			depth++
			if depth > max {
				max = depth
			}
			continue
		}
		if r == ')' {
			if depth > 0 {
				depth--
			}
			continue
		}
		if r == '*' || r == '+' || r == '?' {
			if depth > max {
				max = depth
			}
		}
	}
	return max
}

// LoadWatchlist returns the embedded (or overridden) watchlist.
func LoadWatchlist() (Watchlist, error) {
	if dir := strings.TrimSpace(os.Getenv("CYBERREADY_PACKS_DIR")); dir != "" {
		data, err := os.ReadFile(filepath.Join(dir, "_watchlist.json"))
		if err == nil {
			var w Watchlist
			if err := json.Unmarshal(data, &w); err != nil {
				return Watchlist{}, err
			}
			return w, nil
		}
	}
	data, err := embedded.ReadFile("data/_watchlist.json")
	if err != nil {
		return Watchlist{}, err
	}
	var w Watchlist
	if err := json.Unmarshal(data, &w); err != nil {
		return Watchlist{}, err
	}
	return w, nil
}

// ListIDs returns sorted pack identifiers.
func ListIDs() ([]string, error) {
	packs, err := LoadEmbedded()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(packs))
	for _, p := range packs {
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	return ids, nil
}

// ScaffoldPaths returns unique relative file paths referenced by pack rules (for init).
// Every path is jail-checked via ValidateRelPath (fail closed on escape).
// Uses Compose so extends/overlays contribute scaffold paths.
func ScaffoldPaths(packIDs []string) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	add := func(rel string) error {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			return nil
		}
		if err := ValidateRelPath(rel); err != nil {
			return err
		}
		clean := filepath.ToSlash(filepath.Clean(rel))
		if _, ok := seen[clean]; ok {
			return nil
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
		return nil
	}
	composed, _, err := Compose(packIDs)
	if err != nil {
		return nil, err
	}
	for _, r := range composed.Rules {
		switch r.Check {
		case "annex_file", "file_present":
			if err := add(r.Path); err != nil {
				return nil, fmt.Errorf("composed rule %q: %w", r.ID, err)
			}
		case "anti_placeholder", "text_forbid":
			for _, path := range r.Paths {
				if err := add(path); err != nil {
					return nil, fmt.Errorf("composed rule %q: %w", r.ID, err)
				}
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// DefaultScaffoldBody returns a minimal non-placeholder draft for a relative path.
func DefaultScaffoldBody(rel string) string {
	base := filepath.Base(rel)
	switch {
	case strings.Contains(rel, "risk_assessment"):
		return `# Risk Assessment

## Product Overview

Describe the product, intended use, and operating environment.

## Identified Risks

| Risk ID | Description | Severity | Mitigation |
|---------|-------------|----------|------------|
| R-001   |             |          |            |

## Residual Risk Statement

State residual risk after mitigations.
`
	case strings.Contains(rel, "support_period"):
		return `# Support Period

## End of Support

Declare the date or duration of security update support.

## Rationale

Explain how the support period was chosen.
`
	case strings.Contains(rel, "user_manual"):
		return `# User Manual — Security

## Secure Configuration

Document default-secure settings and hardening steps.

## Product Disposal

Explain secure decommissioning and data wiping.
`
	case strings.Contains(rel, "software_safety_class"):
		return `# Software Safety Class

## Classification Rationale

State IEC 62304 Class A/B/C and why.
`
	case strings.Contains(rel, "soup_list"):
		return `# SOUP List

## Items

| Component | Version | Manufacturer | Residual Risk |
|-----------|---------|--------------|---------------|
|           |         |              |               |
`
	case strings.Contains(rel, "problem_resolution"):
		return `# Problem Resolution

## Process

Describe intake, triage, fix, verification, and release of corrections.
`
	case base == "SECURITY.md":
		return `# Security

## Reporting

Email security@example.com with vulnerability details. Do not open public issues for sensitive reports.

## Response

We acknowledge within two business days and coordinate disclosure timelines.
`
	case strings.HasSuffix(rel, "security.txt"):
		return `Contact: mailto:security@example.com
Expires: 2027-12-31T23:59:59.000Z
Preferred-Languages: en
`
	case base == "README.md":
		return "# Project\n\nDescribe the product briefly.\n"
	default:
		title := strings.TrimSuffix(base, filepath.Ext(base))
		title = strings.ReplaceAll(title, "_", " ")
		title = strings.ReplaceAll(title, "-", " ")
		if title == "" {
			title = "Evidence draft"
		}
		return "# " + title + "\n\nDraft content for human review — expand before release.\n"
	}
}

// RuleTouchesDiff reports whether a rule's declared paths intersect changed files.
// Rules without path/paths always run (true).
// file_present / annex_file always run under --diff: missing or short required files
// never appear in porcelain, so skipping them produces false greens.
func RuleTouchesDiff(rule Rule, changed map[string]struct{}) bool {
	switch rule.Check {
	case "file_present", "annex_file":
		return true
	}
	if len(changed) == 0 {
		return true
	}
	paths := append([]string{}, rule.Paths...)
	if rule.Path != "" {
		paths = append(paths, rule.Path)
	}
	if len(paths) == 0 {
		return true // dep bans / global checks always evaluate
	}
	for _, p := range paths {
		p = filepath.ToSlash(p)
		if _, ok := changed[p]; ok {
			return true
		}
		base := filepath.Base(p)
		for c := range changed {
			if c == p || filepath.Base(c) == base {
				return true
			}
		}
	}
	return false
}

// ExportPackJSON writes a pack JSON to destDir/<id>/pack.json (air-gap helper).
func ExportPackJSON(id, destDir string) error {
	p, err := LoadPack(id)
	if err != nil {
		return err
	}
	dir := filepath.Join(destDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "pack.json"), append(data, '\n'), 0o644)
}
