#!/usr/bin/env bash
# npm wrapper smoke — reusable from CI.
# Positive: pack + temp git repo + scan (Art 14 clock) + clean git status.
# Negative: tampered checksum → install refuses (fail-closed).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -x "$ROOT/bin/curbpack" ]]; then
  go build -o "$ROOT/bin/curbpack" ./cmd/curbpack
fi

echo "== npm-smoke: negative checksum tamper =="
node - <<'NODE'
const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');
const { verifyChecksum } = require('./npm/lib/install');

const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'curbpack-tamper-'));
const bin = path.join(dir, 'curbpack_linux_amd64');
fs.writeFileSync(bin, Buffer.from('not-a-real-binary'));
const checksums = 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef  curbpack_linux_amd64\n';
let threw = false;
try {
  verifyChecksum(bin, checksums, 'curbpack_linux_amd64');
} catch (e) {
  threw = true;
  assert.match(String(e.message), /checksum mismatch/);
}
assert(threw, 'tampered checksum must refuse install');
console.log('  PASS  checksum tamper refused');
NODE

echo "== npm-smoke: positive wrapper scan =="
PKG_TGZ="$(cd "$ROOT/npm" && npm pack --silent)"
PKG_PATH="$ROOT/npm/$PKG_TGZ"

SMOKE=$(mktemp -d)
cleanup() { rm -rf "$SMOKE"; }
trap cleanup EXIT

# Install wrapper outside the git repo under test (npm artifacts must not dirty scan tree).
mkdir -p "$SMOKE/npm-home" "$SMOKE/repo"
cd "$SMOKE/npm-home"
npm init -y >/dev/null 2>&1
npm install "$PKG_PATH" >/dev/null 2>&1

cd "$SMOKE/repo"
git init -q
git config user.email "npm-smoke@curbpack.local"
git config user.name "npm-smoke"
git commit --allow-empty -m init -q

export CURBPACK_BIN="$ROOT/bin/curbpack"
WRAPPER="$SMOKE/npm-home/node_modules/.bin/curbpack"
OUT="$("$WRAPPER" scan 2>&1)"
printf '%s\n' "$OUT"

printf '%s\n' "$OUT" | grep -q 'Art 14 reporting clock'
printf '%s\n' "$OUT" | grep -q 'Packs: cra-baseline'
printf '%s\n' "$OUT" | grep -q 'Read-only'

if [[ -n "$(git status --porcelain)" ]]; then
  echo "git status not clean after scan:" >&2
  git status --porcelain >&2
  exit 1
fi

echo "npm-smoke: OK"
