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

import yaml from 'js-yaml';
import {isEEMode} from '@/config/ee-imports';
import {type SupportedLanguage, DEFAULT_LANGUAGE} from '@/utils/languages';

type TranslationCache = Record<string, Record<string, any>>;

const translationCache: TranslationCache = {};

// Import all YAML files using Vite's import.meta.glob
const ceModules = import.meta.glob<string>('/src/locales/**/*.yaml', { query: '?raw', import: 'default', eager: true });
const eeModules = import.meta.glob<string>('../../../ee/frontend/src/locales/**/*.yaml', { query: '?raw', import: 'default', eager: true });

// Helper function to find module by component path
const findModule = (modules: Record<string, string>, componentPath: string): string | undefined => {
    // Try exact match first
    for (const key in modules) {
        if (key.endsWith(`/${componentPath}.yaml`)) {
            return modules[key];
        }
    }
    return undefined;
};

export const loadTranslationsSync = (
    componentPath: string,
    language: SupportedLanguage
): Record<string, string> => {
    const cacheKey = `${componentPath}-${language}`;

    if (translationCache[cacheKey]) {
        return translationCache[cacheKey];
    }

    try {
        let translations: Record<string, string> | undefined;

        // Load CE locale files as the base
        const ceContent = findModule(ceModules, componentPath);
        if (ceContent) {
            const parsed = yaml.load(ceContent) as Record<string, Record<string, string>>;
            translations = parsed[language] || parsed[DEFAULT_LANGUAGE];
        }

        // In EE mode, merge EE keys on top of CE (EE overrides individual keys)
        if (isEEMode) {
            const eeContent = findModule(eeModules, componentPath);
            if (eeContent) {
                const parsed = yaml.load(eeContent) as Record<string, Record<string, string>>;
                const eeTranslations = parsed[language];
                if (eeTranslations) {
                    translations = { ...translations, ...eeTranslations };
                }
            }
        }

        if (!translations) {
            console.error(`Translation file not found for ${componentPath}`, {
                availableCE: Object.keys(ceModules),
                availableEE: Object.keys(eeModules),
                isEEMode,
                language
            });
            return {};
        }

        translationCache[cacheKey] = translations;
        return translations;
    } catch (error) {
        console.error(`Failed to load translations for ${componentPath}:`, error);
        return {};
    }
};

export const loadTranslations = async (
    componentPath: string,
    language: SupportedLanguage
): Promise<Record<string, string>> => {
    return loadTranslationsSync(componentPath, language);
};

export const getTranslation = (
    translations: Record<string, string>,
    key: string,
    fallbackOrParams?: string | Record<string, any>
): string => {
    const template = translations[key] || (typeof fallbackOrParams === 'string' ? fallbackOrParams : key);

    if (typeof fallbackOrParams === 'object' && fallbackOrParams !== null) {
        return template.replace(/\{(\w+)\}/g, (match, paramKey) => {
            return String(fallbackOrParams[paramKey] ?? match);
        });
    }

    return template;
};
