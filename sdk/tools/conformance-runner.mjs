#!/usr/bin/env node
// conformance-runner.mjs — runs the shared fixtures against a language's SDK
// via its conformance script (stdin/stdout JSON lines protocol, connparse
// style). The fixture format is documented in sdk/spec/fixtures/.
//
// Usage: node tools/conformance-runner.mjs --lang ts [--filter name]
import { readFileSync, readdirSync } from 'node:fs';
import { spawn } from 'node:child_process';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createInterface } from 'node:readline';

const sdkRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const fixturesDir = join(sdkRoot, 'spec/fixtures');

const RUNNERS = {
  ts: { command: 'node', args: [join(sdkRoot, 'packages/typescript/scripts/conformance.mjs')] },
  python: { command: 'python3', args: [join(sdkRoot, 'packages/python/scripts/conformance.py')] },
};

const args = process.argv.slice(2);
const lang = args[args.indexOf('--lang') + 1];
const filterIndex = args.indexOf('--filter');
const filter = filterIndex === -1 ? null : args[filterIndex + 1];

const runner = RUNNERS[lang];
if (!runner) {
  console.error(`usage: conformance-runner.mjs --lang <${Object.keys(RUNNERS).join('|')}>`);
  process.exit(2);
}

const fixtures = readdirSync(fixturesDir)
  .filter(f => f.endsWith('.json'))
  .flatMap(f => JSON.parse(readFileSync(join(fixturesDir, f), 'utf8')).fixtures)
  .filter(f => !filter || f.name.includes(filter));

const child = spawn(runner.command, runner.args, { stdio: ['pipe', 'pipe', 'inherit'] });
const lines = createInterface({ input: child.stdout });
const results = [];

const pending = new Promise((resolveDone) => {
  lines.on('line', (line) => {
    results.push(JSON.parse(line));
    if (results.length === fixtures.length) resolveDone();
  });
  child.on('exit', () => resolveDone());
});

for (const fixture of fixtures) {
  child.stdin.write(JSON.stringify(fixture) + '\n');
}
child.stdin.end();
await pending;

let failed = 0;
for (const result of results) {
  if (result.pass) {
    console.log(`PASS ${result.name}`);
  } else {
    failed += 1;
    console.error(`FAIL ${result.name}\n  ${result.reason}`);
  }
}
if (results.length !== fixtures.length) {
  console.error(`runner exited early: ${results.length}/${fixtures.length} fixtures reported`);
  failed += fixtures.length - results.length;
}
console.log(`\n${results.length - failed}/${fixtures.length} passed (${lang})`);
process.exit(failed > 0 ? 1 : 0);
