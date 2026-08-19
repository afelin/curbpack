package exportx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afelin/curbpack/internal/attest"
	"github.com/afelin/curbpack/internal/packs"
	"github.com/afelin/curbpack/internal/validate"
)

const buyerQuestionsAssuranceClass = "structural_draft"

// BuyerQuestion is one human-review checklist row for buyers/auditors.
type BuyerQuestion struct {
	GateID          string `json:"gate_id"`
	Severity        string `json:"severity"`
	HumanQuestion   string `json:"human_question"`
	ArtifactPath    string `json:"artifact_path"`
	AssuranceClass  string `json:"assurance_class"`
	RemediationHint string `json:"remediation_hint"`
}

// BuyerQuestionsReport is Markdown+JSON checklist export (claim-safe).
type BuyerQuestionsReport struct {
	SchemaVersion     string          `json:"schema_version"`
	Note              string          `json:"note"`
	PackID            string          `json:"pack_id"`
	AssuranceClass    string          `json:"assurance_class"`
	AttestationStatus string          `json:"attestation_status"`
	Questions         []BuyerQuestion `json:"questions"`
}

// CollectBuyerQuestions builds the same checklist rows WriteBuyerQuestions writes,
// without touching the filesystem. Used by prepare-release one-pager cover sheet.
func CollectBuyerQuestions(root string, packIDs []string, res validate.Result) ([]BuyerQuestion, error) {
	ids := packIDs
	if len(ids) == 0 {
		ids = nonzeroPacks(strings.Split(res.Payload.PackID, ","))
	}
	composed, _, err := packs.Compose(ids)
	if err != nil {
		return nil, err
	}
	failRem := map[string]string{}
	failPath := map[string]string{}
	for _, f := range res.Payload.Failures {
		failRem[f.GateID] = f.Remediation.ActionRequired
		if p := strings.TrimSpace(f.ASTCoordinates.TargetFile); p != "" {
			failPath[f.GateID] = p
		}
	}
	assurance := strings.TrimSpace(composed.AssuranceClass)
	if assurance == "" {
		assurance = buyerQuestionsAssuranceClass
	}
	questions := make([]BuyerQuestion, 0, len(composed.Rules))
	for _, r := range composed.Rules {
		path := strings.TrimSpace(r.Path)
		if path == "" && len(r.Paths) > 0 {
			path = strings.Join(r.Paths, ", ")
		}
		if fp, ok := failPath[r.ID]; ok {
			path = fp
		}
		hint := strings.TrimSpace(r.Remediation)
		if h, ok := failRem[r.ID]; ok && strings.TrimSpace(h) != "" {
			hint = h
		}
		questions = append(questions, BuyerQuestion{
			GateID:          r.ID,
			Severity:        r.Severity,
			HumanQuestion:   humanQuestionForRule(r),
			ArtifactPath:    path,
			AssuranceClass:  assurance,
			RemediationHint: hint,
		})
	}
	return questions, nil
}

// PackPlainNames returns human pack names for a CSV of pack ids (fallback: the id).
func PackPlainNames(packIDCSV string) string {
	var names []string
	for _, id := range strings.Split(packIDCSV, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		p, err := packs.LoadPack(id)
		if err != nil || strings.TrimSpace(p.Name) == "" {
			names = append(names, id)
			continue
		}
		names = append(names, strings.TrimSpace(p.Name))
	}
	return strings.Join(names, "; ")
}

// BuildBuyerQuestionsReport runs pack gates and assembles the checklist report (no filesystem writes).
func BuildBuyerQuestionsReport(root string, packIDs []string) (BuyerQuestionsReport, error) {
	return buildBuyerQuestionsReport(root, packIDs, false)
}

// BuildBuyerQuestionsReportReadOnly is like BuildBuyerQuestionsReport but does not write cache under .github/.
func BuildBuyerQuestionsReportReadOnly(root string, packIDs []string) (BuyerQuestionsReport, error) {
	return buildBuyerQuestionsReport(root, packIDs, true)
}

func buildBuyerQuestionsReport(root string, packIDs []string, readOnly bool) (BuyerQuestionsReport, error) {
	res, err := validate.Run(validate.Options{RepoRoot: root, PackIDs: packIDs, Quiet: true, ReadOnly: readOnly})
	if err != nil {
		return BuyerQuestionsReport{}, err
	}
	questions, err := CollectBuyerQuestions(root, packIDs, res)
	if err != nil {
		return BuyerQuestionsReport{}, err
	}
	ids := packIDs
	if len(ids) == 0 {
		ids = nonzeroPacks(strings.Split(res.Payload.PackID, ","))
	}
	composed, _, err := packs.Compose(ids)
	if err != nil {
		return BuyerQuestionsReport{}, err
	}
	assurance := strings.TrimSpace(composed.AssuranceClass)
	if assurance == "" {
		assurance = buyerQuestionsAssuranceClass
	}
	return BuyerQuestionsReport{
		SchemaVersion:     "1",
		Note:              "Local pack gates prepare evidence for human review. Not CE / not notified-body. Not a conformity assessment.",
		PackID:            composed.ID,
		AssuranceClass:    assurance,
		AttestationStatus: attestationStatus(root),
		Questions:         questions,
	}, nil
}

// WriteBuyerQuestions emits buyer-questions.md + .json under cache (or outPath stem).
func WriteBuyerQuestions(root string, packIDs []string, outPath string) (string, int, error) {
	report, err := BuildBuyerQuestionsReport(root, packIDs)
	if err != nil {
		return "", 0, err
	}
	mdPath, jsonPath := buyerQuestionsPaths(root, outPath)
	if err := writeBuyerQuestionsFiles(report, mdPath, jsonPath); err != nil {
		return "", 0, err
	}
	return mdPath, len(report.Questions), nil
}

// SupplierQuestionsPaths resolves review-pack/supplier-checklist paths (or --out stem).
func SupplierQuestionsPaths(root, outPath string) (mdPath, jsonPath string) {
	if outPath == "" {
		base := filepath.Join(root, "review-pack", "supplier-checklist")
		return base + ".md", base + ".json"
	}
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(root, outPath)
	}
	return buyerQuestionsStemPaths(outPath)
}

// WriteSupplierChecklist writes supplier-checklist.md + .json under review-pack/ (or outPath stem).
func WriteSupplierChecklist(root string, packIDs []string, outPath string) (string, int, error) {
	report, err := BuildBuyerQuestionsReportReadOnly(root, packIDs)
	if err != nil {
		return "", 0, err
	}
	return WriteSupplierChecklistReport(root, report, outPath)
}

// WriteSupplierChecklistReport writes a pre-built report to review-pack/ (or outPath stem).
func WriteSupplierChecklistReport(root string, report BuyerQuestionsReport, outPath string) (string, int, error) {
	mdPath, jsonPath := SupplierQuestionsPaths(root, outPath)
	if err := writeBuyerQuestionsFiles(report, mdPath, jsonPath); err != nil {
		return "", 0, err
	}
	return mdPath, len(report.Questions), nil
}

func writeBuyerQuestionsFiles(report BuyerQuestionsReport, mdPath, jsonPath string) error {
	if err := os.MkdirAll(filepath.Dir(mdPath), 0o755); err != nil {
		return err
	}
	md := FormatBuyerQuestionsMarkdown(report)
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jsonPath, append(b, '\n'), 0o644)
}

func buyerQuestionsPaths(root, outPath string) (mdPath, jsonPath string) {
	if outPath == "" {
		base := filepath.Join(root, ".github", "curbpack", "cache", "buyer-questions")
		return base + ".md", base + ".json"
	}
	return buyerQuestionsStemPaths(outPath)
}

func buyerQuestionsStemPaths(outPath string) (mdPath, jsonPath string) {
	ext := strings.ToLower(filepath.Ext(outPath))
	stem := strings.TrimSuffix(outPath, ext)
	switch ext {
	case ".md", ".markdown":
		return outPath, stem + ".json"
	case ".json":
		return stem + ".md", outPath
	default:
		return outPath + ".md", outPath + ".json"
	}
}

func humanQuestionForRule(r packs.Rule) string {
	body := strings.TrimSpace(r.Description)
	if body == "" {
		body = strings.TrimSpace(r.Expected)
	}
	if body == "" {
		body = "Is evidence for gate " + r.ID + " ready for human review?"
	}
	if !strings.HasSuffix(body, "?") {
		body = strings.TrimRight(body, ".") + "?"
	}
	return "For human review: " + body
}

// FormatBuyerQuestionsMarkdown renders the human-review checklist as Markdown.
func FormatBuyerQuestionsMarkdown(report BuyerQuestionsReport) string {
	var b strings.Builder
	b.WriteString("# Buyer questions (human review checklist)\n\n")
	b.WriteString("> Local pack gates. Humans review. Not conformity assessment.\n")
	b.WriteString("> Not CE / not notified-body.\n\n")
	fmt.Fprintf(&b, "- **Packs:** %s\n", report.PackID)
	fmt.Fprintf(&b, "- **Assurance class:** `%s`\n", report.AssuranceClass)
	fmt.Fprintf(&b, "- **Attestation status:** `%s`\n\n", report.AttestationStatus)
	b.WriteString("| gate_id | severity | human_question | artifact_path | assurance_class | remediation_hint |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, q := range report.Questions {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
			mdCell(q.GateID),
			mdCell(q.Severity),
			mdCell(q.HumanQuestion),
			mdCell(q.ArtifactPath),
			mdCell(q.AssuranceClass),
			mdCell(q.RemediationHint),
		)
	}
	b.WriteString("\n")
	return b.String()
}

// FormatSupplierEmailTemplate returns a claim-safe copy-paste email for suppliers.
func FormatSupplierEmailTemplate(report BuyerQuestionsReport) string {
	packLabel := strings.TrimSpace(report.PackID)
	if packLabel == "" {
		packLabel = "our product"
	}
	var b strings.Builder
	b.WriteString("Subject: Supplier evidence checklist — human review (not certification)\n\n")
	b.WriteString("Hi,\n\n")
	fmt.Fprintf(&b, "We are preparing structural evidence for %s under local pack gates. ", packLabel)
	b.WriteString("This is not a conformity assessment and does not claim CE marking or notified-body approval.\n\n")
	b.WriteString("Please review the checklist below (or the attached supplier-checklist.md) and confirm the artifact paths listed, or share your equivalent documentation.\n\n")
	b.WriteString("Thanks,\n")
	b.WriteString("[Your name]\n")
	return b.String()
}

func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// attestationStatus returns none | ssh-agent via LatestBind (not HEAD-only).
func attestationStatus(root string) string {
	bind, _ := attest.LatestBind(root)
	if !bind.Found {
		return "none"
	}
	if bind.UserTouch == "ssh-agent-signed" && bind.StateHash != "" {
		return "ssh-agent"
	}
	return "none"
}
