package exportx_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/cyberready/internal/exportx"
)

func TestWriteLayOfLand(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/demo\n\ngo 1.23\n")
	mustWrite(t, filepath.Join(dir, "README.md"), "# demo\n")

	path, err := exportx.WriteLayOfLand(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if !strings.Contains(body, "Instrument panel") {
		t.Fatalf("missing covenant:\n%s", body)
	}
	if !strings.Contains(body, "buyer-questions") {
		t.Fatalf("missing buyer pointer:\n%s", body)
	}
	jsonPath := strings.TrimSuffix(path, ".md") + ".json"
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatal("missing json twin")
	}
}
