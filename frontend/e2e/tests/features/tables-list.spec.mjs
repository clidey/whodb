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
import { filterSessionKeys } from '../../support/categories/keyvalue.mjs';

test.describe('Storage Unit Listing', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        test('lists expected tables', async ({ whodb, page }) => {
            const tables = await whodb.getTables();
            expect(Array.isArray(tables)).toBeTruthy();
            // Filter out test artifacts (tables like test_table_12345 from mock data tests)
            const filteredTables = tables.filter(t => !t.match(/^test_table_\d+/));
            expect(filteredTables).toEqual(db.expectedTables);
        });
    });

    // Document Databases (MongoDB, Elasticsearch)
    forEachDatabase('document', (db) => {
        test('lists expected collections/indices', async ({ whodb, page }) => {
            const items = await whodb.getTables();
            expect(Array.isArray(items)).toBeTruthy();
            // Create a copy to avoid mutating the config
            let expected = [...(db.expectedIndices || db.expectedTables)];
            // Some databases may include system.views
            if (db.includesSystemViews) {
                expected.push('system.views');
                expected.sort();
            }
            expect(items).toEqual(expected);
        });
    });

    // Key-Value Databases (Redis)
    forEachDatabase('keyvalue', (db) => {
        test('lists expected keys', async ({ whodb, page }) => {
            const keys = await whodb.getTables();
            expect(Array.isArray(keys)).toBeTruthy();
            const filteredKeys = filterSessionKeys(keys);
            expect(filteredKeys.length).toBeGreaterThanOrEqual(db.expectedKeys.length);
            db.expectedKeys.forEach(key => {
                expect(filteredKeys).toContain(key);
            });
        });
    });

});
