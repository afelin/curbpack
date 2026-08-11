package attest_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/curbpack/internal/attest"
)

func TestReproducibleStateHash(t *testing.T) {
	a := attest.ComputeStateHash("abc", "parent", "sbom1", "vex1")
	b := attest.ComputeStateHash("abc", "parent", "sbom1", "vex1")
	if a != b {
		t.Fatal("state hash must be reproducible")
	}
	c := attest.ComputeStateHash("abc", "parent", "sbom2", "vex1")
	if a == c {
		t.Fatal("sbom digest must affect hash")
	}
}

func TestStateHashFieldBoundaryCollision(t *testing.T) {
	// Pipe (or other delimiter) ambiguity must not collide under length-prefixing.
	pairs := [][2][4]string{
		{
			{"a", "b", "c|d", "e"},
			{"a|b", "c", "d", "e"},
		},
		{
			{"ab", "c", "d", "e"},
			{"a", "bc", "d", "e"},
		},
		{
			{"", "x", "y", "z"},
			{"x", "", "y", "z"},
		},
		{
			{"commit", "parent", "sbom=x", "vex"},
			{"commit", "parent", "sbom", "=xvex"},
		},
	}
	for i, p := range pairs {
		left := attest.ComputeStateHash(p[0][0], p[0][1], p[0][2], p[0][3])
		right := attest.ComputeStateHash(p[1][0], p[1][1], p[1][2], p[1][3])
		if left == right {
			t.Fatalf("pair %d: boundary inputs must not collide: %q vs %q → %s", i, p[0], p[1], left)
		}
	}
}

func TestAttestDualReadsLegacyEvidence(t *testing.T) {
	dir := t.TempDir()
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
	run("git", "config", "user.email", "attest@curbpack.local")
	run("git", "config", "user.name", "Attest")
	run("git", "commit", "--allow-empty", "-m", "init", "-q")

	legacyEv := filepath.Join(dir, ".github", "cyberready", "evidence")
	if err := os.MkdirAll(legacyEv, 0o755); err != nil {
		t.Fatal(err)
	}
	sbomBody := []byte(`{"bomFormat":"CycloneDX","specVersion":"1.5","components":[]}` + "\n")
	vexBody := []byte(`{"author":"curbpack","statements":[]}` + "\n")
	if err := os.WriteFile(filepath.Join(legacyEv, "sbom.cdx.json"), sbomBody, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyEv, "vex-pending.json"), vexBody, 0o644); err != nil {
		t.Fatal(err)
	}

	cap, err := attest.Run(attest.Options{RepoRoot: dir, AllowDirty: true})
	if err != nil {
		t.Fatalf("attest: %v", err)
	}
	wantSbom := fmt.Sprintf("%x", sha256.Sum256(sbomBody))
	wantVex := fmt.Sprintf("%x", sha256.Sum256(vexBody))
	wantHash := attest.ComputeStateHash(cap.CommitSHA, cap.ParentStateHash, wantSbom, wantVex)
	if cap.StateHash != wantHash {
		t.Fatalf("state_hash unbound to legacy evidence:\n got %s\nwant %s\n sbom=%s vex=%s",
			cap.StateHash, wantHash, cap.Evidence["sbom_digest"], cap.Evidence["vex_digest"])
	}
	if cap.Evidence["sbom_digest"] != wantSbom || cap.Evidence["vex_digest"] != wantVex {
		t.Fatalf("evidence digests: %#v", cap.Evidence)
	}
	if cap.Evidence["sbom_path"] != ".github/cyberready/evidence/sbom.cdx.json" {
		t.Fatalf("sbom_path=%q", cap.Evidence["sbom_path"])
	}
	if cap.Evidence["vex_path"] != ".github/cyberready/evidence/vex-pending.json" {
		t.Fatalf("vex_path=%q", cap.Evidence["vex_path"])
	}
	// Write-new pointer still lands under curbpack.
	if _, err := os.Stat(filepath.Join(dir, ".github", "curbpack", "evidence", "hpurl-pointer.json")); err != nil {
		t.Fatalf("expected write-new hpurl-pointer: %v", err)
	}
}

func TestAttestRefusesDirtyWithoutAllowDirty(t *testing.T) {
	dir := t.TempDir()
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
	run("git", "config", "user.email", "attest@curbpack.local")
	run("git", "config", "user.name", "Attest")
	run("git", "commit", "--allow-empty", "-m", "init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("uncommitted\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := attest.Run(attest.Options{RepoRoot: dir, AllowDirty: false})
	if err == nil || !strings.Contains(err.Error(), "OCC conflict") {
		t.Fatalf("want OCC conflict without --allow-dirty, got %v", err)
	}
	if _, err := attest.Run(attest.Options{RepoRoot: dir, AllowDirty: true}); err != nil {
		t.Fatalf("--allow-dirty must proceed: %v", err)
	}
}
