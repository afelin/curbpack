# Launch readiness

Public Apache-2.0 launch checklist for [afelin/curbpack](https://github.com/afelin/curbpack).
Coreward is **not** required to build, test, launch, or use Curbpack (optional `sock` only).

## Status (2026-08-12)

| Item | State |
|------|--------|
| CI on `main` | Includes `test` matrix (`ubuntu-latest`, `macos-latest`, `windows-latest`) + **`windows-smoke`** (`windows-latest`) + `smoke` / `gauntlet` / `redteam-pilot` |
| Required checks on `main` | **Configured via API** — exact names below; `strict` (branches up to date) **on**; `enforce_admins` **on** |
| SPDX | **Apache-2.0** |
| Release pin | **`@v0.5.2`** — cut only after merge: [release-v0.5.2 playbook](getting-started/release-v0.5.2.md) |
| Trust freeze (v0.5.2) | **Distribution + UX + repair** (Windows exe, `install.ps1`, `doctor --repair`, dual-fence docs). **Trust track deferred** — no Action resolve / SafeJoin / attest OCC / explain-airlock rewrite |
| Discussion #4 body | **Verified** — claim-safe line + install ladder + Tester report pointer |
| Discussion #4 pin | **Pinned** — confirmed via GraphQL `pinnedDiscussions` (2026-08-10); [Welcome to Curbpack](https://github.com/afelin/curbpack/discussions/4) |
| Tier-3 human pass | **Recorded 2026-08-10** on [Discussion #4](https://github.com/afelin/curbpack/discussions/4#discussioncomment-17960761) — install.sh → doctor → demo PASS (`@v0.5.0`, isolated HOME/PATH) |
| Invite wave | **Ready/open** — items 1–4 done (Welcome Discussion pinned) |
| Understandability (cold-reader four-question bar) | **Done** (2026-08-10) — public rewrite shipped; `scripts/claim-safety.sh` OK; home/builders free of TTFV/HPURL/RKG/IR/airlock/covenant/Δ; sample one-pager shows before/after findings; voice: [voice and terms](voice-and-terms.md) |

Gap matrix: [github-readiness-gaps.md](github-readiness-gaps.md).

### Prior status snapshot (2026-08-10)

CI tip `f714700` verified green for `test (ubuntu-latest)`, `test (macos-latest)`, `smoke`, `gauntlet`, `redteam-pilot` before Cross-OS TAM. Pin at that time was `@v0.5.0`.

### Cold-reader bar (understandability)

From the public home alone, under two minutes, a builder / buyer / CISO each answer:

1. What is it? — Local rule packs → review pack for buyers/auditors; no certification claim.
2. What do I run? — Three first moves (safe try / product repo / CI) via Builders.
3. What do I get? — Gate results + review pack / buyer one-pager for human review.
4. What must I not claim? — Not conformity assessment / CE / notified-body (fence + RISE does not certify).

**Recorded:** Done after public language rewrite (2026-08-10).

### Branch protection (already set; Settings mirror)

Settings → Branches → `main` rule should require:

- `test (ubuntu-latest)`
- `test (macos-latest)`
- `smoke`
- `gauntlet`
- `redteam-pilot`
- Require branches to be up to date before merging
- Do not bypass for admins on routine pushes (`enforce_admins`)

Verify: `gh api repos/afelin/curbpack/branches/main/protection --jq '.required_status_checks'`

## Required GitHub checks (merge gate)

Configure these as **required status checks** on `main` (Settings → Branches → branch protection). Names must match CI check-run strings exactly:

| Check | Workflow job | Kill if |
|-------|--------------|---------|
| `test (ubuntu-latest)` | `.github/workflows/ci.yml` → `test` (ubuntu) | `go test ./...` fails |
| `test (macos-latest)` | `ci.yml` → `test` (macos) | `go test ./...` fails |
| `smoke` | `ci.yml` → `smoke` | doctor/demo or CRA/house happy paths fail |
| `gauntlet` | `ci.yml` → `gauntlet` | claim-safety, heal smoke, baseline ratchet, dead-ends, install-from-release |
| `redteam-pilot` | `ci.yml` → `redteam-pilot` | `./scripts/redteam-pilot.sh` not 13/13 |

Optional (not merge-blocking): `gauntlet-nightly` in `.github/workflows/gauntlet.yml` (realish + adversarial depth + optional OSS clone crash/hang only).

## License hygiene

- `LICENSE` — full Apache-2.0 text (GitHub SPDX)
- `NOTICE` — attribution
- `SECURITY.md` — claim-safe disclosure path
- README license badge → Apache-2.0

After merge, confirm: `gh api repos/afelin/curbpack --jq .license.spdx_id` → `Apache-2.0`.
If still `NOASSERTION`, wait for GitHub re-detect (pin `@v0.5.2` ships with license hygiene).

## Heal (deterministic — not ML)

```bash
curbpack check --heal
```

Loop (max 3): check → form-hints → apply-stub (**missing/empty only**) → remediations cache → re-check.
Cache: `.github/curbpack/cache/remediations.json` keyed by `gate_id`.

**Never** auto-`attest`. Never invents legal prose as approved evidence. Never marks VEX final.

## Claim safety

`scripts/claim-safety.sh` scans docs + runtime CLI captures (doctor/demo/check/prepare-release).
Deny-list blocks certification theater; negation / claim-safe framing is allowed.

## How we know activation works

- Market promise: a stranger’s **first green &lt;10 minutes** on pin `@v0.5.2` (safe try / product repo / CI).
- Maintainer harness: [`scripts/time-to-green.sh`](../scripts/time-to-green.sh) defaults to a **600s** wall-clock bar; use `TTG_MAX_SECONDS=60` for a tight CI smoke.
- Merge gate: required check **`redteam-pilot`** (`./scripts/redteam-pilot.sh` 13/13). No public vanity counter.
- Gap matrix (stakeholder demand → Evidence): [github-readiness-gaps.md](github-readiness-gaps.md).

## Tier 3 — human pass (before invite wave)

Before inviting external testers:

1. Fresh machine / no Go: `install.sh` → `doctor` → `demo` — first green in under **10 minutes** (often much faster)
2. Decision-maker understands: evidence for humans — **not** certification
3. `SECURITY.md` reporting path is usable

**Pass recorded:** 2026-08-10 on pinned Discussions welcome thread ([comment](https://github.com/afelin/curbpack/discussions/4#discussioncomment-17960761)).

## Discussions welcome (claim-safe)

Welcome thread: https://github.com/afelin/curbpack/discussions/4

**Pin:** **Pinned** (GraphQL `pinnedDiscussions`, 2026-08-10). Body must keep:

> Prepares evidence for human review — **not** a conformity assessment, CE mark, or certification.
>
> Try (read-only): `npx curbpack@0.5.2 scan`  
> Try (safe sandbox): `curl -fsSL https://raw.githubusercontent.com/afelin/curbpack/main/scripts/install.sh | sh && curbpack doctor && curbpack demo`
>
> Tester reports: use the **Tester report** issue template.

## Invite wave gate

Invite only when:

1. Tier 0–1 CI green on `main` (required checks above) — **done**
2. SPDX shows Apache-2.0 — **done**
3. One Tier-3 human pass recorded — **done (2026-08-10)**
4. Welcome Discussion pinned — **done**

## Object-owner cadence (30-day freeze)

| Cadence | Action |
|---------|--------|
| Every merge | Required check **`redteam-pilot`** |
| Weekly | `./scripts/time-to-green.sh`; skim [design-partner](design-partners.md) notes; no new trust-surface features |
| Biweekly | Decide kill/keep on first-move friction |
| Day 30 | Freeze review: renew, narrow, or next bugfix only (`v0.5.2` = distribution + UX + repair; Trust track deferred) |

### Freeze review (day-30 from v0.4.0)

| Field | Value |
|-------|--------|
| Freeze start | `v0.4.0` (2026-08-08) |
| Day-30 due | **2026-09-07** |
| Status | **Not yet due** — do not treat this as a completed review |
| Interim stance (until due) | **Renew freeze**; pin stays `@v0.5.2`; **Trust track deferred**; v0.5.2 = distribution + UX + repair only; **no pack catalog unlock** |
| Checklist when due | (1) Renew or narrow trust-surface freeze → document outcome here + [security model](security-model.md) (2) Decide docs-only `v0.4.4` pin — default **no** until Action consumers need editorial in the tag (3) Pack unlock only if partner proof + freeze review say deepen content (still no new domains) |

Explicit nos: OPA/LSP/tracers, badge marketplace, gtm-oss on site, CE language, second pin, pack catalog growth before 5 partners have week-2 greens.

## Non-goals

- Coreward as CI/build requirement
- LLM auto-attest / auto legal prose
- License change away from Apache-2.0
