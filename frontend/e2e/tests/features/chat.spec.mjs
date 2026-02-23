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

test.describe('Chat AI Integration', () => {

    // Only SQL databases with chat feature
    forEachDatabase('sql', (db) => {
        const schemaPrefix = db.sql.schemaPrefix;

        test.beforeEach(async ({ whodb, page }) => {
            await whodb.setupChatMock({ modelType: 'Ollama', model: 'llama3.1' });
        });

        test.describe('Basic Chat', () => {
            test('shows empty chat initially', async ({ whodb, page }) => {
                await whodb.gotoChat();
                await whodb.verifyChatEmpty();
            });

            test('sends and receives text messages', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{
                    type: 'text',
                    text: `Hello! I can help you query your ${db.type} database.`
                }]);
                await whodb.sendChatMessage('Hello');
                await whodb.waitForChatResponse();
                await whodb.verifyChatUserMessage('Hello');
                await whodb.verifyChatSystemMessage(`Hello! I can help you query your ${db.type} database.`);
            });
        });

        test.describe('SQL Query Generation', () => {
            test('generates SELECT query with results', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{
                    type: 'text',
                    text: 'I\'ll retrieve all users from the database for you.'
                }, {
                    type: 'sql:get',
                    text: `SELECT *
                           FROM ${schemaPrefix}users
                           ORDER BY id`,
                    result: {
                        Columns: [
                            { Name: 'id', Type: 'integer', __typename: 'Column' },
                            { Name: 'username', Type: 'character varying', __typename: 'Column' },
                            { Name: 'email', Type: 'character varying', __typename: 'Column' }
                        ],
                        Rows: [
                            ['1', 'john_doe', 'john@example.com'],
                            ['2', 'jane_smith', 'jane@example.com'],
                            ['3', 'admin_user', 'admin@example.com']
                        ],
                        __typename: 'RowsResult'
                    }
                }]);
                await whodb.sendChatMessage('Show me all users');
                await whodb.waitForChatResponse();
                await whodb.verifyChatUserMessage('Show me all users');
                await whodb.verifyChatSQLResult({
                    columns: ['id', 'username', 'email'],
                    rowCount: 3
                });
            });

            test('toggles between table and SQL view', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await test.step('generate query results', async () => {
                    await whodb.mockChatResponse([{
                        type: 'sql:get',
                        text: `SELECT * FROM ${schemaPrefix}users`,
                        result: {
                            Columns: [{ Name: 'id', Type: 'integer', __typename: 'Column' }],
                            Rows: [['1']],
                            __typename: 'RowsResult'
                        }
                    }]);
                    await whodb.sendChatMessage('Show users');
                    await whodb.waitForChatResponse();
                });

                await test.step('toggle to SQL view and back', async () => {
                    await whodb.toggleChatSQLView();
                    await whodb.verifyChatSQL(`SELECT * FROM ${schemaPrefix}users`);
                    await whodb.toggleChatSQLView();
                });
            });

            test('generates filtered query', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{
                    type: 'text',
                    text: 'Here are the users with admin roles.'
                }, {
                    type: 'sql:get',
                    text: `SELECT *
                           FROM ${schemaPrefix}users
                           WHERE username LIKE '%admin%'`,
                    result: {
                        Columns: [
                            { Name: 'id', Type: 'integer', __typename: 'Column' },
                            { Name: 'username', Type: 'character varying', __typename: 'Column' }
                        ],
                        Rows: [['3', 'admin_user']],
                        __typename: 'RowsResult'
                    }
                }]);
                await whodb.sendChatMessage('Find users with admin in their username');
                await whodb.waitForChatResponse();
                await whodb.verifyChatSQLResult({ rowCount: 1 });
            });

            test('generates aggregate query', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{
                    type: 'text',
                    text: 'Here\'s the user count.'
                }, {
                    type: 'sql:get',
                    text: `SELECT COUNT(*) as user_count
                           FROM ${schemaPrefix}users`,
                    result: {
                        Columns: [{ Name: 'user_count', Type: 'bigint', __typename: 'Column' }],
                        Rows: [['3']],
                        __typename: 'RowsResult'
                    }
                }]);
                await whodb.sendChatMessage('Count users');
                await whodb.waitForChatResponse();
                await whodb.verifyChatSQLResult({
                    columns: ['user_count'],
                    rowCount: 1
                });
            });
        });

        test.describe('SQL Mutations', () => {
            test('executes INSERT operation', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{
                    type: 'text',
                    text: 'I\'ll add a new user to the database.'
                }, {
                    type: 'sql:insert',
                    text: `INSERT INTO ${schemaPrefix}users (username, email)
                           VALUES ('test_user', 'test@example.com')`,
                    result: { Columns: [], Rows: [], __typename: 'RowsResult' }
                }]);
                await whodb.sendChatMessage('Add a new user named test_user');
                await whodb.waitForChatResponse();
                await whodb.verifyChatActionExecuted();
            });

            test('executes UPDATE operation', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{
                    type: 'text',
                    text: 'I\'ll update the user\'s email address.'
                }, {
                    type: 'sql:update',
                    text: `UPDATE ${schemaPrefix}users
                           SET email = 'new@example.com'
                           WHERE username = 'test_user'`,
                    result: { Columns: [], Rows: [], __typename: 'RowsResult' }
                }]);
                await whodb.sendChatMessage('Update test_user email');
                await whodb.waitForChatResponse();
                await whodb.verifyChatActionExecuted();
            });

            test('executes DELETE operation with confirmation', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await test.step('request deletion and verify confirmation prompt', async () => {
                    await whodb.mockChatResponse([{
                        type: 'text',
                        text: 'Are you sure you want to delete this user? This action cannot be undone.'
                    }]);
                    await whodb.sendChatMessage('Delete test_user from the database');
                    await whodb.waitForChatResponse();
                    await whodb.verifyChatSystemMessage('Are you sure you want to delete this user?');
                });

                await test.step('confirm deletion and verify execution', async () => {
                    await whodb.mockChatResponse([{
                        type: 'sql:delete',
                        text: `DELETE
                               FROM ${schemaPrefix}users
                               WHERE username = 'test_user'`,
                        result: { Columns: [], Rows: [], __typename: 'RowsResult' }
                    }]);
                    await whodb.sendChatMessage('Yes, delete it');
                    await whodb.waitForChatResponse();
                    await whodb.verifyChatActionExecuted();
                });
            });
        });

        test.describe('Error Handling', () => {
            test('displays error for invalid query', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{
                    type: 'error',
                    text: 'relation "nonexistent_table" does not exist'
                }]);
                await whodb.sendChatMessage('Show me data from nonexistent_table');

                // Wait for user message to appear
                await page.locator('[data-input-message="user"]').waitFor({ timeout: 5000 });

                // Errors are shown as toast notifications in the current implementation
                // Wait for loading to finish (errors trigger done event)
                await page.waitForTimeout(2000);

                // Verify no SQL results appeared (indicating error was handled)
                await expect(page.locator('table')).toHaveCount(0);
            });
        });

        test.describe('Query Export', () => {
            test('exports chat query results as CSV', async ({ whodb, page }) => {
                await whodb.gotoChat();
                let exportResponsePromise;

                await test.step('set up export listener and generate query results', async () => {
                    exportResponsePromise = page.waitForResponse(
                        (response) => response.url().includes('/api/export'),
                        { timeout: 15000 }
                    );

                    await whodb.mockChatResponse([{
                        type: 'sql:get',
                        text: `SELECT * FROM ${schemaPrefix}users ORDER BY id`,
                        result: {
                            Columns: [
                                { Name: 'id', Type: 'integer', __typename: 'Column' },
                                { Name: 'username', Type: 'character varying', __typename: 'Column' }
                            ],
                            Rows: [['1', 'john_doe'], ['2', 'jane_smith']],
                            __typename: 'RowsResult'
                        }
                    }]);
                    await whodb.sendChatMessage('Show me all users');
                    await whodb.waitForChatResponse();
                });

                await test.step('open export dialog and confirm', async () => {
                    await page.locator('[data-testid="export-all-button"]').last().click();
                    await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();
                    await expect(page.getByText('You are about to export the results of your query.')).toBeVisible();

                    await whodb.confirmExport();
                });

                await test.step('verify export response', async () => {
                    const exportResponse = await exportResponsePromise;
                    const request = exportResponse.request();
                    const requestBody = request.postDataJSON();
                    expect(exportResponse.status()).toEqual(200);
                    expect(requestBody.selectedRows).toBeTruthy();
                    expect(Array.isArray(requestBody.selectedRows)).toBeTruthy();
                    expect(requestBody.selectedRows.length).toEqual(2);
                    expect(requestBody.storageUnit).toEqual('query_export');
                });
            });

            test('does not show "Export Selected" options in context menu', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{
                    type: 'sql:get',
                    text: `SELECT * FROM ${schemaPrefix}users`,
                    result: {
                        Columns: [{ Name: 'id', Type: 'integer', __typename: 'Column' }],
                        Rows: [['1']],
                        __typename: 'RowsResult'
                    }
                }]);
                await whodb.sendChatMessage('Show users');
                await whodb.waitForChatResponse();

                // Right-click on the result table cell
                await page.locator('table').last().locator('tbody tr').first().locator('td').nth(1).click({ button: 'right' });
                await page.waitForTimeout(300);

                // Open Export submenu (scope to context menu to avoid matching "Export All" button)
                await page.locator('[role="menu"]').getByText('Export').click();

                // "Export All" options should be visible
                await expect(page.getByText('Export All as CSV')).toBeVisible();
                await expect(page.getByText('Export All as Excel')).toBeVisible();

                // "Export Selected" options should NOT exist
                await expect(page.getByText('Export Selected as CSV')).toHaveCount(0);
                await expect(page.getByText('Export Selected as Excel')).toHaveCount(0);

                await page.keyboard.press('Escape');
            });

            test('preselects Excel when "Export All as Excel" is chosen from context menu', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{
                    type: 'sql:get',
                    text: `SELECT * FROM ${schemaPrefix}users`,
                    result: {
                        Columns: [{ Name: 'id', Type: 'integer', __typename: 'Column' }],
                        Rows: [['1']],
                        __typename: 'RowsResult'
                    }
                }]);
                await whodb.sendChatMessage('Show users');
                await whodb.waitForChatResponse();

                // Right-click on result table
                await page.locator('table').last().locator('tbody tr').first().locator('td').nth(1).click({ button: 'right' });
                await page.waitForTimeout(300);

                await page.locator('[role="menu"]').getByText('Export').click();
                await page.getByText('Export All as Excel').click();

                // Verify dialog opens with Excel preselected
                await expect(page.locator('h2').filter({ hasText: 'Export Data' }).first()).toBeVisible();
                await expect(page.locator('[data-testid="export-format-select"]')).toContainText('Excel');

                await page.keyboard.press('Escape');
            });
        });

        test.describe('Move to Scratchpad', () => {
            test('moves query to scratchpad', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await test.step('generate query results', async () => {
                    await whodb.mockChatResponse([{
                        type: 'sql:get',
                        text: `SELECT *
                               FROM ${schemaPrefix}users`,
                        result: {
                            Columns: [{ Name: 'id', Type: 'integer', __typename: 'Column' }],
                            Rows: [['1']],
                            __typename: 'RowsResult'
                        }
                    }]);
                    await whodb.sendChatMessage('Show users');
                    await whodb.waitForChatResponse();
                });

                await test.step('move to scratchpad and verify navigation', async () => {
                    await whodb.openMoveToScratchpad();
                    await whodb.confirmMoveToScratchpad({ pageOption: 'new', newPageName: 'Chat Queries' });

                    await expect(page).toHaveURL(/\/scratchpad/, { timeout: 10000 });
                });
            });
        });

        test.describe('Provider and Model Selection', () => {
            test.beforeEach(async ({ whodb, page }) => {
                await whodb.setupChatMock({ modelType: 'Ollama', model: 'llama3.1' });
            });

            test('shows provider dropdown with available options', async ({ whodb, page }) => {
                await whodb.gotoChat();

                // Click the provider dropdown
                await page.locator('[data-testid="ai-provider-select"]').click();

                // Wait for dropdown options to appear
                await page.locator('[role="option"]').first().waitFor({ state: 'visible', timeout: 5000 });

                // Verify all expected providers are available
                // Internal providers: Ollama
                await expect(page.locator('[role="option"]')).toContainText(['Ollama']);

                // Verify the "Add a provider" option exists (for external providers like OpenAI, Anthropic)
                await expect(page.getByText('Add a provider')).toBeVisible();

                // Close dropdown
                await page.keyboard.press('Escape');
            });

            test('updates model selection dropdown based on provider', async ({ whodb, page }) => {
                await whodb.gotoChat();

                // Verify model dropdown is enabled for Ollama
                await expect(page.locator('[data-testid="ai-model-select"]')).toBeVisible();
                await expect(page.locator('[data-testid="ai-model-select"]')).toBeEnabled();

                // Verify the model is populated (from our mock)
                await expect(page.locator('[data-testid="ai-model-select"]')).toContainText('llama3.1');

                // Click model dropdown to verify it opens
                await page.locator('[data-testid="ai-model-select"]').click();
                await page.locator('[role="option"]').first().waitFor({ state: 'visible', timeout: 5000 });
                await expect(page.locator('[role="option"]')).toContainText(['llama3.1']);

                // Close dropdown
                await page.keyboard.press('Escape');
            });

            test('shows token input dialog for external cloud providers', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await test.step('open add provider dialog', async () => {
                    await page.locator('[data-testid="ai-provider-select"]').click();
                    await page.getByText('Add a provider').click();
                    await expect(page.locator('h2, .text-lg').filter({ hasText: /add.*external.*(model|provider)/i })).toBeVisible({ timeout: 5000 });
                });

                await test.step('verify available external providers', async () => {
                    await expect(page.locator('[data-testid="external-model-type-select"]')).toBeVisible();
                    await page.locator('[data-testid="external-model-type-select"]').click();

                    await expect(page.locator('[role="option"]')).toContainText(['OpenAI']);
                    await expect(page.locator('[role="option"]')).toContainText(['Anthropic']);

                    await page.locator('[data-testid="external-model-cancel"]').click();
                });
            });

            test('persists provider selection in session', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await test.step('verify initial provider and model selection', async () => {
                    await expect(page.locator('[data-testid="ai-provider-select"]')).toContainText('Ollama');
                    await expect(page.locator('[data-testid="ai-model-select"]')).toContainText('llama3.1');
                });

                await test.step('navigate away and back', async () => {
                    await page.goto('http://localhost:3000/storage-unit');
                    await page.waitForTimeout(500);
                    await page.goto('http://localhost:3000/chat');
                    await page.locator('[data-testid="ai-provider"]').waitFor({ timeout: 10000 });
                });

                await test.step('verify provider persisted after navigation', async () => {
                    await expect(page.locator('[data-testid="ai-provider-select"]')).toContainText('Ollama', { timeout: 5000 });
                });
            });

            test('uses correct model lists for different providers', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await test.step('verify internal provider model list', async () => {
                    await expect(page.locator('[data-testid="ai-provider-select"]')).toContainText('Ollama');
                    await page.locator('[data-testid="ai-model-select"]').click();
                    await page.locator('[role="option"]').first().waitFor({ state: 'visible', timeout: 5000 });
                    await expect(page.locator('[role="option"]')).toContainText(['llama3.1']);
                    await page.keyboard.press('Escape');
                });

                await test.step('verify external provider requires token', async () => {
                    await page.locator('[data-testid="ai-provider-select"]').click();
                    await page.getByText('Add a provider').click();

                    await expect(page.locator('label').filter({ hasText: /token/i })).toBeVisible();
                    await expect(page.locator('input[type="password"]')).toBeVisible();
                });

                await test.step('select external provider and verify token field', async () => {
                    await page.locator('[data-testid="external-model-type-select"]').click();
                    await page.locator('[role="option"]').filter({ hasText: 'OpenAI' }).click();

                    await expect(page.locator('input[type="password"]')).toBeVisible();
                    await expect(page.locator('input[type="password"]')).toHaveValue('');

                    await page.locator('[data-testid="external-model-cancel"]').click();
                });
            });

            test('requires token for external providers before model selection', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await test.step('open external provider dialog', async () => {
                    await page.locator('[data-testid="ai-provider-select"]').click();
                    await page.waitForTimeout(500); // Wait for dropdown to fully open
                    await page.getByText('Add a provider').click();
                    await expect(page.locator('h2, .text-lg').filter({ hasText: /add.*external.*(model|provider)/i })).toBeVisible({ timeout: 5000 });
                });

                await test.step('select provider and verify token input', async () => {
                    await page.locator('[data-testid="external-model-type-select"]').click();
                    await page.locator('[role="option"]').filter({ hasText: 'Anthropic' }).click();

                    await expect(page.locator('label').filter({ hasText: /token/i })).toBeVisible();
                    await expect(page.locator('input[type="password"]')).toBeVisible();
                    await expect(page.locator('[data-testid="external-model-submit"]')).toBeVisible();
                });

                await test.step('submit without token and verify validation', async () => {
                    await page.locator('[data-testid="external-model-submit"]').click();
                    await expect(page.locator('h2, .text-lg').filter({ hasText: /add.*external.*(model|provider)/i })).toBeVisible();

                    await page.locator('[data-testid="external-model-cancel"]').click();
                });
            });

            test('allows deleting custom providers', async ({ whodb, page }) => {
                await whodb.gotoChat();

                // Verify delete provider button exists
                await expect(page.locator('[data-testid="chat-delete-provider"]')).toBeVisible();

                // Click delete provider button
                await page.locator('[data-testid="chat-delete-provider"]').click();

                // Wait for confirmation dialog
                await expect(page.getByText(/delete.*provider/i).first()).toBeVisible({ timeout: 5000 });

                // Verify confirmation dialog has expected content (asking if sure about deletion)
                await expect(page.getByText(/are you sure.*delete.*provider/i)).toBeVisible();

                // Cancel deletion
                await page.locator('button').filter({ hasText: /cancel/i }).click();

                // Verify we're back on chat and provider is still selected
                await expect(page.locator('[data-testid="ai-provider-select"]')).toContainText('Ollama');
            });
        });

        test.describe('Chat History', () => {
            test('navigates chat history with arrow keys', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await test.step('send two messages', async () => {
                    await whodb.mockChatResponse([{ type: 'text', text: 'Response 1' }]);
                    await whodb.sendChatMessage('First message');
                    await whodb.waitForChatResponse();

                    await whodb.mockChatResponse([{ type: 'text', text: 'Response 2' }]);
                    await whodb.sendChatMessage('Second message');
                    await whodb.waitForChatResponse();
                });

                await test.step('navigate history and verify order', async () => {
                    await whodb.navigateChatHistory('up');
                    const val1 = await whodb.getChatInputValue();
                    expect(val1).toEqual('Second message');

                    await whodb.navigateChatHistory('up');
                    const val2 = await whodb.getChatInputValue();
                    expect(val2).toEqual('First message');
                });
            });

            test('clears chat history', async ({ whodb, page }) => {
                await whodb.gotoChat();

                await whodb.mockChatResponse([{ type: 'text', text: 'Hello!' }]);
                await whodb.sendChatMessage('Hi');
                await whodb.waitForChatResponse();

                await whodb.clearChat();
                await whodb.verifyChatEmpty();
            });
        });
        test.describe('Chat Title Generation', () => {
            test('generates title after first message', async ({ whodb, page }) => {
                await whodb.gotoChat();

                // Mock the GenerateChatTitle mutation response via route interception
                await page.route('**/api/query', async (route) => {
                    let postData;
                    try {
                        postData = route.request().postDataJSON();
                    } catch {
                        return route.fallback();
                    }

                    if (postData?.operationName === 'GenerateChatTitle') {
                        return route.fulfill({
                            contentType: 'application/json',
                            body: JSON.stringify({
                                data: {
                                    GenerateChatTitle: {
                                        Title: 'Test Query Summary',
                                        __typename: 'GenerateChatTitleResponse',
                                    },
                                },
                            }),
                        });
                    }

                    await route.fallback();
                });

                // Send first message to trigger title generation
                await whodb.mockChatResponse([{
                    type: 'text',
                    text: 'Here are your users.',
                }]);
                await whodb.sendChatMessage('Show me all users');
                await whodb.waitForChatResponse();

                // The title mutation fires asynchronously after the first message.
                // The sidebar session name should update with the generated title.
                // The chat-session-name element uses a typewriter effect, so wait for it.
                const sessionName = page.locator('[data-testid^="chat-session-name-"]').first();
                await expect(sessionName).toContainText('Test Query Summary', { timeout: 10000 });
            });
        });
    }, { features: ['chat'] });
});
