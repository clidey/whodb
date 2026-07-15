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

import {reduxStore} from '../store';
import {getAnalyticsDistinctId} from '../config/posthog';

/**
 * Optional auth header provider registered by extensions.
 * When set, it fully replaces the CE credential-based header logic.
 */
let authHeaderProvider: (() => string | null) | null = null;
let extraHeadersProvider: (() => Record<string, string>) | null = null;
let asyncExtraHeadersProvider: (() => Promise<Record<string, string>>) | null = null;

type SourceCredentialValueLike = {
    Key: string;
    Value: string;
};

function mapSourceCredentialValues(values: readonly SourceCredentialValueLike[] | undefined): Record<string, string> {
    return (values ?? []).reduce<Record<string, string>>((acc, value) => {
        acc[value.Key] = value.Value;
        return acc;
    }, {});
}

/**
 * Registers an alternative auth header provider instead of CE's base64-encoded
 * database credentials.
 */
export const registerAuthHeaderProvider = (fn: () => string | null): void => {
    authHeaderProvider = fn;
};

/**
 * Registers extra request headers supplied by an extension.
 */
export const registerAuthExtraHeadersProvider = (fn: () => Record<string, string>): void => {
    extraHeadersProvider = fn;
};

/**
 * Registers async request headers supplied by an extension.
 *
 * This is used by request paths that can wait for session state before sending,
 * such as Apollo GraphQL operations.
 */
export const registerAsyncAuthExtraHeadersProvider = (fn: () => Promise<Record<string, string>>): void => {
    asyncExtraHeadersProvider = fn;
};

const analyticsHeaderName = 'X-WhoDB-Analytics-Id';
const csrfCookieName = 'whodb_csrf';
const csrfHeaderName = 'X-CSRF-Token';

/**
 * Reads the readable CSRF cookie set at login for the double-submit CSRF check.
 * Returns null when absent (e.g. desktop, or before login).
 */
function readCSRFCookie(): string | null {
    if (typeof document === 'undefined') {
        return null;
    }
    const match = document.cookie
        .split('; ')
        .find(row => row.startsWith(`${csrfCookieName}=`));
    return match ? decodeURIComponent(match.slice(csrfCookieName.length + 1)) : null;
}

/**
 * Checks if the app is running in a desktop/webview environment
 * where cookies might not be properly supported.
 */
export function isDesktopScheme(): boolean {
    // Check if Wails bindings are available - more reliable than protocol check
    if (typeof window !== 'undefined') {
        const wailsGo = (window as any).go;
        if (wailsGo?.main?.App || wailsGo?.common?.App) {
            return true;
        }
    }
    // Fallback to protocol check for compatibility
  return typeof window !== 'undefined' && !['http:', 'https:'].includes(window.location.protocol);
}

/**
 * Gets the Authorization header value for the current user session.
 * Returns null if no authentication is needed or available.
 *
 * We always send credentials via Authorization header because:
 * - Cookies have a ~4KB size limit
 * - SSL certificates in Advanced fields can exceed this limit
 * - Browser silently drops oversized cookies, causing auth failures
 * - Authorization headers have no practical size limit
 * - Credentials are stored in localStorage via Redux persist
 */
export function getAuthorizationHeader(): string | null {
  if (authHeaderProvider) {
    return authHeaderProvider();
  }
  // Browser clients authenticate via the HttpOnly session cookie (credentials are
  // stored server-side), so no Authorization header is sent. Desktop/webview
  // clients, where cookies are unreliable, keep sending base64 credentials.
  if (!isDesktopScheme()) {
    return null;
  }
  try {
    // @ts-ignore - auth state type not fully defined
    const authState = reduxStore.getState().auth;
    if (authState?.status !== 'logged-in') {
      return null;
    }

    const current = authState?.current;
    if (!current) {
      return null;
    }

    const values = mapSourceCredentialValues(current.Values);

    // For saved profiles, send only Id+Database (credentials stored server-side)
    // For inline credentials, always send full credentials for validation
    const tokenPayload = current.Saved ? {
      Id: current.Id,
      Values: current.Database ? { Database: current.Database } : {},
    } : {
      Id: current.Id,
      SourceType: current.SourceType,
      Values: values,
      AccessToken: current.AccessToken,
      IsProfile: false,
    };

    // Convert to base64 - handling Unicode properly
    // The encodeURIComponent/decodeURIComponent pattern ensures UTF-8 encoding
    const jsonString = JSON.stringify(tokenPayload);
    // Modern replacement for deprecated unescape(encodeURIComponent(...))
    const utf8Bytes = encodeURIComponent(jsonString).replace(/%([0-9A-F]{2})/g,
      (_, p1) => String.fromCharCode(parseInt(p1, 16)));
    const bearer = btoa(utf8Bytes);
    return `Bearer ${bearer}`;
  } catch (error) {
    console.error('Failed to get authorization header:', error);
    return null;
  }
}

/**
 * Adds the Authorization header to the provided headers object if needed.
 * This is used for both GraphQL and direct HTTP requests in desktop environments.
 *
 * @param headers - Existing headers object (or undefined)
 * @returns Headers object with Authorization added if needed
 */
export function addAuthHeader(headers: HeadersInit = {}): HeadersInit {
    const authHeader = getAuthorizationHeader();
    const id = getAnalyticsDistinctId()
    headers = {
        ...headers,
        [analyticsHeaderName]: id ?? ""
    }
    const csrf = readCSRFCookie();
    if (csrf) {
        headers = { ...headers, [csrfHeaderName]: csrf };
    }
    if (extraHeadersProvider) {
        headers = {
            ...headers,
            ...extraHeadersProvider(),
        };
    }
    if (authHeader) {
        return {
            ...headers,
            Authorization: authHeader,
        };
    }
    return headers;
}

/**
 * Adds authentication headers after resolving any async extension headers.
 *
 * Use this for request pipelines that support async header preparation.
 * Synchronous callers should continue using addAuthHeader().
 */
export async function addAuthHeaderAsync(headers: HeadersInit = {}): Promise<HeadersInit> {
    const authHeader = getAuthorizationHeader();
    const id = getAnalyticsDistinctId()
    headers = {
        ...headers,
        [analyticsHeaderName]: id ?? ""
    }
    const csrf = readCSRFCookie();
    if (csrf) {
        headers = { ...headers, [csrfHeaderName]: csrf };
    }
    if (asyncExtraHeadersProvider) {
        headers = {
            ...headers,
            ...await asyncExtraHeadersProvider(),
        };
    } else if (extraHeadersProvider) {
        headers = {
            ...headers,
            ...extraHeadersProvider(),
        };
    }
    if (authHeader) {
        return {
            ...headers,
            Authorization: authHeader,
        };
    }
    return headers;
}
