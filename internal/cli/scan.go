package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/afelin/curbpack/internal/clock"
	"github.com/afelin/curbpack/internal/config"
	"github.com/afelin/curbpack/internal/gitutil"
	"github.com/afelin/curbpack/internal/ir"
	"github.com/afelin/curbpack/internal/packs"
	"github.com/afelin/curbpack/internal/tty"
	"github.com/afelin/curbpack/internal/validate"
)

func cmdScan(args []string) error {
	flags, err := parseScanFlags(args)
	if err != nil {
		return err
	}
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("must run inside a git repository")
	}

	packIDs, err := config.ResolvePackIDsForScan(root, flags.packIDs)
	if err != nil {
		return err
	}

	res, err := validate.Run(validate.Options{
		RepoRoot: root,
		PackIDs:  packIDs,
		Quiet:    true,
		ReadOnly: true,
	})
	if err != nil {
		return err
	}

	composed, _, err := packs.Compose(packIDs)
	if err != nil {
		return err
	}

	notStarted, failing := classifyFindings(res.Payload.Failures)
	days := clock.DaysUntilUTC(clock.Art14ReportingStart)

	if flags.badge || flags.formatMarkdown {
		fmt.Println(art14BadgeLine(root, clock.NowUTC()))
		return nil
	}

	tty.PrintHeader("CURBPACK SCAN")
	fmt.Printf("%s\n", tty.C(tty.Bold+tty.Yellow, "Read-only — no files written, no hooks, no init. Not conformity assessment."))
	fmt.Printf("%s\n\n", tty.C(tty.Dim, "Diagnosis only — use curbpack check --score for readiness %."))

	product, source := productHint(root)
	fmt.Printf("Product hint: %s (%s)\n", product, source)
	fmt.Printf("Repo: %s\n", root)
	fmt.Printf("Packs: %s\n", strings.Join(packIDs, ", "))

	if scanShowsENISAMapping(packIDs) {
		fmt.Printf("%s\n", tty.C(tty.Dim, "ENISA SME maturity mapping (informational): docs/mappings/enisa-cra-mapping.md"))
	}

	switch {
	case days > 0:
		fmt.Printf("Art 14 reporting clock: %d days until 2026-09-11\n", days)
	case days == 0:
		fmt.Println("Art 14 reporting clock: starts today (2026-09-11)")
	default:
		fmt.Printf("Art 14 reporting clock: started %d days ago (2026-09-11)\n", -days)
	}

	failingIDs := failureGateIDs(res.Payload.Failures)
	satisfied := satisfiedRules(composed, failingIDs)

	fmt.Printf("\nOpen signals: %d failing · %d not started\n", len(failing), len(notStarted))

	satShown := 0
	for _, rule := range satisfied {
		if satShown >= 3 {
			break
		}
		fmt.Printf("  ✔ [%s] %s — %s\n", rule.Severity, rule.ID, ruleDisplayPath(rule))
		satShown++
	}
	if len(satisfied) > 3 {
		fmt.Printf("  … and %d more satisfied\n", len(satisfied)-3)
	}

	openShown := 0
	for _, f := range failing {
		if openShown >= 5 {
			break
		}
		fmt.Printf("  ✘ [%s] %s — %s\n", f.Severity, f.GateID, shortFinding(f))
		openShown++
	}
	for _, f := range notStarted {
		if openShown >= 5 {
			break
		}
		fmt.Printf("  ○ [%s] %s — %s (not started)\n", f.Severity, f.GateID, shortFinding(f))
		openShown++
	}
	rest := len(failing) + len(notStarted) - openShown
	if rest > 0 {
		fmt.Printf("  … and %d more\n", rest)
	}

	if res.Passed {
		fmt.Printf("\n%s\n", tty.C(tty.Green, "No open gate findings on this tree — still not certification."))
	} else {
		fmt.Printf("\n%s\n", tty.C(tty.Dim, scanNextLine(res.Payload.Failures)))
	}
	fmt.Printf("%s\n", tty.C(tty.Dim, "Prepares evidence for human review — not a conformity assessment."))
	return nil
}

func scanNextLine(failures []ir.Failure) string {
	if hasGateFailure(failures, "CRA-ART14-PATH") {
		return "Next: curbpack fix --art14 · curbpack init · curbpack check --score"
	}
	return "Next: curbpack init · curbpack check --score"
}

func failureGateIDs(failures []ir.Failure) map[string]struct{} {
	ids := make(map[string]struct{}, len(failures))
	for _, f := range failures {
		if f.GateID != "" {
			ids[f.GateID] = struct{}{}
		}
	}
	return ids
}

func satisfiedRules(composed packs.Pack, failingIDs map[string]struct{}) []packs.Rule {
	var out []packs.Rule
	for _, rule := range composed.Rules {
		if _, fail := failingIDs[rule.ID]; !fail {
			out = append(out, rule)
		}
	}
	return out
}

func ruleDisplayPath(rule packs.Rule) string {
	if p := strings.TrimSpace(rule.Path); p != "" {
		return p
	}
	if len(rule.Paths) > 0 {
		return rule.Paths[0]
	}
	return rule.Description
}

func hasGateFailure(failures []ir.Failure, gateID string) bool {
	for _, f := range failures {
		if f.GateID == gateID {
			return true
		}
	}
	return false
}

func productHint(root string) (name, source string) {
	if n, ok := packs.RepoToken(root); ok {
		if _, err := os.Stat(filepath.Join(root, "package.json")); err == nil {
			return n, "package.json"
		}
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return n, "go.mod"
		}
		return n, "repo name"
	}
	if title := readmeTitle(root); title != "" {
		return title, "README"
	}
	return filepath.Base(root), "directory"
}

func readmeTitle(root string) string {
	b, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func classifyFindings(failures []ir.Failure) (notStarted, failing []ir.Failure) {
	for _, f := range failures {
		desc := strings.ToLower(f.SanitizedDescription)
		if strings.Contains(desc, "scaffold body overlap") ||
			strings.Contains(desc, "missing") ||
			strings.Contains(desc, "too short") ||
			strings.Contains(desc, "too small") {
			notStarted = append(notStarted, f)
			continue
		}
		failing = append(failing, f)
	}
	return notStarted, failing
}

func shortFinding(f ir.Failure) string {
	if p := strings.TrimSpace(f.ASTCoordinates.TargetFile); p != "" {
		return p
	}
	if len(f.SanitizedDescription) > 72 {
		return f.SanitizedDescription[:69] + "..."
	}
	return f.SanitizedDescription
}

func scanShowsENISAMapping(packIDs []string) bool {
	for _, id := range packIDs {
		if id == "cra-baseline" || id == "medtech-iec62304" {
			return true
		}
	}
	return false
}

var lastTabletopDateRE = regexp.MustCompile(`(?im)^Last tabletop:\s*(\d{4}-\d{2}-\d{2})`)

// art14BadgeLine emits a copy-paste badge from Last tabletop: only (never Drafted:).
func art14BadgeLine(root string, now time.Time) string {
	path := filepath.Join(root, packs.Art14RelPath())
	body, err := os.ReadFile(path)
	if err != nil {
		return "Art 14 path · not rehearsed · self-declared · curbpack"
	}
	date, ok := parseLastTabletopDate(string(body))
	if !ok {
		return "Art 14 path · not rehearsed · self-declared · curbpack"
	}
	return fmt.Sprintf("Art 14 path · rehearsed %s %s · self-declared · curbpack",
		date.Format("2006-01-02"), formatMonthsAgo(date, now))
}

func parseLastTabletopDate(body string) (time.Time, bool) {
	m := lastTabletopDateRE.FindStringSubmatch(body)
	if len(m) < 2 || strings.TrimSpace(m[1]) == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", m[1])
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func formatMonthsAgo(t, now time.Time) string {
	months := (now.Year()-t.Year())*12 + int(now.Month()-t.Month())
	if now.Day() < t.Day() {
		months--
	}
	if months < 0 {
		months = 0
	}
	switch months {
	case 0:
		return "(this month)"
	case 1:
		return "(1 month ago)"
	default:
		return fmt.Sprintf("(%d months ago)", months)
	}
}
