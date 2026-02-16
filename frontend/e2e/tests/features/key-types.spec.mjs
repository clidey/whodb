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
import { verifyColumnsForType } from '../../support/categories/keyvalue.mjs';

/**
 * Key Types Test Suite for Key-Value Databases
 *
 * Tests that Redis correctly handles different key types (string, hash, list,
 * set, zset) with proper column structures and operations where supported.
 *
 * This is analogous to data-types.spec.mjs for SQL databases, but adapted for
 * the schema-less nature of key-value stores where "types" are determined by
 * the Redis data structure used for each key.
 */
test.describe('Key Types Operations', () => {

    forEachDatabase('keyvalue', (db) => {
        const keyTypeTests = db.keyTypeTests;

        if (!keyTypeTests) {
            test.skip('keyTypeTests config missing in fixture', async () => {
            });
            return;
        }

        Object.entries(keyTypeTests).forEach(([keyType, testConfig]) => {
            test.describe(`Type: ${testConfig.typeName} (${keyType})`, () => {
                const { testKey, expectedColumns, testData, supportsUpdate } = testConfig;

                test('COLUMNS - displays correct column structure', async ({ whodb, page }) => {
                    await whodb.data(testKey);

                    const { columns } = await whodb.getTableData();
                    // Use the existing verifyColumnsForType helper
                    verifyColumnsForType(columns, keyType);
                    // Also verify against fixture-defined columns
                    expect(columns, `${testConfig.typeName} key should have correct columns`).toEqual(expectedColumns);
                });

                if (keyType === 'string') {
                    test('DATA - displays string value correctly', async ({ whodb, page }) => {
                        await whodb.data(testKey);

                        const { rows } = await whodb.getTableData();
                        expect(rows.length).toEqual(1);
                        expect(rows[0][testData.valueColumnIndex]).toEqual(testData.originalValue);
                    });

                    if (supportsUpdate) {
                        test('UPDATE - modifies string value', async ({ whodb, page }) => {
                            await whodb.data(testKey);

                            // For strings, there's only one row with value column
                            await whodb.updateRow(0, 0, testData.updateValue, false);

                            let { rows } = await whodb.getTableData();
                            expect(rows[0][testData.valueColumnIndex]).toEqual(testData.updateValue);

                            // Revert
                            await whodb.updateRow(0, 0, testData.originalValue, false);

                            ({ rows } = await whodb.getTableData());
                            expect(rows[0][testData.valueColumnIndex]).toEqual(testData.originalValue);
                        });
                    }
                }

                if (keyType === 'hash') {
                    test('DATA - displays hash fields correctly', async ({ whodb, page }) => {
                        await whodb.data(testKey);

                        const { rows } = await whodb.getTableData();
                        expect(rows.length).toBeGreaterThan(0);

                        // Verify the test field exists with expected value (allow modified value from failed tests)
                        const targetRow = rows.find(r => r[testData.fieldColumnIndex] === testData.testField);
                        expect(targetRow, `Hash should contain field: ${testData.testField}`).toBeDefined();
                        const actualValue = targetRow[testData.valueColumnIndex];
                        const validValues = [testData.originalValue, testData.updateValue];
                        expect(validValues, `Hash field ${testData.testField}`).toContain(actualValue);
                    });

                    if (supportsUpdate) {
                        test('UPDATE - modifies hash field value', async ({ whodb, page }) => {
                            await whodb.data(testKey);

                            const { rows } = await whodb.getTableData();
                            const rowIndex = rows.findIndex(r => r[testData.fieldColumnIndex] === testData.testField);
                            expect(rowIndex, `Row with field ${testData.testField} should exist`).toBeGreaterThan(-1);

                            // If data was left from previous failed test, revert first
                            if (rows[rowIndex][testData.valueColumnIndex] === testData.updateValue) {
                                await whodb.updateRow(rowIndex, 1, testData.originalValue, false);
                            }

                            // Edit the hash field - columnIndex 1 triggers edit, value goes to column 2
                            await whodb.updateRow(rowIndex, 1, testData.updateValue, false);

                            let { rows: updatedRows } = await whodb.getTableData();
                            expect(updatedRows[rowIndex][testData.valueColumnIndex]).toEqual(testData.updateValue);

                            // Revert
                            await whodb.updateRow(rowIndex, 1, testData.originalValue, false);

                            ({ rows: updatedRows } = await whodb.getTableData());
                            expect(updatedRows[rowIndex][testData.valueColumnIndex]).toEqual(testData.originalValue);
                        });
                    }
                }

                if (keyType === 'list') {
                    test('DATA - displays list entries with indices', async ({ whodb, page }) => {
                        await whodb.data(testKey);

                        const { rows } = await whodb.getTableData();
                        expect(rows.length).toBeGreaterThan(0);

                        // Verify index column contains numeric indices
                        rows.forEach((row, i) => {
                            const indexValue = row[testData.indexColumnIndex];
                            expect(indexValue).toEqual(String(i));
                        });
                    });

                    if (supportsUpdate) {
                        test('UPDATE - modifies list entry at index', async ({ whodb, page }) => {
                            await whodb.data(testKey);

                            const { rows } = await whodb.getTableData();
                            const originalValue = rows[testData.testIndex][testData.valueColumnIndex];

                            // Edit list item - columnIndex 1 triggers edit for the index row
                            await whodb.updateRow(testData.testIndex, 1, 'keytype_test_value', false);

                            let { rows: updatedRows } = await whodb.getTableData();
                            expect(updatedRows[testData.testIndex][testData.valueColumnIndex]).toEqual('keytype_test_value');

                            // Revert
                            await whodb.updateRow(testData.testIndex, 1, originalValue, false);

                            ({ rows: updatedRows } = await whodb.getTableData());
                            expect(updatedRows[testData.testIndex][testData.valueColumnIndex]).toEqual(originalValue);
                        });
                    }
                }

                if (keyType === 'set') {
                    test('DATA - displays set members correctly', async ({ whodb, page }) => {
                        await whodb.data(testKey);

                        const { rows } = await whodb.getTableData();
                        expect(rows.length).toBeGreaterThan(0);

                        // Verify expected members are present
                        if (testData.expectedMembers) {
                            const actualMembers = rows.map(r => r[testData.valueColumnIndex]);
                            testData.expectedMembers.forEach(member => {
                                expect(actualMembers, `Set should contain member: ${member}`).toContain(member);
                            });
                        }
                    });

                    // Sets don't support UPDATE - members can only be added/removed
                }

                if (keyType === 'zset') {
                    test('DATA - displays sorted set with members and scores', async ({ whodb, page }) => {
                        await whodb.data(testKey);

                        const { rows } = await whodb.getTableData();
                        expect(rows.length).toBeGreaterThan(0);

                        // Verify each row has index, member, and score columns
                        rows.forEach((row, i) => {
                            const index = row[testData.indexColumnIndex];
                            const member = row[testData.memberColumnIndex];
                            const score = row[testData.scoreColumnIndex];

                            expect(index).toEqual(String(i));
                            expect(typeof member).toEqual('string');
                            expect(member.length).toBeGreaterThan(0);
                            expect(score).toMatch(/^-?\d+(\.\d+)?$/);
                        });
                    });

                    // Sorted sets don't support UPDATE through WhoDB's interface
                }
            });
        });
    });
});
