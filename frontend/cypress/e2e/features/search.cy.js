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
        it('highlights matching cells when searching', () => {
            cy.data('users');

            cy.searchTable('john');

            cy.getHighlightedCell().first().should('contain.text', 'john');
        });

        it('finds multiple matches', () => {
            cy.data('users');

            cy.searchTable('example.com');

            cy.getHighlightedRows().then(rows => {
                expect(rows.length).to.be.greaterThan(0);
            });
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        it('highlights matching content in documents', () => {
            cy.data('users');

            cy.searchTable('john');

            cy.getHighlightedCell().first().should('contain.text', 'john');
        });
    });

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        it('highlights matching values', () => {
            cy.data('user:1');

            cy.searchTable('john');

            cy.getHighlightedCell().first().should('contain.text', 'john');
        });
    });

});
