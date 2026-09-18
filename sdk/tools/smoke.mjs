#!/usr/bin/env node
// smoke.mjs — pre-release smoke test: runs the built TypeScript SDK against a
// REAL platform (staging) with a REAL API key, exercising the full chain the
// fixtures can't: auth, workspace discovery, ontology read + write round-trip.
//
// Usage:
//   WHODB_SMOKE_HOST=https://staging.whodb.com WHODB_API_KEY=whodb_sk_... \
//   WHODB_SMOKE_ENTITY=<writable entity apiName> node tools/smoke.mjs
//
// Read-only mode (no entity given): auth + discovery + list only.
import { WhoDB } from '../packages/typescript/dist/index.js';

const host = process.env.WHODB_SMOKE_HOST;
const apiKey = process.env.WHODB_API_KEY;
const entityName = process.env.WHODB_SMOKE_ENTITY;

if (!host || !apiKey) {
  console.error('smoke: WHODB_SMOKE_HOST and WHODB_API_KEY are required');
  process.exit(2);
}

const failures = [];
const check = (name, ok, detail = '') => {
  console.log(`${ok ? 'PASS' : 'FAIL'} ${name}${detail ? ` — ${detail}` : ''}`);
  if (!ok) failures.push(name);
};

const whodb = new WhoDB({ apiKey, host });

// 1. Auth + key-derived workspace discovery
const entities = await whodb.ontologyEntities();
check('auth + ontologyEntities', Array.isArray(entities), `${entities.length} entities`);

// 2. List a real entity (first one, or the configured writable one)
const target = entityName ?? entities[0]?.apiName;
if (target) {
  const handle = whodb.ontology(target);
  const rows = await handle.list({ pageSize: 3 });
  check(`list ${target}`, Array.isArray(rows), `${rows.length} rows`);
  const description = await handle.describe();
  check(`describe ${target}`, description != null);
} else {
  check('list (skipped — no entities in project)', true);
}

// 3. Write round-trip, only when a writable entity is configured
if (entityName) {
  const handle = whodb.ontology(entityName);
  const meta = await handle.entityMeta();
  const marker = `smoke-${Date.now()}`;
  const stringProp = meta.properties?.find(p => p.dataType === 'String' && p.apiName !== meta.primaryKey);
  if (!stringProp) {
    check('write round-trip (skipped — no writable string property)', true);
  } else {
    const created = await handle.createMany([{ [stringProp.apiName]: marker }], { idempotencyKey: marker });
    check('createMany', (created.inserted ?? 0) === 1);
    const pk = created.ids?.[0];
    if (pk != null) {
      const fetched = await handle.get(pk);
      check('get after create', fetched?.[stringProp.apiName] === marker);
      await handle.update(pk, { [stringProp.apiName]: `${marker}-updated` });
      const updated = await handle.get(pk);
      check('update', updated?.[stringProp.apiName] === `${marker}-updated`);
      await handle.delete(pk);
      const gone = await handle.get(pk);
      check('delete', gone === null);
    } else {
      check('write round-trip (no id returned)', false);
    }
  }
}

console.log(`\n${failures.length === 0 ? 'SMOKE OK' : `SMOKE FAILED: ${failures.join(', ')}`}`);
process.exit(failures.length === 0 ? 0 : 1);
