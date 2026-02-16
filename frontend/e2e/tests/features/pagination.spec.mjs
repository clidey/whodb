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

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        // Redis hashes are fetched as a complete unit (HGETALL), so server-side
        // pagination doesn't apply to hash fields
        if (db.type === 'Redis') {
            test.skip('respects page size setting (Redis hashes do not support field pagination)', async ({ whodb, page }) => {
            });
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
