package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afelin/curbpack/internal/gitutil"
	"github.com/afelin/curbpack/internal/packs"
	"github.com/afelin/curbpack/internal/tty"
	"github.com/afelin/curbpack/internal/validate"
)

func cmdFix(args []string) error {
	f, err := parseFixFlags(args)
	if err != nil {
		return err
	}
	if !f.art14 {
		return usageErr("fix requires --art14")
	}
	root, err := gitutil.RepoRoot("")
	if err != nil {
		return usageErr("must run inside a git repository")
	}

	rel := packs.Art14RelPath()
	path, clean, err := validate.SafeJoin(root, rel)
	if err != nil {
		return err
	}

	product, _ := productHint(root)
	body := packs.Art14PathBody(product)

	var existing []byte
	if prev, err := os.ReadFile(path); err == nil {
		existing = prev
	}

	tty.PrintHeader("fix --art14")
	fmt.Printf("Target: %s\n\n", clean)
	fmt.Println("--- proposed content ---")
	fmt.Print(body)
	fmt.Println("--- end ---")

	if len(existing) > 0 {
		fmt.Printf("\n%s\n", tty.C(tty.Yellow, "Existing file will be replaced."))
		fmt.Println(unifiedLineDiff(string(existing), body))
	}

	if !f.yes {
		if !tty.IsTerminal {
			return usageErr("fix --art14: non-interactive shell — pass --yes to write")
		}
		fmt.Printf("\nWrite %s? [y/N]: ", clean)
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" {
			fmt.Println("Aborted — no files written.")
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	tty.PrintStatus("Art 14 path", true, clean+" written")
	fmt.Printf("%s\n", tty.C(tty.Dim, "Gate green means reporting path documented. Fill Last tabletop: after tabletop for badge rehearsed — then re-run curbpack check. Not conformity assessment."))
	return nil
}

func unifiedLineDiff(old, new string) string {
	oldLines := strings.Split(strings.TrimSuffix(old, "\n"), "\n")
	newLines := strings.Split(strings.TrimSuffix(new, "\n"), "\n")
	var b strings.Builder
	b.WriteString("--- existing\n+++ proposed\n")
	// Simple line-wise diff preview (first differing region only).
	max := len(oldLines)
	if len(newLines) > max {
		max = len(newLines)
	}
	for i := 0; i < max; i++ {
		var o, n string
		if i < len(oldLines) {
			o = oldLines[i]
		}
		if i < len(newLines) {
			n = newLines[i]
		}
		if o == n {
			continue
		}
		if o != "" {
			fmt.Fprintf(&b, "- %s\n", o)
		}
		if n != "" {
			fmt.Fprintf(&b, "+ %s\n", n)
		}
	}
	out := b.String()
	if out == "--- existing\n+++ proposed\n" {
		return "(content identical)\n"
	}
	return out
}
