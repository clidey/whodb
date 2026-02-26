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

import { test, expect, forEachDatabase, skipIfNoFeature } from '../../support/test-fixture.mjs';

test.describe('Pagination', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;
        const firstName = testTable.firstName;
        const colIndex = testTable.identifierColIndex;

        test('respects page size setting', async ({ whodb, page }) => {
            await whodb.data(tableName);
            await whodb.sortBy(0);

            // Set page size to 1
            await whodb.setTablePageSize(1);
            await whodb.submitTable();

            let tableData = await whodb.getTableData();
            expect(tableData.rows.length).toEqual(1);
            expect(tableData.rows[0][colIndex + 1]).toEqual(firstName);

            // Set page size to 2
            await whodb.setTablePageSize(2);
            await whodb.submitTable();

            tableData = await whodb.getTableData();
            expect(tableData.rows.length).toEqual(2);

            // Reset to default
            await whodb.setTablePageSize(10);
            await whodb.submitTable();

            tableData = await whodb.getTableData();
            expect(tableData.rows.length).toBeGreaterThan(2);
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        test('respects page size setting', async ({ whodb, page }) => {
            await whodb.data('users');

            // Set page size to 1
            await whodb.setTablePageSize(1);
            await whodb.submitTable();

            let tableData = await whodb.getTableData();
            expect(tableData.rows.length).toEqual(1);

            // Reset to default
            await whodb.setTablePageSize(10);
            await whodb.submitTable();

            tableData = await whodb.getTableData();
            expect(tableData.rows.length).toBeGreaterThan(1);
        });
    });

    // Custom page size from settings
    forEachDatabase('sql', (db) => {
        if (db.type !== 'Postgres') {
            return;
        }

        const tableName = db.testTable.name;

        test('respects custom page size from settings', async ({ whodb, page }) => {
            // Set custom page size to 2 via the settings page UI
            await whodb.goto('settings');
            await page.locator('#default-page-size').click();
            await page.locator('[data-value="custom"]').click();
            await page.locator('input[type="number"]').clear();
            await page.locator('input[type="number"]').fill('2');
            await page.locator('input[type="number"]').press('Enter');

            // Navigate to data view and verify the setting is applied
            await whodb.data(tableName);

            const tableData = await whodb.getTableData();
            expect(tableData.rows.length).toEqual(2);
        });
    });

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        if (skipIfNoFeature(db, 'pagination')) {
            test.skip('respects page size setting', async ({ whodb, page }) => {});
            return;
        }

        test('respects page size setting', async ({ whodb, page }) => {
            await whodb.data('user:1');

            // Set page size to 2
            await whodb.setTablePageSize(2);
            await whodb.submitTable();

            let tableData = await whodb.getTableData();
            expect(tableData.rows.length).toBeLessThanOrEqual(2);

            // Reset to default
            await whodb.setTablePageSize(10);
            await whodb.submitTable();

            tableData = await whodb.getTableData();
            expect(tableData.rows.length).toBeGreaterThan(2);
        });
    });

});
