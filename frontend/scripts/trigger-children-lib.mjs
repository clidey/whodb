// Shared AST helpers for codemod-trigger-aschild.mjs and check-trigger-children.mjs.
// Uses the TypeScript compiler API already installed for the frontend build.

import fs from 'node:fs';
import path from 'node:path';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(path.join(here, '..', 'package.json'));

function loadTypeScript() {
  // TypeScript 7 ships a native compiler whose npm package has no JS compiler API
  // (or only a stub); fall back to the @typescript/typescript6 compatibility package.
  let candidate;
  try {
    candidate = require('typescript');
  } catch {
    candidate = undefined;
  }
  if (candidate && typeof candidate.createSourceFile === 'function') return candidate;
  return require('@typescript/typescript6');
}

/** @type {import('typescript')} */
export const ts = loadTypeScript();

export const config = JSON.parse(fs.readFileSync(path.join(here, 'trigger-children.json'), 'utf8'));

const TRIGGERS = new Set(config.triggers);
const INFERRED_UX = new Set(config.inferredUxChildren);
const INFERRED_INTRINSICS = new Set(config.inferredIntrinsics);
const DENY = new Set(config.denyList);

/** Recursively lists .tsx files under `dir`, skipping generated and dependency folders. */
export function listTsxFiles(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name === 'generated' || entry.name.startsWith('.')) continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...listTsxFiles(full));
    else if (entry.name.endsWith('.tsx')) out.push(full);
  }
  return out;
}

export function parse(file) {
  const text = fs.readFileSync(file, 'utf8');
  return ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
}

/** Maps local identifier → module specifier for every import in the file. */
export function importSources(source) {
  const map = new Map();
  for (const statement of source.statements) {
    if (!ts.isImportDeclaration(statement) || !statement.importClause) continue;
    const from = statement.moduleSpecifier.text;
    const { name, namedBindings } = statement.importClause;
    if (name) map.set(name.text, from);
    if (namedBindings && ts.isNamedImports(namedBindings)) {
      for (const element of namedBindings.elements) map.set(element.name.text, from);
    }
  }
  return map;
}

export function tagName(node) {
  const name = ts.isJsxElement(node) ? node.openingElement.tagName : node.tagName;
  return name.getText();
}

function isMeaningfulChild(child) {
  if (ts.isJsxText(child)) return child.text.trim().length > 0;
  if (ts.isJsxExpression(child)) return child.expression !== undefined;
  return true;
}

/** Classifies the children of a trigger element. */
export function classifyChildren(element) {
  const children = element.children.filter(isMeaningfulChild);
  if (children.length === 0) return { kind: 'empty', children };
  if (children.length > 1) return { kind: 'multiple', children };
  const [child] = children;
  if (ts.isJsxText(child)) return { kind: 'text', children, child };
  if (ts.isJsxFragment(child)) return { kind: 'fragment', children, child };
  if (ts.isJsxExpression(child)) {
    const expr = child.expression;
    if (ts.isStringLiteral(expr) || ts.isNoSubstitutionTemplateLiteral(expr) || ts.isTemplateExpression(expr)) {
      return { kind: 'text', children, child };
    }
    if (ts.isConditionalExpression(expr) || ts.isBinaryExpression(expr)) return { kind: 'conditional', children, child };
    if (ts.isArrayLiteralExpression(expr)) return { kind: 'array', children, child };
    if (ts.isJsxElement(expr) || ts.isJsxSelfClosingElement(expr)) return { kind: 'element', children, child: expr };
    if (ts.isParenthesizedExpression(expr) && (ts.isJsxElement(expr.expression) || ts.isJsxSelfClosingElement(expr.expression))) {
      return { kind: 'element', children, child: expr.expression };
    }
    return { kind: 'expression', children, child };
  }
  if (ts.isJsxElement(child) || ts.isJsxSelfClosingElement(child)) return { kind: 'element', children, child };
  return { kind: 'expression', children, child };
}

/** Returns the `asChild` attribute node and its literal value (true/false/undefined for non-literal). */
export function asChildAttribute(element) {
  const attributes = element.openingElement.attributes.properties;
  for (const attr of attributes) {
    if (!ts.isJsxAttribute(attr) || attr.name.getText() !== 'asChild') continue;
    if (!attr.initializer) return { attr, value: true };
    if (ts.isJsxExpression(attr.initializer) && attr.initializer.expression) {
      const expr = attr.initializer.expression;
      if (expr.kind === ts.SyntaxKind.TrueKeyword) return { attr, value: true };
      if (expr.kind === ts.SyntaxKind.FalseKeyword) return { attr, value: false };
    }
    return { attr, value: undefined };
  }
  return null;
}

/**
 * Decides how the library will treat a child element:
 * 'inferred' (slot-safe, asChild implied), 'denied' (never slot-safe), or 'unknown'.
 */
export function childSlotStatus(childElement, imports) {
  const name = tagName(childElement);
  if (name.includes('.')) return 'unknown';
  if (/^[a-z]/.test(name)) return INFERRED_INTRINSICS.has(name) ? 'inferred' : 'unknown';
  const from = imports.get(name);
  if (from !== config.uxPackage) return 'unknown';
  if (INFERRED_UX.has(name)) return 'inferred';
  if (DENY.has(name)) return 'denied';
  return 'unknown';
}

/** Visits every `<XTrigger>...</XTrigger>` element in the file. */
export function forEachTrigger(source, visit) {
  const imports = importSources(source);
  const walk = (node) => {
    if (ts.isJsxElement(node) && TRIGGERS.has(tagName(node))) {
      visit(node, imports);
    }
    ts.forEachChild(node, walk);
  };
  walk(source);
}

export function location(source, node) {
  const { line, character } = source.getLineAndCharacterOfPosition(node.getStart());
  return `${path.relative(process.cwd(), source.fileName)}:${line + 1}:${character + 1}`;
}
