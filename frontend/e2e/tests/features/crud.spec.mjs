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
import { getTableConfig, hasFeature } from '../../support/database-config.mjs';
import { createUpdatedDocument, parseDocument } from '../../support/categories/document.mjs';
import { getUniqueTestId, TIMEOUT, waitForMutation } from '../../support/helpers/test-utils.mjs';

let uniqueRowIdCounter = 0;

function createUniqueRow(template, testTable, suffix) {
    const uniqueId = getUniqueTestId();
    const row = { ...template };
    if (testTable.idField && row[testTable.idField] !== undefined) {
        uniqueRowIdCounter += 1;
        row[testTable.idField] = String(900000 + ((Date.now() % 10000) * 10) + uniqueRowIdCounter);
    }
    row[testTable.identifierField] = `${uniqueId}_${suffix}`;
    if (row.email) {
        row.email = `${uniqueId}@example.com`;
    }
    return row;
}

async function deleteDocumentRow(whodb, page, rowIndex, refreshDelay = 0) {
    const verifyDelete = waitForMutation(page, 'DeleteRow');
    await whodb.deleteRow(rowIndex, { waitForRowCount: false });
    await verifyDelete();
    if (refreshDelay > 0) {
        await page.waitForTimeout(refreshDelay);
    }
}

async function expectDocumentAbsent(whodb, tableName, text) {
    await whodb.data(tableName);
    await expect(async () => {
        const { rows } = await whodb.getTableData();
        expect(rows.some(row => row.join('\n').includes(text))).toBe(false);
    }).toPass({ timeout: TIMEOUT.SLOW });
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

                const tableConfig = getTableConfig(db, tableName);
                const newRowTemplate = tableConfig?.testData?.newRow;
                if (!newRowTemplate) {
                    return;
                }

                const newRowData = createUniqueRow(newRowTemplate, testTable, 'edit');
                const originalIdentifier = newRowData[testTable.identifierField];
                const modifiedIdentifier = `${originalIdentifier}_updated`;

                const verifyAdd = waitForMutation(page, 'AddRow');
                await whodb.addRow(newRowData);
                await verifyAdd();

                const addedRowIndex = await whodb.waitForRowValue(colIndex + 1, originalIdentifier);

                const verifyUpdate = waitForMutation(page, 'UpdateStorageUnit');
                await whodb.updateRow(addedRowIndex, colIndex, modifiedIdentifier, false);
                await verifyUpdate();

                if (mutationDelay > 0) {
                    await page.waitForTimeout(mutationDelay);
                    await whodb.data(tableName);
                }

                const updatedRowIndex = await whodb.waitForRowValue(colIndex + 1, modifiedIdentifier);
                const { rows } = await whodb.getTableData();
                expect(rows[updatedRowIndex][colIndex + 1]).toEqual(modifiedIdentifier);

                const verifyDelete = waitForMutation(page, 'DeleteRow');
                await whodb.deleteRow(updatedRowIndex);
                await verifyDelete();
            });

            test('cancels edit without saving', async ({ whodb, page }) => {
                await whodb.data(tableName);

                const tableConfig = getTableConfig(db, tableName);
                const newRowTemplate = tableConfig?.testData?.newRow;
                if (!newRowTemplate) {
                    return;
                }

                const newRowData = createUniqueRow(newRowTemplate, testTable, 'cancel');
                const identifierValue = newRowData[testTable.identifierField];

                const verifyAdd = waitForMutation(page, 'AddRow');
                await whodb.addRow(newRowData);
                await verifyAdd();

                const addedRowIndex = await whodb.waitForRowValue(colIndex + 1, identifierValue);

                await whodb.updateRow(addedRowIndex, colIndex, `${identifierValue}_cancelled`, true);

                const { rows } = await whodb.getTableData();
                expect(rows[addedRowIndex][colIndex + 1]).toEqual(identifierValue);

                const verifyDelete = waitForMutation(page, 'DeleteRow');
                await whodb.deleteRow(addedRowIndex);
                await verifyDelete();
            });
        });

        test.describe('Add Row', () => {
            if (!crudSupported) {
                test.skip('adds a new row', async ({ whodb, page }) => {});
                return;
            }
            test('adds a new row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                const tableConfig = getTableConfig(db, tableName);
                const newRowTemplate = tableConfig?.testData?.newRow;

                if (!newRowTemplate) {
                    // No newRow test data configured, skipping
                    return;
                }

                const newRowData = createUniqueRow(newRowTemplate, testTable, 'user');

                const verifyAdd = waitForMutation(page, 'AddRow');
                await whodb.addRow(newRowData);
                await verifyAdd();

                const identifierValue = newRowData[testTable.identifierField];

                // Wait for row to appear using retry-able assertion
                const rowIndex = await whodb.waitForRowValue(colIndex + 1, identifierValue);

                // Clean up - delete the added row
                const verifyDelete = waitForMutation(page, 'DeleteRow');
                await whodb.deleteRow(rowIndex);
                await verifyDelete();
            });
        });

        test.describe('Delete Row', () => {
            if (!crudSupported) {
                test.skip('deletes a row and verifies removal', async ({ whodb, page }) => {});
                return;
            }
            test('deletes a row and verifies removal', async ({ whodb, page }) => {
                await whodb.data(tableName);

                const tableConfig = getTableConfig(db, tableName);
                const newRowTemplate = tableConfig?.testData?.newRow;

                if (!newRowTemplate) {
                    // No newRow test data configured, skipping
                    return;
                }

                const newRowData = createUniqueRow(newRowTemplate, testTable, 'delete');

                const verifyAdd = waitForMutation(page, 'AddRow');
                await whodb.addRow(newRowData);
                await verifyAdd();

                const identifierValue = newRowData[testTable.identifierField];

                // Wait for row to appear using retry-able assertion
                const rowIndex = await whodb.waitForRowValue(colIndex + 1, identifierValue);

                const { rows } = await whodb.getTableData();
                const initialCount = rows.length;

                const verifyDelete = waitForMutation(page, 'DeleteRow');
                await whodb.deleteRow(rowIndex);
                await verifyDelete();

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

                // Wait for row to appear using retry-able assertion
                const rowIndex = await whodb.waitForRowContaining(uniqueId);

                // Clean up
                const verifyDelete = waitForMutation(page, 'DeleteRow');
                await whodb.deleteRow(rowIndex);
                await verifyDelete();
            });
        });

        test.describe('Edit Document', () => {
            // Skip full edit test for databases that don't support document editing
            if (skipIfNoFeature(db, 'documentEdit')) {
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
                    await page.locator('[data-testid="cancel-edit-row"]').click();
                    await expect(page.getByText('Edit Row').first()).not.toBeAttached();
                });
                return;
            }

            test('edits a document and saves changes', async ({ whodb, page }) => {
                await test.step('navigate to table', async () => {
                    await whodb.data(tableName);
                });

                const uniqueId = getUniqueTestId();
                const newDoc = {
                    username: `${uniqueId}_edit`,
                    email: `${uniqueId}@example.com`,
                    password: 'newpassword'
                };

                await test.step('create isolated document', async () => {
                    await whodb.addRow(newDoc, true);
                    await whodb.waitForRowContaining(newDoc.username);
                });

                await test.step('perform edit with network verification', async () => {
                    const rowIndex = await whodb.waitForRowContaining(newDoc.username);
                    const { rows } = await whodb.getTableData();
                    const updatedDoc = createUpdatedDocument(rows[rowIndex], {
                        [testTable.identifierField]: `${uniqueId}_updated`
                    });

                    const verifyUpdate = waitForMutation(page, 'UpdateStorageUnit');
                    await whodb.updateRow(rowIndex, 1, updatedDoc, false);
                    await verifyUpdate();

                    if (refreshDelay > 0) {
                        await page.waitForTimeout(refreshDelay);
                        await whodb.data(tableName);
                    }
                });

                await test.step('verify edit', async () => {
                    const rowIndex = await whodb.waitForRowContaining(`${uniqueId}_updated`);
                    const { rows } = await whodb.getTableData();
                    const editedDoc = parseDocument(rows[rowIndex]);
                    expect(editedDoc[testTable.identifierField]).toEqual(`${uniqueId}_updated`);
                });

                await test.step('delete isolated document', async () => {
                    const rowIndex = await whodb.waitForRowContaining(`${uniqueId}_updated`);
                    await deleteDocumentRow(whodb, page, rowIndex, refreshDelay);
                });
            });

            test('cancels edit without saving', async ({ whodb, page }) => {
                await whodb.data(tableName);

                const uniqueId = getUniqueTestId();
                const newDoc = {
                    username: `${uniqueId}_cancel`,
                    email: `${uniqueId}@example.com`,
                    password: 'newpassword'
                };
                await whodb.addRow(newDoc, true);
                let rowIndex = await whodb.waitForRowContaining(newDoc.username);
                const { rows } = await whodb.getTableData();
                const doc = parseDocument(rows[rowIndex]);
                const currentValue = doc[testTable.identifierField];

                const updatedDoc = createUpdatedDocument(rows[rowIndex], {
                    [testTable.identifierField]: 'temp_value'
                });
                await whodb.updateRow(rowIndex, 1, updatedDoc, true);

                rowIndex = await whodb.waitForRowContaining(newDoc.username);
                const { rows: verifyRows } = await whodb.getTableData();
                const verifyDoc = parseDocument(verifyRows[rowIndex]);
                expect(verifyDoc[testTable.identifierField]).toEqual(currentValue);

                await deleteDocumentRow(whodb, page, rowIndex, refreshDelay);
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

                await deleteDocumentRow(whodb, page, rowIndex, refreshDelay);
                await expectDocumentAbsent(whodb, tableName, uniqueId);
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

        test.describe('Edit Hash Field', () => {
            test('edits a hash field value and saves', async ({ whodb, page }) => {
                await test.step('navigate to key', async () => {
                    await whodb.data(keyName);
                });

                const uniqueField = `test_field_${getUniqueTestId()}`;

                await test.step('create isolated hash field', async () => {
                    const verifyAdd = waitForMutation(page, 'AddRow');
                    await whodb.addRow({ field: uniqueField, value: testValues.original });
                    await verifyAdd();
                    await whodb.waitForRowContaining(uniqueField, { timeout: TIMEOUT.SLOW });
                });

                await test.step('update field with network verification', async () => {
                    const addedIndex = await whodb.waitForRowContaining(uniqueField, { timeout: TIMEOUT.SLOW });

                    const verifyUpdate = waitForMutation(page, 'UpdateStorageUnit');
                    await whodb.updateRow(addedIndex, 1, testValues.modified, false);
                    await verifyUpdate();

                    await expect(async () => {
                        const { rows } = await whodb.getTableData();
                        const updated = rows.find(r => r[1] === uniqueField);
                        expect(updated?.[2]).toEqual(testValues.modified);
                    }).toPass({ timeout: TIMEOUT.SLOW });
                });

                await test.step('delete isolated hash field', async () => {
                    const updatedIndex = await whodb.waitForRowContaining(uniqueField, { timeout: TIMEOUT.SLOW });

                    const verifyDelete = waitForMutation(page, 'DeleteRow');
                    await whodb.deleteRow(updatedIndex);
                    await verifyDelete();

                    await expect(async () => {
                        const { rows } = await whodb.getTableData();
                        expect(rows.some(r => r[1] === uniqueField)).toBe(false);
                    }).toPass({ timeout: TIMEOUT.SLOW });
                });
            });

            test('cancels edit without saving', async ({ whodb, page }) => {
                await whodb.data(keyName);

                const uniqueField = `test_field_${getUniqueTestId()}`;
                const verifyAdd = waitForMutation(page, 'AddRow');
                await whodb.addRow({ field: uniqueField, value: testValues.original });
                await verifyAdd();

                const addedIndex = await whodb.waitForRowContaining(uniqueField, { timeout: TIMEOUT.SLOW });
                const { rows } = await whodb.getTableData();
                expect(addedIndex, `Added field ${uniqueField} should exist`).toBeGreaterThan(-1);
                const currentValue = rows[addedIndex][2];

                await whodb.updateRow(addedIndex, 1, 'temp_value', true);

                let unchangedIndex = -1;
                await expect(async () => {
                    const { rows: verifyRows } = await whodb.getTableData();
                    unchangedIndex = verifyRows.findIndex(r => r[1] === uniqueField);
                    expect(unchangedIndex, `Added field ${uniqueField} should still exist`).toBeGreaterThan(-1);
                    expect(verifyRows[unchangedIndex][2]).toEqual(currentValue);
                }).toPass({ timeout: TIMEOUT.SLOW });

                const verifyDelete = waitForMutation(page, 'DeleteRow');
                await whodb.deleteRow(unchangedIndex);
                await verifyDelete();
            });
        });

        test.describe('Add Hash Field', () => {
            test('adds a new field to a hash key', async ({ whodb, page }) => {
                await whodb.data(keyName);

                const { rows: before } = await whodb.getTableData();
                const initialCount = before.length;

                const uniqueField = `test_field_${getUniqueTestId()}`;
                const verifyAdd = waitForMutation(page, 'AddRow');
                await whodb.addRow({ field: uniqueField, value: 'test_value' });
                await verifyAdd();

                const addedIndex = await whodb.waitForRowValue(1, uniqueField);

                const { rows: after } = await whodb.getTableData();
                expect(after.length).toEqual(initialCount + 1);
                expect(addedIndex, `Added field ${uniqueField} should exist`).toBeGreaterThan(-1);

                const verifyDelete = waitForMutation(page, 'DeleteRow');
                await whodb.deleteRow(addedIndex);
                await verifyDelete();
            });
        });

        test.describe('Delete Hash Field', () => {
            test('deletes a hash field', async ({ whodb, page }) => {
                await whodb.data('user:2');

                const uniqueField = `test_field_${getUniqueTestId()}`;
                const verifyAdd = waitForMutation(page, 'AddRow');
                await whodb.addRow({ field: uniqueField, value: 'delete_value' });
                await verifyAdd();

                const addedIndex = await whodb.waitForRowContaining(uniqueField, { timeout: TIMEOUT.SLOW });
                const { rows } = await whodb.getTableData();
                expect(addedIndex, `Added field ${uniqueField} should exist`).toBeGreaterThan(-1);
                const countAfterAdd = rows.length;

                const verifyDelete = waitForMutation(page, 'DeleteRow');
                await whodb.deleteRow(addedIndex);
                await verifyDelete();

                await expect(async () => {
                    const { rows: newRows } = await whodb.getTableData();
                    expect(newRows.length).toEqual(countAfterAdd - 1);
                    expect(newRows.find(r => r[1] === uniqueField)).toBeUndefined();
                }).toPass({ timeout: TIMEOUT.SLOW });
            });
        });
    });

});
