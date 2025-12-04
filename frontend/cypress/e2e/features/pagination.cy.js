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

describe('Pagination', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        it('respects page size setting', () => {
            cy.data('users');
            cy.sortBy(0);

            // Set page size to 1
            cy.setTablePageSize(1);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.equal(1);
                expect(rows[0][2]).to.equal('john_doe');
            });

            // Set page size to 2
            cy.setTablePageSize(2);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.equal(2);
            });

            // Reset to default
            cy.setTablePageSize(10);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.be.greaterThan(2);
            });
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        it('respects page size setting', () => {
            cy.data('users');

            // Set page size to 1
            cy.setTablePageSize(1);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.equal(1);
            });

            // Reset to default
            cy.setTablePageSize(10);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.be.greaterThan(1);
            });
        });
    });

    // Key-Value Databases (Redis)
    forEachDatabase('keyvalue', (db) => {
        it('respects page size setting', () => {
            // Test with a multi-value key (hash type with 5 fields)
            cy.data('user:1');

            // Set page size to 2
            cy.setTablePageSize(2);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.be.at.most(2);
            });

            // Reset to default
            cy.setTablePageSize(10);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.be.greaterThan(2);
            });
        });
    });

});
