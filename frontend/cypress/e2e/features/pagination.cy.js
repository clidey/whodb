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
        const testTable = db.testTable;
        const tableName = testTable.name;
        const firstName = testTable.firstName;
        const colIndex = testTable.identifierColIndex;

        it('respects page size setting', () => {
            cy.data(tableName);
            cy.sortBy(0);

            // Set page size to 1
            cy.setTablePageSize(1);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.equal(1);
                expect(rows[0][colIndex + 1]).to.equal(firstName);
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

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        // Redis hashes are fetched as a complete unit (HGETALL), so server-side
        // pagination doesn't apply to hash fields
        if (db.type === 'Redis') {
            it.skip('respects page size setting (Redis hashes do not support field pagination)', () => {
            });
            return;
        }

        it('respects page size setting', () => {
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
