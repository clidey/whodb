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

/**
 * UI elements introduced or restored with the CE redesign.
 *
 * These are read-only checks that the redesigned shell still exposes the
 * controls users rely on: object types on the Tables page, grouped objects in
 * the explore sidebar, the connection breadcrumb, the command palette empty
 * state, the embedded scratchpad, scratchpad cancel, and the mobile nav toggle.
 */
test.describe('UI Elements', () => {
    forEachDatabase('sql', (db) => {
        const tableName = db.testTable.name;

        test('labels each object on the Tables page with its own type', async ({ whodb, page }) => {
            // The card root does not carry the test id, so find the card by the name it wraps.
            const cardFor = (name) => page.locator('.ce-table-card').filter({ has: page.locator(`[data-table-name="${name}"]`) }).first();
            await expect(cardFor('users').locator('[data-testid="storage-unit-type"]')).toHaveText(/table/i);
            await expect(cardFor('order_summary').locator('[data-testid="storage-unit-type"]')).toHaveText(/view/i);
        });

        test('groups objects by type in the explore sidebar', async ({ whodb, page }) => {
            await whodb.data(tableName);

            const groups = page.locator('[data-testid="schema-viewer-group"]');
            await expect(groups.first()).toBeVisible();
            expect(await groups.count()).toBeGreaterThan(1);
            await expect(page.locator('[data-testid="schema-viewer-group"][data-group-type*="view" i]')).toHaveCount(1);
        });

        test('shows the connection breadcrumb outside the Tables page', async ({ whodb, page }) => {
            await whodb.data(tableName);
            await expect(page.locator('[data-testid="connection-context"]')).toBeVisible();

            await whodb.goto('scratchpad');
            await expect(page.locator('[data-testid="connection-context"]')).toBeVisible();

            await whodb.goto('chat');
            await expect(page.locator('[data-testid="connection-context"]')).toBeVisible();
        });

        test('keeps the sidebar footer extras on the Tables page', async ({ whodb, page }) => {
            await expect(page.locator('[data-testid="sidebar-version-info"]')).toBeVisible();
        });

        test('shows a no-results state in the command palette', async ({ whodb, page }) => {
            await whodb.typeCmdShortcut('k');
            await page.locator('[data-testid="command-palette-input"]').fill('zzz-no-such-table-zzz');
            // Navigation stays available while searching; the empty state covers tables and sort commands.
            await expect(page.locator('[data-testid="command-palette-empty"]')).toBeVisible();
            await expect(page.locator('[data-testid="command-nav-chat"]')).toBeAttached();
            await page.keyboard.press('Escape');
        });

        test('offers the embedded scratchpad on the data view', async ({ whodb, page }) => {
            await whodb.data(tableName);
            await expect(page.locator('[data-testid="embedded-scratchpad-button"]')).toBeVisible();
        });

        test('shows a cancel control while a scratchpad query runs', async ({ whodb, page }) => {
            await whodb.goto('scratchpad');
            await page.locator('[data-testid="cell-0"]').waitFor({ timeout: TIMEOUT.ACTION });

            // Hold the RawExecute response so the running state is observable.
            await page.route('**/api/query', async (route) => {
                const body = route.request().postDataJSON?.();
                if (body?.operationName !== 'RawExecute') {
                    return route.fallback();
                }
                await new Promise(resolve => setTimeout(resolve, 4000));
                return route.fallback();
            });

            await whodb.writeCode(0, `SELECT * FROM ${db.sql.schemaPrefix}${tableName}`);
            await whodb.queryCellButton(0).click();

            await expect(page.locator('[data-testid="scratchpad-running"]')).toBeVisible();
            await page.locator('[data-testid="scratchpad-cancel"]').click();
            await expect(page.locator('[data-testid="scratchpad-running"]')).not.toBeAttached();
            await expect(page.locator('[data-testid="cell-0"] [data-testid="cell-query-output"]')).toHaveCount(0);

            await page.unroute('**/api/query');
        });

        test('offers a foreign-key picker when adding a row', async ({ whodb, page }) => {
            await whodb.data('orders');
            await page.locator('[data-testid="add-row-button"]').click();

            const picker = page.locator('[data-testid="add-row-fk-user_id"]');
            await expect(picker).toBeVisible();
            await picker.click();
            const options = page.locator('[role="option"]');
            await expect(options.first()).toBeVisible();
            // Referenced ids plus the manual-entry option.
            expect(await options.count()).toBeGreaterThan(1);
            await options.first().click();
            await expect(picker).not.toContainText('Enter');

            await page.locator('[data-testid="cancel-add-row"]').click();
            await expect(page.locator('[data-testid="submit-add-row-button"]')).not.toBeVisible();
        });

        test('shows the auto-scroll toggle on the chat page', async ({ whodb, page }) => {
            await whodb.goto('chat');
            await expect(page.locator('[data-testid="chat-auto-scroll-toggle"]')).toBeAttached();
        });

        test('opens the navigation from the mobile toggle', async ({ whodb, page }) => {
            await page.setViewportSize({ width: 600, height: 900 });
            await whodb.goto('storage-unit');

            const toggle = page.locator('[data-testid="mobile-nav-toggle"]');
            await expect(toggle).toBeVisible();
            await toggle.click();
            await expect(page.locator('[data-sidebar="sidebar"]').getByRole('link', { name: 'Chat' }).first()).toBeVisible();
        });
    }, { databases: ['postgres'] });
});
