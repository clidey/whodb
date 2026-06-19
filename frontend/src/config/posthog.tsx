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

import type {PostHog} from 'posthog-js';
import type * as PostHogModule from 'posthog-js';
import {featureFlags} from './features';
import {getEdition} from './edition';
import { ANALYTICS_EVENTS } from './analytics-events';
import {
    sanitizeAnalyticsIdentityProperties,
    sanitizeAnalyticsProperties,
    type AnalyticsRuntimeContext,
    type SafeAnalyticsPropertyValue,
} from './analytics-sanitize';

type ConsentState = 'granted' | 'denied' | 'unknown';

const CONSENT_STORAGE_KEY = 'whodb.analytics.consent';
const DISTINCT_ID_STORAGE_KEY = 'whodb.analytics.distinct_id';
const SESSION_REPLAY_SAMPLE_KEY = 'whodb.analytics.session_replay_sampled';

type AnalyticsGroupIdentity = {
    type: string;
    key: string;
    properties?: Record<string, unknown>;
};

type AnalyticsUserIdentity = {
    distinctId: string;
    properties?: Record<string, unknown>;
    groups?: AnalyticsGroupIdentity[];
};

const isDesktopApp = () => (
    typeof window !== 'undefined' &&
    (!!(window as any).go?.main?.App || !!(window as any).go?.common?.App)
);

let posthogModulePromise: Promise<typeof PostHogModule> | null = null;
let initPromise: Promise<PostHog | null> | null = null;
let activeClient: PostHog | null = null;
let handlersRegistered = false;
let cachedDistinctId: string | null = null;

const configuredDeploymentName = import.meta.env.VITE_POSTHOG_DEPLOYMENT?.trim();

let deploymentName: string | null = configuredDeploymentName?.length ? configuredDeploymentName : null;

const posthogKey = "phc_hbXcCoPTdxm5ADL8PmLSYTIUvS6oRWFM2JAK8SMbfnH";
const apiHost = "https://z.clidey.com";
const HOSTED_SESSION_REPLAY_SAMPLE_RATE = 0.1;
const getEnvEnvironment = () => import.meta.env.MODE ?? 'development';
const getBuildEdition = () => getEdition();
const isE2ETest = () => import.meta.env.VITE_E2E_TEST === 'true';
const remoteAnalyticsEnabled = () => import.meta.env.VITE_POSTHOG_ENABLED === 'true';
const isSessionReplayEnabled = () => getBuildEdition() === 'ee' && deploymentName === 'whodb-platform-prod';
const sessionReplaySampled = () => {
    if (typeof window === 'undefined') {
        return false;
    }
    try {
        const stored = window.sessionStorage?.getItem(SESSION_REPLAY_SAMPLE_KEY);
        if (stored === 'true') {
            return true;
        }
        if (stored === 'false') {
            return false;
        }
        const sampled = Math.random() < HOSTED_SESSION_REPLAY_SAMPLE_RATE;
        window.sessionStorage?.setItem(SESSION_REPLAY_SAMPLE_KEY, String(sampled));
        return sampled;
    } catch {
        return Math.random() < HOSTED_SESSION_REPLAY_SAMPLE_RATE;
    }
};
const shouldRecordSession = (consent: ConsentState) => consent === 'granted' && isSessionReplayEnabled() && sessionReplaySampled();

const localHostPattern = /^(localhost|127(?:\.\d{1,3}){3}|0\.0\.0\.0|\[?::1\]?|.+\.local)$/i;
const privateHostPattern = /^(10(?:\.\d{1,3}){3}|192\.168(?:\.\d{1,3}){2}|172\.(?:1[6-9]|2\d|3[01])(?:\.\d{1,3}){2})$/;

const isLocalAnalyticsHost = () => {
    if (typeof window === 'undefined') {
        return true;
    }
    const hostname = window.location.hostname;
    return localHostPattern.test(hostname) || privateHostPattern.test(hostname);
};

const remoteAnalyticsAllowed = () => {
    if (isE2ETest() || !remoteAnalyticsEnabled()) {
        return false;
    }
    return getEnvEnvironment() === 'production' && !isLocalAnalyticsHost();
};

const analyticsRuntimeContext = (): AnalyticsRuntimeContext => ({
    buildEdition: getBuildEdition(),
    buildEnvironment: getEnvEnvironment(),
    appType: isDesktopApp() ? 'desktop' : 'web',
    platform: isDesktopApp() ? 'wails' : 'browser',
    deployment: deploymentName,
});

const sanitizeEventProperties = (properties?: Record<string, unknown>) => sanitizeAnalyticsProperties(properties, analyticsRuntimeContext());

const debugSinkEnabled = () => !remoteAnalyticsAllowed();

const emitDebugAnalytics = (
    kind: 'event' | 'identify' | 'group',
    name: string,
    properties: Record<string, SafeAnalyticsPropertyValue>
) => {
    if (!debugSinkEnabled()) {
        return;
    }
    if (typeof window !== 'undefined') {
        const debugWindow = window as typeof window & {
            __WHODB_ANALYTICS_DEBUG__?: Array<{
                kind: string;
                name: string;
                properties: Record<string, SafeAnalyticsPropertyValue>;
            }>;
        };
        debugWindow.__WHODB_ANALYTICS_DEBUG__ ??= [];
        debugWindow.__WHODB_ANALYTICS_DEBUG__.push({ kind, name, properties });
    }
    // eslint-disable-next-line no-console
    console.info('[analytics]', kind, name, properties);
};

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
        cachedDistinctId = window.localStorage?.getItem(DISTINCT_ID_STORAGE_KEY) ?? null;
    } catch (e) {
        console.warn('Failed to load distinct ID from localStorage:', e);
    }
    return cachedDistinctId;
};

const ensurePosthogModule = async () => {
    posthogModulePromise ??= import('posthog-js').catch(err => {
        console.warn('Failed to load PostHog module:', err);
        throw err;
    });
    return posthogModulePromise;
};

const registerContext = (client: PostHog) => {
    if (typeof window === 'undefined') {
        return;
    }
    const domain = window.location.hostname || 'localhost';

    const isDesktop = isDesktopApp();

    const properties: Record<string, string> = {
        site_domain: domain,
        build_environment: getEnvEnvironment(),
        build_edition: getBuildEdition(),
        app_type: isDesktop ? 'desktop' : 'web',
        platform: isDesktop ? 'wails' : 'browser',
        session_replay_enabled: String(isSessionReplayEnabled()),
    };
    if (deploymentName) {
        properties.deployment = deploymentName;
    }
    client.register(properties);
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
    if (!featureFlags.sampleDatabaseTour) {
        return null;
    }
    if (isE2ETest()) {
        return null;
    }
    if (!remoteAnalyticsAllowed()) {
        activeClient = null;
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

        // Debug logging for desktop environments
        const isDesktop = isDesktopApp();

        const enabled = consent === 'granted';

        posthog.init(posthogKey, {
            api_host: apiHost,
            defaults: "2026-01-30",
            capture_pageleave: enabled,
            persistence: 'localStorage+cookie',
            enable_recording_console_log: enabled,
            autocapture: enabled,
            capture_pageview: enabled,
            enable_heatmaps: enabled,
            //@ts-ignore session replay options are available in the installed SDK runtime.
            disable_session_recording: !shouldRecordSession(consent),
            //@ts-ignore session replay options are available in the installed SDK runtime.
            session_recording: {
                maskAllInputs: true,
                maskTextSelector: '*',
                blockSelector: '[data-ph-no-capture], [data-sensitive], .ph-no-capture',
            },
            disable_web_experiments: enabled,
            disable_surveys: enabled,
            //@ts-ignore
            opt_out_capturing_by_default: enabled,
            loaded: (client) => {
                activeClient = client as PostHog;
                registerContext(client as PostHog);
                registerGlobalHandlers(client as PostHog);

                if (enabled) {
                    client.opt_in_capturing();
                    //@ts-ignore
                } else if (consent === 'denied') {
                    client.opt_out_capturing();
                }

                persistDistinctId(client.get_distinct_id());
                if (shouldRecordSession(consent)) {
                    client.startSessionRecording?.();
                } else {
                    client.stopSessionRecording?.();
                }

                // Log successful initialization for desktop
                if (isDesktop) {
                    // Track desktop app launch
                    if (consent === 'granted') {
                        client.capture(ANALYTICS_EVENTS.DESKTOP_APP_LAUNCHED, sanitizeEventProperties({
                            platform: 'wails',
                        }));
                    }
                }
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

    const safeProperties = sanitizeEventProperties(properties);
    emitDebugAnalytics('event', event, safeProperties);
    if (!remoteAnalyticsAllowed()) {
        return;
    }

    try {
        const client = await ensureInitializedClient();
        client?.capture(event, safeProperties);
    } catch {
        // do nothing
    }
};

/** Identifies the current analytics person and optional group memberships. */
export const identifyAnalyticsUser = async (identity: AnalyticsUserIdentity): Promise<boolean> => {
    const distinctId = identity.distinctId.trim();
    if (!distinctId || getStoredConsentState() !== 'granted') {
        return false;
    }

    const identityProperties = sanitizeAnalyticsIdentityProperties(identity.properties);
    emitDebugAnalytics('identify', distinctId, identityProperties);
    for (const group of identity.groups ?? []) {
        const groupType = group.type.trim();
        const groupKey = group.key.trim();
        if (!groupType || !groupKey) {
            continue;
        }
        emitDebugAnalytics('group', `${groupType}:${groupKey}`, sanitizeAnalyticsIdentityProperties(group.properties));
    }
    if (!remoteAnalyticsAllowed()) {
        return true;
    }

    try {
        const client = await ensureInitializedClient();
        if (!client) {
            return false;
        }

        client.identify(distinctId, identityProperties);
        for (const group of identity.groups ?? []) {
            const groupType = group.type.trim();
            const groupKey = group.key.trim();
            if (!groupType || !groupKey) {
                continue;
            }
            const groupProperties = sanitizeAnalyticsIdentityProperties(group.properties);
            client.group(groupType, groupKey, groupProperties);
        }
        persistDistinctId(client.get_distinct_id());
        return true;
    } catch {
        // best-effort — never block app auth on analytics
        return false;
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
        // Stop all automatic capturing features
        client.opt_out_capturing();
        client.stopSessionRecording?.();
        client.config.autocapture = false;
        client.config.capture_pageview = false;
        client.config.capture_pageleave = false;
        client.config.enable_heatmaps = false;
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
    client.config.autocapture = true;
    client.config.capture_pageview = true;
    client.config.capture_pageleave = true;
    client.config.enable_heatmaps = true;
    client.opt_in_capturing();
    if (shouldRecordSession('granted')) {
        client.startSessionRecording?.();
    } else {
        client.stopSessionRecording?.();
    }
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

/** Capture an exception to PostHog if the client is initialized and consent is granted. */
export const captureException = async (error: unknown, properties?: Record<string, unknown>) => {
    if (getStoredConsentState() !== 'granted') {
        return;
    }
    try {
        const client = await ensureInitializedClient();
        if (client) {
            captureClientException(client, error, properties ?? {});
        }
    } catch {
        // best-effort — never throw from error reporting
    }
};

export const setDeploymentName = (name: string) => {
    deploymentName = name;
};
