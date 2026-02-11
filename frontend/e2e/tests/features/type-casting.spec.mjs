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

// Helper to get value from object with case-insensitive key
function getValue(obj, key) {
    const lowerKey = key.toLowerCase();
    const matchingKey = Object.keys(obj).find(k => k.toLowerCase() === lowerKey);
    return matchingKey ? obj[matchingKey] : undefined;
}

test.describe('Type Casting', () => {

    // SQL Databases only - tests numeric type handling
    forEachDatabase('sql', (db) => {
        // Skip type casting tests for databases with async mutations (e.g., ClickHouse)
        if (hasFeature(db, 'typeCasting') === false) {
            test.skip('type casting tests skipped - async mutations not supported', async ({ whodb, page }) => {
            });
            return;
        }

        const testTable = db.testTable;
        const typeCastingTable = testTable.typeCastingTable;
        const tableConfig = getTableConfig(db, typeCastingTable);
        if (!tableConfig || !tableConfig.testData || !tableConfig.testData.newRow) {
            return;
        }

        const mutationDelay = db.mutationDelay || 0;
        const columns = tableConfig.columns;
        const columnNames = Object.keys(columns);
        const bigintCol = columnNames.find(c => c.toLowerCase() === 'bigint_col') || 'bigint_col';
        const integerCol = columnNames.find(c => c.toLowerCase() === 'integer_col') || 'integer_col';
        const smallintCol = columnNames.find(c => c.toLowerCase() === 'smallint_col') || 'smallint_col';
        const numericCol = columnNames.find(c => c.toLowerCase() === 'numeric_col') || 'numeric_col';
        const descriptionCol = columnNames.find(c => c.toLowerCase() === 'description') || 'description';

        test.describe('Add Row Type Casting', () => {
            test('correctly casts string inputs to numeric types', async ({ whodb, page }) => {
                await whodb.data(typeCastingTable);

                const newRow = tableConfig.testData.newRow;

                // Add a row and verify it was added by checking for its description
                const descValue = getValue(newRow, 'description');
                await whodb.addRow(newRow);

                // Wait for row to appear using retry-able assertion
                const rowIndex = await whodb.waitForRowContaining(descValue, { caseSensitive: true });

                await whodb.sortBy(0);

                // Verify the row was added with correct types
                const { rows } = await whodb.getTableData();
                const addedRow = rows.find(r => r.includes(descValue));
                expect(addedRow, 'Added row should exist').toBeDefined();
                expect(addedRow[1]).toMatch(/^\d+$/); // id should be a number
                expect(addedRow[2]).toEqual(getValue(newRow, 'bigint_col'));
                expect(addedRow[3]).toEqual(getValue(newRow, 'integer_col'));
                expect(addedRow[4]).toEqual(getValue(newRow, 'smallint_col'));
                expect(addedRow[5]).toEqual(getValue(newRow, 'numeric_col'));
                expect(addedRow[6]).toEqual(descValue);

                // Clean up - find row index again after sort
                const deleteIndex = rows.findIndex(r => r.includes(descValue));
                if (deleteIndex >= 0) {
                    await whodb.deleteRow(deleteIndex);
                }
            });

            test('handles large bigint values', async ({ whodb, page }) => {
                await whodb.data(typeCastingTable);

                // Build row with correct column names for this database
                const largeNumberRow = {
                    [bigintCol]: '5000000000',
                    [integerCol]: '42',
                    [smallintCol]: '256',
                    [numericCol]: '9876.54',
                    [descriptionCol]: 'Large bigint test'
                };

                await whodb.addRow(largeNumberRow);

                // Wait for row to appear using retry-able assertion
                const rowIndex = await whodb.waitForRowContaining('Large bigint test', { caseSensitive: true });

                const { rows } = await whodb.getTableData();
                const addedRow = rows.find(r => r.includes('Large bigint test'));
                expect(addedRow).toBeDefined();
                expect(addedRow).toContain('5000000000');

                // Clean up
                await whodb.deleteRow(rowIndex);
            });
        });

        test.describe('Edit Row Type Casting', () => {
            test('edits numeric values with type casting', async ({ whodb, page }) => {
                await whodb.data(typeCastingTable);
                await whodb.sortBy(0);

                // Edit bigint_col on second row
                await whodb.updateRow(1, 1, '7500000000', false);

                // Wait for async mutations (e.g., ClickHouse)
                if (mutationDelay > 0) {
                    await page.waitForTimeout(mutationDelay);
                    await whodb.data(typeCastingTable);
                    await whodb.sortBy(0);
                }

                let { rows } = await whodb.getTableData();
                expect(rows[1][2]).toEqual('7500000000');

                // Restore original value
                await whodb.updateRow(1, 1, '1000000', false);

                // Wait for async mutations
                if (mutationDelay > 0) {
                    await page.waitForTimeout(mutationDelay);
                    await whodb.data(typeCastingTable);
                    await whodb.sortBy(0);
                }

                ({ rows } = await whodb.getTableData());
                expect(rows[1][2]).toEqual('1000000');
            });
        });
    });

});
