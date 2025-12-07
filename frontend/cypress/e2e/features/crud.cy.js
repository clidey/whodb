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

describe('CRUD Operations', () => {

    // SQL Databases - Edit and Add
    forEachDatabase('sql', (db) => {
        // Use testTable config if available, otherwise default to users table
        const testTable = db.testTable || {
            name: 'users',
            identifierField: 'username',
            identifierColIndex: 1,
            testValues: {
                original: 'john_doe',
                modified: 'john_doe1',
                rowIndex: 0
            }
        };
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
                cy.updateRow(testValues.rowIndex, colIndex, 'temp_value', true); // true = cancel

                // Verify no change
                cy.getTableData().then(({rows}) => {
                    expect(rows[testValues.rowIndex][colIndex + 1]).to.equal(testValues.original);
                });
            });
        });

        describe('Add Row', () => {
            // Skip ClickHouse - doesn't support INSERT in the same way
            if (db.type === 'ClickHouse') {
                return;
            }

            it('adds a new row', () => {
                cy.data(tableName);

                const tableConfig = getTableConfig(db, tableName);
                const newRowData = tableConfig?.testData?.newRow;

                if (newRowData) {
                    cy.addRow(newRowData);

                    // Get the identifier field value from new row data
                    const identifierValue = newRowData[testTable.identifierField];

                    // Verify row was added
                    cy.getTableData().then(({rows}) => {
                        const addedRow = rows.find(r => r[colIndex + 1] === identifierValue);
                        expect(addedRow).to.exist;
                    });

                    // Clean up - delete the added row
                    cy.getTableData().then(({rows}) => {
                        const rowIndex = rows.findIndex(r => r[colIndex + 1] === identifierValue);
                        if (rowIndex >= 0) {
                            cy.deleteRow(rowIndex);
                        }
                    });
                }
            });
        });

        describe('Delete Row', () => {
            // Skip ClickHouse - doesn't support DELETE in the same way
            if (db.type === 'ClickHouse') {
                return;
            }

            it('deletes a row and verifies removal', () => {
                cy.data(tableName);

                // First add a row to delete
                const tableConfig = getTableConfig(db, tableName);
                const newRowData = tableConfig?.testData?.newRow;

                if (newRowData) {
                    cy.addRow(newRowData);

                    // Get the identifier field value from new row data
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
                }
            });
        });
    });

    // Document Databases - Add, Edit, Delete
    forEachDatabase('document', (db) => {
        const refreshDelay = db.indexRefreshDelay || 0;

        describe('Add Document', () => {
            it('adds a new document', () => {
                cy.data('users');

                const newDoc = {
                    username: 'new_user',
                    email: 'new@example.com',
                    password: 'newpassword'
                };

                cy.addRow(newDoc, true);

                // Wait for index refresh (Elasticsearch needs this)
                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                }

                // Force refresh by clicking Query button
                cy.submitTable();

                // Wait for data to load after refresh
                cy.get('table tbody tr', {timeout: 15000}).should('have.length.at.least', 1);

                cy.getTableData().then(({rows}) => {
                    // Check row count increased
                    expect(rows.length).to.be.at.least(4);
                    // Find the new document - search case-insensitively in all text
                    const addedRow = rows.find(r => {
                        const text = (r[1] || '').toLowerCase();
                        return text.includes('new_user') || text.includes('new@example.com');
                    });
                    expect(addedRow, 'New document should be found').to.exist;
                });

                // Clean up
                cy.getTableData().then(({rows}) => {
                    const rowIndex = rows.findIndex(r => {
                        const text = (r[1] || '').toLowerCase();
                        return text.includes('new_user') || text.includes('new@example.com');
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
                    cy.data('users');

                    // Wait for data to fully load
                    cy.get('table tbody tr', {timeout: 15000}).should('have.length.at.least', 1);

                    cy.getTableData().then(({rows}) => {
                        // Search case-insensitively and handle null/undefined values
                        let janeRowIndex = rows.findIndex(r => {
                            const text = (r[1] || '').toLowerCase();
                            return text.includes('jane@example.com') || text.includes('jane_smith');
                        });
                        expect(janeRowIndex, 'Jane row should exist').to.be.greaterThan(-1);

                        cy.openContextMenu(janeRowIndex);
                        cy.get('[data-testid="context-menu-edit-row"]').should('be.visible').click();
                        cy.contains('Edit Row').should('be.visible');
                        cy.get('body').type('{esc}');
                        cy.contains('Edit Row').should('not.exist');
                    });
                });
                return;
            }

            it('edits a document and saves changes', () => {
                cy.data('users');
                cy.sortBy(0);

                cy.getTableData().then(({rows}) => {
                    const doc = parseDocument(rows[1]);
                    const updatedDoc = createUpdatedDocument(rows[1], {username: 'jane_smith1'});

                    cy.updateRow(1, 1, updatedDoc, false);
                });

                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                    cy.data('users');
                    cy.sortBy(0);
                }

                cy.getTableData().then(({rows}) => {
                    const doc = parseDocument(rows[1]);
                    expect(doc.username).to.equal('jane_smith1');

                    // Revert
                    const revertedDoc = createUpdatedDocument(rows[1], {username: 'jane_smith'});
                    cy.updateRow(1, 1, revertedDoc, false);
                });

                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                    cy.data('users');
                    cy.sortBy(0);
                }

                cy.getTableData().then(({rows}) => {
                    const doc = parseDocument(rows[1]);
                    expect(doc.username).to.equal('jane_smith');
                });
            });

            it('cancels edit without saving', () => {
                cy.data('users');
                cy.sortBy(0);

                cy.getTableData().then(({rows}) => {
                    const updatedDoc = createUpdatedDocument(rows[1], {username: 'temp_value'});
                    cy.updateRow(1, 1, updatedDoc, true); // true = cancel
                });

                cy.getTableData().then(({rows}) => {
                    const doc = parseDocument(rows[1]);
                    expect(doc.username).to.equal('jane_smith');
                });
            });
        });

        describe('Delete Document', () => {
            it('deletes a document and verifies removal', () => {
                cy.data('users');

                // First add a document to delete
                const newDoc = {
                    username: 'temp_delete_user',
                    email: 'temp@example.com',
                    password: 'temppass'
                };

                cy.addRow(newDoc, true);

                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                    cy.data('users');
                }

                cy.getTableData().then(({rows}) => {
                    const initialCount = rows.length;
                    const rowIndex = rows.findIndex(r => r[1].includes('temp_delete_user') || r[1].includes('temp@example.com'));

                    if (rowIndex >= 0) {
                        cy.deleteRow(rowIndex);

                        if (refreshDelay > 0) {
                            cy.wait(refreshDelay);
                            cy.data('users');
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
        describe('Edit Hash Field', () => {
            it('edits a hash field value and saves', () => {
                cy.data('user:1');

                // Edit username field (index 4)
                cy.updateRow(4, 1, 'johndoe_updated', false);

                cy.getTableData().then(({rows}) => {
                    expect(rows[4][2]).to.equal('johndoe_updated');
                });

                // Revert
                cy.updateRow(4, 1, 'johndoe', false);

                cy.getTableData().then(({rows}) => {
                    expect(rows[4][2]).to.equal('johndoe');
                });
            });

            it('cancels edit without saving', () => {
                cy.data('user:1');

                cy.updateRow(4, 1, 'temp_value', true); // true = cancel

                cy.getTableData().then(({rows}) => {
                    expect(rows[4][2]).to.equal('johndoe');
                });
            });
        });

        describe('Delete Hash Field', () => {
            it('deletes a hash field', () => {
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
