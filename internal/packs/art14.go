package packs

import (
	"strings"

	"github.com/afelin/curbpack/internal/clock"
)

const art14RelPath = "docs/incident/art14-path.md"

// Art14RelPath is the single in-repo Art 14 reporting-path file fix --art14 writes.
func Art14RelPath() string { return art14RelPath }

// Art14PathBody returns product-specific Art 14 reporting-path prose (not DefaultScaffoldBody).
// Drafted: is tool-written; Last tabletop: is left empty for human fill after tabletop.
func Art14PathBody(productName string) string {
	product := strings.TrimSpace(productName)
	if product == "" {
		product = "this product"
	}
	today := clock.NowUTC().Format("2006-01-02")
	return `# Art 14 reporting path

## Reporting clock (CRA Art 14)

For ` + product + `, actively exploited or severe incidents are reported by the on-call owner using the in-repo reporting path documented below. This is a file record for CRA Article 14 reporting (clock from 11 September 2026, including products already on the market). It is not a live Single Reporting Platform check and does not assert that EU Login works.

## Handling clock (not this file)

Vulnerability handling and public security contact for ` + product + ` are documented separately. They sit on a later clock than Article 14 reporting and are not this file.

## Named owner

Product security on-call for ` + product + ` owns the reporting path. Escalation is the engineering manager of record in SECURITY.md.

## Rehearsal dated artifact

Rehearsal status: unrehearsed draft — fill Last tabletop after tabletop

Drafted: ` + today + `
Last tabletop:
Record: this file plus the incident mail template under docs/incident/ (in-repo). Not a live submission.
`
}
