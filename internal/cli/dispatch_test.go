package cli_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/afelin/cyberready/internal/cli"
)

func TestRun_Version(t *testing.T) {
	stdout, stderr := capture(t, func() {
		err := cli.Run([]string{"version"})
		if err != nil {
			t.Fatalf("version: %v", err)
		}
		if cli.ExitCode(err) != cli.ExitOK {
			t.Fatal("exit")
		}
	})
	_ = stderr
	if !strings.Contains(stdout, "cyberready") {
		t.Fatalf("stdout=%q", stdout)
	}
	if !strings.Contains(stdout, cli.Version) {
		t.Fatalf("missing version in %q", stdout)
	}
}

func TestRun_Help(t *testing.T) {
	_, stderr := capture(t, func() {
		if err := cli.Run([]string{"help"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(stderr, "Usage: cyberready") {
		t.Fatalf("help stderr=%q", stderr)
	}
}

func TestRun_UnknownCommandExit2(t *testing.T) {
	_, stderr := capture(t, func() {
		err := cli.Run([]string{"not-a-real-cmd"})
		if cli.ExitCode(err) != cli.ExitUsage {
			t.Fatalf("want exit 2, got %d (%v)", cli.ExitCode(err), err)
		}
	})
	if !strings.Contains(stderr, "Usage: cyberready") {
		t.Fatalf("unknown should print usage; stderr=%q", stderr)
	}
}

func capture(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = wOut, wErr
	defer func() {
		os.Stdout, os.Stderr = oldOut, oldErr
	}()

	doneOut := make(chan string)
	doneErr := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		doneOut <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		doneErr <- buf.String()
	}()

	fn()
	_ = wOut.Close()
	_ = wErr.Close()
	stdout = <-doneOut
	stderr = <-doneErr
	_ = rOut.Close()
	_ = rErr.Close()
	return stdout, stderr
}
