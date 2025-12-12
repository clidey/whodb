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

/**
 * Data Types CRUD Operations Test Suite
 *
 * Tests that each SQL database can correctly handle CRUD operations for all
 * supported data types. Each type is tested independently with ADD, UPDATE,
 * and DELETE operations.
 */
describe('Data Types CRUD Operations', () => {

    forEachDatabase('sql', (db) => {
        const tableName = 'data_types';
        const tableConfig = getTableConfig(db, tableName);
        const mutationDelay = db.mutationDelay || 0;

        if (!tableConfig) {
            it.skip('data_types table config missing in fixture', () => {});
            return;
        }

        const typeTests = tableConfig.testData?.typeTests;
        if (!typeTests) {
            it.skip('typeTests config missing in fixture', () => {});
            return;
        }

        Object.entries(typeTests).forEach(([columnName, testConfig]) => {
            describe(`Type: ${testConfig.type} (${columnName})`, () => {
                const columnIndex = Object.keys(tableConfig.columns).indexOf(columnName);
                const expectedAddDisplay = testConfig.displayAddValue || testConfig.addValue;
                const expectedUpdateDisplay = testConfig.displayUpdateValue || testConfig.updateValue;
                // DELETE test uses separate values to avoid conflicts with ADD test
                const deleteValue = testConfig.deleteValue || testConfig.addValue;
                const expectedDeleteDisplay = testConfig.displayDeleteValue || testConfig.displayAddValue || deleteValue;

                it('ADD - creates row with type value', () => {
                    cy.data(tableName);

                    cy.addRow({[columnName]: testConfig.addValue});

                    if (mutationDelay > 0) {
                        cy.wait(mutationDelay);
                        cy.data(tableName);
                    }

                    cy.getTableData().then(({rows}) => {
                        // Debug: Get all values in this column to see actual format
                        const columnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());

                        const rowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === String(expectedAddDisplay).trim();
                        });
                        // Include actual values in assertion message for debugging
                        expect(rowIndex, `Row with ${columnName}=${expectedAddDisplay} should exist. Actual values: ${JSON.stringify(columnValues)}`).to.not.equal(-1);

                        // Clean up - delete the row we just added
                        cy.deleteRow(rowIndex);
                        if (mutationDelay > 0) {
                            cy.wait(mutationDelay);
                        }
                    });
                });

                it('UPDATE - edits type value', () => {
                    cy.data(tableName);
                    cy.sortBy(0);

                    cy.getTableData().then(({rows}) => {
                        if (rows.length === 0) {
                            throw new Error('No rows in data_types table');
                        }

                        // Debug: Get all values in this column to see actual format
                        const columnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());

                        const originalValue = String(testConfig.originalValue).trim();
                        const targetRowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === originalValue;
                        });

                        if (targetRowIndex === -1) {
                            throw new Error(`Row with original value "${originalValue}" not found in column ${columnName}. Actual values: ${JSON.stringify(columnValues)}`);
                        }

                        cy.updateRow(targetRowIndex, columnIndex, testConfig.updateValue, false);

                        if (mutationDelay > 0) {
                            cy.wait(mutationDelay);
                            cy.data(tableName);
                            cy.sortBy(0);
                        }

                        cy.getTableData().then(({rows: updatedRows}) => {
                            const cellValue = String(updatedRows[targetRowIndex][columnIndex + 1] || '').trim();
                            expect(cellValue).to.equal(String(expectedUpdateDisplay).trim());

                            // Revert using original input value
                            const revertValue = testConfig.inputOriginalValue || testConfig.originalValue;
                            cy.updateRow(targetRowIndex, columnIndex, revertValue, false);

                            if (mutationDelay > 0) {
                                cy.wait(mutationDelay);
                            }
                        });
                    });
                });

                it('DELETE - removes row with type value', () => {
                    cy.data(tableName);

                    cy.addRow({[columnName]: deleteValue});

                    if (mutationDelay > 0) {
                        cy.wait(mutationDelay);
                        cy.data(tableName);
                    }

                    cy.getTableData().then(({rows}) => {
                        const initialCount = rows.length;
                        const columnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());
                        const rowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === String(expectedDeleteDisplay).trim();
                        });

                        expect(rowIndex, `Row with ${columnName}=${expectedDeleteDisplay} should exist to delete. Actual values: ${JSON.stringify(columnValues)}`).to.not.equal(-1);

                        cy.deleteRow(rowIndex);

                        if (mutationDelay > 0) {
                            cy.wait(mutationDelay);
                            cy.data(tableName);
                        }

                        cy.getTableData().then(({rows: newRows}) => {
                            expect(newRows.length).to.equal(initialCount - 1);
                        });
                    });
                });
            });
        });
    });
});
