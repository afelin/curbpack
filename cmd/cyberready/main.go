package main

import (
	"fmt"
	"os"

	"github.com/afelin/cyberready/internal/cli"
	"github.com/afelin/cyberready/internal/tty"
)

// Thin dispatcher — command logic lives in internal/cli.
func main() {
	err := cli.Run(os.Args[1:])
	if err == nil {
		return
	}
	if msg := err.Error(); msg != "" {
		fmt.Fprintf(os.Stderr, "%s\n", tty.C(tty.Red, msg))
	}
	os.Exit(cli.ExitCode(err))
}
