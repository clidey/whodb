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

import { test, expect, forEachDatabase, conditionalTest } from '../../support/test-fixture.mjs';
import { hasFeature, getSqlQuery, getErrorPattern } from '../../support/database-config.mjs';

test.describe('Scratchpad', () => {

    // SQL Databases only
    forEachDatabase('sql', (db) => {
        test.describe('Query Execution', () => {
            // Get expected column names from config
            const expectedIdentifierCol = db.testTable.identifierField;
            const expectedCountCol = db.sql.countColumn;
            const expectedUpdatedValue = db.testTable.testValues.modified;

            test('executes SELECT query and shows results', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                const query = getSqlQuery(db, 'selectAllUsers');
                await whodb.writeCode(0, query);
                await whodb.runCode(0);

                const { columns, rows } = await whodb.getCellQueryOutput(0);
                expect(columns.map(c => c.toUpperCase())).toContain(expectedIdentifierCol.toUpperCase());
                expect(rows.length).toBeGreaterThan(0);
            });

            test('executes filtered query', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                const query = getSqlQuery(db, 'selectUserById');
                await whodb.writeCode(0, query);
                await whodb.runCode(0);

                const { rows } = await whodb.getCellQueryOutput(0);
                expect(rows.length).toEqual(1);
            });

            test('executes aggregate query', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                const query = getSqlQuery(db, 'countUsers');
                await whodb.writeCode(0, query);
                await whodb.runCode(0);

                const { columns, rows } = await whodb.getCellQueryOutput(0);
                expect(columns.map(c => c.toUpperCase())).toContain(expectedCountCol.toUpperCase());
                expect(rows.length).toEqual(1);
            });

            test('shows error for invalid query', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                const query = getSqlQuery(db, 'invalidQuery');
                await whodb.writeCode(0, query);
                await whodb.runCode(0);

                const errorPattern = getErrorPattern(db, 'tableNotFound');
                if (errorPattern) {
                    const errorText = await whodb.getCellError(0);
                    expect(errorText).toContain(errorPattern.split(' ')[0]);
                } else {
                    await expect(page.locator('[data-testid="cell-error"]')).toBeVisible();
                }
            });

            // Skip UPDATE test for databases with async mutations (e.g., ClickHouse)
            const updateSupported = hasFeature(db, 'scratchpadUpdate') !== false;

            conditionalTest(updateSupported, 'executes UPDATE query', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                const mutationDelay = db.mutationDelay || 0;

                // Update
                const updateQuery = getSqlQuery(db, 'updateUser');
                await whodb.writeCode(0, updateQuery);
                await whodb.runCode(0);

                const actionOutput = await whodb.getCellActionOutput(0);
                expect(actionOutput).toContain('Action Executed');

                // Wait for async mutations (e.g., ClickHouse)
                if (mutationDelay > 0) {
                    await page.waitForTimeout(mutationDelay);
                }

                // Verify
                await whodb.addCell(0);
                const selectQuery = getSqlQuery(db, 'selectUserById');
                await whodb.writeCode(1, selectQuery);
                await whodb.runCode(1);

                const { rows } = await whodb.getCellQueryOutput(1);
                expect(rows[0]).toContain(expectedUpdatedValue);

                // Revert
                await whodb.addCell(1);
                const revertQuery = getSqlQuery(db, 'revertUser');
                await whodb.writeCode(2, revertQuery);
                await whodb.runCode(2);

                const revertOutput = await whodb.getCellActionOutput(2);
                expect(revertOutput).toContain('Action Executed');
            });
        });

        test.describe('Cell Management', () => {
            test('adds and removes cells', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                // Add cells
                await whodb.addCell(0);
                await whodb.addCell(1);

                // Verify cells exist
                await expect(page.locator('[data-testid="cell-0"]')).toBeAttached();
                await expect(page.locator('[data-testid="cell-1"]')).toBeAttached();
                await expect(page.locator('[data-testid="cell-2"]')).toBeAttached();

                // Remove middle cell
                await whodb.removeCell(1);

                await expect(page.locator('[data-testid="cell-2"]')).not.toBeAttached();
            });
        });

        test.describe('Page Management', () => {
            test('creates and manages multiple pages', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                // Add new page
                await whodb.addScratchpadPage();

                let pages = await whodb.getScratchpadPages();
                expect(pages.length).toEqual(2);

                // Delete page with cancel
                await whodb.deleteScratchpadPage(1, true);

                pages = await whodb.getScratchpadPages();
                expect(pages.length).toEqual(2);

                // Delete page for real
                await whodb.deleteScratchpadPage(1, false);

                pages = await whodb.getScratchpadPages();
                expect(pages.length).toEqual(1);
            });
        });

        test.describe('Query Export', () => {
            test('exports query results as CSV', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                const exportPromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/export') && resp.request().method() === 'POST'
                );

                const query = getSqlQuery(db, 'selectAllUsers');
                await whodb.writeCode(0, query);
                await whodb.runCode(0);

                const { rows } = await whodb.getCellQueryOutput(0);
                expect(rows.length).toBeGreaterThan(0);

                // Click export button inside the query output
                await page.locator('[data-testid="cell-query-output"] [data-testid="export-all-button"]').click();
                await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();

                // Verify the raw query export message
                await expect(page.locator('text=You are about to export the results of your query.')).toBeVisible();

                // Verify CSV is selected by default
                await expect(page.locator('[data-testid="export-format-select"]')).toContainText('CSV');

                await whodb.confirmExport();

                const response = await exportPromise;
                expect(response.status()).toEqual(200);
                const request = response.request();
                const postData = JSON.parse(request.postData());
                expect(postData.selectedRows).toBeDefined();
                expect(Array.isArray(postData.selectedRows)).toBe(true);
                expect(postData.selectedRows.length).toBeGreaterThan(0);
                expect(postData.storageUnit).toEqual('query_export');
            });

            test('exports query results as Excel', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                const exportPromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/export') && resp.request().method() === 'POST'
                );

                const query = getSqlQuery(db, 'selectAllUsers');
                await whodb.writeCode(0, query);
                await whodb.runCode(0);

                const { rows } = await whodb.getCellQueryOutput(0);
                expect(rows.length).toBeGreaterThan(0);

                await page.locator('[data-testid="cell-query-output"] [data-testid="export-all-button"]').click();
                await whodb.selectExportFormat('excel');

                await whodb.confirmExport();

                const response = await exportPromise;
                expect(response.status()).toEqual(200);
                const request = response.request();
                const postData = JSON.parse(request.postData());
                expect(postData.selectedRows).toBeDefined();
                expect(postData.format).toEqual('excel');
                expect(postData.storageUnit).toEqual('query_export');
            });

            test('preselects Excel when "Export All as Excel" is chosen from context menu', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                const query = getSqlQuery(db, 'selectAllUsers');
                await whodb.writeCode(0, query);
                await whodb.runCode(0);

                const { rows } = await whodb.getCellQueryOutput(0);
                expect(rows.length).toBeGreaterThan(0);

                // Right-click on a visible data cell (eq(1) skips the hidden checkbox td)
                await page.locator('[data-testid="cell-query-output"] table tbody tr').first().locator('td').nth(1).click({ button: 'right' });
                await page.waitForTimeout(300);

                // Navigate to Export submenu and click "Export All as Excel"
                await page.locator('[role="menu"]').locator('text=Export').click();
                await expect(page.locator('text=Export All as Excel')).toBeVisible();
                await page.locator('text=Export All as Excel').click();

                // Verify the export dialog opens with Excel preselected
                await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();
                await expect(page.locator('[data-testid="export-format-select"]')).toContainText('Excel');

                await page.keyboard.press('Escape');
            });

            test('does not show "Export Selected" options in context menu', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                const query = getSqlQuery(db, 'selectAllUsers');
                await whodb.writeCode(0, query);
                await whodb.runCode(0);

                const { rows } = await whodb.getCellQueryOutput(0);
                expect(rows.length).toBeGreaterThan(0);

                // Right-click on a visible data cell (eq(1) skips the hidden checkbox td)
                await page.locator('[data-testid="cell-query-output"] table tbody tr').first().locator('td').nth(1).click({ button: 'right' });
                await page.waitForTimeout(300);

                // Open the Export submenu (scope to context menu to avoid matching "Export All" button)
                await page.locator('[role="menu"]').locator('text=Export').click();

                // "Export All" options should be visible
                await expect(page.locator('text=Export All as CSV')).toBeVisible();
                await expect(page.locator('text=Export All as Excel')).toBeVisible();

                // "Export Selected" options should NOT exist
                await expect(page.locator('text=Export Selected as CSV')).not.toBeAttached();
                await expect(page.locator('text=Export Selected as Excel')).not.toBeAttached();

                await page.keyboard.press('Escape');
            });
        });

        test.describe('Embedded Scratchpad Drawer', () => {
            const testTable = db.testTable;
            const tableName = testTable.name;

            test('opens from data view and runs query', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open embedded scratchpad drawer
                await page.locator('[data-testid="embedded-scratchpad-button"]').click();
                await expect(page.locator('[data-slot="drawer-title"]').first()).toBeVisible();

                // Verify default query is populated
                await expect(page.locator('[data-testid="code-editor"]')).toBeAttached();
                const schemaPrefix = db.sql.schemaPrefix;
                await expect(page.locator('[data-testid="code-editor"]')).toContainText('SELECT');
                await expect(page.locator('[data-testid="code-editor"]')).toContainText(`FROM ${schemaPrefix}${tableName}`);

                // Run the query
                await page.locator('[data-testid="run-submit-button"]').filter({ hasText: 'Run' }).first().click();

                // Verify results appear in the drawer
                await expect(page.locator('[role="dialog"] table')).toBeVisible({ timeout: 5000 });
                const rowCount = await page.locator('[role="dialog"] table tbody tr').count();
                expect(rowCount).toBeGreaterThanOrEqual(1);

                // Close the drawer
                await page.keyboard.press('Escape');
                await expect(page.locator('[data-testid="table-search"]')).toBeVisible();
            });
        });
    }, { features: ['scratchpad'] });

});
