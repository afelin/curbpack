package exportx_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/cyberready/internal/exportx"
	"github.com/afelin/cyberready/internal/ir"
)

func TestWriteExplainPacket_NoAbsHome(t *testing.T) {
	dir := t.TempDir()
	mustRealGit(t, dir)
	writeMinimalHouseFail(t, dir)

	// Inject absolute home paths into a synthetic failure description path via Assemble.
	pkt := exportx.AssembleExplainPacket(ir.GateFailurePayload{
		PackID: "house-policy",
		Failures: []ir.Failure{{
			GateID:               "HOUSE-SECURITY-MD",
			Severity:             "high",
			SanitizedDescription: "missing file under /Users/alice/secret/repo/SECURITY.md",
			ASTCoordinates: ir.ASTCoordinates{
				TargetFile: "/Users/alice/secret/repo/SECURITY.md",
			},
			Remediation: ir.Remediation{
				ActionRequired: "create /Users/alice/secret/repo/SECURITY.md",
				ExpectedState:  "file exists at /home/bob/project/SECURITY.md",
			},
		}},
	}, nil, nil, false)

	raw, err := json.Marshal(pkt)
	if err != nil {
		t.Fatal(err)
	}
	if err := exportx.PacketLooksAirlocked(raw); err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, "/Users/") || strings.Contains(s, "/home/") {
		t.Fatalf("absolute home paths leaked: %s", s)
	}
	home, _ := os.UserHomeDir()
	if home != "" && strings.Contains(s, home) {
		t.Fatalf("$HOME leaked: %s", home)
	}

	// End-to-end write also airlocked.
	path, err := exportx.WriteExplainPacket(dir, []string{"house-policy"}, filepath.Join(dir, "explain.json"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := exportx.PacketLooksAirlocked(data); err != nil {
		t.Fatal(err)
	}
}

func TestWriteExplainPacket_NoPEM(t *testing.T) {
	pem := "-----BEGIN RSA PRIVATE KEY-----\n" +
		"MIIEowIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF6PZGFw7N+EXAMPLEKEYMATERIAL12\n" +
		"34567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345\n" +
		"-----END RSA PRIVATE KEY-----"
	pkt := exportx.AssembleExplainPacket(ir.GateFailurePayload{
		PackID: "house-policy",
		Failures: []ir.Failure{{
			GateID:               "HOUSE-SECURITY-MD",
			Severity:             "high",
			SanitizedDescription: "leak: " + pem,
			Remediation: ir.Remediation{
				ActionRequired: "rotate key that looked like " + pem,
			},
		}},
	}, nil, nil, false)

	raw, err := json.Marshal(pkt)
	if err != nil {
		t.Fatal(err)
	}
	if err := exportx.PacketLooksAirlocked(raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "BEGIN RSA PRIVATE KEY") {
		t.Fatal("PEM blob must be stripped from explain-packet")
	}
	if !strings.Contains(string(raw), "[REDACTED_PEM]") {
		t.Fatal("expected REDACTED_PEM marker")
	}
}

func TestWriteExplainPacket_UntrustedWrapper(t *testing.T) {
	dir := t.TempDir()
	mustRealGit(t, dir)
	writeMinimalHouseFail(t, dir)
	path, err := exportx.WriteExplainPacket(dir, []string{"house-policy"}, filepath.Join(dir, "pkt.json"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var pkt exportx.ExplainPacket
	if err := json.Unmarshal(data, &pkt); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pkt.Untrusted, "<untrusted_metadata>") || !strings.Contains(pkt.Untrusted, "</untrusted_metadata>") {
		t.Fatalf("agent-facing body missing untrusted_metadata wrapper: %q", pkt.Untrusted)
	}
	if !strings.Contains(string(data), "<untrusted_metadata>") {
		t.Fatal("serialized packet missing literal untrusted_metadata tags")
	}
}
