# Write your own pack

Curbpack evaluates **declared policy packs**. The engine has no CRA/FDA/SOC2 branches — only generic check kinds.

## Pack shape

```json
{
  "id": "acme-secure-coding",
  "name": "Acme Secure Coding Std",
  "version": "0.1.0",
  "assurance_class": "structural_draft",
  "description": "House rules for Acme eng — informational, not a certification.",
  "extends": "house-policy",
  "jurisdiction": "internal",
  "validity": { "effective_from": "2026-01-01", "effective_to": "2027-12-31" },
  "citations": [
    { "framework": "Acme", "instrument": "Secure Coding Std", "article": "§3" }
  ],
  "rules": [
    {
      "id": "ACME-SECURITY-MD",
      "severity": "high",
      "type": "POLICY_VIOLATION",
      "check": "file_present",
      "path": "SECURITY.md",
      "min_bytes": 80,
      "min_words": 20,
      "require_headers": ["# Security"],
      "description": "SECURITY.md missing or too thin.",
      "remediation": "Add reporting + response sections.",
      "expected": "SECURITY.md meets structural thresholds.",
      "citations": [
        { "framework": "Acme", "instrument": "Secure Coding Std", "article": "§3.1" }
      ]
    }
  ]
}
```

### Pack-level fields

| Field | Notes |
|-------|-------|
| `assurance_class` | **Required on `packs import`.** Use `structural_draft` for informational gates. Import refuses missing class and claim-adjacent names/descriptions (see `internal/packscmd`). |
| `extends` / `overlays` / `overlay` | Composition — see below |

### Rule binding (hollow-draft defense)

| Field | Behavior |
|-------|----------|
| `bind_repo_token` | When true, annex/file draft must mention a resolvable product token (directory name, `package.json` name, or `go.mod` module). |
| `require_tree_paths` | Listed repo-relative paths must exist (e.g. `["README.md"]`). |

### Composition (RKG overlays)

- `extends`: base pack id (loaded first; cycles refused)
- `overlays`: optional ordered pack ids applied after the base
- `overlay`: optional RFC 7386 merge-patch object applied to the pack JSON
- Rules are **unioned by `id`** — later packs win
- `medtech-iec62304` is a real overlay: `"extends": "cra-baseline"`

```bash
curbpack init --packs medtech-iec62304   # composes CRA + medtech rules
curbpack packs export-graph              # writes .github/curbpack/graph/policy-graph.json
curbpack packs doctor                    # expired / superseded / pin skew
```

## Supported `check` values

| Check | Fields | Behavior |
|-------|--------|----------|
| `file_present` / `annex_file` | `path`, `min_bytes`, `min_words`, `require_headers`; optional `bind_repo_token`, `require_tree_paths` | File exists, size/words/headers; binding optional |
| `anti_placeholder` | `paths` | Reject TODO / lorem / `[insert …]` in listed annex drafts |
| `manifest_dep_ban` / `npm_dep_ban` | `package`, `banned_versions` | Ban pins in `package.json` |
| `text_forbid` | `paths`, `pattern` | Regex forbid (e.g. secret-like strings) |
| `import_reach` | — | Optional AST reachability (MVP) |
| `fresh` | `path`, `max_age_days` and/or `since_ref` | File must exist and last commit must be within age or after ref |
| `owned` | `path`, `bind_repo_token: true`; optional `require_git_author_email`, `require_git_author_name` | Repo-bound draft; optional git author match on last commit touching path |

## Load paths

1. **Embedded** in the binary (`internal/packs/data/<id>/pack.json`)
2. **Override dir:** `CURBPACK_PACKS_DIR=/path` with `<id>/pack.json`
3. **Air-gap:** `curbpack packs import ./bundle-dir` — ValidatePack + `assurance_class` + claim-adjacent refuse; writes `.curbpack-pack.sha256` sidecar

## Activate

```bash
curbpack init --packs acme-secure-coding
# or edit .curbpack.json:
# { "packs": ["acme-secure-coding"] }
curbpack check
```

`init` scaffolds **only** paths referenced by composed pack rules (includes `extends` bases).

## Schema validation

Packs are validated on load (id/name/version/rules, supported checks, path jail, citation date windows). Invalid packs fail fast — they never silently skip. `Compose` rejects unknown `extends` cycles.

## Claim safety

Packs prepare evidence for human review. Passing gates is not certification for any regulation or internal audit regime.

## Diff vs full validate

| Command / op | What it does | Release-gate safe? |
|--------------|--------------|-------------------|
| `curbpack check --diff` | Porcelain: evaluates rules that touch the dirty/changed set (`RuleTouchesDiff`; basename match). `file_present` / `annex_file` always run. | **No** — local speed only; do not treat green `--diff` as release evidence |
| `curbpack validate` / sock `validate_delta` | Full quiet validate of composed packs | **Yes** — authoritative |

Do not change `RuleTouchesDiff` matching mid-freeze. Stakeholder matrix: [github-readiness-gaps.md](github-readiness-gaps.md#diff-vs-validate_delta).
