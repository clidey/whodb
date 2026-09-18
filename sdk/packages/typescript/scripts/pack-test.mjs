#!/usr/bin/env node
// pack-test.mjs — consumption test for the BUILT ARTIFACT: packs the package
// exactly as `npm publish` would, installs it into a scratch project, and
// imports it from both module systems. Catches broken exports maps and
// missing files — the class of bug that shipped in 0.127.0 (require() of the
// published package failed despite green source-tree tests).
import { execFileSync } from 'node:child_process';
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const packageDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const scratch = mkdtempSync(join(tmpdir(), 'whodb-pack-test-'));

try {
  const tarball = execFileSync('npm', ['pack', '--pack-destination', scratch], {
    cwd: packageDir, encoding: 'utf8',
  }).trim().split('\n').pop();

  writeFileSync(join(scratch, 'package.json'), JSON.stringify({ name: 'pack-test', private: true }));
  execFileSync('npm', ['install', '--no-audit', '--no-fund', join(scratch, tarball)], {
    cwd: scratch, stdio: 'pipe',
  });

  const esmCheck = `
    import { WhoDB, OntologyHandle, IpcTransport, WhoDBError, ListCall } from '@clidey/whodb-sdk';
    if (typeof WhoDB !== 'function') throw new Error('WhoDB export missing');
    for (const symbol of [OntologyHandle, IpcTransport, WhoDBError, ListCall]) {
      if (typeof symbol !== 'function') throw new Error('named export missing');
    }
    console.log('esm ok');
  `;
  writeFileSync(join(scratch, 'check.mjs'), esmCheck);
  execFileSync('node', ['check.mjs'], { cwd: scratch, stdio: 'inherit' });

  const cjsCheck = `
    const sdk = require('@clidey/whodb-sdk');
    if (typeof sdk.WhoDB !== 'function') throw new Error('CJS require broken (0.127.0 bug class)');
    console.log('cjs ok');
  `;
  writeFileSync(join(scratch, 'check.cjs'), cjsCheck);
  execFileSync('node', ['check.cjs'], { cwd: scratch, stdio: 'inherit' });

  console.log('pack-test: PASS');
} finally {
  rmSync(scratch, { recursive: true, force: true });
}
