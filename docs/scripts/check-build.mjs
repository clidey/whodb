import { readFileSync, readdirSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const output = process.argv[2] ? resolve(process.argv[2]) : join(root, 'dist');
const excluded = new Set(['node_modules', 'dist']);

function pages(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) return excluded.has(entry.name) ? [] : pages(path);
    return entry.name.endsWith('.mdx') ? [path] : [];
  });
}

const routes = ['/', ...pages(root).map(path => `/${relative(root, path).replace(/\.mdx$/, '')}`)];
for (const route of routes) {
  const html = readFileSync(join(output, route, 'index.html'), 'utf8');
  if (!/<div id="app">\s*<\w/.test(html)) {
    throw new Error(`Missing prerendered content for ${route}`);
  }
}
console.log(`Verified prerendered content for ${routes.length} routes.`);
