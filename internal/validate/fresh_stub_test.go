package validate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/curbpack/internal/formhints"
	"github.com/afelin/curbpack/internal/ir"
	"github.com/afelin/curbpack/internal/packs"
	"github.com/afelin/curbpack/internal/validate"
)

func TestFreshStubPathsSkipsAntiPlaceholderSameRun(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	mustWrite(t, filepath.Join(dir, "README.md"), "# Demo\n")
	hints := formhints.ForFailures([]ir.Failure{{GateID: "HOUSE-SECURITY-MD"}})
	out, err := formhints.ApplyStubs(dir, hints)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || !out[0].Applied {
		t.Fatalf("expected SECURITY.md stub applied, got %+v", out)
	}

	fresh := map[string]struct{}{"SECURITY.md": {}}
	resFresh, err := validate.Run(validate.Options{
		RepoRoot:       dir,
		PackIDs:        []string{"house-policy"},
		Quiet:          true,
		FreshStubPaths: fresh,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range resFresh.Payload.Failures {
		if f.GateID == "HOUSE-ANTI-PLACEHOLDER" && strings.Contains(f.ASTCoordinates.TargetFile, "SECURITY.md") {
			t.Fatalf("fresh stub must skip anti_placeholder same run, got %#v", f)
		}
	}

	resLater, err := validate.Run(validate.Options{RepoRoot: dir, PackIDs: []string{"house-policy"}, Quiet: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasScaffoldOverlap(resLater, "HOUSE-ANTI-PLACEHOLDER", "SECURITY.md") {
		t.Fatalf("next run must still score scaffold overlap, got %#v", resLater.Payload.Failures)
	}
}

func TestReadOnlyValidateSkipsCacheWrite(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	mustWrite(t, filepath.Join(dir, "README.md"), "# Demo\n")
	cache := filepath.Join(dir, ".github", "curbpack", "cache", "latest_failure.json")
	if _, err := validate.Run(validate.Options{RepoRoot: dir, PackIDs: []string{"house-policy"}, Quiet: true, ReadOnly: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cache); err == nil {
		t.Fatal("read-only validate must not write cache")
	}
}

func TestFreshStubAllowsHealGreenWithWarning(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	mustWrite(t, filepath.Join(dir, ".well-known", "security.txt"), "Contact: mailto:a@b.c\nExpires: 2027-01-01T00:00:00.000Z\nPreferred-Languages: en\n")
	mustWrite(t, filepath.Join(dir, "README.md"), "# Demo\n")

	hints := formhints.ForFailures([]ir.Failure{{GateID: "HOUSE-SECURITY-MD"}})
	out, err := formhints.ApplyStubs(dir, hints)
	if err != nil || !out[0].Applied {
		t.Fatal("expected SECURITY.md stub")
	}
	res, err := validate.Run(validate.Options{
		RepoRoot:       dir,
		PackIDs:        []string{"house-policy"},
		Quiet:          true,
		FreshStubPaths: map[string]struct{}{"SECURITY.md": {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Passed {
		t.Fatalf("fresh heal stub should pass same run when only anti_placeholder would fail, got %#v", res.Payload.Failures)
	}
}

func TestArt14PathBodyProductName(t *testing.T) {
	body := packs.Art14PathBody("acme")
	if !strings.Contains(body, "acme") {
		t.Fatal("Art14PathBody must insert product name")
	}
	if strings.Contains(body, "YYYY-MM-DD") {
		t.Fatal("Art14PathBody must not contain YYYY-MM-DD placeholders")
	}
	if !strings.Contains(body, "Rehearsal status: unrehearsed draft") {
		t.Fatal("Art14PathBody must include unrehearsed status line")
	}
	if packs.ScaffoldOverlap(body, packs.Art14RelPath(), "acme") {
		t.Fatal("Art14PathBody must not overlap DefaultScaffoldBody")
	}
}

func TestFixArt14NoAntiPlaceholderOnArt14File(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	mustWrite(t, filepath.Join(dir, "package.json"), `{"name":"fixco","version":"1.0.0"}`+"\n")

	body := packs.Art14PathBody("fixco")
	mustWrite(t, filepath.Join(dir, packs.Art14RelPath()), body)

	res, err := validate.Run(validate.Options{
		RepoRoot: dir,
		PackIDs:  []string{"cra-baseline"},
		Quiet:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Payload.Failures {
		if f.GateID == "CRA-ANTI-PLACEHOLDER" && strings.Contains(f.ASTCoordinates.TargetFile, "art14-path.md") {
			t.Fatalf("fix template must not trigger CRA-ANTI-PLACEHOLDER on art14 alone, got %#v", f)
		}
	}
}
