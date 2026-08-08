# GTM OSS — NON-PRODUCT / INTERNAL GTM

> **NON-PRODUCT / INTERNAL GTM — not for Pages, not for adopters.**
> Do not link this tree from the product README. GitHub Pages quarantine refuses `*gtm*` under `site/`.
> Claim-safety still applies to copy here; this is ops amplify kit only.

# GTM OSS — claim-safe templates

Manual amplify kit for X, LinkedIn, and README badges. **No auto-DM. No spam bots.**

Copy numbers from the latest `cyberready-scoreboard` CI artifact or weekly Discussion.

Every post must keep this disclaimer:

> Prepares evidence for human review — not a conformity assessment or certification.

---

## X / Twitter (3 posts)

### A — Time-to-demo

```
Stranger path, no Go:

curl -fsSL …/scripts/install.sh | sh
cyberready doctor && cyberready demo

Sandbox green in ~{N}s. Evidence for humans — not certification.
```

### B — GitHub Action

```
5 lines of YAML → PR comment with readiness thermometer + top fails.

uses: afelin/cyberready@vX

Claim-safe by design. Try the Action or `cyberready demo`.
```

### C — House-policy cold start

```
Cold start without EU-law framing:

cyberready init --packs house-policy --hooks --skill --ide
cyberready check

Packs encode policy. The binary stays industry-agnostic.
```

---

## LinkedIn (1 post)

```
We open-sourced CyberReady+: a local evidence CLI for CRA / house policy / sector packs.

Goal isn’t vanity stars — it’s ~100 people who actually ran `cyberready demo` or the GitHub Action and can say whether they’d recommend it.

Install (no Go): curl install.sh | sh → doctor → demo (temp sandbox, never touches your product).

Important: this prepares evidence for human review. It does not certify conformity, issue CE marks, or replace auditors.

If you try it, open a “Tester report” issue or drop a thermometer screenshot in Discussions → Show and tell.
```

---

## README badge

```markdown
[![cyberready-check](https://img.shields.io/badge/cyberready-check-2ea44f?logo=github)](https://github.com/afelin/cyberready)
```

Optional shields.io workflow badge once the Action runs on your repo:

```markdown
[![CyberReady](https://github.com/OWNER/REPO/actions/workflows/cyberready.yml/badge.svg)](https://github.com/OWNER/REPO/actions/workflows/cyberready.yml)
```

---

## Discussions / Show & tell

See [show-and-tell.md](./show-and-tell.md).
