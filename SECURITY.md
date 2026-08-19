# Security

## Reporting

Email **security@curbpack.dev** (or open a private GitHub Security Advisory on
[afelin/curbpack](https://github.com/afelin/curbpack)) with vulnerability details.
Do **not** open public issues for sensitive reports.

Include: affected version/commit, reproduction steps, impact, and any suggested fix.

## Response

We acknowledge within **two business days** and coordinate disclosure timelines with reporters.
Critical issues affecting install integrity or evidence tampering are prioritized.

## Scope

In scope: the `curbpack` CLI, embedded packs, GitHub Action, `scripts/install.sh`, and the **`curbpack` npm wrapper** (cached binary under `~/.curbpack/bin` or `%LOCALAPPDATA%\curbpack`).

Out of scope: customer product repos scanned by Curbpack; third-party packs you import;
misuse of gate results as certification claims.

## Claim safety

Curbpack prepares evidence for **human review**. A green `check` is **not** a
conformity assessment, CE mark, notified-body approval, or certification.
