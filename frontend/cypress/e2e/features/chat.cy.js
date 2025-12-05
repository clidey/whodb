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

import {forEachDatabase, hasFeature} from '../../support/test-runner';

describe('Chat AI Integration', () => {

    // Only SQL databases with chat feature
    forEachDatabase('sql', (db) => {
        if (!hasFeature(db, 'chat')) {
            return;
        }

        const schemaPrefix = db.sql?.schemaPrefix || '';

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
    });

});
