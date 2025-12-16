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

import {forEachDatabase, getTableConfig, hasFeature} from '../../support/test-runner';

// Helper to get value from object with case-insensitive key
function getValue(obj, key) {
    const lowerKey = key.toLowerCase();
    const matchingKey = Object.keys(obj).find(k => k.toLowerCase() === lowerKey);
    return matchingKey ? obj[matchingKey] : undefined;
}

describe('Type Casting', () => {

    // SQL Databases only - tests numeric type handling
    forEachDatabase('sql', (db) => {
        // Skip type casting tests for databases with async mutations (e.g., ClickHouse)
        if (hasFeature(db, 'typeCasting') === false) {
            it.skip('type casting tests skipped - async mutations not supported', () => {});
            return;
        }

        const testTable = db.testTable || {};
        const typeCastingTable = testTable.typeCastingTable || 'test_casting';
        const tableConfig = getTableConfig(db, typeCastingTable);
        if (!tableConfig || !tableConfig.testData || !tableConfig.testData.newRow) {
            return;
        }

        const mutationDelay = db.mutationDelay || 0;
        const columns = tableConfig.columns || {};
        const columnNames = Object.keys(columns);
        const bigintCol = columnNames.find(c => c.toLowerCase() === 'bigint_col') || 'bigint_col';
        const integerCol = columnNames.find(c => c.toLowerCase() === 'integer_col') || 'integer_col';
        const smallintCol = columnNames.find(c => c.toLowerCase() === 'smallint_col') || 'smallint_col';
        const numericCol = columnNames.find(c => c.toLowerCase() === 'numeric_col') || 'numeric_col';
        const descriptionCol = columnNames.find(c => c.toLowerCase() === 'description') || 'description';

        describe('Add Row Type Casting', () => {
            it('correctly casts string inputs to numeric types', () => {
                cy.data(typeCastingTable);

                const newRow = tableConfig.testData.newRow;

                // Add a row and verify it was added by checking for its description
                const descValue = getValue(newRow, 'description');
                cy.addRow(newRow);

                // Wait for row to appear using retry-able assertion
                cy.waitForRowContaining(descValue, { caseSensitive: true }).then((rowIndex) => {
                    cy.sortBy(0);

                    // Verify the row was added with correct types
                    cy.getTableData().then(({rows}) => {
                        const addedRow = rows.find(r => r.includes(descValue));
                        expect(addedRow, 'Added row should exist').to.exist;
                        expect(addedRow[1]).to.match(/^\d+$/); // id should be a number
                        expect(addedRow[2]).to.equal(getValue(newRow, 'bigint_col'));
                        expect(addedRow[3]).to.equal(getValue(newRow, 'integer_col'));
                        expect(addedRow[4]).to.equal(getValue(newRow, 'smallint_col'));
                        expect(addedRow[5]).to.equal(getValue(newRow, 'numeric_col'));
                        expect(addedRow[6]).to.equal(descValue);

                        // Clean up - find row index again after sort
                        const deleteIndex = rows.findIndex(r => r.includes(descValue));
                        if (deleteIndex >= 0) {
                            cy.deleteRow(deleteIndex);
                        }
                    });
                });
            });

            it('handles large bigint values', () => {
                cy.data(typeCastingTable);

                // Build row with correct column names for this database
                const largeNumberRow = {
                    [bigintCol]: '5000000000',
                    [integerCol]: '42',
                    [smallintCol]: '256',
                    [numericCol]: '9876.54',
                    [descriptionCol]: 'Large bigint test'
                };

                cy.addRow(largeNumberRow);

                // Wait for row to appear using retry-able assertion
                cy.waitForRowContaining('Large bigint test', { caseSensitive: true }).then((rowIndex) => {
                    cy.getTableData().then(({rows}) => {
                        const addedRow = rows.find(r => r.includes('Large bigint test'));
                        expect(addedRow).to.exist;
                        expect(addedRow).to.include('5000000000');

                        // Clean up
                        cy.deleteRow(rowIndex);
                    });
                });
            });
        });

        describe('Edit Row Type Casting', () => {
            it('edits numeric values with type casting', () => {
                cy.data(typeCastingTable);
                cy.sortBy(0);

                // Edit bigint_col on second row
                cy.updateRow(1, 1, '7500000000', false);

                // Wait for async mutations (e.g., ClickHouse)
                if (mutationDelay > 0) {
                    cy.wait(mutationDelay);
                    cy.data(typeCastingTable);
                    cy.sortBy(0);
                }

                cy.getTableData().then(({rows}) => {
                    expect(rows[1][2]).to.equal('7500000000');
                });

                // Restore original value
                cy.updateRow(1, 1, '1000000', false);

                // Wait for async mutations
                if (mutationDelay > 0) {
                    cy.wait(mutationDelay);
                    cy.data(typeCastingTable);
                    cy.sortBy(0);
                }

                cy.getTableData().then(({rows}) => {
                    expect(rows[1][2]).to.equal('1000000');
                });
            });
        });
    });

});
