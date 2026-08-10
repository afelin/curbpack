# Launch readiness

Public Apache-2.0 launch checklist for [afelin/cyberready](https://github.com/afelin/cyberready).
Coreward is **not** required to build, test, launch, or use CyberReady (optional `sock` only).

## Status (2026-08-10)

| Item | State |
|------|--------|
| CI on `main` | **Verified green** at tip `f714700` — `test (ubuntu-latest)`, `test (macos-latest)`, `smoke`, `gauntlet`, `redteam-pilot` (+ dogfood/scoreboard) |
| Required checks on `main` | **Configured via API** — exact names below; `strict` (branches up to date) **on**; `enforce_admins` **on** |
| SPDX | **Apache-2.0** |
| Release pin | **`@v0.4.3`** (no new tag) |
| Discussion #4 body | **Verified** — claim-safe line + install ladder + Tester report pointer |
| Discussion #4 pin | **Pinned** — confirmed via GraphQL `pinnedDiscussions` (2026-08-10); [Welcome to CyberReady+](https://github.com/afelin/cyberready/discussions/4) |
| Tier-3 human pass | **Recorded 2026-08-10** on [Discussion #4](https://github.com/afelin/cyberready/discussions/4#discussioncomment-17960761) — install.sh → doctor → demo PASS (`@v0.4.3`, isolated HOME/PATH) |
| Invite wave | **Ready/open** — items 1–4 done (Welcome Discussion pinned) |

Gap matrix: [github-readiness-gaps.md](github-readiness-gaps.md).

### Branch protection (already set; Settings mirror)

Settings → Branches → `main` rule should require:

- `test (ubuntu-latest)`
- `test (macos-latest)`
- `smoke`
- `gauntlet`
- `redteam-pilot`
- Require branches to be up to date before merging
- Do not bypass for admins on routine pushes (`enforce_admins`)

Verify: `gh api repos/afelin/cyberready/branches/main/protection --jq '.required_status_checks'`

## Required GitHub checks (merge gate)

Configure these as **required status checks** on `main` (Settings → Branches → branch protection). Names must match CI check-run strings exactly:

| Check | Workflow job | Kill if |
|-------|--------------|---------|
| `test (ubuntu-latest)` | `.github/workflows/ci.yml` → `test` (ubuntu) | `go test ./...` fails |
| `test (macos-latest)` | `ci.yml` → `test` (macos) | `go test ./...` fails |
| `smoke` | `ci.yml` → `smoke` | doctor/demo or CRA/house happy paths fail |
| `gauntlet` | `ci.yml` → `gauntlet` | claim-safety, heal smoke, baseline ratchet, dead-ends, install-from-release |
| `redteam-pilot` | `ci.yml` → `redteam-pilot` | `./scripts/redteam-pilot.sh` not 15/15 |

Optional (not merge-blocking): `gauntlet-nightly` in `.github/workflows/gauntlet.yml` (realish + adversarial depth + optional OSS clone crash/hang only).

## License hygiene

- `LICENSE` — full Apache-2.0 text (GitHub SPDX)
- `NOTICE` — attribution
- `SECURITY.md` — claim-safe disclosure path
- README license badge → Apache-2.0

After merge, confirm: `gh api repos/afelin/cyberready --jq .license.spdx_id` → `Apache-2.0`.
If still `NOASSERTION`, wait for GitHub re-detect (pin `@v0.4.3` already shipped with license hygiene).

## Heal (deterministic — not ML)

```bash
cyberready check --heal
```

Loop (max 3): check → form-hints → apply-stub (**missing/empty only**) → remediations cache → re-check.
Cache: `.github/cyberready/cache/remediations.json` keyed by `gate_id`.

**Never** auto-`attest`. Never invents legal prose as approved evidence. Never marks VEX final.

## Claim safety

`scripts/claim-safety.sh` scans docs + runtime CLI captures (doctor/demo/check/prepare-release).
Deny-list blocks certification theater; negation / claim-safe framing is allowed.

## How we know activation works

- Market promise: a stranger’s **first green &lt;10 minutes** on pin `@v0.4.3` (safe try / product repo / CI).
- Maintainer harness: [`scripts/time-to-green.sh`](../scripts/time-to-green.sh) defaults to a **600s** wall-clock bar; use `TTG_MAX_SECONDS=60` for a tight CI smoke.
- Merge gate: required check **`redteam-pilot`** (`./scripts/redteam-pilot.sh` 15/15). No public vanity counter.
- Gap matrix (stakeholder demand → Evidence): [github-readiness-gaps.md](github-readiness-gaps.md).

## Tier 3 — human pass (before invite wave)

Before inviting external testers:

1. Fresh machine / no Go: `install.sh` → `doctor` → `demo` — first green in under **10 minutes** (often much faster)
2. Decision-maker understands: evidence for humans — **not** certification
3. `SECURITY.md` reporting path is usable

**Pass recorded:** 2026-08-10 on pinned Discussions welcome thread ([comment](https://github.com/afelin/cyberready/discussions/4#discussioncomment-17960761)).

## Discussions welcome (claim-safe)

Welcome thread: https://github.com/afelin/cyberready/discussions/4

**Pin:** **Pinned** (GraphQL `pinnedDiscussions`, 2026-08-10). Body must keep:

> Prepares evidence for human review — **not** a conformity assessment, CE mark, or certification.
>
> Try: `curl -fsSL https://raw.githubusercontent.com/afelin/cyberready/main/scripts/install.sh | sh && cyberready doctor && cyberready demo`
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
| Day 30 | Freeze review: renew, narrow, or next `v0.4.x` bugfix only (`v0.4.3` = current instrument-panel pin) |

### Freeze review (day-30 from v0.4.0)

| Field | Value |
|-------|--------|
| Freeze start | `v0.4.0` (2026-08-08) |
| Day-30 due | **2026-09-07** |
| Status | **Not yet due** — do not treat this as a completed review |
| Interim stance (until due) | **Renew freeze**; pin stays `@v0.4.3`; **no `v0.4.4`** cut; **no pack catalog unlock** |
| Checklist when due | (1) Renew or narrow trust-surface freeze → document outcome here + [security model](security-model.md) (2) Decide docs-only `v0.4.4` pin — default **no** until Action consumers need editorial in the tag (3) Pack unlock only if partner proof + freeze review say deepen content (still no new domains) |

Explicit nos: OPA/LSP/tracers, badge marketplace, gtm-oss on site, CE language, second pin, pack catalog growth before 5 partners have week-2 greens.

## Non-goals

- Coreward as CI/build requirement
- LLM auto-attest / auto legal prose
- License change away from Apache-2.0
