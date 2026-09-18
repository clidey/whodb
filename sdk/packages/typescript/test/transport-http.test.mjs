import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { HttpTransport } from '../dist/transport-http.js';
import { WhoDB } from '../dist/client.js';
import { AuthError, NotFoundError, ValidationError } from '../dist/errors.js';

/**
 * Scripted HTTP stub: replays canned responses in order and records the
 * headers and operation name of every request.
 */
function createStub() {
  const requests = [];
  let responses = [];
  const server = createServer((req, res) => {
    let raw = '';
    req.on('data', (chunk) => { raw += chunk; });
    req.on('end', () => {
      const body = JSON.parse(raw || '{}');
      const operation = (body.query ?? '').split(/[\s({]+/)[1] ?? '';
      requests.push({ headers: req.headers, operation });
      const respond = responses.shift() ?? (() => res.writeHead(500).end());
      respond(res);
    });
  });
  return {
    server,
    requests,
    script(...next) { responses = next; requests.length = 0; },
    async listen() {
      await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
      return `http://127.0.0.1:${server.address().port}`;
    },
  };
}

const json = (status, body) => (res) => {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
};

const stub = createStub();
let host;
before(async () => { host = await stub.listen(); });
after(() => stub.server.close());

const staticCredentials = (token) => ({ token: async () => token });

test('sends bearer, user-agent, and workspace headers', async () => {
  stub.script(json(200, { data: { Op: true } }));
  const transport = new HttpTransport({ host, credentials: staticCredentials('tok'), orgId: 'org-1', projectId: 'proj-1' });
  await transport.execute('Op', 'query Op { Op }', {});
  const headers = stub.requests[0].headers;
  assert.equal(headers.authorization, 'Bearer tok');
  assert.match(headers['user-agent'], /^clidey-whodb-ts\//);
  assert.equal(headers['x-whodb-org-id'], 'org-1');
  assert.equal(headers['x-whodb-project-id'], 'proj-1');
});

test('retries once on transient 5xx', async () => {
  stub.script(json(503, {}), json(200, { data: { Op: 42 } }));
  const transport = new HttpTransport({ host, credentials: staticCredentials('tok') });
  const data = await transport.execute('Op', 'query Op { Op }', {});
  assert.equal(data.Op, 42);
  assert.equal(stub.requests.length, 2);
});

test('refreshes credentials once on 401 and retries with the new token', async () => {
  stub.script(json(401, {}), json(200, { data: { Op: true } }));
  let refreshed = 0;
  const credentials = {
    token: async () => (refreshed > 0 ? 'fresh' : 'stale'),
    refresh: async () => { refreshed += 1; },
  };
  const transport = new HttpTransport({ host, credentials });
  await transport.execute('Op', 'query Op { Op }', {});
  assert.equal(refreshed, 1);
  assert.equal(stub.requests[1].headers.authorization, 'Bearer fresh');
});

test('persistent 401 maps to AuthError', async () => {
  stub.script(json(401, {}), json(401, {}));
  const transport = new HttpTransport({ host, credentials: { token: async () => 'bad', refresh: async () => {} } });
  await assert.rejects(
    () => transport.execute('Op', 'query Op { Op }', {}),
    AuthError,
  );
});

test('GraphQL errors map through the taxonomy over real HTTP', async () => {
  stub.script(json(200, { errors: [{ message: 'nope', extensions: { code: 'NOT_FOUND' } }] }));
  const transport = new HttpTransport({ host, credentials: staticCredentials('tok') });
  await assert.rejects(
    () => transport.execute('Op', 'query Op { Op }', {}),
    NotFoundError,
  );
});

test('client resolves org and project slugs to IDs', async () => {
  stub.script(
    json(200, { data: { MyOrganizations: [{ id: '11111111-1111-1111-1111-111111111111', slug: 'acme' }] } }),
    json(200, { data: { Projects: [{ id: '22222222-2222-2222-2222-222222222222', slug: 'analytics' }] } }),
    json(200, { data: { OntologyEntities: [] } }),
  );
  const client = new WhoDB({ token: 'tok', org: 'acme', project: 'analytics', host });
  await client.ontologyEntities();
  assert.deepEqual(stub.requests.map((r) => r.operation), ['MyOrganizations', 'Projects', 'OntologyEntities']);
  const final = stub.requests[2].headers;
  assert.equal(final['x-whodb-org-id'], '11111111-1111-1111-1111-111111111111');
  assert.equal(final['x-whodb-project-id'], '22222222-2222-2222-2222-222222222222');
});

test('API key without workspace discovers scope via MyWorkspace', async () => {
  stub.script(
    json(200, { data: { MyWorkspace: { orgId: '33333333-3333-3333-3333-333333333333', projectId: '44444444-4444-4444-4444-444444444444' } } }),
    json(200, { data: { OntologyEntities: [] } }),
  );
  const client = new WhoDB({ apiKey: 'whodb_sk_test', host });
  await client.ontologyEntities();
  assert.equal(stub.requests[0].operation, 'MyWorkspace');
  assert.equal(stub.requests[1].headers['x-whodb-project-id'], '44444444-4444-4444-4444-444444444444');
});

test('multi-grant API key without project raises ValidationError', async () => {
  stub.script(json(200, { data: { MyWorkspace: { orgId: '33333333-3333-3333-3333-333333333333', projectId: null } } }));
  const client = new WhoDB({ apiKey: 'whodb_sk_test', host });
  await assert.rejects(() => client.ontologyEntities(), ValidationError);
});
