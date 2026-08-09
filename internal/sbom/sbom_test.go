package sbom_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/afelin/cyberready/internal/buildinfo"
	"github.com/afelin/cyberready/internal/sbom"
)

func TestCycloneDXFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "package.json"), `{
  "name": "demo",
  "dependencies": { "left-pad": "1.3.0" },
  "devDependencies": { "typescript": "5.4.0" }
}`)
	doc, path, err := sbom.WriteCycloneDX(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if doc.BomFormat != "CycloneDX" || doc.SpecVersion != "1.5" {
		t.Fatalf("bad bom header: %+v", doc)
	}
	if len(doc.Components) != 2 {
		t.Fatalf("components=%d", len(doc.Components))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var probe map[string]any
	if json.Unmarshal(raw, &probe) != nil {
		t.Fatal("cdx json invalid")
	}
	if probe["bomFormat"] != "CycloneDX" {
		t.Fatal("missing bomFormat")
	}
	if len(doc.Metadata.Tools.Components) == 0 || doc.Metadata.Tools.Components[0].Version != buildinfo.Version {
		t.Fatalf("SBOM tool version=%q want buildinfo.Version=%q", toolVersion(doc), buildinfo.Version)
	}
}

func toolVersion(doc sbom.Document) string {
	if len(doc.Metadata.Tools.Components) == 0 {
		return ""
	}
	return doc.Metadata.Tools.Components[0].Version
}

func TestNPMLockParse(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "package-lock.json"), `{
  "name": "demo",
  "lockfileVersion": 3,
  "packages": {
    "": { "name": "demo" },
    "node_modules/axios": { "version": "1.8.4" }
  }
}`)
	pkgs, src, err := sbom.CollectPackages(dir)
	if err != nil {
		t.Fatal(err)
	}
	if src != "package-lock.json" {
		t.Fatalf("src=%s", src)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "axios" {
		t.Fatalf("pkgs=%v", pkgs)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
