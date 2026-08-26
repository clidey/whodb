import { test } from 'node:test';
import assert from 'node:assert/strict';
import { coerceValue, hydrateRows } from '../dist/hydrate.js';

test('coerceValue handles every kind', () => {
  assert.equal(coerceValue('42', 'bigint'), 42);
  assert.equal(coerceValue('3.14', 'numeric'), 3.14);
  assert.equal(coerceValue('true', 'boolean'), true);
  assert.equal(coerceValue('f', 'boolean'), false);
  assert.ok(coerceValue('2026-01-05T00:00:00Z', 'timestamptz') instanceof Date);
  assert.deepEqual(coerceValue('{"a":1}', 'jsonb'), { a: 1 });
  assert.equal(coerceValue('hello', 'text'), 'hello');
  assert.equal(coerceValue(null, 'bigint'), null);
  assert.equal(coerceValue('not-a-number', 'bigint'), 'not-a-number');
});

test('ontology property dataType overrides column type', () => {
  const result = { columns: ['age'], rows: [['42']], total: 1 };
  const withMeta = hydrateRows(result, new Map([['age', 'Integer']]));
  assert.equal(withMeta.rows[0].age, 42);
  const withoutMeta = hydrateRows(result);
  assert.equal(withoutMeta.rows[0].age, '42'); // no type info → string
});

test('PascalCase RowsResult normalizes', () => {
  const result = { Columns: [{ Name: 'n', Type: 'int4' }], Rows: [['7']], TotalCount: 1, DisableUpdate: false };
  const { rows, totalCount } = hydrateRows(result);
  assert.equal(rows[0].n, 7);
  assert.equal(totalCount, 1);
});
