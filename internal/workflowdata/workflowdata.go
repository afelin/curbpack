package workflowdata

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed cyberready-check.yml
var workflowYAML []byte

// DestRel is the adopter path written by init --workflow.
const DestRel = ".github/workflows/cyberready.yml"

// Install writes the drop-in Action workflow if missing.
// Never overwrites an existing workflow. Opt-in only (callers gate on --workflow).
func Install(repoRoot string) (dest string, created bool, err error) {
	dest = filepath.Join(repoRoot, filepath.FromSlash(DestRel))
	if _, err := os.Stat(dest); err == nil {
		return dest, false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", false, err
	}
	if err := os.WriteFile(dest, workflowYAML, 0o644); err != nil {
		return "", false, err
	}
	return dest, true, nil
}

// Bytes returns the embedded workflow YAML (for parity tests).
func Bytes() []byte {
	if len(workflowYAML) == 0 {
		return nil
	}
	out := make([]byte, len(workflowYAML))
	copy(out, workflowYAML)
	return out
}

// MustMatchExample fails the process if embed drifts from examples/workflows (used by tests).
func MustMatchExample(examplePath string) error {
	b, err := os.ReadFile(examplePath)
	if err != nil {
		return err
	}
	if string(b) != string(workflowYAML) {
		return fmt.Errorf("embedded workflow diverges from %s — keep in sync", examplePath)
	}
	return nil
}
