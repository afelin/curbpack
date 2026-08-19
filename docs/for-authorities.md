# For authorities, auditors, and CISOs

Curbpack prepares **structural evidence** for **human review**; it does not perform **conformity assessment**.

Sign-off oriented brief for NCSC/EU-style authorities, internal auditors, and CISOs who need to know what Curbpack is — without reading integrator notes or any optional sibling product.

Voice: [voice and terms](voice-and-terms.md). Abbreviations: [glossary and audience](glossary-and-audience.md) (CE, CRA, SBOM, SARIF, GRC, notified body, conformity assessment, RISE, NCSC).

## What Curbpack is

- A **local-first command-line tool** that evaluates **rule packs** (JSON checklists shaped like regulatory annex drafts—not law) against a git repository on the supplier’s machine.
- A producer of **structural evidence** (JSON + markdown + optional buyer one-pager) for **human review**.
- Documentation and dependency checks for product repos — pair with SCA and secret scanners for depth; not a full security program.
- Default pack is **house-policy**. Cyber Resilience Act (CRA)–shaped packs are opt-in only.

## Evidence artifact catalog (trust levels)

| Artifact | How produced | Trust level (honest) |
|----------|--------------|----------------------|
| **Scan output** (`scan`, `npx curbpack scan`) | Read-only pack evaluation; no cache, no init | **Structural diagnosis** — defaults to `cra-baseline` on cold repos; not certification |
| Gate JSON / action report (`check`, `validate`) | Local deterministic pack evaluation | **Structural evidence** — reproducible on the same tree; not a legal finding |
| SARIF export | `export --sarif` / Action upload | Same findings in IDE/CI format; still pack gates, not certification |
| Buyer-questions checklist | `export --buyer-questions` | Human Q&A aid; rows carry `assurance_class: structural_draft` |
| ContextPack | `export --context-pack` (or `share`) | One washed assistant/auditor snapshot — still structural evidence, not certification |
| Lay-of-land | `export --lay-of-land` | Shareable map (deps summary, secret-hit count, informational watchlist∩SBOM) — **not** a CVE product |
| Review pack + buyer one-pager | `prepare-release` | Procurement snapshot; **not** a certificate of conformity |
| CycloneDX SBOM (best-effort) | From common lockfiles when present | Inventory draft; completeness depends on lockfiles |
| OpenVEX draft | Bound at attest time when applicable | Draft exploitability notes — not a vulnerability program |
| Git Notes attest capsule | Human `attest` | **ssh-agent-signed** = cryptographic signature present; **unsigned** = present but **not** cryptographically verified |
| Explain-packet | `export --explain-packet` | Sanitized tutor surface; **never** greenlights gates |

**Unsigned ≠ verified.** Local gate score on this tree is not a certification score.

## What Curbpack is not

- **Not** conformity assessment, CE marking, or notified-body approval.
- **Not** a certificate that green gates equal legal market access or CRA conformity.
- **Not** a GRC SaaS, cloud policy brain, or LLM-as-judge.
- **Not** an official NCSC/FRA/agency product or endorsement vehicle.
- **Not** dependent on any private tutor product to operate.

Gate pass means: deterministic pack rules did not fail on the files present — a human still judges risk, annex drafts, and legal posture.

## Institute neutrality

Development supported by RISE Research Institutes of Sweden as an applied research / competence object. RISE does not certify products that use Curbpack gate results. Never claim “RISE-approved,” “NCSC-approved,” or agency-endorsed product claims.

Full MoU / co-promotion boundary: [promotion firewall](promotion-firewall.md).

## Offline

- Daily `check` needs **no** remote policy service.
- Packs ship **embedded**; refresh via offline `packs import`, or network update only with an explicit sha256 pin.
- Install / Action downloads verify release `checksums.txt` (sha256, fail-closed).
- Optional tutors (any chat) receive only sanitized explain-packets; raw source is not the default export path.

## Claim-safe boundaries (sign-off language)

Safe to say:

- “Prepares structural evidence for human review.”
- “Local pack gates; humans retain judgment.”
- “Not a conformity assessment / not CE / not notified-body.”

Unsafe / forbidden as product claims:

- Never claim that Curbpack certified the product, completed conformity assessment, or issued CE marking.
- Never claim that gate green equals CRA-compliant, NIS2-compliant, or market-ready by law.
- Never claim that RISE, NCSC, FRA, or any agency endorses or approves the tool as official guidance.

CI enforces wording via `scripts/claim-safety.sh`. This page is **not** certification theater and does not replace legal counsel or a notified body.

## Pack chooser (what suppliers should run)

**Scan** on a cold repo defaults to **`cra-baseline`** (Art 14 clock + CRA-shaped gates, read-only).  
**Init / check** cold start defaults to **`house-policy`**. CRA-style (`cra-baseline`) and medtech (`medtech-iec62304`) for init are also available via `--packs` — catalog frozen until freeze review. Assistants and auditors should ask which pack ids were composed (see ContextPack `pack_ids` / GateFailure `pack_id`). Details: [assistant-loop pack chooser](assistant-loop.md#pack-chooser-cold-start).

CRA Art 14 reporting (11 September 2026) is not the same clock as vulnerability-handling / public SPOC. Opt-in `cra-baseline` may require an in-repo dated rehearsal file — not a live SRP or EU Login check. AI Act Art 50 marking grace is **not** blanket (only systems already on the market before 2 August 2026). Counsel note: [Art 14 reporting vs handling](getting-started/art14-reporting-vs-handling.md).

## Suggested reviewer path

1. Read this page + [Intent vs Scope](intent-vs-scope.md).
2. Ask the supplier for ContextPack / buyer-questions / one-pager + attest status (`curbpack share` recipe).
3. Optionally deep-read [security model](security-model.md) and the [white paper](../papers/curbpack-whitepaper.md).
4. Do **not** require [coreward-bridge.md](coreward-bridge.md) — that file is for integrators only. Coreward stays an external optional pointer — not in activation.

Site mirror: [for-authorities on Pages](../site/for-authorities/).

---

> **Optional, separate product:** Coreward is a private tutor/enforce client that may consume Curbpack explain-packets over an optional Unix socket. Curbpack is fully self-sustaining without it — adopters do not need Coreward. Brief architecture note (public Pages, not the private repo): https://afelin.github.io/coreward/

(Wording source: [coreward-pointer.md](coreward-pointer.md).)
