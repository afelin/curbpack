package validate_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/afelin/curbpack/internal/validate"
)

func TestFreshAndOwnedFixturePack(t *testing.T) {
	dir := t.TempDir()
	fixtureRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "packs-fixtures"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CURBPACK_PACKS_DIR", fixtureRoot)

	initFreshOwnedGit(t, dir, "fresh-owned-widget", "owner@example.com")
	ownedBody := "# Owned policy for fresh-owned-widget\n\nProduct-specific draft.\n"
	reviewBody := "# Review log\n\nUpdated today with enough words to pass structural min_bytes for the fresh-owned-widget product.\n"
	mustWriteFresh(t, dir, "docs/owned-policy.md", ownedBody)
	mustWriteFresh(t, dir, "docs/review-log.md", reviewBody)
	runGitFresh(t, dir, "add", "docs/owned-policy.md", "docs/review-log.md")
	runGitFresh(t, dir, "commit", "-m", "owned docs", "-q")

	res, err := validate.Run(validate.Options{
		RepoRoot: dir,
		PackIDs:  []string{"fresh-owned-test"},
		Quiet:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Passed {
		t.Fatalf("expected green fresh-owned fixture: %v", res.Payload.Failures)
	}

	// Stale review log beyond max_age_days
	old := time.Now().AddDate(-2, 0, 0).Format(time.RFC3339)
	cmd := exec.Command("git", "commit", "--amend", "--no-edit", "--date", old)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_DATE="+old, "GIT_COMMITTER_DATE="+old)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("amend old date: %v %s", err, out)
	}
	res, err = validate.Run(validate.Options{RepoRoot: dir, PackIDs: []string{"fresh-owned-test"}, Quiet: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed {
		t.Fatal("fresh rule must fail when review log commit is too old")
	}
}

func initFreshOwnedGit(t *testing.T, dir, product, email string) {
	t.Helper()
	runGitFresh(t, dir, "init", "-q")
	runGitFresh(t, dir, "config", "user.email", email)
	runGitFresh(t, dir, "config", "user.name", "Owner")
	mustWriteFresh(t, dir, "README.md", "# "+product+"\n")
	mustWriteFresh(t, dir, "package.json", `{"name":"`+product+`","version":"1.0.0"}`+"\n")
	runGitFresh(t, dir, "add", ".")
	runGitFresh(t, dir, "commit", "-m", "init", "-q")
}

func runGitFresh(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func mustWriteFresh(t *testing.T, dir, rel, body string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFreshOwnedPackSchema(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "packs-fixtures", "fresh-owned-test", "pack.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"check": "fresh"`) {
		t.Fatal("fixture must include fresh check")
	}
}
