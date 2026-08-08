# 60-second paths

CyberReady prepares evidence for **human review**. It does **not** certify conformity.

Cold-start default pack: **`house-policy`**. CRA / medtech are opt-in later via `--packs` (Advanced).

Pick **exactly one** first move for your audience.

## Human — safe try

```bash
curl -fsSL https://raw.githubusercontent.com/afelin/cyberready/main/scripts/install.sh | sh
cyberready doctor
cyberready demo                          # sandbox green + one-pager path (no browser)
# cyberready demo --open                 # opt-in: open the one-pager in the OS browser
```

## Human — product repo

```bash
cd my-product
cyberready init                          # house-policy + hooks + skill + ide
cyberready check                         # or bare: cyberready
```

## Agent

```bash
cyberready init                          # skill lands at .cursor/skills/cyberready/SKILL.md
cyberready check
cyberready check --form-hints            # propose-only snippets
# optional: cyberready check --form-hints --apply-stub   # write missing stubs only
```

Agent rule: after doc/dep edits, re-run `check`. Never claim certification.

## CI-only

**Action-only path** (no local install required):

1. Copy [`examples/workflows/cyberready-check.yml`](../../examples/workflows/cyberready-check.yml) → `.github/workflows/cyberready.yml`.
2. Push / open a PR. Pin stays **`@v0.4.0`**. Minimal permissions: `contents: read`, `pull-requests: write`, `security-events: write`.
3. Expect: uninitialized repos resolve **`house-policy`**; with `heal: true`, missing stubs are written; green sticky once, or red with heal stubs + top-3 ask pointer — still felt value. Claim-safe: gate pass ≠ certification.

Optional local equivalent: `cyberready init --workflow` writes the same drop-in workflow **only if missing** (never overwrites; not enabled by default `init`).

Local Action-equivalent smoke: temp git repo **without** `.cyberready.json` → `cyberready check --heal` → exit 0 (or deterministic red after stubs if content gates remain).

## Decision-maker

1. Open the supplier’s `review-pack/buyer-onepager.html` (from `prepare-release` or the Action artifact), or the committed sample at `site/samples/onepager.html`.
2. Or open the HPURL proof page (`proof/index.html`) with a hash fragment.
3. One screen: thermometer, top gaps, disclaimer — no account required. Not a certificate.

## Advanced

| Flag / path | When |
|-------------|------|
| `cyberready init --bare` | Minimal scaffold (no hooks/skill/ide) |
| `cyberready init --packs a,b` | Override default house-policy |
| `cyberready init --workflow` | Opt-in drop-in Action workflow if missing |
| `cyberready demo --open` | Opt-in browser for the sandbox one-pager |

> Prepares evidence for human review — not a conformity assessment.
