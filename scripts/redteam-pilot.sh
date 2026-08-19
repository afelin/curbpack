#!/usr/bin/env bash
# Pilot-prod adversarial scoreboard — the three invariants in docs/security-model.md.
# Exit non-zero if any case fails.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${CURBPACK_BIN:-$ROOT/bin/curbpack}"
if [[ ! -x "$BIN" ]]; then
  go build -o "$BIN" ./cmd/curbpack
fi

PASS=0
FAIL=0

ok() { echo "  PASS  $*"; PASS=$((PASS + 1)); }
bad() { echo "  FAIL  $*"; FAIL=$((FAIL + 1)); }

echo "== redteam-pilot (pilot-prod contract) =="

# --- 1) Fake ./bin/curbpack must not be preferred by Action resolve ---
if grep -q 'Never prefer consumer \./bin/curbpack' action.yml && \
   ! grep -E '^\s*if \[ -x (\./)?bin/curbpack' action.yml && \
   grep -q 'source=built\|source=release' action.yml; then
  ok "1 Action resolve does not prefer workspace ./bin/curbpack"
else bad "1 Action resolve must not prefer unverified ./bin/curbpack"; fi

# --- 2) Missing SECURITY.md + dirty README — check --diff fails ---
TMP2="$(mktemp -d)"
cleanup() { rm -rf "$TMP2"; }
trap cleanup EXIT
set +e
(
  set -e
  cd "$TMP2"
  git init -q
  git config user.email "redteam@curbpack.local"
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
[[ "$diff_code" -ne 0 ]] && ok "2 check --diff fails when SECURITY.md missing (dirty README only)" || \
  bad "2 check --diff false-greened missing SECURITY.md"

# --- 5) Claim-safety still green ---
./scripts/claim-safety.sh >/dev/null 2>&1 && ok "5 claim-safety green" || bad "5 claim-safety failed"

# --- 8) policy-graph schema_version present ---
TMPG="$(mktemp -d)"
set +e
(
  set -e
  cd "$TMPG"
  git init -q
  git config user.email "redteam@curbpack.local"
  git config user.name "Redteam"
  git commit --allow-empty -m init -q
  "$BIN" init --bare --packs house-policy >/dev/null
  "$BIN" packs export-graph >/dev/null
  grep -q '"schema_version"' .github/curbpack/graph/policy-graph.json
)
graph_code=$?
set -e
rm -rf "$TMPG"
[[ "$graph_code" -eq 0 ]] && ok "8 policy-graph schema_version present" || \
  bad "8 policy-graph export missing schema_version"

# --- 10) Pack catalog freeze — only allowlisted pack ids ---
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
embed_count="$(find internal/packs/data -mindepth 2 -maxdepth 2 -name pack.json | wc -l | tr -d ' ')"
tree_count="$(find packs -mindepth 2 -maxdepth 2 -name pack.json | wc -l | tr -d ' ')"
if [[ "$embed_count" != "3" || "$tree_count" != "3" ]]; then
  echo "  pack.json count mismatch: packs=$tree_count embed=$embed_count (want 3)" >&2
  pack_allow_fail=1
fi
[[ "$pack_allow_fail" -eq 0 ]] && ok "10 pack catalog allowlist (house-policy, cra-baseline, medtech-iec62304)" || \
  bad "10 pack catalog allowlist — new pack ids require freeze review + explicit PR"

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
IMPORT_OUT="$(CURBPACK_PACKS_DIR="$IMPORT_DEST" "$BIN" packs import "$THEATER" 2>&1)"
IMPORT_RC=$?
set -e
if [[ "$IMPORT_RC" -ne 0 ]] && echo "$IMPORT_OUT" | grep -qi 'assurance_class'; then
  ok "11 import refuses pack without assurance_class"
else
  bad "11 import must fail closed without assurance_class (rc=$IMPORT_RC)"
  echo "$IMPORT_OUT" >&2
fi
rm -rf "$THEATER" "$IMPORT_DEST"

# --- 12) Stable contracts nave — internal/contract/ + go test ./... (CI) ---
[[ -f docs/stable-contracts.md && -f internal/contract/stable_ops_test.go && \
    -f internal/contract/explain_coreward_consumer_test.go ]] && \
  ok "12 stable contracts (sock ops + explain consumer + docs sync)" || \
  bad "12 stable contracts nave — missing contract tests or docs/stable-contracts.md"

# --- 16) share --bundle offline schema marker ---
TMPB="$(mktemp -d)"
set +e
(
  set -e
  cd "$TMPB"
  git init -q
  git config user.email "redteam@curbpack.local"
  git config user.name "Redteam"
  git commit --allow-empty -m init -q
  "$BIN" init --packs house-policy >/dev/null
  "$BIN" share --bundle >/dev/null 2>&1 || true
  test -f review-pack/evidence-bundle.html
  grep -q 'curbpack-bundle-schema:1' review-pack/evidence-bundle.html
)
bundle_code=$?
set -e
rm -rf "$TMPB"
[[ "$bundle_code" -eq 0 ]] && ok "16 share --bundle offline evidence-bundle schema" || \
  bad "16 share --bundle must write evidence-bundle.html with schema marker"

# --- 17) drift: attest then commit → attest_commit_behind ---
TMPD="$(mktemp -d)"
set +e
(
  set -e
  cd "$TMPD"
  export PATH="$(dirname "$BIN"):$PATH"
  git init -q
  git config user.email "redteam@curbpack.local"
  git config user.name "Redteam"
  git commit --allow-empty -m init -q
  "$BIN" init --packs house-policy >/dev/null
  git add -A && git -c commit.gpgsign=false commit --no-verify -m stubs -q || true
  "$BIN" attest --allow-dirty >/dev/null 2>&1 || "$BIN" attest >/dev/null 2>&1 || true
  git commit --allow-empty --no-verify -m after-attest -q
  out="$("$BIN" drift --json 2>/dev/null)"
  echo "$out" | grep -q 'attest_commit_behind'
)
drift_code=$?
set -e
rm -rf "$TMPD"
[[ "$drift_code" -eq 0 ]] && ok "17 drift attest_commit_behind after new commit" || \
  bad "17 drift must emit attest_commit_behind when HEAD moves past bind"

# --- 18) Windows asset name + Action Linux/macOS-only lock ---
if grep -q 'curbpack_windows_amd64.exe' scripts/install-manifest.json && \
   grep -Eq 'windows/amd64|windows_amd64' .github/workflows/release.yml && \
   grep -q 'curbpack_windows_amd64.exe' scripts/install.ps1 && \
   grep -qi 'Linux/macOS' docs/getting-started/install.md && \
   ! grep -qiE 'runs-on:.*windows' action.yml && \
   grep -q 'mingw\*|msys\*|cygwin\*|windows\*' action.yml && \
   grep -A5 "^  heal:" action.yml | grep -q "default: 'false'" && \
   grep -q 'Unix-only' docs/stable-contracts.md; then
  ok "18 windows asset + Action Linux/macOS-only + sock Unix-only doc lock"
else bad "18 windows asset / Action honesty / sock Unix-only regression"; fi

# --- 19) Forged note user_touch=ssh-agent-signed must not show verified ---
TMP19="$(mktemp -d)"
set +e
(
  set -e
  cd "$TMP19"
  git init -q && git config user.email "redteam@curbpack.local" && git config user.name "Redteam"
  git commit --allow-empty -m init -q
  head="$(git rev-parse HEAD)"
  forge='{"schema_version":"v3.34-result-bind","commit_sha":"'"$head"'","state_hash":"deadbeef","signer":"attacker","user_touch":"ssh-agent-signed","ssh_signature":"agent-bind:fake"}'
  git notes --ref=curbpack add -f -m "$forge" "$head"
  out="$("$BIN" view 2>&1)"
  echo "$out" | grep -q 'UNSIGNED — not cryptographically verified'
  ! echo "$out" | grep -qE 'Signature:.*ssh-agent-signed'
)
case19_code=$?
set -e
rm -rf "$TMP19"
[[ "$case19_code" -eq 0 ]] && ok "19 forged attest note must not show verified" || \
  bad "19 forged user_touch/agent-bind must stay unsigned"

# --- 20) Hand-edited latest_failure.json fake green must not false-green ---
TMP20="$(mktemp -d)"
set +e
(
  set -e
  cd "$TMP20"
  git init -q && git config user.email "redteam@curbpack.local" && git config user.name "Redteam"
  git commit --allow-empty -m init -q
  "$BIN" init --packs house-policy >/dev/null
  head="$(git rev-parse HEAD)"
  mkdir -p .github/curbpack/cache
  printf '%s\n' '{"schema_version":"1","pack_id":"house-policy","readiness_score":100,"concurrency_control":{"expected_parent_commit_sha":"'"$head"'"},"failures":[]}' > .github/curbpack/cache/latest_failure.json
  if "$BIN" check >/dev/null 2>&1; then exit 1; fi
  printf '%s\n' '{"schema_version":"1","pack_id":"house-policy","readiness_score":100,"concurrency_control":{"expected_parent_commit_sha":"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"},"failures":[]}' > .github/curbpack/cache/latest_failure.json
  "$BIN" export --context-pack >/dev/null 2>&1
  grep -q '"ok": false' .github/curbpack/cache/context-pack.json
)
case20_code=$?
set -e
rm -rf "$TMP20"
[[ "$case20_code" -eq 0 ]] && ok "20 tampered cache fake green re-validates (check + export)" || \
  bad "20 hand-edited latest_failure.json must not false-green"

# --- 21) SECURITY.md symlink outside repo refused via pathjail ---
TMP21="$(mktemp -d)"
set +e
(
  set -e
  cd "$TMP21"
  git init -q && git config user.email "redteam@curbpack.local" && git config user.name "Redteam"
  git commit --allow-empty -m init -q
  "$BIN" init --packs house-policy >/dev/null
  rm -f SECURITY.md && mkdir -p /tmp/outside && echo "# outside" > /tmp/outside/real.md && ln -s /tmp/outside/real.md SECURITY.md
  out="$("$BIN" check 2>&1 || true)"
  echo "$out" | grep -qi 'path escapes repository root'
)
case21_code=$?
set -e
rm -rf "$TMP21"
[[ "$case21_code" -eq 0 ]] && ok "21 SECURITY.md symlink escape refused (pathjail)" || \
  bad "21 symlink escape must refuse via pathjail"

echo -e "\nredteam-pilot: $PASS passed, $FAIL failed"
exit $(( FAIL > 0 ))
