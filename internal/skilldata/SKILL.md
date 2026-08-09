---
description: Run CyberReady+ local compliance gates and explain GateFailure JSON. Use when the user asks about pack gates, prepare-release review packs, or cyberready check/validate/ask/attest/doctor/demo/export. Propose-only — never claim certification.
---

# CyberReady+

Local-first evidence CLI. Prepares review packs for **human review**. Does not certify conformity.

**Instrument panel:** after edits, one `cyberready check` yields an honest map for *this* repo — structural evidence, not a certificate. Green is expensive to fake.

## When to use

- User wants pack-based documentation readiness checked offline (CRA, house-policy, medtech, custom)
- After editing annex / SECURITY.md / pack-referenced files, re-run `check`
- Explaining a `GateFailure` JSON payload from `.github/cyberready/cache/`
- Safe try without touching product: `cyberready demo`
- Exporting SARIF / RKG / explain-packet / lay-of-land for IDEs, tutors, or humans

## Commands

```bash
cyberready                 # doctor if uninitialized, else check
cyberready doctor
cyberready demo
cyberready init            # house-policy + hooks + skill + ide (use --bare for minimal)
cyberready init --packs cra-baseline,house-policy
cyberready check
cyberready check --heal
cyberready validate --json
cyberready prepare-release
cyberready export --sarif
cyberready export --explain-packet
cyberready export --watchlist-join
cyberready export --lay-of-land
cyberready export --buyer-questions
cyberready packs list
cyberready packs export-graph
cyberready packs doctor
cyberready ask .github/cyberready/cache/latest_failure.json --propose
cyberready attest
```

## Exit codes (stable)

| Code | Meaning |
|------|---------|
| 0 | Pass / success |
| 1 | Gate failures (or operational error on check/validate) |
| 2 | Usage / environment (unknown command, not a git repo when required) |

JSON payloads include `schema_version` for agents. SARIF `ruleId` equals `gate_id`.

## Agent rules

1. Treat `cyberready check` / `validate` exit code as authoritative for gate pass/fail.
2. Prefer dual-rep markdown + JSON IR — do not invent legal prose as source of truth.
3. `ask --propose` and `check --form-hints` suggest edits only; apply in the editor (or `--apply-stub` / `--heal` for missing stubs), then re-check.
4. Never claim the product is certified, CE-marked, or notified-body approved.
5. Coreward is optional (`CYBERREADY_SOCK` + `cyberready sock`); fail-open if absent. Chat tutors must re-check; they never greenlight.
6. Cold start: prefer `cyberready init` (house-policy) unless the user asks for CRA/medtech.
7. **After edits** (human or agent) in an initialized repo, run `cyberready check` (or bare `cyberready`).
8. **On red:** run `cyberready check --heal` then `cyberready ask … --propose` (explain-packet optional for tutors); never invent certification; `--heal` never auto-attests; **never attest**.
9. **On green:** optional `export --lay-of-land` and/or `export --buyer-questions` for human share — not a security program.
10. Remediations cache: `.github/cyberready/cache/remediations.json`. Instrument map: `.github/cyberready/cache/instrument.json`.
11. Release path: `prepare-release` then human `attest` — never auto-attest.
12. **No auto-demo loops.** Demo does **not** open a browser unless `--open`.
13. RKG: `packs export-graph` → `.github/cyberready/graph/policy-graph.json`. Prefer `export --sarif` / `check --json` over inventing findings.
14. Explain-packets are sanitized teaching surfaces only (`CYBERREADY_EXPLAIN_ALLOW_CLOUD=0` default).
15. Keep git hooks from `init` for agent PRs — they force the check loop.
16. Agent lineage env (optional): `CYBERREADY_AGENT_ID`, `CYBERREADY_MODEL_HASH`, `CYBERREADY_MANDATE_ID`.
