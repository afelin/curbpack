# Buyer evidence (quiet path)

Share structural gate evidence with a buyer or auditor without GRC SaaS.

```bash
cyberready check
cyberready export --buyer-questions
# → .github/cyberready/cache/buyer-questions.md (+ .json)
```

Hand the Markdown checklist to the human reviewer. When drafts are ready, `cyberready prepare-release` then human `cyberready attest`.

**Local pack gates. Humans review. Not conformity assessment.** Not CE / not notified-body. Rows carry `assurance_class: structural_draft`.

See also: [Daily loop](daily-loop.md) · [Intent vs Scope](../intent-vs-scope.md)
