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
import { waitForMutation } from '../../support/helpers/test-utils.mjs';

/**
 * Data Types CRUD Operations Test Suite
 *
 * Tests that each SQL database can correctly handle CRUD operations for all
 * supported data types. Each type is tested independently with ADD, UPDATE,
 * and DELETE operations.
 */
test.describe('Data Types CRUD Operations', () => {

    forEachDatabase('sql', (db) => {
        const tableName = db.dataTypesTable;
        const tableConfig = tableName ? getTableConfig(db, tableName) : null;
        const mutationDelay = db.mutationDelay || 0;

        if (!tableConfig) {
            test.skip('data_types table config missing in fixture', async ({ whodb, page }) => {
            });
            return;
        }

        const typeTests = tableConfig.testData?.typeTests;
        if (!typeTests) {
            test.skip('typeTests config missing in fixture', async ({ whodb, page }) => {
            });
            return;
        }

        Object.entries(typeTests).forEach(([columnName, testConfig]) => {
            test.describe(`Type: ${testConfig.type} (${columnName})`, () => {
                const columnIndex = Object.keys(tableConfig.columns).indexOf(columnName);
                const expectedAddDisplay = testConfig.displayAddValue || testConfig.addValue;
                const expectedUpdateDisplay = testConfig.displayUpdateValue || testConfig.updateValue;
                // DELETE test uses separate values to avoid conflicts with ADD test
                const deleteValue = testConfig.deleteValue || testConfig.addValue;
                const expectedDeleteDisplay = testConfig.displayDeleteValue || testConfig.displayAddValue || deleteValue;

                test('READ - displays seed data with correct format', async ({ whodb, page }) => {
                    await whodb.data(tableName);
                    await whodb.sortBy(0);

                    const { rows } = await whodb.getTableData();
                    if (rows.length === 0) {
                        throw new Error('No rows in data_types table');
                    }

                    const expectedOriginal = String(testConfig.originalValue).trim();
                    const expectedUpdate = String(testConfig.displayUpdateValue || testConfig.updateValue).trim();
                    const columnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());

                    // Accept either original or update value (handles leftover state from failed UPDATE tests)
                    const seedRowIndex = rows.findIndex(r => {
                        const cellValue = String(r[columnIndex + 1] || '').trim();
                        return cellValue === expectedOriginal || cellValue === expectedUpdate;
                    });

                    expect(
                        seedRowIndex,
                        `Seed data with ${columnName}=${expectedOriginal} or ${expectedUpdate} should exist. Actual values: ${JSON.stringify(columnValues)}`
                    ).not.toEqual(-1);

                    // Verify the value matches one of the expected formats
                    const actualValue = String(rows[seedRowIndex][columnIndex + 1] || '').trim();
                    expect(
                        actualValue === expectedOriginal || actualValue === expectedUpdate,
                        `${testConfig.type} should display as "${expectedOriginal}" or "${expectedUpdate}", got "${actualValue}"`
                    ).toBeTruthy();
                });

                test('ADD - creates row with type value', async ({ whodb, page }) => {
                    await whodb.data(tableName);

                    const verifyAdd = waitForMutation(page, 'AddRow');
                    await whodb.addRow({[columnName]: testConfig.addValue});
                    await verifyAdd();

                    // Wait for async mutations (e.g., ClickHouse)
                    if (mutationDelay > 0) {
                        await page.waitForTimeout(mutationDelay);
                        await whodb.data(tableName);
                        await whodb.sortBy(0);
                    }

                    // Use retry-able assertion to wait for the new row to appear
                    // columnIndex + 1 accounts for the checkbox column
                    const rowIndex = await whodb.waitForRowValue(columnIndex + 1, expectedAddDisplay);

                    // Clean up - delete the row we just added
                    const verifyDelete = waitForMutation(page, 'DeleteRow');
                    await whodb.deleteRow(rowIndex);
                    await verifyDelete();

                    // Wait for async delete mutation
                    if (mutationDelay > 0) {
                        await page.waitForTimeout(mutationDelay);
                    }
                });

                test('UPDATE - edits type value', async ({ whodb, page }) => {
                    const originalValue = String(testConfig.originalValue).trim();
                    const updateDisplayValue = String(expectedUpdateDisplay).trim();
                    const revertValue = testConfig.inputOriginalValue || testConfig.originalValue;

                    await test.step('navigate to table', async () => {
                        await whodb.data(tableName);
                        await whodb.sortBy(0);
                    });

                    await test.step('check and revert leftover data', async () => {
                        let tableData = await whodb.getTableData();
                        let rows = tableData.rows;
                        if (rows.length === 0) {
                            throw new Error('No rows in data_types table');
                        }

                        const columnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());

                        let targetRowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === originalValue;
                        });

                        if (targetRowIndex === -1) {
                            targetRowIndex = rows.findIndex(r => {
                                const cellValue = String(r[columnIndex + 1] || '').trim();
                                return cellValue === updateDisplayValue;
                            });
                            if (targetRowIndex !== -1) {
                                await whodb.updateRow(targetRowIndex, columnIndex, revertValue, false);

                                if (mutationDelay > 0) {
                                    await page.waitForTimeout(mutationDelay);
                                    await whodb.data(tableName);
                                    await whodb.sortBy(0);
                                }

                                await whodb.waitForRowValue(columnIndex + 1, originalValue);
                            }
                        }

                        if (targetRowIndex === -1) {
                            throw new Error(`Row with value "${originalValue}" or "${updateDisplayValue}" not found in column ${columnName}. Actual values: ${JSON.stringify(columnValues)}`);
                        }
                    });

                    await test.step('perform update with network verification', async () => {
                        const tableData = await whodb.getTableData();
                        const rows = tableData.rows;
                        const finalTargetRowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === originalValue;
                        });

                        if (finalTargetRowIndex === -1) {
                            const finalColumnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());
                            throw new Error(`Row with original value "${originalValue}" not found. Actual values: ${JSON.stringify(finalColumnValues)}`);
                        }

                        const verifyUpdate = waitForMutation(page, 'UpdateStorageUnit');
                        await whodb.updateRow(finalTargetRowIndex, columnIndex, testConfig.updateValue, false);
                        await verifyUpdate();

                        if (mutationDelay > 0) {
                            await page.waitForTimeout(mutationDelay);
                            await whodb.data(tableName);
                            await whodb.sortBy(0);
                        }
                    });

                    await test.step('verify update', async () => {
                        await whodb.waitForRowValue(columnIndex + 1, updateDisplayValue);

                        const { rows: updatedRows } = await whodb.getTableData();
                        const targetIdx = updatedRows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === updateDisplayValue;
                        });
                        expect(targetIdx, 'Updated row should exist').not.toEqual(-1);
                        const cellValue = String(updatedRows[targetIdx][columnIndex + 1] || '').trim();
                        expect(cellValue).toEqual(updateDisplayValue);
                    });

                    await test.step('revert to original', async () => {
                        const { rows } = await whodb.getTableData();
                        const revertIdx = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === updateDisplayValue;
                        });
                        await whodb.updateRow(revertIdx, columnIndex, revertValue, false);

                        if (mutationDelay > 0) {
                            await page.waitForTimeout(mutationDelay);
                            await whodb.data(tableName);
                            await whodb.sortBy(0);
                        }

                        await whodb.waitForRowValue(columnIndex + 1, originalValue);
                    });
                });

                test('DELETE - removes row with type value', async ({ whodb, page }) => {
                    await whodb.data(tableName);

                    const verifyAdd = waitForMutation(page, 'AddRow');
                    await whodb.addRow({[columnName]: deleteValue});
                    await verifyAdd();

                    // Wait for async mutations (e.g., ClickHouse)
                    if (mutationDelay > 0) {
                        await page.waitForTimeout(mutationDelay);
                        await whodb.data(tableName);
                        await whodb.sortBy(0);
                    }

                    // Use retry-able assertion to wait for the new row to appear
                    const rowIndex = await whodb.waitForRowValue(columnIndex + 1, expectedDeleteDisplay);

                    // Get initial count after the row has appeared
                    const { rows } = await whodb.getTableData();
                    const initialCount = rows.length;

                    const verifyDelete = waitForMutation(page, 'DeleteRow');
                    await whodb.deleteRow(rowIndex);
                    await verifyDelete();

                    // Wait for async mutations (e.g., ClickHouse)
                    if (mutationDelay > 0) {
                        await page.waitForTimeout(mutationDelay);
                        await whodb.data(tableName);
                        await whodb.sortBy(0);
                    }

                    // Verify row count decreased
                    const { rows: newRows } = await whodb.getTableData();
                    expect(newRows.length).toEqual(initialCount - 1);
                });
            });
        });
    });
});
