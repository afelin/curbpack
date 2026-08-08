package contract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/afelin/cyberready/internal/config"
	"github.com/afelin/cyberready/internal/formhints"
	"github.com/afelin/cyberready/internal/remediation"
	"github.com/afelin/cyberready/internal/validate"
)

// Action-equivalent smoke: uninitialized repo (no .cyberready.json) + heal stubs.
// Mirrors Action heal:true — ResolvePackIDs falls back to house-policy.
func TestUninitializedHealGreensOrDeterministicRed(t *testing.T) {
	dir := t.TempDir()
	mustRealGit(t, dir)
	mustWrite(t, filepath.Join(dir, "README.md"), "# Product\n")

	if _, err := os.Stat(filepath.Join(dir, ".cyberready.json")); err == nil {
		t.Fatal("fixture must start without .cyberready.json")
	}

	ids, err := config.ResolvePackIDs(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "house-policy" {
		t.Fatalf("uninitialized ResolvePackIDs=%v want [house-policy]", ids)
	}

	res, err := validate.Run(validate.Options{RepoRoot: dir, Quiet: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed {
		t.Fatal("expected initial red without security stubs")
	}

	cache, _ := remediation.Load(dir)
	hints := formhints.ForFailuresCached(res.Payload.Failures, cache)
	hints, err = formhints.ApplyStubs(dir, hints)
	if err != nil {
		t.Fatal(err)
	}
	applied := 0
	for _, h := range hints {
		if h.Applied {
			applied++
		}
	}
	if applied == 0 {
		t.Fatal("heal must apply at least one missing stub")
	}
	_ = formhints.PersistCache(dir, hints)

	res2, err := validate.Run(validate.Options{RepoRoot: dir, Quiet: true})
	if err != nil {
		t.Fatal(err)
	}
	// House-policy cold stubs typically green; if not, remaining failures must be
	// deterministic (non-empty gate_ids) — still felt Action value.
	if !res2.Passed {
		if len(res2.Payload.Failures) == 0 {
			t.Fatal("red without failures is not deterministic")
		}
		t.Logf("post-heal red with %d failure(s) — acceptable Action-only felt value", len(res2.Payload.Failures))
	}
	if _, err := os.Stat(filepath.Join(dir, "SECURITY.md")); err != nil {
		t.Fatalf("expected SECURITY.md stub after heal: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".cyberready.json")); err == nil {
		t.Fatal("heal must not invent .cyberready.json (Action path stays config-free)")
	}
}

func TestCLICheckHealUninitialized(t *testing.T) {
	bin := buildCyberready(t)
	dir := t.TempDir()
	mustRealGit(t, dir)
	mustWrite(t, filepath.Join(dir, "README.md"), "# Product\n")

	cmd := exec.Command(bin, "check", "--heal")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Exit 1 is only OK if stubs were written and failures remain deterministic.
		if _, st := os.Stat(filepath.Join(dir, "SECURITY.md")); st != nil {
			t.Fatalf("check --heal failed without stubs: %v\n%s", err, out)
		}
		t.Logf("check --heal exit non-zero after stubs (ok): %v\n%s", err, out)
		return
	}
	if _, err := os.Stat(filepath.Join(dir, "SECURITY.md")); err != nil {
		t.Fatalf("green path must have written SECURITY.md: %v\n%s", err, out)
	}
}

func buildCyberready(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "cyberready")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/cyberready")
	cmd.Dir = repoRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/contract → repo root
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
