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
import { getTableConfig, hasFeature } from '../../support/database-config.mjs';
import { createUpdatedDocument, parseDocument } from '../../support/categories/document.mjs';

/**
 * Generates a unique test identifier for this test run.
 * Uses timestamp to avoid conflicts within and across test runs.
 */
function getUniqueTestId() {
    return `test_${Date.now()}`;
}

test.describe('CRUD Operations', () => {

    // SQL Databases - Edit and Add
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        if (!testTable) {
            test.skip('testTable config missing in fixture', async ({ whodb, page }) => {
            });
            return;
        }

        const tableName = testTable.name;
        const colIndex = testTable.identifierColIndex;
        const testValues = testTable.testValues;
        const mutationDelay = db.mutationDelay || 0;

        // Skip Edit Row tests for databases with async mutations (e.g., ClickHouse)
        const crudSupported = hasFeature(db, 'crud') !== false;

        test.describe('Edit Row', () => {
            if (!crudSupported) {
                test.skip('edits a row and saves changes', async ({ whodb, page }) => {});
                test.skip('cancels edit without saving', async ({ whodb, page }) => {});
                return;
            }

            test('edits a row and saves changes', async ({ whodb, page }) => {
                await whodb.data(tableName);
                await whodb.sortBy(0);

                // Edit row with network verification
                const updateResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') &&
                    resp.request().postDataJSON?.()?.operationName === 'UpdateStorageUnit'
                );
                await whodb.updateRow(testValues.rowIndex, colIndex, testValues.modified, false);
                const updateResponse = await updateResponsePromise;
                const updateResult = await updateResponse.json();
                expect(updateResult.errors, 'UpdateStorageUnit mutation should succeed').toBeUndefined();

                // Wait for async mutations (e.g., ClickHouse)
                if (mutationDelay > 0) {
                    await page.waitForTimeout(mutationDelay);
                    await whodb.data(tableName);
                    await whodb.sortBy(0);
                }

                // Verify change
                let { rows } = await whodb.getTableData();
                expect(rows[testValues.rowIndex][colIndex + 1]).toEqual(testValues.modified);

                // Revert
                await whodb.updateRow(testValues.rowIndex, colIndex, testValues.original, false);

                // Wait for async mutations
                if (mutationDelay > 0) {
                    await page.waitForTimeout(mutationDelay);
                    await whodb.data(tableName);
                    await whodb.sortBy(0);
                }

                ({ rows } = await whodb.getTableData());
                expect(rows[testValues.rowIndex][colIndex + 1]).toEqual(testValues.original);
            });

            test('cancels edit without saving', async ({ whodb, page }) => {
                await whodb.data(tableName);
                await whodb.sortBy(0);

                // Edit and cancel
                await whodb.updateRow(testValues.rowIndex, colIndex, 'temp_value', true);

                // Verify no change
                const { rows } = await whodb.getTableData();
                expect(rows[testValues.rowIndex][colIndex + 1]).toEqual(testValues.original);
            });
        });

        test.describe('Add Row', () => {
            test('adds a new row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                const tableConfig = getTableConfig(db, tableName);
                const newRowTemplate = tableConfig?.testData?.newRow;

                if (!newRowTemplate) {
                    // No newRow test data configured, skipping
                    return;
                }

                // Create unique row data to avoid conflicts
                const uniqueId = getUniqueTestId();
                const newRowData = {...newRowTemplate};
                newRowData[testTable.identifierField] = `${uniqueId}_user`;
                if (newRowData.email) {
                    newRowData.email = `${uniqueId}@example.com`;
                }

                const addResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') &&
                    resp.request().postDataJSON?.()?.operationName === 'AddRow'
                );
                await whodb.addRow(newRowData);
                const addResponse = await addResponsePromise;
                const addResult = await addResponse.json();
                expect(addResult.errors, 'AddRow mutation should succeed').toBeUndefined();

                const identifierValue = newRowData[testTable.identifierField];

                // Wait for row to appear using retry-able assertion
                const rowIndex = await whodb.waitForRowValue(colIndex + 1, identifierValue);

                // Clean up - delete the added row
                const deleteResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') &&
                    resp.request().postDataJSON?.()?.operationName === 'DeleteRow'
                );
                await whodb.deleteRow(rowIndex);
                const deleteResponse = await deleteResponsePromise;
                const deleteResult = await deleteResponse.json();
                expect(deleteResult.errors, 'DeleteRow mutation should succeed').toBeUndefined();
            });
        });

        test.describe('Delete Row', () => {
            test('deletes a row and verifies removal', async ({ whodb, page }) => {
                await whodb.data(tableName);

                const tableConfig = getTableConfig(db, tableName);
                const newRowTemplate = tableConfig?.testData?.newRow;

                if (!newRowTemplate) {
                    // No newRow test data configured, skipping
                    return;
                }

                // Create unique row data
                const uniqueId = getUniqueTestId();
                const newRowData = {...newRowTemplate};
                newRowData[testTable.identifierField] = `${uniqueId}_delete`;
                if (newRowData.email) {
                    newRowData.email = `${uniqueId}@example.com`;
                }

                const addResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') &&
                    resp.request().postDataJSON?.()?.operationName === 'AddRow'
                );
                await whodb.addRow(newRowData);
                const addResponse = await addResponsePromise;
                const addResult = await addResponse.json();
                expect(addResult.errors, 'AddRow mutation should succeed').toBeUndefined();

                const identifierValue = newRowData[testTable.identifierField];

                // Wait for row to appear using retry-able assertion
                const rowIndex = await whodb.waitForRowValue(colIndex + 1, identifierValue);

                const { rows } = await whodb.getTableData();
                const initialCount = rows.length;

                const deleteResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') &&
                    resp.request().postDataJSON?.()?.operationName === 'DeleteRow'
                );
                await whodb.deleteRow(rowIndex);
                const deleteResponse = await deleteResponsePromise;
                const deleteResult = await deleteResponse.json();
                expect(deleteResult.errors, 'DeleteRow mutation should succeed').toBeUndefined();

                // Verify row count decreased
                const { rows: newRows } = await whodb.getTableData();
                expect(newRows.length).toEqual(initialCount - 1);
            });
        });
    });

    // Document Databases - Add, Edit, Delete
    forEachDatabase('document', (db) => {
        const testTable = db.testTable;
        if (!testTable) {
            test.skip('testTable config missing in fixture', async ({ whodb, page }) => {
            });
            return;
        }

        const tableName = testTable.name;
        const testValues = testTable.testValues;
        const refreshDelay = db.indexRefreshDelay || 0;

        test.describe('Add Document', () => {
            test('adds a new document', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Create unique document to avoid conflicts
                const uniqueId = getUniqueTestId();
                const newDoc = {
                    username: `${uniqueId}_user`,
                    email: `${uniqueId}@example.com`,
                    password: 'newpassword'
                };

                await whodb.addRow(newDoc, true);

                const addResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') &&
                    resp.request().postDataJSON?.()?.operationName === 'AddRow'
                );
                await whodb.submitTable();
                const addResponse = await addResponsePromise;
                const addResult = await addResponse.json();
                expect(addResult.errors, 'AddRow mutation should succeed').toBeUndefined();

                // Wait for row to appear using retry-able assertion
                const rowIndex = await whodb.waitForRowContaining(uniqueId);

                // Clean up
                const deleteResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') &&
                    resp.request().postDataJSON?.()?.operationName === 'DeleteRow'
                );
                await whodb.deleteRow(rowIndex);
                const deleteResponse = await deleteResponsePromise;
                const deleteResult = await deleteResponse.json();
                expect(deleteResult.errors, 'DeleteRow mutation should succeed').toBeUndefined();
            });
        });

        test.describe('Edit Document', () => {
            // Skip full edit test for Elasticsearch due to truncated JSON display
            if (db.type === 'ElasticSearch') {
                test('cancels edit without saving', async ({ whodb, page }) => {
                    await whodb.data(tableName);

                    await page.locator('table tbody tr', { timeout: 15000 }).first().waitFor();

                    const { rows } = await whodb.getTableData();
                    const targetRowIndex = rows.findIndex(r => {
                        const text = (r[1] || '').toLowerCase();
                        return text.includes(testValues.original);
                    });
                    expect(targetRowIndex, `Row with ${testValues.original} should exist`).toBeGreaterThan(-1);

                    await whodb.openContextMenu(targetRowIndex);
                    await expect(page.locator('[data-testid="context-menu-edit-row"]')).toBeVisible();
                    await page.locator('[data-testid="context-menu-edit-row"]').click();
                    await expect(page.getByText('Edit Row').first()).toBeVisible();
                    await page.keyboard.press('Escape');
                    await expect(page.getByText('Edit Row').first()).not.toBeAttached();
                });
                return;
            }

            test('edits a document and saves changes', async ({ whodb, page }) => {
                await test.step('navigate to table', async () => {
                    await whodb.data(tableName);
                    await whodb.sortBy(0);
                });

                await test.step('check and revert leftover data', async () => {
                    const { rows } = await whodb.getTableData();
                    const doc = parseDocument(rows[testValues.rowIndex]);
                    const currentValue = doc[testTable.identifierField];

                    if (currentValue === testValues.modified) {
                        const revertDoc = createUpdatedDocument(rows[testValues.rowIndex], {
                            [testTable.identifierField]: testValues.original
                        });
                        await whodb.updateRow(testValues.rowIndex, 1, revertDoc, false);
                        if (refreshDelay > 0) {
                            await page.waitForTimeout(refreshDelay);
                            await whodb.data(tableName);
                            await whodb.sortBy(0);
                        }
                    }
                });

                await test.step('perform edit with network verification', async () => {
                    const { rows } = await whodb.getTableData();
                    const updatedDoc = createUpdatedDocument(rows[testValues.rowIndex], {
                        [testTable.identifierField]: testValues.modified
                    });

                    const responsePromise = page.waitForResponse(resp =>
                        resp.url().includes('/api/query') &&
                        resp.request().postDataJSON?.()?.operationName === 'UpdateStorageUnit'
                    );
                    await whodb.updateRow(testValues.rowIndex, 1, updatedDoc, false);
                    const response = await responsePromise;
                    const result = await response.json();
                    expect(result.errors, 'UpdateStorageUnit mutation should succeed').toBeUndefined();

                    if (refreshDelay > 0) {
                        await page.waitForTimeout(refreshDelay);
                        await whodb.data(tableName);
                        await whodb.sortBy(0);
                    }
                });

                await test.step('verify edit', async () => {
                    const { rows } = await whodb.getTableData();
                    const editedDoc = parseDocument(rows[testValues.rowIndex]);
                    expect(editedDoc[testTable.identifierField]).toEqual(testValues.modified);
                });

                await test.step('revert to original', async () => {
                    const { rows } = await whodb.getTableData();
                    const revertedDoc = createUpdatedDocument(rows[testValues.rowIndex], {
                        [testTable.identifierField]: testValues.original
                    });
                    await whodb.updateRow(testValues.rowIndex, 1, revertedDoc, false);

                    if (refreshDelay > 0) {
                        await page.waitForTimeout(refreshDelay);
                        await whodb.data(tableName);
                        await whodb.sortBy(0);
                    }

                    const { rows: revertedRows } = await whodb.getTableData();
                    const revertedParsedDoc = parseDocument(revertedRows[testValues.rowIndex]);
                    expect(revertedParsedDoc[testTable.identifierField]).toEqual(testValues.original);
                });
            });

            test('cancels edit without saving', async ({ whodb, page }) => {
                await whodb.data(tableName);
                await whodb.sortBy(0);

                const { rows } = await whodb.getTableData();
                const doc = parseDocument(rows[testValues.rowIndex]);
                const currentValue = doc[testTable.identifierField];

                // Store the current value for later verification (might be original or modified from failed test)
                const updatedDoc = createUpdatedDocument(rows[testValues.rowIndex], {
                    [testTable.identifierField]: 'temp_value'
                });
                await whodb.updateRow(testValues.rowIndex, 1, updatedDoc, true);

                // Verify the value didn't change (should still be whatever it was before)
                const { rows: verifyRows } = await whodb.getTableData();
                const verifyDoc = parseDocument(verifyRows[testValues.rowIndex]);
                expect(verifyDoc[testTable.identifierField]).toEqual(currentValue);
            });
        });

        test.describe('Delete Document', () => {
            test('deletes a document and verifies removal', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Create unique document to delete
                const uniqueId = getUniqueTestId();
                const newDoc = {
                    username: `${uniqueId}_delete`,
                    email: `${uniqueId}@example.com`,
                    password: 'temppass'
                };

                await whodb.addRow(newDoc, true);

                // Wait for row to appear using retry-able assertion
                const rowIndex = await whodb.waitForRowContaining(uniqueId);

                const { rows } = await whodb.getTableData();
                const initialCount = rows.length;

                const deleteResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') &&
                    resp.request().postDataJSON?.()?.operationName === 'DeleteRow'
                );
                await whodb.deleteRow(rowIndex);
                const deleteResponse = await deleteResponsePromise;
                const deleteResult = await deleteResponse.json();
                expect(deleteResult.errors, 'DeleteRow mutation should succeed').toBeUndefined();

                // Verify row count decreased
                const { rows: newRows } = await whodb.getTableData();
                expect(newRows.length).toEqual(initialCount - 1);
            });
        });
    });

    // Key-Value Databases - Edit hash fields
    forEachDatabase('keyvalue', (db) => {
        const testTable = db.testTable;
        if (!testTable) {
            test.skip('testTable config missing in fixture', async ({ whodb, page }) => {
            });
            return;
        }

        const keyName = testTable.name;
        const testValues = testTable.testValues;
        const rowIndex = testTable.identifierRowIndex || testValues.rowIndex;

        test.describe('Edit Hash Field', () => {
            test('edits a hash field value and saves', async ({ whodb, page }) => {
                await test.step('navigate to key', async () => {
                    await whodb.data(keyName);
                });

                await test.step('check and revert leftover data', async () => {
                    const { rows } = await whodb.getTableData();
                    if (rows[rowIndex][2] === testValues.modified) {
                        await whodb.updateRow(rowIndex, 1, testValues.original, false);
                    }
                });

                await test.step('update field with network verification', async () => {
                    const responsePromise = page.waitForResponse(resp =>
                        resp.url().includes('/api/query') &&
                        resp.request().postDataJSON?.()?.operationName === 'UpdateStorageUnit'
                    );
                    await whodb.updateRow(rowIndex, 1, testValues.modified, false);
                    const response = await responsePromise;
                    const result = await response.json();
                    expect(result.errors, 'UpdateStorageUnit mutation should succeed').toBeUndefined();

                    const { rows } = await whodb.getTableData();
                    expect(rows[rowIndex][2]).toEqual(testValues.modified);
                });

                await test.step('revert to original', async () => {
                    await whodb.updateRow(rowIndex, 1, testValues.original, false);

                    const { rows } = await whodb.getTableData();
                    expect(rows[rowIndex][2]).toEqual(testValues.original);
                });
            });

            test('cancels edit without saving', async ({ whodb, page }) => {
                await whodb.data(keyName);

                // Store current value for verification (might be original or modified from failed test)
                const { rows } = await whodb.getTableData();
                const currentValue = rows[rowIndex][2];

                await whodb.updateRow(rowIndex, 1, 'temp_value', true);

                const { rows: verifyRows } = await whodb.getTableData();
                expect(verifyRows[rowIndex][2]).toEqual(currentValue);
            });
        });

        test.describe('Delete Hash Field', () => {
            test('deletes a hash field', async ({ whodb, page }) => {
                // Use user:2 for delete test to avoid affecting user:1 used in edit tests
                await whodb.data('user:2');

                const { rows } = await whodb.getTableData();
                const initialCount = rows.length;

                const deleteResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') &&
                    resp.request().postDataJSON?.()?.operationName === 'DeleteRow'
                );
                await whodb.deleteRow(2);
                const deleteResponse = await deleteResponsePromise;
                const deleteResult = await deleteResponse.json();
                expect(deleteResult.errors, 'DeleteRow mutation should succeed').toBeUndefined();

                const { rows: newRows } = await whodb.getTableData();
                expect(newRows.length).toEqual(initialCount - 1);
            });
        });
    });

});
