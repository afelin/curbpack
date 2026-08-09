# Launch readiness

Public Apache-2.0 launch checklist for [afelin/cyberready](https://github.com/afelin/cyberready).
Coreward is **not** required to build, test, launch, or use CyberReady (optional `sock` only).

## Required GitHub checks (merge gate)

Configure these as **required status checks** on `main` (Settings → Branches → branch protection):

| Check | Workflow job | Kill if |
|-------|--------------|---------|
| `test` | `.github/workflows/ci.yml` → `test` | `go test ./...` fails |
| `smoke` | `ci.yml` → `smoke` | doctor/demo or CRA/house happy paths fail |
| `gauntlet` | `ci.yml` → `gauntlet` | claim-safety, heal smoke, baseline ratchet, dead-ends, install-from-release |

Optional (not merge-blocking): `gauntlet-nightly` in `.github/workflows/gauntlet.yml` (realish + adversarial depth + optional OSS clone crash/hang only).

## License hygiene

- `LICENSE` — full Apache-2.0 text (GitHub SPDX)
- `NOTICE` — attribution
- `SECURITY.md` — claim-safe disclosure path
- README license badge → Apache-2.0

After merge, confirm: `gh api repos/afelin/cyberready --jq .license.spdx_id` → `Apache-2.0`.
If still `NOASSERTION`, wait for GitHub re-detect or tag `v0.4.2` with the hygiene commit.

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

- Market promise: a stranger’s **first green &lt;10 minutes** on pin `@v0.4.2` (safe try / product repo / CI).
- Maintainer harness: [`scripts/time-to-green.sh`](../scripts/time-to-green.sh) defaults to a **600s** wall-clock bar; use `TTG_MAX_SECONDS=60` for a tight CI smoke.
- Merge gate: required check **`redteam-pilot`** (`./scripts/redteam-pilot.sh` 9/9). No public vanity counter.

## Tier 3 — human pass (before invite wave)

Before inviting external testers:

1. Fresh machine / no Go: `install.sh` → `doctor` → `demo` — first green in under **10 minutes** (often much faster)
2. Decision-maker understands: evidence for humans — **not** certification
3. `SECURITY.md` reporting path is usable

Record the pass date in the pinned Discussions welcome thread.

## Discussions welcome (claim-safe)

Pinned welcome (created): https://github.com/afelin/cyberready/discussions/4

If not pinned in the UI yet: Discussions → Welcome → Pin. Body must keep:

> Prepares evidence for human review — **not** a conformity assessment, CE mark, or certification.
>
> Try: `curl -fsSL https://raw.githubusercontent.com/afelin/cyberready/main/scripts/install.sh | sh && cyberready doctor && cyberready demo`
>
> Tester reports: use the **Tester report** issue template.

## Invite wave gate

Invite only when:

1. Tier 0–1 CI green on `main` (required checks above)
2. SPDX shows Apache-2.0
3. One Tier-3 human pass recorded
4. Welcome Discussion pinned

## Object-owner cadence (30-day freeze)

| Cadence | Action |
|---------|--------|
| Every merge | Required check **`redteam-pilot`** |
| Weekly | `./scripts/time-to-green.sh`; skim [design-partner](design-partners.md) notes; no new trust-surface features |
| Biweekly | Decide kill/keep on first-move friction |
| Day 30 | Freeze review: renew, narrow, or `v0.4.2` bugfix only |

Explicit nos: OPA/LSP/tracers, badge marketplace, gtm-oss on site, CE language, second pin, pack catalog growth before 5 partners have week-2 greens.

## Non-goals

- Coreward as CI/build requirement
- LLM auto-attest / auto legal prose
- License change away from Apache-2.0
