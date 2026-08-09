# Buyer evidence (quiet path)

Share structural gate evidence with a buyer or auditor without GRC SaaS.

```bash
cyberready check
cyberready export --buyer-questions
# → .github/cyberready/cache/buyer-questions.md (+ .json)
```

Optional shareable map (deps summary, secret-hit count, informational watchlist∩SBOM — not a CVE product):

```bash
cyberready export --lay-of-land
# → .github/cyberready/cache/lay-of-land.md (+ .json)
```

Hand the Markdown checklist to the human reviewer. When drafts are ready, `cyberready prepare-release` then human `cyberready attest`. Unsigned one-pagers say **UNSIGNED — not cryptographically verified** until ssh-agent attest.

**Local pack gates. Humans review. Not conformity assessment.** Not CE / not notified-body. Rows carry `assurance_class: structural_draft`. Buyer-questions header includes `attestation_status: none | ssh-agent`.

See also: [Daily loop](daily-loop.md) · [Intent vs Scope](../intent-vs-scope.md)
