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

import { reduxStore } from '../store';

/**
 * Checks if the app is running in a desktop/webview environment
 * where cookies might not be properly supported.
 */
function isDesktopScheme(): boolean {
  return typeof window !== 'undefined' && !['http:', 'https:'].includes(window.location.protocol);
}

/**
 * Gets the Authorization header value for the current user session.
 * Returns null if no authentication is needed or available.
 *
 * In desktop/webview environments (wails://), cookies don't work reliably,
 * so we need to pass credentials via Authorization header.
 */
export function getAuthorizationHeader(): string | null {
  try {
    // Only attach for non-HTTP(s) schemes (e.g., wails://)
    if (!isDesktopScheme()) {
      return null;
    }

    // @ts-ignore - auth state type not fully defined
    const authState = reduxStore.getState().auth;
    const current = authState?.current;
    if (!current) {
      return null;
    }

    // For saved profiles, send only Id+Database (credentials stored server-side)
    // For inline credentials, always send full credentials for validation
    const tokenPayload = current.Saved ? {
      Id: current.Id,
      Database: current.Database,
    } : {
      Id: current.Id,
      Type: current.Type,
      Hostname: current.Hostname,
      Username: current.Username,
      Password: current.Password,
      Database: current.Database,
      Advanced: current.Advanced || [],
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
  if (authHeader) {
    return {
      ...headers,
      Authorization: authHeader,
    };
  }
  return headers;
}