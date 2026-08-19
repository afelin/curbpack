# ENISA SME maturity ↔ Curbpack pack mapping

**Informational mapping only — not conformity assessment.**  
`assurance_class: structural_draft` — Curbpack rules check file/header/dependency **structure** for human review, not ENISA maturity scores or CRA essential-requirements conformity.

This table maps the five domains and twenty-five criteria from [ENISA's SME Cyber Resilience Maturity Assessment Model](https://www.enisa.europa.eu/publications/sme-cyber-resilience-maturity-assessment-model) to the embedded packs `cra-baseline`, `house-policy`, and `medtech-iec62304`. Criteria text is summarized from ENISA domain descriptions — see ENISA for authoritative wording.

CRA legal text: [EUR-Lex CELEX:32024R2847](https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX:32024R2847).

**Gap rows** mark criteria with no dedicated Curbpack rule today. Passing mapped gates does **not** certify ENISA advanced maturity or CRA conformity.

| ENISA domain | Criterion (1–5) | Summary | Pack | Rule ID | Notes |
|--------------|-----------------|---------|------|---------|-------|
| Governance and documentation | 1 | Defined roles and responsibilities for product security | cra-baseline | CRA-ART14-PATH | Structural **Named owner** header only — not org chart |
| Governance and documentation | 2 | Management-approved cybersecurity policies | — | **gap** | No policy-approval gate |
| Governance and documentation | 3 | Product documentation (features, assumptions, limits) | cra-baseline | CRA-ANNEX-VII-RISK, CRA-ANNEX-VII-MANUAL | Annex VII draft structure |
| Governance and documentation | 4 | Supplier / third-party cybersecurity expectations | — | **gap** | No supplier-contract gate |
| Governance and documentation | 5 | Management review of risks and compliance | — | **gap** | No management-review artifact gate |
| Risk management and security by design/default | 1 | Product-specific cybersecurity risks | cra-baseline | CRA-ANNEX-VII-RISK | Risk assessment draft sections |
| Risk management and security by design/default | 2 | Threats, misuse, attack surfaces | cra-baseline | CRA-ANNEX-VII-RISK | ## Identified Risks section |
| Risk management and security by design/default | 3 | Third-party / supply-chain dependencies | cra-baseline, medtech-iec62304 | CRA-DEP-AXIOS-PIN, MD-SOUP | Pin ban + SOUP list draft — not full SCA |
| Risk management and security by design/default | 4 | Security in design and development | — | **gap** | No SDLC process gate |
| Risk management and security by design/default | 5 | Secure by default / reduced exposure | cra-baseline | CRA-ANNEX-VII-MANUAL | ## Secure Configuration section |
| Vulnerability and patch management | 1 | Identify, receive, track vulnerabilities | house-policy, cra-baseline | HOUSE-SECURITY-MD, CRA-ART14-PATH | Disclosure path + Art 14 rehearsal — not live SRP |
| Vulnerability and patch management | 2 | External advisories / public databases | — | **gap** | No advisory-ingest gate |
| Vulnerability and patch management | 3 | Prioritise by risk / impact | — | **gap** | No triage-process gate |
| Vulnerability and patch management | 4 | Develop, test, distribute updates | cra-baseline | CRA-DEP-AXIOS-PIN | Banned vulnerable pin only |
| Vulnerability and patch management | 5 | Communicate vulnerabilities to users | house-policy | HOUSE-SECURITY-TXT | security.txt structural gate |
| Product life cycle management | 1 | Security support periods | cra-baseline | CRA-ANNEX-VII-SUPPORT | Support period draft |
| Product life cycle management | 2 | Updates throughout support period | cra-baseline | CRA-ANNEX-VII-SUPPORT | End-of-support section |
| Product life cycle management | 3 | Life-cycle security processes | medtech-iec62304 | MD-PROBLEM-RESOLUTION | Problem-resolution draft |
| Product life cycle management | 4 | Communicate support / EoL timelines | cra-baseline | CRA-ANNEX-VII-SUPPORT | Structural headers only |
| Product life cycle management | 5 | Secure retirement / replacement | cra-baseline | CRA-ANNEX-VII-MANUAL | ## Product Disposal section |
| Awareness, competence and skills | 1 | Basic product security awareness | — | **gap** | No training-record gate |
| Awareness, competence and skills | 2 | Role-appropriate guidance / training | — | **gap** | No training artifact gate |
| Awareness, competence and skills | 3 | Secure dev/config for technical roles | — | **gap** | No competence gate |
| Awareness, competence and skills | 4 | Share lessons learned | medtech-iec62304 | MD-PROBLEM-RESOLUTION | Process doc only — not incident DB |
| Awareness, competence and skills | 5 | Use external security information | — | **gap** | No threat-intel gate |

## Cross-pack rules (all CRA-shaped trees)

| Rule ID | Pack(s) | Role in mapping |
|---------|---------|-----------------|
| CRA-ANTI-PLACEHOLDER | cra-baseline (+ medtech overlay) | Rejects placeholder boilerplate in mapped annex drafts |
| MD-ANTI-PLACEHOLDER | medtech-iec62304 | Same for medtech paths |
| HOUSE-ANTI-PLACEHOLDER | house-policy | Placeholder guard on security docs |
| HOUSE-SECRET-PATHS | house-policy | Secret-like patterns in policy paths |

## How to use

1. Run read-only `curbpack scan` (defaults to `cra-baseline` on cold repos).
2. Use this table to see which ENISA criteria have **structural_draft** file gates vs explicit **gap** rows.
3. For gaps, add human evidence outside Curbpack — do not invent ENISA endorsement or CRA conformity claims.

See also: [for-authorities.md](../for-authorities.md) · [promotion-firewall.md](../promotion-firewall.md) · [voice-and-terms.md](../voice-and-terms.md).
