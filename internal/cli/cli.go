package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afelin/cyberready/internal/ask"
	"github.com/afelin/cyberready/internal/attest"
	"github.com/afelin/cyberready/internal/buildinfo"
	"github.com/afelin/cyberready/internal/config"
	"github.com/afelin/cyberready/internal/demo"
	"github.com/afelin/cyberready/internal/doctor"
	"github.com/afelin/cyberready/internal/exportx"
	"github.com/afelin/cyberready/internal/formhints"
	"github.com/afelin/cyberready/internal/gitutil"
	"github.com/afelin/cyberready/internal/packs"
	"github.com/afelin/cyberready/internal/packscmd"
	"github.com/afelin/cyberready/internal/release"
	"github.com/afelin/cyberready/internal/remediation"
	"github.com/afelin/cyberready/internal/sbom"
	"github.com/afelin/cyberready/internal/skilldata"
	"github.com/afelin/cyberready/internal/sock"
	"github.com/afelin/cyberready/internal/tty"
	"github.com/afelin/cyberready/internal/validate"
	"github.com/afelin/cyberready/internal/vex"
	"github.com/afelin/cyberready/internal/workflowdata"
)

// Version aliases buildinfo.Version for CLI surfaces.
// Release builds set buildinfo via -ldflags "-X github.com/afelin/cyberready/internal/buildinfo.Version=...".
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
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	return ExitGates
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
		fmt.Println("cyberready", Version)
		return nil
	case "init":
		return cmdInit(rest)
	case "check":
		return cmdCheck(rest)
	case "validate":
		return cmdValidate(rest)
	case "prepare-release":
		return cmdPrepareRelease(rest)
	case "packs":
		return cmdPacks(rest)
	case "ask":
		return cmdAsk(rest)
	case "attest":
		return cmdAttest(rest)
	case "view":
		return attest.View("")
	case "sock":
		return cmdSock(rest)
	case "doctor":
		return doctor.Run(doctor.Options{Version: Version})
	case "demo":
		return cmdDemo(rest)
	case "export":
		return cmdExport(rest)
	default:
		fmt.Printf("%s\n\n", tty.C(tty.Red, "Unknown command '"+cmd+"'"))
		usage()
		return usageErr("")
	}
}

// cmdDefault: bare `cyberready` → doctor if not inited, else check (one mental model).
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
	fmt.Fprintf(os.Stderr, "%s\n", tty.C(tty.Bold+tty.Cyan, "CyberReady+ "+Version))
	fmt.Fprintf(os.Stderr, "Local evidence CLI — packs encode policy. Not a certification product.\n\n")
	fmt.Fprintf(os.Stderr, "Usage: cyberready [<command>] [args]\n")
	fmt.Fprintf(os.Stderr, "  (no command)     doctor if uninitialized, else check\n\n")
	fmt.Fprintf(os.Stderr, "Ladder:\n")
	fmt.Fprintf(os.Stderr, "  doctor           Environment confidence\n")
	fmt.Fprintf(os.Stderr, "  demo [--open]    Sandbox check (browser only with --open)\n")
	fmt.Fprintf(os.Stderr, "  init [--bare] [--packs a,b] [--workflow]\n")
	fmt.Fprintf(os.Stderr, "                   Default: house-policy + hooks + skill + ide\n")
	fmt.Fprintf(os.Stderr, "                   --workflow: write .github/workflows/cyberready.yml if missing\n")
	fmt.Fprintf(os.Stderr, "  check [--heal]   Daily loop\n")
	fmt.Fprintf(os.Stderr, "  prepare-release  Review-pack + evidence\n")
	fmt.Fprintf(os.Stderr, "  attest           Human Git Notes capsule\n\n")
	fmt.Fprintf(os.Stderr, "Advanced:\n")
	fmt.Fprintf(os.Stderr, "  validate [--json] [--delta]   Dual-rep gates (--delta not release-safe)\n")
	fmt.Fprintf(os.Stderr, "  check --diff                  Delta mode — not release-gate safe\n")
	fmt.Fprintf(os.Stderr, "  ask [file] [--propose]        Explain GateFailure JSON\n")
	fmt.Fprintf(os.Stderr, "  packs list|update|import|export-graph|doctor\n")
	fmt.Fprintf(os.Stderr, "  export --sarif|--explain-packet|--watchlist-join [--spdx] [--slsa]\n")
	fmt.Fprintf(os.Stderr, "  sock                          Optional Coreward Unix IPC\n")
	fmt.Fprintf(os.Stderr, "  view                          Show attest capsule for HEAD\n\n")
	fmt.Fprintf(os.Stderr, "Exit codes: 0=pass  1=gates/error  2=usage/env\n")
}

func cmdDemo(args []string) error {
	keep := false
	openBrowser := false
	out := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--keep":
			keep = true
		case "--open":
			openBrowser = true
		case "--out":
			if i+1 < len(args) {
				out = args[i+1]
				i++
			}
		}
	}
	return demo.Run(demo.Options{KeepDir: keep, OutDir: out, Version: Version, OpenBrowser: openBrowser})
}

func cmdInit(args []string) error {
	tty.PrintHeader("cyberready init")
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("workspace is not a git repository")
	}
	tty.PrintStatus("Git repository", true, root)

	crPath := filepath.Join(root, ".github", "cyberready")
	_ = os.MkdirAll(filepath.Join(crPath, "policies"), 0o755)
	_ = os.MkdirAll(filepath.Join(crPath, "cache"), 0o755)
	_ = os.MkdirAll(filepath.Join(crPath, "evidence"), 0o755)

	// Opinionated cold start: house-policy + hooks/skill/ide unless --bare.
	// Workflow is opt-in (--workflow) to avoid surprising CI commits.
	packList := []string{"house-policy"}
	hooks := true
	skill := true
	ide := true
	writeWorkflow := false
	explicitPacks := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--bare":
			hooks = false
			skill = false
			ide = false
			writeWorkflow = false
		case a == "--packs" && i+1 < len(args):
			packList = config.ParsePacksFlag(args[i+1])
			explicitPacks = true
			i++
		case strings.HasPrefix(a, "--packs="):
			packList = config.ParsePacksFlag(strings.TrimPrefix(a, "--packs="))
			explicitPacks = true
		case a == "--medtech":
			if !explicitPacks {
				packList = []string{"medtech-iec62304"}
			} else {
				packList = appendUnique(packList, "medtech-iec62304")
			}
			fmt.Printf("%s\n", tty.C(tty.Yellow, "[!] --medtech is deprecated; prefer --packs medtech-iec62304 (extends cra-baseline)"))
		case a == "--hooks":
			hooks = true
		case a == "--skill":
			skill = true
		case a == "--ide":
			ide = true
		case a == "--workflow":
			writeWorkflow = true
		case a == "--no-hooks":
			hooks = false
		case a == "--no-skill":
			skill = false
		case a == "--no-ide":
			ide = false
		}
	}
	if len(packList) == 0 {
		packList = []string{"house-policy"}
	}

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
	cfgPath := filepath.Join(root, ".cyberready.json")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := config.Write(root, cfg); err != nil {
			return err
		}
		tty.PrintStatus(".cyberready.json", true, "created packs="+strings.Join(packList, ","))
	} else {
		tty.PrintStatus(".cyberready.json", true, "exists (not overwritten)")
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
		tty.PrintStatus("pre-commit hook", true, "cyberready check --heal")
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

	fmt.Printf("\n%s\n", tty.C(tty.Bold+tty.Green, "[+] Init complete. Next: cyberready check"))
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

func installPreCommitHook(root string) error {
	hookDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(hookDir, "pre-commit")
	script := `#!/bin/sh
# CyberReady — fail commit on high/critical gate findings
# --heal: create missing stubs only (never overwrite filled docs; never attest)
# Hooks enabled ⇒ missing binary is fail-closed (no silent skip).
if command -v cyberready >/dev/null 2>&1; then
  exec cyberready check --heal
elif [ -x ./bin/cyberready ]; then
  exec ./bin/cyberready check --heal
elif [ -x ./cyberready ]; then
  exec ./cyberready check --heal
else
  echo "cyberready not on PATH — refusing commit (hooks enabled)" >&2
  exit 1
fi
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return err
	}
	if cfg, err := config.Load(root); err == nil && cfg != nil {
		cfg.Hooks = true
		_ = config.Write(root, *cfg)
	}
	return nil
}

func parseValidateFlags(args []string) (packIDs []string, jsonOut, diffOnly, formHints, applyStub, heal bool) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--pack":
			if i+1 < len(args) {
				packIDs = append(packIDs, args[i+1])
				i++
			}
		case "--packs":
			if i+1 < len(args) {
				packIDs = append(packIDs, config.ParsePacksFlag(args[i+1])...)
				i++
			}
		case "--json":
			jsonOut = true
		case "--diff", "--delta":
			diffOnly = true
		case "--form-hints":
			formHints = true
		case "--apply-stub":
			applyStub = true
			formHints = true // apply implies show hints
		case "--heal":
			heal = true
			applyStub = true
			formHints = true
		}
	}
	return packIDs, jsonOut, diffOnly, formHints, applyStub, heal
}

const healMaxRounds = 3

func cmdCheck(args []string) error {
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("must run inside a git repository")
	}
	packIDs, jsonOut, diffOnly, wantHints, applyStub, heal := parseValidateFlags(args)

	// Snapshot prior evidence deposit before validate overwrites cache.
	prior := loadPriorCache(root)

	if !jsonOut {
		tty.PrintHeader("CYBERREADY CHECK")
	}

	var res validate.Result
	var lastHints []formhints.Hint
	checkDiff := diffOnly
	for round := 0; round <= healMaxRounds; round++ {
		res, err = validate.Run(validate.Options{RepoRoot: root, PackIDs: packIDs, DiffOnly: checkDiff, Quiet: jsonOut})
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
			}
		}
		if !jsonOut {
			fmt.Printf("%s\n", tty.C(tty.Yellow, fmt.Sprintf("heal round %d/%d: applied %d missing stub(s); re-checking…", round+1, healMaxRounds, applied)))
		}
		if applied == 0 {
			break // nothing more heal can write
		}
		// After any stub write, force a full scan so --diff cannot false-green remaining failures.
		checkDiff = false
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res.Payload)
	} else if res.Passed {
		// Green: thermometer + claim + optional one-line accumulation whisper.
		if tty.IsTerminal {
			tty.RenderThermometer(res.Score)
		} else {
			fmt.Printf("readiness=%d%% gates=green\n", res.Score)
		}
		fmt.Printf("%s\n", tty.C(tty.Dim, "Prepares evidence for human review — not a conformity assessment."))
		if line := accumulationDeltaLine(prior, res.Score); line != "" {
			fmt.Printf("%s\n", tty.C(tty.Dim, line))
		}
	} else {
		if tty.IsTerminal {
			tty.RenderThermometer(res.Score)
		}
		// Failures: top 3 + ask pointer (no log archaeology).
		fmt.Println(topFailuresSummary(res, 3))
		fmt.Printf("%s\n", tty.C(tty.Dim, "ask: cyberready ask .github/cyberready/cache/latest_failure.json --propose"))
		fmt.Printf("%s\n", tty.C(tty.Dim, "cache: .github/cyberready/cache/latest_*.json"))
		fmt.Printf("%s\n", tty.C(tty.Dim, "Prepares evidence for human review — not a conformity assessment."))
	}

	if wantHints && (len(res.Payload.Failures) > 0 || len(lastHints) > 0) {
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
	packIDs, jsonOut, diffOnly, _, _, _ := parseValidateFlags(args)
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
	var packIDs []string
	out := ""
	allowFailing := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--pack":
			if i+1 < len(args) {
				packIDs = append(packIDs, args[i+1])
				i++
			}
		case "--packs":
			if i+1 < len(args) {
				packIDs = append(packIDs, config.ParsePacksFlag(args[i+1])...)
				i++
			}
		case "--out":
			if i+1 < len(args) {
				out = args[i+1]
				i++
			}
		case "--allow-failing-gates":
			allowFailing = true
		}
	}
	return release.Prepare(release.Options{
		RepoRoot:          root,
		PackIDs:           packIDs,
		OutDir:            out,
		AllowFailingGates: allowFailing,
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

func cmdExport(args []string) error {
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("must run inside a git repository")
	}
	var packIDs []string
	out := ""
	wantSARIF := false
	wantExplain := false
	wantJoin := false
	wantSPDX := false
	wantSLSA := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--sarif":
			wantSARIF = true
		case "--explain-packet":
			wantExplain = true
		case "--watchlist-join":
			wantJoin = true
		case "--spdx":
			wantSPDX = true
		case "--slsa":
			wantSLSA = true
		case "--out":
			if i+1 < len(args) {
				out = args[i+1]
				i++
			}
		case "--packs":
			if i+1 < len(args) {
				packIDs = append(packIDs, config.ParsePacksFlag(args[i+1])...)
				i++
			}
		}
	}
	if !wantSARIF && !wantExplain && !wantJoin && !wantSPDX && !wantSLSA {
		return usageErr("export requires --sarif, --explain-packet, --watchlist-join, and/or --spdx/--slsa")
	}
	if wantSARIF {
		path, n, err := exportx.WriteSARIF(root, packIDs, out)
		if err != nil {
			return err
		}
		tty.PrintStatus("SARIF", true, fmt.Sprintf("%s results=%d", path, n))
		out = "" // don't reuse path for subsequent exporters
	}
	if wantJoin {
		path, err := exportx.WriteWatchlistJoin(root, ternary(wantSARIF || wantExplain, "", out))
		if err != nil {
			return err
		}
		tty.PrintStatus("watchlist∩SBOM", true, path)
	}
	if wantExplain {
		path, err := exportx.WriteExplainPacket(root, packIDs, ternary(wantSARIF || wantJoin, "", out))
		if err != nil {
			return err
		}
		data, _ := os.ReadFile(path)
		if err := exportx.PacketLooksAirlocked(data); err != nil {
			return err
		}
		tty.PrintStatus("explain-packet", true, path)
		fmt.Printf("%s\n", tty.C(tty.Dim, "CYBERREADY_EXPLAIN_ALLOW_CLOUD=0 by default — set =1 only for explicit cloud tutor export"))
	}
	if wantSPDX {
		path, err := exportx.WriteSPDXOptional(root, "")
		if err != nil {
			return err
		}
		tty.PrintStatus("SPDX (optional)", true, path)
	}
	if wantSLSA {
		path, err := exportx.WriteSLSAOptional(root, "")
		if err != nil {
			return err
		}
		tty.PrintStatus("SLSA sidecar (optional)", true, path)
	}
	return nil
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func cmdAsk(args []string) error {
	propose := false
	path := ""
	for _, a := range args {
		if a == "--propose" {
			propose = true
			continue
		}
		if !strings.HasPrefix(a, "-") {
			path = a
		}
	}
	return ask.Run(path, propose)
}

func cmdAttest(args []string) error {
	tty.PrintHeader("cyberready attest")
	allowDirty := false
	for _, a := range args {
		if a == "--allow-dirty" {
			allowDirty = true
		}
	}
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr(err.Error())
	}
	if !allowDirty && gitutil.IsDirty(root) {
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

	// Self-written evidence dirties the tree — require explicit --allow-dirty (no silent force).
	_, err = attest.Run(attest.Options{RepoRoot: root, AllowDirty: allowDirty})
	return err
}

func cmdSock(args []string) error {
	path := ""
	root := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 < len(args) {
				path = args[i+1]
				i++
			}
		case "--repo":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		}
	}
	if root == "" {
		var err error
		root, err = gitutil.RepoRoot("")
		if err != nil {
			root, _ = os.Getwd()
		}
	}
	return sock.Serve(path, root)
}
