# Curbpack security model

Plain language. This is how trust works in Curbpack — and what it does **not** claim.

## What Curbpack is

A **local-first evidence CLI**. It runs pack rules against files in a git repo, writes review artifacts, and can bind a state hash into Git Notes. Humans review those artifacts. Curbpack does **not** certify conformity, issue CE marks, or replace auditors.

## Pilot-prod contract

Pilot-prod (CLI + Action on other git repos) means exactly these three invariants. `scripts/redteam-pilot.sh` is the adversarial grade.

1. **No adversarial false-green** — a consumer PR cannot make Action/`check` exit 0 via `./bin/curbpack`, skipped required-file gates, or a missing hook binary when hooks are enabled.
2. **No path escape** — pack/init/heal/demo cannot write outside the repo (or under `.git/`).
3. **Attest does not lie** — capsule digests are not silently bound to uncommitted self-written evidence without explicit `--allow-dirty`.

### Pilot deploy + freeze

- Grade: `./scripts/redteam-pilot.sh` must be green (**13/13** cases).
- **CI required check:** merges to `main` require the GitHub Actions job named **`redteam-pilot`** green under branch protection. Feature count cannot replace this scoreboard.
- Scoreboard locks (wrap existing unit tests — no logic fork): Action resolve, `--diff` false-green, ApplyStubs `.git` jail, pack path escape, claim-safety (incl. promotion-firewall endorsement DENY), overlay compose, SARIF `ruleId`, policy-graph schema, explain airlock, pack catalog freeze, import `assurance_class`, stable sock ops, attest dirty/`--allow-dirty`, forged attest note, tampered cache, symlink escape, packs SHA256 pin, demo `--out` jail.
- Pin: Action/consumers use `@v0.5.2` (prefer tag + commit SHA).
- **30-day trust-surface freeze continues from `v0.4.0` through `v0.5.0`** (`v0.5.0` = instrument-panel honesty only; no trust-surface rewrite): Action binary resolve, `SafeJoin` / pack path jail, attest OCC / `--allow-dirty` honesty, claim-safety, and explain-packet airlock — bugfixes only; no new trust-surface features.
- Stakeholder gap matrix: [github-readiness-gaps.md](github-readiness-gaps.md).
- **Day-30 freeze review due 2026-09-07** (from `v0.4.0` on 2026-08-08). Until then: **renew freeze**, no `v0.4.4`, no pack unlock. Checklist + outcome log: [launch readiness](launch-readiness.md#freeze-review-day-30-from-v040). Stable nave: [stable contracts](stable-contracts.md).

## Trust boundaries

| Boundary | What you can trust | What you must not assume |
|----------|--------------------|--------------------------|
| Local `check` / `validate` | Deterministic gates on the files present | That a green score is a certificate |
| `install.sh` / Action download | Binary matches release `checksums.txt` (sha256, fail-closed) | That "downloaded from GitHub" alone is enough without checksum |
| `npm` wrapper (`npx curbpack`) | Same release asset + `checksums.txt` verify; lazy fetch to cache dir | That npm install alone attests to binary integrity without sha256 check |
| `attest` capsule | Reproducible `state_hash` from commit + evidence digests | That unsigned or agent-bind placeholders are cryptographic signatures |
| HPURL / proof page | Client-side compare of fragment `h=` to local pointer | Remote server verification or certification |
| Optional sock IPC | Optional; private socket (mode `0600`); fail-open if absent | Auth beyond filesystem permissions |
| Pack network update | Only when `CURBPACK_PACKS_URL` **and** `CURBPACK_PACKS_SHA256` are set | Fetching packs from a URL without a pin |

## Install integrity

Release installs (shell script, **npm wrapper**, and composite Action) download the binary **and** `checksums.txt`, then compare sha256. Mismatch or missing entry → refuse install. Prefer building from a known checkout when dogfooding this repo.

**npm wrapper:** `npx curbpack` / `npm install -g curbpack` use a thin Node entrypoint. Default pin is **`v0.5.2`** from bundled `install-manifest.json` (not floating `latest` unless `CURBPACK_VERSION=latest`). No network `postinstall` — download runs on first exec. Cached path: `~/.curbpack/bin` (Unix) or `%LOCALAPPDATA%\curbpack` (Windows).

The composite Action does **not** prefer a consumer `./bin/curbpack` (that path skipped checksums and enabled PR binary hijack). In this repo it builds from `go.mod`; elsewhere it downloads a release and verifies sha256.

**Rejected: Action cache-as-written.** A proposed consumer `hashFiles('**/*.go')` cache key plus skipping checksum verify on cache hit is a trust regression. Keep fail-closed download + `checksums.txt` verify (or dogfood `go build`). Any future cache must key on version + expected sha256 and **re-verify** before exec — not land in this track.

## Attestation honesty

- **Signed:** SSH-agent successfully produced a signature → `user_touch=ssh-agent-signed`.
- **Unsigned:** No agent, or sign failed → capsule is still written, but UI says **UNSIGNED — not cryptographically verified**.
- Fake `agent-bind:…` tokens are **never** treated as verified signatures.

Unsigned ≠ verified. A green readiness score and an unsigned capsule are compatible — and must be labeled honestly.

## Socket (optional integrator IPC)

Default path is **not** a shared `/tmp/curbpack.sock`. Prefer:

1. `CURBPACK_SOCK` if set
2. `$XDG_RUNTIME_DIR/curbpack/curbpack.sock`
3. `$TMPDIR/curbpack-$UID/curbpack.sock`
4. `.curbpack/curbpack.sock` under the working directory

Parent directories are created mode `0700`. The socket file is `0600`. World-writable parents are refused.

## Pack updates

Embedded packs ship in the binary. Air-gap: `curbpack packs import`. Network update requires an explicit sha256 pin; without it, network update is refused (offline-safe default).

## Pack regex (ReDoS)

`text_forbid` patterns are length-capped (`MaxRegexPatternLen`) and compiled at pack load. Matching uses a size cap (`MaxRegexMatchBytes`) and a short timeout. Invalid patterns fail closed at load or emit a timeout finding — they do not hang the CLI forever. Nested quantifiers that RE2 accepts under the length cap are allowed (no separate nesting heuristic).

## CI / PR surfaces

Prefer **annotations and SARIF** for full gate detail. Sticky PR comments are truncated and red-only by default (`comment_on: never` disables them). Comments are summaries, not a dump of repository secrets or full document bodies.

## What we never claim on public pages

- Certification, CE marking, notified-body approval
- That gate pass equals legal conformity
- That unsigned attest is "verified install" or "signed proof"

See also: [For authorities](for-authorities.md) · [Promotion firewall](promotion-firewall.md) · [SECURITY.md](../SECURITY.md) for vulnerability reporting · [Coreward pointer](coreward-pointer.md)
