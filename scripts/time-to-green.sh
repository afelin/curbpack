#!/usr/bin/env bash
# Maintainer activation bar: wall-clock time-to-green for Ladder A paths.
# Not a trust-surface — local/CI smoke only. Fail if >600s or non-zero exit.
#
# Paths:
#   (A) demo from built/installed binary
#   (B) temp git repo → init → check --heal → green
#
# Usage:
#   ./scripts/time-to-green.sh
#   CYBERREADY_BIN=/path/to/cyberready ./scripts/time-to-green.sh
#   TTG_MAX_SECONDS=60 ./scripts/time-to-green.sh   # tighter local bar
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MAX_SECONDS="${TTG_MAX_SECONDS:-600}"
BIN="${CYBERREADY_BIN:-}"

resolve_bin() {
  if [[ -n "$BIN" ]]; then
    if [[ ! -x "$BIN" ]]; then
      echo "time-to-green: CYBERREADY_BIN not executable: $BIN" >&2
      exit 1
    fi
    return
  fi
  # Prefer a freshly built workspace binary so TTG measures current tree.
  mkdir -p "$ROOT/bin"
  go build -o "$ROOT/bin/cyberready" ./cmd/cyberready
  BIN="$ROOT/bin/cyberready"
}

elapsed() {
  local start="$1"
  local end
  end="$(date +%s)"
  echo $((end - start))
}

echo "== time-to-green (activation bar; max ${MAX_SECONDS}s) =="
START_ALL="$(date +%s)"
resolve_bin
echo "binary: $BIN"

# --- (A) demo ---
START_A="$(date +%s)"
DEMO_DIR="$(mktemp -d)"
trap 'rm -rf "$DEMO_DIR" "$REPO_DIR"' EXIT
"$BIN" demo --out "$DEMO_DIR" --keep >/dev/null
test -f "$DEMO_DIR/review-pack/buyer-onepager.html"
SEC_A="$(elapsed "$START_A")"
echo "  PASS  A demo green in ${SEC_A}s"

# --- (B) init → check --heal ---
START_B="$(date +%s)"
REPO_DIR="$(mktemp -d)"
(
  set -euo pipefail
  cd "$REPO_DIR"
  git init -q
  git config user.email "ttg@cyberready.local"
  git config user.name "TTG"
  git commit --allow-empty -m init -q
  echo '# product' > README.md
  git add README.md
  git -c commit.gpgsign=false commit --no-verify -m scaffold -q
  "$BIN" init >/dev/null
  "$BIN" check --heal >/dev/null
)
SEC_B="$(elapsed "$START_B")"
echo "  PASS  B init→check --heal green in ${SEC_B}s"

TOTAL="$(elapsed "$START_ALL")"
echo "time-to-green: ${TOTAL}s (A=${SEC_A}s B=${SEC_B}s)"

if (( TOTAL > MAX_SECONDS )); then
  echo "time-to-green: FAIL — ${TOTAL}s exceeds ${MAX_SECONDS}s bar" >&2
  exit 1
fi

echo "time-to-green: OK"
