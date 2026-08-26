#!/usr/bin/env node
// generate-core.mjs — renders per-language SDK wire cores from the committed
// spec snapshot (sdk/spec/). See ee SDK_DESIGN.md §2.2 for the pipeline.
//
// Usage:
//   node tools/generate-core.mjs           # write generated files
//   node tools/generate-core.mjs --check   # verify committed output matches
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse, buildASTSchema, extendSchema, Kind, isObjectType, isEnumType, isScalarType, isNonNullType, isListType, getNamedType } from 'graphql';
import YAML from 'yaml';
import { renderTypeScript } from './render/ts.mjs';
import { renderPython } from './render/python.mjs';

const here = dirname(fileURLToPath(import.meta.url));
const sdkRoot = resolve(here, '..');
const specDir = join(sdkRoot, 'spec');

const SELECTION_DEPTH = 4;

function loadSchema() {
  const sdl = readFileSync(join(specDir, 'platform-schema.graphql'), 'utf8');
  const document = parse(sdl);
  const baseDefs = document.definitions.filter(d => !d.kind.endsWith('TypeExtension'));
  const extensions = document.definitions.filter(d => d.kind.endsWith('TypeExtension'));
  let schema = buildASTSchema({ kind: Kind.DOCUMENT, definitions: baseDefs }, { assumeValid: true });
  if (extensions.length > 0) {
    schema = extendSchema(schema, { kind: Kind.DOCUMENT, definitions: extensions }, { assumeValid: true });
  }
  return schema;
}

/** Derives a selection set string for a GraphQL output type, expanding object
 * fields to a bounded depth with a cycle guard. */
function deriveSelection(schema, typeName, override, depth = SELECTION_DEPTH, visiting = new Set()) {
  if (override) return override;
  const type = schema.getType(typeName);
  if (!type || isScalarType(type) || isEnumType(type)) return '';
  if (!isObjectType(type)) return '';
  if (visiting.has(typeName) || depth === 0) return null; // cycle/depth exhausted
  visiting.add(typeName);
  const parts = [];
  for (const [fieldName, field] of Object.entries(type.getFields())) {
    if (field.args.length > 0) continue; // parameterized fields are never auto-selected
    const named = getNamedType(field.type);
    if (isScalarType(named) || isEnumType(named)) {
      parts.push(fieldName);
      continue;
    }
    const nested = deriveSelection(schema, named.name, null, depth - 1, visiting);
    if (nested) parts.push(`${fieldName} ${nested}`);
    // null nested = cycle or depth limit: drop the field rather than emit invalid GraphQL
  }
  visiting.delete(typeName);
  return parts.length > 0 ? `{ ${parts.join(' ')} }` : null;
}

/** Renders a manifest field type ("ID", required, list) as GraphQL SDL syntax. */
function gqlTypeOf(arg) {
  let t = arg.type;
  if (arg.list) t = `[${t}!]`;
  if (arg.required) t = `${t}!`;
  return t;
}

/** Collects the closure of named input/output types reachable from the surface ops. */
function collectTypeClosure(schema, rootTypeNames) {
  const seen = new Set();
  const queue = [...rootTypeNames];
  while (queue.length > 0) {
    const name = queue.shift();
    if (seen.has(name)) continue;
    const type = schema.getType(name);
    if (!type || isScalarType(type) || isEnumType(type)) {
      if (type && isEnumType(type)) seen.add(name);
      continue;
    }
    seen.add(name);
    const fields = type.getFields?.();
    if (!fields) continue;
    for (const field of Object.values(fields)) {
      queue.push(getNamedType(field.type).name);
      for (const fieldArg of field.args ?? []) {
        queue.push(getNamedType(fieldArg.type).name);
      }
    }
  }
  return seen;
}

export function buildIR() {
  const schema = loadSchema();
  const manifest = JSON.parse(readFileSync(join(specDir, 'platform-manifest.json'), 'utf8'));
  const surface = YAML.parse(readFileSync(join(specDir, 'surface.yaml'), 'utf8'));
  const manifestOps = new Map(manifest.operations.map(op => [op.name, op]));
  const excluded = new Set(surface.excluded ?? []);
  const selections = surface.selections ?? {};

  const operations = [];
  const rootTypes = new Set();

  for (const [domainName, domain] of Object.entries(surface.domains)) {
    for (const [methodName, method] of Object.entries(domain.methods)) {
      const op = manifestOps.get(method.operation);
      if (op && !Array.isArray(op.args)) op.args = [];
      if (!op) {
        throw new Error(`surface.yaml: ${domainName}.${methodName} references operation "${method.operation}" not present in platform-manifest.json — re-export the spec or fix the mapping`);
      }
      if (excluded.has(method.operation)) {
        throw new Error(`surface.yaml: ${domainName}.${methodName} references excluded operation "${method.operation}"`);
      }
      const rootField = (op.kind === 'Mutation' ? schema.getMutationType() : schema.getQueryType())?.getFields()[op.name];
      if (!rootField) {
        throw new Error(`operation ${op.name} is in the manifest but missing from the schema — spec files out of sync`);
      }
      const returnTypeName = getNamedType(rootField.type).name;
      const selection = deriveSelection(schema, returnTypeName, selections[op.name]) ?? '';
      const varDefs = op.args.map(a => `$${a.name}: ${gqlTypeOf(a)}`).join(', ');
      const callArgs = op.args.map(a => `${a.name}: $${a.name}`).join(', ');
      const document = `${op.kind.toLowerCase()} ${op.name}${varDefs ? `(${varDefs})` : ''} { ${op.name}${callArgs ? `(${callArgs})` : ''} ${selection} }`
        .replace(/\s+/g, ' ').trim();

      rootTypes.add(returnTypeName);
      for (const arg of op.args) rootTypes.add(arg.type);

      operations.push({
        domain: domainName,
        method: methodName,
        internal: domain.internal === true,
        name: op.name,
        kind: op.kind,
        args: op.args,
        returns: op.returns,
        returnTypeName,
        document,
        autofill: method.autofill ?? {},
        rename: method.rename ?? {},
        paginated: method.paginated ?? null,
        deprecated: op.deprecated ?? false,
        sunsetAt: op.sunsetAt ?? null,
        behaviorChanged: op.behaviorChanged ?? false,
        note: op.note ?? null,
      });
    }
  }

  const typeClosure = collectTypeClosure(schema, [...rootTypes]);
  const manifestHash = createHash('sha256')
    .update(readFileSync(join(specDir, 'platform-manifest.json')))
    .digest('hex');
  const hydrationRules = JSON.parse(readFileSync(join(here, 'hydration-rules.json'), 'utf8'));

  return { schema, operations, typeClosure, manifestHash, hydrationRules, protocolVersion: manifest.manifestProtocolVersion };
}

function main() {
  const check = process.argv.includes('--check');
  const ir = buildIR();

  const renderers = [
    { name: 'typescript', outDir: join(sdkRoot, 'packages/typescript/src/generated'), render: renderTypeScript },
    { name: 'python', outDir: join(sdkRoot, 'packages/python/src/whodb/_generated'), render: renderPython },
  ];

  let failed = false;
  for (const renderer of renderers) {
    const renderFn = renderer.render;
    const files = renderFn(ir); // [ [relPath, content], ... ]
    for (const [relPath, content] of files) {
      const outPath = join(renderer.outDir, relPath);
      if (check) {
        const existing = existsSync(outPath) ? readFileSync(outPath, 'utf8') : null;
        if (existing !== content) {
          console.error(`MISMATCH: ${outPath} — run: node tools/generate-core.mjs`);
          failed = true;
        }
      } else {
        mkdirSync(dirname(outPath), { recursive: true });
        writeFileSync(outPath, content);
        console.log(`wrote ${outPath}`);
      }
    }
  }
  if (check && failed) process.exit(1);
  if (check) console.log('generated cores are up to date');
}

main();
