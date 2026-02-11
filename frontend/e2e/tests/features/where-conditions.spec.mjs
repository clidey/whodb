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

import { test, expect, forEachDatabase, conditionalTest } from '../../support/test-fixture.mjs';
import { hasFeature } from '../../support/database-config.mjs';
import { getDocumentId, parseDocument } from '../../support/categories/document.mjs';

/**
 * Get the operator for a given database
 * @param {Object} db - Database configuration
 * @param {string} operatorKey - Operator key (e.g., 'equals', 'notEquals')
 * @returns {string} Operator string for this database
 */
function getOperator(db, operatorKey) {
    return db.whereOperators[operatorKey];
}

test.describe('Where Conditions', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;
        const idField = testTable.idField;
        const nameField = testTable.identifierField;
        const firstName = testTable.firstName;
        const colIndex = testTable.identifierColIndex;
        const eq = getOperator(db, 'equals');

        const whereConfig = testTable.whereConditions;
        const testId1 = whereConfig.testId1;
        const testId2 = whereConfig.testId2;
        const testId3 = whereConfig.testId3;

        test('applies where condition and filters data', async ({ whodb, page }) => {
            await whodb.data(tableName);

            await whodb.whereTable([[idField, eq, testId3]]);
            await whodb.submitTable();

            const { rows } = await whodb.getTableData();
            expect(rows.length).toEqual(1);
            expect(rows[0][1]).toEqual(testId3); // id column

            // Clear conditions
            await whodb.clearWhereConditions();
            await whodb.submitTable();

            const { rows: clearedRows } = await whodb.getTableData();
            expect(clearedRows.length).toBeGreaterThan(1);
        });

        // Skip multi-condition tests for databases where data may be affected by async mutations
        const multiConditionSupported = hasFeature(db, 'multiConditionFilter') !== false;

        conditionalTest(multiConditionSupported, 'applies multiple conditions', async ({ whodb, page }) => {
            await whodb.setWhereConditionMode('sheet');
            await whodb.data(tableName);

            await whodb.whereTable([
                [idField, eq, testId1],
                [nameField, eq, firstName],
            ]);
            await whodb.submitTable();

            const condCount = await whodb.getConditionCount();
            expect(condCount).toEqual(2);

            const { rows } = await whodb.getTableData();
            expect(rows.length).toEqual(1);
            expect(rows[0][colIndex + 1]).toEqual(firstName);

            await whodb.clearWhereConditions();
            await whodb.submitTable();
        });

        test('edits existing condition', async ({ whodb, page }) => {
            await whodb.data(tableName);

            await whodb.whereTable([[idField, eq, testId1]]);
            await whodb.submitTable();

            const mode = await whodb.getWhereConditionMode();
            if (mode === 'popover') {
                await whodb.clickConditionToEdit(0);
                await whodb.updateConditionValue(testId2);
                await page.locator('[data-testid="update-condition-button"]').click();
                await whodb.submitTable();

                await whodb.verifyCondition(0, `${idField} ${eq} ${testId2}`);

                const { rows } = await whodb.getTableData();
                expect(rows[0][1]).toEqual(testId2);
            }

            await whodb.clearWhereConditions();
            await whodb.submitTable();
        });

        test('removes individual condition', async ({ whodb, page }) => {
            await whodb.data(tableName);

            await whodb.whereTable([
                [idField, eq, testId1],
                [nameField, eq, firstName],
            ]);
            await whodb.submitTable();

            const condCount = await whodb.getConditionCount();
            expect(condCount).toEqual(2);

            await whodb.removeCondition(1);
            await whodb.submitTable();

            const condCount2 = await whodb.getConditionCount();
            expect(condCount2).toEqual(1);

            await whodb.clearWhereConditions();
            await whodb.submitTable();
        });

        test('shows more conditions button when exceeding visible limit', async ({ whodb, page }) => {
            await whodb.data(tableName);

            // Clear any existing conditions first
            await whodb.clearWhereConditions();
            await page.waitForTimeout(500);

            const neq = getOperator(db, 'notEquals');

            // Get config-driven values for third condition
            const thirdCol = whereConfig.thirdColumn;
            const thirdVal = whereConfig.thirdValue;
            const expectedName = whereConfig.expectedValue;

            // Add 3 conditions - should show "+1 more" button
            await whodb.whereTable([
                [idField, eq, testId3],
                [nameField, eq, expectedName],
                [thirdCol, neq, thirdVal],
            ]);

            const mode = await whodb.getWhereConditionMode();
            if (mode === 'popover') {
                // Should show first 2 conditions as badges
                const condCount = await whodb.getConditionCount();
                expect(condCount).toEqual(3);
                await whodb.verifyCondition(0, `${idField} ${eq} ${testId3}`);
                await whodb.verifyCondition(1, `${nameField} ${eq} ${expectedName}`);

                // Check for more conditions button
                await whodb.checkMoreConditionsButton('+1 more');

                // Click to open sheet with all conditions
                await whodb.clickMoreConditions();

                // Remove conditions in sheet
                await whodb.removeConditionsInSheet(true);
                await whodb.saveSheetChanges();

                // After closing sheet, should have only 1 condition
                const condCount2 = await whodb.getConditionCount();
                expect(condCount2).toEqual(1);
                await whodb.verifyCondition(0, `${idField} ${eq} ${testId3}`);
            } else {
                // In sheet mode, just verify count
                const condCount = await whodb.getConditionCount();
                expect(condCount).toEqual(3);
            }

            await whodb.submitTable();
            const { rows } = await whodb.getTableData();
            expect(rows[0][colIndex + 1]).toEqual(expectedName);

            await whodb.clearWhereConditions();
            await whodb.submitTable();
        });

        test('cancels condition edit', async ({ whodb, page }) => {
            await whodb.data(tableName);

            await whodb.whereTable([[idField, eq, '1']]);
            await whodb.submitTable();

            const mode = await whodb.getWhereConditionMode();
            if (mode === 'popover') {
                await whodb.clickConditionToEdit(0);
                await whodb.updateConditionValue('2');
                await page.locator('[data-testid="cancel-button"]').click();

                // Condition should remain unchanged
                await whodb.verifyCondition(0, `${idField} ${eq} 1`);
            }

            await whodb.clearWhereConditions();
            await whodb.submitTable();
        });
    }, { features: ['whereConditions'] });

    // Document Databases
    forEachDatabase('document', (db) => {
        const eq = getOperator(db, 'equals');

        test('applies where condition and filters documents', async ({ whodb, page }) => {
            await whodb.data('users');
            await whodb.sortBy(0);

            const refreshDelay = db.indexRefreshDelay || 0;

            const { rows } = await whodb.getTableData();
            // Use _id field for filtering (works for both Elasticsearch and MongoDB)
            const firstDocId = getDocumentId(rows[0]);

            await whodb.whereTable([['_id', eq, firstDocId]]);
            await whodb.submitTable();

            // Wait for query to process
            if (refreshDelay > 0) {
                await page.waitForTimeout(refreshDelay);
            }

            // Use Playwright retry to wait for filtered results
            await expect(page.locator('table').filter({ visible: true }).locator('tbody tr')).toHaveCount(1, { timeout: 10000 });

            const { rows: filteredRows } = await whodb.getTableData();
            expect(filteredRows.length).toEqual(1);
            expect(getDocumentId(filteredRows[0])).toEqual(firstDocId);

            await whodb.clearWhereConditions();
            await whodb.submitTable();

            // Wait for data to reload after clearing conditions
            if (refreshDelay > 0) {
                await page.waitForTimeout(refreshDelay);
            }

            const { rows: clearedRows } = await whodb.getTableData();
            expect(clearedRows.length).toBeGreaterThan(1);
        });

        test('applies multiple conditions on documents', async ({ whodb, page }) => {
            await whodb.setWhereConditionMode('sheet');
            await whodb.data('users');
            await whodb.sortBy(0);

            const refreshDelay = db.indexRefreshDelay || 0;

            const { rows } = await whodb.getTableData();
            // Use _id for exact matching (works for both Elasticsearch and MongoDB)
            const firstDocId = getDocumentId(rows[0]);
            const firstDoc = parseDocument(rows[0]);

            await whodb.whereTable([
                ['_id', eq, firstDocId],
                ['username', eq, firstDoc.username],
            ]);
            await whodb.submitTable();

            // Wait for query to process
            if (refreshDelay > 0) {
                await page.waitForTimeout(refreshDelay);
            }

            const condCount = await whodb.getConditionCount();
            expect(condCount).toEqual(2);

            // Wait for filtered results
            await expect(page.locator('table').filter({ visible: true }).locator('tbody tr')).toHaveCount(1, { timeout: 10000 });

            const { rows: filtered } = await whodb.getTableData();
            const doc = parseDocument(filtered[0]);
            expect(doc.username).toEqual(firstDoc.username);

            await whodb.clearWhereConditions();
            await whodb.submitTable();

            // Wait for data to reload after clearing
            if (refreshDelay > 0) {
                await page.waitForTimeout(refreshDelay);
            }
        });
    }, { features: ['whereConditions'] });

});
