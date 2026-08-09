package packscmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/cyberready/internal/packscmd"
)

func TestImportAirGapRequiresAssuranceClass(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()
	t.Setenv("CYBERREADY_PACKS_DIR", dest)
	mustWrite(t, filepath.Join(src, "theater-pack", "pack.json"), `{
  "id": "theater-pack",
  "name": "Theater pack",
  "version": "0.0.1",
  "description": "Missing assurance_class on purpose.",
  "rules": [{
    "id": "T-1",
    "severity": "low",
    "type": "POLICY_VIOLATION",
    "check": "file_present",
    "path": "README.md",
    "description": "README present",
    "remediation": "Add README.md",
    "expected": "README.md exists"
  }]
}`)
	err := packscmd.ImportAirGap(src)
	if err == nil || !strings.Contains(err.Error(), "assurance_class") {
		t.Fatalf("want assurance_class refuse, got %v", err)
	}
}

func TestImportAirGapWritesDigest(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()
	t.Setenv("CYBERREADY_PACKS_DIR", dest)
	body := `{
  "id": "ok-pack",
  "name": "OK pack",
  "version": "0.0.1",
  "assurance_class": "structural_draft",
  "description": "Structural draft for import honesty tests.",
  "rules": [{
    "id": "T-1",
    "severity": "low",
    "type": "POLICY_VIOLATION",
    "check": "file_present",
    "path": "README.md",
    "description": "README present",
    "remediation": "Add README.md",
    "expected": "README.md exists"
  }]
}`
	mustWrite(t, filepath.Join(src, "ok-pack", "pack.json"), body)
	if err := packscmd.ImportAirGap(src); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "ok-pack", ".cyberready-pack.sha256")); err != nil {
		t.Fatal("missing digest sidecar")
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

func TestUpdateRequiresSHA256Pin(t *testing.T) {
	t.Setenv("CYBERREADY_PACKS_URL", "https://example.invalid/bundle.json")
	t.Setenv("CYBERREADY_PACKS_SHA256", "")
	err := packscmd.UpdateStub()
	if err == nil {
		t.Fatal("expected refuse without sha256 pin")
	}
	if !strings.Contains(err.Error(), "CYBERREADY_PACKS_SHA256") {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateOfflineInstructions(t *testing.T) {
	t.Setenv("CYBERREADY_PACKS_URL", "")
	t.Setenv("CYBERREADY_PACKS_SHA256", "")
	// Capture via ensuring no panic / nil error when URL unset.
	if err := packscmd.UpdateStub(); err != nil {
		t.Fatal(err)
	}
	_ = os.Getenv // keep os used for clarity in parallel tests
}
