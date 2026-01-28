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

import {useCallback, useEffect, useState} from 'react';
import {Button, cn} from '@clidey/ux';
import {useAppDispatch, useAppSelector} from '../../store/hooks';
import {SettingsActions} from '../../store/settings';
import {getStoredConsentState, optInUser, optOutUser} from '../../config/posthog';
import {isEEMode} from '../../config/ee-imports';
import {useTranslation} from '../../hooks/use-translation';

export const PosthogConsentBanner = () => {
    const { t } = useTranslation('components/posthog-consent-banner');
    const dispatch = useAppDispatch();
    const metricsEnabled = useAppSelector((state) => state.settings.metricsEnabled);
    const [visible, setVisible] = useState(false);

    useEffect(() => {
        if (isEEMode) {
            setVisible(false);
            return;
        }
        // Hide consent banner during E2E tests
        if (import.meta.env.VITE_E2E_TEST === 'true') {
            setVisible(false);
            return;
        }
        setVisible(getStoredConsentState() === 'unknown');
    }, [metricsEnabled]);

    const handleDecline = useCallback(async () => {
        await optOutUser();
        dispatch(SettingsActions.setMetricsEnabled(false));
        setVisible(false);
    }, [dispatch]);

    const handleAllow = useCallback(async () => {
        await optInUser();
        dispatch(SettingsActions.setMetricsEnabled(true));
        setVisible(false);
    }, [dispatch]);

    if (!visible) {
        return null;
    }

    return (
        <div className="fixed bottom-3 left-1/2 z-50 w-full max-w-xl -translate-x-1/2 px-4">
            <div
                className={cn(
                    'rounded-lg border border-neutral-200 bg-background/95 p-4 shadow-xl',
                    'backdrop-blur supports-[backdrop-filter]:bg-background/80',
                    'dark:border-neutral-800'
                )}
            >
                <div className="flex flex-col gap-3 text-sm">
                    <div>
                        <p className="text-base font-semibold">{t('title')}</p>
                        <p className="text-muted-foreground mt-1 leading-relaxed">
                            {t('message')}
                        </p>
                    </div>
                    <div className="flex flex-wrap justify-end gap-2">
                        <Button variant="ghost" size="sm" onClick={handleDecline}>
                            {t('decline')}
                        </Button>
                        <Button size="sm" onClick={handleAllow}>
                            {t('accept')}
                        </Button>
                    </div>
                </div>
            </div>
        </div>
    );
};
