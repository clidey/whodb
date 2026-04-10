#!/usr/bin/env node
//
// Translates missing and stale translation keys using google-translate-api-x.
// Reads drift.json (produced by detect.py) to know exactly which keys need work.
//
// Usage:
//   python3 detect.py              # step 1: detect drift
//   node translate.mjs             # step 2: translate + apply
//   node translate.mjs fr_FR de_DE # step 2: translate specific locales only
//
// Or use the wrapper:
//   ./run.sh                       # runs both steps
//   ./run.sh fr_FR de_DE           # specific locales

import translate from 'google-translate-api-x';
import { load as yamlLoad } from 'js-yaml';
import { readFileSync, writeFileSync, existsSync, unlinkSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = join(__dirname, '../..');

// ─── Configuration ──────────────────────────────────────────────

// Our locale codes → Google Translate language codes
const LANG_MAP = {
    af_ZA: 'af',
    am_ET: 'am',
    ar_AE: 'ar',
    az_AZ: 'az',
    bg_BG: 'bg',
    bn_BD: 'bn',
    ca_ES: 'ca',
    cs_CZ: 'cs',
    da_DK: 'da',
    de_DE: 'de',
    el_GR: 'el',
    en_GB: null,       // handled separately — US→UK spelling
    es_ES: 'es',
    et_EE: 'et',
    fa_IR: 'fa',
    fi_FI: 'fi',
    fr_FR: 'fr',
    gu_IN: 'gu',
    ha_NG: 'ha',
    hi_IN: 'hi',
    hr_HR: 'hr',
    hu_HU: 'hu',
    id_ID: 'id',
    it_IT: 'it',
    iw_IL: 'iw',
    ja_JP: 'ja',
    ka_GE: 'ka',
    km_KH: 'km',
    kn_IN: 'kn',
    ko_KR: 'ko',
    lt_LT: 'lt',
    lv_LV: 'lv',
    ml_IN: 'ml',
    mr_IN: 'mr',
    ms_MY: 'ms',
    nb_NO: 'no',
    ne_NP: 'ne',
    nl_NL: 'nl',
    pa_IN: 'pa',
    pl_PL: 'pl',
    pt_BR: 'pt',
    ro_RO: 'ro',
    ru_RU: 'ru',
    si_LK: 'si',
    sk_SK: 'sk',
    sl_SI: 'sl',
    sr_RS: 'sr',
    sv_SE: 'sv',
    sw_KE: 'sw',
    ta_IN: 'ta',
    te_IN: 'te',
    th_TH: 'th',
    tl_PH: 'tl',
    tr_TR: 'tr',
    uk_UA: 'uk',
    ur_PK: 'ur',
    uz_UZ: 'uz',
    vi_VN: 'vi',
    yo_NG: 'yo',
    zh_CN: 'zh',
    zh_TW: 'zh-TW',
};

const CHUNK_SIZE = 25;
const DELAY_BETWEEN_CHUNKS = 1000;
const DELAY_BETWEEN_LANGUAGES = 3000;
const MAX_RETRIES = 3;

// ─── Placeholder handling ───────────────────────────────────────

function protectPlaceholders(text) {
    const placeholders = [];
    const result = text.replace(/\{(\w+)\}/g, (match) => {
        const idx = placeholders.length;
        placeholders.push(match);
        return `[[[${idx}]]]`;
    });
    return { text: result, placeholders };
}

function restorePlaceholders(text, placeholders) {
    if (placeholders.length === 0) return text;
    return text.replace(/\[\[\[\s*(\d+)\s*\]\]\]/g, (_match, idx) => {
        return placeholders[parseInt(idx)] ?? _match;
    });
}

// ─── YAML formatting ────────────────────────────────────────────

const YAML_KEYWORDS = new Set([
    'yes', 'no', 'true', 'false', 'null', 'on', 'off', 'y', 'n',
    'Yes', 'No', 'True', 'False', 'Null', 'On', 'Off', 'Y', 'N',
    'YES', 'NO', 'TRUE', 'FALSE', 'NULL', 'ON', 'OFF',
]);

function formatYamlKey(key) {
    if (YAML_KEYWORDS.has(key)) return `"${key}"`;
    return key;
}

function formatYamlValue(val) {
    const str = String(val);
    if (str === '') return '""';
    if (YAML_KEYWORDS.has(str)) return `"${str}"`;
    // Quote if value contains YAML-special characters
    if (
        /^[-?:,\[{}#&*!|>'"@%`\]]/.test(str) ||
        str.includes(': ') || str.includes(' #') ||
        /[{}\[\]]/.test(str) ||
        /^\s|\s$/.test(str) ||
        str.endsWith(':')
    ) {
        const escaped = str.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
        return `"${escaped}"`;
    }
    return str;
}

function buildYamlBlock(locale, translations) {
    const lines = [`${locale}:`];
    for (const [key, value] of Object.entries(translations)) {
        lines.push(`  ${formatYamlKey(key)}: ${formatYamlValue(String(value))}`);
    }
    return lines.join('\n');
}

// ─── File rebuilding ────────────────────────────────────────────

function extractEnUsSection(content) {
    // Find the en_US section boundaries in the raw text
    const lines = content.split('\n');
    const startIdx = lines.findIndex(l => /^en_US:/.test(l));
    if (startIdx === -1) return null;

    let endIdx = startIdx + 1;
    while (endIdx < lines.length) {
        // Next top-level section: non-indented line matching a locale header
        if (/^[a-zA-Z_]+:/.test(lines[endIdx])) break;
        endIdx++;
    }
    return lines.slice(startIdx, endIdx).join('\n').trimEnd();
}

function getOriginalSectionOrder(content) {
    const order = [];
    const headerRegex = /^([a-zA-Z_]+):/gm;
    let match;
    while ((match = headerRegex.exec(content)) !== null) {
        if (match[1] !== 'en_US' && !order.includes(match[1])) {
            order.push(match[1]);
        }
    }
    return order;
}

function rebuildFile(originalContent, data) {
    // Preserve en_US section formatting exactly as written
    const enUsSection = extractEnUsSection(originalContent) ||
        buildYamlBlock('en_US', data.en_US);

    // Preserve original section order, append new sections at end
    const originalOrder = getOriginalSectionOrder(originalContent);
    const allLocales = Object.keys(data).filter(k => k !== 'en_US');
    const orderedLocales = [
        ...originalOrder.filter(l => allLocales.includes(l)),
        ...allLocales.filter(l => !originalOrder.includes(l)),
    ];

    // Rebuild: en_US (preserved) + translation sections (rebuilt)
    const sections = [enUsSection];
    for (const locale of orderedLocales) {
        const localeData = data[locale];
        if (localeData && typeof localeData === 'object' && Object.keys(localeData).length > 0) {
            sections.push(buildYamlBlock(locale, localeData));
        }
    }

    return sections.join('\n\n') + '\n';
}

// ─── Translation ────────────────────────────────────────────────

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

async function translateBatch(texts, targetLang, retries = MAX_RETRIES) {
    try {
        const results = await translate(texts, {
            from: 'en',
            to: targetLang,
            rejectOnPartialFail: false,
            forceTo: true,
        });
        if (Array.isArray(results)) {
            return results.map(r => r?.text ?? null);
        }
        return [results?.text ?? null];
    } catch (err) {
        if (retries > 0 && (
            err.message?.includes('429') ||
            err.message?.includes('Too Many') ||
            err.message?.includes('Partial Translation') ||
            err.code === 'ECONNRESET' ||
            err.code === 'ETIMEDOUT'
        )) {
            const wait = (MAX_RETRIES - retries + 1) * 5000;
            console.log(`      Retrying batch — waiting ${wait / 1000}s (${retries} retries left)...`);
            await sleep(wait);
            return translateBatch(texts, targetLang, retries - 1);
        }

        // Last resort: translate one-by-one
        if (texts.length > 1) {
            console.log(`      Batch failed, falling back to one-by-one for ${texts.length} strings...`);
            const results = [];
            for (const text of texts) {
                try {
                    const res = await translate(text, { from: 'en', to: targetLang });
                    results.push(res.text);
                } catch {
                    results.push(null);
                }
                await sleep(300);
            }
            return results;
        }

        throw err;
    }
}

async function translateEntries(entries, googleLang) {
    const keys = Object.keys(entries);
    const results = {};

    for (let i = 0; i < keys.length; i += CHUNK_SIZE) {
        const chunkKeys = keys.slice(i, i + CHUNK_SIZE);
        const chunkValues = chunkKeys.map(k => String(entries[k]));

        const protectedData = chunkValues.map(v => protectPlaceholders(v));
        const textsToTranslate = protectedData.map(p => p.text);

        const translated = await translateBatch(textsToTranslate, googleLang);

        for (let j = 0; j < chunkKeys.length; j++) {
            if (translated[j] == null) {
                results[chunkKeys[j]] = chunkValues[j]; // fallback to English
                continue;
            }
            let value = restorePlaceholders(translated[j], protectedData[j].placeholders);
            if (!value.trim()) value = chunkValues[j];
            results[chunkKeys[j]] = value;
        }

        if (i + CHUNK_SIZE < keys.length) {
            await sleep(DELAY_BETWEEN_CHUNKS);
        }
    }

    return results;
}

// ─── en_GB handling ─────────────────────────────────────────────

function usToUk(text) {
    return text
        .replace(/\bCustomize\b/g, 'Customise')
        .replace(/\bcustomize\b/g, 'customise')
        .replace(/\bCustomizing\b/g, 'Customising')
        .replace(/\bcustomizing\b/g, 'customising')
        .replace(/\bPersonalize\b/g, 'Personalise')
        .replace(/\bpersonalize\b/g, 'personalise')
        .replace(/\bPersonalizing\b/g, 'Personalising')
        .replace(/\bpersonalizing\b/g, 'personalising')
        .replace(/\bOrganize\b/g, 'Organise')
        .replace(/\borganize\b/g, 'organise')
        .replace(/\bRecognize\b/g, 'Recognise')
        .replace(/\brecognize\b/g, 'recognise')
        .replace(/\bAnalyze\b/g, 'Analyse')
        .replace(/\banalyze\b/g, 'analyse')
        .replace(/\bSummarize\b/g, 'Summarise')
        .replace(/\bsummarize\b/g, 'summarise')
        .replace(/\bSynchronize\b/g, 'Synchronise')
        .replace(/\bsynchronize\b/g, 'synchronise')
        .replace(/\bAuthorize\b/g, 'Authorise')
        .replace(/\bauthorize\b/g, 'authorise')
        .replace(/\bInitialize\b/g, 'Initialise')
        .replace(/\binitialize\b/g, 'initialise')
        .replace(/\bBehavior\b/g, 'Behaviour')
        .replace(/\bbehavior\b/g, 'behaviour')
        .replace(/\bColor\b/g, 'Colour')
        .replace(/\bcolor\b/g, 'colour')
        .replace(/\bFavor\b/g, 'Favour')
        .replace(/\bfavor\b/g, 'favour')
        .replace(/\bFavorite\b/g, 'Favourite')
        .replace(/\bfavorite\b/g, 'favourite')
        .replace(/\bCanceled\b/g, 'Cancelled')
        .replace(/\bcanceled\b/g, 'cancelled')
        .replace(/\bCanceling\b/g, 'Cancelling')
        .replace(/\bcanceling\b/g, 'cancelling')
        .replace(/\bModeling\b/g, 'Modelling')
        .replace(/\bmodeling\b/g, 'modelling')
        .replace(/\bLabeled\b/g, 'Labelled')
        .replace(/\blabeled\b/g, 'labelled');
}

// ─── Main ───────────────────────────────────────────────────────

async function main() {
    // Load drift.json (produced by detect.py)
    const driftPath = join(__dirname, 'drift.json');
    if (!existsSync(driftPath)) {
        console.error('No drift.json found. Run detect.py first:');
        console.error('  python3 detect.py');
        process.exit(1);
    }

    const drift = JSON.parse(readFileSync(driftPath, 'utf-8'));
    const driftFiles = drift.files || {};

    if (Object.keys(driftFiles).length === 0) {
        console.log('No drift detected. All translations up to date.');
        if (drift.checksums) {
            writeFileSync(
                join(__dirname, 'checksums.json'),
                JSON.stringify(drift.checksums, null, 2) + '\n',
                'utf-8',
            );
        }
        unlinkSync(driftPath);
        return;
    }

    // CLI locale filtering
    const cliLocales = process.argv.slice(2);
    if (cliLocales.length > 0) {
        const invalid = cliLocales.filter(l => !LANG_MAP.hasOwnProperty(l));
        if (invalid.length > 0) {
            console.error(`Unknown locales: ${invalid.join(', ')}`);
            process.exit(1);
        }
        for (const filePath of Object.keys(driftFiles)) {
            for (const locale of Object.keys(driftFiles[filePath])) {
                if (!cliLocales.includes(locale)) {
                    delete driftFiles[filePath][locale];
                }
            }
            if (Object.keys(driftFiles[filePath]).length === 0) {
                delete driftFiles[filePath];
            }
        }
        if (Object.keys(driftFiles).length === 0) {
            console.log('No drift for the specified locales.');
            unlinkSync(driftPath);
            return;
        }
    }

    // ── Phase 1: Collect and group translation tasks ─────────────

    // Group by Google Translate language code for efficient batching.
    // Each task: { filePath, locale, keysToTranslate }
    const langTasks = new Map();

    for (const [filePath, fileData] of Object.entries(driftFiles)) {
        for (const [locale, localeData] of Object.entries(fileData)) {
            const keysToTranslate = {
                ...(localeData.missing || {}),
                ...(localeData.stale || {}),
            };
            if (Object.keys(keysToTranslate).length === 0) continue;

            const googleLang = LANG_MAP[locale];
            if (googleLang === undefined) continue;
            const key = googleLang ?? '__en_GB__';

            if (!langTasks.has(key)) langTasks.set(key, []);
            langTasks.get(key).push({ filePath, locale, keysToTranslate });
        }
    }

    // translatedKeys[filePath][locale] = { key: translatedValue }
    const translatedKeys = {};

    // ── Phase 2: Translate ───────────────────────────────────────

    let groupIdx = 0;
    const totalGroups = langTasks.size;

    for (const [googleLang, tasks] of langTasks) {
        groupIdx++;
        const isEnGB = googleLang === '__en_GB__';
        const label = isEnGB ? 'UK spelling' : googleLang;

        const localesInGroup = [...new Set(tasks.map(t => t.locale))];
        console.log(`\n[${groupIdx}/${totalGroups}] ${localesInGroup.join(', ')} (${label})`);
        console.log('─'.repeat(50));

        // Group by file — same file + same Google lang = translate once
        const byFile = new Map();
        for (const task of tasks) {
            if (!byFile.has(task.filePath)) byFile.set(task.filePath, []);
            byFile.get(task.filePath).push(task);
        }

        for (const [filePath, fileTasks] of byFile) {
            // Union of all keys needed across locales sharing this Google lang
            const allKeys = {};
            for (const task of fileTasks) {
                Object.assign(allKeys, task.keysToTranslate);
            }

            const keyCount = Object.keys(allKeys).length;
            let translated;

            if (isEnGB) {
                translated = {};
                for (const [key, value] of Object.entries(allKeys)) {
                    translated[key] = usToUk(String(value));
                }
                console.log(`  ${filePath}: ${keyCount} keys (UK spelling)`);
            } else {
                try {
                    translated = await translateEntries(allKeys, googleLang);
                    console.log(`  ${filePath}: ${keyCount} keys translated`);
                } catch (err) {
                    console.error(`  ${filePath}: FAILED — ${err.message}`);
                    translated = {};
                    for (const [key, value] of Object.entries(allKeys)) {
                        translated[key] = String(value);
                    }
                }
            }

            // Assign to each locale — only the keys that locale needs
            for (const task of fileTasks) {
                if (!translatedKeys[task.filePath]) translatedKeys[task.filePath] = {};
                const localeResult = {};
                for (const key of Object.keys(task.keysToTranslate)) {
                    if (translated[key] != null) {
                        localeResult[key] = translated[key];
                    }
                }
                translatedKeys[task.filePath][task.locale] = localeResult;
            }
        }

        // Delay between language groups
        if (!isEnGB && groupIdx < totalGroups) {
            console.log(`\n  Pausing ${DELAY_BETWEEN_LANGUAGES / 1000}s before next language...`);
            await sleep(DELAY_BETWEEN_LANGUAGES);
        }
    }

    // ── Phase 3: Apply changes to YAML files ─────────────────────

    console.log('\n' + '═'.repeat(50));
    console.log('Applying changes...\n');

    for (const [filePath, fileData] of Object.entries(driftFiles)) {
        const fullPath = join(PROJECT_ROOT, filePath);
        const content = readFileSync(fullPath, 'utf-8');
        const parsed = yamlLoad(content);

        if (!parsed?.en_US) continue;

        let addedCount = 0;
        let updatedCount = 0;
        let orphanedCount = 0;

        // Merge translated keys into parsed data
        const fileTranslations = translatedKeys[filePath] || {};
        for (const [locale, translations] of Object.entries(fileTranslations)) {
            if (!parsed[locale]) parsed[locale] = {};
            for (const [key, value] of Object.entries(translations)) {
                if (parsed[locale][key] !== undefined) {
                    updatedCount++;
                } else {
                    addedCount++;
                }
                parsed[locale][key] = value;
            }
        }

        // Remove orphaned keys
        for (const [locale, localeData] of Object.entries(fileData)) {
            const orphaned = localeData.orphaned || [];
            if (orphaned.length > 0 && parsed[locale]) {
                for (const key of orphaned) {
                    delete parsed[locale][key];
                }
                orphanedCount += orphaned.length;
                if (Object.keys(parsed[locale]).length === 0) {
                    delete parsed[locale];
                }
            }
        }

        // Rebuild and write file
        const newContent = rebuildFile(content, parsed);
        writeFileSync(fullPath, newContent, 'utf-8');

        const parts = [];
        if (addedCount > 0) parts.push(`${addedCount} added`);
        if (updatedCount > 0) parts.push(`${updatedCount} updated`);
        if (orphanedCount > 0) parts.push(`${orphanedCount} orphaned removed`);
        console.log(`  ${filePath}: ${parts.join(', ')}`);
    }

    // ── Phase 4: Finalize ────────────────────────────────────────

    if (drift.checksums) {
        writeFileSync(
            join(__dirname, 'checksums.json'),
            JSON.stringify(drift.checksums, null, 2) + '\n',
            'utf-8',
        );
        console.log('\nUpdated checksums.json');
    }

    unlinkSync(driftPath);

    console.log('\n' + '═'.repeat(50));
    console.log('Done! All translations applied.');
    console.log('\nNotes:');
    console.log('  - en_GB: US→UK spelling substitutions only — review recommended');
    console.log('  - All translations are machine-generated — native speaker review recommended');
}

main().catch(err => {
    console.error('\nFatal error:', err);
    process.exit(1);
});
