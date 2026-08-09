package exportx_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/afelin/cyberready/internal/exportx"
)

func TestWriteBuyerQuestions_HousePolicy(t *testing.T) {
	dir := t.TempDir()
	mustRealGit(t, dir)
	writeMinimalHouseFail(t, dir)

	out := filepath.Join(dir, "buyer-questions.md")
	path, n, err := exportx.WriteBuyerQuestions(dir, []string{"house-policy"}, out)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("expected n>0 buyer questions for house-policy")
	}
	md, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(md)
	if !strings.Contains(text, "Not CE") || !strings.Contains(text, "not notified-body") {
		t.Fatal("markdown header must deny CE / notified-body")
	}
	if !strings.Contains(text, "structural_draft") {
		t.Fatal("every export must stamp structural_draft")
	}
	if !strings.Contains(text, "For human review:") {
		t.Fatal("questions must be prefixed For human review:")
	}
	deny := []string{"we are CE certified", "CE marking issued", "notified-body approved", "EU CRA Baseline"}
	lower := strings.ToLower(text)
	for _, d := range deny {
		if strings.Contains(lower, strings.ToLower(d)) {
			t.Fatalf("claim-unsafe phrase %q in buyer-questions", d)
		}
	}

	jsonPath := strings.TrimSuffix(path, ".md") + ".json"
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var report exportx.BuyerQuestionsReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Questions) == 0 {
		t.Fatal("json questions empty")
	}
	for _, q := range report.Questions {
		if q.AssuranceClass != "structural_draft" {
			t.Fatalf("row %s assurance_class=%q", q.GateID, q.AssuranceClass)
		}
		if !strings.HasPrefix(q.HumanQuestion, "For human review:") {
			t.Fatalf("bad prefix: %q", q.HumanQuestion)
		}
	}
}
