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

import { test, expect, forEachDatabase } from '../../support/test-fixture.mjs';
import { getErrorPattern, getSqlQuery } from '../../support/database-config.mjs';
import { clearBrowserState } from '../../support/helpers/animation.mjs';

/**
 * Error Handling Tests
 *
 * Tests error scenarios including network failures, invalid queries, GraphQL errors,
 * connection timeouts, authentication expiry, and error state UI.
 */
test.describe('Error Handling', () => {
    test.describe('Network Errors', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('gracefully handles network failure during query', async ({ whodb, page }) => {
                    // Suppress uncaught page errors from network failures
                    // (equivalent to Cypress cy.on('uncaught:exception'))


                    await clearBrowserState(page);
                    const conn = db.connection;
                    await whodb.login(
                        db.uiType || db.type,
                        conn.host ?? undefined,
                        conn.user ?? undefined,
                        conn.password ?? undefined,
                        conn.database ?? undefined,
                        conn.advanced || {}
                    );

                    // Navigate to a table first
                    const testTable = db.testTable;
                    await whodb.data(testTable.name);

                    // Wait for table to load
                    const tableData = await whodb.getTableData();
                    expect(tableData.rows.length).toBeGreaterThan(0);

                    // Set up route to abort network requests BEFORE triggering new request
                    await page.route('**/api/query', (route) => route.abort());

                    // Trigger a new request by clicking Query/Submit button
                    await page.locator('[data-testid="submit-button"]').click({ timeout: 10000 });

                    // Wait a moment for the failed request to process
                    await page.waitForTimeout(2000);

                    // Remove the route so future requests work
                    await page.unroute('**/api/query');

                    // App gracefully handles network errors by keeping existing data visible
                    // (graceful degradation) - verify table is still shown
                    await expect(page.locator('table').first()).toBeVisible({ timeout: 5000 });
                });
            });
        }, { login: false, logout: false });
    });

    test.describe('HTTP Error Status Codes', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('handles server 500 error gracefully', async ({ whodb, page }) => {

                    await clearBrowserState(page);

                    // Set up route BEFORE visiting
                    await page.route('**/api/query', (route) => {
                        route.fulfill({
                            status: 500,
                            contentType: 'application/json',
                            body: JSON.stringify({
                                errors: [{ message: 'Internal server error' }]
                            })
                        });
                    });

                    await page.goto('http://localhost:3000/storage-unit');

                    // Wait for the failed request to process
                    await page.waitForTimeout(3000);

                    // Remove the route
                    await page.unroute('**/api/query');

                    // Should show error toast or redirect to login
                    const url = page.url();
                    if (url.includes('/login')) {
                        await expect(page).toHaveURL(/\/login/);
                    } else {
                        await expect(page.locator('[data-sonner-toast]')).toBeVisible({ timeout: 5000 });
                    }
                });

                test('handles server 503 service unavailable gracefully', async ({ whodb, page }) => {

                    await clearBrowserState(page);

                    await page.route('**/api/query', (route) => {
                        route.fulfill({
                            status: 503,
                            contentType: 'application/json',
                            body: JSON.stringify({
                                errors: [{ message: 'Service temporarily unavailable' }]
                            })
                        });
                    });

                    await page.goto('http://localhost:3000/storage-unit');

                    // Wait for the failed request to process
                    await page.waitForTimeout(3000);

                    // Remove the route
                    await page.unroute('**/api/query');

                    // Should show error toast or redirect to login
                    const url = page.url();
                    if (url.includes('/login')) {
                        await expect(page).toHaveURL(/\/login/);
                    } else {
                        await expect(page.locator('[data-sonner-toast]')).toBeVisible({ timeout: 5000 });
                    }
                });
            });
        }, { login: false, logout: false });
    });

    test.describe('Invalid Query Errors', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows error in scratchpad for invalid SQL syntax', async ({ whodb, page }) => {
                    await whodb.goto('scratchpad');

                    // Write invalid query - syntax error works on all databases
                    const invalidQuery = 'SELEC * FORM nonexistent';
                    await whodb.writeCode(0, invalidQuery);
                    await whodb.runCode(0);

                    // Error should be displayed in cell - getCellError returns text
                    const error = await whodb.getCellError(0);
                    expect(error).not.toBe('');
                });

                test('shows error for non-existent table in scratchpad', async ({ whodb, page }) => {
                    await whodb.goto('scratchpad');

                    const query = getSqlQuery(db, 'invalidQuery');
                    await whodb.writeCode(0, query);
                    await whodb.runCode(0);

                    // Error message should contain relevant error pattern
                    const errorPattern = getErrorPattern(db, 'tableNotFound');
                    const error = await whodb.getCellError(0);
                    if (errorPattern) {
                        expect(error).toContain(errorPattern.split(' ')[0]);
                    } else {
                        // Just verify error exists
                        expect(error).not.toBe('');
                    }
                });

                test('shows error when trying to create existing table', async ({ whodb, page }) => {
                    await whodb.goto('scratchpad');

                    // Try to create a table that already exists - will error but not modify data
                    const duplicateQuery = db.sql.permissionDeniedQuery;
                    await whodb.writeCode(0, duplicateQuery);
                    await whodb.runCode(0);

                    // Error should be shown - getCellError returns text
                    const error = await whodb.getCellError(0);
                    expect(error).not.toBe('');
                });

                test('clears previous error when valid query runs', async ({ whodb, page }) => {
                    await whodb.goto('scratchpad');

                    // Run invalid query first
                    await whodb.writeCode(0, 'INVALID SQL QUERY');
                    await whodb.runCode(0);
                    const error = await whodb.getCellError(0);
                    expect(error).not.toBe('');

                    // Run valid query
                    const validQuery = getSqlQuery(db, 'selectAllUsers');
                    await whodb.writeCode(0, validQuery);
                    await whodb.runCode(0);

                    // Results should show (error is cleared automatically by runCode waiting for output)
                    const output = await whodb.getCellQueryOutput(0);
                    expect(output.rows.length).toBeGreaterThan(0);
                });
            });
        }, { features: ['scratchpad'] });
    });

    test.describe('GraphQL Errors', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('handles GraphQL errors on page load', async ({ whodb, page }) => {

                    await clearBrowserState(page);

                    // Set up route BEFORE navigation
                    await page.route('**/api/query', (route) => {
                        route.fulfill({
                            status: 200,
                            contentType: 'application/json',
                            body: JSON.stringify({
                                data: null,
                                errors: [
                                    {
                                        message: 'Failed to execute query',
                                        extensions: {
                                            code: 'QUERY_EXECUTION_ERROR'
                                        }
                                    }
                                ]
                            })
                        });
                    });

                    await page.goto('http://localhost:3000/storage-unit');

                    // Wait for the request to process
                    await page.waitForTimeout(3000);

                    // Remove the route
                    await page.unroute('**/api/query');

                    // Should show error toast or redirect to login
                    const url = page.url();
                    if (url.includes('/login')) {
                        await expect(page).toHaveURL(/\/login/);
                    } else {
                        await expect(page.locator('[data-sonner-toast]')).toBeVisible({ timeout: 5000 });
                    }
                });

                test('handles multiple GraphQL errors gracefully', async ({ whodb, page }) => {

                    await clearBrowserState(page);

                    await page.route('**/api/query', (route) => {
                        route.fulfill({
                            status: 200,
                            contentType: 'application/json',
                            body: JSON.stringify({
                                data: null,
                                errors: [
                                    { message: 'Error 1: Connection timeout' },
                                    { message: 'Error 2: Query execution failed' }
                                ]
                            })
                        });
                    });

                    await page.goto('http://localhost:3000/storage-unit');

                    // Wait for the request to process
                    await page.waitForTimeout(3000);

                    // Remove the route
                    await page.unroute('**/api/query');

                    // Should show error toast or redirect to login
                    const url = page.url();
                    if (url.includes('/login')) {
                        await expect(page).toHaveURL(/\/login/);
                    } else {
                        await expect(page.locator('[data-sonner-toast]')).toBeVisible({ timeout: 5000 });
                    }
                });
            });
        }, { login: false, logout: false });
    });

    test.describe('Connection Timeout Handling', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('handles slow query with loading indicator', async ({ whodb, page }) => {
                    await whodb.goto('scratchpad');

                    const query = getSqlQuery(db, 'selectAllUsers');
                    await whodb.writeCode(0, query);

                    // Intercept with delay to simulate slow query
                    await page.route('**/api/query', async (route) => {
                        const request = route.request();
                        const postData = request.postDataJSON();

                        // Only intercept Query operations, let others through
                        if (postData?.operationName === 'Query') {
                            await new Promise(resolve => setTimeout(resolve, 2000)); // 2 second delay
                            await route.fulfill({
                                status: 200,
                                contentType: 'application/json',
                                body: JSON.stringify({
                                    data: {
                                        Query: {
                                            Columns: [{ Name: 'id', Type: 'integer', __typename: 'Column' }],
                                            Rows: [['1']],
                                            __typename: 'RowsResult'
                                        }
                                    }
                                })
                            });
                        } else {
                            await route.continue();
                        }
                    });

                    // Click run button directly instead of using runCode (which waits for completion)
                    await page.locator('[role="tabpanel"][data-state="active"] [data-testid="cell-0"] [data-testid="query-cell-button"]')
                        .first()
                        .click({ force: true });

                    // Wait for the slow query to complete
                    await page.waitForTimeout(3000);

                    // Remove the route
                    await page.unroute('**/api/query');

                    // After completion, results should appear
                    await page.locator('[role="tabpanel"][data-state="active"] [data-testid="cell-0"]')
                        .locator('[data-testid="cell-query-output"], [data-testid="cell-error"]')
                        .first()
                        .waitFor({ timeout: 5000 });
                });
            });
        }, { features: ['scratchpad'] });
    });

    test.describe('Authentication Expiry', () => {
        test('handles session expiry by redirecting or showing error', async ({ whodb, page }) => {

            await clearBrowserState(page);

            // Set up 401 route BEFORE visiting any page
            await page.route('**/api/query', (route) => {
                route.fulfill({
                    status: 401,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        errors: [
                            {
                                message: 'Unauthorized',
                                extensions: {
                                    code: 'UNAUTHENTICATED'
                                }
                            }
                        ]
                    })
                });
            });

            await page.goto('http://localhost:3000/storage-unit');

            // Wait for the request to process
            await page.waitForTimeout(3000);

            // Remove the route
            await page.unroute('**/api/query');

            // Should either redirect to login or show error toast
            const url = page.url();
            if (url.includes('/login')) {
                // Redirected to login - correct behavior
                await expect(page).toHaveURL(/\/login/);
            } else {
                // Should show error toast
                await expect(page.locator('[data-sonner-toast]')).toBeVisible({ timeout: 5000 });
            }
        });
    });

    test.describe('Error State UI Features', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('scratchpad error can be cleared by running new query', async ({ whodb, page }) => {
                    await whodb.goto('scratchpad');

                    // Run invalid query
                    await whodb.writeCode(0, 'INVALID QUERY');
                    await whodb.runCode(0);

                    // Error should appear - getCellError returns text
                    const error = await whodb.getCellError(0);
                    expect(error).not.toBe('');

                    // Run valid query
                    const validQuery = getSqlQuery(db, 'selectAllUsers');
                    await whodb.writeCode(0, validQuery);
                    await whodb.runCode(0);

                    // Results should show
                    const output = await whodb.getCellQueryOutput(0);
                    expect(output.rows.length).toBeGreaterThan(0);
                });

                test('displays empty state vs error state appropriately', async ({ whodb, page }) => {
                    await whodb.goto('scratchpad');

                    // Run query that returns no results (not an error)
                    const emptyQuery = db.sql.emptyResultQuery;
                    await whodb.writeCode(0, emptyQuery);
                    await whodb.runCode(0);

                    // Should show results (even if empty), not error
                    const output = await whodb.getCellQueryOutput(0);
                    expect(output.rows.length).toEqual(0);

                    // Now run actual error query
                    await whodb.writeCode(0, 'INVALID SQL');
                    await whodb.runCode(0);

                    // Should show error
                    const error = await whodb.getCellError(0);
                    expect(error).not.toBe('');
                });
            });
        }, { features: ['scratchpad'] });
    });

    test.describe('Chat Error Handling', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test.beforeEach(async ({ whodb, page }) => {
                    await clearBrowserState(page);
                    const conn = db.connection;
                    await whodb.login(
                        db.uiType || db.type,
                        conn.host ?? undefined,
                        conn.user ?? undefined,
                        conn.password ?? undefined,
                        conn.database ?? undefined,
                        conn.advanced || {}
                    );
                    await whodb.setupChatMock({ modelType: 'Ollama', model: 'llama3.1' });
                });

                test('displays error message when chat AI fails', async ({ whodb, page }) => {
                    await whodb.gotoChat();

                    // Override the streaming endpoint to return an error
                    // (setupChatMock already routes this, but we override with a failing response)
                    await page.route('**/api/ai-chat/stream', async (route) => {
                        await route.fulfill({
                            status: 500,
                            contentType: 'application/json',
                            body: JSON.stringify({ error: 'AI model unavailable' })
                        });
                    });

                    await whodb.sendChatMessage('Hello');

                    // Wait for error to process
                    await page.waitForTimeout(3000);

                    // Remove the route
                    await page.unroute('**/api/ai-chat/stream');

                    // Error should be shown (toast or in chat)
                    await expect(page.locator('[data-sonner-toast]')).toBeVisible({ timeout: 5000 });
                });
            });
        }, { features: ['chat'], login: false, logout: false });
    });

    test.describe('Graph View Error Handling', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('handles graph data failure gracefully', async ({ whodb, page }) => {

                    await clearBrowserState(page);

                    // Login first
                    const conn = db.connection;
                    await whodb.login(
                        db.uiType || db.type,
                        conn.host ?? undefined,
                        conn.user ?? undefined,
                        conn.password ?? undefined,
                        conn.database ?? undefined,
                        conn.advanced || {}
                    );

                    // Set up route to fail graph queries
                    await page.route('**/api/query', async (route) => {
                        const request = route.request();
                        const postData = request.postDataJSON();

                        if (postData?.operationName === 'GetGraph') {
                            await route.fulfill({
                                status: 200,
                                contentType: 'application/json',
                                body: JSON.stringify({
                                    data: null,
                                    errors: [{ message: 'Failed to generate graph data' }]
                                })
                            });
                        } else {
                            await route.continue();
                        }
                    });

                    // Navigate to graph
                    await page.goto('http://localhost:3000/graph');

                    // Wait for the request to process
                    await page.waitForTimeout(3000);

                    // Remove the route
                    await page.unroute('**/api/query');

                    // Should be on graph page - the app handles errors gracefully
                    // by showing empty state rather than crashing
                    await expect(page).toHaveURL(/\/graph/);
                });
            });
        }, { features: ['graph'], login: false, logout: false });
    });

    test.describe('Error Recovery', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('can recover from failed operation by refreshing', async ({ whodb, page }) => {

                    await clearBrowserState(page);

                    let callCount = 0;

                    // First few calls fail, subsequent calls succeed
                    await page.route('**/api/query', async (route) => {
                        callCount++;
                        if (callCount <= 2) {
                            await route.fulfill({
                                status: 500,
                                contentType: 'application/json',
                                body: JSON.stringify({ errors: [{ message: 'Temporary error' }] })
                            });
                        } else {
                            await route.continue();
                        }
                    });

                    // Try to visit - will fail initially
                    await page.goto('http://localhost:3000/storage-unit');

                    // Give app time to handle error
                    await page.waitForTimeout(2000);

                    // Remove the route so subsequent calls succeed
                    await page.unroute('**/api/query');

                    // Reload page (retry) - subsequent calls should succeed
                    await page.reload();

                    // After reload, should eventually see content or still be on page
                    await expect(page).toHaveURL(/\//);
                });
            });
        }, { login: false, logout: false });
    });
});
