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
    | 'bg_BG'
    | 'bn_BD'
    | 'ca_ES'
    | 'cs_CZ'
    | 'da_DK'
    | 'de_DE'
    | 'el_GR'
    | 'en_GB'
    | 'en_US'
    | 'es_ES'
    | 'fa_IR'
    | 'fi_FI'
    | 'fr_FR'
    | 'hi_IN'
    | 'hr_HR'
    | 'hu_HU'
    | 'id_ID'
    | 'it_IT'
    | 'iw_IL'
    | 'ja_JP'
    | 'ko_KR'
    | 'ms_MY'
    | 'nb_NO'
    | 'nl_NL'
    | 'pl_PL'
    | 'pt_BR'
    // | 'pt_PT'  // TODO: Google Translate doesn't differentiate PT-BR vs PT-PT — add back when we have distinct translations
    | 'ro_RO'
    | 'ru_RU'
    | 'sk_SK'
    | 'sv_SE'
    | 'sw_KE'
    | 'ta_IN'
    | 'th_TH'
    | 'tr_TR'
    | 'uk_UA'
    | 'ur_PK'
    | 'vi_VN'
    | 'zh_CN'
    // | 'zh_HK'  // TODO: Google Translate has no zh-HK variant (produces same as zh-TW) — add back when we have distinct translations
    | 'zh_TW';

export const DEFAULT_LANGUAGE: SupportedLanguage = 'en_US';

export const SUPPORTED_LANGUAGES: Record<SupportedLanguage, string> = {
    en_US: 'English (US)',
    en_GB: 'English (UK)',
    ar_AE: 'العربية',
    bg_BG: 'Български',
    bn_BD: 'বাংলা',
    ca_ES: 'Català',
    cs_CZ: 'Čeština',
    da_DK: 'Dansk',
    de_DE: 'Deutsch',
    el_GR: 'Ελληνικά',
    es_ES: 'Español',
    fa_IR: 'فارسی',
    fi_FI: 'Suomi',
    fr_FR: 'Français',
    hi_IN: 'हिन्दी',
    hr_HR: 'Hrvatski',
    hu_HU: 'Magyar',
    id_ID: 'Bahasa Indonesia',
    it_IT: 'Italiano',
    iw_IL: 'עברית',
    ja_JP: '日本語',
    ko_KR: '한국어',
    ms_MY: 'Bahasa Melayu',
    nb_NO: 'Norsk Bokmål',
    nl_NL: 'Nederlands',
    pl_PL: 'Polski',
    pt_BR: 'Português (Brasil)',
    // pt_PT: 'Português (Portugal)',  // TODO: add back when we have distinct translations
    ro_RO: 'Română',
    ru_RU: 'Русский',
    sk_SK: 'Slovenčina',
    sv_SE: 'Svenska',
    sw_KE: 'Kiswahili',
    ta_IN: 'தமிழ்',
    th_TH: 'ไทย',
    tr_TR: 'Türkçe',
    uk_UA: 'Українська',
    ur_PK: 'اردو',
    vi_VN: 'Tiếng Việt',
    zh_CN: '中文 (简体)',
    // zh_HK: '中文 (香港)',  // TODO: add back when we have distinct translations
    zh_TW: '中文 (繁體)',
};

export function isSupportedLanguage(value: string): value is SupportedLanguage {
    return value in SUPPORTED_LANGUAGES;
}
