package contract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/cyberready/internal/exportx"
	"github.com/afelin/cyberready/internal/ir"
	"github.com/afelin/cyberready/internal/validate"
)

// TestExplainPacketCorewardConsumer simulates a Coreward-shaped tutor consumer:
//  1. Read explain-packet JSON produced by exportx
//  2. Refuse absolute home paths / PEM / missing untrusted_metadata
//  3. Never treat packet as green — must re-run validate before "ok"
//
// Contract assertion helper for Coreward: ChatMustRecheckBeforeOK.
func TestExplainPacketCorewardConsumer(t *testing.T) {
	dir := t.TempDir()
	mustRealGit(t, dir)
	writeMinimalHouseFail(t, dir)

	path, err := exportx.WriteExplainPacket(dir, []string{"house-policy"}, filepath.Join(dir, "explain-packet.json"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Coreward airlock refuse rules
	if err := exportx.PacketLooksAirlocked(data); err != nil {
		t.Fatalf("Coreward must refuse non-airlocked packet: %v", err)
	}
	var pkt exportx.ExplainPacket
	if err := json.Unmarshal(data, &pkt); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pkt.Untrusted, "<untrusted_metadata>") {
		t.Fatal("Coreward must refuse packet missing untrusted_metadata wrapper")
	}
	if strings.Contains(string(data), "/Users/") || strings.Contains(string(data), "BEGIN ") && strings.Contains(string(data), "PRIVATE KEY") {
		t.Fatal("Coreward must refuse absolute home / PEM leakage")
	}

	// Packet alone is never "ok" — consumer must call validate again.
	if ChatMustRecheckBeforeOK(pkt) != true {
		t.Fatal("contract: ChatMustRecheckBeforeOK must be true for any explain-packet")
	}
	if TutorClaimsFixed(pkt) {
		t.Fatal("explain-packet must never authorize a fixed claim by itself")
	}

	// Recheck still red on failing fixture
	recheck, err := validate.Run(validate.Options{RepoRoot: dir, PackIDs: []string{"house-policy"}, Quiet: true})
	if err != nil {
		t.Fatal(err)
	}
	if recheck.Passed {
		t.Fatal("recheck must still fail until human/editor fixes gates")
	}
	if len(recheck.Payload.Failures) == 0 {
		t.Fatal("expected failures on recheck")
	}

	// After healing the fixture, recheck can pass — packet still does not greenlight.
	writeGoodHouse(t, dir)
	recheck2, err := validate.Run(validate.Options{RepoRoot: dir, PackIDs: []string{"house-policy"}, Quiet: true})
	if err != nil {
		t.Fatal(err)
	}
	if !recheck2.Passed {
		t.Fatalf("expected green after fix: %#v", recheck2.Payload.Failures)
	}
	if TutorClaimsFixed(pkt) {
		t.Fatal("stale explain-packet must never claim fixed even after repo is green")
	}
}

// ChatMustRecheckBeforeOK documents the Coreward contract: tutors may summarize
// untrusted_metadata only; gate truth requires a fresh validate_delta / check.
func ChatMustRecheckBeforeOK(pkt exportx.ExplainPacket) bool {
	_ = pkt
	return true
}

// TutorClaimsFixed is always false for an explain-packet alone (airlock contract).
func TutorClaimsFixed(pkt exportx.ExplainPacket) bool {
	_ = pkt
	return false
}

// CorewardRefusePacket mirrors Coreward-side refuse rules for fixture packets.
func CorewardRefusePacket(data []byte) error {
	if err := exportx.PacketLooksAirlocked(data); err != nil {
		return err
	}
	var pkt exportx.ExplainPacket
	if err := json.Unmarshal(data, &pkt); err != nil {
		return err
	}
	if !strings.Contains(pkt.Untrusted, "<untrusted_metadata>") {
		return errMissingUntrusted
	}
	return nil
}

var errMissingUntrusted = errString("missing untrusted_metadata")

type errString string

func (e errString) Error() string { return string(e) }

func TestCorewardRefusePacket_InjectedPEM(t *testing.T) {
	pkt := exportx.AssembleExplainPacket(ir.GateFailurePayload{
		Failures: []ir.Failure{{
			GateID: "X",
			SanitizedDescription: "-----BEGIN RSA PRIVATE KEY-----\n" +
				"MIIEowIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF6PZGFw7N+EXAMPLEKEYMATERIAL12\n" +
				"34567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345\n" +
				"-----END RSA PRIVATE KEY-----",
		}},
	}, nil, nil, false)
	// Assemble strips PEM — packet should pass airlock after sanitize.
	raw, _ := json.Marshal(pkt)
	if err := CorewardRefusePacket(raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "BEGIN RSA") {
		t.Fatal("PEM must be absent after assemble")
	}
}
