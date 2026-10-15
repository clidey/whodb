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
import {featureFlags, getAppName} from '../../config/features';
import {useTranslation} from '../../hooks/use-translation';
import {XMarkIcon} from '../heroicons';

export const PosthogConsentBanner = () => {
    const { t } = useTranslation('components/posthog-consent-banner');
    const appName = getAppName();
    const dispatch = useAppDispatch();
    const metricsEnabled = useAppSelector((state) => state.settings.metricsEnabled);
    const [visible, setVisible] = useState(false);

    useEffect(() => {
        if (!featureFlags.settingsPage) {
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
        <div className="ce-telemetry-notice fixed bottom-4 right-4 z-50 w-[min(360px,calc(100vw-32px))]">
            <div
                className={cn(
                    'relative rounded-lg border border-border bg-card p-4 shadow-xl'
                )}
            >
                <Button variant="ghost" size="icon" onClick={() => { setVisible(false); }} className="absolute right-1 top-1 size-7" aria-label={t('dismiss')}><XMarkIcon className="size-3" /></Button>
                <div className="flex flex-col gap-3 pr-2 text-sm">
                    <div>
                        <p className="text-sm font-semibold">{t('title')}</p>
                        <p className="text-muted-foreground mt-1 text-xs leading-relaxed">
                            {t('message', { appName })}
                        </p>
                    </div>
                    <div className="flex flex-wrap justify-end gap-2">
                        <Button variant="ghost" size="sm" onClick={() => { void handleDecline(); }}>
                            {t('decline')}
                        </Button>
                        <Button size="sm" onClick={() => { void handleAllow(); }}>
                            {t('accept')}
                        </Button>
                    </div>
                </div>
            </div>
        </div>
    );
};
