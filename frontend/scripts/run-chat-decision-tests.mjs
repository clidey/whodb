import { execFileSync } from 'node:child_process';
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const outDir = '.chat-decision-test-build';

try {
    rmSync(outDir, { recursive: true, force: true });
    mkdirSync(outDir, { recursive: true });
    writeFileSync(join(outDir, 'package.json'), '{"type":"commonjs"}\n');

    execFileSync('node_modules/.bin/tsc', [
        '--module', 'CommonJS',
        '--target', 'ES2022',
        '--moduleResolution', 'bundler',
        '--lib', 'ES2022,DOM',
        '--strict',
        '--skipLibCheck',
        '--esModuleInterop',
        '--ignoreConfig',
        '--rootDir', 'src',
        '--outDir', outDir,
        'src/generated/schema-types.ts',
        'src/pages/chat/decisions.ts',
        'src/pages/chat/decisions.test.ts',
    ], { stdio: 'inherit' });

    execFileSync('node', [join(outDir, 'pages/chat/decisions.test.js')], { stdio: 'inherit' });
} finally {
    rmSync(outDir, { recursive: true, force: true });
}
