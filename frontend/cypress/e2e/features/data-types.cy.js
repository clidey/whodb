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

                it('ADD - creates row with type value', () => {
                    cy.data(tableName);

                    const newRowData = {
                        [columnName]: testConfig.addValue
                    };

                    cy.addRow(newRowData);

                    if (mutationDelay > 0) {
                        cy.wait(mutationDelay);
                        cy.data(tableName);
                    }

                    const columnIndex = Object.keys(tableConfig.columns).indexOf(columnName);

                    cy.getTableData().then(({rows}) => {
                        const addedRow = rows.find(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            const expected = String(testConfig.addValue).trim();
                            return cellValue === expected;
                        });
                        expect(addedRow, `Row with ${columnName}=${testConfig.addValue} should exist`).to.exist;

                        if (addedRow) {
                            const rowIndex = rows.indexOf(addedRow);
                            cy.deleteRow(rowIndex);

                            if (mutationDelay > 0) {
                                cy.wait(mutationDelay);
                            }
                        }
                    });
                });

                it('UPDATE - edits type value', () => {
                    cy.data(tableName);
                    cy.sortBy(0);

                    const columnIndex = Object.keys(tableConfig.columns).indexOf(columnName);

                    cy.getTableData().then(({rows}) => {
                        if (rows.length === 0) {
                            throw new Error('No rows in data_types table');
                        }

                        const originalValue = String(testConfig.originalValue).trim();
                        const targetRowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === originalValue;
                        });

                        if (targetRowIndex === -1) {
                            throw new Error(`Row with original value "${originalValue}" not found in column ${columnName}`);
                        }

                        cy.updateRow(targetRowIndex, columnIndex, testConfig.updateValue, false);

                        if (mutationDelay > 0) {
                            cy.wait(mutationDelay);
                            cy.data(tableName);
                            cy.sortBy(0);
                        }

                        cy.getTableData().then(({rows: updatedRows}) => {
                            const cellValue = String(updatedRows[targetRowIndex][columnIndex + 1] || '').trim();
                            const expected = String(testConfig.updateValue).trim();
                            expect(cellValue).to.equal(expected);

                            // Revert
                            cy.updateRow(targetRowIndex, columnIndex, testConfig.originalValue, false);

                            if (mutationDelay > 0) {
                                cy.wait(mutationDelay);
                            }
                        });
                    });
                });

                it('DELETE - removes row with type value', () => {
                    cy.data(tableName);

                    const newRowData = {
                        [columnName]: testConfig.addValue
                    };

                    cy.addRow(newRowData);

                    if (mutationDelay > 0) {
                        cy.wait(mutationDelay);
                        cy.data(tableName);
                    }

                    const columnIndex = Object.keys(tableConfig.columns).indexOf(columnName);

                    cy.getTableData().then(({rows}) => {
                        const initialCount = rows.length;
                        const addValue = String(testConfig.addValue).trim();
                        const rowIndex = rows.findIndex(r => {
                            const cellValue = String(r[columnIndex + 1] || '').trim();
                            return cellValue === addValue;
                        });

                        if (rowIndex === -1) {
                            throw new Error(`Could not find row with ${columnName}=${testConfig.addValue} to delete`);
                        }

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
