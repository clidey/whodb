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

test.describe('Data Export', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;

        test.describe('Export All', () => {
            test('exports table data as CSV with default comma delimiter', async ({ whodb, page }) => {
                await whodb.data(tableName);

                const exportPromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/export') && resp.request().method() === 'POST'
                );

                // Use Export All button
                await page.locator('[data-testid="export-all-button"]').click();
                await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();

                // Verify default format is CSV with comma delimiter
                const dialog = page.locator('[role="dialog"]');
                await expect(dialog.locator('[data-testid="export-format-select"]')).toContainText('CSV');
                await expect(dialog.locator('[data-testid="export-delimiter-select"]')).toContainText('Comma');

                // Export
                await whodb.confirmExport();

                const response = await exportPromise;
                expect(response.status()).toEqual(200);
                const headers = response.headers();
                const cd = headers['content-disposition'];
                expect(typeof cd).toEqual('string');
                expect(cd).toMatch(/\.csv/i);

                await page.keyboard.press('Escape');
                await expect(page.locator('[role="dialog"]')).not.toBeAttached();
            });

            test('exports table data as Excel', async ({ whodb, page }) => {
                await whodb.data(tableName);

                const exportPromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/export') && resp.request().method() === 'POST'
                );

                await page.locator('[data-testid="export-all-button"]').click();
                await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();

                // Change format to Excel
                await whodb.selectExportFormat('excel');

                // Verify Excel description shows (locale uses capital F in Format)
                await expect(page.locator('text=Excel XLSX Format')).toBeVisible();

                await whodb.confirmExport();

                const response = await exportPromise;
                expect(response.status()).toEqual(200);
                const headers = response.headers();
                const cd = headers['content-disposition'];
                expect(typeof cd).toEqual('string');
                expect(cd).toMatch(/\.xlsx/i);

                await page.keyboard.press('Escape');
                await expect(page.locator('[role="dialog"]')).not.toBeAttached();
            });
        });

        test.describe('Export Selected Rows', () => {
            test('exports selected rows with pipe delimiter', async ({ whodb, page }) => {
                await whodb.data(tableName);

                const exportPromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/export') && resp.request().method() === 'POST'
                );

                // Wait for table to stabilize after data load
                await page.locator('table tbody tr').first().waitFor({ timeout: 5000 });
                await page.waitForTimeout(1000); // Wait for any re-renders to complete

                // Select a row via context menu - click on first visible data cell
                const targetCell = page.locator('table tbody tr').first().locator('td').nth(1);
                await targetCell.scrollIntoViewIfNeeded();
                await targetCell.click({ button: 'right', position: { x: 5, y: 5 } });
                await page.waitForTimeout(500); // Wait for context menu animation
                await expect(page.locator('text=Select Row')).toBeVisible();
                await page.locator('text=Select Row').click({ force: true });
                await page.waitForTimeout(300); // Wait for selection to register

                // Verify row was selected - button should change to "Export 1 Selected"
                await expect(page.locator('button', { hasText: 'Export 1 Selected' })).toBeVisible();
                await page.locator('button', { hasText: 'Export 1 Selected' }).click();

                await expect(page.locator('[role="dialog"]')).toBeVisible();
                // Note: UI shows {count} with braces due to translation format
                await expect(page.locator('text=You are about to export {1} selected rows.')).toBeVisible();

                // Ensure CSV format is selected
                await whodb.selectExportFormat('csv');

                // Change delimiter to pipe
                await whodb.selectExportDelimiter('|');

                // Verify pipe delimiter selected
                await expect(page.locator('[data-testid="export-delimiter-select"]')).toContainText('|');

                await whodb.confirmExport();

                const response = await exportPromise;
                expect(response.status()).toEqual(200);
                const request = response.request();
                const postData = JSON.parse(request.postData());
                expect(postData.delimiter).toEqual('|');
                expect(postData.selectedRows).toBeDefined();
                expect(Array.isArray(postData.selectedRows)).toBe(true);
                expect(postData.selectedRows.length).toBeGreaterThan(0);

                await expect(page.locator('[role="dialog"]')).not.toBeAttached();
            });
        });

        test.describe('Export All Ignores Selection', () => {
            test('shows "all data" message when "Export All" is chosen from context menu with rows selected', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Wait for table to stabilize
                await page.locator('table tbody tr').first().waitFor({ timeout: 5000 });
                await page.waitForTimeout(1000);

                // Select a row via context menu
                const targetCell = page.locator('table tbody tr').first().locator('td').nth(1);
                await targetCell.scrollIntoViewIfNeeded();
                await targetCell.click({ button: 'right', position: { x: 5, y: 5 } });
                await page.waitForTimeout(500);
                await expect(page.locator('text=Select Row')).toBeVisible();
                await page.locator('text=Select Row').click({ force: true });
                await page.waitForTimeout(300);

                // Verify row is selected
                await expect(page.locator('button', { hasText: 'Export 1 Selected' })).toBeVisible();

                // Now right-click again and choose "Export All as CSV"
                await page.locator('table tbody tr').first().locator('td').nth(1).click({ button: 'right', position: { x: 5, y: 5 } });
                await page.waitForTimeout(500);
                await page.locator('[role="menu"]').locator('text=Export').click();
                await expect(page.locator('text=Export All as CSV')).toBeVisible();
                await page.locator('text=Export All as CSV').click();

                // Verify dialog shows "all data" message, NOT "selected rows"
                await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();
                await expect(page.locator('text=You are about to export all data from the table.')).toBeVisible();

                await page.keyboard.press('Escape');
                await expect(page.locator('[role="dialog"]')).not.toBeAttached();
            });
        });
    }, { features: ['export'] });

    // Document Databases
    forEachDatabase('document', (db) => {
        test('exports collection/index data as NDJSON', async ({ whodb, page }) => {
            await whodb.data('users');

            const exportPromise = page.waitForResponse(resp =>
                resp.url().includes('/api/export') && resp.request().method() === 'POST'
            );

            await page.locator('[data-testid="export-all-button"]').click();
            await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();

            // Verify NDJSON format is default for NoSQL
            await expect(page.locator('[data-testid="export-format-select"]')).toContainText('JSON');

            await whodb.confirmExport();

            const response = await exportPromise;
            expect(response.status()).toEqual(200);
            const request = response.request();
            const postData = JSON.parse(request.postData());
            expect(postData.format).toEqual('ndjson');
            const headers = response.headers();
            const cd = headers['content-disposition'];
            expect(typeof cd).toEqual('string');
            expect(cd).toMatch(/\.ndjson/i);

            await page.keyboard.press('Escape');
            await expect(page.locator('[role="dialog"]')).not.toBeAttached();
        });

        test('exports collection/index data as CSV when selected', async ({ whodb, page }) => {
            await whodb.data('users');

            const exportPromise = page.waitForResponse(resp =>
                resp.url().includes('/api/export') && resp.request().method() === 'POST'
            );

            await page.locator('[data-testid="export-all-button"]').click();
            await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();

            // Switch format to CSV
            await whodb.selectExportFormat('csv');

            // Delimiter control should appear for CSV
            await expect(page.locator('label', { hasText: 'Delimiter' })).toBeVisible();

            await whodb.confirmExport();

            const response = await exportPromise;
            expect(response.status()).toEqual(200);
            const request = response.request();
            const postData = JSON.parse(request.postData());
            expect(postData.format).toEqual('csv');
            const headers = response.headers();
            const cd = headers['content-disposition'];
            expect(typeof cd).toEqual('string');
            expect(cd).toMatch(/\.csv/i);

            await page.keyboard.press('Escape');
            await expect(page.locator('[role="dialog"]')).not.toBeAttached();
        });
    }, { features: ['export'] });

    // Key/Value Databases (e.g., Redis)
    forEachDatabase('keyvalue', (db) => {
        const tableName = db.testTable.name;

        test('exports key data as NDJSON by default', async ({ whodb, page }) => {
            await whodb.data(tableName);

            const exportPromise = page.waitForResponse(resp =>
                resp.url().includes('/api/export') && resp.request().method() === 'POST'
            );

            await page.locator('[data-testid="export-all-button"]').click();
            await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();

            await expect(page.locator('[data-testid="export-format-select"]')).toContainText('JSON');

            await whodb.confirmExport();

            const response = await exportPromise;
            expect(response.status()).toEqual(200);
            const request = response.request();
            const postData = JSON.parse(request.postData());
            expect(postData.format).toEqual('ndjson');
            const headers = response.headers();
            const cd = headers['content-disposition'];
            expect(typeof cd).toEqual('string');
            expect(cd).toMatch(/\.ndjson/i);

            await page.keyboard.press('Escape');
            await expect(page.locator('[role="dialog"]')).not.toBeAttached();
        });
    }, { features: ['export'] });
});
