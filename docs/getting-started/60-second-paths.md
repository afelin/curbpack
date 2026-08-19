# 60-second paths

Curbpack checks your repository against local rule packs and writes a review pack you can hand to a buyer or auditor—on your machine, without claiming certification.

Not conformity assessment. Not CE marking. Not a notified-body opinion.

**Scan-first (cold repo, no init):** `npx curbpack@0.5.2 scan` — read-only; defaults to **`cra-baseline`** + Art 14 clock.  
**Init / check cold default:** **`house-policy`** (via `curbpack init` or uninitialized `check`). CRA / medtech are opt-in via `--packs`. Pick **exactly one** first move for your audience.

## Scan-first (fastest)

No install, no init — inside any git repo:

```bash
npx curbpack@0.5.2 scan
```

Read-only diagnosis. Packs line shows `cra-baseline` on a cold tree. Use `curbpack fix --art14` → `init` → `check --score` when you are ready to write files.

## Three ways in

Same local `check` for all three — Write adds optional draft choice first; Bring and CI go straight to check.

| Way | What you do |
|-----|-------------|
| **Write→Check** | Optional [pathway](pathway.md) interview that suggests checklists → confirm packs (`--i-am-human` or `CURBPACK_ALLOW_CONFIRM=1`) → optional research brief → two drafts + Recommended A\|B → you pick → cite-check (refuses ungrounded Claims; stub-only confirm refuses) → `check`. |
| **Bring-docs→Check** | Place existing policies on pack paths (or point a custom pack JSON at your paths), then `check`. No portal PDF ingest. |
| **CI** | Action-only (or local `check` alone). Pin **`@v0.5.2`**. Action = Linux/macOS runners. |

Builders site: [Three ways in](../../site/for-builders/). Install hub: [install](install.md) · [troubleshooting](troubleshooting.md). Write depth: [pathway](pathway.md).

## Human — safe try

Under ten minutes (pin **`@v0.5.2`**): install → `doctor` → `demo`. Gate green ≠ certification. Full ladders: [install](install.md).

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/afelin/curbpack/main/scripts/install.ps1 | iex
```

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/afelin/curbpack/main/scripts/install.sh | sh
```

Then:

```bash
curbpack doctor
curbpack demo                          # sandbox green + one-pager path (no browser)
# curbpack demo --open                 # opt-in: open the one-pager in the OS browser
```

## Human — product repo

```bash
cd my-product
curbpack init                          # house-policy default; --profile cra|medtech
curbpack check                         # or bare: curbpack / curb
curbpack share                         # optional --bundle
# human when ready: curbpack attest → proof/index.html
```

On red: `curbpack check --heal` then `curbpack ask … --propose`, then re-check. After first green: `curbpack share` for the handoff recipe (review pack + buyer one-pager). Optional: `curbpack drift` (informational checklist, exit 0).

## Agent

```bash
curbpack init                          # skill lands at .cursor/skills/curbpack/SKILL.md
curbpack check
curbpack check --form-hints            # propose-only snippets
# optional: curbpack check --form-hints --apply-stub   # write missing stubs only
```

Agent rule: after doc/dep edits, re-run `check`. On red: heal + ask propose. On green: prefer `share` or `export --context-pack`. Never claim certification. Full contract: [assistant-loop](../assistant-loop.md) (Cursor / Copilot / Claude / others).

## CI-only

**Action-only path** (no local install required):

1. Copy [`examples/workflows/curbpack-check.yml`](../../examples/workflows/curbpack-check.yml) → `.github/workflows/curbpack.yml`.
2. Push / open a PR. Pin stays **`@v0.5.2`**. Action = Linux/macOS only. Minimal permissions: `contents: read`, `pull-requests: write`, `security-events: write`.
3. Expect: uninitialized repos resolve **`house-policy`**; Action `heal` defaults to **false**; set `heal: true` to write missing stubs (scaffold ≠ readiness). Green sticky once, or red with heal stubs + top-3 ask pointer — still felt value. Claim-safe: gate pass ≠ certification.

Optional local equivalent: `curbpack init --workflow` writes the same drop-in workflow **only if missing** (never overwrites; not enabled by default `init`).

Local Action-equivalent smoke: temp git repo **without** `.curbpack.json` → `curbpack check --heal` → exit 0 (or deterministic red after stubs if content gates remain).

Maintainer bar: `./scripts/time-to-green.sh` (demo + init→check wall-clock; fail if &gt;10 min).

## Decision-maker

1. Open the supplier’s `review-pack/buyer-onepager.html` (from `prepare-release` or the Action artifact), or the committed sample at `site/samples/onepager.html`.
2. Or open the proof page (`proof/index.html`) with a hash fragment.
3. One screen: local gate score, top gaps, disclaimer — no account required. Not a certificate.

Runs deposit cache + review-pack under `.github/curbpack/`; attest when a human is ready.

## Advanced

Habit after first green: [daily loop](daily-loop.md). Optional share: [buyer evidence](buyer-evidence.md) (`export --lay-of-land` / `--buyer-questions`). Doctor soft-exit tips and binary size notes: README Advanced.

| Flag / path | When |
|-------------|------|
| `npx curbpack@0.5.2 scan` | Read-only first contact; scan defaults to **cra-baseline** |
| `curbpack init --bare` | Minimal scaffold (no hooks/skill/ide) |
| `curbpack init --packs a,b` | Override init default **house-policy** |
| `curbpack init --workflow` | Opt-in drop-in Action `@v0.5.2` workflow if missing |
| `curbpack demo --open` | Opt-in browser for the sandbox one-pager |
