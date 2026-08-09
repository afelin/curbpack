#!/usr/bin/env bash
# Claim-safety killer: deny certification theater in docs + runtime CLI captures.
# Tool does not prevent regulatory action; it must not present as conformity.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${CYBERREADY_BIN:-$ROOT/bin/cyberready}"
if [[ ! -x "$BIN" ]]; then
  go build -o "$BIN" ./cmd/cyberready
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Combined deny patterns (positive certification theater).
# Lines with claim-safe negation framing are filtered out in Python.
DENY_RE='we are (CE[- ])?certified|product is certified|officially certified|cyberready certifies|notified[- ]body approved|approved by (a )?notified body|conformity assessment (complete|passed|successful)|CE marking (issued|granted|obtained)|is CE[- ]marked|has been CE[- ]marked|certified conformity|EU CRA Baseline|we are CRA compliant|CRA compliant'

SAFE_RE='not (a |an )?(conformity|certif|CE)|does not certify|never claim|no certification|not CE|replace a notified|notified-body approval|certification_claimed.: false|Certification claimed: \*\*no\*\*|not a certification product|Not a certification|informational|draft structure|not essential-requirements|structural_draft|structural (file/header )?gates|not conformity assessment'

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

FAIL=0

echo "== claim-safety: docs/README/skills =="
DOC_FILES=()
while IFS= read -r f; do
  DOC_FILES+=("$f")
done < <(
  find README.md SECURITY.md NOTICE LICENSE docs papers site .cursor/skills internal/skilldata action.yml \
    .github/ISSUE_TEMPLATE .github/workflows scripts \
    \( -type f \( -name '*.md' -o -name '*.yml' -o -name '*.yaml' -o -name '*.sh' -o -name '*.html' -o -name 'LICENSE' -o -name 'NOTICE' \) \) \
    2>/dev/null | grep -v 'scripts/claim-safety\.sh$' | grep -v '/gtm-oss/' | grep -v 'workflows/pages\.yml$' | sort -u
)

for f in "${DOC_FILES[@]}"; do
  if ! scan_text "$f" "$f"; then
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

DEMO="$TMP/demo"
"$BIN" demo --out "$DEMO" --keep >"$TMP/demo.out" 2>&1
scan_text "demo" "$TMP/demo.out" || FAIL=1
if [[ -f "$DEMO/review-pack/buyer-onepager.html" ]]; then
  scan_text "buyer-onepager" "$DEMO/review-pack/buyer-onepager.html" || FAIL=1
fi

FIX="$TMP/fix"
mkdir -p "$FIX"
(
  cd "$FIX"
  git init -q
  git config user.email "ci@cyberready.local"
  git config user.name "CI"
  git commit --allow-empty -m init -q
  "$BIN" init --packs house-policy >"$TMP/init.out" 2>&1
  "$BIN" check >"$TMP/check.out" 2>&1 || true
  "$BIN" prepare-release >"$TMP/prepare.out" 2>&1 || true
)
scan_text "init" "$TMP/init.out" || FAIL=1
scan_text "check" "$TMP/check.out" || FAIL=1
scan_text "prepare-release" "$TMP/prepare.out" || FAIL=1
if [[ -f "$FIX/review-pack/buyer-onepager.html" ]]; then
  scan_text "prepare-onepager" "$FIX/review-pack/buyer-onepager.html" || FAIL=1
fi
if [[ -f "$FIX/.github/cyberready/cache/latest_action_report.md" ]]; then
  scan_text "action-report" "$FIX/.github/cyberready/cache/latest_action_report.md" || FAIL=1
fi

"$BIN" help >"$TMP/help.out" 2>&1 || true
scan_text "help" "$TMP/help.out" || FAIL=1

if [[ "$FAIL" -ne 0 ]]; then
  echo "claim-safety: FAILED — certification theater detected" >&2
  exit 1
fi
echo "claim-safety: OK"
exit 0
