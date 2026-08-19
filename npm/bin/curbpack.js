#!/usr/bin/env node
'use strict';

const { spawnSync } = require('child_process');

const { ensureInstalled } = require('../lib/install');

async function main() {
  let binary;
  try {
    binary = await ensureInstalled();
  } catch (err) {
    process.stderr.write(
      `curbpack: install failed — ${err.message || err}\n`,
    );
    process.stderr.write(
      'Build from source: go install github.com/afelin/curbpack/cmd/curbpack@latest\n',
    );
    process.exit(2);
  }

  const result = spawnSync(binary, process.argv.slice(2), {
    stdio: 'inherit',
    env: process.env,
  });

  if (result.error) {
    process.stderr.write(`curbpack: ${result.error.message}\n`);
    process.exit(2);
  }
  process.exit(result.status === null ? 1 : result.status);
}

main();
