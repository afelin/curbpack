package instrument_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/afelin/cyberready/internal/instrument"
)

func TestComputeWriteLoadAndDepDelta(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "package.json"), `{"name":"demo","version":"1.0.0","dependencies":{"left-pad":"1.0.0"}}`+"\n")
	mustWrite(t, filepath.Join(dir, ".env"), "API_KEY=abcdefghijklmnopqrstuvwxyz0123456789\n")

	s1 := instrument.Compute(dir)
	if s1.DepsFP == "" {
		t.Fatal("expected deps_fp")
	}
	if s1.SecretHits < 1 {
		t.Fatalf("expected secret hit, got %d", s1.SecretHits)
	}
	if err := instrument.Write(dir, s1); err != nil {
		t.Fatal(err)
	}
	loaded, ok := instrument.Load(dir)
	if !ok || loaded.DepsFP != s1.DepsFP {
		t.Fatalf("load mismatch ok=%v fp=%q", ok, loaded.DepsFP)
	}

	mustWrite(t, filepath.Join(dir, "package.json"), `{"name":"demo","version":"1.0.0","dependencies":{"left-pad":"1.0.0","axios":"1.8.4"}}`+"\n")
	s2 := instrument.Compute(dir)
	add, rem := instrument.DepDelta(s1, s2)
	if add < 1 {
		t.Fatalf("expected +deps, got +%d/−%d", add, rem)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
