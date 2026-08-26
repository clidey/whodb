#!/usr/bin/env node
// conformance.mjs — TypeScript SDK fixture runner. Reads fixtures as JSON
// lines on stdin, executes each against the built SDK with a mock transport,
// and writes one JSON result line per fixture to stdout.
import { createInterface } from 'node:readline';
import { WhoDB } from '../dist/index.js';

/** Mock transport that replays a fixture's scripted transcript. */
class MockTransport {
  constructor(transcript) {
    this.transcript = [...transcript];
    this.callIndex = 0;
  }

  async execute(operationName, _document, variables) {
    const step = this.transcript[this.callIndex];
    if (!step) throw new Error(`unexpected call #${this.callIndex + 1}: ${operationName}`);
    this.callIndex += 1;
    const expect = step.expectRequest ?? {};
    if (expect.operation && expect.operation !== operationName) {
      throw new Error(`expected operation ${expect.operation}, got ${operationName}`);
    }
    for (const [key, value] of Object.entries(expect.variables ?? {})) {
      assertDeepContains(variables[key], value, `variables.${key}`);
    }
    for (const [key, value] of Object.entries(expect.variablesContain ?? {})) {
      assertDeepContains(variables[key], value, `variables.${key}`);
    }
    for (const [key, value] of Object.entries(expect.inputContains ?? {})) {
      assertDeepContains(variables.input?.[key], value, `input.${key}`);
    }
    const response = step.response;
    if (response.errors) {
      const { mapGraphQLErrors } = await import('../dist/errors.js');
      throw mapGraphQLErrors(response.errors);
    }
    return response.data;
  }
}

function assertDeepContains(actual, expected, path) {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`mismatch at ${path}: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
}

/** Serializes result values for comparison; Dates become "@date:<iso>". */
function canonical(value) {
  if (value instanceof Date) return `@date:${value.toISOString().replace('.000Z', 'Z')}`;
  if (Array.isArray(value)) return value.map(canonical);
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, canonical(v)]));
  }
  return value;
}

async function runFixture(fixture) {
  const transport = new MockTransport(fixture.transcript);
  const client = new WhoDB({ apiKey: 'test', org: '00000000-0000-0000-0000-000000000001', project: 'proj-1' }, transport);
  // proj-1 is not a UUID, but with a custom transport the workspace resolver
  // would try MyOrganizations/Projects lookups — fixtures use a UUID org and
  // treat "proj-1" as an ID by overriding below.
  try {
    const handle = client[fixture.call.domain](fixture.call.handle);
    const method = handle[fixture.call.method].bind(handle);
    let result = method(...fixture.call.args);
    if (fixture.call.collect === 'pages') {
      const pages = [];
      for await (const page of result.pages()) pages.push(page.rows);
      result = pages;
    } else {
      result = await result;
    }
    if (fixture.expectError) {
      return { name: fixture.name, pass: false, reason: `expected ${fixture.expectError.type}, got success` };
    }
    const got = canonical(result ?? null);
    const want = fixture.expectResult ?? null;
    if (JSON.stringify(got) !== JSON.stringify(want)) {
      return { name: fixture.name, pass: false, reason: `result mismatch:\n  want ${JSON.stringify(want)}\n  got  ${JSON.stringify(got)}` };
    }
    return { name: fixture.name, pass: true };
  } catch (error) {
    if (fixture.expectError) {
      const typeOk = error.constructor.name === fixture.expectError.type;
      const messageOk = !fixture.expectError.messageContains || String(error.message).includes(fixture.expectError.messageContains);
      if (typeOk && messageOk) return { name: fixture.name, pass: true };
      return { name: fixture.name, pass: false, reason: `expected ${fixture.expectError.type}(${fixture.expectError.messageContains ?? ''}), got ${error.constructor.name}: ${error.message}` };
    }
    return { name: fixture.name, pass: false, reason: `${error.constructor.name}: ${error.message}` };
  }
}

const lines = createInterface({ input: process.stdin });
for await (const line of lines) {
  if (!line.trim()) continue;
  const result = await runFixture(JSON.parse(line));
  process.stdout.write(JSON.stringify(result) + '\n');
}
