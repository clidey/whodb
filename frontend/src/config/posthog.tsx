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

type AnalyticsProfileLike = {
    Id?: string;
    Type?: string;
    Hostname?: string;
    Username?: string;
    Database?: string;
    Saved?: boolean;
    IsEnvironmentDefined?: boolean;
};

const CONSENT_STORAGE_KEY = 'whodb.analytics.consent';
const DISTINCT_ID_STORAGE_KEY = 'whodb.analytics.distinct_id';

let posthogModulePromise: Promise<typeof import('posthog-js')> | null = null;
let initPromise: Promise<PostHog | null> | null = null;
let activeClient: PostHog | null = null;
let handlersRegistered = false;
let cachedDistinctId: string | null = null;

const posthogKey = "phc_hbXcCoPTdxm5ADL8PmLSYTIUvS6oRWFM2JAK8SMbfnH"
const apiHost = "https://us.i.posthog.com"
const getEnvEnvironment = () => import.meta.env.MODE ?? 'development';
const getBuildEdition = () => import.meta.env.VITE_BUILD_EDITION ?? 'ce';

const getStoredConsent = (): ConsentState => {
    if (typeof window === 'undefined') {
        return 'unknown';
    }
    const stored = window.localStorage.getItem(CONSENT_STORAGE_KEY);
    if (stored === 'granted' || stored === 'denied') {
        return stored;
    }
    return 'unknown';
};

const persistConsent = (consent: ConsentState) => {
    if (typeof window === 'undefined') {
        return;
    }
    if (consent === 'unknown') {
        window.localStorage.removeItem(CONSENT_STORAGE_KEY);
    } else {
        window.localStorage.setItem(CONSENT_STORAGE_KEY, consent);
    }
};

const persistDistinctId = (distinctId: string | null) => {
    cachedDistinctId = distinctId;
    if (typeof window === 'undefined') {
        return;
    }
    if (distinctId) {
        window.localStorage.setItem(DISTINCT_ID_STORAGE_KEY, distinctId);
    } else {
        window.localStorage.removeItem(DISTINCT_ID_STORAGE_KEY);
    }
};

const loadStoredDistinctId = (): string | null => {
    if (cachedDistinctId) {
        return cachedDistinctId;
    }
    if (typeof window === 'undefined') {
        return null;
    }
    cachedDistinctId = window.localStorage.getItem(DISTINCT_ID_STORAGE_KEY);
    return cachedDistinctId;
};

const ensurePosthogModule = async () => {
    if (!posthogModulePromise) {
        posthogModulePromise = import('posthog-js');
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
};

const normalizeIdentityComponent = (value: unknown, fallback: string): string => {
    if (typeof value !== 'string') {
        return fallback;
    }
    const normalized = value.trim().toLowerCase();
    return normalized.length > 0 ? normalized : fallback;
};

const hashString = async (value: string): Promise<string> => {
    if (value.length === 0) {
        return '';
    }

    const encoder = new TextEncoder();
    const data = encoder.encode(value);

    if (typeof crypto !== 'undefined' && crypto.subtle) {
        const digest = await crypto.subtle.digest('SHA-256', data);
        return Array.from(new Uint8Array(digest))
            .map((byte) => byte.toString(16).padStart(2, '0'))
            .join('');
    }

    console.warn('Secure hashing unavailable; analytics identity may not align with backend data.');
    return value;
};

const buildProfileHash = async (profile?: AnalyticsProfileLike | null): Promise<string | null> => {
    if (!profile) {
        return null;
    }

    const id = typeof profile.Id === 'string' ? profile.Id.trim() : '';
    const hostname = typeof profile.Hostname === 'string' ? profile.Hostname.trim() : '';
    const username = typeof profile.Username === 'string' ? profile.Username.trim() : '';

    if (id && !hostname && !username) {
        return hashString(id);
    }

    const components = [
        normalizeIdentityComponent(profile.Type, 'unknown'),
        normalizeIdentityComponent(hostname, 'localhost'),
        normalizeIdentityComponent(username, 'anonymous'),
        normalizeIdentityComponent(profile.Database, 'default'),
    ];

    return hashString(components.join('|'));
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
        if (activeClient) {
            try {
                activeClient.opt_out_capturing();
                activeClient.reset();
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
            autocapture: true,
            capture_pageview: 'history_change',
            capture_pageleave: true,
            persistence: 'localStorage+cookie',
            cross_subdomain_cookie: true,
            debug: import.meta.env.DEV,
            session_recording: {
                maskAllInputs: false,
                // @ts-ignore
                recordCanvas: true,
            },
            enable_recording_console_log: true,
            disable_surveys: false,
            opt_out_capturing_by_default: consent === 'denied',
            error_tracking: {},
            loaded: (client) => {
                activeClient = client;
                registerContext(client);
                registerGlobalHandlers(client);

                if (consent === 'granted') {
                    client.opt_in_capturing();
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

export const optOutUser = async (): Promise<void> => {
    persistConsent('denied');
    const client = activeClient ?? await ensureInitializedClient();
    if (client) {
        try {
            client.opt_out_capturing();
            client.reset();
        } catch {
            // best-effort shutdown
        }
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

export const identifyProfile = async (profile?: AnalyticsProfileLike | null): Promise<void> => {
    const client = await ensureInitializedClient();
    if (!client) {
        return;
    }

    if (!profile) {
        client.unregister('current_profile_type');
        client.unregister('current_profile_saved');
        client.unregister('current_profile_hash');
        client.unregister('current_profile_host_hash');
        client.unregister('current_profile_database_hash');
        persistDistinctId(client.get_distinct_id());
        return;
    }

    const distinctId = client.get_distinct_id();
    const profileHash = await buildProfileHash(profile);
    const hashedHost = profile?.Hostname ? await hashString(profile.Hostname) : '';
    const hashedDatabase = profile?.Database ? await hashString(profile.Database) : '';

    const traits: Record<string, unknown> = {
        last_used_profile_type: profile.Type ?? 'unknown',
        last_used_profile_saved: Boolean(profile.Saved),
    };

    if (profile.IsEnvironmentDefined) {
        traits.environment_defined_profile = true;
    }
    if (profileHash) {
        traits.last_used_profile_hash = profileHash;
    }
    if (hashedHost) {
        traits.last_used_host_hash = hashedHost;
    }
    if (hashedDatabase) {
        traits.last_used_database_hash = hashedDatabase;
    }

    client.identify(distinctId, traits);
    client.register({
        current_profile_type: profile.Type ?? 'unknown',
        current_profile_saved: Boolean(profile.Saved),
        ...(profileHash ? {current_profile_hash: profileHash} : {}),
        ...(hashedHost ? {current_profile_host_hash: hashedHost} : {}),
        ...(hashedDatabase ? {current_profile_database_hash: hashedDatabase} : {}),
    });

    persistDistinctId(distinctId);
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
