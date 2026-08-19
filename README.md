# Curbpack

[![ci](https://github.com/afelin/curbpack/actions/workflows/ci.yml/badge.svg)](https://github.com/afelin/curbpack/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

> Article 14 reporting starts **11 September 2026**. One command, writes nothing:
>
> ```bash
> npx curbpack@0.5.2 scan   # or curbpack scan after install
> ```
>
> Not conformity assessment. We never see your repo.

Curbpack checks your repository against local rule packs and writes a review pack you can hand to a buyer or auditor—on your machine, without claiming certification.

> Not conformity assessment. Not CE marking. Not a notified-body opinion.

[Site](https://ri-se.github.io/curbpack/) (Pages at ri-se.github.io) · canonical repo [afelin/curbpack](https://github.com/afelin/curbpack) · mirror [RI-SE/curbpack](https://github.com/RI-SE/curbpack) · [White paper](papers/curbpack-whitepaper.md) · [Voice and terms](docs/voice-and-terms.md) · [For builders](site/for-builders/) · [Art 14 scan](site/art14/) · [Docs index](docs/README.md)

## Quickstart (one command)

Inside any git repo — read-only, no install, no files written. Pin **`@v0.5.2`**.

```bash
npx curbpack@0.5.2 scan
```

Scan defaults to **`cra-baseline`** and prints the Art 14 reporting clock. When you want hooks and a daily score, use the full ladder below. Stuck? [troubleshooting](docs/getting-started/troubleshooting.md).

## Full ladder (below)

Install when you want a local binary (Windows PowerShell · macOS/Linux curl): [install](docs/getting-started/install.md). After install, `curb` is a short alias for `curbpack`.

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/afelin/curbpack/main/scripts/install.ps1 | iex
```

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/afelin/curbpack/main/scripts/install.sh | sh
```

Then the same ladder on every OS:

```bash
curbpack doctor
curbpack demo              # sandbox; optional --open
cd /path/to/your/product   # git repo
curbpack scan              # read-only diagnosis — no init, no hooks, no score
curbpack fix --art14       # one Art 14 rehearsal file (diff preview; human confirm)
curbpack share             # optional --bundle; --reveal opens review-pack in Explorer/Finder
curbpack init              # when ready: house-policy default; --profile cra|medtech
curbpack check --score     # daily loop — exit code is authoritative
# human only when ready:
curbpack attest
# verify: proof/index.html vs hpurl-pointer.json
```

`curbpack ask-my-suppliers` emits the same buyer checklist as `export --buyer-questions`. On red: `curbpack check --heal` then `curbpack ask .github/curbpack/cache/latest_failure.json --propose`, then re-check. Optional drift checklist: `curbpack drift` (exit 0 always). After OS update / PATH loss: `curbpack doctor --repair` (local only — not auto-update; Windows also: `install.ps1 -Repair`).

## Three ways in

Every path ends in the same local `check`. Write adds optional pathway drafts first; Bring and CI skip outlines. Full table and ladders: [60-second paths](docs/getting-started/60-second-paths.md) · [pathway guide](docs/getting-started/pathway.md).

## What you get

| Artifact | When | What it is |
|----------|------|------------|
| **Gate report** | Every `check` | JSON + markdown findings—structural evidence, not a legal finding |
| **Review pack** | `prepare-release` or `share` | Layered reports for human review |
| **Buyer one-pager** | After green + `share` | Supplier evidence summary HTML you hand to a buyer |
| **Evidence bundle** | `share --bundle` | Offline `review-pack/evidence-bundle.html` with embedded hpurl pointer |
| **Reveal / Attach** | `share --reveal` | Opens review-pack (or bundle) in Explorer/Finder; stdout `Attach: <abs path>` on every OS |
| **Drift checklist** | `curbpack drift` | Multi-signal human checklist (exit 0; not a compliance meter) |
| **Attest capsule** | Human `attest` when ready | Git Notes hash bind—**unsigned ≠ verified** |
| **Proof page** | After attest | Local `proof/index.html` vs evidence pointer—still human judgment |

Optional exports: SARIF, ContextPack, buyer-questions, lay-of-land. Teaching sample: [site/samples/onepager.html](site/samples/onepager.html).

## How to interpret results

| Signal | Meaning |
|--------|---------|
| Exit **0** | Gates passed on this tree—for human review, not certification |
| Exit **1** | Findings remain or operational error |
| Exit **2** | Usage / environment (unknown command, not a git repo when required) |
| **Unsigned** attest | Capsule present; **not cryptographically verified** |
| **ssh-agent-signed** | Real SSH signature produced |

Gate pass is **not** certification, CE marking, or notified-body approval. Humans decide what to claim.

## Where to go deeper

| Reader | Start |
|--------|--------|
| Builder | [For builders](site/for-builders/) · [pathway](docs/getting-started/pathway.md) · [daily loop](docs/getting-started/daily-loop.md) |
| Buyer / reviewer | [Buyer evidence](docs/getting-started/buyer-evidence.md) · [for-reviewers](site/for-reviewers/) |
| CISO / authority | [For authorities](docs/for-authorities.md) · [site Authorities](site/for-authorities/) |
| Full system | [White paper](papers/curbpack-whitepaper.md) · [how it works](site/how-it-works/) |
| Abbreviations | [Glossary and audience](docs/glossary-and-audience.md) |

## GitHub Action

Action runners are **Linux/macOS only** (local Windows CLI is supported separately).

```yaml
- uses: afelin/curbpack@v0.5.2
  with:
    heal: "true" # opt-in; Action default is false (scaffold ≠ readiness)
    comment_on: red
    upload_sarif: "true"
```

Pin **`@v0.5.2`**. Drop-in example: [`examples/workflows/curbpack-check.yml`](examples/workflows/curbpack-check.yml). Pilot deploy: `./scripts/redteam-pilot.sh`.

## Dual-remote vibe-safe flow

Use this exact flow for low-cognitive-load day-to-day work:

1. Open repo in Cursor.
2. Build feature in a feature branch.
3. Say: **"Open a PR."**
4. Wait for CI checks to pass.
5. Merge PR in GitHub (or let Cursor merge if token allows).
6. Say: **"Sync both curbpack remotes."**
7. If sync pauses, do exactly what the pause message says (auth, conflict resolution, or branch protection issue).

Two hardening checks:

- Before step 6, confirm `main` is clean and up to date.
- After step 6, verify both remotes are at the same commit SHA.

For this repository, "Sync both curbpack remotes" means running `./scripts/curb-sync.sh` (merge-only; never force-push).

Ops runbook: [`docs/getting-started/repo-ops-hardening.md`](docs/getting-started/repo-ops-hardening.md).

## Advanced

Binary size (~10 MB, Go CGO=0 `-s -w`), doctor soft-exit tips, and Zig non-goals live here—not on the first screen.

**Compose, do not conquer:** Curbpack prepares structural evidence for product repos. Pair with SCA (e.g. Trivy/OSV) and secret scanners (e.g. Gitleaks) for depth — not a security program. Boundary: [strategy boundary](docs/strategy-boundary.md).

Confirms are human-only (`--i-am-human` or `CURBPACK_ALLOW_CONFIRM=1`; TTY alone is not enough). Research briefs never gate pass/fail. Assistants: [docs/assistant-loop.md](docs/assistant-loop.md) · thin MCP [examples/mcp/](examples/mcp/).

| Command | Purpose |
|---------|---------|
| *(bare)* | `doctor` if uninitialized, else `check` |
| `demo [--keep] [--open]` | Sandbox check; `--open` opt-in browser |
| `validate [--json]` | Pack gates (dual-rep); prefer `check` daily |
| `check --diff` | Delta mode — **not** release-gate safe |
| `ask [file] --propose` | Explain GateFailure JSON (propose-only) |
| `packs list\|update\|import\|export-graph\|doctor` | Packs, local pack→rule map export, validity doctor |
| `export --sarif\|--explain-packet\|--watchlist-join\|--buyer-questions\|--lay-of-land\|--context-pack` | Standards / tutor packet / buyer checklist / map / ContextPack |
| `share` | Thin recipe: check → context-pack → buyer-questions → prepare-release |
| `pathway status` | One next ask (human default; `--technical` for phase path) |
| `pathway suggest\|note` | Warm-start seed + session notes — not a gate input |
| `pathway confirm-*` | Human only — `--i-am-human` or `CURBPACK_ALLOW_CONFIRM=1` |
| `research [--fetch]\|--cite-check` | Allowlisted citation packet + human brief — never gates check |
| `completion bash\|zsh\|fish` | Print shell completions |
| `sock` | Optional Unix IPC for integrators (continues if unused) |
| `init --bare` | Minimal scaffold (no hooks/skill/ide) |
| `init --packs a,b` | Override default house-policy packs |
| `init --workflow` | Opt-in: write `.github/workflows/curbpack.yml` if missing |

Deep docs: [Intent vs Scope](docs/intent-vs-scope.md) · [Security model](docs/security-model.md) · [Write your own pack](docs/write-your-own-pack.md) · [Migration](docs/migration-cyberready-to-curbpack.md) · [Adopters](ADOPTERS.md)

Claim-safe wording enforced by `scripts/claim-safety.sh`. Preferred public language: [docs/voice-and-terms.md](docs/voice-and-terms.md).
