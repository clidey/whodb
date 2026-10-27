/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Compiles the 404 motion model and its keyframe checks, then runs them with
// plain Node. Same shape as run-analytics-tests.mjs.
import { execFileSync } from 'node:child_process';
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const outDir = '.not-found-test-build';

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
        'src/components/not-found/pose.ts',
        'src/components/not-found/pose.test.ts',
    ], { stdio: 'inherit' });
    execFileSync('node', [join(outDir, 'components/not-found/pose.test.js')], { stdio: 'inherit' });
} finally {
    rmSync(outDir, { recursive: true, force: true });
}
