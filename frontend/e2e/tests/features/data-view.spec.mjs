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
import { getTableConfig } from '../../support/database-config.mjs';
import { parseDocument, verifyDocumentRows } from '../../support/categories/document.mjs';
import { verifyColumnsForType, verifyMembers, verifyStringValue } from '../../support/categories/keyvalue.mjs';

test.describe('Data View', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;

        test('displays table data with correct columns', async ({ whodb, page }) => {
            await whodb.data(tableName);
            await whodb.sortBy(0); // Sort by first column (id)

            const { columns, rows } = await whodb.getTableData();
            const tableConfig = getTableConfig(db, tableName);

            // Verify columns
            if (tableConfig && tableConfig.expectedColumns) {
                expect(columns).toEqual(tableConfig.expectedColumns);
            }

            // Verify data exists
            expect(rows.length).toBeGreaterThan(0);

            // Verify first row data if configured (initial should be array of arrays)
            if (tableConfig && tableConfig.testData && tableConfig.testData.initial && Array.isArray(tableConfig.testData.initial[0])) {
                const expectedFirst = tableConfig.testData.initial[0];
                expectedFirst.forEach((val, idx) => {
                    if (val !== '') {
                        expect(rows[0][idx]).toEqual(val);
                    }
                });
            }
        });

        test('respects page size pagination', async ({ whodb, page }) => {
            await whodb.data(tableName);
            await whodb.setTablePageSize(1);
            await whodb.submitTable();

            const { rows } = await whodb.getTableData();
            expect(rows.length).toEqual(1);
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        test('displays document data', async ({ whodb, page }) => {
            await whodb.data('users');
            await whodb.sortBy(0);

            const { columns, rows } = await whodb.getTableData();
            const tableConfig = getTableConfig(db, 'users');
            const testValues = db.testTable?.testValues;

            // Document DBs have [checkbox, document] columns
            if (tableConfig && tableConfig.expectedColumns) {
                expect(columns).toEqual(tableConfig.expectedColumns);
            }

            // Verify document content (allowing for modified values from CRUD tests)
            if (tableConfig && tableConfig.testData && tableConfig.testData.initial) {
                const expectedDocs = tableConfig.testData.initial.map((doc, idx) => {
                    // For the row that CRUD tests modify, accept either original or modified value
                    if (testValues && idx === testValues.rowIndex && testValues.original && testValues.modified) {
                        return {
                            ...doc,
                            [db.testTable.identifierField]: [testValues.original, testValues.modified]
                        };
                    }
                    return doc;
                });

                expect(rows.length).toEqual(expectedDocs.length);
                expectedDocs.forEach((expected, idx) => {
                    const doc = parseDocument(rows[idx]);
                    Object.entries(expected).forEach(([key, value]) => {
                        if (Array.isArray(value)) {
                            // Accept any of the allowed values
                            expect(value, `Document field ${key}`).toContain(doc[key]);
                        } else {
                            expect(doc[key], `Document field ${key}`).toEqual(value);
                        }
                    });
                });
            }
        });

        test('respects page size pagination', async ({ whodb, page }) => {
            await whodb.data('users');
            await whodb.setTablePageSize(1);
            await whodb.submitTable();

            const { rows } = await whodb.getTableData();
            expect(rows.length).toEqual(1);
        });
    });

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        test('displays hash data correctly', async ({ whodb, page }) => {
            await whodb.data('user:2');
            const { columns, rows } = await whodb.getTableData();
            const keyConfig = db.keyTypes['user:2'];
            verifyColumnsForType(columns, 'hash');

            if (keyConfig.expectedRowCount) {
                expect(rows.length).toEqual(keyConfig.expectedRowCount);
            }
            if (keyConfig.testData && keyConfig.testData.firstRow) {
                expect(rows[0]).toEqual(keyConfig.testData.firstRow);
            }
        });

        test('displays list data correctly', async ({ whodb, page }) => {
            await whodb.data('orders:recent');
            const { columns, rows } = await whodb.getTableData();
            verifyColumnsForType(columns, 'list');
            expect(rows.length).toBeGreaterThan(0);
        });

        test('displays set data correctly', async ({ whodb, page }) => {
            await whodb.data('category:electronics');
            const { columns, rows } = await whodb.getTableData();
            const keyConfig = db.keyTypes['category:electronics'];
            verifyColumnsForType(columns, 'set');

            if (keyConfig.expectedMembers) {
                const members = rows.map(row => row[2]);
                verifyMembers(rows, keyConfig.expectedMembers);
            }
        });

        test('displays sorted set data correctly', async ({ whodb, page }) => {
            await whodb.data('products:by_price');
            const { columns, rows } = await whodb.getTableData();
            verifyColumnsForType(columns, 'zset');
            expect(rows.length).toBeGreaterThan(0);
        });

        test('displays string data correctly', async ({ whodb, page }) => {
            await whodb.data('inventory:product:1');
            const { columns, rows } = await whodb.getTableData();
            const keyConfig = db.keyTypes['inventory:product:1'];
            verifyColumnsForType(columns, 'string');

            if (keyConfig.expectedValue) {
                verifyStringValue(rows, keyConfig.expectedValue);
            }
        });

        // Redis hashes are fetched as a complete unit (HGETALL), so server-side
        // pagination doesn't apply to hash fields
        if (db.type === 'Redis') {
            test.skip('respects page size pagination (Redis hashes do not support field pagination)', async ({ whodb, page }) => {
            });
            return;
        }

        test('respects page size pagination', async ({ whodb, page }) => {
            await whodb.data('user:1');
            await whodb.setTablePageSize(2);
            await whodb.submitTable();

            const { rows } = await whodb.getTableData();
            expect(rows.length).toBeLessThanOrEqual(2);
        });
    });

});
