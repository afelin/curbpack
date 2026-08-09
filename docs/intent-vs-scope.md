# Intent vs Scope

**Local pack gates. Humans review. Not conformity assessment.**

Sixty-second clarity for buyers, auditors, and agents. Gate pass is **evidence for human review** — not conformity assessment, CE marking, or certification.

| Column | Content |
|--------|---------|
| **Intent (why)** | SMEs and suppliers need continuous, local, shareable *evidence* for buyers, auditors, and agents — without GRC SaaS or uploading IP to a cloud policy brain. |
| **CyberReady+ scope (now)** | Pack JSON gates, `check` / `heal`, review-pack, CycloneDX / OpenVEX drafts, Git Notes attest, HPURL pointer, Action + SARIF, sock `validate_delta`, local regulation knowledge graph export, explain-packets for tutors. |
| **Not in scope (OSS)** | Conformity assessment, CE, OPA/Rego, LSP, syscall tracers, FIDO/EFOS, DNSSEC, cloud policy brain, LLM-as-judge, badge marketplace, gtm-oss on site, second pin, pack catalog growth before partner habit proof. |
| **Pack catalog freeze** | Only `house-policy`, `cra-baseline`, `medtech-iec62304` (ids). Enforced by `scripts/redteam-pilot.sh` allowlist; unlock only via freeze review + explicit PR (no CI env escape hatch). |
| **v3.33 spec** | R&D north star / EE backlog — not the adoption contract. |
| **IP / chat boundary** | Raw source and secrets never leave the machine for “compliance chat.” Only sanitized GateFailure / RKG explain-packets may be sent to Coreward or an optional cloud tutor the operator explicitly chooses. |
| **Promotion bar** | `./scripts/redteam-pilot.sh` green. |

## Deterministic judge / generative tutor

```
repo → CyberReady (gates, packs, RKG, attest)
         ↓
   explain-packet (airlock)
         ↓
   Coreward / local chat / optional Flash  ← summarizes only; must re-check
```

CyberReady decides pass/fail. Chat may draft prose or remediation suggestions. Chat never greenlights gates and never writes attest capsules.

**Optional tutor:** Coreward (or local chat) may consume an airlocked explain-packet. The packet never greenlights — after any proposed fix you must re-check (`validate_delta` / `cyberready check`). Recorded loop: [`scripts/dogfood-explain-recheck.sh`](../scripts/dogfood-explain-recheck.sh) · [Coreward bridge](coreward-bridge.md).

## Agentic coding: instrument panel / not AI security product

Agents and humans share one loop: edit → `cyberready check` → read the instrument panel (covenant + optional Δ readiness/deps/secret-hits) → on red heal/ask; on green optional `--lay-of-land` / `--buyer-questions` for humans. This is **not** an AI security product, SCA/CVE platform, or certification engine. Hooks keep agent PRs honest; tutors still require re-`validate_delta` / re-check before any “fixed” claim.

**Activate in 60s:** [getting-started / 60-second paths](getting-started/60-second-paths.md).

See also: [Promotion firewall](promotion-firewall.md) · [Coreward bridge](coreward-bridge.md) · [Write your own pack](write-your-own-pack.md) · [Security model](security-model.md)
