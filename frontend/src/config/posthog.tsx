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

import type {PostHog} from 'posthog-js';
import {isEEMode} from './ee-imports';

type ConsentState = 'granted' | 'denied' | 'unknown';

const CONSENT_STORAGE_KEY = 'whodb.analytics.consent';
const DISTINCT_ID_STORAGE_KEY = 'whodb.analytics.distinct_id';

let posthogModulePromise: Promise<typeof import('posthog-js')> | null = null;
let initPromise: Promise<PostHog | null> | null = null;
let activeClient: PostHog | null = null;
let handlersRegistered = false;
let cachedDistinctId: string | null = null;

const posthogKey = "phc_hbXcCoPTdxm5ADL8PmLSYTIUvS6oRWFM2JAK8SMbfnH";
const apiHost = "https://us.i.posthog.com";
const getEnvEnvironment = () => import.meta.env.MODE ?? 'development';
const getBuildEdition = () => import.meta.env.VITE_BUILD_EDITION ?? 'ce';

const getStoredConsent = (): ConsentState => {
    if (typeof window === 'undefined') {
        return 'unknown';
    }
    try {
        const stored = window.localStorage?.getItem(CONSENT_STORAGE_KEY);
        if (stored === 'granted' || stored === 'denied') {
            return stored;
        }
    } catch (e) {
        console.warn('Failed to access localStorage for consent:', e);
    }
    return 'unknown';
};

const persistConsent = (consent: ConsentState) => {
    if (typeof window === 'undefined') {
        return;
    }
    try {
        if (consent === 'unknown') {
            window.localStorage?.removeItem(CONSENT_STORAGE_KEY);
        } else {
            window.localStorage?.setItem(CONSENT_STORAGE_KEY, consent);
        }
    } catch (e) {
        console.warn('Failed to persist consent to localStorage:', e);
    }
};

const persistDistinctId = (distinctId: string | null) => {
    cachedDistinctId = distinctId;
    if (typeof window === 'undefined') {
        return;
    }
    try {
        if (distinctId) {
            window.localStorage?.setItem(DISTINCT_ID_STORAGE_KEY, distinctId);
        } else {
            window.localStorage?.removeItem(DISTINCT_ID_STORAGE_KEY);
        }
    } catch (e) {
        console.warn('Failed to persist distinct ID to localStorage:', e);
    }
};

const loadStoredDistinctId = (): string | null => {
    if (cachedDistinctId) {
        return cachedDistinctId;
    }
    if (typeof window === 'undefined') {
        return null;
    }
    try {
        cachedDistinctId = window.localStorage?.getItem(DISTINCT_ID_STORAGE_KEY) || null;
    } catch (e) {
        console.warn('Failed to load distinct ID from localStorage:', e);
    }
    return cachedDistinctId;
};

const ensurePosthogModule = async () => {
    if (!posthogModulePromise) {
        posthogModulePromise = import('posthog-js').catch(err => {
            console.warn('Failed to load PostHog module:', err);
            throw err;
        });
    }
    return posthogModulePromise;
};

const registerContext = (client: PostHog) => {
    if (typeof window === 'undefined') {
        return;
    }
    const domain = window.location.hostname || 'localhost';
    client.register({
        site_domain: domain,
        build_environment: getEnvEnvironment(),
        build_edition: getBuildEdition(),
    });
};

const captureClientException = (client: PostHog, error: unknown, properties: Record<string, unknown>) => {
    try {
        client.captureException(error, properties);
    } catch (captureError) {
        console.warn('PostHog exception capture failed', captureError);
    }
};

const registerGlobalHandlers = (client: PostHog) => {
    if (handlersRegistered || typeof window === 'undefined') {
        return;
    }
    handlersRegistered = true;

    // Delay handler registration to ensure Wails is fully initialized
    setTimeout(() => {
        try {
            window.addEventListener('error', (event) => {
                if (!event?.error) {
                    return;
                }
                captureClientException(client, event.error, {source: 'window.error'});
            });

            window.addEventListener('unhandledrejection', (event) => {
                if (!event) {
                    return;
                }
                const reason = event.reason instanceof Error ? event.reason : new Error(String(event.reason ?? 'unknown rejection'));
                captureClientException(client, reason, {source: 'window.unhandledrejection'});
            });
        } catch (e) {
            console.warn('Failed to register global error handlers:', e);
        }
    }, 100);
};

const ensureInitializedClient = async (): Promise<PostHog | null> => {
    if (activeClient) {
        return activeClient;
    }
    if (initPromise) {
        return initPromise;
    }
    if (isEEMode) {
        return null;
    }
    if (!posthogKey) {
        return null;
    }

    const consent = getStoredConsent();
    if (consent === 'denied') {
        persistDistinctId(null);
        const existingClient = activeClient;
        if (existingClient) {
            try {
                // @ts-ignore
                existingClient.opt_out_capturing();
                // @ts-ignore
                existingClient.reset();
            } catch {
                // ignore errors during shutdown
            }
        }
        activeClient = null;
        return null;
    }

    initPromise = (async () => {
        const {default: posthog} = await ensurePosthogModule();

        posthog.init(posthogKey, {
            api_host: apiHost,
            capture_pageleave: true,
            persistence: 'localStorage+cookie',
            enable_recording_console_log: true,
            //@ts-ignore
            opt_out_capturing_by_default: consent === 'denied',
            loaded: (client) => {
                activeClient = client;
                registerContext(client);
                registerGlobalHandlers(client);

                if (consent === 'granted') {
                    client.opt_in_capturing();
                    //@ts-ignore
                } else if (consent === 'denied') {
                    client.opt_out_capturing();
                }

                persistDistinctId(client.get_distinct_id());
            },
        });

        // posthog.init invokes loaded synchronously, so activeClient should now be set.
        activeClient = posthog;
        return activeClient;
    })()
        .catch((error) => {
            console.warn('PostHog initialization failed', error);
            activeClient = null;
            return null;
        })
        .finally(() => {
            // Allow subsequent callers to rely on the activeClient cache instead of the init promise.
            initPromise = null;
        });

    return initPromise;
};

export const initPosthog = async (): Promise<PostHog | null> => {
    return ensureInitializedClient();
};

export const getStoredConsentState = (): ConsentState => getStoredConsent();

export const trackFrontendEvent = async (event: string, properties?: Record<string, unknown>) => {
    if (!event) {
        return;
    }

    if (getStoredConsentState() !== 'granted') {
        return;
    }

    try {
        const client = await ensureInitializedClient();
        client?.capture(event, properties ?? {});
    } catch (error) {
        // do nothing
    }
};

export const optOutUser = async (): Promise<void> => {
    persistConsent('denied');
    const client = activeClient ?? await ensureInitializedClient();
    if (!client) {
        activeClient = null;
        persistDistinctId(null);
        return;
    }

    try {
        client.opt_out_capturing();
        client.reset();
    } catch {
        // best-effort shutdown
    }

    activeClient = null;
    persistDistinctId(null);
};

export const optInUser = async (): Promise<void> => {
    persistConsent('granted');
    const client = await ensureInitializedClient();
    if (!client) {
        return;
    }
    client.opt_in_capturing();
    persistDistinctId(client.get_distinct_id());
};

export const resetAnalyticsIdentity = async (): Promise<void> => {
    const client = await ensureInitializedClient();
    if (!client) {
        persistDistinctId(null);
        return;
    }

    client.reset();
    persistDistinctId(client.get_distinct_id());
};

export const getAnalyticsDistinctId = (): string | null => {
    return loadStoredDistinctId();
};
