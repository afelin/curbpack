package validate

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/afelin/curbpack/internal/clock"
	"github.com/afelin/curbpack/internal/config"
	"github.com/afelin/curbpack/internal/gitutil"
	"github.com/afelin/curbpack/internal/ir"
	"github.com/afelin/curbpack/internal/pathjail"
	"github.com/afelin/curbpack/internal/packs"
	"github.com/afelin/curbpack/internal/pathway"
	"github.com/afelin/curbpack/internal/tty"
)

// High-signal placeholders / LLM boilerplate — alternation kept short for ReDoS safety.
var placeholderRE = regexp.MustCompile(`(?i)(lorem ipsum|\[insert[^\]]*\]|TODO:|FIXME:|placeholder|xxxx|<\s*company\s*>|as an ai language model|certainly[!.,]?\s+here('s| is)|in today's digital landscape|delve into|it is important to note that)`)

// Options controls validate / check.
type Options struct {
	RepoRoot string
	PackIDs  []string
	Quiet    bool
	DiffOnly bool // skip rules whose paths are untouched by git diff
	ReadOnly bool // skip cache writes (scan / read-only paths)
	// FreshStubPaths: paths written by init/--heal in the same CLI invocation —
	// skip anti_placeholder scaffold overlap (next run still scores them).
	FreshStubPaths map[string]struct{}
}

// Result is the outcome of a validate run.
type Result struct {
	Payload      ir.GateFailurePayload
	Passed       bool
	Score        int
	SkippedRules int
	ActionReport string
}

// Run evaluates embedded pack rules against the repo tree.
func Run(opts Options) (Result, error) {
	prevFresh := freshStubPathsEval
	freshStubPathsEval = opts.FreshStubPaths
	defer func() { freshStubPathsEval = prevFresh }()

	root := opts.RepoRoot
	if root == "" {
		var err error
		root, err = gitutil.RepoRoot("")
		if err != nil {
			return Result{}, err
		}
	}

	ids, err := config.ResolvePackIDs(root, opts.PackIDs)
	if err != nil {
		return Result{}, err
	}

	var changed map[string]struct{}
	if opts.DiffOnly {
		changed, err = gitutil.ChangedFiles(root)
		if err != nil {
			// Fall back to full scan if git diff unavailable
			changed = nil
			opts.DiffOnly = false
		}
	}

	var failures []ir.Failure
	var regions []string
	skipped := 0
	composed, _, err := packs.Compose(ids)
	if err != nil {
		return Result{}, err
	}
	for _, rule := range composed.Rules {
		if opts.DiffOnly && !packs.RuleTouchesDiff(rule, changed) {
			skipped++
			if !opts.Quiet {
				tty.PrintStatus("Gate "+rule.ID, true, "skipped (diff)")
			}
			continue
		}
		fs := evalRule(root, rule)
		if len(fs) > 0 {
			regions = append(regions, rule.ID)
			failures = append(failures, fs...)
			if !opts.Quiet {
				tty.PrintStatus("Gate "+rule.ID, false, rule.Description)
			}
		} else if !opts.Quiet {
			tty.PrintStatus("Gate "+rule.ID, true, "ok")
		}
	}

	// Built-in AST reachability (MVP lift) — only if file exists
	if !opts.DiffOnly || pathChanged(changed, "src/payment.go") {
		failures = append(failures, auditASTReachability(root)...)
	}

	score := tty.ScoreFromFailures(len(failures))
	parent, err := gitutil.HeadSHA(root)
	if err != nil {
		if !opts.Quiet {
			tty.WarnOCCParent("HEAD unresolved; OCC parent omitted (best-effort): " + err.Error())
		}
		parent = ""
	}
	parentPath := []string{"Root", "ActiveVerification", "PackEval"}
	if seed, serr := pathway.Load(root); serr == nil && seed != nil {
		if phase, perr := pathway.DerivePhase(root, seed); perr == nil {
			parentPath = pathway.ParentStatePath(phase)
		}
	}
	payload := ir.GateFailurePayload{
		SchemaVersion: ir.SchemaVersion,
		Timestamp:     clock.RFC3339(),
		ConcurrencyControl: ir.ConcurrencyControl{
			ExpectedParentCommitSHA: parent,
			StateVersionToken:       "v3.33-OCC",
		},
		StatechartContext: ir.StatechartContext{
			ActiveParentStatePath:   parentPath,
			FailedOrthogonalRegions: unique(regions),
		},
		AgentIdentity:  ir.ResolveAgentIdentity(),
		Failures:       failures,
		PackID:         strings.Join(ids, ","),
		ReadinessScore: score,
	}

	action := ActionReportMarkdown(payload, skipped)
	if !opts.ReadOnly {
		cacheDir := filepath.Join(root, ".github", "curbpack", "cache")
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			tty.WarnCacheWrite("mkdir cache: " + err.Error())
		}
		b, _ := json.MarshalIndent(payload, "", "  ")
		writeCache := func(name string) {
			if err := os.WriteFile(filepath.Join(cacheDir, name), b, 0o644); err != nil {
				tty.WarnCacheWrite("write " + name + ": " + err.Error())
			}
		}
		writeCache("latest_failure.json")
		writeCache("latest_result.json")
		if err := os.WriteFile(filepath.Join(cacheDir, "latest_action_report.md"), []byte(action), 0o644); err != nil {
			tty.WarnCacheWrite("write latest_action_report.md: " + err.Error())
		}
	}

	return Result{
		Payload:      payload,
		Passed:       len(failures) == 0,
		Score:        score,
		SkippedRules: skipped,
		ActionReport: action,
	}, nil
}

func pathChanged(changed map[string]struct{}, rel string) bool {
	if changed == nil {
		return true
	}
	_, ok := changed[filepath.ToSlash(rel)]
	return ok
}

func evalRule(root string, rule packs.Rule) []ir.Failure {
	fn, ok := checkRegistry[CheckKind(rule.Check)]
	if !ok {
		return []ir.Failure{{
			GateID:               rule.ID,
			Severity:             "medium",
			Type:                 "CONFIG_ERROR",
			SanitizedDescription: fmt.Sprintf("Unknown check type %q in pack rule", rule.Check),
			Remediation: ir.Remediation{
				ActionRequired: "Fix pack JSON check field.",
				ExpectedState:  "Supported check type.",
			},
		}}
	}
	return fn(root, rule)
}

// SafeJoin resolves rel under root with symlink-aware containment and .git jail (fail closed).
func SafeJoin(root, rel string) (string, string, error) {
	return pathjail.Join(root, rel)
}

func checkFilePresent(root string, rule packs.Rule) []ir.Failure {
	path, rel, err := SafeJoin(root, rule.Path)
	if err != nil {
		return []ir.Failure{failFromRule(rule, rule.Path, err.Error())}
	}
	info, err := os.Stat(path)
	min := rule.MinBytes
	if min <= 0 {
		min = 50
	}
	if os.IsNotExist(err) || (err == nil && info.Size() < int64(min)) {
		return []ir.Failure{failFromRule(rule, rel, "")}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return []ir.Failure{failFromRule(rule, rel, err.Error())}
	}
	content := string(data)
	for _, h := range rule.RequireHeaders {
		if !strings.Contains(content, h) {
			return []ir.Failure{failFromRule(rule, rel, "missing header: "+h)}
		}
	}
	if rule.MinWords > 0 && wordCount(content) < rule.MinWords {
		return []ir.Failure{failFromRule(rule, rel, fmt.Sprintf("min_words=%d not met (have %d)", rule.MinWords, wordCount(content)))}
	}
	if rule.BindRepoToken {
		token, ok := packs.RepoToken(root)
		if !ok {
			return []ir.Failure{failFromRule(rule, rel, "bind_repo_token: no resolvable repo token (directory name, package.json name, or go.mod module)")}
		}
		if !strings.Contains(content, token) {
			return []ir.Failure{failFromRule(rule, rel, "bind_repo_token: draft must mention repo token "+token)}
		}
	}
	for _, tp := range rule.RequireTreePaths {
		full, clean, err := SafeJoin(root, tp)
		if err != nil {
			return []ir.Failure{failFromRule(rule, rel, "require_tree_paths: "+err.Error())}
		}
		if _, err := os.Stat(full); err != nil {
			return []ir.Failure{failFromRule(rule, rel, "require_tree_paths: missing "+clean)}
		}
	}
	return nil
}

func checkFresh(root string, rule packs.Rule) []ir.Failure {
	if fs := checkFilePresent(root, rule); len(fs) > 0 {
		return fs
	}
	rel := filepath.ToSlash(rule.Path)
	if rule.MaxAgeDays > 0 {
		meta, err := gitutil.FileLastCommit(root, rel)
		if err != nil {
			return []ir.Failure{failFromRule(rule, rel, "fresh: "+err.Error())}
		}
		age := time.Since(meta.Time)
		if age > time.Duration(rule.MaxAgeDays)*24*time.Hour {
			return []ir.Failure{failFromRule(rule, rel, fmt.Sprintf("fresh: last commit %s older than %d days", meta.Time.Format(time.RFC3339), rule.MaxAgeDays))}
		}
	}
	if ref := strings.TrimSpace(rule.SinceRef); ref != "" {
		ok, err := gitutil.FileTouchedSinceRef(root, ref, rel)
		if err != nil {
			return []ir.Failure{failFromRule(rule, rel, "fresh since_ref: "+err.Error())}
		}
		if !ok {
			return []ir.Failure{failFromRule(rule, rel, "fresh: no commit since "+ref)}
		}
	}
	return nil
}

func checkOwned(root string, rule packs.Rule) []ir.Failure {
	if fs := checkFilePresent(root, rule); len(fs) > 0 {
		return fs
	}
	rel := filepath.ToSlash(rule.Path)
	meta, err := gitutil.FileLastCommit(root, rel)
	if err != nil {
		return []ir.Failure{failFromRule(rule, rel, "owned: "+err.Error())}
	}
	if want := strings.TrimSpace(rule.RequireGitAuthorEmail); want != "" && !strings.EqualFold(meta.Email, want) {
		return []ir.Failure{failFromRule(rule, rel, fmt.Sprintf("owned: last commit author email %q want %q", meta.Email, want))}
	}
	if want := strings.TrimSpace(rule.RequireGitAuthorName); want != "" && meta.Name != want {
		return []ir.Failure{failFromRule(rule, rel, fmt.Sprintf("owned: last commit author name %q want %q", meta.Name, want))}
	}
	return nil
}

func wordCount(s string) int {
	n := 0
	in := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			in = false
			continue
		}
		if !in {
			n++
			in = true
		}
	}
	return n
}

func checkAntiPlaceholder(root string, rule packs.Rule) []ir.Failure {
	var out []ir.Failure
	token, _ := packs.RepoToken(root)
	for _, rel := range rule.Paths {
		path, clean, err := SafeJoin(root, rel)
		if err != nil {
			out = append(out, failFromRule(rule, rel, err.Error()))
			continue
		}
		if freshStubPathsEval != nil {
			if _, skip := freshStubPathsEval[filepath.ToSlash(clean)]; skip {
				continue
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue // missing handled by file_present / annex_file rules
		}
		if placeholderRE.Match(data) {
			out = append(out, failFromRule(rule, clean, "placeholder pattern matched"))
			continue
		}
		if packs.ScaffoldOverlap(string(data), clean, token) {
			out = append(out, failFromRule(rule, clean, "scaffold body overlap"))
		}
	}
	return out
}

var freshStubPathsEval map[string]struct{}

func checkTextForbid(root string, rule packs.Rule) []ir.Failure {
	if err := packs.ValidateRegexPattern(rule.Pattern); err != nil {
		return []ir.Failure{failFromRule(rule, strings.Join(rule.Paths, ","), err.Error())}
	}
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return []ir.Failure{failFromRule(rule, strings.Join(rule.Paths, ","), "invalid pattern: "+err.Error())}
	}
	var out []ir.Failure
	for _, rel := range rule.Paths {
		path, clean, err := SafeJoin(root, rel)
		if err != nil {
			out = append(out, failFromRule(rule, rel, err.Error()))
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if len(data) > packs.MaxRegexMatchBytes {
			data = data[:packs.MaxRegexMatchBytes]
		}
		matched, timedOut := matchWithTimeout(re, data, 50*time.Millisecond)
		if timedOut {
			out = append(out, failFromRule(rule, clean, "pattern match timed out (ReDoS guard)"))
			continue
		}
		if matched {
			out = append(out, failFromRule(rule, clean, "forbidden pattern matched"))
		}
	}
	return out
}

// matchWithTimeout runs re.Match with a soft timeout via goroutine.
// On timeout returns timedOut=true (treat as failure, not silent pass).
func matchWithTimeout(re *regexp.Regexp, data []byte, timeout time.Duration) (matched, timedOut bool) {
	done := make(chan bool, 1)
	go func() {
		done <- re.Match(data)
	}()
	select {
	case m := <-done:
		return m, false
	case <-time.After(timeout):
		return false, true
	}
}

func checkNPMDepBan(root string, rule packs.Rule) []ir.Failure {
	packagePath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(packagePath)
	if err != nil {
		return nil
	}
	var manifest ir.PackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		f := failFromRule(rule, "package.json", "invalid package.json: "+err.Error())
		f.ASTCoordinates.NodePath = "."
		return []ir.Failure{f}
	}
	checkMap := func(deps map[string]string) []ir.Failure {
		if deps == nil {
			return nil
		}
		ver, ok := deps[rule.Package]
		if !ok {
			return nil
		}
		for _, banned := range rule.BannedVersions {
			if ver == banned {
				f := failFromRule(rule, "package.json", fmt.Sprintf("%s@%s", rule.Package, ver))
				f.ASTCoordinates.NodePath = "dependencies." + rule.Package
				f.ASTCoordinates.TargetSymbol = ver
				return []ir.Failure{f}
			}
		}
		return nil
	}
	if f := checkMap(manifest.Dependencies); len(f) > 0 {
		return f
	}
	return checkMap(manifest.DevDependencies)
}

func failFromRule(rule packs.Rule, file, detail string) ir.Failure {
	desc := rule.Description
	if detail != "" {
		desc = desc + " (" + detail + ")"
	}
	target := filepath.ToSlash(file)
	if target == "" {
		target = filepath.Base(rule.Path)
	}
	return ir.Failure{
		GateID:               rule.ID,
		Severity:             rule.Severity,
		Type:                 rule.Type,
		SanitizedDescription: desc,
		ASTCoordinates:       ir.ASTCoordinates{TargetFile: target},
		Remediation: ir.Remediation{
			ActionRequired: rule.Remediation,
			ExpectedState:  rule.Expected,
		},
	}
}

func auditASTReachability(gitRoot string) []ir.Failure {
	targetFile := filepath.Join(gitRoot, "src", "payment.go")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		return nil
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, targetFile, nil, parser.ParseComments)
	if err != nil {
		return nil
	}
	found := false
	var nodePos token.Position
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "axios" && sel.Sel.Name == "Post" {
			found = true
			nodePos = fset.Position(n.Pos())
			return false
		}
		return true
	})
	if !found {
		return nil
	}
	return []ir.Failure{{
		GateID:               "CR-AST-01",
		Severity:             "high",
		Type:                 "POLICY_VIOLATION",
		SanitizedDescription: "Unsafe direct execution of vulnerable module detected via AST Inspector.",
		ASTCoordinates: ir.ASTCoordinates{
			TargetFile:    "src/payment.go",
			NodePath:      "CallExpr.SelectorExpr[axios.Post]",
			TargetSymbol:  "axios.Post",
			FallbackLines: fmt.Sprintf("Line %d", nodePos.Line),
		},
		Remediation: ir.Remediation{
			ActionRequired: "Route calls through validated wrapper function in 'safe_http.go'.",
			ExpectedState:  "No direct unmitigated function calls found in AST.",
		},
	}}
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// ActionReportMarkdown is the short QA checklist written by check / validate.
func ActionReportMarkdown(payload ir.GateFailurePayload, skipped int) string {
	var b strings.Builder
	b.WriteString("# Action Report\n\n")
	b.WriteString("> Curbpack prepares evidence for **human review**. Gate pass is not certification.\n\n")
	fmt.Fprintf(&b, "- **Packs:** %s\n", payload.PackID)
	fmt.Fprintf(&b, "- **Readiness:** %d%%\n", payload.ReadinessScore)
	fmt.Fprintf(&b, "- **Findings:** %d\n", len(payload.Failures))
	if skipped > 0 {
		fmt.Fprintf(&b, "- **Skipped (diff):** %d rules\n", skipped)
	}
	b.WriteString("\n")
	if len(payload.Failures) == 0 {
		b.WriteString("All evaluated gates passed. Open buyer one-pager / HPURL after `prepare-release` + `attest`.\n")
		return b.String()
	}
	b.WriteString("## Checklist\n\n")
	for i, f := range payload.Failures {
		if i >= 12 {
			fmt.Fprintf(&b, "\n_…and %d more — see latest_failure.json_\n", len(payload.Failures)-12)
			break
		}
		fmt.Fprintf(&b, "- [ ] **[%s]** (%s) `%s` — %s\n",
			f.GateID, f.Severity, f.ASTCoordinates.TargetFile, f.Remediation.ActionRequired)
	}
	return b.String()
}

// SemanticMarkdown renders dual-rep agent-facing markdown from a payload.
func SemanticMarkdown(payload ir.GateFailurePayload) string {
	var b strings.Builder
	parent := payload.ConcurrencyControl.ExpectedParentCommitSHA
	if len(parent) > 8 {
		parent = parent[:8]
	}
	fmt.Fprintf(&b, "# COMPLIANCE ALERT: GATE FAILURE [OCC-ID: %s:%s]\n",
		parent, payload.ConcurrencyControl.StateVersionToken)
	fmt.Fprintf(&b, "**Statechart Path:** %s\n", strings.Join(payload.StatechartContext.ActiveParentStatePath, " / "))
	fmt.Fprintf(&b, "**Failed Region:** %s\n\n", strings.Join(payload.StatechartContext.FailedOrthogonalRegions, ", "))
	for i, f := range payload.Failures {
		fmt.Fprintf(&b, "## VIOLATION %d: %s [%s] (%s)\n", i+1, f.Type, f.GateID, f.Severity)
		fmt.Fprintf(&b, "* **Location:** `%s`\n", f.ASTCoordinates.TargetFile)
		fmt.Fprintf(&b, "* **AST Path:** `%s`\n", f.ASTCoordinates.NodePath)
		fmt.Fprintf(&b, "* **Symbol Target:** `%s`\n", f.ASTCoordinates.TargetSymbol)
		b.WriteString("* **Context:**\n")
		b.WriteString("<untrusted_metadata>\n")
		b.WriteString(f.SanitizedDescription + "\n")
		b.WriteString("</untrusted_metadata>\n\n")
		b.WriteString("### REQUIRED REMEDIATION\n")
		fmt.Fprintf(&b, "* **Goal State:** %s\n", f.Remediation.ExpectedState)
		fmt.Fprintf(&b, "* **Resolution Path:** %s\n\n", f.Remediation.ActionRequired)
	}
	return b.String()
}
