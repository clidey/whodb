import { test } from 'node:test';
import assert from 'node:assert/strict';
import { resolveConfig } from '../dist/config.js';

test('explicit apiKey wins over env and CLI', async () => {
  const resolved = resolveConfig({ apiKey: 'whodb_sk_ctor' }, { WHODB_API_KEY: 'whodb_sk_env' });
  assert.equal(await resolved.credentials.token(), 'whodb_sk_ctor');
  assert.equal(resolved.usingCliCredentials, false);
});

test('WHODB_API_KEY env wins over CLI fallback', async () => {
  const resolved = resolveConfig({}, { WHODB_API_KEY: 'whodb_sk_env' });
  assert.equal(await resolved.credentials.token(), 'whodb_sk_env');
  assert.equal(resolved.usingCliCredentials, false);
});

test('no credentials falls back to CLI provider', () => {
  const resolved = resolveConfig({}, {});
  assert.equal(resolved.usingCliCredentials, true);
});

test('token callback is honored', async () => {
  const resolved = resolveConfig({ token: async () => 'tok-123' }, {});
  assert.equal(await resolved.credentials.token(), 'tok-123');
});

test('workspace env vars apply when constructor args absent', () => {
  const resolved = resolveConfig({}, { WHODB_ORG: 'acme', WHODB_PROJECT: 'growth', WHODB_HOST: 'https://example.com', WHODB_API_KEY: 'k' });
  assert.equal(resolved.org, 'acme');
  assert.equal(resolved.project, 'growth');
  assert.equal(resolved.host, 'https://example.com');
});

test('constructor workspace args win over env', () => {
  const resolved = resolveConfig({ org: 'ctor-org' }, { WHODB_ORG: 'env-org', WHODB_API_KEY: 'k' });
  assert.equal(resolved.org, 'ctor-org');
});
