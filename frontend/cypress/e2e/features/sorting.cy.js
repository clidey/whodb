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
                    // Click first sortable column (after checkbox column)
                    cy.sortBy(0);

                    // Verify ascending indicator appears (ChevronUpIcon inside column text)
                    cy.get('th').eq(1).find('p.flex.items-center svg').should('exist');

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
                cy.sortBy(0);
                cy.sortBy(0);

                // Verify descending indicator appears (ChevronDownIcon inside column text)
                cy.get('th').eq(1).find('p.flex.items-center svg').should('exist');

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

                // Click once - should show ascending indicator (ChevronUp inside the p tag)
                cy.sortBy(0);
                cy.get('th').eq(1).find('p.flex.items-center svg').should('exist');

                // Click twice - should show descending indicator (ChevronDown)
                cy.sortBy(0);
                cy.get('th').eq(1).find('p.flex.items-center svg').should('exist');

                // Click three times - should remove sort indicator
                cy.sortBy(0);
                // Sort icons are inside p.flex.items-center - key icons are elsewhere
                cy.get('th').eq(1).find('p.flex.items-center svg').should('not.exist');
            });

            it('can sort by different columns', () => {
                cy.data(tableName);

                // Sort by second column (index 1)
                cy.sortBy(1);

                // Verify indicator on second column
                cy.get('th').eq(2).find('p.flex.items-center svg').should('exist');

                // Sort by third column (should clear second, add third)
                cy.sortBy(2);

                // Both columns may have sort indicators (multi-column sort)
                cy.get('th').eq(3).find('p.flex.items-center svg').should('exist');
            });
        });

        describe('Sort with Other Features', () => {
            it('maintains sort after search', () => {
                cy.data(tableName);

                // Sort ascending
                cy.sortBy(0);

                // Get sorted data
                cy.getTableData().then(({rows: sortedRows}) => {
                    const firstValue = sortedRows[0][1];

                    // Perform search that still includes first row
                    cy.searchTable(firstValue.substring(0, 2));

                    // Verify sort indicator still present
                    cy.get('th').eq(1).find('p.flex.items-center svg').should('exist');
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

                // Click to sort
                cy.sortBy(0);

                // Verify sort indicator appears (inside column text p tag)
                cy.get('th').eq(1).find('p.flex.items-center svg').should('exist');
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

                // Click to sort
                cy.sortBy(0);

                // Verify sort indicator appears (inside column text p tag)
                cy.get('th').eq(1).find('p.flex.items-center svg').should('exist');
            });
        });
    });

});
