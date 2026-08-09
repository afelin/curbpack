# Daily loop (habit)

Value recurs without babysitting. Pin stays **`@v0.4.2`** until the next honesty cut. Gate green is evidence for human review — not certification.

**Instrument panel pitch:** after every change (human or agent), one `cyberready check` yields an honest map for *this* repo — structural evidence, not a certificate. Keep hooks: they are the agent force-multiplier.

```text
every PR  → Action @v0.4.2 (comment_on: red, upload_sarif) + hooks
local day → cyberready check   # Δ readiness / deps / secret-hits + covenant
release   → prepare-release → attest
optional  → export --lay-of-land · export --buyer-questions
```

## Every PR

Copy [`examples/workflows/cyberready-check.yml`](../../examples/workflows/cyberready-check.yml) once, or `cyberready init --workflow` (writes only if missing). Keep `comment_on: red` — no PR noise on green.

Opinionated `cyberready init` installs git hooks by default — **keep them** for agent PRs so every edit re-enters the check loop.

## Local day

```bash
cyberready check
```

When a prior evidence cache exists, quiet dim lines show `Δ readiness`, and (when prior `instrument.json` exists) `Δ deps` / `Δ secret-hits`. Every check also prints the instrument-panel covenant. No dashboard.

## Release

```bash
cyberready prepare-release
cyberready attest   # human only; never auto
```

## Self-dogfood

This repo runs [`.github/workflows/cyberready-dogfood.yml`](../../.github/workflows/cyberready-dogfood.yml) on every PR. Treat flakes as P0.

## Buyer / map share (optional)

```bash
cyberready export --lay-of-land      # shareable instrument map
cyberready export --buyer-questions  # human checklist
```

Details: [Buyer evidence](buyer-evidence.md).

See also: [60-second paths](60-second-paths.md) · [Design partners](../design-partners.md)
