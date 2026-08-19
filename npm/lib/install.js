'use strict';

const crypto = require('crypto');
const fs = require('fs');
const fsp = require('fs/promises');
const http = require('http');
const https = require('https');
const path = require('path');

const { getBinaryPath, getInstallDir, getMarkerPath } = require('./paths');

const MANIFEST = require('../install-manifest.json');
const REPO = process.env.CURBPACK_REPO || 'afelin/curbpack';
const CLAIM =
  MANIFEST.claim ||
  'Prepares evidence for human review — not a conformity assessment.';

function defaultVersion() {
  return process.env.CURBPACK_VERSION || MANIFEST.default_version || 'v0.5.2';
}

function resolvePlatform() {
  const platform = process.platform;
  let os;
  switch (platform) {
    case 'darwin':
      os = 'darwin';
      break;
    case 'linux':
      os = 'linux';
      break;
    case 'win32':
      os = 'windows';
      break;
    default:
      throw new Error(`unsupported OS: ${platform} (need darwin, linux, or win32)`);
  }

  const machine = process.arch;
  let arch;
  switch (machine) {
    case 'x64':
      arch = 'amd64';
      break;
    case 'arm64':
      arch = 'arm64';
      break;
    default:
      throw new Error(`unsupported arch: ${machine} (need x64 or arm64)`);
  }

  const asset =
    os === 'windows'
      ? `curbpack_${os}_${arch}.exe`
      : `curbpack_${os}_${arch}`;
  return { os, arch, asset };
}

function fetchBuffer(url, headers = {}) {
  return new Promise((resolve, reject) => {
    const lib = url.startsWith('https:') ? https : http;
    const req = lib.get(
      url,
      {
        headers: {
          Accept: 'application/vnd.github+json',
          'User-Agent': 'curbpack-npm-wrapper',
          ...headers,
        },
      },
      (res) => {
        if (
          res.statusCode &&
          res.statusCode >= 300 &&
          res.statusCode < 400 &&
          res.headers.location
        ) {
          fetchBuffer(res.headers.location, headers).then(resolve, reject);
          return;
        }
        if (res.statusCode !== 200) {
          reject(
            new Error(`HTTP ${res.statusCode} fetching ${url}`),
          );
          res.resume();
          return;
        }
        const chunks = [];
        res.on('data', (c) => chunks.push(c));
        res.on('end', () => resolve(Buffer.concat(chunks)));
        res.on('error', reject);
      },
    );
    req.on('error', reject);
    req.setTimeout(120_000, () => {
      req.destroy(new Error(`timeout fetching ${url}`));
    });
  });
}

async function fetchText(url) {
  const buf = await fetchBuffer(url, authHeaders());
  return buf.toString('utf8');
}

function authHeaders() {
  const token = process.env.GITHUB_TOKEN;
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

async function resolveReleaseUrls(version, asset) {
  const api = `https://api.github.com/repos/${REPO}/releases`;
  if (version === 'latest') {
    const json = JSON.parse(await fetchText(`${api}/latest`));
    const tag = json.tag_name;
    const assetEntry = (json.assets || []).find((a) => a.name === asset);
    const checksumsEntry = (json.assets || []).find(
      (a) => a.name === 'checksums.txt',
    );
    if (!assetEntry?.browser_download_url) {
      throw new Error(`could not resolve download URL for ${asset} (latest)`);
    }
    if (!checksumsEntry?.browser_download_url) {
      throw new Error('checksums.txt URL missing — refusing install (fail closed)');
    }
    return {
      tag,
      url: assetEntry.browser_download_url,
      checksumsUrl: checksumsEntry.browser_download_url,
    };
  }

  const tag = version;
  return {
    tag,
    url: `https://github.com/${REPO}/releases/download/${tag}/${asset}`,
    checksumsUrl: `https://github.com/${REPO}/releases/download/${tag}/checksums.txt`,
  };
}

function parseExpectedChecksum(checksumsText, asset) {
  const lines = checksumsText.split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const parts = trimmed.split(/\s+/);
    if (parts.length < 2) continue;
    const name = parts[parts.length - 1];
    if (name === asset) {
      return parts[0].toLowerCase();
    }
  }
  return null;
}

function sha256File(filePath) {
  const hash = crypto.createHash('sha256');
  hash.update(fs.readFileSync(filePath));
  return hash.digest('hex');
}

/**
 * Verify binary against checksums.txt content. Throws on mismatch (fail-closed).
 * @param {string} binaryPath
 * @param {string} checksumsText
 * @param {string} asset
 */
function verifyChecksum(binaryPath, checksumsText, asset) {
  const expected = parseExpectedChecksum(checksumsText, asset);
  if (!expected) {
    throw new Error(
      `no checksum entry for ${asset} in checksums.txt — refusing install`,
    );
  }
  const actual = sha256File(binaryPath);
  if (actual !== expected) {
    throw new Error(
      `checksum mismatch for ${asset}\n  expected: ${expected}\n  actual:   ${actual}`,
    );
  }
  return actual;
}

async function atomicWriteBinary(dest, buffer) {
  const dir = path.dirname(dest);
  await fsp.mkdir(dir, { recursive: true });
  const tmp = `${dest}.new`;
  await fsp.writeFile(tmp, buffer);
  if (process.platform !== 'win32') {
    await fsp.chmod(tmp, 0o755);
  }
  try {
    await fsp.rename(tmp, dest);
  } catch (err) {
    await fsp.unlink(tmp).catch(() => {});
    throw err;
  }
}

async function writeMarker(tag, binaryPath) {
  const marker = {
    schema: 'curbpack-install-marker:1',
    version: tag,
    install_dir: getInstallDir(),
    binary: binaryPath,
    installed_at: new Date().toISOString(),
    goos: resolvePlatform().os,
    source: 'npm-wrapper',
  };
  const markerPath = getMarkerPath();
  await fsp.mkdir(path.dirname(markerPath), { recursive: true });
  await fsp.writeFile(markerPath, `${JSON.stringify(marker, null, 2)}\n`, 'utf8');
}

function isExecutable(filePath) {
  try {
    fs.accessSync(filePath, fs.constants.X_OK);
    return true;
  } catch {
    return process.platform === 'win32' && fs.existsSync(filePath);
  }
}

/**
 * Ensure native binary is installed. Lazy download on first exec.
 * @returns {string} Path to executable.
 */
async function ensureInstalled() {
  if (process.env.CURBPACK_BIN) {
    const bin = path.resolve(process.env.CURBPACK_BIN);
    if (!fs.existsSync(bin)) {
      throw new Error(`CURBPACK_BIN not found: ${bin}`);
    }
    return bin;
  }

  const binaryPath = getBinaryPath();
  if (isExecutable(binaryPath)) {
    return binaryPath;
  }

  const version = defaultVersion();
  const { asset, os } = resolvePlatform();
  const { tag, url, checksumsUrl } = await resolveReleaseUrls(version, asset);

  process.stderr.write(
    `Curbpack npm wrapper — downloading ${tag} → ${asset}\n`,
  );
  process.stderr.write(`  ${CLAIM}\n`);

  const [binaryBuf, checksumsText] = await Promise.all([
    fetchBuffer(url, authHeaders()),
    fetchText(checksumsUrl),
  ]);

  const tmpDir = await fsp.mkdtemp(path.join(require('os').tmpdir(), 'curbpack-'));
  const tmpBinary = path.join(tmpDir, asset);
  try {
    await fsp.writeFile(tmpBinary, binaryBuf);
    if (process.platform !== 'win32') {
      await fsp.chmod(tmpBinary, 0o755);
    }
    verifyChecksum(tmpBinary, checksumsText, asset);
    await atomicWriteBinary(binaryPath, binaryBuf);
    await writeMarker(tag, binaryPath);

    if (process.platform !== 'win32' && os !== 'windows') {
      const alias = path.join(getInstallDir(), 'curb');
      try {
        await fsp.unlink(alias).catch(() => {});
        await fsp.symlink('curbpack', alias);
      } catch {
        /* alias optional */
      }
    }

    process.stderr.write(`Installed: ${binaryPath}\n`);
    return binaryPath;
  } finally {
    await fsp.rm(tmpDir, { recursive: true, force: true }).catch(() => {});
  }
}

module.exports = {
  ensureInstalled,
  verifyChecksum,
  parseExpectedChecksum,
  resolvePlatform,
  defaultVersion,
  CLAIM,
};
