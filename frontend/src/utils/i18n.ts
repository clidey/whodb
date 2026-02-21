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

type TranslationCache = Record<string, Record<string, any>>;

const translationCache: TranslationCache = {};

// Import all YAML files using Vite's import.meta.glob
const ceModules = import.meta.glob<string>('/src/locales/**/*.yaml', { as: 'raw', eager: true });
const eeModules = import.meta.glob<string>('../../../ee/frontend/src/locales/**/*.yaml', { as: 'raw', eager: true });

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

export const loadTranslations = async (
    componentPath: string,
    language: 'en' | 'es' | 'de' | 'fr'
): Promise<Record<string, string>> => {
    const cacheKey = `${componentPath}-${language}`;

    if (translationCache[cacheKey]) {
        return translationCache[cacheKey];
    }

    try {
        let translations: Record<string, string> | undefined;

        // In EE mode, check EE locale files first
        if (isEEMode) {
            const eeContent = findModule(eeModules, componentPath);
            if (eeContent) {
                const parsed = yaml.load(eeContent) as Record<string, Record<string, string>>;
                if (parsed[language]) {
                    translations = parsed[language];
                }
            }
        }

        // Fallback to CE locale files
        if (!translations) {
            const ceContent = findModule(ceModules, componentPath);
            if (ceContent) {
                const parsed = yaml.load(ceContent) as Record<string, Record<string, string>>;
                translations = parsed[language] || parsed['en'];
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
