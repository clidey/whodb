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
import { getSqlQuery } from '../../support/database-config.mjs';

test.describe('Query History', () => {

    // SQL Databases with scratchpad support
    forEachDatabase('sql', (db) => {
        test('stores executed queries in history', async ({ whodb, page }) => {
            await whodb.goto('scratchpad');

            // Execute a query
            const query = getSqlQuery(db, 'selectAllUsers');
            await whodb.writeCode(0, query);
            await whodb.runCode(0);

            // Open history
            await whodb.openQueryHistory(0);

            const items = await whodb.getQueryHistoryItems();
            expect(items.length).toBeGreaterThan(0);
            expect(items[0]).toContain('SELECT');

            await whodb.closeQueryHistory();
        });

        test('clones query from history to editor', async ({ whodb, page }) => {
            await whodb.goto('scratchpad');

            // Execute first query
            const query1 = getSqlQuery(db, 'selectAllUsers');
            await whodb.writeCode(0, query1);
            await whodb.runCode(0);

            // Execute second query
            await whodb.addCell(0);
            const query2 = getSqlQuery(db, 'countUsers');
            await whodb.writeCode(1, query2);
            await whodb.runCode(1);

            // Open history and clone first query
            await whodb.openQueryHistory(1);
            await whodb.cloneQueryToEditor(0, 1);

            // Verify cloned
            await whodb.verifyQueryInEditor(1, 'COUNT');
        });

        test('copies query to clipboard', async ({ whodb, page }) => {
            await whodb.goto('scratchpad');

            const query = getSqlQuery(db, 'selectAllUsers');
            await whodb.writeCode(0, query);
            await whodb.runCode(0);

            await whodb.openQueryHistory(0);
            await whodb.copyQueryFromHistory(0);
            await whodb.closeQueryHistory();
        });

        test('executes query directly from history', async ({ whodb, page }) => {
            await whodb.goto('scratchpad');

            const query = getSqlQuery(db, 'selectAllUsers');
            await whodb.writeCode(0, query);
            await whodb.runCode(0);

            // Clear editor
            await whodb.writeCode(0, '-- cleared');

            await whodb.openQueryHistory(0);
            await whodb.executeQueryFromHistory(0);
            await whodb.closeQueryHistory();

            // Verify results appeared
            const { rows } = await whodb.getCellQueryOutput(0);
            expect(rows.length).toBeGreaterThan(0);
        });
    }, { features: ['queryHistory', 'scratchpad'] });

});
