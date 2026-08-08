# Intent vs Scope

Sixty-second clarity for buyers, auditors, and agents. Gate pass is **evidence for human review** — not conformity assessment, CE marking, or certification.

| Column | Content |
|--------|---------|
| **Intent (why)** | SMEs and suppliers need continuous, local, shareable *evidence* for buyers, auditors, and agents — without GRC SaaS or uploading IP to a cloud policy brain. |
| **CyberReady+ scope (now)** | Pack JSON gates, `check` / `heal`, review-pack, CycloneDX / OpenVEX drafts, Git Notes attest, HPURL pointer, Action + SARIF, sock `validate_delta`, local regulation knowledge graph export, explain-packets for tutors. |
| **Not in scope (OSS)** | Conformity assessment, CE, OPA/Rego, LSP, syscall tracers, FIDO/EFOS, DNSSEC, cloud policy brain, LLM-as-judge. |
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

**Activate in 60s:** [getting-started / 60-second paths](getting-started/60-second-paths.md).

See also: [Coreward bridge](coreward-bridge.md) · [Write your own pack](write-your-own-pack.md) · [Security model](security-model.md)
