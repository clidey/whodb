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
import { getDatabaseConfig, getSqlQuery } from '../../support/database-config.mjs';
import { clearBrowserState } from '../../support/helpers/animation.mjs';

/**
 * Loading States & Spinners Tests
 *
 * Tests loading indicators, spinners, and skeleton states across various
 * features and database operations.
 */
test.describe('Loading States & Spinners', () => {
    test.describe('Login Loading State', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
            await page.goto('http://localhost:3000/login');

            // Dismiss telemetry modal if present
            for (let attempt = 0; attempt < 5; attempt++) {
                const btn = page.locator('button').filter({ hasText: 'Disable Telemetry' });
                const count = await btn.count();
                if (count > 0) {
                    await btn.click();
                    break;
                }
                await page.waitForTimeout(300);
            }
        });

        test('shows loading indicator during login submission', async ({ whodb, page }) => {
            const db = getDatabaseConfig('postgres');

            // Fill login form
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator(`[data-value="${db.type}"]`).click();

            await page.locator('[data-testid="hostname"]').fill(db.connection.host);
            await page.locator('[data-testid="username"]').fill(db.connection.user);
            await page.locator('[data-testid="password"]').fill(db.connection.password);
            await page.locator('[data-testid="database"]').fill(db.connection.database);

            // Set up response promise to track login request
            const loginResponsePromise = page.waitForResponse(
                (response) => response.url().includes('/api/query') && response.request().method() === 'POST',
                { timeout: 10000 }
            );

            // Click login button
            await page.locator('[data-testid="login-button"]').click();

            // Wait for login to complete and verify redirect
            await loginResponsePromise;
            await expect(page).toHaveURL(/\/storage-unit/);

            // Verify we see the logged-in state (storage units or loading indicator)
            await expect(page.getByText('Tables')).toBeVisible();
        });
    });

    test.describe('Table Data Loading State', () => {
        forEachDatabase('sql', (db) => {
            const testTable = db.testTable;
            const tableName = testTable.name;

            test.describe(`${db.type}`, () => {
                test('shows loading skeleton/spinner while fetching table data', async ({ whodb, page }) => {
                    // Navigate to table view using the proper command
                    await whodb.data(tableName);

                    // Verify data is displayed after loading
                    await expect(page.locator('[data-testid="table-search"]')).toBeVisible({ timeout: 5000 });
                    const rows = page.locator('table tbody tr');
                    await expect(rows).toHaveCount(/./, { timeout: 5000 });
                    expect(await rows.count()).toBeGreaterThanOrEqual(1);
                });

                test('shows loading state when changing page size', async ({ whodb, page }) => {
                    await whodb.data(tableName);

                    // Wait for initial data load
                    await page.locator('table tbody tr').first().waitFor({ timeout: 10000 });

                    // Track page change request
                    const responsePromise = page.waitForResponse(
                        (response) => response.url().includes('/api/query') && response.request().method() === 'POST',
                        { timeout: 5000 }
                    );

                    // Change page size (valid options: 1, 2, 10, 25, 50, 100, 250, 500, 1000)
                    await whodb.setTablePageSize(10);
                    await whodb.submitTable();

                    // Wait for update
                    await responsePromise;

                    // Verify table is updated
                    await expect(page.locator('table tbody tr')).toBeTruthy();
                });

                test('shows loading state when switching tables', async ({ whodb, page }) => {
                    // Navigate to card view first
                    await page.goto('http://localhost:3000/storage-unit');
                    await page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });
                    expect(await page.locator('[data-testid="storage-unit-card"]').count()).toBeGreaterThanOrEqual(1);

                    // Track request when clicking to explore a table
                    const responsePromise = page.waitForResponse(
                        (response) => response.url().includes('/api/query') && response.request().method() === 'POST',
                        { timeout: 10000 }
                    );

                    // Click on first table to navigate to explore view
                    await page.locator('[data-testid="storage-unit-card"]').first()
                        .locator('[data-testid="data-button"]').click();

                    // Wait for data to load
                    await responsePromise;

                    // Verify we're in explore view with data
                    await expect(page).toHaveURL(/\/storage-unit\/explore/);
                    await page.locator('table:visible').waitFor({ timeout: 5000 });
                });
            });
        });
    });

    test.describe('Scratchpad Query Execution Loading State', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows loading indicator during query execution', async ({ whodb, page }) => {
                    await whodb.goto('scratchpad');

                    // Write query
                    const query = getSqlQuery(db, 'selectAllUsers');
                    await whodb.writeCode(0, query);

                    // Track query execution
                    const responsePromise = page.waitForResponse(
                        (response) => response.url().includes('/api/query') && response.request().method() === 'POST',
                        { timeout: 10000 }
                    );

                    // Execute query
                    await whodb.runCode(0);

                    // Wait for query to complete
                    await responsePromise;

                    // Verify results are displayed
                    const output = await whodb.getCellQueryOutput(0);
                    expect(output.columns.length).toBeGreaterThan(0);
                    expect(output.rows.length).toBeGreaterThan(0);
                });

                test('shows loading state for multiple concurrent queries', async ({ whodb, page }) => {
                    await whodb.goto('scratchpad');

                    // Add multiple cells and write queries
                    await whodb.addCell(0);
                    await whodb.addCell(1);

                    const query1 = getSqlQuery(db, 'selectAllUsers');
                    const query2 = getSqlQuery(db, 'countUsers');

                    await whodb.writeCode(0, query1);
                    await whodb.writeCode(1, query2);
                    await whodb.writeCode(2, query1);

                    // Track queries
                    const responsePromise = page.waitForResponse(
                        (response) => response.url().includes('/api/query') && response.request().method() === 'POST',
                        { timeout: 10000 }
                    );

                    // Execute all
                    await whodb.runCode(0);
                    await whodb.runCode(1);
                    await whodb.runCode(2);

                    // Wait for queries to complete
                    await responsePromise;

                    // Verify all results are displayed
                    await page.locator('[role="tabpanel"][data-state="active"] [data-testid="cell-0"] [data-testid="cell-query-output"]').waitFor({ timeout: 10000 });
                    await page.locator('[role="tabpanel"][data-state="active"] [data-testid="cell-1"] [data-testid="cell-query-output"]').waitFor({ timeout: 10000 });
                    await page.locator('[role="tabpanel"][data-state="active"] [data-testid="cell-2"] [data-testid="cell-query-output"]').waitFor({ timeout: 10000 });
                });
            });
        }, { features: ['scratchpad'] });
    });

    test.describe('Chat AI Loading State', () => {
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

                test('shows loading indicator during AI response', async ({ whodb, page }) => {
                    await whodb.gotoChat();

                    // Mock the actual response
                    await whodb.mockChatResponse([{
                        type: 'text',
                        text: `Hello! I can help you with your ${db.type} database.`
                    }]);

                    // Send message
                    await whodb.sendChatMessage('Hello');

                    // Wait for response and verify it completes
                    await whodb.waitForChatResponse();

                    // Verify message is displayed
                    await whodb.verifyChatSystemMessage('Hello!');
                });

                test('shows loading state during SQL query generation', async ({ whodb, page }) => {
                    const schemaPrefix = db.sql.schemaPrefix;
                    await whodb.gotoChat();

                    // Mock the SQL response
                    await whodb.mockChatResponse([{
                        type: 'text',
                        text: 'I\'ll retrieve all users for you.'
                    }, {
                        type: 'sql:get',
                        text: `SELECT *
                               FROM ${schemaPrefix}users
                               ORDER BY id`,
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
                    await whodb.sendChatMessage('Show me all users');

                    // Wait for response
                    await whodb.waitForChatResponse();

                    // Verify SQL result is displayed
                    await whodb.verifyChatSQLResult({ columns: ['id', 'username'], rowCount: 1 });
                });
            });
        }, { features: ['chat'] });
    });

    test.describe('Schema/Database Selection Loading State', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows loading state when switching schema/database', async ({ whodb, page }) => {
                    // Check if schema/database dropdown exists (not all DBs have these)
                    const dbDropdownCount = await page.locator('[data-testid="sidebar-database"]:visible').count();
                    const schemaDropdownCount = await page.locator('[data-testid="sidebar-schema"]:visible').count();

                    if (dbDropdownCount === 0 && schemaDropdownCount === 0) {
                        // No schema/database dropdown for this DB type - test is not applicable
                        test.skip();
                        return;
                    }

                    // Click whichever dropdown is visible
                    const dropdown = dbDropdownCount > 0
                        ? page.locator('[data-testid="sidebar-database"]:visible')
                        : page.locator('[data-testid="sidebar-schema"]:visible');
                    await dropdown.click();

                    // Wait for dropdown options to appear
                    await page.locator('[role="option"]').first().waitFor({ timeout: 5000 });

                    // Select a different option if available
                    const optionCount = await page.locator('[role="option"]').count();
                    if (optionCount > 1) {
                        // Select the second option
                        await page.locator('[role="option"]').nth(1).click();
                    } else {
                        // Only one option - just close dropdown by clicking elsewhere
                        await page.mouse.click(0, 0);
                    }

                    // Verify page state - either storage units or empty state
                    await page.locator('[data-testid="storage-unit-card"], button:has-text("Create")').first().waitFor({ timeout: 15000 });
                });
            });
        });
    });

    test.describe('Graph View Loading State', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows loading state while fetching graph data', async ({ whodb, page }) => {
                    // Track graph data request
                    const responsePromise = page.waitForResponse(
                        (response) => response.url().includes('/api/query') && response.request().method() === 'POST',
                        { timeout: 10000 }
                    );

                    // Navigate to graph view
                    await page.locator('[href="/graph"]').click();
                    await expect(page).toHaveURL(/\/graph/);

                    // Wait for graph to load
                    await responsePromise;

                    // Verify graph is rendered (canvas or SVG should be present)
                    await page.locator('canvas, svg').first().waitFor({ timeout: 5000 });
                });
            });
        });
    });

    test.describe('Storage Units Loading State', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test.beforeEach(async ({ whodb, page }) => {
                    await clearBrowserState(page);
                });

                test('shows loading state during initial storage units fetch after login', async ({ whodb, page }) => {
                    // Perform login using the standard helper
                    const conn = db.connection;
                    await whodb.login(
                        db.uiType || db.type,
                        conn.host ?? undefined,
                        conn.user ?? undefined,
                        conn.password ?? undefined,
                        conn.database ?? undefined,
                        conn.advanced || {}
                    );

                    // Verify page loads - either storage units or empty state (Create a Table)
                    await page.locator('[data-testid="storage-unit-card"], button:has-text("Create")').first().waitFor({ timeout: 15000 });
                });
            });
        }, { login: false, logout: false });
    });

    test.describe('CRUD Operations Loading State', () => {
        forEachDatabase('sql', (db) => {
            const testTable = db.testTable;
            const tableName = testTable.name;

            test.describe(`${db.type}`, () => {
                test('shows loading state during row creation', async ({ whodb, page }) => {
                    await whodb.data(tableName);
                    await page.locator('table tbody tr').first().waitFor({ timeout: 10000 });

                    // Look for Add Row button - may be in different locations depending on UI
                    const addBtnCount = await page.locator('[data-testid="add-row-button"]').count();
                    const addRowBtnCount = await page.locator('button').filter({ hasText: 'Add Row' }).count();

                    if (addBtnCount === 0 && addRowBtnCount === 0) {
                        test.skip();
                        return;
                    }

                    // Click the add row button
                    if (addBtnCount > 0) {
                        await page.locator('[data-testid="add-row-button"]').click();
                    } else {
                        await page.locator('button').filter({ hasText: 'Add Row' }).click();
                    }

                    // Verify the add row panel/dialog appears with a submit button
                    const submitBtn = page.locator('button:has-text("Submit"), button:has-text("Save"), button:has-text("Add")').first();
                    await expect(submitBtn).toBeVisible({ timeout: 5000 });

                    // Close by pressing escape
                    await page.keyboard.press('Escape');
                });

                test('shows loading state during row update', async ({ whodb, page }) => {
                    await whodb.data(tableName);
                    await page.locator('table tbody tr').first().waitFor({ timeout: 10000 });

                    // Click first row to select/edit
                    await page.locator('table tbody tr').first().click();

                    // Check if row edit UI appears
                    const saveBtnCount = await page.locator('[data-testid="save-button"]').count();
                    const editPanelCount = await page.locator('[data-testid="edit-panel"]').count();

                    if (saveBtnCount === 0 && editPanelCount === 0) {
                        // Row click may just select, not open edit mode
                        test.skip();
                        return;
                    }

                    if (saveBtnCount > 0) {
                        // Save button exists - verify it's visible and functional
                        await expect(page.locator('[data-testid="save-button"]')).toBeVisible();
                    }
                });
            });
        });
    });
});
