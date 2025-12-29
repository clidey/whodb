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

import {forEachDatabase} from '../../support/test-runner';

describe('Chat AI Integration', () => {

    // Only SQL databases with chat feature
    forEachDatabase('sql', (db) => {
        const schemaPrefix = db.sql.schemaPrefix;

        beforeEach(() => {
            cy.setupChatMock({modelType: 'Ollama', model: 'llama3.1'});
        });

        describe('Basic Chat', () => {
            it('shows empty chat initially', () => {
                cy.gotoChat();
                cy.verifyChatEmpty();
            });

            it('sends and receives text messages', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'text',
                    text: `Hello! I can help you query your ${db.type} database.`
                }]);
                cy.sendChatMessage('Hello');
                cy.waitForChatResponse();
                cy.verifyChatUserMessage('Hello');
                cy.verifyChatSystemMessage(`Hello! I can help you query your ${db.type} database.`);
            });
        });

        describe('SQL Query Generation', () => {
            it('generates SELECT query with results', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'text',
                    text: 'I\'ll retrieve all users from the database for you.'
                }, {
                    type: 'sql:get',
                    text: `SELECT *
                           FROM ${schemaPrefix}users
                           ORDER BY id`,
                    result: {
                        Columns: [
                            {Name: 'id', Type: 'integer', __typename: 'Column'},
                            {Name: 'username', Type: 'character varying', __typename: 'Column'},
                            {Name: 'email', Type: 'character varying', __typename: 'Column'}
                        ],
                        Rows: [
                            ['1', 'john_doe', 'john@example.com'],
                            ['2', 'jane_smith', 'jane@example.com'],
                            ['3', 'admin_user', 'admin@example.com']
                        ],
                        __typename: 'RowsResult'
                    }
                }]);
                cy.sendChatMessage('Show me all users');
                cy.waitForChatResponse();
                cy.verifyChatUserMessage('Show me all users');
                cy.verifyChatSQLResult({
                    columns: ['id', 'username', 'email'],
                    rowCount: 3
                });
            });

            it('toggles between table and SQL view', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'sql:get',
                    text: `SELECT * FROM ${schemaPrefix}users`,
                    result: {
                        Columns: [{Name: 'id', Type: 'integer', __typename: 'Column'}],
                        Rows: [['1']],
                        __typename: 'RowsResult'
                    }
                }]);
                cy.sendChatMessage('Show users');
                cy.waitForChatResponse();

                cy.toggleChatSQLView();
                cy.verifyChatSQL(`SELECT * FROM ${schemaPrefix}users`);
                cy.toggleChatSQLView();
            });

            it('generates filtered query', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'text',
                    text: 'Here are the users with admin roles.'
                }, {
                    type: 'sql:get',
                    text: `SELECT *
                           FROM ${schemaPrefix}users
                           WHERE username LIKE '%admin%'`,
                    result: {
                        Columns: [
                            {Name: 'id', Type: 'integer', __typename: 'Column'},
                            {Name: 'username', Type: 'character varying', __typename: 'Column'}
                        ],
                        Rows: [['3', 'admin_user']],
                        __typename: 'RowsResult'
                    }
                }]);
                cy.sendChatMessage('Find users with admin in their username');
                cy.waitForChatResponse();
                cy.verifyChatSQLResult({rowCount: 1});
            });

            it('generates aggregate query', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'text',
                    text: 'Here\'s the user count.'
                }, {
                    type: 'sql:get',
                    text: `SELECT COUNT(*) as user_count
                           FROM ${schemaPrefix}users`,
                    result: {
                        Columns: [{Name: 'user_count', Type: 'bigint', __typename: 'Column'}],
                        Rows: [['3']],
                        __typename: 'RowsResult'
                    }
                }]);
                cy.sendChatMessage('Count users');
                cy.waitForChatResponse();
                cy.verifyChatSQLResult({
                    columns: ['user_count'],
                    rowCount: 1
                });
            });
        });

        describe('SQL Mutations', () => {
            it('executes INSERT operation', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'text',
                    text: 'I\'ll add a new user to the database.'
                }, {
                    type: 'sql:insert',
                    text: `INSERT INTO ${schemaPrefix}users (username, email)
                           VALUES ('test_user', 'test@example.com')`,
                    result: {Columns: [], Rows: [], __typename: 'RowsResult'}
                }]);
                cy.sendChatMessage('Add a new user named test_user');
                cy.waitForChatResponse();
                cy.verifyChatActionExecuted();
            });

            it('executes UPDATE operation', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'text',
                    text: 'I\'ll update the user\'s email address.'
                }, {
                    type: 'sql:update',
                    text: `UPDATE ${schemaPrefix}users
                           SET email = 'new@example.com'
                           WHERE username = 'test_user'`,
                    result: {Columns: [], Rows: [], __typename: 'RowsResult'}
                }]);
                cy.sendChatMessage('Update test_user email');
                cy.waitForChatResponse();
                cy.verifyChatActionExecuted();
            });

            it('executes DELETE operation with confirmation', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'text',
                    text: 'Are you sure you want to delete this user? This action cannot be undone.'
                }]);
                cy.sendChatMessage('Delete test_user from the database');
                cy.waitForChatResponse();
                cy.verifyChatSystemMessage('Are you sure you want to delete this user?');

                cy.mockChatResponse([{
                    type: 'sql:delete',
                    text: `DELETE
                           FROM ${schemaPrefix}users
                           WHERE username = 'test_user'`,
                    result: {Columns: [], Rows: [], __typename: 'RowsResult'}
                }]);
                cy.sendChatMessage('Yes, delete it');
                cy.waitForChatResponse();
                cy.verifyChatActionExecuted();
            });
        });

        describe('Error Handling', () => {
            it('displays error for invalid query', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'error',
                    text: 'ERROR: relation "nonexistent_table" does not exist'
                }]);
                cy.sendChatMessage('Show me data from nonexistent_table');
                cy.waitForChatResponse();
                cy.verifyChatError('relation "nonexistent_table" does not exist');
            });
        });

        describe('Move to Scratchpad', () => {
            it('moves query to scratchpad', () => {
                cy.gotoChat();

                cy.mockChatResponse([{
                    type: 'sql:get',
                    text: `SELECT *
                           FROM ${schemaPrefix}users`,
                    result: {
                        Columns: [{Name: 'id', Type: 'integer', __typename: 'Column'}],
                        Rows: [['1']],
                        __typename: 'RowsResult'
                    }
                }]);
                cy.sendChatMessage('Show users');
                cy.waitForChatResponse();

                cy.openMoveToScratchpad();
                cy.confirmMoveToScratchpad({pageOption: 'new', newPageName: 'Chat Queries'});

                cy.url({timeout: 10000}).should('include', '/scratchpad');
            });
        });

        // describe('Provider and Model Selection', () => {
        //     beforeEach(() => {
        //         cy.setupChatMock({modelType: 'Ollama', model: 'llama3.1'});
        //     });
        //
        //     it('shows provider dropdown with available options', () => {
        //         cy.gotoChat();
        //
        //         // Click the provider dropdown
        //         cy.get('[data-testid="ai-provider-select"]').click();
        //
        //         // Wait for dropdown options to appear
        //         cy.get('[role="option"]', { timeout: 5000 }).should('be.visible');
        //
        //         // Verify all expected providers are available
        //         // Internal providers: Ollama
        //         cy.get('[role="option"]').should('contain.text', 'Ollama');
        //
        //         // Verify the "Add Provider" option exists (for external providers like OpenAI, Anthropic)
        //         cy.contains('[role="menuitem"]', 'Add Provider').should('exist');
        //
        //         // Close dropdown
        //         cy.get('body').type('{esc}');
        //     });
        //
        //     it('updates model selection dropdown based on provider', () => {
        //         cy.gotoChat();
        //
        //         // Verify model dropdown is enabled for Ollama
        //         cy.get('[data-testid="ai-model-select"]')
        //             .should('be.visible')
        //             .should('not.be.disabled');
        //
        //         // Verify the model is populated (from our mock)
        //         cy.get('[data-testid="ai-model-select"]').should('contain.text', 'llama3.1');
        //
        //         // Click model dropdown to verify it opens
        //         cy.get('[data-testid="ai-model-select"]').click();
        //         cy.get('[role="option"]', { timeout: 5000 }).should('be.visible');
        //         cy.get('[role="option"]').should('contain.text', 'llama3.1');
        //
        //         // Close dropdown
        //         cy.get('body').type('{esc}');
        //     });
        //
        //     it('shows token input dialog for external cloud providers', () => {
        //         cy.gotoChat();
        //
        //         // Click provider dropdown
        //         cy.get('[data-testid="ai-provider-select"]').click();
        //
        //         // Click "Add Provider" option
        //         cy.contains('[role="menuitem"]', 'Add Provider').click();
        //
        //         // Wait for the external model sheet to open
        //         cy.contains('h2, .text-lg', /add.*external.*model/i, { timeout: 5000 }).should('be.visible');
        //
        //         // Verify model type dropdown exists
        //         cy.get('[data-testid="external-model-type-select"]').should('be.visible');
        //
        //         // Click to open external model type dropdown
        //         cy.get('[data-testid="external-model-type-select"]').click();
        //
        //         // Verify external providers are available (ChatGPT, Anthropic)
        //         cy.get('[role="option"]').should('contain.text', 'ChatGPT');
        //         cy.get('[role="option"]').should('contain.text', 'Anthropic');
        //
        //         // Close the dialog
        //         cy.get('[data-testid="external-model-cancel"]').click();
        //     });
        //
        //     it('persists provider selection in session', () => {
        //         cy.gotoChat();
        //
        //         // Verify the initial provider is Ollama (from our mock)
        //         cy.get('[data-testid="ai-provider-select"]').should('contain.text', 'Ollama');
        //
        //         // Verify the model is also persisted
        //         cy.get('[data-testid="ai-model-select"]').should('contain.text', 'llama3.1');
        //
        //         // Navigate away and back
        //         cy.visit('/storage-unit');
        //         cy.wait(500);
        //         cy.visit('/chat');
        //
        //         // Wait for chat to load
        //         cy.get('[data-testid="ai-provider"]', { timeout: 10000 }).should('exist');
        //
        //         // Verify provider is still selected (persistence)
        //         cy.get('[data-testid="ai-provider-select"]', { timeout: 5000 }).should('contain.text', 'Ollama');
        //     });
        //
        //     it('uses correct model lists for different providers', () => {
        //         cy.gotoChat();
        //
        //         // For Ollama (internal provider)
        //         cy.get('[data-testid="ai-provider-select"]').should('contain.text', 'Ollama');
        //         cy.get('[data-testid="ai-model-select"]').click();
        //         cy.get('[role="option"]', { timeout: 5000 }).should('be.visible');
        //         cy.get('[role="option"]').should('contain.text', 'llama3.1');
        //         cy.get('body').type('{esc}');
        //
        //         // Open Add Provider dialog to verify external providers have different structure
        //         cy.get('[data-testid="ai-provider-select"]').click();
        //         cy.contains('[role="menuitem"]', 'Add Provider').click();
        //
        //         // Verify external model setup requires token input
        //         cy.contains('label', /token/i).should('be.visible');
        //         cy.get('input[type="password"]').should('be.visible');
        //
        //         // Select an external provider type
        //         cy.get('[data-testid="external-model-type-select"]').click();
        //         cy.contains('[role="option"]', 'ChatGPT').click();
        //
        //         // Verify token input is required for external providers
        //         cy.get('input[type="password"]').should('be.visible').and('have.value', '');
        //
        //         // Cancel dialog
        //         cy.get('[data-testid="external-model-cancel"]').click();
        //     });
        //
        //     it('requires token for external providers before model selection', () => {
        //         cy.gotoChat();
        //
        //         // Open Add Provider dialog
        //         cy.get('[data-testid="ai-provider-select"]').click();
        //         cy.contains('[role="menuitem"]', 'Add Provider').click();
        //
        //         // Select external provider type
        //         cy.get('[data-testid="external-model-type-select"]').click();
        //         cy.contains('[role="option"]', 'Anthropic').click();
        //
        //         // Verify token input is shown
        //         cy.contains('label', /token/i).should('be.visible');
        //         cy.get('input[type="password"]').should('be.visible');
        //
        //         // Verify submit button exists but would require token to proceed
        //         cy.get('[data-testid="external-model-submit"]').should('be.visible');
        //
        //         // Try to submit without token (should stay on dialog)
        //         cy.get('[data-testid="external-model-submit"]').click();
        //
        //         // Dialog should still be visible (validation prevents submission)
        //         cy.contains('h2, .text-lg', /add.*external.*model/i).should('be.visible');
        //
        //         // Cancel dialog
        //         cy.get('[data-testid="external-model-cancel"]').click();
        //     });
        //
        //     it('allows deleting custom providers', () => {
        //         cy.gotoChat();
        //
        //         // Verify delete provider button exists
        //         cy.get('[data-testid="chat-delete-provider"]').should('be.visible');
        //
        //         // Click delete provider button
        //         cy.get('[data-testid="chat-delete-provider"]').click();
        //
        //         // Wait for confirmation dialog
        //         cy.contains(/delete.*provider/i, { timeout: 5000 }).should('be.visible');
        //
        //         // Verify confirmation dialog has expected content
        //         cy.contains(/delete.*provider.*confirm/i).should('be.visible');
        //
        //         // Cancel deletion
        //         cy.contains('button', /cancel/i).click();
        //
        //         // Verify we're back on chat and provider is still selected
        //         cy.get('[data-testid="ai-provider-select"]').should('contain.text', 'Ollama');
        //     });
        // });

        describe('Chat History', () => {
            it('navigates chat history with arrow keys', () => {
                cy.gotoChat();

                cy.mockChatResponse([{type: 'text', text: 'Response 1'}]);
                cy.sendChatMessage('First message');
                cy.waitForChatResponse();

                cy.mockChatResponse([{type: 'text', text: 'Response 2'}]);
                cy.sendChatMessage('Second message');
                cy.waitForChatResponse();

                cy.navigateChatHistory('up');
                cy.getChatInputValue().should('equal', 'Second message');

                cy.navigateChatHistory('up');
                cy.getChatInputValue().should('equal', 'First message');
            });

            it('clears chat history', () => {
                cy.gotoChat();

                cy.mockChatResponse([{type: 'text', text: 'Hello!'}]);
                cy.sendChatMessage('Hi');
                cy.waitForChatResponse();

                cy.clearChat();
                cy.verifyChatEmpty();
            });
        });
    }, {features: ['chat']});
});
