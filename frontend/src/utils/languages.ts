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

export type SupportedLanguage =
    | 'en_US'
    | 'en_GB'
    | 'es_ES'
    | 'fr_FR'
    | 'de_DE'
    | 'pt_BR'
    | 'pt_PT'
    | 'zh_CN'
    | 'zh_HK'
    | 'zh_TW';

export const DEFAULT_LANGUAGE: SupportedLanguage = 'en_US';

export const SUPPORTED_LANGUAGES: Record<SupportedLanguage, string> = {
    en_US: 'English (US)',
    en_GB: 'English (UK)',
    es_ES: 'Español',
    fr_FR: 'Français',
    de_DE: 'Deutsch',
    pt_BR: 'Português (Brasil)',
    pt_PT: 'Português (Portugal)',
    zh_CN: '中文 (简体)',
    zh_HK: '中文 (香港)',
    zh_TW: '中文 (繁體)',
};

export function isSupportedLanguage(value: string): value is SupportedLanguage {
    return value in SUPPORTED_LANGUAGES;
}
