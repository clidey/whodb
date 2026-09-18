/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { IpcTransport } from '../dist/transport-ipc.js';
import { TransportCapabilityError } from '../dist/errors.js';

/** In-process IPC server stub recording requests and replaying responses. */
let server;
let address;
const received = [];
const responses = new Map();

before(async () => {
  server = createServer((req, res) => {
    const chunks = [];
    req.on('data', (chunk) => chunks.push(chunk));
    req.on('end', () => {
      received.push({
        path: req.url,
        headers: { jobId: req.headers['x-job-id'], authorization: req.headers.authorization },
        body: JSON.parse(Buffer.concat(chunks).toString('utf8') || '{}'),
      });
      const reply = responses.get(req.url) ?? {};
      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify(reply));
    });
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  address = `127.0.0.1:${server.address().port}`;
});

after(async () => {
  server.closeAllConnections();
  await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
});

const makeTransport = () => new IpcTransport({ address, jobId: 'job-1', token: 'tok-1' });

test('OntologyQuery maps whereJson and strips nulls', async () => {
  received.length = 0;
  responses.set('/query', { columns: ['id'], rows: [['1']], total: 1 });
  const transport = makeTransport();
  const data = await transport.execute('OntologyQuery', '', {
    projectId: 'p',
    input: { entity: 'User', whereJson: '{"age":{"gt":30}}', sort: null, pageSize: 5, offset: 0 },
  });
  assert.deepEqual(data.OntologyQuery, { columns: ['id'], rows: [['1']], total: 1 });
  assert.equal(received[0].path, '/query');
  assert.deepEqual(received[0].body.where, { age: { gt: 30 } });
  assert.equal(received[0].body.whereJson, undefined);
  assert.equal(received[0].body.sort, undefined);
  assert.equal(received[0].headers.jobId, 'job-1');
  assert.equal(received[0].headers.authorization, 'tok-1');
});

test('entity-ID operations resolve apiName via cached /entities', async () => {
  received.length = 0;
  responses.set('/entities', [{ id: 'ent-1', apiName: 'user', primaryKey: 'id' }]);
  responses.set('/create_many', ['u_1', 'u_2']);
  const transport = makeTransport();
  const data = await transport.execute('OntologyAddRows', '', {
    projectId: 'p',
    entityId: 'ent-1',
    rows: [{ values: [{ Key: 'email', Value: 'a@b.co' }] }, { values: [{ Key: 'email', Value: 'c@d.co' }] }],
    idempotencyKey: 'batch-1',
  });
  assert.deepEqual(data.OntologyAddRows, { inserted: 2, ids: ['u_1', 'u_2'] });
  const createRequest = received.find(r => r.path === '/create_many');
  assert.equal(createRequest.body.entity, 'user');
  assert.deepEqual(createRequest.body.rows, [{ email: 'a@b.co' }, { email: 'c@d.co' }]);
  assert.equal(createRequest.body.idempotencyKey, 'batch-1');

  // Second entity-addressed call must not re-fetch /entities.
  responses.set('/update', {});
  await transport.execute('OntologyUpdateRow', '', {
    projectId: 'p',
    entityId: 'ent-1',
    values: [{ Key: 'id', Value: 'u_1' }, { Key: 'email', Value: 'x@y.z' }],
    updatedColumns: ['email'],
  });
  const entityCalls = received.filter(r => r.path === '/entities');
  assert.equal(entityCalls.length, 1);
  const updateRequest = received.find(r => r.path === '/update');
  assert.equal(updateRequest.body.pk, 'u_1');
  assert.deepEqual(updateRequest.body.data, { email: 'x@y.z' });
});

test('behavior operations map entity IDs and action options to IPC endpoints', async () => {
  received.length = 0;
  responses.set('/entities', [{ id: 'ent-order', apiName: 'Order', primaryKey: 'id' }]);
  responses.set('/capabilities', { currentState: 'pending', recordVersion: 7, actions: [] });
  responses.set('/preview_action', { allowed: true, proposedChanges: { status: 'approved' } });
  responses.set('/action', { allowed: true, recordVersion: 8 });
  responses.set('/action_executions', [{ actionName: 'approve' }]);
  const transport = makeTransport();

  await transport.execute('OntologyRecordCapabilities', '', {
    projectId: 'p', ontologyId: 'ent-order', recordKey: 'order-1',
  });
  await transport.execute('PreviewOntologyAction', '', {
    input: { projectId: 'p', ontologyId: 'ent-order', action: 'approve', recordKey: 'order-1', values: { note: 'ok' } },
  });
  await transport.execute('ExecuteOntologyAction', '', {
    input: { projectId: 'p', ontologyId: 'ent-order', action: 'approve', recordKey: 'order-1', values: { note: 'ok' }, expectedVersion: 7, idempotencyKey: 'approve-1' },
  });
  await transport.execute('OntologyActionExecutions', '', {
    projectId: 'p', ontologyId: 'ent-order', recordKey: 'order-1', limit: 20,
  });

  assert.deepEqual(received.find(r => r.path === '/capabilities').body, { entity: 'Order', recordKey: 'order-1' });
  assert.deepEqual(received.find(r => r.path === '/preview_action').body, {
    entity: 'Order', action: 'approve', recordKey: 'order-1', values: { note: 'ok' },
  });
  assert.deepEqual(received.find(r => r.path === '/action').body, {
    entity: 'Order', action: 'approve', recordKey: 'order-1', values: { note: 'ok' }, expectedVersion: 7, idempotencyKey: 'approve-1',
  });
  assert.deepEqual(received.find(r => r.path === '/action_executions').body, {
    entity: 'Order', recordKey: 'order-1', limit: 20,
  });
});

test('unmapped operations throw TransportCapabilityError', async () => {
  const transport = makeTransport();
  await assert.rejects(
    () => transport.execute('QueryDataset', '', {}),
    TransportCapabilityError,
  );
});
