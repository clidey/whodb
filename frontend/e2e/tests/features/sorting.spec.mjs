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

test.describe('Sorting', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;
        const idCol = testTable.idField;
        const nameCol = testTable.identifierField;
        const thirdCol = testTable.whereConditions.thirdColumn;

        test.describe('Column Header Sorting', () => {
            test('sorts ascending on first click', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Get initial data
                const { rows: initialRows } = await whodb.getTableData();

                // Click first sortable column
                await whodb.sortBy(idCol);

                // Verify ascending indicator appears
                await expect(page.locator(`[data-column-name="${idCol}"] [data-testid="sort-indicator"]`)).toBeAttached();
                await expect(page.locator(`[data-column-name="${idCol}"]`)).toHaveAttribute('data-sort-direction', 'asc');

                // Verify data is sorted
                const { rows: sortedRows } = await whodb.getTableData();
                expect(sortedRows.length).toEqual(initialRows.length);
                // Data should be sorted ascending by first column
                const values = sortedRows.map(r => r[1]);
                const sortedValues = [...values].sort((a, b) => {
                    // Handle numeric sorting
                    const aNum = parseFloat(a);
                    const bNum = parseFloat(b);
                    if (!isNaN(aNum) && !isNaN(bNum)) {
                        return aNum - bNum;
                    }
                    return a.localeCompare(b);
                });
                expect(values).toEqual(sortedValues);
            });

            test('sorts descending on second click', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Click twice to sort descending
                await whodb.sortBy(idCol);
                await whodb.sortBy(idCol);

                // Verify descending indicator appears
                await expect(page.locator(`[data-column-name="${idCol}"] [data-testid="sort-indicator"]`)).toBeAttached();
                await expect(page.locator(`[data-column-name="${idCol}"]`)).toHaveAttribute('data-sort-direction', 'desc');

                // Verify data is sorted descending
                const { rows } = await whodb.getTableData();
                const values = rows.map(r => r[1]);
                const sortedDesc = [...values].sort((a, b) => {
                    const aNum = parseFloat(a);
                    const bNum = parseFloat(b);
                    if (!isNaN(aNum) && !isNaN(bNum)) {
                        return bNum - aNum;
                    }
                    return b.localeCompare(a);
                });
                expect(values).toEqual(sortedDesc);
            });

            test('removes sort on third click', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Click once - should show ascending indicator
                await whodb.sortBy(idCol);
                await expect(page.locator(`[data-column-name="${idCol}"] [data-testid="sort-indicator"]`)).toBeAttached();
                await expect(page.locator(`[data-column-name="${idCol}"]`)).toHaveAttribute('data-sort-direction', 'asc');

                // Click twice - should show descending indicator
                await whodb.sortBy(idCol);
                await expect(page.locator(`[data-column-name="${idCol}"] [data-testid="sort-indicator"]`)).toBeAttached();
                await expect(page.locator(`[data-column-name="${idCol}"]`)).toHaveAttribute('data-sort-direction', 'desc');

                // Click three times - should remove sort indicator
                await whodb.sortBy(idCol);
                await expect(page.locator(`[data-column-name="${idCol}"] [data-testid="sort-indicator"]`)).not.toBeAttached();
                await expect(page.locator(`[data-column-name="${idCol}"]`)).not.toHaveAttribute('data-sort-direction');
            });

            test('can sort by different columns', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Sort by identifier column
                await whodb.sortBy(nameCol);

                // Verify indicator on identifier column
                await expect(page.locator(`[data-column-name="${nameCol}"] [data-testid="sort-indicator"]`)).toBeAttached();
                await expect(page.locator(`[data-column-name="${nameCol}"]`)).toHaveAttribute('data-sort-direction', 'asc');

                // Sort by third column (should clear first, add second)
                await whodb.sortBy(thirdCol);

                // Both columns may have sort indicators (multi-column sort)
                await expect(page.locator(`[data-column-name="${thirdCol}"] [data-testid="sort-indicator"]`)).toBeAttached();
                await expect(page.locator(`[data-column-name="${thirdCol}"]`)).toHaveAttribute('data-sort-direction', 'asc');
            });
        });

        test.describe('Sort with Other Features', () => {
            test('maintains sort after search', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Sort ascending
                await whodb.sortBy(idCol);

                // Get sorted data
                const { rows: sortedRows } = await whodb.getTableData();
                const firstValue = sortedRows[0][1];

                // Perform search that still includes first row
                await whodb.searchTable(firstValue.substring(0, 2));

                // Verify sort indicator still present
                await expect(page.locator(`[data-column-name="${idCol}"] [data-testid="sort-indicator"]`)).toBeAttached();
                await expect(page.locator(`[data-column-name="${idCol}"]`)).toHaveAttribute('data-sort-direction', 'asc');
            });
        });
    });

    // Document Databases (MongoDB, Elasticsearch)
    forEachDatabase('document', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;

        if (!tableName) {
            test.skip('testTable config missing in fixture', async ({ whodb, page }) => {
            });
            return;
        }

        test.describe('Document Sorting', () => {
            test('can sort document list', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Click to sort by document column
                await whodb.sortBy('document');

                // Verify sort indicator appears
                await expect(page.locator('[data-column-name="document"] [data-testid="sort-indicator"]')).toBeAttached();
                await expect(page.locator('[data-column-name="document"]')).toHaveAttribute('data-sort-direction', 'asc');
            });
        });
    });

    // Key-Value Databases (Redis)
    forEachDatabase('keyvalue', (db) => {
        const testTable = db.testTable;
        const keyName = testTable.name;

        if (!keyName) {
            test.skip('testTable config missing in fixture', async ({ whodb, page }) => {
            });
            return;
        }

        test.describe('Key-Value Sorting', () => {
            test('can sort hash fields', async ({ whodb, page }) => {
                await whodb.data(keyName);

                // Click to sort by field column
                await whodb.sortBy('field');

                // Verify sort indicator appears
                await expect(page.locator('[data-column-name="field"] [data-testid="sort-indicator"]')).toBeAttached();
                await expect(page.locator('[data-column-name="field"]')).toHaveAttribute('data-sort-direction', 'asc');
            });
        });
    });

});
