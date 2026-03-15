/*
 * Copyright 2025 Clidey, Inc.
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

import { type ReactNode, useEffect, useState } from 'react';
import { useAppSelector } from '@/store/hooks';
import { loadTranslationsSync, getTranslation } from '@/utils/i18n';

/**
 * Hook for loading and using translations from YAML locale files.
 * Automatically reloads translations when the language setting changes.
 *
 * @param componentPath - Path to the YAML file relative to locales directory (e.g., "components/sidebar")
 * @returns Object containing:
 *   - t: Translation function that accepts a key and optional interpolation params
 *   - isLoading: Whether translations are currently being loaded
 *   - language: The current language code
 *
 * @example
 * ```tsx
 * const { t } = useTranslation('components/sidebar');
 * return <span>{t('menuItem')}</span>;
 * ```
 */
export const useTranslation = (componentPath: string) => {
    const language = useAppSelector(state => state.settings.language);
    const [translations, setTranslations] = useState<Record<string, string>>(() =>
        loadTranslationsSync(componentPath, language)
    );
    const [isLoading, setIsLoading] = useState(false);

    useEffect(() => {
        setTranslations(loadTranslationsSync(componentPath, language));
    }, [componentPath, language]);

    /**
     * Translates a key with optional interpolation.
     * - String/number params: returns a string (e.g., `t('greeting', { name: 'Alice' })`)
     * - ReactNode params (JSX elements): returns ReactNode, allowing translated strings
     *   to contain embedded components like links that translators can freely reorder.
     *
     * @example
     * ```tsx
     * // String interpolation → string
     * t('greeting', { name: 'Alice' })
     *
     * // JSX interpolation → ReactNode
     * t('details', { link: <a href="/privacy">Privacy Policy</a> })
     * ```
     */
    const t: {
        (key: string): string;
        (key: string, fallback: string): string;
        (key: string, params: Record<string, string | number>): string;
        (key: string, params: Record<string, ReactNode>): ReactNode;
    } = (key: string, fallbackOrParams?: string | Record<string, any>): any => {
        if (typeof fallbackOrParams !== 'object' || fallbackOrParams === null) {
            return getTranslation(translations, key, fallbackOrParams);
        }

        const hasJsx = Object.values(fallbackOrParams).some(
            v => v !== null && v !== undefined && typeof v === 'object'
        );

        if (!hasJsx) {
            return getTranslation(translations, key, fallbackOrParams);
        }

        // JSX interpolation: split template on {placeholders} and interleave with ReactNode values
        const template = translations[key] || key;
        const parts: ReactNode[] = [];
        let lastIndex = 0;
        const regex = /\{(\w+)\}/g;
        let match;

        while ((match = regex.exec(template)) !== null) {
            if (match.index > lastIndex) {
                parts.push(template.slice(lastIndex, match.index));
            }
            parts.push(fallbackOrParams[match[1]] ?? match[0]);
            lastIndex = match.index + match[0].length;
        }

        if (lastIndex < template.length) {
            parts.push(template.slice(lastIndex));
        }

        return parts;
    };

    return { t, isLoading, language };
};
