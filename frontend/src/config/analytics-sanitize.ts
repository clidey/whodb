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

import {
    SAFE_ANALYTICS_PROPERTY_KEYS,
    SAFE_ANALYTICS_PROPERTY_SUFFIXES,
} from './analytics-events';

export type SafeAnalyticsPropertyValue = string | number | boolean | null;

export type AnalyticsRuntimeContext = {
    buildEdition: string;
    buildEnvironment: string;
    appType: string;
    platform: string;
    deployment?: string | null;
};

const SENSITIVE_PROPERTY_PATTERN = /(api[_-]?key|credential|email|host|hostname|name|password|path|prompt|query|secret|sql|text|token|url|username|value)/i;

const safePropertyValue = (value: unknown): SafeAnalyticsPropertyValue | undefined => {
    if (value == null) {
        return null;
    }
    if (typeof value === 'string') {
        const trimmed = value.trim();
        if (!trimmed) {
            return undefined;
        }
        return trimmed.length > 96 ? trimmed.slice(0, 96) : trimmed;
    }
    if (typeof value === 'number') {
        return Number.isFinite(value) ? value : undefined;
    }
    if (typeof value === 'boolean') {
        return value;
    }
    return undefined;
};

const isSafePropertyKey = (key: string) => {
    if (SENSITIVE_PROPERTY_PATTERN.test(key)) {
        return key.endsWith('_present') || key.endsWith('_bucket') || key.endsWith('_count') || key.endsWith('_type');
    }
    return SAFE_ANALYTICS_PROPERTY_KEYS.has(key) || SAFE_ANALYTICS_PROPERTY_SUFFIXES.some(suffix => key.endsWith(suffix));
};

const STACK_LOCATION_PATTERN = /([^/\\\s()]+\.[cm]?js):(\d+):(\d+)/;

/**
 * Reduces an error stack to the bundle file name, line and column of the first
 * in-app frame, e.g. "index-abc123.js:120:45". Frames from other origins and the
 * message line are skipped so no URL, origin or message text is reported.
 */
export const summarizeErrorLocation = (stack: unknown, origin: string): string | undefined => {
    if (typeof stack !== 'string' || !origin) {
        return undefined;
    }
    for (const line of stack.split('\n')) {
        if (!line.includes(origin)) {
            continue;
        }
        const match = STACK_LOCATION_PATTERN.exec(line);
        if (match) {
            return `${match[1]}:${match[2]}:${match[3]}`;
        }
    }
    return undefined;
};

/** Returns the innermost component name from a React component stack. */
export const summarizeComponentStack = (componentStack: unknown): string | undefined => {
    if (typeof componentStack !== 'string') {
        return undefined;
    }
    const match = /^\s*(?:at\s+|in\s+)([A-Za-z_$][\w$.]*)/m.exec(componentStack.trim());
    return match?.[1];
};

/** Sanitizes event properties and attaches non-sensitive runtime context. */
export const sanitizeAnalyticsProperties = (
    properties: Record<string, unknown> | undefined,
    context: AnalyticsRuntimeContext
): Record<string, SafeAnalyticsPropertyValue> => {
    const base: Record<string, SafeAnalyticsPropertyValue> = {
        source: 'web',
        build_edition: context.buildEdition,
        build_environment: context.buildEnvironment,
        app_type: context.appType,
        platform: context.platform,
    };
    if (context.deployment) {
        base.deployment = context.deployment;
    }

    for (const [key, value] of Object.entries(properties ?? {})) {
        if (!isSafePropertyKey(key)) {
            continue;
        }
        const safeValue = safePropertyValue(value);
        if (safeValue !== undefined) {
            base[key] = safeValue;
        }
    }
    return base;
};

/** Sanitizes person or group properties without adding event runtime context. */
export const sanitizeAnalyticsIdentityProperties = (
    properties?: Record<string, unknown>
): Record<string, SafeAnalyticsPropertyValue> => {
    const safe: Record<string, SafeAnalyticsPropertyValue> = {};
    for (const [key, value] of Object.entries(properties ?? {})) {
        if (!isSafePropertyKey(key)) {
            continue;
        }
        const safeValue = safePropertyValue(value);
        if (safeValue !== undefined) {
            safe[key] = safeValue;
        }
    }
    return safe;
};
