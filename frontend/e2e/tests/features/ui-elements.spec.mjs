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

        test('keeps secondary navigation below the shared header in both themes', async ({ whodb, page }) => {
            await whodb.data(tableName);
            for (const colorScheme of ['light', 'dark']) {
                await page.emulateMedia({ colorScheme });
                const list = page.getByTestId('schema-viewer');
                const header = page.getByTestId('connection-context');
                await expect(list).toBeVisible();
                const listBox = await list.boundingBox();
                const headerBox = await header.boundingBox();
                expect(listBox.y).toBeGreaterThanOrEqual(headerBox.y + headerBox.height);
                const colors = await list.evaluate(element => ({
                    list: getComputedStyle(element).backgroundColor,
                    primary: getComputedStyle(document.querySelector('[data-slot="sidebar-inner"]')).backgroundColor,
                }));
                expect(colors.list).not.toBe(colors.primary);
            }
        });

        test('opens the object list on mobile and returns to the selected object', async ({ whodb, page }) => {
            await whodb.data(tableName);
            await page.setViewportSize({ width: 390, height: 844 });
            await page.getByRole('button', { name: 'Browse items', exact: true }).click();
            const list = page.getByTestId('schema-viewer');
            await expect(list).toBeVisible();
            await expect(list.getByRole('textbox')).toBeFocused();
            await list.locator('button[aria-current="page"]').click();
            await expect(list).toBeHidden();
            await expect(page.getByTestId('page-content')).toBeVisible();
            await expect(page.getByTestId('page-content')).toBeFocused();
        });

        test('keeps chat renaming reachable from the mobile list', async ({ whodb, page }) => {
            await whodb.goto('chat');
            await page.setViewportSize({ width: 390, height: 844 });
            await page.getByRole('button', { name: 'Browse items', exact: true }).click();
            const list = page.getByTestId('chat-history-sidebar');
            await list.getByTestId('add-chat-session-button').click();
            await page.getByRole('button', { name: 'Browse items', exact: true }).click();
            await list.getByRole('button', { name: 'Rename chat', exact: true }).first().click();
            const input = list.locator('[data-testid^="chat-session-name-input-"]');
            await input.fill('Renamed session');
            await input.press('Enter');
            const session = list.locator('[data-testid^="chat-session-item-"]').filter({ hasText: 'Renamed session' });
            await expect(session).toBeVisible();
            await session.getByRole('button', { name: 'Delete Chat', exact: true }).click();
            await expect(page.getByRole('alertdialog')).toBeVisible();
            await page.getByTestId('delete-chat-cancel').click();
            await session.click();
            await expect(list).toBeHidden();
            await expect(page.getByTestId('page-content')).toBeFocused();
        });

        test('preserves pending cell edits while browsing the mobile object list', async ({ whodb, page }) => {
            await page.setViewportSize({ width: 390, height: 844 });
            await whodb.data(tableName);
            await whodb.dataCell(0, db.testTable.identifierColIndex + 1).dblclick();
            const editor = page.getByTestId('cell-editor');
            await editor.fill('pending_mobile_change');
            await editor.press('Enter');
            const saveBar = page.getByTestId('cell-save-bar');
            await expect(saveBar).toBeVisible();
            await page.getByRole('button', { name: 'Browse items', exact: true }).click();
            await page.getByRole('button', { name: 'View content', exact: true }).click();
            await expect(saveBar).toBeVisible();
            await saveBar.getByRole('button', { name: 'Discard', exact: true }).click();
        });

        test('shows the connection breadcrumb outside the Tables page', async ({ whodb, page }) => {
            await whodb.data(tableName);
            await expect(page.locator('[data-testid="connection-context"]')).toBeVisible();

            await whodb.goto('scratchpad');
            await expect(page.locator('[data-testid="connection-context"]')).toBeVisible();

            await whodb.goto('chat');
            await expect(page.locator('[data-testid="connection-context"]')).toBeVisible();
        });

        test('keeps the sidebar footer extras on the Tables page', async ({ whodb, page }, testInfo) => {
            test.skip(testInfo.project.metadata.uiFeatures?.sourceVersionFooter === false, 'This UI does not use the source version footer');
            await expect(page.locator('[data-testid="sidebar-version-info"]')).toBeVisible();
        });

        test('shows a no-results state in the command palette', async ({ whodb, page }, testInfo) => {
            test.skip(testInfo.project.metadata.uiFeatures?.sourceCommandPalette === false, 'This UI supplies its own command palette');
            await whodb.typeCmdShortcut('k');
            await page.locator('[data-testid="command-palette-input"]').fill('zzz-no-such-table-zzz');
            // Navigation stays available while searching; the empty state covers tables and sort commands.
            await expect(page.locator('[data-testid="command-palette-empty"]')).toBeVisible();
            await expect(page.locator('[data-testid="command-nav-chat"]')).toBeAttached();
            await page.keyboard.press('Escape');
        });

        test('shows completed statements as compact information alerts', async ({ whodb, page }) => {
            await page.route('**/api/query', async route => {
                if (route.request().postDataJSON()?.operationName !== 'RawExecute') {
                    await route.fallback();
                    return;
                }
                await route.fulfill({ json: { data: { RawExecute: {
                    Columns: [], Rows: [], DisableUpdate: true, TotalCount: 0, __typename: 'RowsResult',
                } } } });
            });
            await whodb.goto('scratchpad');
            await whodb.writeCode(0, 'UPDATE docs_preview SET value = value WHERE 1 = 0;');
            await whodb.runCode(0);
            const output = page.getByTestId('cell-action-output');
            const alert = output.getByRole('alert');
            await expect(alert).toHaveAttribute('data-variant', 'info');
            await expect(alert).toContainText('Action Executed');
            for (const width of [390, 1440]) {
                await page.setViewportSize({ width, height: 900 });
                const box = await output.boundingBox();
                const cell = await page.getByTestId('cell-0').boundingBox();
                expect(box.width).toBeLessThan(cell.width * 0.8);
            }
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
