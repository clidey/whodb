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

describe('Sorting', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable || {};
        const tableName = testTable.name || 'users';

        describe('Column Header Sorting', () => {
            it('sorts ascending on first click', () => {
                cy.data(tableName);

                // Get initial data
                cy.getTableData().then(({rows: initialRows}) => {
                    // Click first sortable column
                    cy.sortBy('id');

                    // Verify ascending indicator appears
                    cy.get('[data-column-name="id"] [data-testid="sort-indicator"]').should('exist');
                    cy.get('[data-column-name="id"]').should('have.attr', 'data-sort-direction', 'asc');

                    // Verify data is sorted
                    cy.getTableData().then(({rows: sortedRows}) => {
                        expect(sortedRows.length).to.equal(initialRows.length);
                        // Data should be sorted ascending by first column
                        const values = sortedRows.map(r => r[1]);
                        const sortedValues = [...values].sort((a, b) => {
                            // Handle numeric sorting
                            const aNum = parseFloat(a);
                            const bNum = parseFloat(b);
                            if (!isNaN(aNum) && !isNaN(bNum)) {
                                return aNum - bNum;
                            }
                            return a.localeCompare(b);
                        });
                        expect(values).to.deep.equal(sortedValues);
                    });
                });
            });

            it('sorts descending on second click', () => {
                cy.data(tableName);

                // Click twice to sort descending
                cy.sortBy('id');
                cy.sortBy('id');

                // Verify descending indicator appears
                cy.get('[data-column-name="id"] [data-testid="sort-indicator"]').should('exist');
                cy.get('[data-column-name="id"]').should('have.attr', 'data-sort-direction', 'desc');

                // Verify data is sorted descending
                cy.getTableData().then(({rows}) => {
                    const values = rows.map(r => r[1]);
                    const sortedDesc = [...values].sort((a, b) => {
                        const aNum = parseFloat(a);
                        const bNum = parseFloat(b);
                        if (!isNaN(aNum) && !isNaN(bNum)) {
                            return bNum - aNum;
                        }
                        return b.localeCompare(a);
                    });
                    expect(values).to.deep.equal(sortedDesc);
                });
            });

            it('removes sort on third click', () => {
                cy.data(tableName);

                // Click once - should show ascending indicator
                cy.sortBy('id');
                cy.get('[data-column-name="id"] [data-testid="sort-indicator"]').should('exist');
                cy.get('[data-column-name="id"]').should('have.attr', 'data-sort-direction', 'asc');

                // Click twice - should show descending indicator
                cy.sortBy('id');
                cy.get('[data-column-name="id"] [data-testid="sort-indicator"]').should('exist');
                cy.get('[data-column-name="id"]').should('have.attr', 'data-sort-direction', 'desc');

                // Click three times - should remove sort indicator
                cy.sortBy('id');
                cy.get('[data-column-name="id"] [data-testid="sort-indicator"]').should('not.exist');
                cy.get('[data-column-name="id"]').should('not.have.attr', 'data-sort-direction');
            });

            it('can sort by different columns', () => {
                cy.data(tableName);

                // Sort by username column
                cy.sortBy('username');

                // Verify indicator on username column
                cy.get('[data-column-name="username"] [data-testid="sort-indicator"]').should('exist');
                cy.get('[data-column-name="username"]').should('have.attr', 'data-sort-direction', 'asc');

                // Sort by email column (should clear username, add email)
                cy.sortBy('email');

                // Both columns may have sort indicators (multi-column sort)
                cy.get('[data-column-name="email"] [data-testid="sort-indicator"]').should('exist');
                cy.get('[data-column-name="email"]').should('have.attr', 'data-sort-direction', 'asc');
            });
        });

        describe('Sort with Other Features', () => {
            it('maintains sort after search', () => {
                cy.data(tableName);

                // Sort ascending
                cy.sortBy('id');

                // Get sorted data
                cy.getTableData().then(({rows: sortedRows}) => {
                    const firstValue = sortedRows[0][1];

                    // Perform search that still includes first row
                    cy.searchTable(firstValue.substring(0, 2));

                    // Verify sort indicator still present
                    cy.get('[data-column-name="id"] [data-testid="sort-indicator"]').should('exist');
                    cy.get('[data-column-name="id"]').should('have.attr', 'data-sort-direction', 'asc');
                });
            });
        });
    });

    // Document Databases (MongoDB, Elasticsearch)
    forEachDatabase('document', (db) => {
        const testTable = db.testTable || {};
        const tableName = testTable.name;

        if (!tableName) {
            it.skip('testTable config missing in fixture', () => {});
            return;
        }

        describe('Document Sorting', () => {
            it('can sort document list', () => {
                cy.data(tableName);

                // Click to sort by document column
                cy.sortBy('document');

                // Verify sort indicator appears
                cy.get('[data-column-name="document"] [data-testid="sort-indicator"]').should('exist');
                cy.get('[data-column-name="document"]').should('have.attr', 'data-sort-direction', 'asc');
            });
        });
    });

    // Key-Value Databases (Redis)
    forEachDatabase('keyvalue', (db) => {
        const testTable = db.testTable || {};
        const keyName = testTable.name;

        if (!keyName) {
            it.skip('testTable config missing in fixture', () => {});
            return;
        }

        describe('Key-Value Sorting', () => {
            it('can sort hash fields', () => {
                cy.data(keyName);

                // Click to sort by field column
                cy.sortBy('field');

                // Verify sort indicator appears
                cy.get('[data-column-name="field"] [data-testid="sort-indicator"]').should('exist');
                cy.get('[data-column-name="field"]').should('have.attr', 'data-sort-direction', 'asc');
            });
        });
    });

});
