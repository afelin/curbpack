# CyberReady site (GitHub Pages)

Static explainer for https://afelin.github.io/cyberready/

Deploy via `.github/workflows/pages.yml` (Actions → Pages). Enable **GitHub Pages → Source: GitHub Actions** in repo settings if the site 404s.

Pilot pin: `@v0.4.2`. Adversarial grade: `scripts/redteam-pilot.sh`. Trust-surface freeze: see `docs/security-model.md`.

## Contents

Public IA only: home, how-it-works, for-builders, for-reviewers, security, whitepaper, samples.

## Ops quarantine

Do **not** deploy or link from Pages:

- `docs/gtm-oss/` (social/launch copy)
- Launch invite checklists / unfixed exploit writeups
- Internal CI runbooks

Claim-safety applies to site HTML copy the same as docs.

## Local preview

Open `site/index.html` in a browser, or:

```bash
python3 -m http.server -d site 8080
```
