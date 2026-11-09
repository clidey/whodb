import { execFileSync } from 'node:child_process';
import { cpSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { tmpdir } from 'node:os';
import { basename, dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const require = createRequire(import.meta.url);
const dory = join(dirname(require.resolve('@clidey/dory/package.json')), 'bin/dist/dory.js');
const workspace = mkdtempSync(join(tmpdir(), 'whodb-pages-'));
const output = join(root, '..', 'dist', 'docs-pages');
const excluded = new Set(['node_modules', 'dist', '.git']);

try {
  cpSync(root, workspace, {
    recursive: true,
    filter: source => !excluded.has(basename(source)) && !basename(source).startsWith('.env'),
  });
  const configPath = join(workspace, 'dory.json');
  const config = JSON.parse(readFileSync(configPath, 'utf8'));
  config.url = 'https://clidey.github.io';
  config.basePath = '/whodb';
  writeFileSync(configPath, `${JSON.stringify(config, null, 2)}\n`);

  for (const command of ['check', 'build']) {
    execFileSync(process.execPath, [dory, command], { cwd: workspace, stdio: 'inherit' });
  }
  execFileSync(process.execPath, [join(root, 'scripts/check-build.mjs'), join(workspace, 'dist')], {
    stdio: 'inherit',
  });
  writeFileSync(join(workspace, 'dist/.nojekyll'), '');
  rmSync(output, { recursive: true, force: true });
  cpSync(join(workspace, 'dist'), output, { recursive: true });
  console.log(`GitHub Pages site ready in ${output}`);
} finally {
  rmSync(workspace, { recursive: true, force: true });
}
