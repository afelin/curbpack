# Design partners

Product brief for five external repos that keep Action `@v0.4.1` or `init`+hooks green. Outreach is human-operated; this file is the ask + scoreboard.

**Local pack gates. Humans review. Not conformity assessment.**

| Field | Content |
|-------|---------|
| Ask | Add Action `@v0.4.1` **or** `cyberready init` + hooks; keep for 14 days |
| Success | First green &lt;10 min; second green ≤7 days; “judge clicked without pitch?” Y/N |
| Forbidden asks | Certification claims; uploading IP to a cloud policy brain |
| Weekly ritual | 15-min note: path taken (A/B/C), stall step, keep/kill |

## Target mix

| Slot | Profile |
|------|---------|
| 2 | OSS maintainers |
| 2 | SME / supplier-ish product repos |
| 1 | Internal Coreward / vibe-engine-os |

## Partner issue shape

Prefer [First move stuck](../.github/ISSUE_TEMPLATE/first_move_stuck.yml) when activation fails. Prefer Discussions “I went green” when it works.

Do **not** send partners to `docs/gtm-oss/` (non-product).

## What we measure

- First-move completion (≥4/5)
- Second green within 7 days (≥3/5)
- Pin / “is this certified?” / main≠tag support → ~0

Stars are not the scoreboard.

## Object-owner cadence checklist

Ship the rhythm in-repo; the calendar is human-operated.

| Cadence | Action |
|---------|--------|
| Every merge | `redteam-pilot` required green |
| Weekly | Run `./scripts/time-to-green.sh`; skim partner notes; **zero** new trust-surface features |
| Biweekly | Kill/keep on friction from first-move issues |
| Day 30 of freeze | Explicit freeze review: renew, narrow, or cut `v0.4.2` bugfix only |

### Explicit nos

OPA/LSP/tracers · badge marketplace · `gtm-oss` on site · CE language · second pin · expanding pack catalog before 5 partners have week-2 greens.

**Pack catalog freeze:** only `house-policy`, `cra-baseline`, `medtech-iec62304`. `scripts/redteam-pilot.sh` fails on any new pack id under `packs/` or the embed twin. Unlock requires freeze review + an explicit PR that updates the allowlist — not an env escape hatch in CI.

Also mirrored in [launch readiness](launch-readiness.md), [Intent vs Scope](intent-vs-scope.md), and [Promotion firewall](promotion-firewall.md).
