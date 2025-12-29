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
        const tableName = db.dataTypesTable;
        const tableConfig = tableName ? getTableConfig(db, tableName) : null;
        const mutationDelay = db.mutationDelay || 0;

        if (!tableConfig) {
            it.skip('data_types table config missing in fixture', () => {
            });
            return;
        }

        const typeTests = tableConfig.testData?.typeTests;
        if (!typeTests) {
            it.skip('typeTests config missing in fixture', () => {
            });
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

                it('READ - displays seed data with correct format', () => {
                    cy.data(tableName);
                    cy.sortBy(0);

                    cy.getTableData().then(({rows}) => {
                        if (rows.length === 0) {
                            throw new Error('No rows in data_types table');
                        }

                        const expectedOriginal = String(testConfig.originalValue).trim();
                        const expectedUpdate = String(testConfig.displayUpdateValue || testConfig.updateValue).trim();
                        const columnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());

                        // Accept either original or update value (handles leftover state from failed UPDATE tests)
                        const seedRowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === expectedOriginal || cellValue === expectedUpdate;
                        });

                        expect(
                            seedRowIndex,
                            `Seed data with ${columnName}=${expectedOriginal} or ${expectedUpdate} should exist. Actual values: ${JSON.stringify(columnValues)}`
                        ).to.not.equal(-1);

                        // Verify the value matches one of the expected formats
                        const actualValue = String(rows[seedRowIndex][columnIndex + 1] || '').trim();
                        expect(
                            actualValue === expectedOriginal || actualValue === expectedUpdate,
                            `${testConfig.type} should display as "${expectedOriginal}" or "${expectedUpdate}", got "${actualValue}"`
                        ).to.be.true;
                    });
                });

                it('ADD - creates row with type value', () => {
                    cy.data(tableName);

                    cy.addRow({[columnName]: testConfig.addValue});

                    // Use retry-able assertion to wait for the new row to appear
                    // columnIndex + 1 accounts for the checkbox column
                    cy.waitForRowValue(columnIndex + 1, expectedAddDisplay).then((rowIndex) => {
                        // Clean up - delete the row we just added
                        cy.deleteRow(rowIndex);
                    });
                });

                it('UPDATE - edits type value', () => {
                    const originalValue = String(testConfig.originalValue).trim();
                    const updateDisplayValue = String(expectedUpdateDisplay).trim();
                    const revertValue = testConfig.inputOriginalValue || testConfig.originalValue;

                    cy.data(tableName);
                    cy.sortBy(0);

                    // First pass: check if we need to revert leftover data from failed tests
                    cy.getTableData().then(({rows}) => {
                        if (rows.length === 0) {
                            throw new Error('No rows in data_types table');
                        }

                        const columnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());

                        // Find row with either original OR update value
                        let targetRowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === originalValue;
                        });

                        // If not found with original, try finding with update value (leftover from failed test)
                        if (targetRowIndex === -1) {
                            targetRowIndex = rows.findIndex(r => {
                                const cellValue = String(r[columnIndex + 1] || '').trim();
                                return cellValue === updateDisplayValue;
                            });
                            if (targetRowIndex !== -1) {
                                // Revert - table will auto-refresh
                                cy.updateRow(targetRowIndex, columnIndex, revertValue, false);
                                // Wait for auto-refresh to show original value
                                cy.waitForRowValue(columnIndex + 1, originalValue);
                            }
                        }

                        if (targetRowIndex === -1) {
                            throw new Error(`Row with value "${originalValue}" or "${updateDisplayValue}" not found in column ${columnName}. Actual values: ${JSON.stringify(columnValues)}`);
                        }
                    });

                    // Second pass: perform the actual UPDATE test with clean data
                    cy.getTableData().then(({rows}) => {
                        const targetRowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === originalValue;
                        });

                        if (targetRowIndex === -1) {
                            const columnValues = rows.map(r => String(r[columnIndex + 1] || '').trim());
                            throw new Error(`Row with original value "${originalValue}" not found. Actual values: ${JSON.stringify(columnValues)}`);
                        }

                        cy.updateRow(targetRowIndex, columnIndex, testConfig.updateValue, false);

                        // Wait for auto-refresh to show updated value (Cypress retries until value appears)
                        cy.waitForRowValue(columnIndex + 1, updateDisplayValue).then(() => {
                            // Verify the update succeeded by reading final state
                            cy.getTableData().then(({rows: updatedRows}) => {
                                const cellValue = String(updatedRows[targetRowIndex][columnIndex + 1] || '').trim();
                                expect(cellValue).to.equal(updateDisplayValue);

                                // Revert to original value
                                cy.updateRow(targetRowIndex, columnIndex, revertValue, false);

                                // Wait for revert to complete
                                cy.waitForRowValue(columnIndex + 1, originalValue);
                            });
                        });
                    });
                });

                it('DELETE - removes row with type value', () => {
                    cy.data(tableName);

                    cy.addRow({[columnName]: deleteValue});

                    // Use retry-able assertion to wait for the new row to appear
                    cy.waitForRowValue(columnIndex + 1, expectedDeleteDisplay).then((rowIndex) => {
                        // Get initial count after the row has appeared
                        cy.getTableData().then(({rows}) => {
                            const initialCount = rows.length;

                            cy.deleteRow(rowIndex);

                            // Verify row count decreased
                            cy.getTableData().then(({rows: newRows}) => {
                                expect(newRows.length).to.equal(initialCount - 1);
                            });
                        });
                    });
                });
            });
        });
    });
});
