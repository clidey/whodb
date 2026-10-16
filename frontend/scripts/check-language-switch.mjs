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

// Lint guard: the UI language may only change through changeLanguage (src/utils/change-language.ts),
// which loads the language's translations before switching. Setting it directly would show
// English until the translations arrive.
//
//   node scripts/check-language-switch.mjs <dir> [<dir>...]

import fs from 'node:fs';
import path from 'node:path';

const ALLOWED = [path.join('utils', 'change-language.ts'), path.join('store', 'settings.ts')];
const PATTERN = /SettingsActions\s*\.\s*setLanguage\s*\(/;

const dirs = process.argv.slice(2);
if (dirs.length === 0) {
    console.error('usage: check-language-switch.mjs <dir> [<dir>...]');
    process.exit(2);
}

const listSources = dir => fs.readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return entry.name === 'node_modules' || entry.name === 'generated' ? [] : listSources(full);
    return /\.(ts|tsx)$/.test(entry.name) ? [full] : [];
});

const errors = [];
for (const dir of dirs) {
    for (const file of listSources(dir)) {
        if (ALLOWED.some(allowed => file.endsWith(allowed))) continue;
        fs.readFileSync(file, 'utf8').split('\n').forEach((line, index) => {
            if (PATTERN.test(line)) {
                errors.push(`${file}:${index + 1}: set the language with changeLanguage() from @/utils/change-language`);
            }
        });
    }
}

if (errors.length > 0) {
    console.error(errors.join('\n'));
    process.exit(1);
}
