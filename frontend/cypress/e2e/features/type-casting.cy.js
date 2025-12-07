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

describe('Type Casting', () => {

    // SQL Databases only - tests numeric type handling
    forEachDatabase('sql', (db) => {
        // Skip ClickHouse - doesn't support INSERT in the same way
        if (db.type === 'ClickHouse') {
            return;
        }

        const testTable = db.testTable || {};
        const typeCastingTable = testTable.typeCastingTable || 'test_casting';
        const tableConfig = getTableConfig(db, typeCastingTable);
        if (!tableConfig || !tableConfig.testData || !tableConfig.testData.newRow) {
            return;
        }

        describe('Add Row Type Casting', () => {
            it('correctly casts string inputs to numeric types', () => {
                cy.data(typeCastingTable);

                const newRow = tableConfig.testData.newRow;

                // Add a row and verify it was added by checking for its description
                cy.addRow(newRow);

                // Verify the row was added with correct types by finding it via description
                cy.sortBy(0);
                cy.getTableData().then(({rows}) => {
                    const addedRow = rows.find(r => r.includes(newRow.description));
                    expect(addedRow, 'Added row should exist').to.exist;
                    expect(addedRow[1]).to.match(/^\d+$/); // id should be a number
                    expect(addedRow[2]).to.equal(newRow.bigint_col);
                    expect(addedRow[3]).to.equal(newRow.integer_col);
                    expect(addedRow[4]).to.equal(newRow.smallint_col);
                    expect(addedRow[5]).to.equal(newRow.numeric_col);
                    expect(addedRow[6]).to.equal(newRow.description);
                });

                // Clean up
                cy.getTableData().then(({rows}) => {
                    const rowIndex = rows.findIndex(r => r.includes(newRow.description));
                    if (rowIndex >= 0) {
                        cy.deleteRow(rowIndex);
                    }
                });
            });

            it('handles large bigint values', () => {
                cy.data(typeCastingTable);

                const largeNumberRow = {
                    bigint_col: '5000000000',
                    integer_col: '42',
                    smallint_col: '256',
                    numeric_col: '9876.54',
                    description: 'Large bigint test'
                };

                cy.addRow(largeNumberRow);

                cy.getTableData().then(({rows}) => {
                    const addedRow = rows.find(r => r.includes('Large bigint test'));
                    expect(addedRow).to.exist;
                    expect(addedRow).to.include('5000000000');
                });

                // Clean up
                cy.getTableData().then(({rows}) => {
                    const rowIndex = rows.findIndex(r => r.includes('Large bigint test'));
                    if (rowIndex >= 0) {
                        cy.deleteRow(rowIndex);
                    }
                });
            });
        });

        describe('Edit Row Type Casting', () => {
            it('edits numeric values with type casting', () => {
                cy.data(typeCastingTable);
                cy.sortBy(0);

                // Edit bigint_col on second row
                cy.updateRow(1, 1, '7500000000', false);

                cy.getTableData().then(({rows}) => {
                    expect(rows[1][2]).to.equal('7500000000');
                });

                // Restore original value
                cy.updateRow(1, 1, '1000000', false);

                cy.getTableData().then(({rows}) => {
                    expect(rows[1][2]).to.equal('1000000');
                });
            });
        });
    });

});
