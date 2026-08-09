# CyberReady security model

Plain language. This is how trust works in CyberReady+ — and what it does **not** claim.

## What CyberReady is

A **local-first evidence CLI**. It runs pack rules against files in a git repo, writes review artifacts, and can bind a state hash into Git Notes. Humans review those artifacts. CyberReady does **not** certify conformity, issue CE marks, or replace auditors.

## Pilot-prod contract

Pilot-prod (CLI + Action on other git repos) means exactly these three invariants. `scripts/redteam-pilot.sh` is the adversarial grade.

1. **No adversarial false-green** — a consumer PR cannot make Action/`check` exit 0 via `./bin/cyberready`, skipped required-file gates, or a missing hook binary when hooks are enabled.
2. **No path escape** — pack/init/heal/demo cannot write outside the repo (or under `.git/`).
3. **Attest does not lie** — capsule digests are not silently bound to uncommitted self-written evidence without explicit `--allow-dirty`.

### Pilot deploy + freeze

- Grade: `./scripts/redteam-pilot.sh` must be green.
- **CI required check:** merges to `main` require the GitHub Actions job named **`redteam-pilot`** green (enable it under branch protection if not already). Feature count cannot replace this scoreboard.
- Pin: Action/consumers use `@v0.4.2` (prefer tag + commit SHA).
- **30-day trust-surface freeze continues from `v0.4.0` through `v0.4.2`** (`v0.4.2` = honesty + SME buyer-questions only; no trust-surface rewrite): Action binary resolve, `SafeJoin` / pack path jail, attest OCC / `--allow-dirty` honesty, claim-safety, and explain-packet airlock — bugfixes only; no new trust-surface features.

## Trust boundaries

| Boundary | What you can trust | What you must not assume |
|----------|--------------------|--------------------------|
| Local `check` / `validate` | Deterministic gates on the files present | That a green score is a certificate |
| `install.sh` / Action download | Binary matches release `checksums.txt` (sha256, fail-closed) | That "downloaded from GitHub" alone is enough without checksum |
| `attest` capsule | Reproducible `state_hash` from commit + evidence digests | That unsigned or agent-bind placeholders are cryptographic signatures |
| HPURL / proof page | Client-side compare of fragment `h=` to local pointer | Remote server verification or certification |
| Coreward sock bridge | Optional; private socket (mode `0600`); fail-open if absent | Auth beyond filesystem permissions |
| Pack network update | Only when `CYBERREADY_PACKS_URL` **and** `CYBERREADY_PACKS_SHA256` are set | Fetching packs from a URL without a pin |

## Install integrity

Release installs (shell script and composite Action) download the binary **and** `checksums.txt`, then compare sha256. Mismatch or missing entry → refuse install. Prefer building from a known checkout when dogfooding this repo.

The composite Action does **not** prefer a consumer `./bin/cyberready` (that path skipped checksums and enabled PR binary hijack). In this repo it builds from `go.mod`; elsewhere it downloads a release and verifies sha256.

## Attestation honesty

- **Signed:** SSH-agent successfully produced a signature → `user_touch=ssh-agent-signed`.
- **Unsigned:** No agent, or sign failed → capsule is still written, but UI says **UNSIGNED — not cryptographically verified**.
- Fake `agent-bind:…` tokens are **never** treated as verified signatures.

Unsigned ≠ verified. A green readiness score and an unsigned capsule are compatible — and must be labeled honestly.

## Socket (optional Coreward bridge)

Default path is **not** a shared `/tmp/cyberready.sock`. Prefer:

1. `CYBERREADY_SOCK` if set
2. `$XDG_RUNTIME_DIR/cyberready/cyberready.sock`
3. `$TMPDIR/cyberready-$UID/cyberready.sock`
4. `.cyberready/cyberready.sock` under the working directory

Parent directories are created mode `0700`. The socket file is `0600`. World-writable parents are refused.

## Pack updates

Embedded packs ship in the binary. Air-gap: `cyberready packs import`. Network update requires an explicit sha256 pin; without it, network update is refused (offline-safe default).

## Pack regex (ReDoS)

`text_forbid` patterns are length-capped and validated at pack load. Matching uses a size cap and a short timeout. Pathological packs fail closed at load or emit a timeout finding — they do not hang the CLI forever.

## CI / PR surfaces

Prefer **annotations and SARIF** for full gate detail. Sticky PR comments are truncated and red-only by default (`comment_on: never` disables them). Comments are summaries, not a dump of repository secrets or full document bodies.

## What we never claim on public pages

- Certification, CE marking, notified-body approval
- That gate pass equals legal conformity
- That unsigned attest is "verified install" or "signed proof"

See also: [Promotion firewall](promotion-firewall.md) · [SECURITY.md](../SECURITY.md) for vulnerability reporting.
