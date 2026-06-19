import { execFileSync } from 'node:child_process';
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const outDir = '.analytics-test-build';

try {
    rmSync(outDir, { recursive: true, force: true });
    mkdirSync(outDir, { recursive: true });
    writeFileSync(join(outDir, 'package.json'), '{"type":"commonjs"}\n');

    execFileSync('node_modules/.bin/tsc', [
        '--module', 'CommonJS',
        '--target', 'ES2022',
        '--moduleResolution', 'node',
        '--lib', 'ES2022,DOM',
        '--strict',
        '--skipLibCheck',
        '--esModuleInterop',
        '--rootDir', 'src',
        '--outDir', outDir,
        'src/config/analytics-events.ts',
        'src/config/analytics-sanitize.ts',
        'src/config/analytics-sanitize.test.ts',
    ], { stdio: 'inherit' });

    execFileSync('node', [join(outDir, 'config/analytics-sanitize.test.js')], { stdio: 'inherit' });
} finally {
    rmSync(outDir, { recursive: true, force: true });
}
