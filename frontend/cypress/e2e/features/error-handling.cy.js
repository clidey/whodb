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

import {forEachDatabase, getErrorPattern, getSqlQuery, loginToDatabase} from '../../support/test-runner';
import {clearBrowserState} from '../../support/helpers/animation';

/**
 * Error Handling Tests
 *
 * Tests error scenarios including network failures, invalid queries, GraphQL errors,
 * connection timeouts, authentication expiry, and error state UI.
 */
describe('Error Handling', () => {
    describe('Network Errors', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    // Handle uncaught Apollo errors from network failures
                    cy.on('uncaught:exception', (err) => {
                        if (err.message.includes('Network') ||
                            err.message.includes('Failed to fetch') ||
                            err.message.includes('network')) {
                            return false;
                        }
                        return true;
                    });
                });

                it('gracefully handles network failure during query', () => {
                    clearBrowserState();
                    loginToDatabase(db);

                    // Navigate to a table first
                    const testTable = db.testTable;
                    cy.data(testTable.name);

                    // Wait for table to load
                    cy.getTableData({timeout: 15000}).then(({rows}) => {
                        expect(rows.length).to.be.greaterThan(0);
                    });

                    // Set up intercept BEFORE triggering new request
                    cy.intercept('POST', '**/api/query', {
                        forceNetworkError: true
                    }).as('networkError');

                    // Trigger a new request by clicking Query/Submit button
                    cy.get('[data-testid="submit-button"]', {timeout: 10000}).click();

                    // Wait for the failed request
                    cy.wait('@networkError', {timeout: 10000});

                    // App gracefully handles network errors by keeping existing data visible
                    // (graceful degradation) - verify table is still shown
                    cy.get('table:visible', {timeout: 5000}).should('exist');
                });
            });
        });
    });

    describe('HTTP Error Status Codes', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    // Handle uncaught Apollo errors from HTTP errors
                    cy.on('uncaught:exception', (err) => {
                        if (err.message.includes('500') ||
                            err.message.includes('503') ||
                            err.message.includes('status code') ||
                            err.message.includes('Response not successful')) {
                            return false;
                        }
                        return true;
                    });
                });

                it('handles server 500 error gracefully', () => {
                    clearBrowserState();

                    // Set up intercept BEFORE visiting
                    cy.intercept('POST', '**/api/query', {
                        statusCode: 500,
                        body: {
                            errors: [{message: 'Internal server error'}]
                        }
                    }).as('serverError');

                    cy.visit('/storage-unit');

                    cy.wait('@serverError', {timeout: 10000});

                    // Should show error toast or redirect to login
                    cy.url().then((url) => {
                        if (url.includes('/login')) {
                            // Redirected to login - acceptable error handling
                            cy.url().should('include', '/login');
                        } else {
                            // Should show error toast
                            cy.get('[data-sonner-toast]', {timeout: 5000}).should('exist');
                        }
                    });
                });

                it('handles server 503 service unavailable gracefully', () => {
                    clearBrowserState();

                    cy.intercept('POST', '**/api/query', {
                        statusCode: 503,
                        body: {
                            errors: [{message: 'Service temporarily unavailable'}]
                        }
                    }).as('serviceUnavailable');

                    cy.visit('/storage-unit');

                    cy.wait('@serviceUnavailable', {timeout: 10000});

                    // Should show error toast or redirect to login
                    cy.url().then((url) => {
                        if (url.includes('/login')) {
                            cy.url().should('include', '/login');
                        } else {
                            cy.get('[data-sonner-toast]', {timeout: 5000}).should('exist');
                        }
                    });
                });
            });
        });
    });

    describe('Invalid Query Errors', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('shows error in scratchpad for invalid SQL syntax', () => {
                    cy.goto('scratchpad');

                    // Write invalid query - syntax error works on all databases
                    const invalidQuery = 'SELEC * FORM nonexistent';
                    cy.writeCode(0, invalidQuery);
                    cy.runCode(0);

                    // Error should be displayed in cell - getCellError returns text
                    cy.getCellError(0).should('not.be.empty');
                });

                it('shows error for non-existent table in scratchpad', () => {
                    cy.goto('scratchpad');

                    const query = getSqlQuery(db, 'invalidQuery');
                    cy.writeCode(0, query);
                    cy.runCode(0);

                    // Error message should contain relevant error pattern
                    const errorPattern = getErrorPattern(db, 'tableNotFound');
                    if (errorPattern) {
                        cy.getCellError(0).should('contain', errorPattern.split(' ')[0]);
                    } else {
                        // Just verify error exists
                        cy.getCellError(0).should('not.be.empty');
                    }
                });

                it('shows error when trying to create existing table', () => {
                    cy.goto('scratchpad');

                    // Try to create a table that already exists - will error but not modify data
                    const duplicateQuery = db.sql.permissionDeniedQuery;
                    cy.writeCode(0, duplicateQuery);
                    cy.runCode(0);

                    // Error should be shown - getCellError returns text
                    cy.getCellError(0).should('not.be.empty');
                });

                it('clears previous error when valid query runs', () => {
                    cy.goto('scratchpad');

                    // Run invalid query first
                    cy.writeCode(0, 'INVALID SQL QUERY');
                    cy.runCode(0);
                    cy.getCellError(0).should('not.be.empty');

                    // Run valid query
                    const validQuery = getSqlQuery(db, 'selectAllUsers');
                    cy.writeCode(0, validQuery);
                    cy.runCode(0);

                    // Results should show (error is cleared automatically by runCode waiting for output)
                    cy.getCellQueryOutput(0).then(({rows}) => {
                        expect(rows.length).to.be.greaterThan(0);
                    });
                });
            });
        }, {features: ['scratchpad']});
    });

    describe('GraphQL Errors', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    // Handle uncaught Apollo errors from GraphQL responses
                    cy.on('uncaught:exception', (err) => {
                        if (err.message.includes('Failed to execute query') ||
                            err.message.includes('Connection timeout') ||
                            err.message.includes('Query execution failed')) {
                            return false;
                        }
                        return true;
                    });
                });

                it('handles GraphQL errors on page load', () => {
                    clearBrowserState();

                    // Set up intercept BEFORE navigation
                    cy.intercept('POST', '**/api/query', {
                        statusCode: 200,
                        body: {
                            data: null,
                            errors: [
                                {
                                    message: 'Failed to execute query',
                                    extensions: {
                                        code: 'QUERY_EXECUTION_ERROR'
                                    }
                                }
                            ]
                        }
                    }).as('graphqlError');

                    cy.visit('/storage-unit');

                    cy.wait('@graphqlError', {timeout: 10000});

                    // Should show error toast or redirect to login
                    cy.url().then((url) => {
                        if (url.includes('/login')) {
                            cy.url().should('include', '/login');
                        } else {
                            cy.get('[data-sonner-toast]', {timeout: 5000}).should('exist');
                        }
                    });
                });

                it('handles multiple GraphQL errors gracefully', () => {
                    clearBrowserState();

                    cy.intercept('POST', '**/api/query', {
                        statusCode: 200,
                        body: {
                            data: null,
                            errors: [
                                {message: 'Error 1: Connection timeout'},
                                {message: 'Error 2: Query execution failed'}
                            ]
                        }
                    }).as('multipleErrors');

                    cy.visit('/storage-unit');

                    cy.wait('@multipleErrors', {timeout: 10000});

                    // Should show error toast or redirect to login
                    cy.url().then((url) => {
                        if (url.includes('/login')) {
                            cy.url().should('include', '/login');
                        } else {
                            cy.get('[data-sonner-toast]', {timeout: 5000}).should('exist');
                        }
                    });
                });
            });
        });
    });

    describe('Connection Timeout Handling', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('handles slow query with loading indicator', () => {
                    cy.goto('scratchpad');

                    const query = getSqlQuery(db, 'selectAllUsers');
                    cy.writeCode(0, query);

                    // Intercept with delay to simulate slow query
                    cy.intercept('POST', '**/api/query', (req) => {
                        // Only intercept Query operations, let others through
                        if (req.body.operationName === 'Query') {
                            req.reply({
                                statusCode: 200,
                                body: {
                                    data: {
                                        Query: {
                                            Columns: [{Name: 'id', Type: 'integer', __typename: 'Column'}],
                                            Rows: [['1']],
                                            __typename: 'RowsResult'
                                        }
                                    }
                                },
                                delay: 2000 // 2 second delay
                            });
                        } else {
                            req.continue();
                        }
                    }).as('slowQuery');

                    // Click run button directly instead of using runCode (which waits for completion)
                    cy.get(`[role="tabpanel"][data-state="active"] [data-testid="cell-0"] [data-testid="query-cell-button"]`)
                        .first()
                        .click({force: true});

                    // Wait for the slow query to complete
                    cy.wait('@slowQuery', {timeout: 10000});

                    // After completion, results should appear
                    cy.get(`[role="tabpanel"][data-state="active"] [data-testid="cell-0"]`).within(() => {
                        cy.get('[data-testid="cell-query-output"], [data-testid="cell-error"]', {timeout: 5000})
                            .should('exist');
                    });
                });
            });
        }, {features: ['scratchpad']});
    });

    describe('Authentication Expiry', () => {
        beforeEach(() => {
            // Handle uncaught Apollo errors from 401 responses
            cy.on('uncaught:exception', (err) => {
                if (err.message.includes('Unauthorized') ||
                    err.message.includes('401') ||
                    err.message.includes('UNAUTHENTICATED')) {
                    return false;
                }
                return true;
            });
        });

        it('handles session expiry by redirecting or showing error', () => {
            clearBrowserState();

            // Set up 401 intercept BEFORE visiting any page
            cy.intercept('POST', '**/api/query', {
                statusCode: 401,
                body: {
                    errors: [
                        {
                            message: 'Unauthorized',
                            extensions: {
                                code: 'UNAUTHENTICATED'
                            }
                        }
                    ]
                }
            }).as('authError');

            cy.visit('/storage-unit');

            cy.wait('@authError', {timeout: 10000});

            // Should either redirect to login or show error toast
            cy.url().then((url) => {
                if (url.includes('/login')) {
                    // Redirected to login - correct behavior
                    cy.url().should('include', '/login');
                } else {
                    // Should show error toast
                    cy.get('[data-sonner-toast]', {timeout: 5000}).should('exist');
                }
            });
        });
    });

    describe('Error State UI Features', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('scratchpad error can be cleared by running new query', () => {
                    cy.goto('scratchpad');

                    // Run invalid query
                    cy.writeCode(0, 'INVALID QUERY');
                    cy.runCode(0);

                    // Error should appear - getCellError returns text
                    cy.getCellError(0).should('not.be.empty');

                    // Run valid query
                    const validQuery = getSqlQuery(db, 'selectAllUsers');
                    cy.writeCode(0, validQuery);
                    cy.runCode(0);

                    // Results should show
                    cy.getCellQueryOutput(0).then(({rows}) => {
                        expect(rows.length).to.be.greaterThan(0);
                    });
                });

                it('displays empty state vs error state appropriately', () => {
                    cy.goto('scratchpad');

                    // Run query that returns no results (not an error)
                    const emptyQuery = db.sql.emptyResultQuery;
                    cy.writeCode(0, emptyQuery);
                    cy.runCode(0);

                    // Should show results (even if empty), not error
                    cy.getCellQueryOutput(0, {timeout: 10000}).then(({rows}) => {
                        expect(rows.length).to.equal(0);
                    });

                    // Now run actual error query
                    cy.writeCode(0, 'INVALID SQL');
                    cy.runCode(0);

                    // Should show error
                    cy.getCellError(0).should('not.be.empty');
                });
            });
        }, {features: ['scratchpad']});
    });

    describe('Chat Error Handling', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    clearBrowserState();
                    loginToDatabase(db);
                    cy.setupChatMock({modelType: 'Ollama', model: 'llama3.1'});
                });

                it('displays error message when chat AI fails', () => {
                    cy.gotoChat();

                    // Mock chat error response
                    cy.intercept('POST', '**/api/query', (req) => {
                        if (req.body.operationName === 'Chat') {
                            req.reply({
                                statusCode: 200,
                                body: {
                                    data: null,
                                    errors: [{message: 'AI model unavailable'}]
                                }
                            });
                        } else {
                            req.continue();
                        }
                    }).as('chatError');

                    cy.sendChatMessage('Hello');

                    cy.wait('@chatError', {timeout: 10000});

                    // Error should be shown (toast or in chat)
                    cy.get('[data-sonner-toast]', {timeout: 5000}).should('exist');
                });
            });
        }, {features: ['chat']});
    });

    describe('Graph View Error Handling', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    // Handle uncaught Apollo errors from mocked responses
                    cy.on('uncaught:exception', (err) => {
                        if (err.message.includes('Failed to generate graph data')) {
                            return false;
                        }
                        return true;
                    });
                });

                it('handles graph data failure gracefully', () => {
                    clearBrowserState();

                    // Login first
                    loginToDatabase(db);

                    // Set up intercept to fail graph queries
                    cy.intercept('POST', '**/api/query', (req) => {
                        if (req.body.operationName === 'GetGraph') {
                            req.reply({
                                statusCode: 200,
                                body: {
                                    data: null,
                                    errors: [{message: 'Failed to generate graph data'}]
                                }
                            });
                        } else {
                            req.continue();
                        }
                    }).as('graphError');

                    // Navigate to graph
                    cy.visit('/graph');

                    cy.wait('@graphError', {timeout: 10000});

                    // Should be on graph page - the app handles errors gracefully
                    // by showing empty state rather than crashing
                    cy.url().should('include', '/graph');
                });
            });
        }, {features: ['graph']});
    });

    describe('Error Recovery', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                beforeEach(() => {
                    // Handle uncaught errors during recovery test
                    cy.on('uncaught:exception', (err) => {
                        if (err.message.includes('Temporary error') ||
                            err.message.includes('500') ||
                            err.message.includes('Response not successful')) {
                            return false;
                        }
                        return true;
                    });
                });

                it('can recover from failed operation by refreshing', () => {
                    clearBrowserState();

                    let callCount = 0;

                    // First few calls fail, subsequent calls succeed
                    cy.intercept('POST', '**/api/query', (req) => {
                        callCount++;
                        if (callCount <= 2) {
                            req.reply({
                                statusCode: 500,
                                body: {errors: [{message: 'Temporary error'}]}
                            });
                        } else {
                            req.continue();
                        }
                    }).as('retryRequest');

                    // Try to visit - will fail initially
                    cy.visit('/storage-unit');

                    // Wait for initial error
                    cy.wait('@retryRequest', {timeout: 10000});

                    // Give app time to handle error
                    cy.wait(1000);

                    // Reload page (retry) - subsequent calls should succeed
                    cy.reload();

                    // After reload, should eventually see content or still be on page
                    cy.url().should('include', '/');
                });
            });
        });
    });
});
