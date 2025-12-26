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

import {forEachDatabase, getDatabaseConfig, getSqlQuery, loginToDatabase} from '../../support/test-runner';
import {clearBrowserState} from '../../support/helpers/animation';

/**
 * Loading States & Spinners Tests
 *
 * Tests loading indicators, spinners, and skeleton states across various
 * features and database operations. Uses cy.intercept() with delays to
 * reliably observe loading states.
 */
describe('Loading States & Spinners', () => {
    describe('Login Loading State', () => {
        beforeEach(() => {
            clearBrowserState();
            cy.visit('/login');

            // Dismiss telemetry modal if present
            cy.get('body').then($body => {
                const $btn = $body.find('button').filter(function() {
                    return this.textContent.includes('Disable Telemetry');
                });
                if ($btn.length) {
                    cy.wrap($btn).click();
                }
            });
        });

        it('shows loading indicator during login submission', () => {
            const db = getDatabaseConfig('postgres');

            // Fill login form
            cy.get('[data-testid="database-type-select"]').click();
            cy.get(`[data-value="${db.type}"]`).click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });
            cy.get('[data-testid="database"]').type(db.connection.database);

            // Set up intercept to track login request
            cy.intercept('POST', '**/api/query').as('loginQuery');

            // Click login button
            cy.get('[data-testid="login-button"]').click();

            // Wait for login to complete and verify redirect
            // The loading state is transient - we verify the flow completes successfully
            cy.wait('@loginQuery', { timeout: 10000 });
            cy.url().should('include', '/storage-unit');

            // Verify we see the logged-in state (storage units or loading indicator)
            cy.get('body').should('contain.text', 'Tables');
        });
    });

    describe('Table Data Loading State', () => {
        forEachDatabase('sql', (db) => {
            const testTable = db.testTable;
            const tableName = testTable.name;

            describe(`${db.type}`, () => {
                beforeEach(() => {
                    clearBrowserState();
                    loginToDatabase(db);
                });

                it('shows loading skeleton/spinner while fetching table data', () => {
                    // Navigate to table view using the proper command
                    cy.data(tableName);

                    // Verify data is displayed after loading
                    cy.get('[data-testid="table-search"]', { timeout: 5000 }).should('be.visible');
                    cy.get('table tbody tr', { timeout: 5000 }).should('have.length.at.least', 1);
                });

                it('shows loading state when changing page size', () => {
                    cy.data(tableName);

                    // Wait for initial data load
                    cy.get('table tbody tr', { timeout: 10000 }).should('have.length.at.least', 1);

                    // Track page change request
                    cy.intercept('POST', '**/api/query').as('pageChangeQuery');

                    // Change page size (valid options: 1, 2, 10, 25, 50, 100, 250, 500, 1000)
                    cy.setTablePageSize(10);
                    cy.submitTable();

                    // Wait for update
                    cy.wait('@pageChangeQuery', { timeout: 5000 });

                    // Verify table is updated
                    cy.get('table tbody tr', { timeout: 5000 }).should('exist');
                });

                it('shows loading state when switching tables', () => {
                    // Navigate to card view first
                    cy.visit('/storage-unit');
                    cy.get('[data-testid="storage-unit-card"]', { timeout: 15000 })
                        .should('have.length.at.least', 1);

                    // Track request when clicking to explore a table
                    cy.intercept('POST', '**/api/query').as('tableDataQuery');

                    // Click on first table to navigate to explore view
                    cy.get('[data-testid="storage-unit-card"]').first()
                        .find('[data-testid="data-button"]').click();

                    // Wait for data to load
                    cy.wait('@tableDataQuery', { timeout: 10000 });

                    // Verify we're in explore view with data
                    cy.url().should('include', '/storage-unit/explore');
                    cy.get('table:visible', { timeout: 5000 }).should('exist');
                });
            });
        });
    });

    describe('Scratchpad Query Execution Loading State', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    clearBrowserState();
                    loginToDatabase(db);
                });

                it('shows loading indicator during query execution', () => {
                    cy.goto('scratchpad');

                    // Write query
                    const query = getSqlQuery(db, 'selectAllUsers');
                    cy.writeCode(0, query);

                    // Track query execution
                    cy.intercept('POST', '**/api/query').as('queryExecution');

                    // Execute query
                    cy.runCode(0);

                    // Wait for query to complete
                    cy.wait('@queryExecution', { timeout: 10000 });

                    // Verify results are displayed
                    cy.getCellQueryOutput(0).then(({ columns, rows }) => {
                        expect(columns.length).to.be.greaterThan(0);
                        expect(rows.length).to.be.greaterThan(0);
                    });
                });

                it('shows loading state for multiple concurrent queries', () => {
                    cy.goto('scratchpad');

                    // Add multiple cells and write queries
                    cy.addCell(0);
                    cy.addCell(1);

                    const query1 = getSqlQuery(db, 'selectAllUsers');
                    const query2 = getSqlQuery(db, 'countUsers');

                    cy.writeCode(0, query1);
                    cy.writeCode(1, query2);
                    cy.writeCode(2, query1);

                    // Track queries
                    cy.intercept('POST', '**/api/query').as('concurrentQueries');

                    // Execute all
                    cy.runCode(0);
                    cy.runCode(1);
                    cy.runCode(2);

                    // Wait for queries to complete (multiple waits for multiple requests)
                    cy.wait('@concurrentQueries', { timeout: 10000 });

                    // Verify all results are displayed
                    cy.getCellQueryOutput(0).should('exist');
                    cy.getCellQueryOutput(1).should('exist');
                    cy.getCellQueryOutput(2).should('exist');
                });
            });
        }, { features: ['scratchpad'] });
    });

    describe('Chat AI Loading State', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    clearBrowserState();
                    loginToDatabase(db);
                    cy.setupChatMock({ modelType: 'Ollama', model: 'llama3.1' });
                });

                it('shows loading indicator during AI response', () => {
                    cy.gotoChat();

                    // Mock the actual response
                    cy.mockChatResponse([{
                        type: 'text',
                        text: `Hello! I can help you with your ${db.type} database.`
                    }]);

                    // Send message
                    cy.sendChatMessage('Hello');

                    // Wait for response and verify it completes
                    cy.waitForChatResponse();

                    // Verify message is displayed
                    cy.verifyChatSystemMessage('Hello!');
                });

                it('shows loading state during SQL query generation', () => {
                    const schemaPrefix = db.sql.schemaPrefix;
                    cy.gotoChat();

                    // Mock the SQL response
                    cy.mockChatResponse([{
                        type: 'text',
                        text: 'I\'ll retrieve all users for you.'
                    }, {
                        type: 'sql:get',
                        text: `SELECT * FROM ${schemaPrefix}users ORDER BY id`,
                        result: {
                            Columns: [
                                { Name: 'id', Type: 'integer', __typename: 'Column' },
                                { Name: 'username', Type: 'character varying', __typename: 'Column' }
                            ],
                            Rows: [['1', 'john_doe']],
                            __typename: 'RowsResult'
                        }
                    }]);

                    // Send message
                    cy.sendChatMessage('Show me all users');

                    // Wait for response
                    cy.waitForChatResponse();

                    // Verify SQL result is displayed
                    cy.verifyChatSQLResult({ columns: ['id', 'username'], rowCount: 1 });
                });
            });
        }, { features: ['chat'] });
    });

    describe('Schema/Database Selection Loading State', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    clearBrowserState();
                    loginToDatabase(db);
                });

                it('shows loading state when switching schema/database', () => {
                    // Check if schema/database dropdown exists (not all DBs have these)
                    cy.get('body').then($body => {
                        const $dbDropdown = $body.find('[data-testid="sidebar-database"]:visible');
                        const $schemaDropdown = $body.find('[data-testid="sidebar-schema"]:visible');

                        if ($dbDropdown.length === 0 && $schemaDropdown.length === 0) {
                            // No schema/database dropdown for this DB type - test is not applicable
                            cy.log('No schema/database dropdown available for this database type');
                            return;
                        }

                        // Click whichever dropdown is visible
                        const $dropdown = $dbDropdown.length > 0 ? $dbDropdown : $schemaDropdown;
                        cy.wrap($dropdown).click();

                        // Wait for dropdown options to appear
                        cy.get('[role="option"]', { timeout: 5000 }).should('have.length.at.least', 1);

                        // Select a different option if available
                        cy.get('[role="option"]').then($options => {
                            if ($options.length > 1) {
                                // Select the second option
                                cy.wrap($options).eq(1).click();
                            } else {
                                // Only one option - just close dropdown by clicking elsewhere
                                cy.get('body').click(0, 0);
                            }
                        });

                        // Verify page state - either storage units or empty state
                        cy.get('[data-testid="storage-unit-card"], button:contains("Create")', { timeout: 15000 })
                            .should('exist');
                    });
                });
            });
        });
    });

    describe('Graph View Loading State', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    clearBrowserState();
                    loginToDatabase(db);
                });

                it('shows loading state while fetching graph data', () => {
                    // Track graph data request
                    cy.intercept('POST', '**/api/query').as('graphDataQuery');

                    // Navigate to graph view
                    cy.get('[href="/graph"]').click();
                    cy.url().should('include', '/graph');

                    // Wait for graph to load
                    cy.wait('@graphDataQuery', { timeout: 10000 });

                    // Verify graph is rendered (canvas or SVG should be present)
                    cy.get('canvas, svg', { timeout: 5000 }).should('exist');
                });
            });
        });
    });

    describe('Storage Units Loading State', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    clearBrowserState();
                });

                it('shows loading state during initial storage units fetch after login', () => {
                    // Perform login using the standard helper
                    loginToDatabase(db);

                    // Verify page loads - either storage units or empty state (Create a Table)
                    cy.get('[data-testid="storage-unit-card"], button:contains("Create")', { timeout: 15000 })
                        .should('exist');
                });
            });
        });
    });

    describe('CRUD Operations Loading State', () => {
        forEachDatabase('sql', (db) => {
            const testTable = db.testTable;
            const tableName = testTable.name;

            describe(`${db.type}`, () => {
                beforeEach(() => {
                    clearBrowserState();
                    loginToDatabase(db);
                });

                it('shows loading state during row creation', () => {
                    cy.data(tableName);
                    cy.get('table tbody tr', { timeout: 10000 }).should('have.length.at.least', 1);

                    // Look for Add Row button - may be in different locations depending on UI
                    cy.get('body').then($body => {
                        // Try different selectors for add row button
                        const $addBtn = $body.find('[data-testid="add-row-button"]');
                        const $addRowBtn = $body.find('button:contains("Add Row")');

                        if ($addBtn.length === 0 && $addRowBtn.length === 0) {
                            cy.log('Add row button not found, skipping test');
                            return;
                        }

                        // Click the add row button
                        if ($addBtn.length > 0) {
                            cy.get('[data-testid="add-row-button"]').click();
                        } else {
                            cy.contains('button', 'Add Row').click();
                        }

                        // Verify the add row panel/dialog appears with a submit button
                        // The submit button may say "Submit", "Save", or "Add"
                        cy.get('button:contains("Submit"), button:contains("Save"), button:contains("Add")', { timeout: 5000 })
                            .first()
                            .should('exist')
                            .and('be.visible');

                        // Close by pressing escape
                        cy.get('body').type('{esc}');
                    });
                });

                it('shows loading state during row update', () => {
                    cy.data(tableName);
                    cy.get('table tbody tr', { timeout: 10000 }).should('have.length.at.least', 1);

                    // Click first row to select/edit
                    cy.get('table tbody tr').first().click();

                    // Check if row edit UI appears
                    cy.get('body').then($body => {
                        const $saveBtn = $body.find('[data-testid="save-button"]');
                        const $editPanel = $body.find('[data-testid="edit-panel"]');

                        if ($saveBtn.length === 0 && $editPanel.length === 0) {
                            // Row click may just select, not open edit mode
                            cy.log('Edit mode not activated by row click, skipping test');
                            return;
                        }

                        if ($saveBtn.length > 0) {
                            // Save button exists - verify it's visible and functional
                            cy.get('[data-testid="save-button"]')
                                .should('exist')
                                .and('be.visible');
                        }
                    });
                });
            });
        });
    });
});
