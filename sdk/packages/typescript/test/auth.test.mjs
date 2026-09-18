import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync, chmodSync, readFileSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { cliProvider } from '../dist/auth.js';
import { CliCredentialsError } from '../dist/errors.js';

/** Writes a fake whodb CLI script; appends to a counter file per invocation. */
function fakeCli(body) {
  const dir = mkdtempSync(join(tmpdir(), 'whodb-cli-test-'));
  const countFile = join(dir, 'count');
  const command = join(dir, 'whodb');
  writeFileSync(command, `#!/bin/sh\necho x >> "${countFile}"\n${body}\n`);
  chmodSync(command, 0o755);
  return { command, execCount: () => (existsSync(countFile) ? readFileSync(countFile, 'utf8').length / 2 : 0) };
}

test('missing CLI binary maps to CliCredentialsError', async () => {
  const provider = cliProvider(join(tmpdir(), 'no-such-whodb-binary'));
  await assert.rejects(() => provider.token(), CliCredentialsError);
});

test('invalid CLI JSON maps to CliCredentialsError', async () => {
  const { command } = fakeCli('echo "not json"');
  await assert.rejects(() => cliProvider(command).token(), CliCredentialsError);
});

test('CLI failure surfaces as CliCredentialsError', async () => {
  const { command } = fakeCli('echo "run: whodb login" >&2; exit 1');
  await assert.rejects(() => cliProvider(command).token(), CliCredentialsError);
});

test('fresh tokens are served from cache without re-exec', async () => {
  const expiry = new Date(Date.now() + 3600_000).toISOString();
  const { command, execCount } = fakeCli(
    `echo '{"access_token":"tok-1","expires_at":"${expiry}"}'`,
  );
  const provider = cliProvider(command);
  for (let i = 0; i < 3; i += 1) {
    assert.equal(await provider.token(), 'tok-1');
  }
  assert.equal(execCount(), 1);
});

test('near-expiry tokens re-exec the CLI', async () => {
  const expiry = new Date(Date.now() + 30_000).toISOString(); // inside the 60s skew
  const { command, execCount } = fakeCli(
    `echo '{"access_token":"tok-1","expires_at":"${expiry}"}'`,
  );
  const provider = cliProvider(command);
  await provider.token();
  await provider.token();
  assert.equal(execCount(), 2);
});

test('refresh drops the cache and re-execs', async () => {
  const expiry = new Date(Date.now() + 3600_000).toISOString();
  const { command, execCount } = fakeCli(
    `echo '{"access_token":"tok-1","expires_at":"${expiry}"}'`,
  );
  const provider = cliProvider(command);
  await provider.token();
  await provider.refresh();
  await provider.token();
  assert.equal(execCount(), 2);
});
