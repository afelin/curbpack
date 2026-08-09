# Contributing

Thanks for helping. CyberReady+ prepares **evidence for human review** — not a conformity assessment.

## Dev loop

```bash
go test ./...
./scripts/claim-safety.sh
./scripts/redteam-pilot.sh   # required merge check name: redteam-pilot
```

Optional activation smoke (maintainer bar defaults to 600s; use `TTG_MAX_SECONDS=60` for a tight local run):

```bash
./scripts/time-to-green.sh
TTG_MAX_SECONDS=60 ./scripts/time-to-green.sh
```

## First-move failed?

Prefer the **First move stuck** issue template: path A/B/C + the step that failed. That shape beats a vague “it didn’t work.”

## Claim safety

Do not introduce certification / CE / notified-body language. Run `scripts/claim-safety.sh` before opening a PR that touches docs or CLI strings. Public co-promotion rules: [`docs/promotion-firewall.md`](docs/promotion-firewall.md) (RISE = funder, not certifier).

## Trust surface

Action binary resolve, `SafeJoin` / pack path jail, attest honesty, claim-safety, and explain-packet airlock are under freeze through the v0.4.x line — bugfixes only. See `docs/security-model.md`.

## Non-product docs

`docs/gtm-oss/` is quarantined. Do not link it from the product site, README hero, or Action copy.
