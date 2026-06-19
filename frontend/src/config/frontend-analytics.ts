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

import { trackFrontendEvent } from './posthog';
import { ANALYTICS_EVENTS, type AnalyticsEventName } from './analytics-events';

export type FrontendAnalyticsProperties = Record<string, unknown>;

/** Emits a safe frontend analytics event without blocking the current UI action. */
export const trackFrontendIntent = (event: AnalyticsEventName | string, properties?: FrontendAnalyticsProperties): void => {
    void trackFrontendEvent(event, properties);
};

/** Emits a screen-view event with low-cardinality route metadata. */
export const trackScreenViewed = (route: string, properties?: FrontendAnalyticsProperties): void => {
    trackFrontendIntent(ANALYTICS_EVENTS.UI_SCREEN_VIEWED, {
        route,
        ...properties,
    });
};

/** Emits a form-open event for funnel and abandonment analysis. */
export const trackFormOpened = (form: string, properties?: FrontendAnalyticsProperties): void => {
    trackFrontendIntent(ANALYTICS_EVENTS.UI_FORM_OPENED, {
        form,
        ...properties,
    });
};

/** Emits a form-submission intent event before the backend outcome is known. */
export const trackFormSubmitted = (form: string, properties?: FrontendAnalyticsProperties): void => {
    trackFrontendIntent(ANALYTICS_EVENTS.UI_FORM_SUBMITTED, {
        form,
        ...properties,
    });
};

/** Emits a form-abandonment event when the user leaves after changing safe form state. */
export const trackFormAbandoned = (form: string, properties?: FrontendAnalyticsProperties): void => {
    trackFrontendIntent(ANALYTICS_EVENTS.UI_FORM_ABANDONED, {
        form,
        ...properties,
    });
};

/** Emits a user option change event with a bounded field/value pair. */
export const trackOptionChanged = (field: string, selected: string | boolean | number, properties?: FrontendAnalyticsProperties): void => {
    trackFrontendIntent(ANALYTICS_EVENTS.UI_OPTION_CHANGED, {
        field,
        selected,
        ...properties,
    });
};

/** Returns a coarse count bucket for analytics properties. */
export const countBucket = (count: number): string => {
    if (!Number.isFinite(count) || count <= 0) return 'zero';
    if (count === 1) return 'one';
    if (count <= 5) return '2_5';
    if (count <= 20) return '6_20';
    if (count <= 100) return '21_100';
    if (count <= 1000) return '101_1000';
    return 'gte_1000';
};

/** Returns a coarse text-length bucket without sending the text itself. */
export const textLengthBucket = (text: string): string => {
    const length = text.trim().length;
    if (length === 0) return 'empty';
    if (length < 32) return 'lt_32';
    if (length < 128) return '32_127';
    if (length < 512) return '128_511';
    if (length < 2048) return '512_2047';
    return 'gte_2048';
};

/** Maps a raw error object to a coarse frontend analytics error code. */
export const frontendAnalyticsErrorCode = (error: unknown): string => {
    const message = error instanceof Error ? error.message : String(error ?? '');
    const lower = message.toLowerCase();
    if (lower.includes('unauthorized') || lower.includes('forbidden') || lower.includes('denied')) return 'access_denied';
    if (lower.includes('not found') || lower.includes('does not exist')) return 'not_found';
    if (lower.includes('invalid') || lower.includes('validation') || lower.includes('required')) return 'invalid_input';
    if (lower.includes('timeout') || lower.includes('deadline')) return 'timeout';
    if (lower.includes('network') || lower.includes('fetch') || lower.includes('connection') || lower.includes('econnrefused')) return 'connection_failed';
    return 'unknown';
};
