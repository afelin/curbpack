# Promotion firewall

**Local pack gates. Humans review. Not conformity assessment.**

Repo anchor for MoUs and co-promotion. Keep public language institute-neutral and claim-safe.

## Forbidden phrases

Never claim institute or agency endorsement (examples that must not appear as product claims):

- Never claim “RISE-approved”, “RISE-certified”, or “agency-endorsed”
- Never claim “NCSC-approved”, “FRA-approved”, or that this tool is official guidance
- Never claim CE certification theater, completed conformity assessment, or notified-body approval
- Never claim that CyberReady+ gate green equals legal conformity

## Allowed phrases

- “Institute-supported OSS evidence habit tool”
- “Development supported by RISE as an applied research / competence object”
- “Prepares structural evidence for human review”
- “Not a conformity assessment” / “Not CE / not notified-body”
- Awareness links to public NCSC/FRA materials **without** endorsement claims

## Separate lines of work

| Line | Role |
|------|------|
| **CyberReady+ (this repo)** | Local deterministic judge + human-review evidence packs. Apache-2.0 AS IS. |
| **RISE Center for Cybersecurity (accreditation / services)** | Separate commercial/accreditation services — **not** this product’s certifier. |
| **NCSC / FRA promotion** | Awareness only. Do not imply official product endorsement. |

RISE funding acknowledgment lives in [`NOTICE`](../NOTICE). Copyright remains with CyberReady+ contributors unless counsel assigns otherwise.

## MoU exhibit checklist (1 page)

1. Cite this file + Intent vs Scope as the public claim boundary.
2. State RISE is funder / applied-research supporter — **not** product certifier.
3. State CyberReady+ does not perform conformity assessment or CE marking.
4. Name pack catalog freeze (`house-policy`, `cra-baseline`, `medtech-iec62304`) and trust-surface freeze.
5. Prefer pin tag + checksum verify over floating `latest`.
6. Any co-branded materials must pass `scripts/claim-safety.sh` wording.
7. Do **not** publish via `docs/gtm-oss/` on product Pages.

See also: [Intent vs Scope](intent-vs-scope.md) · [Security model](security-model.md) · [Design partners](design-partners.md) · [CONTRIBUTING](../CONTRIBUTING.md)
