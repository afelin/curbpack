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

Drop [`examples/workflows/cyberready-check.yml`](../../examples/workflows/cyberready-check.yml) into `.github/workflows/cyberready.yml` (pin `@v0.4.0`). First push/PR runs heal stubs on red or greens once — still felt value.

## Decision-maker

1. Open the supplier’s `review-pack/buyer-onepager.html` (from `prepare-release` or the Action artifact), or the committed sample at `site/samples/onepager.html`.
2. Or open the HPURL proof page (`proof/index.html`) with a hash fragment.
3. One screen: thermometer, top gaps, disclaimer — no account required. Not a certificate.

## Advanced

| Flag / path | When |
|-------------|------|
| `cyberready init --bare` | Minimal scaffold (no hooks/skill/ide) |
| `cyberready init --packs a,b` | Override default house-policy |
| `cyberready demo --open` | Opt-in browser for the sandbox one-pager |

> Prepares evidence for human review — not a conformity assessment.
