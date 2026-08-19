#!/usr/bin/env bash
# Claim-safety killer: deny certification theater in docs + runtime CLI captures.
# Tool does not prevent regulatory action; it must not present as conformity.
# Brand: product mark is Curbpack. "CyberReady" allowed only in migration / NOTICE /
# changelog historical lines (and this script's allowlist).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CHANGED_ONLY=0; for arg in "$@"; do [[ "$arg" == --changed-only ]] && CHANGED_ONLY=1; done

BIN="${CURBPACK_BIN:-${CYBERREADY_BIN:-$ROOT/bin/curbpack}}"
go build -o "$BIN" ./cmd/curbpack

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Combined deny patterns (positive certification theater).
# Lines with claim-safe negation framing are filtered out in Python.
DENY_RE='we are (CE[- ])?certified|product is certified|officially certified|curbpack certifies|cyberready certifies|notified[- ]body approved|approved by (a )?notified body|conformity assessment (complete|passed|successful)|CE marking (issued|granted|obtained)|is CE[- ]marked|has been CE[- ]marked|certified conformity|EU CRA Baseline|we are CRA compliant|CRA compliant|RISE[- ]approved|RISE[- ]certified|FRA[- ]approved|NCSC[- ]approved|agency[- ]endorsed'

SAFE_RE='not (a |an )?(conformity|certif|CE)|does not certify|never claim|no certification|not CE|replace a notified|notified-body approval|certification_claimed.: false|Certification claimed: \*\*no\*\*|not a certification product|Not a certification|informational|draft structure|not essential-requirements|structural_draft|structural (file/header )?gates|not conformity assessment|funder, not certifier|not product certifier|not (this product.s )?certifier'

scan_text() {
  local label="$1"
  local file="$2"
  python3 - "$label" "$file" "$DENY_RE" "$SAFE_RE" <<'PY'
import re, sys
label, path, deny_s, safe_s = sys.argv[1:5]
deny = re.compile(deny_s, re.I)
safe = re.compile(safe_s, re.I)
hit = 0
try:
    text = open(path, errors="replace").read()
except FileNotFoundError:
    sys.exit(0)
for i, line in enumerate(text.splitlines(), 1):
    if safe.search(line):
        continue
    m = deny.search(line)
    if m:
        print(f"CLAIM-SAFETY FAIL [{label}:{i}]: /{m.group(0)}/ → {line}", file=sys.stderr)
        hit = 1
sys.exit(hit)
PY
}

# Brand fence: CyberReady / CyberReady+ only in historical allowlist files.
scan_brand() {
  local file="$2"
  local label="$1"
  python3 - "$label" "$file" <<'PY'
import re, sys
label, path = sys.argv[1:3]
# Allow migration/NOTICE/CHANGELOG wholly; elsewhere forbid brand leftovers.
allow_names = {
    "docs/migration-cyberready-to-curbpack.md",
    "NOTICE",
    "CHANGELOG.md",
    "scripts/claim-safety.sh",
}
rel = path
# normalize
if rel.startswith("./"):
    rel = rel[2:]
if rel in allow_names or rel.endswith("/migration-cyberready-to-curbpack.md"):
    sys.exit(0)
pat = re.compile(r"CyberReady\+?|cyberready", re.I)
hit = 0
try:
    text = open(path, errors="replace").read()
except FileNotFoundError:
    sys.exit(0)
for i, line in enumerate(text.splitlines(), 1):
    # allow links / titles pointing at the migration doc
    if "migration-cyberready-to-curbpack" in line:
        continue
    # allow code comments that document dual-read legacy keys explicitly
    if "legacy" in line.lower() and ("cyberready" in line.lower() or "CYBERREADY" in line):
        continue
    if "dual-read" in line.lower() or "fallback" in line.lower():
        if "CYBERREADY" in line or "cyberready" in line.lower():
            continue
    if ".cyberready.json" in line or ".github/cyberready" in line or "refs/notes/cyberready" in line:
        continue
    if "CYBERREADY_" in line and ("CURBPACK_" in line or "legacy" in line.lower() or "fallback" in line.lower()):
        continue
    m = pat.search(line)
    if m:
        print(f"BRAND-SAFETY FAIL [{label}:{i}]: /{m.group(0)}/ → {line}", file=sys.stderr)
        hit = 1
sys.exit(hit)
PY
}

FAIL=0

echo "== claim-safety: docs/README/skills =="
DOC_FILES=()
while IFS= read -r f; do
  DOC_FILES+=("$f")
done < <(
  find README.md SECURITY.md NOTICE LICENSE AGENTS.md CLAUDE.md docs papers site .cursor/skills internal/skilldata action.yml examples \
    .github/ISSUE_TEMPLATE .github/workflows .github/copilot-instructions.md scripts \
    \( -type f \( -name '*.md' -o -name '*.yml' -o -name '*.yaml' -o -name '*.sh' -o -name '*.ps1' -o -name '*.html' -o -name '*.txt' -o -name 'LICENSE' -o -name 'NOTICE' -o -name 'copilot-instructions.md' \) \) \
    2>/dev/null | grep -v 'scripts/claim-safety\.sh$' | grep -v '/gtm-oss/' | grep -v 'workflows/pages\.yml$' | sort -u
)

if [[ "$CHANGED_ONLY" -eq 1 ]]; then
  mapfile -t CHANGED < <(git diff --name-only origin/main...HEAD 2>/dev/null || git diff --name-only HEAD~1)
  NEXT=(); for f in "${DOC_FILES[@]}"; do for c in "${CHANGED[@]}"; do [[ "$f" == "$c" ]] && NEXT+=("$f") && break; done; done; DOC_FILES=("${NEXT[@]}")
fi

for f in "${DOC_FILES[@]}"; do
  if ! scan_text "$f" "$f"; then
    FAIL=1
  fi
  if ! scan_brand "$f" "$f"; then
    FAIL=1
  fi
done

echo "== claim-safety: pack.json display strings =="
PACK_FILES=()
while IFS= read -r f; do
  PACK_FILES+=("$f")
done < <(
  find packs internal/packs/data \
    \( -type f -name 'pack.json' \) \
    2>/dev/null | sort -u
)
for f in "${PACK_FILES[@]}"; do
  if ! scan_text "$f" "$f"; then
    FAIL=1
  fi
done

echo "== claim-safety: runtime CLI captures =="
"$BIN" doctor >"$TMP/doctor.out" 2>&1 || true
scan_text "doctor" "$TMP/doctor.out" || FAIL=1
scan_brand "doctor" "$TMP/doctor.out" || FAIL=1

DEMO="$TMP/demo"
"$BIN" demo --out "$DEMO" --keep >"$TMP/demo.out" 2>&1
scan_text "demo" "$TMP/demo.out" || FAIL=1
scan_brand "demo" "$TMP/demo.out" || FAIL=1
if [[ -f "$DEMO/review-pack/buyer-onepager.html" ]]; then
  scan_text "buyer-onepager" "$DEMO/review-pack/buyer-onepager.html" || FAIL=1
  scan_brand "buyer-onepager" "$DEMO/review-pack/buyer-onepager.html" || FAIL=1
fi

FIX="$TMP/fix"
mkdir -p "$FIX"
(
  cd "$FIX"
  git init -q
  git config user.email "ci@curbpack.local"
  git config user.name "CI"
  git commit --allow-empty -m init -q
  "$BIN" init --packs house-policy >"$TMP/init.out" 2>&1
  "$BIN" check >"$TMP/check.out" 2>&1 || true
  "$BIN" prepare-release >"$TMP/prepare.out" 2>&1 || true
)
scan_text "init" "$TMP/init.out" || FAIL=1
scan_brand "init" "$TMP/init.out" || FAIL=1
scan_text "check" "$TMP/check.out" || FAIL=1
scan_brand "check" "$TMP/check.out" || FAIL=1
scan_text "prepare-release" "$TMP/prepare.out" || FAIL=1
scan_brand "prepare-release" "$TMP/prepare.out" || FAIL=1
if [[ -f "$FIX/review-pack/buyer-onepager.html" ]]; then
  scan_text "prepare-onepager" "$FIX/review-pack/buyer-onepager.html" || FAIL=1
  scan_brand "prepare-onepager" "$FIX/review-pack/buyer-onepager.html" || FAIL=1
fi
(
  cd "$FIX"
  "$BIN" share --bundle >"$TMP/share-bundle.out" 2>&1 || true
)
scan_text "share-bundle" "$TMP/share-bundle.out" || FAIL=1
scan_brand "share-bundle" "$TMP/share-bundle.out" || FAIL=1
if [[ -f "$FIX/review-pack/evidence-bundle.html" ]]; then
  if ! grep -q 'curbpack-bundle-schema:1' "$FIX/review-pack/evidence-bundle.html"; then
    echo "CLAIM-SAFETY FAIL [bundle]: missing curbpack-bundle-schema marker" >&2
    FAIL=1
  fi
  scan_text "evidence-bundle" "$FIX/review-pack/evidence-bundle.html" || FAIL=1
  scan_brand "evidence-bundle" "$FIX/review-pack/evidence-bundle.html" || FAIL=1
fi
if [[ -f "$FIX/.github/curbpack/cache/latest_action_report.md" ]]; then
  scan_text "action-report" "$FIX/.github/curbpack/cache/latest_action_report.md" || FAIL=1
  scan_brand "action-report" "$FIX/.github/curbpack/cache/latest_action_report.md" || FAIL=1
fi

"$BIN" help >"$TMP/help.out" 2>&1 || true
scan_text "help" "$TMP/help.out" || FAIL=1
scan_brand "help" "$TMP/help.out" || FAIL=1

# scan --badge: deny state assertions (grep-based; badge text is time-dependent).
BADGE="$TMP/badge"
mkdir -p "$BADGE"
(
  cd "$BADGE"
  git init -q
  git config user.email "ci@curbpack.local"
  git config user.name "CI"
  git commit --allow-empty -m init -q
  echo '{"name":"badgeco","version":"1.0.0"}' > package.json
  "$BIN" scan --badge >"$TMP/badge-cold.out" 2>&1 || true
  "$BIN" fix --art14 --yes >"$TMP/badge-fix.out" 2>&1 || true
  "$BIN" scan --badge >"$TMP/badge-postfix.out" 2>&1 || true
)
scan_text "scan-badge-cold" "$TMP/badge-cold.out" || FAIL=1
scan_text "scan-badge-postfix" "$TMP/badge-postfix.out" || FAIL=1
for f in "$TMP/badge-cold.out" "$TMP/badge-postfix.out"; do
  if grep -qiE 'failing|not started|CRA[- ]ready|CRA compliant|\bgreen\b|passing|0 failing' "$f"; then
    echo "CLAIM-SAFETY FAIL [scan-badge]: state assertion in badge output → $(grep -iE 'failing|not started|CRA|green|passing' "$f" | head -1)" >&2
    FAIL=1
  fi
  if grep -q 'Drafted' "$f"; then
    echo "CLAIM-SAFETY FAIL [scan-badge]: badge must not expose Drafted field" >&2
    FAIL=1
  fi
done
if ! grep -q 'not rehearsed' "$TMP/badge-postfix.out"; then
  echo "CLAIM-SAFETY FAIL [scan-badge-postfix]: fix alone must not produce rehearsed badge" >&2
  FAIL=1
fi

# Skill install path must be curbpack (not legacy cyberready skill dir name in output).
if [[ -f "$FIX/.cursor/skills/curbpack/SKILL.md" ]]; then
  if ! grep -q '^name: curbpack$' "$FIX/.cursor/skills/curbpack/SKILL.md"; then
    echo "BRAND-SAFETY FAIL [skill]: missing frontmatter name: curbpack" >&2
    FAIL=1
  fi
fi
if [[ -d "$FIX/.cursor/skills/cyberready" ]]; then
  echo "BRAND-SAFETY FAIL [skill]: init wrote legacy .cursor/skills/cyberready/" >&2
  FAIL=1
fi

if [[ "$FAIL" -ne 0 ]]; then
  echo "claim-safety: FAILED — certification theater or brand leftovers detected" >&2
  exit 1
fi
echo "claim-safety: OK"
exit 0
