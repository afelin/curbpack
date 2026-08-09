package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/afelin/cyberready/internal/config"
	"github.com/afelin/cyberready/internal/gitutil"
	"github.com/afelin/cyberready/internal/packs"
	"github.com/afelin/cyberready/internal/tty"
)

const Claim = "Prepares evidence for human review — not a conformity assessment."

// InstrumentPanelCovenant is the always-on honesty line for doctor / instrument surfaces.
const InstrumentPanelCovenant = "instrument panel · not a security program · not conformity assessment"

// Options controls doctor checks.
type Options struct {
	RepoRoot string // optional; empty = discover from cwd
	Version  string
}

// Run prints environment confidence checks. Always exits 0 from CLI unless fatal I/O.
func Run(opts Options) error {
	tty.PrintHeader("CYBERREADY DOCTOR")
	fmt.Printf("%s\n", tty.C(tty.Dim, Claim))
	fmt.Printf("%s\n\n", tty.C(tty.Dim, InstrumentPanelCovenant))

	ok := true
	printCheck := func(name string, passed bool, detail string) {
		tty.PrintStatus(name, passed, detail)
		if !passed {
			ok = false
		}
	}

	printCheck("binary", true, fmt.Sprintf("cyberready %s (%s/%s)", opts.Version, runtime.GOOS, runtime.GOARCH))
	printCheck("Go toolchain", true, "not required for released binaries (stdlib build only)")

	if _, err := exec.LookPath("git"); err != nil {
		printCheck("git on PATH", false, "git not found — required for check/demo/attest")
	} else {
		out, _ := exec.Command("git", "version").Output()
		printCheck("git on PATH", true, strings.TrimSpace(string(out)))
	}

	root := opts.RepoRoot
	inRepo := false
	if root == "" {
		if r, err := gitutil.RepoRoot(""); err == nil {
			root = r
			inRepo = true
		}
	} else {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			inRepo = true
		}
	}

	if inRepo {
		printCheck("git repository", true, root)
		if cfg, err := config.Load(root); err != nil {
			printCheck(".cyberready.json", false, err.Error())
		} else if cfg == nil {
			printCheck(".cyberready.json", false, "missing — run: cyberready init")
		} else {
			printCheck(".cyberready.json", true, "packs="+strings.Join(cfg.Packs, ","))
			if cfg.Hooks {
				hook := filepath.Join(root, ".git", "hooks", "pre-commit")
				b, err := os.ReadFile(hook)
				if err != nil {
					printCheck("pre-commit hook", false, "hooks=true but hook missing")
				} else if !strings.Contains(string(b), "cyberready") {
					printCheck("pre-commit hook", false, "present but does not call cyberready")
				} else {
					detail := "cyberready check"
					if strings.Contains(string(b), "--heal") {
						detail = "cyberready check --heal"
					}
					printCheck("pre-commit hook", true, detail)
				}
			} else {
				printCheck("pre-commit hook", true, "not enabled (optional: cyberready init)")
			}
		}
		skill := filepath.Join(root, ".cursor", "skills", "cyberready", "SKILL.md")
		if _, err := os.Stat(skill); err == nil {
			printCheck("Cursor skill", true, skill)
		} else {
			printCheck("Cursor skill", true, "absent (optional: cyberready init)")
		}
		tasks := filepath.Join(root, ".vscode", "tasks.json")
		if _, err := os.Stat(tasks); err == nil {
			printCheck("VS Code/Cursor tasks", true, tasks)
		} else {
			printCheck("VS Code/Cursor tasks", true, "absent (optional: cyberready init)")
		}
	} else {
		printCheck("git repository", true, "cwd is not a product repo (ok — use cyberready demo)")
	}

	ids, err := packs.ListIDs()
	if err != nil {
		printCheck("embedded packs", false, err.Error())
	} else {
		printCheck("embedded packs", true, strings.Join(ids, ", "))
	}

	fmt.Println()
	if ok {
		if inRepo {
			cfg, _ := config.Load(root)
			if cfg == nil {
				fmt.Printf("%s\n", tty.C(tty.Bold+tty.Green, "[+] Doctor OK — next: cyberready init"))
			} else {
				fmt.Printf("%s\n", tty.C(tty.Bold+tty.Green, "[+] Doctor OK — next: cyberready check  (or bare: cyberready)"))
			}
		} else {
			fmt.Printf("%s\n", tty.C(tty.Bold+tty.Green, "[+] Doctor OK — try: cyberready demo"))
		}
	} else {
		fmt.Printf("%s\n", tty.C(tty.Bold+tty.Yellow, "[!] Doctor found issues — fix above, then re-run"))
	}
	fmt.Printf("%s\n", tty.C(tty.Dim, Claim))
	return nil
}
