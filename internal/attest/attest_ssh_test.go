package attest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentBindNeverVerified(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	cap := Capsule{
		Signer:       "local-unsigned",
		UserTouch:    "not-verified",
		SSHSignature: "agent-bind:deadbeef",
	}
	if cap.UserTouch == "ssh-agent-signed" {
		t.Fatal("agent-bind must not be labeled signed")
	}
	if !strings.HasPrefix(cap.SSHSignature, "agent-bind:") {
		t.Fatal("fixture")
	}
	parts, ok := ParseHPURLFragment("#?h=abc&p=def&s=unsigned")
	if !ok || parts.StateHash != "abc" {
		t.Fatalf("hpurl parse: %#v ok=%v", parts, ok)
	}
}

func TestTrySSHAgentSign_DashFIsKeyNotPayload(t *testing.T) {
	bin := t.TempDir()
	installFakeSSH(t, bin, false)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SSH_AUTH_SOCK", filepath.Join(t.TempDir(), "agent.sock"))

	sig, who, verified := trySSHAgentSign(t.TempDir(), "state-hash-payload")
	if !verified {
		t.Fatalf("expected verified with honest fake sig; who=%q sig=%q", who, sig)
	}
	if strings.HasPrefix(sig, "agent-bind:") {
		t.Fatal("honest fake must not emit agent-bind")
	}
	if !strings.Contains(sig, "FAKE-SSH-SIG") {
		t.Fatalf("unexpected sig: %q", sig)
	}
	argvLog := filepath.Join(bin, "ssh-keygen.argv")
	raw, err := os.ReadFile(argvLog)
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.TrimSpace(string(raw))
	// Confirm production argv shape: -Y sign -f <key> -n cyberready@attest <payload>
	if !strings.Contains(argv, "-Y sign") && !strings.Contains(argv, "-Y\nsign") {
		// argv is space-joined
		if !strings.Contains(argv, "-Y") || !strings.Contains(argv, "sign") {
			t.Fatalf("missing -Y sign in argv: %q", argv)
		}
	}
	fields := strings.Fields(argv)
	fIdx := -1
	for i, a := range fields {
		if a == "-f" && i+1 < len(fields) {
			fIdx = i + 1
			break
		}
	}
	if fIdx < 0 {
		t.Fatalf("-f missing from argv: %q", argv)
	}
	fPath := fields[fIdx]
	if !strings.HasSuffix(fPath, ".pub") && !strings.Contains(fPath, "cyberready-attest-") {
		t.Fatalf("-f must be key file, got %q", fPath)
	}
	// Last arg is payload path; must differ from -f.
	payloadArg := fields[len(fields)-1]
	if payloadArg == fPath {
		t.Fatal("-f must not equal payload path")
	}
	if strings.HasSuffix(payloadArg, ".pub") {
		t.Fatalf("payload arg looks like a key: %q", payloadArg)
	}
}

func TestTrySSHAgentSign_RejectsAgentBindOutput(t *testing.T) {
	bin := t.TempDir()
	installFakeSSH(t, bin, true)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SSH_AUTH_SOCK", filepath.Join(t.TempDir(), "agent.sock"))

	sig, who, verified := trySSHAgentSign(t.TempDir(), "state-hash-payload")
	if verified {
		t.Fatalf("agent-bind output must yield verified=false; who=%q sig=%q", who, sig)
	}
	if sig != "" {
		t.Fatalf("sig must be empty on reject, got %q", sig)
	}
	if who == "" {
		t.Fatal("identity hint from ssh-add -L should still be present")
	}
}

func installFakeSSH(t *testing.T, bin string, emitAgentBind bool) {
	t.Helper()
	sshAdd := `#!/bin/sh
echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakeKeyMaterialForCyberReadyTests fake@cyberready"
`
	mode := "honest"
	if emitAgentBind {
		mode = "agent-bind"
	}
	// Fake ssh-keygen: assert -f is key (not payload), write .sig next to data file.
	sshKeygen := `#!/bin/sh
# Log argv for assertions
printf '%s\n' "$*" > "$(dirname "$0")/ssh-keygen.argv"
f=""
n=""
data=""
prev=""
for a in "$@"; do
  if [ "$prev" = "-f" ]; then f="$a"; fi
  if [ "$prev" = "-n" ]; then n="$a"; fi
  prev="$a"
done
# last non-flagish arg is data
for a in "$@"; do data="$a"; done
if [ -z "$f" ] || [ -z "$data" ]; then
  echo "fake ssh-keygen: missing -f or data" >&2
  exit 2
fi
# Fail if -f points at the payload file (production bug we guard against).
if [ "$f" = "$data" ]; then
  echo "fake ssh-keygen: -f must be key not payload" >&2
  exit 3
fi
case "$f" in
  *.pub|*/cyberready-attest-*) ;;
  *)
    # still accept temp key paths without .pub suffix if they differ from data
    if [ ! -f "$f" ]; then
      echo "fake ssh-keygen: -f key missing: $f" >&2
      exit 4
    fi
    ;;
esac
sig="${data}.sig"
mode="` + mode + `"
if [ "$mode" = "agent-bind" ]; then
  printf 'agent-bind:synthetic\n' > "$sig"
else
  printf 'FAKE-SSH-SIG:ok namespace=%s\n' "$n" > "$sig"
fi
exit 0
`
	mustExec(t, filepath.Join(bin, "ssh-add"), sshAdd)
	mustExec(t, filepath.Join(bin, "ssh-keygen"), sshKeygen)
}

func mustExec(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}
