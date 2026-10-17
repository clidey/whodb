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

// Plain JavaScript so both the CE config and extension configs can load it.

import fs from 'node:fs';
import path from 'node:path';
import { load } from 'js-yaml';

const INDEX_ID = 'virtual:whodb-locales';
const PACK_PREFIX = 'virtual:whodb-locale-pack/';

/**
 * @typedef {{ layer: 'base' | 'extension', dir: string }} LocaleRoot
 * @typedef {Record<string, Record<string, Record<string, string>>>} LanguagePack
 *   layer -> component path (e.g. "components/sidebar") -> key -> translation
 */

function listYamlFiles(dir) {
    if (!fs.existsSync(dir)) return [];
    return fs.readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) return listYamlFiles(full);
        return /\.ya?ml$/.test(entry.name) ? [full] : [];
    });
}

/**
 * Splits the YAML locale files into one module per language so the browser loads only the
 * languages it uses. Every locale file keeps its current layout (all languages side by side).
 *
 * - `virtual:whodb-locales` exports the default language's pack (bundled with the app,
 *   because it is the fallback for missing translations) and a lazy loader per language.
 * - `virtual:whodb-locale-pack/<language>` holds that language's strings for every file.
 *
 * The plugin's `api.languagePackFiles` maps each language to its emitted chunk after a
 * build, so other plugins (for example a service worker manifest) can treat them specially.
 *
 * @param {{ roots: LocaleRoot[], defaultLanguage: string }} options
 * @returns {import('vite').Plugin}
 */
export default function localePacksPlugin({ roots, defaultLanguage }) {
    /** @type {Map<string, LanguagePack> | null} */
    let packs = null;
    /** @type {Record<string, string>} */
    const languagePackFiles = {};

    const collect = () => {
        if (packs) return packs;
        packs = new Map();
        for (const { layer, dir } of roots) {
            for (const file of listYamlFiles(dir)) {
                const componentPath = path.relative(dir, file).replace(/\\/g, '/').replace(/\.ya?ml$/, '');
                const parsed = load(fs.readFileSync(file, 'utf8')) ?? {};
                for (const [language, translations] of Object.entries(parsed)) {
                    if (!packs.has(language)) packs.set(language, {});
                    const pack = packs.get(language);
                    pack[layer] ??= {};
                    pack[layer][componentPath] = translations;
                }
            }
        }
        return packs;
    };

    const isLocaleFile = file => /\.ya?ml$/.test(file) && roots.some(({ dir }) => file.startsWith(path.resolve(dir) + path.sep));

    return {
        name: 'whodb-locale-packs',
        api: { languagePackFiles },
        resolveId(id) {
            if (id === INDEX_ID || id.startsWith(PACK_PREFIX)) return `\0${id}`;
            return null;
        },
        load(id) {
            if (!id.startsWith('\0')) return null;
            const moduleId = id.slice(1);
            if (moduleId !== INDEX_ID && !moduleId.startsWith(PACK_PREFIX)) return null;
            for (const { dir } of roots) {
                for (const file of listYamlFiles(dir)) this.addWatchFile(file);
            }
            const all = collect();
            if (moduleId === INDEX_ID) {
                const loaders = [...all.keys()]
                    .filter(language => language !== defaultLanguage)
                    .map(language => `    ${JSON.stringify(language)}: () => import(${JSON.stringify(PACK_PREFIX + language)}),`)
                    .join('\n');
                return [
                    `export const defaultLanguagePack = ${JSON.stringify(all.get(defaultLanguage) ?? {})};`,
                    `export const languagePackLoaders = {\n${loaders}\n};`,
                ].join('\n');
            }
            const language = moduleId.slice(PACK_PREFIX.length);
            return `export default ${JSON.stringify(all.get(language) ?? {})};`;
        },
        watchChange(file) {
            if (isLocaleFile(file)) packs = null;
        },
        configureServer(server) {
            const invalidate = file => {
                if (!isLocaleFile(file)) return;
                packs = null;
                for (const module of server.moduleGraph.idToModuleMap.values()) {
                    if (module.id === `\0${INDEX_ID}` || module.id?.startsWith(`\0${PACK_PREFIX}`)) {
                        server.moduleGraph.invalidateModule(module);
                    }
                }
                server.ws.send({ type: 'full-reload' });
            };
            for (const { dir } of roots) server.watcher.add(dir);
            server.watcher.on('change', invalidate);
            server.watcher.on('add', invalidate);
            server.watcher.on('unlink', invalidate);
        },
        generateBundle(_options, bundle) {
            for (const chunk of Object.values(bundle)) {
                if (chunk.type === 'chunk' && chunk.facadeModuleId?.startsWith(`\0${PACK_PREFIX}`)) {
                    languagePackFiles[chunk.facadeModuleId.slice(PACK_PREFIX.length + 1)] = chunk.fileName;
                }
            }
        },
    };
}
