package pathjail_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/curbpack/internal/pathjail"
)

func TestUnderGitCaseInsensitive(t *testing.T) {
	for _, p := range []string{".git/config", ".Git/hooks/pre-commit", ".GIT/HEAD"} {
		if !pathjail.UnderGit(p) {
			t.Fatalf("expected .git jail for %q", p)
		}
		if err := pathjail.ValidateRel(p); err == nil {
			t.Fatalf("ValidateRel must refuse %q", p)
		}
	}
}

func TestAllowedRelMatchesValidate(t *testing.T) {
	cases := []string{"docs/a.md", "../x", ".git/x", "/abs", "", "ok\x00bad", "\r0", "\x10", " /000"}
	for _, c := range cases {
		err := pathjail.ValidateRel(c)
		allowed := pathjail.AllowedRel(c)
		if allowed && err != nil {
			t.Fatalf("AllowedRel true but ValidateRel err for %q: %v", c, err)
		}
		if !allowed && err == nil {
			t.Fatalf("AllowedRel false but ValidateRel ok for %q", c)
		}
	}
}

func TestJoin_RefusesSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "real.md")
	if err := os.WriteFile(target, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "SECURITY.md")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	_, _, err := pathjail.Join(root, "SECURITY.md")
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("Join must refuse symlink escape, got %v", err)
	}
}
