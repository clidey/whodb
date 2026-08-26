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

after(() => server.close());

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

test('unmapped operations throw TransportCapabilityError', async () => {
  const transport = makeTransport();
  await assert.rejects(
    () => transport.execute('QueryDataset', '', {}),
    TransportCapabilityError,
  );
});
