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
    | 'ar_AE'
    | 'de_DE'
    | 'el_GR'
    | 'en_GB'
    | 'en_US'
    | 'es_ES'
    | 'fi_FI'
    | 'fr_FR'
    | 'it_IT'
    | 'iw_IL'
    | 'ja_JP'
    | 'ko_KR'
    | 'nl_NL'
    | 'pl_PL'
    | 'pt_BR'
    | 'pt_PT'
    | 'ro_RO'
    | 'ru_RU'
    | 'sk_SK'
    | 'sv_SE'
    | 'zh_CN'
    | 'zh_HK'
    | 'zh_TW';

export const DEFAULT_LANGUAGE: SupportedLanguage = 'en_US';

export const SUPPORTED_LANGUAGES: Record<SupportedLanguage, string> = {
    en_US: 'English (US)',
    en_GB: 'English (UK)',
    ar_AE: 'العربية',
    de_DE: 'Deutsch',
    el_GR: 'Ελληνικά',
    es_ES: 'Español',
    fi_FI: 'Suomi',
    fr_FR: 'Français',
    it_IT: 'Italiano',
    iw_IL: 'עברית',
    ja_JP: '日本語',
    ko_KR: '한국어',
    nl_NL: 'Nederlands',
    pl_PL: 'Polski',
    pt_BR: 'Português (Brasil)',
    pt_PT: 'Português (Portugal)',
    ro_RO: 'Română',
    ru_RU: 'Русский',
    sk_SK: 'Slovenčina',
    sv_SE: 'Svenska',
    zh_CN: '中文 (简体)',
    zh_HK: '中文 (香港)',
    zh_TW: '中文 (繁體)',
};

export function isSupportedLanguage(value: string): value is SupportedLanguage {
    return value in SUPPORTED_LANGUAGES;
}
