package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/curbpack/internal/paths"
)

func TestLoadLegacyWriteNew(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, paths.LegacyConfigFile)
	if err := os.WriteFile(legacy, []byte(`{"packs":["house-policy","cra-baseline"]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || len(cfg.Packs) != 2 {
		t.Fatalf("legacy load: %+v", cfg)
	}
	if err := Write(root, File{Packs: []string{"house-policy"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.ConfigPath(root)); err != nil {
		t.Fatalf("expected write to %s: %v", paths.ConfigFile, err)
	}
	// Legacy file must remain untouched.
	b, err := os.ReadFile(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "cra-baseline") {
		t.Fatalf("legacy file mutated: %s", b)
	}
	cfg2, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg2.Packs) != 1 || cfg2.Packs[0] != "house-policy" {
		t.Fatalf("new config should win on read: %+v", cfg2)
	}
}

func TestResolvePackIDsColdDefault(t *testing.T) {
	root := t.TempDir()
	got, err := ResolvePackIDs(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "house-policy" {
		t.Fatalf("ResolvePackIDs cold default = house-policy, got %v", got)
	}
}

func TestResolvePackIDsForScanColdDefault(t *testing.T) {
	root := t.TempDir()
	got, err := ResolvePackIDsForScan(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "cra-baseline" {
		t.Fatalf("ResolvePackIDsForScan cold default = cra-baseline, got %v", got)
	}
}

func TestResolvePackIDsForScanCLIOverride(t *testing.T) {
	root := t.TempDir()
	got, err := ResolvePackIDsForScan(root, []string{"house-policy"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "house-policy" {
		t.Fatalf("CLI override: got %v", got)
	}
}

func TestResolvePackIDsForScanConfigPacks(t *testing.T) {
	root := t.TempDir()
	if err := Write(root, File{Packs: []string{"medtech-iec62304"}}); err != nil {
		t.Fatal(err)
	}
	got, err := ResolvePackIDsForScan(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "medtech-iec62304" {
		t.Fatalf("config packs: got %v", got)
	}
}
