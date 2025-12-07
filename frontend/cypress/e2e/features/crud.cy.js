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
import {createUpdatedDocument, parseDocument} from '../../support/categories/document';

/**
 * Generates a unique test identifier for this test run.
 * Uses timestamp to avoid conflicts within and across test runs.
 */
function getUniqueTestId() {
    return `test_${Date.now()}`;
}

describe('CRUD Operations', () => {

    // SQL Databases - Edit and Add
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        if (!testTable) {
            it.skip('testTable config missing in fixture', () => {});
            return;
        }

        const tableName = testTable.name;
        const colIndex = testTable.identifierColIndex;
        const testValues = testTable.testValues;

        describe('Edit Row', () => {
            it('edits a row and saves changes', () => {
                cy.data(tableName);
                cy.sortBy(0);

                // Edit row
                cy.updateRow(testValues.rowIndex, colIndex, testValues.modified, false);

                // Verify change
                cy.getTableData().then(({rows}) => {
                    expect(rows[testValues.rowIndex][colIndex + 1]).to.equal(testValues.modified);
                });

                // Revert
                cy.updateRow(testValues.rowIndex, colIndex, testValues.original, false);

                cy.getTableData().then(({rows}) => {
                    expect(rows[testValues.rowIndex][colIndex + 1]).to.equal(testValues.original);
                });
            });

            it('cancels edit without saving', () => {
                cy.data(tableName);
                cy.sortBy(0);

                // Edit and cancel
                cy.updateRow(testValues.rowIndex, colIndex, 'temp_value', true);

                // Verify no change
                cy.getTableData().then(({rows}) => {
                    expect(rows[testValues.rowIndex][colIndex + 1]).to.equal(testValues.original);
                });
            });
        });

        describe('Add Row', () => {
            it('adds a new row', () => {
                cy.data(tableName);

                const tableConfig = getTableConfig(db, tableName);
                const newRowTemplate = tableConfig?.testData?.newRow;

                if (!newRowTemplate) {
                    cy.log('No newRow test data configured, skipping');
                    return;
                }

                // Create unique row data to avoid conflicts
                const uniqueId = getUniqueTestId();
                const newRowData = {...newRowTemplate};
                newRowData[testTable.identifierField] = `${uniqueId}_user`;
                if (newRowData.email) {
                    newRowData.email = `${uniqueId}@example.com`;
                }

                cy.addRow(newRowData);

                const identifierValue = newRowData[testTable.identifierField];

                // Verify row was added
                cy.getTableData().then(({rows}) => {
                    const addedRow = rows.find(r => r[colIndex + 1] === identifierValue);
                    expect(addedRow, `Row with ${testTable.identifierField}=${identifierValue} should exist`).to.exist;
                });

                // Clean up - delete the added row
                cy.getTableData().then(({rows}) => {
                    const rowIndex = rows.findIndex(r => r[colIndex + 1] === identifierValue);
                    if (rowIndex >= 0) {
                        cy.deleteRow(rowIndex);
                    }
                });
            });
        });

        describe('Delete Row', () => {
            it('deletes a row and verifies removal', () => {
                cy.data(tableName);

                const tableConfig = getTableConfig(db, tableName);
                const newRowTemplate = tableConfig?.testData?.newRow;

                if (!newRowTemplate) {
                    cy.log('No newRow test data configured, skipping');
                    return;
                }

                // Create unique row data
                const uniqueId = getUniqueTestId();
                const newRowData = {...newRowTemplate};
                newRowData[testTable.identifierField] = `${uniqueId}_delete`;
                if (newRowData.email) {
                    newRowData.email = `${uniqueId}@example.com`;
                }

                cy.addRow(newRowData);

                const identifierValue = newRowData[testTable.identifierField];

                cy.getTableData().then(({rows}) => {
                    const initialCount = rows.length;
                    const rowIndex = rows.findIndex(r => r[colIndex + 1] === identifierValue);

                    if (rowIndex >= 0) {
                        cy.deleteRow(rowIndex);

                        cy.getTableData().then(({rows: newRows}) => {
                            expect(newRows.length).to.equal(initialCount - 1);
                        });
                    }
                });
            });
        });
    });

    // Document Databases - Add, Edit, Delete
    forEachDatabase('document', (db) => {
        const testTable = db.testTable;
        if (!testTable) {
            it.skip('testTable config missing in fixture', () => {});
            return;
        }

        const tableName = testTable.name;
        const testValues = testTable.testValues;
        const refreshDelay = db.indexRefreshDelay || 0;

        describe('Add Document', () => {
            it('adds a new document', () => {
                cy.data(tableName);

                // Create unique document to avoid conflicts
                const uniqueId = getUniqueTestId();
                const newDoc = {
                    username: `${uniqueId}_user`,
                    email: `${uniqueId}@example.com`,
                    password: 'newpassword'
                };

                cy.addRow(newDoc, true);

                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                }

                cy.submitTable();
                cy.get('table tbody tr', {timeout: 15000}).should('have.length.at.least', 1);

                cy.getTableData().then(({rows}) => {
                    const addedRow = rows.find(r => {
                        const text = (r[1] || '').toLowerCase();
                        return text.includes(uniqueId);
                    });
                    expect(addedRow, 'New document should be found').to.exist;
                });

                // Clean up
                cy.getTableData().then(({rows}) => {
                    const rowIndex = rows.findIndex(r => {
                        const text = (r[1] || '').toLowerCase();
                        return text.includes(uniqueId);
                    });
                    if (rowIndex >= 0) {
                        cy.deleteRow(rowIndex);
                        if (refreshDelay > 0) {
                            cy.wait(refreshDelay);
                        }
                    }
                });
            });
        });

        describe('Edit Document', () => {
            // Skip full edit test for Elasticsearch due to truncated JSON display
            if (db.type === 'ElasticSearch') {
                it('cancels edit without saving', () => {
                    cy.data(tableName);

                    cy.get('table tbody tr', {timeout: 15000}).should('have.length.at.least', 1);

                    cy.getTableData().then(({rows}) => {
                        const targetRowIndex = rows.findIndex(r => {
                            const text = (r[1] || '').toLowerCase();
                            return text.includes(testValues.original);
                        });
                        expect(targetRowIndex, `Row with ${testValues.original} should exist`).to.be.greaterThan(-1);

                        cy.openContextMenu(targetRowIndex);
                        cy.get('[data-testid="context-menu-edit-row"]').should('be.visible').click();
                        cy.contains('Edit Row').should('be.visible');
                        cy.get('body').type('{esc}');
                        cy.contains('Edit Row').should('not.exist');
                    });
                });
                return;
            }

            it('edits a document and saves changes', () => {
                cy.data(tableName);
                cy.sortBy(0);

                cy.getTableData().then(({rows}) => {
                    const updatedDoc = createUpdatedDocument(rows[testValues.rowIndex], {
                        [testTable.identifierField]: testValues.modified
                    });

                    cy.updateRow(testValues.rowIndex, 1, updatedDoc, false);
                });

                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                    cy.data(tableName);
                    cy.sortBy(0);
                }

                cy.getTableData().then(({rows}) => {
                    const doc = parseDocument(rows[testValues.rowIndex]);
                    expect(doc[testTable.identifierField]).to.equal(testValues.modified);

                    // Revert
                    const revertedDoc = createUpdatedDocument(rows[testValues.rowIndex], {
                        [testTable.identifierField]: testValues.original
                    });
                    cy.updateRow(testValues.rowIndex, 1, revertedDoc, false);
                });

                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                    cy.data(tableName);
                    cy.sortBy(0);
                }

                cy.getTableData().then(({rows}) => {
                    const doc = parseDocument(rows[testValues.rowIndex]);
                    expect(doc[testTable.identifierField]).to.equal(testValues.original);
                });
            });

            it('cancels edit without saving', () => {
                cy.data(tableName);
                cy.sortBy(0);

                cy.getTableData().then(({rows}) => {
                    const updatedDoc = createUpdatedDocument(rows[testValues.rowIndex], {
                        [testTable.identifierField]: 'temp_value'
                    });
                    cy.updateRow(testValues.rowIndex, 1, updatedDoc, true);
                });

                cy.getTableData().then(({rows}) => {
                    const doc = parseDocument(rows[testValues.rowIndex]);
                    expect(doc[testTable.identifierField]).to.equal(testValues.original);
                });
            });
        });

        describe('Delete Document', () => {
            it('deletes a document and verifies removal', () => {
                cy.data(tableName);

                // Create unique document to delete
                const uniqueId = getUniqueTestId();
                const newDoc = {
                    username: `${uniqueId}_delete`,
                    email: `${uniqueId}@example.com`,
                    password: 'temppass'
                };

                cy.addRow(newDoc, true);

                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                    cy.data(tableName);
                }

                cy.getTableData().then(({rows}) => {
                    const initialCount = rows.length;
                    const rowIndex = rows.findIndex(r => {
                        const text = (r[1] || '');
                        return text.includes(uniqueId);
                    });

                    if (rowIndex >= 0) {
                        cy.deleteRow(rowIndex);

                        if (refreshDelay > 0) {
                            cy.wait(refreshDelay);
                            cy.data(tableName);
                            cy.wait(1000);
                        }

                        cy.getTableData().then(({rows: newRows}) => {
                            expect(newRows.length).to.equal(initialCount - 1);
                        });
                    }
                });
            });
        });
    });

    // Key-Value Databases - Edit hash fields
    forEachDatabase('keyvalue', (db) => {
        const testTable = db.testTable;
        if (!testTable) {
            it.skip('testTable config missing in fixture', () => {});
            return;
        }

        const keyName = testTable.name;
        const testValues = testTable.testValues;
        const rowIndex = testTable.identifierRowIndex || testValues.rowIndex;

        describe('Edit Hash Field', () => {
            it('edits a hash field value and saves', () => {
                cy.data(keyName);

                cy.updateRow(rowIndex, 1, testValues.modified, false);

                cy.getTableData().then(({rows}) => {
                    expect(rows[rowIndex][2]).to.equal(testValues.modified);
                });

                // Revert
                cy.updateRow(rowIndex, 1, testValues.original, false);

                cy.getTableData().then(({rows}) => {
                    expect(rows[rowIndex][2]).to.equal(testValues.original);
                });
            });

            it('cancels edit without saving', () => {
                cy.data(keyName);

                cy.updateRow(rowIndex, 1, 'temp_value', true);

                cy.getTableData().then(({rows}) => {
                    expect(rows[rowIndex][2]).to.equal(testValues.original);
                });
            });
        });

        describe('Delete Hash Field', () => {
            it('deletes a hash field', () => {
                // Use user:2 for delete test to avoid affecting user:1 used in edit tests
                cy.data('user:2');

                cy.getTableData().then(({rows}) => {
                    const initialCount = rows.length;

                    cy.deleteRow(2);

                    cy.getTableData().then(({rows: newRows}) => {
                        expect(newRows.length).to.equal(initialCount - 1);
                    });
                });
            });
        });
    });

});
