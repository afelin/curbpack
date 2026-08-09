# Daily loop (habit)

Value recurs without babysitting. Pin stays **`@v0.4.2`**. Gate green is evidence for human review — not certification.

```text
every PR  → Action @v0.4.2 (comment_on: red, upload_sarif)
local day → cyberready check   # Δ whisper when prior cache
release   → prepare-release → attest
```

## Every PR

Copy [`examples/workflows/cyberready-check.yml`](../../examples/workflows/cyberready-check.yml) once, or `cyberready init --workflow` (writes only if missing). Keep `comment_on: red` — no PR noise on green.

## Local day

```bash
cyberready check
```

When a prior evidence cache exists, a quiet one-line `Δ readiness` (or “gates green · evidence cache updated”) is proof you were here before. No dashboard.

Opinionated `cyberready init` installs git hooks by default — that is the stickiness layer.

## Release

```bash
cyberready prepare-release
cyberready attest   # human only; never auto
```

## Self-dogfood

This repo runs [`.github/workflows/cyberready-dogfood.yml`](../../.github/workflows/cyberready-dogfood.yml) on every PR. Treat flakes as P0.

## Buyer share (optional)

After a green (or red) check, export a human checklist:

```bash
cyberready export --buyer-questions
```

Details: [Buyer evidence](buyer-evidence.md).

See also: [60-second paths](60-second-paths.md) · [Design partners](../design-partners.md)
