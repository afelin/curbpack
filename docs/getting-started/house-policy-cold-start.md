# House-policy cold start

For internal IT / engineering teams who want evidence gates **without** EU CRA framing on day one.

```bash
cyberready init
cyberready check
```

`init` defaults (one line):

| Default | What you get |
|---------|----------------|
| Pack | `house-policy` |
| Hooks | pre-commit → `cyberready check` |
| Skill | `.cursor/skills/cyberready/SKILL.md` |
| IDE | VS Code / Cursor tasks |

Use `--bare` to skip hooks/skill/ide. Override packs with `--packs` only when you need CRA/medtech on day one.

Add CRA or medtech later:

```bash
# edit .cyberready.json packs array, or re-init in a fresh branch:
cyberready init --packs cra-baseline,house-policy
```

Claim safety: gate pass prepares evidence for human review — not certification.
