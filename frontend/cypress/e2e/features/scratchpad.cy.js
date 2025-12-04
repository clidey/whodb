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

import {forEachDatabase, getErrorPattern, getSqlQuery, hasFeature} from '../../support/test-runner';

describe('Scratchpad', () => {

    // SQL Databases only
    forEachDatabase('sql', (db) => {
        if (!hasFeature(db, 'scratchpad')) {
            return;
        }

        describe('Query Execution', () => {
            it('executes SELECT query and shows results', () => {
                cy.goto('scratchpad');

                const query = getSqlQuery(db, 'selectAllUsers');
                cy.writeCode(0, query);
                cy.runCode(0);

                cy.getCellQueryOutput(0).then(({columns, rows}) => {
                    expect(columns).to.include('username');
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
                    expect(columns).to.include('user_count');
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

            // Skip UPDATE tests for ClickHouse
            if (db.type !== 'ClickHouse') {
                it('executes UPDATE query', () => {
                    cy.goto('scratchpad');

                    // Update
                    const updateQuery = getSqlQuery(db, 'updateUser');
                    cy.writeCode(0, updateQuery);
                    cy.runCode(0);

                    cy.getCellActionOutput(0).should('contain', 'Action Executed');

                    // Verify
                    cy.addCell(0);
                    const selectQuery = getSqlQuery(db, 'selectUserById');
                    cy.writeCode(1, selectQuery);
                    cy.runCode(1);

                    cy.getCellQueryOutput(1).then(({rows}) => {
                        expect(rows[0]).to.include('john_doe1');
                    });

                    // Revert
                    cy.addCell(1);
                    const revertQuery = getSqlQuery(db, 'revertUser');
                    cy.writeCode(2, revertQuery);
                    cy.runCode(2);

                    cy.getCellActionOutput(2).should('contain', 'Action Executed');
                });
            }
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

        describe('Embedded Scratchpad Drawer', () => {
            it('opens from data view and runs query', () => {
                cy.data('users');

                // Open embedded scratchpad drawer
                cy.get('[data-testid="embedded-scratchpad-button"]').click();
                cy.contains('h2', 'Scratchpad').should('be.visible');

                // Verify default query is populated
                cy.get('[data-testid="code-editor"]').should('exist');
                const schemaPrefix = db.sql?.schemaPrefix || '';
                cy.get('[data-testid="code-editor"]').should('contain', 'SELECT *');
                cy.get('[data-testid="code-editor"]').should('contain', `FROM ${schemaPrefix}users`);

                // Run the query
                cy.get('[data-testid="run-submit-button"]').filter(':contains("Run")').first().click();

                // Verify results appear in the drawer
                cy.get('[role="dialog"] table', {timeout: 5000}).should('be.visible');
                cy.get('[role="dialog"] table thead th').should('contain', 'id');
                cy.get('[role="dialog"] table thead th').should('contain', 'username');
                cy.get('[role="dialog"] table tbody tr').should('have.length.at.least', 1);

                // Close the drawer
                cy.get('body').type('{esc}');
                cy.get('[data-testid="table-search"]').should('be.visible');
            });
        });
    });

});
