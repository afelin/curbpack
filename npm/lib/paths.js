'use strict';

const os = require('os');
const path = require('path');

/** @returns {string} Install/cache directory for the native binary. */
function getInstallDir() {
  if (process.env.CURBPACK_INSTALL_DIR) {
    return path.resolve(process.env.CURBPACK_INSTALL_DIR);
  }
  if (process.platform === 'win32') {
    const base =
      process.env.LOCALAPPDATA ||
      path.join(os.homedir(), 'AppData', 'Local');
    return path.join(base, 'curbpack');
  }
  return path.join(os.homedir(), '.curbpack', 'bin');
}

/** @returns {string} Path to the curbpack native executable. */
function getBinaryPath() {
  const name = process.platform === 'win32' ? 'curbpack.exe' : 'curbpack';
  return path.join(getInstallDir(), name);
}

/** @returns {string} Marker file path (install provenance). */
function getMarkerPath() {
  if (process.platform === 'win32') {
    return path.join(getInstallDir(), 'install-marker.json');
  }
  const dataHome =
    process.env.XDG_DATA_HOME ||
    path.join(os.homedir(), '.local', 'share');
  return path.join(dataHome, 'curbpack', 'install-marker.json');
}

module.exports = { getInstallDir, getBinaryPath, getMarkerPath };
