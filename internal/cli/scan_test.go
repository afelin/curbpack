package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/curbpack/internal/cli"
	"github.com/afelin/curbpack/internal/packs"
)

func TestRun_ScanReadOnly(t *testing.T) {
	dir := t.TempDir()
	initScanGit(t, dir)
	mustWriteScan(t, filepath.Join(dir, "package.json"), `{"name":"scan-widget","version":"1.0.0"}`+"\n")
	mustWriteScan(t, filepath.Join(dir, "README.md"), "# Scan Widget\n")

	stdout, _ := capture(t, func() {
		old, _ := os.Getwd()
		_ = os.Chdir(dir)
		defer func() { _ = os.Chdir(old) }()
		if err := cli.Run([]string{"scan"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(stdout, "Read-only") {
		t.Fatalf("missing read-only banner: %q", stdout)
	}
	if !strings.Contains(stdout, "scan-widget") {
		t.Fatalf("missing product hint: %q", stdout)
	}
	if !strings.Contains(stdout, "Art 14 reporting clock") {
		t.Fatalf("missing Art 14 clock: %q", stdout)
	}
	if !strings.Contains(stdout, "Packs: cra-baseline") {
		t.Fatalf("uninitialized scan must default to cra-baseline: %q", stdout)
	}
	if !strings.Contains(stdout, "ENISA SME maturity mapping") {
		t.Fatalf("cra-baseline scan must show ENISA mapping pointer: %q", stdout)
	}
	if strings.Contains(stdout, "Readiness Score") {
		t.Fatal("scan must not show readiness thermometer")
	}
	cache := filepath.Join(dir, ".github", "curbpack", "cache", "latest_failure.json")
	if _, err := os.Stat(cache); err == nil {
		t.Fatal("scan must not write cache")
	}
}

func TestRun_ScanHousePolicyNoENISA(t *testing.T) {
	dir := t.TempDir()
	initScanGit(t, dir)
	mustWriteScan(t, filepath.Join(dir, "README.md"), "# Demo\n")

	stdout, _ := capture(t, func() {
		old, _ := os.Getwd()
		_ = os.Chdir(dir)
		defer func() { _ = os.Chdir(old) }()
		if err := cli.Run([]string{"scan", "--packs", "house-policy"}); err != nil {
			t.Fatal(err)
		}
	})
	if strings.Contains(stdout, "ENISA SME maturity mapping") {
		t.Fatalf("house-policy scan must not show ENISA line: %q", stdout)
	}
}

func TestRun_FixArt14Yes(t *testing.T) {
	dir := t.TempDir()
	initScanGit(t, dir)
	mustWriteScan(t, filepath.Join(dir, "package.json"), `{"name":"fixco","version":"1.0.0"}`+"\n")

	stdout, _ := capture(t, func() {
		old, _ := os.Getwd()
		_ = os.Chdir(dir)
		defer func() { _ = os.Chdir(old) }()
		if err := cli.Run([]string{"fix", "--art14", "--yes"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(stdout, "docs/incident/art14-path.md") {
		t.Fatalf("expected target path in output: %q", stdout)
	}
	path := filepath.Join(dir, "docs/incident", "art14-path.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "fixco") {
		t.Fatalf("Art 14 body must include product name: %q", b)
	}
	if string(b) == packs.DefaultScaffoldBody(packs.Art14RelPath()) {
		t.Fatal("fix --art14 must write Art14PathBody not DefaultScaffoldBody")
	}
	if strings.Contains(string(b), "YYYY-MM-DD") {
		t.Fatal("fix --art14 must prefill dates, not YYYY-MM-DD placeholders")
	}
}

func TestRun_ScanAfterFixArt14(t *testing.T) {
	dir := t.TempDir()
	initScanGit(t, dir)
	mustWriteScan(t, filepath.Join(dir, "package.json"), `{"name":"fixco","version":"1.0.0"}`+"\n")

	runScan := func() string {
		t.Helper()
		stdout, _ := capture(t, func() {
			old, _ := os.Getwd()
			_ = os.Chdir(dir)
			defer func() { _ = os.Chdir(old) }()
			if err := cli.Run([]string{"scan"}); err != nil {
				t.Fatal(err)
			}
		})
		return stdout
	}

	before := runScan()
	if !strings.Contains(before, "fix --art14") {
		t.Fatalf("cold scan must suggest fix --art14 in Next: %q", before)
	}

	stdoutFix, _ := capture(t, func() {
		old, _ := os.Getwd()
		_ = os.Chdir(dir)
		defer func() { _ = os.Chdir(old) }()
		if err := cli.Run([]string{"fix", "--art14", "--yes"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(stdoutFix, "docs/incident/art14-path.md") {
		t.Fatalf("fix output: %q", stdoutFix)
	}

	after := runScan()
	if !strings.Contains(after, "✔") || !strings.Contains(after, "CRA-ART14-PATH") {
		t.Fatalf("post-fix scan must show satisfied CRA-ART14-PATH: %q", after)
	}
	if strings.Contains(after, "fix --art14") {
		t.Fatalf("post-fix scan must not suggest fix --art14: %q", after)
	}
}

func TestRun_ScanBadge(t *testing.T) {
	dir := t.TempDir()
	initScanGit(t, dir)
	mustWriteScan(t, filepath.Join(dir, "package.json"), `{"name":"badgeco","version":"1.0.0"}`+"\n")

	assertBadge := func(t *testing.T, stdout string, wantRehearsed bool) {
		t.Helper()
		if !strings.Contains(stdout, "Art 14 path ·") {
			t.Fatalf("missing badge prefix: %q", stdout)
		}
		if !strings.Contains(stdout, "self-declared · curbpack") {
			t.Fatalf("missing self-declared suffix: %q", stdout)
		}
		for _, bad := range []string{"failing", "not started", "CRA compliant", "passing", "green", "Drafted"} {
			if strings.Contains(strings.ToLower(stdout), strings.ToLower(bad)) {
				t.Fatalf("badge must not contain %q: %q", bad, stdout)
			}
		}
		if wantRehearsed {
			if !strings.Contains(stdout, "rehearsed") || strings.Contains(stdout, "not rehearsed") {
				t.Fatalf("expected rehearsed badge: %q", stdout)
			}
		} else if !strings.Contains(stdout, "not rehearsed") {
			t.Fatalf("expected not rehearsed badge: %q", stdout)
		}
		if strings.Contains(stdout, "CURBPACK SCAN") {
			t.Fatalf("badge mode must not print full scan header: %q", stdout)
		}
	}

	runBadge := func(t *testing.T, args []string) string {
		t.Helper()
		stdout, _ := capture(t, func() {
			old, _ := os.Getwd()
			_ = os.Chdir(dir)
			defer func() { _ = os.Chdir(old) }()
			if err := cli.Run(args); err != nil {
				t.Fatal(err)
			}
		})
		return stdout
	}

	for _, args := range [][]string{
		{"scan", "--badge"},
		{"scan", "--format", "markdown"},
	} {
		args := args
		t.Run(strings.Join(args, " ")+"/cold", func(t *testing.T) {
			assertBadge(t, runBadge(t, args), false)
		})
	}

	t.Run("after fix --art14 still not rehearsed", func(t *testing.T) {
		_, _ = capture(t, func() {
			old, _ := os.Getwd()
			_ = os.Chdir(dir)
			defer func() { _ = os.Chdir(old) }()
			if err := cli.Run([]string{"fix", "--art14", "--yes"}); err != nil {
				t.Fatal(err)
			}
		})
		assertBadge(t, runBadge(t, []string{"scan", "--badge"}), false)
	})

	t.Run("after human fills Last tabletop", func(t *testing.T) {
		path := filepath.Join(dir, "docs", "incident", "art14-path.md")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(body), "Last tabletop:\n", "Last tabletop: 2026-03-01\n", 1)
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			t.Fatal(err)
		}
		stdout := runBadge(t, []string{"scan", "--badge"})
		if !strings.Contains(stdout, "rehearsed 2026-03-01") {
			t.Fatalf("expected human-filled date in badge: %q", stdout)
		}
		if !strings.Contains(stdout, "months ago") && !strings.Contains(stdout, "month ago") {
			t.Fatalf("expected staleness suffix: %q", stdout)
		}
	})
}

func TestRun_AskMySuppliers(t *testing.T) {
	dir := t.TempDir()
	initScanGit(t, dir)
	mustWriteScan(t, filepath.Join(dir, "README.md"), "# Demo\n")
	mustWriteScan(t, filepath.Join(dir, "SECURITY.md"), "# Security\n\n"+strings.Repeat("word ", 80)+"\n")
	mustWriteScan(t, filepath.Join(dir, ".well-known", "security.txt"), "Contact: mailto:a@b.c\nExpires: 2027-01-01T00:00:00.000Z\nPreferred-Languages: en\n")
	cfg := filepath.Join(dir, ".curbpack.json")
	mustWriteScan(t, cfg, `{"packs":["house-policy"]}`+"\n")

	stdout, _ := capture(t, func() {
		old, _ := os.Getwd()
		_ = os.Chdir(dir)
		defer func() { _ = os.Chdir(old) }()
		if err := cli.Run([]string{"ask-my-suppliers"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(stdout, "For human review:") {
		t.Fatalf("stdout must include human question text: %q", stdout)
	}
	if !strings.Contains(stdout, "Subject: Supplier evidence checklist") {
		t.Fatalf("stdout must include supplier email subject: %q", stdout)
	}
	if !strings.Contains(stdout, "Writes review-pack/supplier-checklist.md") {
		t.Fatalf("stdout must warn that review-pack is written: %q", stdout)
	}
	if !strings.Contains(stdout, "---") {
		t.Fatalf("stdout must separate checklist and email with ---: %q", stdout)
	}

	mdPath := filepath.Join(dir, "review-pack", "supplier-checklist.md")
	if _, err := os.Stat(mdPath); err != nil {
		t.Fatalf("expected review-pack/supplier-checklist.md: %v", err)
	}
	jsonPath := filepath.Join(dir, "review-pack", "supplier-checklist.json")
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("expected review-pack/supplier-checklist.json: %v", err)
	}
	cacheMD := filepath.Join(dir, ".github", "curbpack", "cache", "buyer-questions.md")
	if _, err := os.Stat(cacheMD); err == nil {
		t.Fatal("ask-my-suppliers must not write cache buyer-questions.md")
	}
	if _, err := os.Stat(filepath.Join(dir, ".github")); !os.IsNotExist(err) {
		t.Fatal("virgin repo: ask-my-suppliers must not create .github/")
	}
}

func TestRun_AskMySuppliersStdoutOnly(t *testing.T) {
	dir := t.TempDir()
	initScanGit(t, dir)
	mustWriteScan(t, filepath.Join(dir, "README.md"), "# Demo\n")
	mustWriteScan(t, filepath.Join(dir, ".curbpack.json"), `{"packs":["house-policy"]}`+"\n")

	stdout, _ := capture(t, func() {
		old, _ := os.Getwd()
		_ = os.Chdir(dir)
		defer func() { _ = os.Chdir(old) }()
		if err := cli.Run([]string{"ask-my-suppliers", "--stdout-only"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(stdout, "Subject: Supplier evidence checklist") {
		t.Fatalf("stdout-only must still print email: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, "review-pack")); !os.IsNotExist(err) {
		t.Fatal("--stdout-only must not write review-pack/")
	}
}

func initScanGit(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	run("git", "init", "-q")
	run("git", "config", "user.email", "t@example.com")
	run("git", "config", "user.name", "t")
	mustWriteScan(t, filepath.Join(dir, ".gitkeep"), "")
	run("git", "add", ".")
	run("git", "commit", "-m", "init", "-q")
}

func mustWriteScan(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
