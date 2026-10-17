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

import {toast} from '@clidey/ux';
import {reduxStore} from '@/store';
import {SettingsActions} from '@/store/settings';
import {getTranslation, isLanguageLoaded, loadLanguage, loadTranslationsSync} from '@/utils/i18n';
import type {SupportedLanguage} from '@/utils/languages';

// Once shown, the message stays up for one full cycle of the loader animation
// (`whodb-loader-*` keyframes in index.css), so a switch reads as deliberate rather than a
// flash. The download runs during this time.
const SWITCHING_MESSAGE_MIN_MS = 2400;

let latestRequest = 0;
let switching = false;
const listeners = new Set<() => void>();

const setSwitching = (value: boolean) => {
    if (switching === value) return;
    switching = value;
    listeners.forEach(listener => { listener(); });
};

/** Returns whether a language switch is waiting on its translations. */
export const isSwitchingLanguage = (): boolean => switching;

/** Subscribes to changes of the language-switch state. */
export const subscribeToLanguageSwitch = (listener: () => void): (() => void) => {
    listeners.add(listener);
    return () => listeners.delete(listener);
};

/**
 * Switches the UI language once its translations have loaded, so text never drops back to
 * English mid-switch. Every language change must go through here. If the download fails the
 * current language stays and an error is shown. When called again before an earlier switch
 * finishes, only the latest call is applied. Resolves to whether this call was applied.
 */
export const changeLanguage = async (language: SupportedLanguage): Promise<boolean> => {
    const request = ++latestRequest;
    if (!isLanguageLoaded(language)) {
        setSwitching(true);
        const minimumDisplay = new Promise(resolve => window.setTimeout(resolve, SWITCHING_MESSAGE_MIN_MS));
        try {
            // A failed download rejects at once, without waiting out the minimum display time.
            await Promise.all([loadLanguage(language), minimumDisplay]);
        } catch {
            if (request === latestRequest) {
                setSwitching(false);
                const current = reduxStore.getState().settings.language;
                toast.error(getTranslation(loadTranslationsSync('components/language-switch', current), 'switchFailed', current));
            }
            return false;
        }
    }
    if (request !== latestRequest) return false;
    reduxStore.dispatch(SettingsActions.setLanguage(language));
    setSwitching(false);
    return true;
};
