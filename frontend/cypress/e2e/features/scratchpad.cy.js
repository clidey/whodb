/*
 * Copyright 2026 Clidey, Inc.
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

import {forEachDatabase, getErrorPattern, getSqlQuery, hasFeature} from '../../support/test-runner';

describe('Scratchpad', () => {

    // SQL Databases only
    forEachDatabase('sql', (db) => {
        describe('Query Execution', () => {
            // Get expected column names from config
            const expectedIdentifierCol = db.testTable.identifierField;
            const expectedCountCol = db.sql.countColumn;
            const expectedUpdatedValue = db.testTable.testValues.modified;

            it('executes SELECT query and shows results', () => {
                cy.goto('scratchpad');

                const query = getSqlQuery(db, 'selectAllUsers');
                cy.writeCode(0, query);
                cy.runCode(0);

                cy.getCellQueryOutput(0).then(({columns, rows}) => {
                    expect(columns.map(c => c.toUpperCase())).to.include(expectedIdentifierCol.toUpperCase());
                    expect(rows.length).to.be.greaterThan(0);
                });
            });

            it('executes filtered query', () => {
                cy.goto('scratchpad');

                const query = getSqlQuery(db, 'selectUserById');
                cy.writeCode(0, query);
                cy.runCode(0);

                cy.getCellQueryOutput(0).then(({rows}) => {
                    expect(rows.length).to.equal(1);
                });
            });

            it('executes aggregate query', () => {
                cy.goto('scratchpad');

                const query = getSqlQuery(db, 'countUsers');
                cy.writeCode(0, query);
                cy.runCode(0);

                cy.getCellQueryOutput(0).then(({columns, rows}) => {
                    expect(columns.map(c => c.toUpperCase())).to.include(expectedCountCol.toUpperCase());
                    expect(rows.length).to.equal(1);
                });
            });

            it('shows error for invalid query', () => {
                cy.goto('scratchpad');

                const query = getSqlQuery(db, 'invalidQuery');
                cy.writeCode(0, query);
                cy.runCode(0);

                const errorPattern = getErrorPattern(db, 'tableNotFound');
                if (errorPattern) {
                    cy.getCellError(0).should('contain', errorPattern.split(' ')[0]);
                } else {
                    cy.get('[data-testid="cell-error"]').should('be.visible');
                }
            });

            // Skip UPDATE test for databases with async mutations (e.g., ClickHouse)
            const updateSupported = hasFeature(db, 'scratchpadUpdate') !== false;

            (updateSupported ? it : it.skip)('executes UPDATE query', () => {
                cy.goto('scratchpad');

                const mutationDelay = db.mutationDelay || 0;

                // Update
                const updateQuery = getSqlQuery(db, 'updateUser');
                cy.writeCode(0, updateQuery);
                cy.runCode(0);

                cy.getCellActionOutput(0).should('contain', 'Action Executed');

                // Wait for async mutations (e.g., ClickHouse)
                if (mutationDelay > 0) {
                    cy.wait(mutationDelay);
                }

                // Verify
                cy.addCell(0);
                const selectQuery = getSqlQuery(db, 'selectUserById');
                cy.writeCode(1, selectQuery);
                cy.runCode(1);

                cy.getCellQueryOutput(1).then(({rows}) => {
                    expect(rows[0]).to.include(expectedUpdatedValue);
                });

                // Revert
                cy.addCell(1);
                const revertQuery = getSqlQuery(db, 'revertUser');
                cy.writeCode(2, revertQuery);
                cy.runCode(2);

                cy.getCellActionOutput(2).should('contain', 'Action Executed');
            });
        });

        describe('Cell Management', () => {
            it('adds and removes cells', () => {
                cy.goto('scratchpad');

                // Add cells
                cy.addCell(0);
                cy.addCell(1);

                // Verify cells exist
                cy.get('[data-testid="cell-0"]').should('exist');
                cy.get('[data-testid="cell-1"]').should('exist');
                cy.get('[data-testid="cell-2"]').should('exist');

                // Remove middle cell
                cy.removeCell(1);

                cy.get('[data-testid="cell-2"]').should('not.exist');
            });
        });

        describe('Page Management', () => {
            it('creates and manages multiple pages', () => {
                cy.goto('scratchpad');

                // Add new page
                cy.addScratchpadPage();

                cy.getScratchpadPages().then(pages => {
                    expect(pages.length).to.equal(2);
                });

                // Delete page with cancel
                cy.deleteScratchpadPage(1, true);

                cy.getScratchpadPages().then(pages => {
                    expect(pages.length).to.equal(2);
                });

                // Delete page for real
                cy.deleteScratchpadPage(1, false);

                cy.getScratchpadPages().then(pages => {
                    expect(pages.length).to.equal(1);
                });
            });
        });

        describe('Query Export', () => {
            it('exports query results as CSV', () => {
                cy.goto('scratchpad');
                cy.intercept('POST', '/api/export').as('export');

                const query = getSqlQuery(db, 'selectAllUsers');
                cy.writeCode(0, query);
                cy.runCode(0);

                cy.getCellQueryOutput(0).then(({rows}) => {
                    expect(rows.length).to.be.greaterThan(0);
                });

                // Click export button inside the query output
                cy.get('[data-testid="cell-query-output"] [data-testid="export-all-button"]').click();
                cy.contains('h2', 'Export Data').should('be.visible');

                // Verify the raw query export message
                cy.contains('You are about to export the results of your query.').should('be.visible');

                // Verify CSV is selected by default
                cy.get('[data-testid="export-format-select"]').should('contain.text', 'CSV');

                cy.confirmExport();

                cy.wait('@export').then(({request, response}) => {
                    expect(response?.statusCode).to.equal(200);
                    // Data is sent as selectedRows (frontend-only approach)
                    expect(request.body.selectedRows).to.exist;
                    expect(Array.isArray(request.body.selectedRows)).to.be.true;
                    expect(request.body.selectedRows.length).to.be.greaterThan(0);
                    expect(request.body.storageUnit).to.equal('query_export');
                });
            });

            it('exports query results as Excel', () => {
                cy.goto('scratchpad');
                cy.intercept('POST', '/api/export').as('export');

                const query = getSqlQuery(db, 'selectAllUsers');
                cy.writeCode(0, query);
                cy.runCode(0);

                cy.getCellQueryOutput(0).then(({rows}) => {
                    expect(rows.length).to.be.greaterThan(0);
                });

                cy.get('[data-testid="cell-query-output"] [data-testid="export-all-button"]').click();
                cy.selectExportFormat('excel');

                cy.confirmExport();

                cy.wait('@export').then(({request, response}) => {
                    expect(response?.statusCode).to.equal(200);
                    expect(request.body.selectedRows).to.exist;
                    expect(request.body.format).to.equal('excel');
                    expect(request.body.storageUnit).to.equal('query_export');
                });
            });

            it('preselects Excel when "Export All as Excel" is chosen from context menu', () => {
                cy.goto('scratchpad');

                const query = getSqlQuery(db, 'selectAllUsers');
                cy.writeCode(0, query);
                cy.runCode(0);

                cy.getCellQueryOutput(0).then(({rows}) => {
                    expect(rows.length).to.be.greaterThan(0);
                });

                // Right-click on a visible data cell (eq(1) skips the hidden checkbox td)
                cy.get('[data-testid="cell-query-output"] table tbody tr').first().find('td').eq(1).rightclick();
                cy.wait(300);

                // Navigate to Export submenu and click "Export All as Excel"
                cy.get('[role="menu"]').contains('Export').click();
                cy.contains('Export All as Excel').should('be.visible').click();

                // Verify the export dialog opens with Excel preselected
                cy.contains('h2', 'Export Data').should('be.visible');
                cy.get('[data-testid="export-format-select"]').should('contain.text', 'Excel');

                cy.get('body').type('{esc}');
            });

            it('does not show "Export Selected" options in context menu', () => {
                cy.goto('scratchpad');

                const query = getSqlQuery(db, 'selectAllUsers');
                cy.writeCode(0, query);
                cy.runCode(0);

                cy.getCellQueryOutput(0).then(({rows}) => {
                    expect(rows.length).to.be.greaterThan(0);
                });

                // Right-click on a visible data cell (eq(1) skips the hidden checkbox td)
                cy.get('[data-testid="cell-query-output"] table tbody tr').first().find('td').eq(1).rightclick();
                cy.wait(300);

                // Open the Export submenu (scope to context menu to avoid matching "Export All" button)
                cy.get('[role="menu"]').contains('Export').click();

                // "Export All" options should be visible
                cy.contains('Export All as CSV').should('be.visible');
                cy.contains('Export All as Excel').should('be.visible');

                // "Export Selected" options should NOT exist
                cy.contains('Export Selected as CSV').should('not.exist');
                cy.contains('Export Selected as Excel').should('not.exist');

                cy.get('body').type('{esc}');
            });
        });

        describe('Embedded Scratchpad Drawer', () => {
            const testTable = db.testTable;
            const tableName = testTable.name;

            it('opens from data view and runs query', () => {
                cy.data(tableName);

                // Open embedded scratchpad drawer
                cy.get('[data-testid="embedded-scratchpad-button"]').click();
                cy.contains('h2', 'Scratchpad').should('be.visible');

                // Verify default query is populated
                cy.get('[data-testid="code-editor"]').should('exist');
                const schemaPrefix = db.sql.schemaPrefix;
                cy.get('[data-testid="code-editor"]').should('contain', 'SELECT');
                cy.get('[data-testid="code-editor"]').should('contain', `FROM ${schemaPrefix}${tableName}`);

                // Run the query
                cy.get('[data-testid="run-submit-button"]').filter(':contains("Run")').first().click();

                // Verify results appear in the drawer
                cy.get('[role="dialog"] table', {timeout: 5000}).should('be.visible');
                cy.get('[role="dialog"] table tbody tr').should('have.length.at.least', 1);

                // Close the drawer
                cy.get('body').type('{esc}');
                cy.get('[data-testid="table-search"]').should('be.visible');
            });
        });
    }, {features: ['scratchpad']});

});
