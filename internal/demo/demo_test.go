package demo

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyEmbedded(t *testing.T) {
	dir := t.TempDir()
	if err := copyEmbedded(dir); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"SECURITY.md", ".well-known/security.txt", "README.md"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
}

func TestRunSandbox(t *testing.T) {
	dir := t.TempDir()
	if err := Run(Options{OutDir: dir, KeepDir: true, Version: "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "review-pack", "buyer-onepager.html")); err != nil {
		t.Fatalf("onepager: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".cyberready.json")); err != nil {
		t.Fatalf("config: %v", err)
	}
}

func TestRunPrintsProductNextLine(t *testing.T) {
	dir := t.TempDir()
	out := captureStdout(t, func() {
		if err := Run(Options{OutDir: dir, KeepDir: true, Version: "test"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "next on your repo: cyberready init && cyberready check") {
		t.Fatalf("missing product next-line in output:\n%s", out)
	}
	if !strings.Contains(out, "afelin/cyberready@v0.4.0") {
		t.Fatalf("missing Action pin pointer in output:\n%s", out)
	}
	if strings.Contains(out, "init --packs house-policy --hooks") {
		t.Fatalf("must not teach flag soup after demo:\n%s", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}
