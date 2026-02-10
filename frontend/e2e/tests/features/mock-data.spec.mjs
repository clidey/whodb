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
import { hasFeature } from '../../support/database-config.mjs';

test.describe('Mock Data Generation', () => {

    // SQL Databases with mock data support
    forEachDatabase('sql', (db) => {
        const supportedTable = db.mockData.supportedTable;
        const tableWithFKs = db.mockData.tableWithFKs;

        test('shows mock data generation UI', async ({ whodb, page }) => {
            await whodb.data(supportedTable);

            await whodb.selectMockData();

            // Verify dialog and note appeared
            await expect(page.locator('[data-testid="mock-data-sheet"]')).toBeVisible();
            await expect(page.locator('text=Note')).toBeVisible();
            await expect(page.locator('[data-testid="mock-data-generate-button"]')).toBeVisible();

            // Close dialog
            await page.keyboard.press('Escape');
        });

        test('enforces maximum row count limit', async ({ whodb, page }) => {
            await whodb.data(supportedTable);

            await whodb.selectMockData();

            // Try to exceed max count (should clamp to 200)
            const rowsInput = page.locator('[data-testid="mock-data-rows-input"]');
            await rowsInput.fill('300');
            const val = await rowsInput.inputValue();
            expect(parseInt(val, 10)).toEqual(200);

            await page.keyboard.press('Escape');
        });

        test('shows overwrite confirmation dialog', async ({ whodb, page }) => {
            await whodb.data(supportedTable);

            await whodb.selectMockData();

            // Set row count
            await whodb.setMockDataRows(10);

            // Switch to Overwrite mode
            await whodb.setMockDataHandling('overwrite');

            // Click Generate
            await whodb.generateMockData();

            // Should show confirmation
            await expect(page.locator('[data-testid="mock-data-overwrite-button"]')).toBeVisible();

            // Cancel instead of confirming
            await page.keyboard.press('Escape');
        });

        test('generates mock data and adds rows to table', async ({ whodb, page }) => {
            await whodb.data(supportedTable);

            // Get initial total count from UI
            const totalText = await page.locator('[data-testid="total-count-top"]').textContent();
            const initialCount = parseInt(totalText.replace(/[^0-9]/g, ''), 10) || 0;

            await whodb.selectMockData();

            // Generate 5 rows (append mode - default)
            await whodb.setMockDataRows(5);
            await whodb.generateMockData();

            // Wait for success toast
            await expect(page.locator('text=Successfully Generated')).toBeVisible({ timeout: 30000 });

            // Sheet should close after success
            await expect(page.locator('[data-testid="mock-data-sheet"]')).not.toBeAttached();

            // Verify Total Count increased by exactly 5
            await expect(page.locator('[data-testid="total-count-top"]')).toHaveText(
                new RegExp(String(initialCount + 5)),
                { timeout: 5000 }
            );
        });

        // Skip FK dependency test for databases without FK support (e.g., ClickHouse)
        if (tableWithFKs) {
            test('shows dependency preview for tables with foreign keys', async ({ whodb, page }) => {
                await whodb.data(tableWithFKs);

                await whodb.selectMockData();

                // Set row count to trigger dependency analysis
                await whodb.setMockDataRows(10);

                // Should show dependency preview with parent tables
                await expect(page.locator('text=Tables to populate')).toBeVisible();
                await expect(page.locator('text=Total:')).toBeVisible();

                // Generate button should be enabled (FK tables now supported)
                await expect(page.locator('[data-testid="mock-data-generate-button"]')).not.toBeDisabled();

                await page.keyboard.press('Escape');
            });

            test('generates mock data for FK table and populates parent tables', async ({ whodb, page }) => {
                await whodb.data(tableWithFKs);

                // Get initial total count
                const totalText = await page.locator('[data-testid="total-count-top"]').textContent();
                const initialCount = parseInt(totalText.replace(/[^0-9]/g, ''), 10) || 0;

                await whodb.selectMockData();

                // Generate rows for FK table
                await whodb.setMockDataRows(5);
                await whodb.generateMockData();

                // Wait for success toast - FK generation may take longer
                await expect(page.locator('text=Successfully Generated')).toBeVisible({ timeout: 60000 });

                // Sheet should close after success
                await expect(page.locator('[data-testid="mock-data-sheet"]')).not.toBeAttached();

                // Verify Total Count increased by at least 5
                await expect(async () => {
                    const newText = await page.locator('[data-testid="total-count-top"]').textContent();
                    const newCount = parseInt(newText.replace(/[^0-9]/g, ''), 10);
                    expect(newCount).toBeGreaterThanOrEqual(initialCount + 5);
                }).toPass({ timeout: 5000 });
            });

            // Edge case: Low row count with FK tables (bug fix verification)
            // Previously failed with FK constraint error when generating < 5 rows
            test('generates low row count (1-3 rows) for FK table successfully', async ({ whodb, page }) => {
                await whodb.data(tableWithFKs);

                // Get initial total count
                const totalText = await page.locator('[data-testid="total-count-top"]').textContent();
                const initialCount = parseInt(totalText.replace(/[^0-9]/g, ''), 10) || 0;

                await whodb.selectMockData();

                // Generate only 2 rows - previously this would fail with FK constraint error
                await whodb.setMockDataRows(2);
                await whodb.generateMockData();

                // Should succeed without FK constraint errors
                await expect(page.locator('text=Successfully Generated')).toBeVisible({ timeout: 60000 });

                // Sheet should close
                await expect(page.locator('[data-testid="mock-data-sheet"]')).not.toBeAttached();

                // Verify Total Count increased by at least 2
                await expect(async () => {
                    const newText = await page.locator('[data-testid="total-count-top"]').textContent();
                    const newCount = parseInt(newText.replace(/[^0-9]/g, ''), 10);
                    expect(newCount).toBeGreaterThanOrEqual(initialCount + 2);
                }).toPass({ timeout: 5000 });
            });

            // Comprehensive FK verification: checks row counts and FK->PK relationships
            // Note: Uses append mode because overwrite can fail when target table has child FK references
            // (e.g., orders is referenced by order_items and payments)
            test('verifies correct row counts and FK references after generation', async ({ whodb, page }) => {
                const fkRelationship = db.mockData.fkRelationships?.[tableWithFKs];
                if (!fkRelationship) {
                    test.skip();
                    return;
                }

                const { parentTable, fkColumn, parentPkColumn } = fkRelationship;
                const rowsToGenerate = 5;

                // Step 1: Get initial parent table Total Count
                await whodb.data(parentTable);
                const parentTotalText = await page.locator('[data-testid="total-count-top"]').textContent();
                const initialParentCount = parseInt(parentTotalText.replace(/[^0-9]/g, ''), 10) || 0;

                // Step 2: Get initial FK table Total Count
                await whodb.data(tableWithFKs);
                const fkTotalText = await page.locator('[data-testid="total-count-top"]').textContent();
                const initialFkTableCount = parseInt(fkTotalText.replace(/[^0-9]/g, ''), 10) || 0;

                // Step 3: Open mock data dialog and set row count
                await whodb.selectMockData();
                await whodb.setMockDataRows(rowsToGenerate);

                // Step 4: Parse the dependency preview to get expected parent row count
                await expect(page.locator('text=Tables to populate')).toBeVisible();
                let expectedParentRows = 0;
                const sheetEl = page.locator('[data-testid="mock-data-sheet"]');
                const parentRow = sheetEl.locator(`text=${parentTable}`).locator('..');
                const rowText = await parentRow.textContent();
                const match = rowText.match(/(\d+)\s*rows?/);
                if (match) {
                    expectedParentRows = parseInt(match[1], 10);
                }

                // Step 5: Generate mock data (append mode - default)
                await whodb.generateMockData();
                await expect(page.locator('text=Successfully Generated')).toBeVisible({ timeout: 60000 });

                // Step 6: Verify parent table Total Count increased
                await whodb.data(parentTable);
                await expect(async () => {
                    const newText = await page.locator('[data-testid="total-count-top"]').textContent();
                    const newCount = parseInt(newText.replace(/[^0-9]/g, ''), 10);
                    expect(newCount).toBeGreaterThanOrEqual(initialParentCount + expectedParentRows);
                }).toPass({ timeout: 5000 });

                // Step 7: Collect parent PKs
                const parentPKs = new Set();
                const headers = await page.locator('table thead th').allTextContents();
                let pkColIndex = -1;
                headers.forEach((text, i) => {
                    if (text.trim().toLowerCase() === parentPkColumn.toLowerCase()) {
                        pkColIndex = i;
                    }
                });

                if (pkColIndex !== -1) {
                    const rows = await page.locator('table tbody tr').all();
                    for (const row of rows) {
                        const pkValue = await row.locator('td').nth(pkColIndex).textContent();
                        if (pkValue && pkValue.trim()) {
                            parentPKs.add(pkValue.trim());
                        }
                    }
                }

                // Step 8: Navigate to FK table and verify Total Count increased
                await whodb.data(tableWithFKs);
                await expect(async () => {
                    const newText = await page.locator('[data-testid="total-count-top"]').textContent();
                    const newCount = parseInt(newText.replace(/[^0-9]/g, ''), 10);
                    expect(newCount).toBeGreaterThanOrEqual(initialFkTableCount + rowsToGenerate);
                }).toPass({ timeout: 5000 });

                // Step 9: Verify FK values exist in parent PKs
                if (parentPKs.size > 0) {
                    const fkHeaders = await page.locator('table thead th').allTextContents();
                    let fkColIndex = -1;
                    fkHeaders.forEach((text, i) => {
                        if (text.trim().toLowerCase() === fkColumn.toLowerCase()) {
                            fkColIndex = i;
                        }
                    });

                    if (fkColIndex !== -1) {
                        const fkRows = await page.locator('table tbody tr').all();
                        for (const row of fkRows) {
                            const fkValue = await row.locator('td').nth(fkColIndex).textContent();
                            if (fkValue && fkValue.trim() !== 'NULL' && fkValue.trim() !== '') {
                                expect(parentPKs.has(fkValue.trim()),
                                    `FK value ${fkValue.trim()} should exist in parent PKs`).toBe(true);
                            }
                        }
                    }
                }
            });
        }

        // Test overwrite mode LAST - it clears child tables via FK-safe deletion
        // which would affect other tests if run earlier
        // Note: For ClickHouse, this tests simple overwrite (no FK support in ClickHouse)
        const testName = tableWithFKs
            ? 'executes overwrite mode and clears table with FK references'
            : 'executes overwrite mode for single table (no FK support)';

        test(testName, async ({ whodb, page }) => {
            await whodb.data(supportedTable);

            await whodb.selectMockData();

            // Set row count
            await whodb.setMockDataRows(5);

            // Switch to Overwrite mode
            await whodb.setMockDataHandling('overwrite');

            // Click Generate - shows confirmation
            await whodb.generateMockData();

            // Confirm overwrite
            await expect(page.locator('[data-testid="mock-data-overwrite-button"]')).toBeVisible();
            await page.locator('[data-testid="mock-data-overwrite-button"]').click();

            // Wait for success toast - FK-safe clearing may take longer
            await expect(page.locator('text=Successfully Generated')).toBeVisible({ timeout: 60000 });

            // Sheet should close after success
            await expect(page.locator('[data-testid="mock-data-sheet"]')).not.toBeAttached();

            // Verify Total Count is exactly 5 (overwrite replaces all)
            await expect(async () => {
                const text = await page.locator('[data-testid="total-count-top"]').textContent();
                const count = parseInt(text.replace(/[^0-9]/g, ''), 10);
                expect(count).toEqual(5);
            }).toPass({ timeout: 5000 });
        });

        // Test mock data generation for data_types table (covers all column types)
        // This validates type-specific generation: smallint limits, decimal precision, etc.
        test('generates mock data for data_types table with various column types', async ({ whodb, page }) => {
            const dataTypesTable = db.dataTypesTable;
            if (!dataTypesTable) {
                test.skip();
                return;
            }

            await whodb.data(dataTypesTable);

            await whodb.selectMockData();

            await whodb.setMockDataRows(100);
            await whodb.setMockDataHandling('overwrite');

            await whodb.generateMockData();

            // Confirm overwrite
            await expect(page.locator('[data-testid="mock-data-overwrite-button"]')).toBeVisible();
            await page.locator('[data-testid="mock-data-overwrite-button"]').click();

            // Wait for success - type generation may require more time
            await expect(page.locator('text=Successfully Generated')).toBeVisible({ timeout: 60000 });

            // Sheet should close
            await expect(page.locator('[data-testid="mock-data-sheet"]')).not.toBeAttached();

            // Verify Total Count is exactly 100
            await expect(async () => {
                const text = await page.locator('[data-testid="total-count-top"]').textContent();
                const count = parseInt(text.replace(/[^0-9]/g, ''), 10);
                expect(count).toEqual(100);
            }).toPass({ timeout: 5000 });
        });
    }, { features: ['mockData'] });

    // Document Databases - mock data not supported (inverse: runs when feature is NOT present)
    // Note: ElasticSearch is excluded from mock data tests entirely due to connection instability
    forEachDatabase('document', (db) => {
        if (hasFeature(db, 'mockData')) {
            return; // Only run if mock data is NOT supported
        }
        if (db.type === 'ElasticSearch') {
            return; // Skip ElasticSearch - mock data not supported
        }

        test('shows not allowed message for document databases', async ({ whodb, page }) => {
            await whodb.data('orders');

            await whodb.selectMockData();

            await whodb.generateMockData();

            await expect(page.locator('text=Mock Data Generation Is Not Allowed for This Table')).toBeAttached();
        });
    });

});
