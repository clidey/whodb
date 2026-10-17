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

import {type SupportedLanguage, DEFAULT_LANGUAGE} from '@/utils/languages';
import {defaultLanguagePack, languagePackLoaders, type LanguagePack} from 'virtual:whodb-locales';

type TranslationMap = Record<string, string>;
type TranslationCache = Record<string, TranslationMap>;

const translationCache: TranslationCache = {};

// Translations are split into one pack per language at build time (see
// plugins/vite-plugin-locale-packs.mjs). The default language is bundled because it is the
// fallback for files without a translation; every other language is loaded on demand.
const languagePacks: Partial<Record<SupportedLanguage, LanguagePack>> = {
    [DEFAULT_LANGUAGE]: defaultLanguagePack,
};
const pendingLanguageLoads = new Map<SupportedLanguage, Promise<void>>();
const languagePackListeners = new Set<() => void>();
let languagePackVersion = 0;

/** Returns whether a language's translations are available synchronously. */
export const isLanguageLoaded = (language: SupportedLanguage): boolean => languagePacks[language] != null;

/** Loads a language's translations. Concurrent calls for the same language share one download. */
export const loadLanguage = (language: SupportedLanguage): Promise<void> => {
    if (languagePacks[language]) return Promise.resolve();
    const loader = languagePackLoaders[language];
    if (!loader) {
        // No file has this language yet; every lookup falls back to the default language.
        languagePacks[language] = {};
        return Promise.resolve();
    }
    let pending = pendingLanguageLoads.get(language);
    if (!pending) {
        pending = loader()
            .then(module => {
                languagePacks[language] = module.default;
                languagePackVersion++;
                languagePackListeners.forEach(listener => { listener(); });
            })
            .finally(() => pendingLanguageLoads.delete(language));
        pendingLanguageLoads.set(language, pending);
    }
    return pending;
};

/** Subscribes to language pack loads, for components that must re-render when one arrives. */
export const subscribeToLanguagePacks = (listener: () => void): (() => void) => {
    languagePackListeners.add(listener);
    return () => languagePackListeners.delete(listener);
};

/** Increments whenever a language pack finishes loading. */
export const getLanguagePackVersion = (): number => languagePackVersion;

/**
 * Cached Intl.PluralRules instances per locale.
 * Constructing PluralRules loads CLDR data internally — one instance per locale is sufficient.
 */
const pluralRulesCache = new Map<string, Intl.PluralRules>();

/**
 * Returns the CLDR plural category for a given count and locale.
 * Categories: "zero" | "one" | "two" | "few" | "many" | "other"
 */
export const getPluralCategory = (locale: string, count: number): Intl.LDMLPluralRule => {
    let rules = pluralRulesCache.get(locale);
    if (!rules) {
        rules = new Intl.PluralRules(locale.replace('_', '-'));
        pluralRulesCache.set(locale, rules);
    }
    return rules.select(count);
};

export const loadTranslationsSync = (
    componentPath: string,
    language: SupportedLanguage
): TranslationMap => {
    const cacheKey = `${componentPath}-${language}`;

    if (translationCache[cacheKey]) {
        return translationCache[cacheKey];
    }

    try {
        const pack = languagePacks[language];
        // A file without this language falls back to its default-language section, as does
        // every file while the language pack is still loading.
        const layer = (name: keyof LanguagePack, path: string): TranslationMap | undefined =>
            pack?.[name]?.[path] ?? defaultLanguagePack[name]?.[path];

        // Start with shared common translations as the base layer, merge component-specific
        // translations on top, then extension translations (which override base keys).
        const base = layer('base', componentPath);
        const extension = layer('extension', componentPath);
        const translations: TranslationMap = { ...layer('base', 'common'), ...base, ...extension };

        if (!base && !extension) {
            console.error(`Translation file not found for ${componentPath}`, { language });
            return translations;
        }

        if (pack) translationCache[cacheKey] = translations;
        return translations;
    } catch (error) {
        console.error(`Failed to load translations for ${componentPath}:`, error);
        return {};
    }
};

/**
 * Resolves a translation key, handling pluralization and interpolation.
 *
 * Pluralization: when `params` contains a numeric `count` property, the function
 * looks up `key_<category>` (e.g. `key_one`, `key_other`) using Intl.PluralRules
 * for the given locale. Falls back to the base `key` if no plural form is found.
 *
 * Interpolation: `{placeholder}` tokens in the template are replaced with
 * corresponding values from `params`.
 */
export const getTranslation = (
    translations: TranslationMap,
    key: string,
    language: SupportedLanguage,
    params?: Record<string, any>
): string => {
    let resolvedKey = key;

    // Pluralization: if params has a numeric `count`, resolve the plural form
    if (params != null && typeof params.count === 'number') {
        const category = getPluralCategory(language, params.count);
        const pluralKey = `${key}_${category}`;
        if (translations[pluralKey]) {
            resolvedKey = pluralKey;
        }
    }

    const template = translations[resolvedKey] || resolvedKey;

    // Skip regex when the template has no placeholders
    if (params != null && template.includes('{')) {
        return template.replace(/\{(\w+)\}/g, (match, paramKey) => {
            return String(params[paramKey] ?? match);
        });
    }

    return template;
};
