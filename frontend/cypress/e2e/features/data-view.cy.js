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
import {verifyDocumentRows} from '../../support/categories/document';
import {verifyColumnsForType, verifyMembers, verifyStringValue,} from '../../support/categories/keyvalue';

describe('Data View', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;

        it('displays table data with correct columns', () => {
            cy.data(tableName);
            cy.sortBy(0); // Sort by first column (id)

            cy.getTableData().then(({columns, rows}) => {
                const tableConfig = getTableConfig(db, tableName);

                // Verify columns
                if (tableConfig && tableConfig.expectedColumns) {
                    expect(columns).to.deep.equal(tableConfig.expectedColumns);
                }

                // Verify data exists
                expect(rows.length).to.be.greaterThan(0);

                // Verify first row data if configured (initial should be array of arrays)
                if (tableConfig && tableConfig.testData && tableConfig.testData.initial && Array.isArray(tableConfig.testData.initial[0])) {
                    const expectedFirst = tableConfig.testData.initial[0];
                    expectedFirst.forEach((val, idx) => {
                        if (val !== '') {
                            expect(rows[0][idx]).to.equal(val);
                        }
                    });
                }
            });
        });

        it('respects page size pagination', () => {
            cy.data(tableName);
            cy.setTablePageSize(1);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.equal(1);
            });
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        it('displays document data', () => {
            cy.data('users');
            cy.sortBy(0);

            cy.getTableData().then(({columns, rows}) => {
                const tableConfig = getTableConfig(db, 'users');

                // Document DBs have [checkbox, document] columns
                if (tableConfig && tableConfig.expectedColumns) {
                    expect(columns).to.deep.equal(tableConfig.expectedColumns);
                }

                // Verify document content
                if (tableConfig && tableConfig.testData && tableConfig.testData.initial) {
                    verifyDocumentRows(rows, tableConfig.testData.initial);
                }
            });
        });

        it('respects page size pagination', () => {
            cy.data('users');
            cy.setTablePageSize(1);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.equal(1);
            });
        });
    });

    // Key-Value Databases
    forEachDatabase('keyvalue', (db) => {
        it('displays hash data correctly', () => {
            cy.data('user:2');
            cy.getTableData().then(({columns, rows}) => {
                const keyConfig = db.keyTypes['user:2'];
                verifyColumnsForType(columns, 'hash');

                if (keyConfig.expectedRowCount) {
                    expect(rows.length).to.equal(keyConfig.expectedRowCount);
                }
                if (keyConfig.testData && keyConfig.testData.firstRow) {
                    expect(rows[0]).to.deep.equal(keyConfig.testData.firstRow);
                }
            });
        });

        it('displays list data correctly', () => {
            cy.data('orders:recent');
            cy.getTableData().then(({columns, rows}) => {
                verifyColumnsForType(columns, 'list');
                expect(rows.length).to.be.greaterThan(0);
            });
        });

        it('displays set data correctly', () => {
            cy.data('category:electronics');
            cy.getTableData().then(({columns, rows}) => {
                const keyConfig = db.keyTypes['category:electronics'];
                verifyColumnsForType(columns, 'set');

                if (keyConfig.expectedMembers) {
                    const members = rows.map(row => row[2]);
                    verifyMembers(rows, keyConfig.expectedMembers);
                }
            });
        });

        it('displays sorted set data correctly', () => {
            cy.data('products:by_price');
            cy.getTableData().then(({columns, rows}) => {
                verifyColumnsForType(columns, 'zset');
                expect(rows.length).to.be.greaterThan(0);
            });
        });

        it('displays string data correctly', () => {
            cy.data('inventory:product:1');
            cy.getTableData().then(({columns, rows}) => {
                const keyConfig = db.keyTypes['inventory:product:1'];
                verifyColumnsForType(columns, 'string');

                if (keyConfig.expectedValue) {
                    verifyStringValue(rows, keyConfig.expectedValue);
                }
            });
        });

        // Redis hashes are fetched as a complete unit (HGETALL), so server-side
        // pagination doesn't apply to hash fields
        if (db.type === 'Redis') {
            it.skip('respects page size pagination (Redis hashes do not support field pagination)', () => {
            });
            return;
        }

        it('respects page size pagination', () => {
            cy.data('user:1');
            cy.setTablePageSize(2);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.be.at.most(2);
            });
        });
    });

});
