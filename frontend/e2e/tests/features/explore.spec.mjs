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
import { verifyColumnTypes, verifyMetadata } from '../../support/categories/sql.mjs';
import { verifyMetadata as verifyDocMetadata } from '../../support/categories/document.mjs';
import { verifyKeyMetadata } from '../../support/categories/keyvalue.mjs';

test.describe('Explore Metadata', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;

        test('shows table metadata and column types', async ({ whodb, page }) => {
            await whodb.explore(tableName);
            const fields = await whodb.getExploreFields();
            const tableConfig = getTableConfig(db, tableName);
            if (tableConfig) {
                verifyColumnTypes(fields, tableConfig.columns);
                if (tableConfig.metadata) {
                    verifyMetadata(fields, tableConfig.metadata);
                }
            }
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        test('shows collection/index metadata', async ({ whodb, page }) => {
            await whodb.explore('users');
            const fields = await whodb.getExploreFields();
            const tableConfig = getTableConfig(db, 'users');
            if (tableConfig && tableConfig.metadata) {
                verifyDocMetadata(fields, tableConfig.metadata);
            }
        });
    });

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        test('shows key metadata', async ({ whodb, page }) => {
            await whodb.explore('user:1');
            const fields = await whodb.getExploreFields();
            const keyConfig = db.keyTypes['user:1'];
            if (keyConfig) {
                verifyKeyMetadata(fields, keyConfig.type);
            }
        });
    });

});
