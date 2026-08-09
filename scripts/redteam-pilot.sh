#!/usr/bin/env bash
# Pilot-prod adversarial scoreboard — the three invariants in docs/security-model.md.
# Exit non-zero if any case fails.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${CYBERREADY_BIN:-$ROOT/bin/cyberready}"
if [[ ! -x "$BIN" ]]; then
  go build -o "$BIN" ./cmd/cyberready
fi

PASS=0
FAIL=0

ok() { echo "  PASS  $*"; PASS=$((PASS + 1)); }
bad() { echo "  FAIL  $*"; FAIL=$((FAIL + 1)); }

echo "== redteam-pilot (pilot-prod contract) =="

# --- 1) Fake ./bin/cyberready must not be preferred by Action resolve ---
if grep -q 'Never prefer consumer \./bin/cyberready' action.yml && \
   ! grep -E '^\s*if \[ -x (\./)?bin/cyberready' action.yml && \
   grep -q 'source=built\|source=release' action.yml; then
  ok "1 Action resolve does not prefer workspace ./bin/cyberready"
else
  bad "1 Action resolve must not prefer unverified ./bin/cyberready"
fi

# --- 2) Missing SECURITY.md + dirty README — check --diff fails ---
TMP2="$(mktemp -d)"
cleanup() { rm -rf "$TMP2"; }
trap cleanup EXIT
set +e
(
  set -e
  cd "$TMP2"
  git init -q
  git config user.email "redteam@cyberready.local"
  git config user.name "Redteam"
  git commit --allow-empty -m init -q
  "$BIN" init --packs house-policy >/dev/null
  rm -f SECURITY.md
  git add -A && git -c commit.gpgsign=false commit --no-verify -m stubs -q || true
  echo "# dirty" >> README.md
  "$BIN" check --diff >/dev/null 2>&1
)
diff_code=$?
set -e
if [[ "$diff_code" -ne 0 ]]; then
  ok "2 check --diff fails when SECURITY.md missing (dirty README only)"
else
  bad "2 check --diff false-greened missing SECURITY.md"
fi

# --- 3) ApplyStubs .git/hooks/pre-commit refused ---
if go test ./internal/formhints/ -run TestApplyStubsRefusesDotGit -count=1 >/dev/null 2>&1; then
  ok "3 ApplyStubs refuses .git/hooks/pre-commit"
else
  bad "3 ApplyStubs .git jail regression"
fi

# --- 4) Pack path ../outside refused ---
if go test ./internal/packs/ -run 'TestValidatePackRefusesPathEscape|TestValidatePackSchema' -count=1 >/dev/null 2>&1; then
  ok "4 pack path ../outside refused at ValidatePack"
else
  bad "4 pack path escape not refused"
fi

# --- 5) Claim-safety still green ---
if ./scripts/claim-safety.sh >/dev/null 2>&1; then
  ok "5 claim-safety green"
else
  bad "5 claim-safety failed"
fi

# --- 6) Overlay compose loads medtech+CRA ---
if go test ./internal/packs/ -run TestComposeMedtechExtendsCRA -count=1 >/dev/null 2>&1; then
  ok "6 medtech extends cra-baseline compose"
else
  bad "6 overlay compose regression"
fi

# --- 7) SARIF non-empty on failure + ruleId=gate_id ---
if go test ./internal/contract/ -run TestSARIFRuleIDEqualsGateID -count=1 >/dev/null 2>&1; then
  ok "7 SARIF ruleId equals gate_id"
else
  bad "7 SARIF contract regression"
fi

# --- 8) policy-graph schema_version present ---
TMPG="$(mktemp -d)"
set +e
(
  set -e
  cd "$TMPG"
  git init -q
  git config user.email "redteam@cyberready.local"
  git config user.name "Redteam"
  git commit --allow-empty -m init -q
  "$BIN" init --bare --packs house-policy >/dev/null
  "$BIN" packs export-graph >/dev/null
  grep -q '"schema_version"' .github/cyberready/graph/policy-graph.json
)
graph_code=$?
set -e
rm -rf "$TMPG"
if [[ "$graph_code" -eq 0 ]]; then
  ok "8 policy-graph schema_version present"
else
  bad "8 policy-graph export missing schema_version"
fi

# --- 9) explain-packet airlock ---
if go test ./internal/contract/ -run TestExplainPacketAirlock -count=1 >/dev/null 2>&1; then
  ok "9 explain-packet airlock"
else
  bad "9 explain-packet airlock regression"
fi

# --- 10) Pack catalog freeze — only allowlisted pack ids ---
# Unlock: freeze review + explicit PR that updates this allowlist (no CI env escape hatch).
ALLOWED_PACK_IDS=$'cra-baseline\nhouse-policy\nmedtech-iec62304'
pack_allow_fail=0
for root in packs internal/packs/data; do
  [[ -d "$root" ]] || continue
  while IFS= read -r -d '' d; do
    id="$(basename "$d")"
    [[ "$id" == _* ]] && continue
    if ! printf '%s\n' "$ALLOWED_PACK_IDS" | grep -qx "$id"; then
      echo "  unexpected pack id under $root: $id" >&2
      pack_allow_fail=1
    fi
  done < <(find "$root" -mindepth 1 -maxdepth 1 -type d -print0 | sort -z)
done
# Embed twin must list the same three pack.json files.
embed_count="$(find internal/packs/data -mindepth 2 -maxdepth 2 -name pack.json | wc -l | tr -d ' ')"
tree_count="$(find packs -mindepth 2 -maxdepth 2 -name pack.json | wc -l | tr -d ' ')"
if [[ "$embed_count" != "3" || "$tree_count" != "3" ]]; then
  echo "  pack.json count mismatch: packs=$tree_count embed=$embed_count (want 3)" >&2
  pack_allow_fail=1
fi
if [[ "$pack_allow_fail" -eq 0 ]]; then
  ok "10 pack catalog allowlist (house-policy, cra-baseline, medtech-iec62304)"
else
  bad "10 pack catalog allowlist — new pack ids require freeze review + explicit PR"
fi

# --- 11) Import refuses theater pack without assurance_class ---
THEATER="$(mktemp -d)"
mkdir -p "$THEATER/theater-pack"
cat >"$THEATER/theater-pack/pack.json" <<'EOF'
{
  "id": "theater-pack",
  "name": "Theater pack",
  "version": "0.0.1",
  "description": "Structural draft without assurance_class for import honesty.",
  "rules": [
    {
      "id": "T-1",
      "severity": "low",
      "type": "POLICY_VIOLATION",
      "check": "file_present",
      "path": "README.md",
      "description": "README present",
      "remediation": "Add README.md",
      "expected": "README.md exists"
    }
  ]
}
EOF
IMPORT_DEST="$(mktemp -d)"
set +e
IMPORT_OUT="$(CYBERREADY_PACKS_DIR="$IMPORT_DEST" "$BIN" packs import "$THEATER" 2>&1)"
IMPORT_RC=$?
set -e
if [[ "$IMPORT_RC" -ne 0 ]] && echo "$IMPORT_OUT" | grep -qi 'assurance_class'; then
  ok "11 import refuses pack without assurance_class"
else
  bad "11 import must fail closed without assurance_class (rc=$IMPORT_RC)"
  echo "$IMPORT_OUT" >&2
fi
rm -rf "$THEATER" "$IMPORT_DEST"

echo ""
echo "redteam-pilot: $PASS passed, $FAIL failed"
if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
exit 0
