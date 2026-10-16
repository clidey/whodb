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

import {type FC, type ReactNode, useEffect, useState, useSyncExternalStore} from 'react';
import {createPortal} from 'react-dom';
import {Spinner} from '@/components/loading';
import {useTranslation} from '@/hooks/use-translation';
import {useAppSelector} from '@/store/hooks';
import {isSwitchingLanguage, subscribeToLanguageSwitch} from '@/utils/change-language';
import {isLanguageLoaded, loadLanguage} from '@/utils/i18n';

/** Full-screen message shown while changeLanguage downloads the new language. */
export const LanguageSwitchOverlay: FC = () => {
    const switching = useSyncExternalStore(subscribeToLanguageSwitch, isSwitchingLanguage);
    const {t} = useTranslation('components/language-switch');
    if (!switching) return null;
    return createPortal(
        <div
            className="fixed inset-0 z-[10000] flex flex-col items-center justify-center gap-3 bg-background/80 backdrop-blur-sm"
            role="status"
            aria-live="polite"
            data-testid="language-switch-overlay"
        >
            <Spinner aria-hidden="true" className="size-10" />
            <p className="text-sm text-muted-foreground">{t('switchingLanguage')}</p>
        </div>,
        document.body,
    );
};

/**
 * Holds the first render until the saved language's translations have loaded, so a
 * returning user never sees English first. The placeholder has no text for the same reason.
 * If the download fails (for example offline before the language was cached) the app renders
 * with default-language fallbacks instead of waiting forever. Later language changes that did
 * not go through changeLanguage are loaded here too, without unmounting the app.
 */
export const LanguageGate: FC<{children: ReactNode}> = ({children}) => {
    const language = useAppSelector(state => state.settings.language);
    const [ready, setReady] = useState(() => isLanguageLoaded(language));

    useEffect(() => {
        if (isLanguageLoaded(language)) {
            setReady(true);
            return;
        }
        let active = true;
        loadLanguage(language)
            .catch(() => undefined)
            .finally(() => {
                if (active) setReady(true);
            });
        return () => {
            active = false;
        };
    }, [language]);

    if (!ready) {
        return (
            <div className="flex h-screen w-screen items-center justify-center bg-background">
                <Spinner aria-hidden="true" className="size-10" />
            </div>
        );
    }
    return children;
};
