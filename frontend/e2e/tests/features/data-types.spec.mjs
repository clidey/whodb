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

function rowSignature(row) {
    return JSON.stringify(row.map(value => String(value ?? '').trim()));
}

async function waitForNewRowValue(whodb, columnIndex, expectedValue, existingSignatures, timeout = 10000) {
    const expectedStr = String(expectedValue).trim();
    let foundIndex = -1;

    await expect(async () => {
        const { rows } = await whodb.getTableData();
        foundIndex = rows.findIndex(row => {
            return !existingSignatures.has(rowSignature(row)) &&
                String(row[columnIndex] || '').trim() === expectedStr;
        });
        expect(foundIndex).not.toEqual(-1);
    }).toPass({ timeout });

    return foundIndex;
}

/**
 * For databases with asynchronous mutations (e.g. ClickHouse), reload the table until
 * `isSettled(rows)` holds, waiting `mutationDelay` between reloads. Gives up after `timeout`
 * so the caller's own assertion reports the mismatch.
 */
async function settleAsyncMutation(whodb, page, tableName, mutationDelay, isSettled, timeout = 30000) {
    if (mutationDelay <= 0) return;
    const deadline = Date.now() + timeout;
    for (;;) {
        await page.waitForTimeout(mutationDelay);
        await whodb.data(tableName);
        await whodb.sortBy(0);
        const { rows } = await whodb.getTableData();
        if (isSettled(rows) || Date.now() > deadline) return;
    }
}

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
                    const columnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());

                    const seedRowIndex = rows.findIndex(r => {
                        const cellValue = String(r[columnIndex + 1] || '').trim();
                        return cellValue === expectedOriginal;
                    });

                    expect(
                        seedRowIndex,
                        `Seed data with ${columnName}=${expectedOriginal} should exist. Actual values: ${JSON.stringify(columnValues)}`
                    ).not.toEqual(-1);

                    const actualValue = String(rows[seedRowIndex][columnIndex + 1] || '').trim();
                    expect(actualValue).toEqual(expectedOriginal);
                });

                test('ADD - creates row with type value', async ({ whodb, page }) => {
                    await whodb.data(tableName);

                    const { rows: existingRows } = await whodb.getTableData();
                    const existingSignatures = new Set(existingRows.map(rowSignature));

                    const verifyAdd = waitForMutation(page, 'AddRow');
                    await whodb.addRow({[columnName]: testConfig.addValue});
                    await verifyAdd();

                    // Wait for async mutations (e.g., ClickHouse)
                    await settleAsyncMutation(whodb, page, tableName, mutationDelay, rows =>
                        rows.some(row => !existingSignatures.has(rowSignature(row)) && String(row[columnIndex + 1] || '').trim() === String(expectedAddDisplay).trim()));

                    const rowIndex = await waitForNewRowValue(
                        whodb,
                        columnIndex + 1,
                        expectedAddDisplay,
                        existingSignatures
                    );

                    // Clean up - delete the row we just added
                    const verifyDelete = waitForMutation(page, 'DeleteRow');
                    await whodb.deleteRow(rowIndex);
                    await verifyDelete();

                    // Wait for the async delete so the next test starts from seed data
                    await settleAsyncMutation(whodb, page, tableName, mutationDelay, rows => rows.length === existingRows.length);
                });

                test('UPDATE - edits type value', async ({ whodb, page }) => {
                    const originalValue = String(testConfig.originalValue).trim();
                    const updateDisplayValue = String(expectedUpdateDisplay).trim();

                    await test.step('navigate to table', async () => {
                        await whodb.data(tableName);
                        await whodb.sortBy(0);
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

                        await settleAsyncMutation(whodb, page, tableName, mutationDelay, rows =>
                            rows.some(r => String(r[columnIndex + 1] || '').trim() === updateDisplayValue));
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
                });

                test('DELETE - removes row with type value', async ({ whodb, page }) => {
                    await whodb.data(tableName);

                    const { rows: existingRows } = await whodb.getTableData();
                    const existingSignatures = new Set(existingRows.map(rowSignature));

                    const verifyAdd = waitForMutation(page, 'AddRow');
                    await whodb.addRow({[columnName]: deleteValue});
                    await verifyAdd();

                    // Wait for async mutations (e.g., ClickHouse)
                    await settleAsyncMutation(whodb, page, tableName, mutationDelay, rows =>
                        rows.some(row => !existingSignatures.has(rowSignature(row)) && String(row[columnIndex + 1] || '').trim() === String(expectedDeleteDisplay).trim()));

                    const rowIndex = await waitForNewRowValue(
                        whodb,
                        columnIndex + 1,
                        expectedDeleteDisplay,
                        existingSignatures
                    );

                    // Get initial count after the row has appeared
                    const { rows } = await whodb.getTableData();
                    const initialCount = rows.length;

                    const verifyDelete = waitForMutation(page, 'DeleteRow');
                    await whodb.deleteRow(rowIndex);
                    await verifyDelete();

                    // Wait for async mutations (e.g., ClickHouse)
                    await settleAsyncMutation(whodb, page, tableName, mutationDelay, rows => rows.length === initialCount - 1);

                    // Verify row count decreased
                    const { rows: newRows } = await whodb.getTableData();
                    expect(newRows.length).toEqual(initialCount - 1);
                });
            });
        });
    });
});
