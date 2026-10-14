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
import { getUniqueTestId, TIMEOUT, waitForMutation } from '../../support/helpers/test-utils.mjs';

/**
 * Inline editing and bulk delete
 *
 * Covers the grid interactions added with the CE redesign: editing a cell in
 * place with the save bar, deleting selected rows from the selection bar, and
 * undoing a delete from the toast. Each test adds its own row and removes it.
 */
test.describe('Inline Editing', () => {
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;
        const identifierColumnIndex = testTable.identifierColIndex + 1;

        const addUniqueRow = async (whodb, page) => {
            const template = getTableConfig(db, tableName)?.testData?.newRow;
            test.skip(!template, 'No newRow test data configured');

            const uniqueId = getUniqueTestId();
            const row = { ...template, [testTable.identifierField]: `inline_${uniqueId}` };
            if (row.email) row.email = `inline_${uniqueId}@example.com`;

            const verifyAdd = waitForMutation(page, 'AddRow');
            await whodb.addRow(row);
            await verifyAdd();

            const rowIndex = await whodb.waitForRowValue(identifierColumnIndex, row[testTable.identifierField]);
            return { row, rowIndex };
        };

        test('edits a cell in place and saves from the save bar', async ({ whodb, page }) => {
            await whodb.data(tableName);
            const { row, rowIndex } = await addUniqueRow(whodb, page);
            const updatedValue = `${row[testTable.identifierField]}_edited`;

            await whodb.dataCell(rowIndex, identifierColumnIndex).dblclick();
            const editor = page.locator('[data-testid="cell-editor"]');
            await expect(editor).toBeVisible();
            await editor.fill(updatedValue);
            await editor.press('Enter');

            const saveBar = page.locator('[data-testid="cell-save-bar"]');
            await expect(saveBar).toBeVisible();
            await expect(saveBar).toContainText('1');

            const verifyUpdate = waitForMutation(page, 'UpdateStorageUnit');
            await page.locator('[data-testid="save-cell-changes"]').click();
            await verifyUpdate();
            await expect(saveBar).not.toBeAttached({ timeout: TIMEOUT.ACTION });

            const updatedIndex = await whodb.waitForRowValue(identifierColumnIndex, updatedValue);

            const verifyDelete = waitForMutation(page, 'DeleteRow');
            await whodb.deleteRow(updatedIndex);
            await verifyDelete();
        });

        test('discards pending cell edits without saving', async ({ whodb, page }) => {
            await whodb.data(tableName);
            const { row, rowIndex } = await addUniqueRow(whodb, page);

            await whodb.dataCell(rowIndex, identifierColumnIndex).dblclick();
            const editor = page.locator('[data-testid="cell-editor"]');
            await editor.fill('discarded_value');
            await editor.press('Enter');

            const saveBar = page.locator('[data-testid="cell-save-bar"]');
            await expect(saveBar).toBeVisible();
            await saveBar.getByRole('button', { name: 'Discard' }).click();
            await expect(saveBar).not.toBeAttached();

            const { rows } = await whodb.getTableData();
            expect(rows[rowIndex][identifierColumnIndex]).toEqual(row[testTable.identifierField]);

            const verifyDelete = waitForMutation(page, 'DeleteRow');
            await whodb.deleteRow(rowIndex);
            await verifyDelete();
        });

        test('deletes selected rows from the selection bar and undoes it', async ({ whodb, page }) => {
            await whodb.data(tableName);
            const { row, rowIndex } = await addUniqueRow(whodb, page);
            const identifier = row[testTable.identifierField];

            await whodb.openContextMenu(rowIndex);
            await page.locator('[role="menu"]').getByText('Select Row').click({ force: true });
            const selectionBar = page.locator('[data-testid="selected-rows-bar"]');
            await expect(selectionBar).toContainText(/1 rows? selected/);

            const verifyDelete = waitForMutation(page, 'DeleteRow');
            await selectionBar.locator('[data-testid="delete-selected-rows"]').click();
            await page.locator('[data-testid="confirm-delete-row-button"]').click();
            await verifyDelete();

            const toast = page.locator('.ce-delete-toast');
            await expect(toast).toBeVisible({ timeout: TIMEOUT.ACTION });
            await expect(toast).toContainText(/1 rows? deleted/);

            const verifyRestore = waitForMutation(page, 'AddRow');
            await toast.getByRole('button', { name: 'Undo' }).click();
            await verifyRestore();
            await expect(page.getByText('Deleted rows restored')).toBeVisible({ timeout: TIMEOUT.ACTION });

            const restoredIndex = await whodb.waitForRowValue(identifierColumnIndex, identifier);

            const verifyCleanup = waitForMutation(page, 'DeleteRow');
            await whodb.deleteRow(restoredIndex);
            await verifyCleanup();
        });
    }, { databases: ['postgres', 'sqlite'] });
});
