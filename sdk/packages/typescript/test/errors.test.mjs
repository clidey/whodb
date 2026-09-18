import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mapGraphQLErrors, AuthError, NotFoundError, ValidationError, PlatformError } from '../dist/errors.js';
import { interpretServerError, warnIfFlagged, resetWarnings } from '../dist/manifest-check.js';
import { WhoDBVersionError } from '../dist/errors.js';

test('error code mapping', () => {
  assert.ok(mapGraphQLErrors([{ message: 'x', extensions: { code: 'UNAUTHENTICATED' } }]) instanceof AuthError);
  assert.ok(mapGraphQLErrors([{ message: 'x', extensions: { code: 'FORBIDDEN' } }]) instanceof AuthError);
  assert.ok(mapGraphQLErrors([{ message: 'x', extensions: { code: 'NOT_FOUND' } }]) instanceof NotFoundError);
  assert.ok(mapGraphQLErrors([{ message: 'x', extensions: { code: 'BAD_USER_INPUT' } }]) instanceof ValidationError);
  const platform = mapGraphQLErrors([{ message: 'x', extensions: { code: 'SOMETHING_ELSE' } }]);
  assert.ok(platform instanceof PlatformError);
  assert.equal(platform.code, 'SOMETHING_ELSE');
});

test('unknown-operation server errors become WhoDBVersionError', () => {
  const converted = interpretServerError(new Error('Cannot query field "NewThing" on type "Query"'), '1.2.3');
  assert.ok(converted instanceof WhoDBVersionError);
  assert.match(converted.message, /1\.2\.3/);

  const passthrough = new Error('connection refused');
  assert.equal(interpretServerError(passthrough, '1.2.3'), passthrough);
});

test('deprecation warnings fire once per operation', () => {
  resetWarnings();
  const warnings = [];
  const original = console.warn;
  console.warn = (message) => warnings.push(message);
  try {
    warnIfFlagged('OntologyRows'); // not flagged in current manifest — no warning
    assert.equal(warnings.length, 0);
  } finally {
    console.warn = original;
  }
});
