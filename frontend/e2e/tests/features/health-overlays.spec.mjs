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

import { test, expect, forEachDatabase } from '../../support/test-fixture.mjs';
import { TIMEOUT } from '../../support/helpers/test-utils.mjs';

test.describe('Health Check Overlays', () => {

    // Health overlays are global UI behavior — only test with Postgres
    forEachDatabase('sql', (db) => {
        if (db.type !== 'Postgres') {
            return;
        }

        /**
         * Helper to intercept GetHealth GraphQL queries and return custom responses.
         * Must be called before navigation to ensure the route is in place when the
         * health polling service fires.
         */
        async function mockHealthResponse(page, { server, database }) {
            await page.route('**/api/query', async (route) => {
                let postData;
                try {
                    postData = route.request().postDataJSON();
                } catch {
                    return route.fallback();
                }

                if (postData?.operationName === 'GetHealth') {
                    return route.fulfill({
                        contentType: 'application/json',
                        body: JSON.stringify({
                            data: {
                                Health: {
                                    Server: server,
                                    Database: database,
                                    __typename: 'HealthStatus',
                                },
                            },
                        }),
                    });
                }

                await route.fallback();
            });
        }

        test('shows server down overlay when backend is unreachable', async ({ whodb, page }) => {
            // Mock health check to return server error before navigating
            await mockHealthResponse(page, { server: 'error', database: 'error' });
            await whodb.goto('storage-unit');

            // Server down overlay should appear with reconnection spinner text
            const overlay = page.locator('.fixed.inset-0').filter({
                has: page.locator('.animate-spin'),
            });
            await expect(overlay).toBeVisible({ timeout: TIMEOUT.SLOW });
        });

        test('shows database down overlay when DB connection lost', async ({ whodb, page }) => {
            // Mock health check to return server healthy but database error
            await mockHealthResponse(page, { server: 'healthy', database: 'error' });
            await whodb.goto('storage-unit');

            // Database down overlay should appear — it contains a "Logout" button (destructive variant)
            const logoutButton = page.locator('button').filter({ hasText: /logout/i });
            await expect(logoutButton).toBeVisible({ timeout: TIMEOUT.SLOW });
        });

        test('recovers when health returns to healthy', async ({ whodb, page }) => {
            // Start with server error
            await mockHealthResponse(page, { server: 'error', database: 'error' });
            await whodb.goto('storage-unit');

            // Wait for the overlay to appear
            const overlay = page.locator('.fixed.inset-0').filter({
                has: page.locator('.animate-spin'),
            });
            await expect(overlay).toBeVisible({ timeout: TIMEOUT.SLOW });

            // Now switch to healthy responses — remove old route and add new one
            await page.unrouteAll({ behavior: 'wait' });
            await mockHealthResponse(page, { server: 'healthy', database: 'healthy' });

            // Overlay should disappear once the next health poll returns healthy
            await expect(overlay).not.toBeVisible({ timeout: TIMEOUT.SLOW });
        });
    });

});
