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

import {expect} from "@playwright/test";

/**
 * Named timeouts for Playwright locator waits.
 * Use these instead of raw numbers so values are tunable from one place.
 */
export const TIMEOUT = Object.freeze({
    /** Short wait — animations, preview loads (3s) */
    SHORT:      3_000,
    /** Element interaction — buttons, dialogs, menus, popovers (5s) */
    ELEMENT:    5_000,
    /** Default action — data load, table render, API response (10s) */
    ACTION:    10_000,
    /** Navigation — page load, storage-unit cards after route change (15s) */
    NAVIGATION:15_000,
    /** Slow operation — login completion, async DB mutations, mock data gen (30s) */
    SLOW:      30_000,
    /** Graph metadata — CockroachDB column discovery can exceed slow waits (90s) */
    GRAPH:     90_000,
    /** Login API — full auth flow including potential retries (60s) */
    LOGIN:     60_000,
});

let uniqueIdCounter = 0;

/**
 * Generates a unique test identifier for this test run.
 * Uses timestamp to avoid conflicts within and across test runs.
 */
export function getUniqueTestId() {
    uniqueIdCounter += 1;
    return `test_${Date.now()}_${uniqueIdCounter}`;
}

/**
 * Sets up a listener for a GraphQL mutation response and returns a function to
 * await and verify the result.
 *
 * Usage:
 *   const verify = waitForMutation(page, 'AddRow');
 *   await whodb.addRow(data);
 *   await verify();
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} operationName - GraphQL operation name (e.g. 'AddRow', 'DeleteRow', 'UpdateStorageUnit')
 * @returns {() => Promise<object>} Async function that awaits the response and asserts no errors. Returns parsed JSON.
 */
export function waitForMutation(page, operationName) {
    const responsePromise = page.waitForResponse(
        resp =>
            resp.url().includes('/api/query') &&
            resp.request().postDataJSON?.()?.operationName === operationName,
        { timeout: TIMEOUT.SLOW }
    );

    return async () => {
        const response = await responsePromise;
        const result = await response.json();
        expect(result.errors, `${operationName} mutation should succeed`).toBeUndefined();
        return result;
    };
}

/**
 * Reads the backend-owned session for the currently logged-in profile via the
 * SourceSession GraphQL query. Auth state now lives server-side (session
 * cookie) rather than in `persist:auth` localStorage, so tests that need to
 * check "what profile/database is currently active" must ask the backend
 * instead of reading localStorage.
 *
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<{id: string | null, sourceType: string, database: string} | null>}
 */
export async function fetchSourceSession(page) {
    return page.evaluate(async () => {
        const csrfMatch = document.cookie
            .split('; ')
            .find(row => row.startsWith('whodb_csrf='));
        const csrf = csrfMatch ? decodeURIComponent(csrfMatch.slice('whodb_csrf='.length)) : null;

        const response = await fetch('/api/query', {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
                ...(csrf ? { 'X-Csrf-Token': csrf } : {}),
            },
            body: JSON.stringify({
                operationName: 'SourceSession',
                query: 'query SourceSession { SourceSession { id: Id sourceType: SourceType database: Database } }',
                variables: {},
            }),
        });
        const result = await response.json();
        return result?.data?.SourceSession ?? null;
    });
}
