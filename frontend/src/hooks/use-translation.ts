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

import { useEffect, useState } from 'react';
import { useAppSelector } from '@/store/hooks';
import { loadTranslations, getTranslation } from '@/utils/i18n';

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
    const [translations, setTranslations] = useState<Record<string, string>>({});
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        setIsLoading(true);
        loadTranslations(componentPath, language)
            .then(setTranslations)
            .finally(() => setIsLoading(false));
    }, [componentPath, language]);

    const t = (key: string, fallbackOrParams?: string | Record<string, any>): string => {
        return getTranslation(translations, key, fallbackOrParams);
    };

    return { t, isLoading, language };
};
