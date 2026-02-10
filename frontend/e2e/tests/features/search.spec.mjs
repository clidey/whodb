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

test.describe('Table Search', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;
        const searchTerm = testTable.searchTerm;

        test('highlights matching cells when searching', async ({ whodb, page }) => {
            await whodb.data(tableName);

            await whodb.searchTable(searchTerm);

            // Search highlights one cell at a time, verify it contains the search term
            const highlighted = whodb.getHighlightedCell({ timeout: 5000 });
            await highlighted.first().waitFor({ timeout: 5000 });
            await expect(highlighted.first()).toContainText(searchTerm);
        });

        test('finds multiple matches by cycling through', async ({ whodb, page }) => {
            await whodb.data(tableName);

            // First search highlights first match
            await whodb.searchTable(searchTerm);
            const highlighted = whodb.getHighlightedCell({ timeout: 5000 });
            await expect(highlighted.first()).toBeAttached({ timeout: 5000 });

            // Verify we can cycle through matches by searching again
            await whodb.searchTable(searchTerm);
            const highlighted2 = whodb.getHighlightedCell({ timeout: 5000 });
            await expect(highlighted2.first()).toBeAttached({ timeout: 5000 });
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        test('filters matching content in documents', async ({ whodb, page }) => {
            await whodb.data('users');

            await whodb.searchTable('john');

            // Search filters server-side; verify results contain the search term
            const { rows } = await whodb.getTableData();
            const hasMatch = rows.some(row => row.some(cell => cell.toLowerCase().includes('john')));
            expect(hasMatch).toBe(true);
        });
    });

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        test('filters matching values', async ({ whodb, page }) => {
            await whodb.data('user:1');

            await whodb.searchTable('john');

            // Search filters server-side; verify results contain the search term
            const { rows } = await whodb.getTableData();
            const hasMatch = rows.some(row => row.some(cell => cell.toLowerCase().includes('john')));
            expect(hasMatch).toBe(true);
        });
    });

});
