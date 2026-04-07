#!/usr/bin/env node
//
// Translates all en_US YAML translation files to the remaining 22 supported
// languages using google-translate-api-x (unofficial Google Translate wrapper).
//
// Processes both CE (frontend/src/locales/) and EE (ee/frontend/src/locales/)
// if the EE directory exists.
//
// Usage:
//   pnpm install
//   node translate.mjs              # translate all languages
//   node translate.mjs fr_FR de_DE  # translate specific languages only
//
// The script is resumable — it checks each YAML file for existing locale keys
// and skips them, so you can safely re-run after a partial failure.

import translate from 'google-translate-api-x';
import { load as yamlLoad } from 'js-yaml';
import { readFileSync, writeFileSync, readdirSync, existsSync } from 'fs';
import { join, relative, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = join(__dirname, '../..');
const CE_LOCALES_DIR = join(PROJECT_ROOT, 'frontend/src/locales');
const EE_LOCALES_DIR = join(PROJECT_ROOT, 'ee/frontend/src/locales');

// ─── Configuration ──────────────────────────────────────────────

// Our locale codes → Google Translate language codes
const LANG_MAP = {
    ar_AE: 'ar',
    bg_BG: 'bg',
    bn_BD: 'bn',
    ca_ES: 'ca',
    cs_CZ: 'cs',
    da_DK: 'da',
    de_DE: 'de',
    el_GR: 'el',
    en_GB: null,       // handled separately — copy en_US with UK spelling adjustments
    es_ES: 'es',
    fa_IR: 'fa',
    fi_FI: 'fi',
    fr_FR: 'fr',
    hi_IN: 'hi',
    hr_HR: 'hr',
    hu_HU: 'hu',
    id_ID: 'id',
    it_IT: 'it',
    iw_IL: 'iw',
    ja_JP: 'ja',
    ko_KR: 'ko',
    ms_MY: 'ms',
    nb_NO: 'no',
    nl_NL: 'nl',
    pl_PL: 'pl',
    pt_BR: 'pt',
    // pt_PT: 'pt',    // TODO: same as pt_BR — add back when we have distinct translations
    ro_RO: 'ro',
    ru_RU: 'ru',
    sk_SK: 'sk',
    sv_SE: 'sv',
    sw_KE: 'sw',
    ta_IN: 'ta',
    th_TH: 'th',
    tr_TR: 'tr',
    uk_UA: 'uk',
    ur_PK: 'ur',
    vi_VN: 'vi',
    zh_CN: 'zh',
    // zh_HK: 'zh-tw', // TODO: same as zh_TW — add back when we have distinct translations
    zh_TW: 'zh-TW',
};

const CHUNK_SIZE = 25;           // strings per Google Translate request
const DELAY_BETWEEN_CHUNKS = 1000;
const DELAY_BETWEEN_LANGUAGES = 3000;
const MAX_RETRIES = 3;

// ─── Placeholder handling ───────────────────────────────────────
// Replace {name} with [[[n]]] before translating, restore after.
// Google Translate reliably preserves triple-bracket number tokens.

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
    const needsQuotes =
        val.includes(':') ||
        val.includes('{') ||
        val.includes('}') ||
        val.includes('#') ||
        val.includes('"') ||
        val.startsWith('- ') ||
        val.startsWith('* ') ||
        val.startsWith('&') ||
        val.startsWith('!') ||
        val.startsWith('%') ||
        val.startsWith('@') ||
        val.startsWith('`') ||
        val.startsWith('[') ||
        val.startsWith('>') ||
        val.startsWith('|') ||
        val.startsWith('?') ||
        YAML_KEYWORDS.has(val);

    if (needsQuotes) {
        const escaped = val.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
        return `"${escaped}"`;
    }
    return val;
}

function buildYamlBlock(locale, translations) {
    const lines = [`${locale}:`];
    for (const [key, value] of Object.entries(translations)) {
        lines.push(`  ${formatYamlKey(key)}: ${formatYamlValue(String(value))}`);
    }
    return lines.join('\n');
}

// ─── File discovery ─────────────────────────────────────────────

function findYamlFiles(dir, base = dir) {
    const results = [];
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
        const fullPath = join(dir, entry.name);
        if (entry.isDirectory()) {
            results.push(...findYamlFiles(fullPath, base));
        } else if (entry.name.endsWith('.yaml')) {
            results.push(relative(base, fullPath));
        }
    }
    return results.sort();
}

// ─── Translation ────────────────────────────────────────────────

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

async function translateBatch(texts, targetLang, retries = MAX_RETRIES) {
    try {
        // rejectOnPartialFail: false — if one string in a batch fails,
        // still return the rest (failed ones become null).
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

        // Last resort: if batch keeps failing, translate one-by-one
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

async function translateEntries(enTranslations, googleLang) {
    const keys = Object.keys(enTranslations);
    const results = {};

    for (let i = 0; i < keys.length; i += CHUNK_SIZE) {
        const chunkKeys = keys.slice(i, i + CHUNK_SIZE);
        const chunkValues = chunkKeys.map(k => String(enTranslations[k]));

        // Protect placeholders
        const protectedData = chunkValues.map(v => protectPlaceholders(v));
        const textsToTranslate = protectedData.map(p => p.text);

        // Skip pure-placeholder values (e.g. "{type}")
        const translated = await translateBatch(textsToTranslate, googleLang);

        for (let j = 0; j < chunkKeys.length; j++) {
            // If translation returned null (failed), fall back to English
            if (translated[j] == null) {
                results[chunkKeys[j]] = chunkValues[j];
                continue;
            }
            let value = restorePlaceholders(translated[j], protectedData[j].placeholders);
            if (!value.trim()) value = chunkValues[j];
            results[chunkKeys[j]] = value;
        }

        // Delay between chunks (skip after last chunk)
        if (i + CHUNK_SIZE < keys.length) {
            await sleep(DELAY_BETWEEN_CHUNKS);
        }
    }

    return results;
}

// ─── en_GB handling ─────────────────────────────────────────────
// Simple US → UK spelling substitutions for common UI words.
// This is a rough first pass — manual review recommended.

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
    // Collect YAML files from CE and (optionally) EE
    const localeDirs = [{ label: 'CE', dir: CE_LOCALES_DIR }];
    if (existsSync(EE_LOCALES_DIR)) {
        localeDirs.push({ label: 'EE', dir: EE_LOCALES_DIR });
    }

    // Build a flat list of { label, baseDir, relativePath } for all YAML files
    const allFiles = [];
    for (const { label, dir } of localeDirs) {
        const files = findYamlFiles(dir);
        console.log(`Found ${files.length} YAML files in ${label} (${dir})`);
        for (const f of files) {
            allFiles.push({ label, baseDir: dir, relativePath: f });
        }
    }
    console.log(`Total: ${allFiles.length} YAML files\n`);

    // Determine which locales to process
    const cliLocales = process.argv.slice(2);
    let localesToProcess = Object.keys(LANG_MAP);
    if (cliLocales.length > 0) {
        const invalid = cliLocales.filter(l => !LANG_MAP.hasOwnProperty(l));
        if (invalid.length > 0) {
            console.error(`Unknown locales: ${invalid.join(', ')}`);
            console.error(`Valid locales: ${Object.keys(LANG_MAP).join(', ')}`);
            process.exit(1);
        }
        localesToProcess = cliLocales;
    }

    // Group locales by Google Translate code to avoid duplicate API calls.
    // e.g. zh_HK and zh_TW both use 'zh-tw', so we translate once and copy.
    const langGroups = new Map();
    for (const locale of localesToProcess) {
        const googleLang = LANG_MAP[locale];
        const key = googleLang ?? '__en_GB__';
        if (!langGroups.has(key)) langGroups.set(key, []);
        langGroups.get(key).push(locale);
    }

    console.log(`Locales to translate: ${localesToProcess.length}`);
    console.log(`Unique translation passes: ${langGroups.size} (after dedup)\n`);

    let groupIdx = 0;
    for (const [googleLang, locales] of langGroups) {
        groupIdx++;
        const isEnGB = googleLang === '__en_GB__';
        const label = isEnGB ? 'UK spelling' : googleLang;
        console.log(`\n[${groupIdx}/${langGroups.size}] ${locales.join(', ')} (${label})`);
        console.log('─'.repeat(50));

        for (const { label: fileLabel, baseDir, relativePath } of allFiles) {
            const fullPath = join(baseDir, relativePath);
            const displayName = `[${fileLabel}] ${relativePath}`;
            const content = readFileSync(fullPath, 'utf-8');
            const parsed = yamlLoad(content);

            if (!parsed?.en_US) {
                console.log(`  ${displayName}: no en_US key, skipping`);
                continue;
            }

            // Which locales from this group still need translating in this file?
            const needed = locales.filter(l => !parsed[l]);
            if (needed.length === 0) {
                console.log(`  ${displayName}: already done`);
                continue;
            }

            // Translate once for this Google language code
            let translated;
            const keyCount = Object.keys(parsed.en_US).length;

            if (isEnGB) {
                translated = {};
                for (const [key, value] of Object.entries(parsed.en_US)) {
                    translated[key] = usToUk(String(value));
                }
                console.log(`  ${displayName}: ${keyCount} keys (UK spelling) → ${needed.join(', ')}`);
            } else {
                try {
                    translated = await translateEntries(parsed.en_US, googleLang);
                    console.log(`  ${displayName}: ${keyCount} keys translated → ${needed.join(', ')}`);
                } catch (err) {
                    console.error(`  ${displayName}: FAILED — ${err.message}`);
                    console.error(`  Using en_US as fallback for ${needed.join(', ')}`);
                    translated = {};
                    for (const [key, value] of Object.entries(parsed.en_US)) {
                        translated[key] = String(value);
                    }
                }
            }

            // Append a YAML block for each locale that needed it
            let appendContent = '';
            for (const locale of needed) {
                appendContent += '\n' + buildYamlBlock(locale, translated) + '\n';
            }

            writeFileSync(fullPath, content.trimEnd() + '\n' + appendContent, 'utf-8');
        }

        // Delay between language groups (skip after last or after en_GB)
        if (!isEnGB && groupIdx < langGroups.size) {
            console.log(`\n  Pausing ${DELAY_BETWEEN_LANGUAGES / 1000}s before next language...`);
            await sleep(DELAY_BETWEEN_LANGUAGES);
        }
    }

    console.log('\n' + '═'.repeat(50));
    console.log('Done! All translations written.');
    console.log('\nNotes:');
    console.log('  - en_GB: US→UK spelling substitutions only — review recommended');
    console.log('  - zh_HK/zh_TW: identical (Google has no zh-HK) — HK review recommended');
    console.log('  - pt_BR/pt_PT: identical (Google doesn\'t differentiate) — PT-PT review recommended');
    console.log('  - All translations are machine-generated — native speaker review recommended');
}

main().catch(err => {
    console.error('\nFatal error:', err);
    process.exit(1);
});
