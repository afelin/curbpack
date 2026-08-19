# GitHub-readiness gap matrix

Stakeholder asks mapped to the **current** Curbpack stack. Evidence paths are real; Reject rows match [strategy boundary](strategy-boundary.md). This is not a second strategy narrative.

**Stack reuse:** `claim-safety.sh`, `redteam-pilot.sh`, `exportx` airlock, packs RKG + `RuleTouchesDiff`, `attest.ComputeStateHash` tests, install/doctor/demo — not parallel systems.

| Stakeholder demand | Status | Evidence |
|--------------------|--------|----------|
| Claim-safe public language (no CE / notified-body theater) | Done | [`scripts/claim-safety.sh`](../scripts/claim-safety.sh) · [promotion firewall](promotion-firewall.md) · [voice and terms](voice-and-terms.md) |
| Understandability (cold-reader four-question bar) | Done | Home alone answers: what is it / what do I run / what do I get / what must I not claim — [voice and terms](voice-and-terms.md) · [launch readiness](launch-readiness.md); claim-safety green; home/builders jargon spot-check clean |
| Institute / agency endorsement refuse (never claim `RISE-approved` / `FRA-approved` / `NCSC-approved` / `agency-endorsed`) | Done | DENY list in `claim-safety.sh` · [promotion firewall](promotion-firewall.md) |
| Adversarial false-green scoreboard | Done | [`scripts/redteam-pilot.sh`](../scripts/redteam-pilot.sh) cases 1, 2, 5, 8, 10–12, 16–21 · [security model](security-model.md) |
| Forged attest note not verified | Done | `internal/attest` · redteam case 19 |
| Tampered cache fake green | Done | `internal/exportx` context-pack re-validate · redteam case 20 |
| Symlink escape via pathjail | Done | `pathjail.Join` · redteam case 21 |
| Action must not prefer workspace `./bin/curbpack` | Done | `action.yml` resolve · redteam case 1 |
| Pack path jail / `.git` refuse | Done | `SafeJoin` · `internal/formhints` ApplyStubs · redteam cases 3–4 |
| Attest OCC / `--allow-dirty` honesty | Done | `internal/attest` · redteam case (attest dirty) |
| Packs network update requires sha256 pin | Done | `internal/packscmd` · redteam case (SHA256) |
| Demo `--out` product-cwd jail | Done | `internal/invariants/demo_jail_test.go` · redteam case |
| Explain-packet airlock (no PEM / home paths) | Done | `exportx.PacketLooksAirlocked` · redteam case 9 |
| SARIF export airlock (shared `sanitizeText`) | Done | `internal/exportx` `FromGateFailures` + export tests |
| SARIF `ruleId` = `gate_id` | Done | `internal/contract` · redteam case 7 |
| CycloneDX / watchlist∩SBOM informational join | Done | `export --watchlist-join` · `internal/exportx` |
| Optional SLSA sidecar / provenance theater | Ops | Launch checklist only — [launch readiness](launch-readiness.md); no L3/4 claim |
| Sock ops (`validate_delta`, …) frozen nave | Done | [stable contracts](stable-contracts.md) · redteam case 12 |
| CapBAC / C2PA / SLSA L3–4 product claims | Reject | Out of public Curbpack · [strategy boundary](strategy-boundary.md) |
| `check --diff` as release gate | Reject (honesty) | Porcelain rule-skip only — see [Δ honesty](#diff-vs-validate_delta); use `validate` / sock `validate_delta` |
| Sock `validate_delta` = full quiet validate | Done | [stable contracts](stable-contracts.md) · [Coreward bridge](coreward-bridge.md) |
| RKG export + digest | Done | `packs export-graph` · `internal/packs/graph.go` |
| Capsule `state_hash` twin-run / no wall-clock | Done | `TestReproducibleStateHash` · `TestCapsuleHashReproducibleNoWallClock` |
| Sub-MB Zig binary | Reject | Go stdlib CGO=0 `-s -w` ~10 MB accepted |
| OPA / Rego evaluator | Reject | [strategy boundary](strategy-boundary.md) · CONTRIBUTING |
| `events.ndjson` BLAKE3 event SoR | Reject | Not in public stack |
| Full local AST / SQLite regulation graph | Reject | RKG JSON export only |
| Legal-metrology / CE / never claim RISE-certified product | Reject | [promotion firewall](promotion-firewall.md) |
| Enforce-before-execute | Reject | Private Coreward only |
| Doctor soft-exit (non-blocking tips) | Done (accepted) | `curbpack doctor` — soft diagnostics; not a hard gate redesign |
| First green &lt;10 min (TTFV) | Polish | `install.sh` → `doctor` → `demo` · [60-second paths](getting-started/60-second-paths.md) · pin `@v0.5.2` |
| Install checksum fail-closed | Done | [`scripts/install.sh`](../scripts/install.sh) · [security model](security-model.md) |
| Action binary cache via consumer `hashFiles` + checksum-skip | Reject | Trust regression — keep download/`checksums.txt` fail-closed · [security model](security-model.md#install-integrity) |

## Δ honesty: `--diff` vs `validate_delta`

| Surface | Behavior | Release-gate safe? |
|---------|----------|-------------------|
| `curbpack check --diff` | Porcelain: skip rules whose paths do not intersect the dirty/changed set (`RuleTouchesDiff`). Basename match semantics stay frozen. | **No** — local speed / PR feedback only |
| `curbpack validate` / sock `validate_delta` | Full quiet validate of composed packs | **Yes** — authoritative pass/fail |

Do **not** retarget sock `validate_delta` to `--diff`. Changing `RuleTouchesDiff` mid-freeze is out of scope (false-green risk). Pack authoring note: [write your own pack](write-your-own-pack.md#diff-vs-full-validate).

## Explicit rejects (do not build in OSS)

Sub-MB Zig binary · OPA evaluator · `events.ndjson` SoR · full local AST/SQLite graph · legal-metrology / CE / never claim RISE-certified product · enforce-before-execute · Action cache that skips sha256 verify (consumer `hashFiles` key).

See also: [strategy boundary](strategy-boundary.md) · [launch readiness](launch-readiness.md) · [security model](security-model.md)
