/*
 * Copyright 2025 Clidey, Inc.
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

import {forEachDatabase, getTableConfig} from '../../support/test-runner';
import {verifyColumnTypes, verifyMetadata} from '../../support/categories/sql';
import {verifyMetadata as verifyDocMetadata} from '../../support/categories/document';
import {verifyKeyMetadata} from '../../support/categories/keyvalue';

describe('Explore Metadata', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        it('shows table metadata and column types for users table', () => {
            cy.explore('users');
            cy.getExploreFields().then(fields => {
                const tableConfig = getTableConfig(db, 'users');
                if (tableConfig) {
                    verifyColumnTypes(fields, tableConfig.columns);
                    if (tableConfig.metadata) {
                        verifyMetadata(fields, tableConfig.metadata);
                    }
                }
            });
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        it('shows collection/index metadata', () => {
            cy.explore('users');
            cy.getExploreFields().then(fields => {
                const tableConfig = getTableConfig(db, 'users');
                if (tableConfig && tableConfig.metadata) {
                    verifyDocMetadata(fields, tableConfig.metadata);
                }
            });
        });
    });

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        it('shows key metadata', () => {
            cy.explore('user:1');
            cy.getExploreFields().then(fields => {
                const keyConfig = db.keyTypes['user:1'];
                if (keyConfig) {
                    verifyKeyMetadata(fields, keyConfig.type);
                }
            });
        });
    });

});
