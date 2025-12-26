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

import {forEachDatabase} from '../../support/test-runner';

describe('Table Search', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;
        const searchTerm = testTable.searchTerm;

        it('highlights matching cells when searching', () => {
            cy.data(tableName);

            cy.searchTable(searchTerm);

            // Search highlights one cell at a time, verify it contains the search term
            cy.getHighlightedCell({timeout: 5000}).first().should('contain.text', searchTerm);
        });

        it('finds multiple matches by cycling through', () => {
            cy.data(tableName);

            // First search highlights first match
            cy.searchTable(searchTerm);
            cy.getHighlightedCell({timeout: 5000}).should('exist');

            // Verify we can cycle through matches by searching again
            cy.searchTable(searchTerm);
            cy.getHighlightedCell({timeout: 5000}).should('exist');
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        it('highlights matching content in documents', () => {
            cy.data('users');

            cy.searchTable('john');

            // Search highlights one cell at a time
            cy.getHighlightedCell({timeout: 5000}).first().should('contain.text', 'john');
        });
    });

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        it('highlights matching values', () => {
            cy.data('user:1');

            cy.searchTable('john');

            // Search highlights one cell at a time
            cy.getHighlightedCell({timeout: 5000}).first().should('contain.text', 'john');
        });
    });

});
