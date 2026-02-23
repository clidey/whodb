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
 * Generates a unique test identifier for this test run.
 * Uses timestamp to avoid conflicts within and across test runs.
 */
export function getUniqueTestId() {
    return `test_${Date.now()}`;
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
    const responsePromise = page.waitForResponse(resp =>
        resp.url().includes('/api/query') &&
        resp.request().postDataJSON?.()?.operationName === operationName
    );

    return async () => {
        const response = await responsePromise;
        const result = await response.json();
        expect(result.errors, `${operationName} mutation should succeed`).toBeUndefined();
        return result;
    };
}
