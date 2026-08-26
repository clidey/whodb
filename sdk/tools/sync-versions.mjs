#!/usr/bin/env node
// sync-versions.mjs — keeps all SDK package versions in lockstep with the
// repo release version (SDK_DESIGN.md §2.3).
//
// Usage:
//   node tools/sync-versions.mjs --set 0.66.0   # stamp all manifests
//   node tools/sync-versions.mjs --check        # verify all match
import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const sdkRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');

const manifests = [
  {
    path: join(sdkRoot, 'packages/typescript/package.json'),
    read: (text) => JSON.parse(text).version,
    write: (text, version) => {
      const pkg = JSON.parse(text);
      pkg.version = version;
      return JSON.stringify(pkg, null, 2) + '\n';
    },
  },
  {
    path: join(sdkRoot, 'packages/python/pyproject.toml'),
    read: (text) => text.match(/^version\s*=\s*"([^"]+)"/m)?.[1],
    write: (text, version) => text.replace(/^version\s*=\s*"[^"]+"/m, `version = "${version}"`),
  },
];

const args = process.argv.slice(2);
const setIndex = args.indexOf('--set');
const check = args.includes('--check');

const present = manifests.filter(m => existsSync(m.path));
if (present.length === 0) {
  console.error('sync-versions: no package manifests found');
  process.exit(1);
}

if (setIndex !== -1) {
  const version = args[setIndex + 1];
  if (!/^\d+\.\d+\.\d+(-[\w.]+)?$/.test(version ?? '')) {
    console.error(`sync-versions: invalid version "${version}"`);
    process.exit(1);
  }
  for (const manifest of present) {
    writeFileSync(manifest.path, manifest.write(readFileSync(manifest.path, 'utf8'), version));
    console.log(`stamped ${version} into ${manifest.path}`);
  }
} else if (check) {
  const versions = present.map(m => ({ path: m.path, version: m.read(readFileSync(m.path, 'utf8')) }));
  const unique = new Set(versions.map(v => v.version));
  if (unique.size !== 1 || versions.some(v => !v.version)) {
    console.error('sync-versions: version mismatch across SDK packages:');
    for (const v of versions) console.error(`  ${v.path}: ${v.version ?? 'MISSING'}`);
    process.exit(1);
  }
  console.log(`all SDK packages at ${versions[0].version}`);
} else {
  console.error('usage: sync-versions.mjs --set <version> | --check');
  process.exit(1);
}
