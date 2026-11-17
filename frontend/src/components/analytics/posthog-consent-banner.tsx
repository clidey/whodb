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
import {Button, toast} from '@clidey/ux';
import {useAppDispatch, useAppSelector} from '../../store/hooks';
import {SettingsActions} from '../../store/settings';
import {getStoredConsentState, optInUser, optOutUser} from '../../config/posthog';
import {featureFlags, getAppName} from '../../config/features';
import {useTranslation} from '../../hooks/use-translation';

/** Presents the telemetry choice using the shared UX toast stack. */
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

    useEffect(() => {
        if (!visible) return;

        const id = toast(t('title'), {
            id: 'posthog-consent-notice',
            testId: 'posthog-consent-notice',
            description: <>
                <p>{t('message', { appName })}</p>
                <div className="mt-3 flex justify-end gap-2">
                    <Button variant="ghost" size="sm" onClick={() => { void handleDecline(); }}>{t('decline')}</Button>
                    <Button size="sm" onClick={() => { void handleAllow(); }}>{t('accept')}</Button>
                </div>
            </>,
            duration: Infinity,
            closeButton: true,
            onDismiss: () => { setVisible(false); },
        });

        return () => { toast.dismiss(id); };
    }, [appName, handleAllow, handleDecline, t, visible]);

    return null;
};
