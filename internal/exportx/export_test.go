package exportx_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/cyberready/internal/exportx"
	"github.com/afelin/cyberready/internal/ir"
	"github.com/afelin/cyberready/internal/validate"
)

func TestWriteSARIF_RuleIDsMatchGates(t *testing.T) {
	dir := t.TempDir()
	mustRealGit(t, dir)
	writeMinimalHouseFail(t, dir)

	out := filepath.Join(dir, "out.sarif")
	path, n, err := exportx.WriteSARIF(dir, []string{"house-policy"}, out)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("expected n > 0 SARIF results on house-policy failure")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc exportx.SARIFDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Runs) == 0 || len(doc.Runs[0].Results) == 0 {
		t.Fatal("empty SARIF runs/results")
	}
	res, err := validate.Run(validate.Options{RepoRoot: dir, PackIDs: []string{"house-policy"}, Quiet: true})
	if err != nil {
		t.Fatal(err)
	}
	gates := map[string]bool{}
	for _, f := range res.Payload.Failures {
		gates[f.GateID] = true
	}
	for _, r := range doc.Runs[0].Results {
		if r.RuleID == "" || !gates[r.RuleID] {
			t.Fatalf("SARIF ruleId %q must equal a gate_id; gates=%v", r.RuleID, gates)
		}
	}
}

func TestWriteSARIF_EmptyOnGreen(t *testing.T) {
	dir := t.TempDir()
	mustRealGit(t, dir)
	writeGoodHouse(t, dir)

	out := filepath.Join(dir, "green.sarif")
	_, n, err := exportx.WriteSARIF(dir, []string{"house-policy"}, out)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("green fixture must yield 0 SARIF results, got %d", n)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var doc exportx.SARIFDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Runs) == 0 {
		t.Fatal("expected a run shell even when green")
	}
	if len(doc.Runs[0].Results) != 0 {
		t.Fatalf("want empty results, got %d", len(doc.Runs[0].Results))
	}
}

func TestFromGateFailures_RuleIDEqualsGateID(t *testing.T) {
	payload := ir.GateFailurePayload{
		Failures: []ir.Failure{{
			GateID:               "HOUSE-SECURITY-MD",
			Severity:             "high",
			SanitizedDescription: "missing SECURITY.md",
		}},
	}
	doc := exportx.FromGateFailures(payload)
	if len(doc.Runs[0].Results) != 1 || doc.Runs[0].Results[0].RuleID != "HOUSE-SECURITY-MD" {
		t.Fatalf("ruleId mismatch: %+v", doc.Runs[0].Results)
	}
}

func TestWatchlistJoin_Informational(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "package-lock.json"), `{
  "name": "demo",
  "lockfileVersion": 3,
  "packages": {
    "": { "name": "demo" },
    "node_modules/axios": { "version": "1.6.0" }
  }
}`)
	out := filepath.Join(dir, "join.json")
	path, err := exportx.WriteWatchlistJoin(dir, out)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report exportx.WatchlistJoinReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) == 0 {
		t.Fatal("expected non-empty watchlist∩SBOM findings for axios@1.6.0")
	}
	if report.Status != "ok" {
		t.Fatalf("status=%q", report.Status)
	}
	blob := string(data)
	if strings.Contains(strings.ToLower(blob), `"cve-`) {
		t.Fatal("join must not invent CVE ids")
	}
	if !strings.Contains(strings.ToLower(report.Note), "informational") {
		t.Fatalf("note must stress informational: %q", report.Note)
	}
	found := false
	for _, f := range report.Findings {
		if f.Package == "axios" && f.Version == "1.6.0" {
			found = true
			if f.WatchlistID == "" {
				t.Fatal("missing watchlist_id")
			}
		}
	}
	if !found {
		t.Fatalf("axios@1.6.0 not joined: %#v", report.Findings)
	}
}

func mustRealGit(t *testing.T, dir string) {
	t.Helper()
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
	run("git", "config", "user.email", "exportx@cyberready.local")
	run("git", "config", "user.name", "ExportX")
	run("git", "commit", "--allow-empty", "-m", "init", "-q")
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

func writeMinimalHouseFail(t *testing.T, dir string) {
	t.Helper()
	mustWrite(t, filepath.Join(dir, "README.md"), "# Project\n")
	mustWrite(t, filepath.Join(dir, ".well-known/security.txt"), "Contact: mailto:a@b.c\nExpires: 2027-01-01T00:00:00.000Z\nPreferred-Languages: en\n")
}

func writeGoodHouse(t *testing.T, dir string) {
	t.Helper()
	mustWrite(t, filepath.Join(dir, "README.md"), "# Project\n")
	mustWrite(t, filepath.Join(dir, ".well-known/security.txt"), "Contact: mailto:a@b.c\nExpires: 2027-01-01T00:00:00.000Z\nPreferred-Languages: en\n")
	mustWrite(t, filepath.Join(dir, "SECURITY.md"), `# Security Policy

## Reporting

Report vulnerabilities to security@example.com with reproduction steps.

## Supported Versions

We support the latest major release with security patches for twelve months.

## Disclosure

Coordinated disclosure within 90 days after fix availability.
`+strings.Repeat("word ", 40)+"\n")
}
