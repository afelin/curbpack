package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afelin/curbpack/internal/ask"
	"github.com/afelin/curbpack/internal/attest"
	"github.com/afelin/curbpack/internal/buildinfo"
	"github.com/afelin/curbpack/internal/config"
	"github.com/afelin/curbpack/internal/demo"
	"github.com/afelin/curbpack/internal/doctor"
	"github.com/afelin/curbpack/internal/drift"
	"github.com/afelin/curbpack/internal/exportx"
	"github.com/afelin/curbpack/internal/formhints"
	"github.com/afelin/curbpack/internal/gitutil"
	"github.com/afelin/curbpack/internal/instrument"
	"github.com/afelin/curbpack/internal/ir"
	"github.com/afelin/curbpack/internal/packs"
	"github.com/afelin/curbpack/internal/packscmd"
	"github.com/afelin/curbpack/internal/release"
	"github.com/afelin/curbpack/internal/remediation"
	"github.com/afelin/curbpack/internal/sbom"
	"github.com/afelin/curbpack/internal/skilldata"
	"github.com/afelin/curbpack/internal/sock"
	"github.com/afelin/curbpack/internal/tty"
	"github.com/afelin/curbpack/internal/validate"
	"github.com/afelin/curbpack/internal/vex"
	"github.com/afelin/curbpack/internal/workflowdata"
)

// research command lives in research.go (allowlisted citation packet; never gates check).

// Version aliases buildinfo.Version for CLI surfaces.
// Release builds set buildinfo via -ldflags "-X github.com/afelin/curbpack/internal/buildinfo.Version=...".
var Version = buildinfo.Version

// Stable exit codes (document in README):
//
//	0 = pass / success
//	1 = gate failures (or operational error during check/validate)
//	2 = usage / environment (unknown command, bad flags, not a git repo when required)
const (
	ExitOK    = 0
	ExitGates = 1
	ExitUsage = 2
)

type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func usageErr(msg string) error { return &exitError{code: ExitUsage, msg: msg} }
func gatesErr() error           { return &exitError{code: ExitGates, msg: ""} }

// ExitCode maps a Run error to a process exit code.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var miss *doctor.ErrMissingBinary
	if errors.As(err, &miss) {
		return ExitUsage
	}
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	return ExitGates
}

func cmdDoctor(args []string) error {
	f, err := parseDoctorFlags(args)
	if err != nil {
		return err
	}
	return doctor.Run(doctor.Options{Version: Version, Repair: f.repair})
}

// Run dispatches CLI args (without the program name). Returns a typed exitError for usage/gates.
func Run(args []string) error {
	if len(args) == 0 {
		return cmdDefault()
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "help", "-h", "--help":
		usage()
		return nil
	case "version", "-v", "--version":
		fmt.Println("curbpack", Version)
		return nil
	default:
		if h, ok := lookupCommand(cmd); ok {
			return h(rest)
		}
		fmt.Printf("%s\n\n", tty.C(tty.Red, "Unknown command '"+cmd+"'"))
		usage()
		return usageErr("")
	}
}

// cmdDefault: bare `curbpack` → doctor if not inited, else check (one mental model).
func cmdDefault() error {
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return doctor.Run(doctor.Options{Version: Version})
	}
	cfg, err := config.Load(root)
	if err != nil {
		return usageErr(err.Error())
	}
	if cfg == nil {
		return doctor.Run(doctor.Options{Version: Version})
	}
	return cmdCheck(nil)
}

func usage() {
	fmt.Fprintf(os.Stderr, "%s\n", tty.C(tty.Bold+tty.Cyan, "Curbpack "+Version))
	fmt.Fprintf(os.Stderr, "Local evidence CLI — packs encode policy. Not a certification product.\n\n")
	fmt.Fprintf(os.Stderr, "Usage: curbpack [<command>] [args]\n")
	fmt.Fprintf(os.Stderr, "  (no command)     doctor if uninitialized, else check\n\n")
	fmt.Fprintf(os.Stderr, "Ladder:\n")
	fmt.Fprintf(os.Stderr, "  doctor [--repair] Environment confidence; --repair = local PATH/alias only\n")
	fmt.Fprintf(os.Stderr, "  demo [--open]    Sandbox check (browser only with --open)\n")
	fmt.Fprintf(os.Stderr, "  scan [--packs a,b] Read-only repo diagnosis (alias: reality-check)\n")
	fmt.Fprintf(os.Stderr, "  fix --art14      Write Art 14 rehearsal file (one file; diff preview)\n")
	fmt.Fprintf(os.Stderr, "  init [--profile house|cra|medtech] [--packs a,b] [--workflow]\n")
	fmt.Fprintf(os.Stderr, "                   Default: house-policy + hooks + skill + ide\n")
	fmt.Fprintf(os.Stderr, "  check [--heal] [--score]  Daily loop (--score shows readiness %%)\n")
	fmt.Fprintf(os.Stderr, "  ask-my-suppliers [--stdout-only] [--out path]\n")
	fmt.Fprintf(os.Stderr, "                   Supplier checklist → stdout + review-pack/ (writes files)\n")
	fmt.Fprintf(os.Stderr, "  share [--bundle] [--reveal] check → context-pack → buyer-questions → prepare-release\n")
	fmt.Fprintf(os.Stderr, "  drift [--json]   Multi-signal evidence checklist (exit 0 always)\n")
	fmt.Fprintf(os.Stderr, "  prepare-release  Review-pack + evidence\n")
	fmt.Fprintf(os.Stderr, "  attest [--allow-dirty] [--reviewed-by=Name]  Human Git Notes capsule (then proof verify)\n\n")
	fmt.Fprintf(os.Stderr, "Advanced:\n")
	fmt.Fprintf(os.Stderr, "  validate [--json] [--delta]   Dual-rep gates (--delta not release-safe)\n")
	fmt.Fprintf(os.Stderr, "  check --diff                  Delta mode — not release-gate safe\n")
	fmt.Fprintf(os.Stderr, "  ask [file] [--propose]        Explain GateFailure JSON\n")
	fmt.Fprintf(os.Stderr, "  packs list|update|import|export-graph|doctor\n")
	fmt.Fprintf(os.Stderr, "  export --sarif|--explain-packet|--watchlist-join|--buyer-questions|--lay-of-land|--context-pack [--spdx] [--slsa]\n")
	fmt.Fprintf(os.Stderr, "                                Standards / airlock / buyer checklist / instrument map / ContextPack\n")
	fmt.Fprintf(os.Stderr, "  share [--bundle] [--reveal]   Recipe + optional Explorer/Finder reveal\n")
	fmt.Fprintf(os.Stderr, "  drift [--json]                Evidence drift checklist (informational; exit 0)\n")
	fmt.Fprintf(os.Stderr, "  pathway status|suggest|confirm-packs|confirm-prose|confirm-share|note\n")
	fmt.Fprintf(os.Stderr, "                                Warm-start seed + HITL ticks + session notes (sole writer of pathway-seed.json)\n")
	fmt.Fprintf(os.Stderr, "  research [--fetch] [--cite-check <md>] [--list-sources]\n")
	fmt.Fprintf(os.Stderr, "                                Allowlisted citation packet + human brief (never gates check)\n")
	fmt.Fprintf(os.Stderr, "  completion bash|zsh|fish      Print shell completions to stdout\n")
	fmt.Fprintf(os.Stderr, "  sock                          Optional Coreward Unix IPC (macOS/Linux only)\n")
	fmt.Fprintf(os.Stderr, "  view                          Show attest capsule for HEAD\n\n")
	fmt.Fprintf(os.Stderr, "Exit codes: 0=pass  1=gates/error  2=usage/env (incl. doctor --repair missing binary)\n")
}

func cmdDemo(args []string) error {
	f, err := parseDemoFlags(args)
	if err != nil {
		return err
	}
	return demo.Run(demo.Options{KeepDir: f.keep, OutDir: f.out, Version: Version, OpenBrowser: f.openBrowser})
}

func cmdInit(args []string) error {
	tty.PrintHeader("curbpack init")
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("workspace is not a git repository")
	}
	tty.PrintStatus("Git repository", true, root)

	crPath := filepath.Join(root, ".github", "curbpack")
	_ = os.MkdirAll(filepath.Join(crPath, "policies"), 0o755)
	_ = os.MkdirAll(filepath.Join(crPath, "cache"), 0o755)
	_ = os.MkdirAll(filepath.Join(crPath, "evidence"), 0o755)

	if added, gerr := ensureCurbpackGitignore(root); gerr != nil {
		return gerr
	} else if len(added) > 0 {
		tty.PrintStatus(".gitignore", true, "cache/evidence ignored ("+strings.Join(added, ", ")+")")
	} else {
		tty.PrintStatus(".gitignore", true, "cache/evidence already ignored")
	}

	iflags, err := parseInitFlags(args)
	if err != nil {
		return err
	}
	packList := iflags.packList
	hooks := iflags.hooks
	skill := iflags.skill
	ide := iflags.ide
	writeWorkflow := iflags.writeWorkflow
	if iflags.showMedtechWarn {
		fmt.Printf("%s\n", tty.C(tty.Yellow, "[!] --medtech is deprecated; prefer --profile medtech or --packs medtech-iec62304"))
	}
	_ = iflags.explicitProfile // reserved for future init messaging

	for _, id := range packList {
		if _, err := packs.LoadPack(id); err != nil {
			return err
		}
	}

	cfg := config.File{
		Packs:   packList,
		Hooks:   hooks,
		Version: Version,
		Claim:   "Prepares evidence for human review — not a conformity assessment.",
	}
	cfgPath := filepath.Join(root, ".curbpack.json")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := config.Write(root, cfg); err != nil {
			return err
		}
		tty.PrintStatus(".curbpack.json", true, "created packs="+strings.Join(packList, ","))
	} else {
		tty.PrintStatus(".curbpack.json", true, "exists (not overwritten)")
	}

	paths, err := packs.ScaffoldPaths(packList)
	if err != nil {
		return err
	}
	for _, rel := range paths {
		p, clean, err := validate.SafeJoin(root, rel)
		if err != nil {
			return fmt.Errorf("scaffold path refused: %s: %w", rel, err)
		}
		if _, err := os.Stat(p); err == nil {
			tty.PrintStatus("stub "+clean, true, "found")
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		body := packs.DefaultScaffoldBody(clean)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			return err
		}
		tty.PrintStatus("stub "+clean, true, "created")
	}

	_ = os.MkdirAll(filepath.Join(root, "proof"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "proof", "index.html"), []byte(release.ProofPageHTML()), 0o644)
	tty.PrintStatus("proof/index.html", true, "HPURL viewer + verify")

	if hooks {
		if err := installPreCommitHook(root); err != nil {
			return err
		}
		tty.PrintStatus("pre-commit hook", true, "curbpack check --heal")
	}

	if skill {
		dest, err := skilldata.Install(root)
		if err != nil {
			return err
		}
		tty.PrintStatus("Cursor skill", true, dest)
	}

	if ide {
		dest, err := skilldata.WriteIDETasks(root)
		if err != nil {
			return err
		}
		tty.PrintStatus("VS Code tasks", true, dest)
	}

	if writeWorkflow {
		dest, created, err := workflowdata.Install(root)
		if err != nil {
			return err
		}
		if created {
			tty.PrintStatus("Action workflow", true, dest+" (created)")
		} else {
			tty.PrintStatus("Action workflow", true, dest+" (exists, not overwritten)")
		}
	}

	fmt.Printf("\n%s\n", tty.C(tty.Bold+tty.Green, "[+] Init complete. Next: curbpack check"))
	fmt.Printf("%s\n", tty.C(tty.Dim, "Prepares evidence for human review — not a conformity assessment."))
	return nil
}

func appendUnique(in []string, id string) []string {
	for _, x := range in {
		if x == id {
			return in
		}
	}
	return append(in, id)
}

func profilePacks(name string) []string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "house":
		return []string{"house-policy"}
	case "cra":
		return []string{"cra-baseline"}
	case "medtech":
		return []string{"medtech-iec62304"}
	default:
		return []string{"house-policy"}
	}
}

func installPreCommitHook(root string) error {
	hookDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(hookDir, "pre-commit")
	// LF-only: never write CRLF (Windows Git may otherwise break sh hooks).
	script := "#!/bin/sh\n" +
		"# Curbpack — fail commit on high/critical gate findings\n" +
		"# --heal: create missing stubs only (never overwrite filled docs; never attest)\n" +
		"# Hooks enabled ⇒ missing binary is fail-closed (no silent skip).\n" +
		"if command -v curbpack >/dev/null 2>&1; then\n" +
		"  exec curbpack check --heal\n" +
		"elif [ -x ./bin/curbpack ]; then\n" +
		"  exec ./bin/curbpack check --heal\n" +
		"elif [ -x ./curbpack ]; then\n" +
		"  exec ./curbpack check --heal\n" +
		"else\n" +
		"  echo \"curbpack not on PATH — refusing commit (hooks enabled)\" >&2\n" +
		"  exit 1\n" +
		"fi\n"
	if strings.Contains(script, "\r") {
		return fmt.Errorf("internal: hook script must be LF-only")
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return err
	}
	if cfg, err := config.Load(root); err == nil && cfg != nil {
		cfg.Hooks = true
		_ = config.Write(root, *cfg)
	}
	return nil
}

func parseCheckFlags(args []string) (packIDs []string, jsonOut, diffOnly, formHints, applyStub, heal, showScore bool, err error) {
	f, err := parseCheckValidateFlags("check", args)
	if err != nil {
		return nil, false, false, false, false, false, false, err
	}
	return f.packIDs, f.jsonOut, f.diffOnly, f.formHints, f.applyStub, f.heal, f.showScore, nil
}

func parseValidateFlags(args []string) (packIDs []string, jsonOut, diffOnly, formHints, applyStub, heal bool, err error) {
	f, err := parseCheckValidateFlags("validate", args)
	if err != nil {
		return nil, false, false, false, false, false, err
	}
	return f.packIDs, f.jsonOut, f.diffOnly, f.formHints, f.applyStub, f.heal, nil
}

const healMaxRounds = 3

func cmdCheck(args []string) error {
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("must run inside a git repository")
	}
	packIDs, jsonOut, diffOnly, wantHints, applyStub, heal, showScore, err := parseCheckFlags(args)
	if err != nil {
		return err
	}

	// Snapshot prior evidence deposit before validate overwrites cache.
	prior := loadPriorCache(root)
	priorInst, priorInstOK := instrument.Load(root)

	if !jsonOut {
		tty.PrintHeader("CURBPACK CHECK")
	}

	var res validate.Result
	var lastHints []formhints.Hint
	checkDiff := diffOnly
	stubsWritten := 0
	freshStubs := map[string]struct{}{}
	for round := 0; round <= healMaxRounds; round++ {
		res, err = validate.Run(validate.Options{
			RepoRoot:       root,
			PackIDs:        packIDs,
			DiffOnly:       checkDiff,
			Quiet:          jsonOut,
			FreshStubPaths: freshStubs,
		})
		if err != nil {
			return err
		}
		if res.Passed || !heal || round == healMaxRounds {
			break
		}
		// Heal: form-hints → apply-stub (missing only) → persist cache → re-check.
		// Never auto-attest / never invent approved legal prose.
		cache, _ := remediation.Load(root)
		hints := formhints.ForFailuresCached(res.Payload.Failures, cache)
		hints, err = formhints.ApplyStubs(root, hints)
		if err != nil {
			return err
		}
		_ = formhints.PersistCache(root, hints)
		lastHints = hints
		applied := 0
		for _, h := range hints {
			if h.Applied {
				applied++
				freshStubs[filepath.ToSlash(h.File)] = struct{}{}
			}
		}
		stubsWritten += applied
		if !jsonOut {
			fmt.Printf("%s\n", tty.C(tty.Yellow, fmt.Sprintf("heal round %d/%d: applied %d missing stub(s); re-checking…", round+1, healMaxRounds, applied)))
		}
		if applied == 0 {
			break // nothing more heal can write
		}
		// After any stub write, force a full scan so --diff cannot false-green remaining failures.
		checkDiff = false
	}

	// Instrument map: always refresh after a successful validate write path.
	nowInst := instrument.Compute(root)
	_ = instrument.Write(root, nowInst)

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res.Payload)
	} else if res.Passed {
		// Green: optional thermometer + claim + optional accumulation / instrument whispers.
		if showScore {
			if tty.IsTerminal {
				tty.RenderThermometer(res.Score)
			} else {
				fmt.Printf("readiness=%d%% gates=green\n", res.Score)
			}
		}
		if heal && stubsWritten > 0 {
			fmt.Printf("%s\n", tty.C(tty.Bold+tty.Yellow, "[!] scaffold green ≠ readiness / not certification — heal wrote missing stubs only"))
		}
		fmt.Printf("%s\n", tty.C(tty.Dim, "Prepares evidence for human review — not a conformity assessment."))
		fmt.Printf("%s\n", tty.C(tty.Dim, instrumentPanelCovenant))
		for _, line := range instrumentWhisperLines(prior, priorInst, priorInstOK, res.Score, nowInst) {
			fmt.Printf("%s\n", tty.C(tty.Dim, line))
		}
		if line := drift.BindDriftLine(root); line != "" {
			fmt.Printf("%s\n", tty.C(tty.Dim, line))
		}
	} else {
		if showScore {
			if tty.IsTerminal {
				tty.RenderThermometer(res.Score)
			} else {
				fmt.Printf("readiness=%d%% gates=open\n", res.Score)
			}
		}
		if heal && stubsWritten > 0 {
			fmt.Printf("%s\n", tty.C(tty.Bold+tty.Yellow, "[!] scaffold green ≠ readiness / not certification — heal wrote missing stubs only"))
		}
		// Failures: top 3 + ask pointer (no log archaeology).
		fmt.Println(topFailuresSummary(res, 3))
		fmt.Printf("%s\n", tty.C(tty.Dim, "ask: curbpack ask .github/curbpack/cache/latest_failure.json --propose"))
		fmt.Printf("%s\n", tty.C(tty.Dim, "cache: .github/curbpack/cache/latest_*.json"))
		fmt.Printf("%s\n", tty.C(tty.Dim, "Prepares evidence for human review — not a conformity assessment."))
		fmt.Printf("%s\n", tty.C(tty.Dim, instrumentPanelCovenant))
	}

	if wantHints && !jsonOut && (len(res.Payload.Failures) > 0 || len(lastHints) > 0) {
		hints := lastHints
		if len(hints) == 0 && len(res.Payload.Failures) > 0 {
			cache, _ := remediation.Load(root)
			hints = formhints.ForFailuresCached(res.Payload.Failures, cache)
			if applyStub && !heal {
				hints, err = formhints.ApplyStubs(root, hints)
				if err != nil {
					return err
				}
				_ = formhints.PersistCache(root, hints)
			}
		}
		fmt.Println()
		fmt.Print(formhints.Format(hints))
	}

	if !res.Passed {
		return gatesErr()
	}
	return nil
}

func topFailuresSummary(res validate.Result, n int) string {
	var b strings.Builder
	b.WriteString(res.ActionReport)
	if len(res.Payload.Failures) == 0 {
		return b.String()
	}
	b.WriteString("\n## Top findings\n\n")
	for i, f := range res.Payload.Failures {
		if i >= n {
			fmt.Fprintf(&b, "\n_…and %d more — see latest_failure.json_\n", len(res.Payload.Failures)-n)
			break
		}
		fmt.Fprintf(&b, "%d. `%s` (%s) %s\n", i+1, f.GateID, f.Severity, f.SanitizedDescription)
	}
	return b.String()
}

func cmdValidate(args []string) error {
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("must run inside a git repository")
	}
	packIDs, jsonOut, diffOnly, _, _, _, err := parseValidateFlags(args)
	if err != nil {
		return err
	}
	if !jsonOut {
		tty.PrintHeader("EXECUTING COMPLIANCE GATES")
	}
	res, err := validate.Run(validate.Options{RepoRoot: root, PackIDs: packIDs, DiffOnly: diffOnly, Quiet: jsonOut})
	if err != nil {
		return err
	}
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res.Payload)
	} else {
		if tty.IsTerminal {
			tty.RenderThermometer(res.Score)
		}
		if !res.Passed {
			fmt.Printf("\n%s\n", tty.C(tty.Bold+tty.Magenta, "--- DUAL-REPRESENTATION OUTPUT ---"))
			fmt.Println(validate.SemanticMarkdown(res.Payload))
		} else {
			fmt.Printf("\n%s\n", tty.C(tty.BGGreen, "[✔] ALL DETERMINISTIC GATES PASSED"))
		}
	}
	if !res.Passed {
		return gatesErr()
	}
	return nil
}

func cmdPrepareRelease(args []string) error {
	tty.PrintHeader("PREPARE RELEASE REVIEW PACK")
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr(err.Error())
	}
	f, err := parsePrepareReleaseFlags(args)
	if err != nil {
		return err
	}
	return release.Prepare(release.Options{
		RepoRoot:          root,
		PackIDs:           f.packIDs,
		OutDir:            f.out,
		AllowFailingGates: f.allowFailing,
	})
}

func cmdPacks(args []string) error {
	if len(args) == 0 {
		return packscmd.List()
	}
	switch args[0] {
	case "list":
		return packscmd.List()
	case "update":
		return packscmd.UpdateStub()
	case "import":
		src := ""
		if len(args) > 1 {
			src = args[1]
		}
		return packscmd.ImportAirGap(src)
	case "export-graph":
		return packscmd.ExportGraph(args[1:])
	case "doctor":
		return packscmd.Doctor()
	default:
		return usageErr(fmt.Sprintf("unknown packs subcommand %q (list|update|import|export-graph|doctor)", args[0]))
	}
}

func cmdAskMySuppliers(args []string) error {
	f, err := parseAskMySuppliersFlags(args)
	if err != nil {
		return err
	}
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("must run inside a git repository")
	}

	report, err := exportx.BuildBuyerQuestionsReportReadOnly(root, f.packIDs)
	if err != nil {
		return err
	}

	tty.PrintHeader("curbpack ask-my-suppliers")
	fmt.Printf("%s\n", tty.C(tty.Bold+tty.Yellow, "Writes review-pack/supplier-checklist.md — unlike scan, this command creates files. Not conformity assessment."))
	fmt.Printf("%s\n\n", tty.C(tty.Dim, "Copy the checklist and email below to suppliers — not CE / not notified-body."))

	fmt.Print(exportx.FormatBuyerQuestionsMarkdown(report))
	fmt.Println("---")
	fmt.Print(exportx.FormatSupplierEmailTemplate(report))

	if f.stdoutOnly {
		return nil
	}

	path, n, err := exportx.WriteSupplierChecklistReport(root, report, f.out)
	if err != nil {
		return err
	}
	fmt.Printf("\n%s\n", tty.C(tty.Dim, fmt.Sprintf("Wrote %s (%d questions)", path, n)))
	return nil
}

func cmdExport(args []string) error {
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("must run inside a git repository")
	}
	f, err := parseExportFlags(args)
	if err != nil {
		return err
	}
	usedOut := false
	takeOut := func() string {
		if usedOut || f.out == "" {
			return ""
		}
		usedOut = true
		return f.out
	}
	if f.wantSARIF {
		path, n, err := exportx.WriteSARIF(root, f.packIDs, takeOut())
		if err != nil {
			return err
		}
		tty.PrintStatus("SARIF", true, fmt.Sprintf("%s results=%d", path, n))
	}
	if f.wantJoin {
		path, err := exportx.WriteWatchlistJoin(root, takeOut())
		if err != nil {
			return err
		}
		tty.PrintStatus("watchlist∩SBOM", true, path)
	}
	if f.wantExplain {
		path, err := exportx.WriteExplainPacket(root, f.packIDs, takeOut())
		if err != nil {
			return err
		}
		data, _ := os.ReadFile(path)
		if err := exportx.PacketLooksAirlocked(data); err != nil {
			return err
		}
		tty.PrintStatus("explain-packet", true, path)
		fmt.Printf("%s\n", tty.C(tty.Dim, "CURBPACK_EXPLAIN_ALLOW_CLOUD=0 by default — set =1 only for explicit cloud tutor export"))
	}
	if f.wantBuyerQ {
		path, n, err := exportx.WriteBuyerQuestions(root, f.packIDs, takeOut())
		if err != nil {
			return err
		}
		tty.PrintStatus("buyer-questions", true, fmt.Sprintf("%s questions=%d", path, n))
	}
	if f.wantLayOfLand {
		path, err := exportx.WriteLayOfLand(root, takeOut())
		if err != nil {
			return err
		}
		tty.PrintStatus("lay-of-land", true, path)
	}
	if f.wantContextPack {
		path, err := exportx.WriteContextPack(root, f.packIDs, takeOut())
		if err != nil {
			return err
		}
		data, _ := os.ReadFile(path)
		if err := exportx.PacketLooksAirlocked(data); err != nil {
			return err
		}
		tty.PrintStatus("context-pack", true, path)
	}
	if f.wantSPDX {
		path, err := exportx.WriteSPDXOptional(root, "")
		if err != nil {
			return err
		}
		tty.PrintStatus("SPDX (optional)", true, path)
	}
	if f.wantSLSA {
		path, err := exportx.WriteSLSAOptional(root, "")
		if err != nil {
			return err
		}
		tty.PrintStatus("SLSA sidecar (optional)", true, path)
	}
	return nil
}

func cmdAsk(args []string) error {
	f, err := parseAskFlags(args)
	if err != nil {
		return err
	}
	return ask.Run(f.path, f.propose)
}

func cmdView(_ []string) error {
	return attest.View("")
}

func cmdAttest(args []string) error {
	tty.PrintHeader("curbpack attest")
	f, err := parseAttestFlags(args)
	if err != nil {
		return err
	}
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr(err.Error())
	}
	if !f.allowDirty && gitutil.IsDirty(root) {
		return usageErr("OCC conflict: working directory has uncommitted files (pass --allow-dirty to bind digests of uncommitted evidence)")
	}

	// Best-effort SBOM: missing lockfile/manifest is OK (empty digest). I/O failures are not.
	if _, _, werr := sbom.WriteCycloneDX(root, ""); werr != nil && !sbom.IsUnavailable(werr) {
		return fmt.Errorf("SBOM write failed while binding digests: %w", werr)
	}
	res, verr := validate.Run(validate.Options{RepoRoot: root, Quiet: true})
	if verr != nil {
		return fmt.Errorf("VEX evidence: validate failed while binding digests: %w", verr)
	}
	doc := vex.FromGateFailures(filepath.Base(root), res.Payload)
	if _, werr := vex.Write(root, doc, ""); werr != nil {
		return fmt.Errorf("VEX write failed while binding digests: %w", werr)
	}
	resultDigest := ir.ComputeResultDigest(res.Payload)
	packID := res.Payload.PackID

	// Self-written evidence dirties the tree — require explicit --allow-dirty (no silent force).
	_, err = attest.Run(attest.Options{
		RepoRoot:     root,
		AllowDirty:   f.allowDirty,
		ReviewedBy:   f.reviewedBy,
		ResultDigest: resultDigest,
		PackIDs:      packID,
	})
	return err
}

func cmdSock(args []string) error {
	f, err := parseSockFlags(args)
	if err != nil {
		return err
	}
	root := f.root
	if root == "" {
		var err error
		root, err = gitutil.RepoRoot("")
		if err != nil {
			root, _ = os.Getwd()
		}
	}
	return sock.Serve(f.path, root)
}
