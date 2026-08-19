package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/afelin/curbpack/internal/config"
)

func newCommandFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})
	return fs
}

func flagUsageErr(cmd, detail string) error {
	if detail == "" {
		return usageErr(cmd + ": invalid flags")
	}
	return usageErr(cmd + ": " + detail)
}

type checkValidateFlags struct {
	packIDs   []string
	jsonOut   bool
	diffOnly  bool
	formHints bool
	applyStub bool
	heal      bool
	showScore bool
}

func parseCheckValidateFlags(cmd string, args []string) (checkValidateFlags, error) {
	fs := newCommandFlagSet(cmd)
	var f checkValidateFlags
	var packsFlag, packSingle string
	fs.BoolVar(&f.jsonOut, "json", false, "")
	fs.BoolVar(&f.diffOnly, "diff", false, "")
	fs.BoolVar(&f.diffOnly, "delta", false, "")
	fs.BoolVar(&f.formHints, "form-hints", false, "")
	fs.BoolVar(&f.applyStub, "apply-stub", false, "")
	fs.BoolVar(&f.heal, "heal", false, "")
	fs.BoolVar(&f.showScore, "score", false, "")
	fs.StringVar(&packsFlag, "packs", "", "")
	fs.StringVar(&packSingle, "pack", "", "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr(cmd, err.Error())
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("%s: unknown argument %q", cmd, fs.Arg(0)))
	}
	if packsFlag != "" {
		f.packIDs = config.ParsePacksFlag(packsFlag)
	}
	if packSingle != "" {
		f.packIDs = append(f.packIDs, packSingle)
	}
	if f.heal {
		f.applyStub, f.formHints = true, true
	}
	if f.applyStub {
		f.formHints = true
	}
	return f, nil
}

type doctorFlags struct{ repair bool }

func parseDoctorFlags(args []string) (doctorFlags, error) {
	var f doctorFlags
	for _, a := range args {
		switch a {
		case "--repair":
			f.repair = true
		case "-h", "--help":
			fmt.Fprintf(os.Stderr, "Usage: curbpack doctor [--repair]\n  --repair  Re-assert install dir on PATH + refresh curb alias (local only; never downloads)\n")
			return f, nil
		default:
			return f, usageErr(fmt.Sprintf("doctor: unknown flag %q (use --repair)", a))
		}
	}
	return f, nil
}

type demoFlags struct {
	keep, openBrowser bool
	out               string
}

func parseDemoFlags(args []string) (demoFlags, error) {
	fs := newCommandFlagSet("demo")
	var f demoFlags
	fs.BoolVar(&f.keep, "keep", false, "")
	fs.BoolVar(&f.openBrowser, "open", false, "")
	fs.StringVar(&f.out, "out", "", "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("demo", err.Error())
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("demo: unknown argument %q", fs.Arg(0)))
	}
	return f, nil
}

type initFlags struct {
	packList                         []string
	hooks, skill, ide, writeWorkflow bool
	showMedtechWarn, explicitProfile bool
}

func parseInitFlags(args []string) (initFlags, error) {
	var f initFlags
	f.packList = []string{"house-policy"}
	f.hooks, f.skill, f.ide = true, true, true
	explicitPacks := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--bare":
			f.hooks, f.skill, f.ide, f.writeWorkflow = false, false, false, false
		case a == "--packs" && i+1 < len(args):
			f.packList = config.ParsePacksFlag(args[i+1])
			explicitPacks, i = true, i+1
		case strings.HasPrefix(a, "--packs="):
			f.packList = config.ParsePacksFlag(strings.TrimPrefix(a, "--packs="))
			explicitPacks = true
		case a == "--profile" && i+1 < len(args):
			if !explicitPacks {
				f.packList = profilePacks(args[i+1])
				f.explicitProfile = true
			}
			i++
		case strings.HasPrefix(a, "--profile="):
			if !explicitPacks {
				f.packList = profilePacks(strings.TrimPrefix(a, "--profile="))
				f.explicitProfile = true
			}
		case a == "--medtech":
			if !explicitPacks {
				f.packList = profilePacks("medtech")
				f.explicitProfile = true
			} else {
				f.packList = appendUnique(f.packList, "medtech-iec62304")
			}
			f.showMedtechWarn = true
		case a == "--hooks":
			f.hooks = true
		case a == "--skill":
			f.skill = true
		case a == "--ide":
			f.ide = true
		case a == "--workflow":
			f.writeWorkflow = true
		case a == "--no-hooks":
			f.hooks = false
		case a == "--no-skill":
			f.skill = false
		case a == "--no-ide":
			f.ide = false
		default:
			return f, usageErr(fmt.Sprintf("init: unknown flag %q", a))
		}
	}
	if len(f.packList) == 0 {
		f.packList = []string{"house-policy"}
	}
	return f, nil
}

type prepareReleaseFlags struct {
	packIDs      []string
	out          string
	allowFailing bool
}

func parsePrepareReleaseFlags(args []string) (prepareReleaseFlags, error) {
	fs := newCommandFlagSet("prepare-release")
	var f prepareReleaseFlags
	var packSingle, packsFlag string
	fs.StringVar(&f.out, "out", "", "")
	fs.BoolVar(&f.allowFailing, "allow-failing-gates", false, "")
	fs.StringVar(&packSingle, "pack", "", "")
	fs.StringVar(&packsFlag, "packs", "", "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("prepare-release", err.Error())
	}
	if packsFlag != "" {
		f.packIDs = append(f.packIDs, config.ParsePacksFlag(packsFlag)...)
	}
	if packSingle != "" {
		f.packIDs = append(f.packIDs, packSingle)
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("prepare-release: unknown argument %q", fs.Arg(0)))
	}
	return f, nil
}

type exportFlags struct {
	packIDs                                           []string
	out                                               string
	wantSARIF, wantExplain, wantJoin, wantBuyerQ        bool
	wantLayOfLand, wantContextPack, wantSPDX, wantSLSA bool
}

func parseExportFlags(args []string) (exportFlags, error) {
	fs := newCommandFlagSet("export")
	var f exportFlags
	var packsFlag string
	fs.BoolVar(&f.wantSARIF, "sarif", false, "")
	fs.BoolVar(&f.wantExplain, "explain-packet", false, "")
	fs.BoolVar(&f.wantJoin, "watchlist-join", false, "")
	fs.BoolVar(&f.wantBuyerQ, "buyer-questions", false, "")
	fs.BoolVar(&f.wantLayOfLand, "lay-of-land", false, "")
	fs.BoolVar(&f.wantContextPack, "context-pack", false, "")
	fs.BoolVar(&f.wantSPDX, "spdx", false, "")
	fs.BoolVar(&f.wantSLSA, "slsa", false, "")
	fs.StringVar(&f.out, "out", "", "")
	fs.StringVar(&packsFlag, "packs", "", "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("export", err.Error())
	}
	if packsFlag != "" {
		f.packIDs = config.ParsePacksFlag(packsFlag)
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("export: unknown argument %q", fs.Arg(0)))
	}
	if !f.wantSARIF && !f.wantExplain && !f.wantJoin && !f.wantBuyerQ && !f.wantLayOfLand && !f.wantContextPack && !f.wantSPDX && !f.wantSLSA {
		return f, usageErr("export requires --sarif, --explain-packet, --watchlist-join, --buyer-questions, --lay-of-land, --context-pack, and/or --spdx/--slsa")
	}
	return f, nil
}

type askMySuppliersFlags struct {
	packIDs    []string
	out        string
	stdoutOnly bool
}

func parseAskMySuppliersFlags(args []string) (askMySuppliersFlags, error) {
	fs := newCommandFlagSet("ask-my-suppliers")
	var f askMySuppliersFlags
	var packsFlag string
	fs.BoolVar(&f.stdoutOnly, "stdout-only", false, "")
	fs.StringVar(&f.out, "out", "", "")
	fs.StringVar(&packsFlag, "packs", "", "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("ask-my-suppliers", err.Error())
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("ask-my-suppliers: unknown argument %q", fs.Arg(0)))
	}
	if packsFlag != "" {
		f.packIDs = config.ParsePacksFlag(packsFlag)
	}
	return f, nil
}

type askFlags struct{ path string; propose bool }

func parseAskFlags(args []string) (askFlags, error) {
	fs := newCommandFlagSet("ask")
	var f askFlags
	fs.BoolVar(&f.propose, "propose", false, "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("ask", err.Error())
	}
	if fs.NArg() > 1 {
		return f, usageErr("ask: too many arguments")
	}
	if fs.NArg() == 1 {
		f.path = fs.Arg(0)
	}
	return f, nil
}

type attestFlags struct{ allowDirty bool; reviewedBy string }

func parseAttestFlags(args []string) (attestFlags, error) {
	var f attestFlags
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--allow-dirty":
			f.allowDirty = true
		case a == "--reviewed-by":
			if i+1 >= len(args) {
				return f, usageErr("attest --reviewed-by requires a name")
			}
			f.reviewedBy = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(a, "--reviewed-by="):
			f.reviewedBy = strings.TrimSpace(strings.TrimPrefix(a, "--reviewed-by="))
		default:
			return f, usageErr(fmt.Sprintf("attest: unknown flag %q", a))
		}
	}
	return f, nil
}

type sockFlags struct{ path, root string }

func parseSockFlags(args []string) (sockFlags, error) {
	fs := newCommandFlagSet("sock")
	var f sockFlags
	fs.StringVar(&f.path, "path", "", "")
	fs.StringVar(&f.root, "repo", "", "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("sock", err.Error())
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("sock: unknown argument %q", fs.Arg(0)))
	}
	return f, nil
}

type shareFlags struct {
	packIDs                   []string
	skipPrepare, wantBundle, wantReveal bool
}

func parseShareFlags(args []string) (shareFlags, error) {
	fs := newCommandFlagSet("share")
	var f shareFlags
	var packsFlag string
	fs.StringVar(&packsFlag, "packs", "", "")
	fs.BoolVar(&f.skipPrepare, "skip-prepare-release", false, "")
	fs.BoolVar(&f.wantBundle, "bundle", false, "")
	fs.BoolVar(&f.wantReveal, "reveal", false, "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("share", err.Error())
	}
	if packsFlag != "" {
		f.packIDs = config.ParsePacksFlag(packsFlag)
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("share: unknown argument %q", fs.Arg(0)))
	}
	return f, nil
}

type scanFlags struct {
	packIDs        []string
	badge          bool
	formatMarkdown bool
}

func parseScanFlags(args []string) (scanFlags, error) {
	fs := newCommandFlagSet("scan")
	var f scanFlags
	var packsFlag, format string
	fs.StringVar(&packsFlag, "packs", "", "")
	fs.BoolVar(&f.badge, "badge", false, "")
	fs.StringVar(&format, "format", "", "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("scan", err.Error())
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("scan: unknown argument %q", fs.Arg(0)))
	}
	if packsFlag != "" {
		f.packIDs = config.ParsePacksFlag(packsFlag)
	}
	switch format {
	case "":
	case "markdown":
		f.formatMarkdown = true
	default:
		return f, usageErr(fmt.Sprintf("scan: unknown --format %q (use markdown)", format))
	}
	return f, nil
}

type fixFlags struct {
	art14 bool
	yes   bool
}

func parseFixFlags(args []string) (fixFlags, error) {
	fs := newCommandFlagSet("fix")
	var f fixFlags
	fs.BoolVar(&f.art14, "art14", false, "")
	fs.BoolVar(&f.yes, "yes", false, "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("fix", err.Error())
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("fix: unknown argument %q", fs.Arg(0)))
	}
	return f, nil
}

type driftFlags struct{ jsonOut bool }

func parseDriftFlags(args []string) (driftFlags, error) {
	fs := newCommandFlagSet("drift")
	var f driftFlags
	fs.BoolVar(&f.jsonOut, "json", false, "")
	if err := fs.Parse(args); err != nil {
		return f, flagUsageErr("drift", err.Error())
	}
	if fs.NArg() > 0 {
		return f, usageErr(fmt.Sprintf("drift: unknown argument %q", fs.Arg(0)))
	}
	return f, nil
}
