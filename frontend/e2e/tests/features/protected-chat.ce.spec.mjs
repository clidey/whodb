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

import { createServer } from 'node:http';
import { test, expect, forEachDatabase } from '../../support/test-fixture.mjs';

// Only the model is stubbed: the browser SSE request, BAML parsing, SQL guard,
// source session, database execution and confirmation mutation remain real.
forEachDatabase('sql', (db) => {
    test('executes protected chat reads and requires approval despite a misleading model label', async ({ whodb, page }) => {
        let modelSQL = db.sql.selectUserById;
        let providerCalls = 0;
        const provider = createServer((request, response) => {
            request.resume();
            providerCalls++;
            const content = JSON.stringify([{ type: 'sql', operation: 'get', text: modelSQL }]);
            response.writeHead(200, { 'Content-Type': 'text/event-stream' });
            response.end(
                `data: ${JSON.stringify({ id: 'local-readguard', object: 'chat.completion.chunk', model: 'llama3.1', choices: [{ index: 0, delta: { role: 'assistant', content }, finish_reason: null }] })}\n\n` +
                `data: ${JSON.stringify({ id: 'local-readguard', object: 'chat.completion.chunk', model: 'llama3.1', choices: [{ index: 0, delta: {}, finish_reason: 'stop' }] })}\n\n` +
                'data: [DONE]\n\n'
            );
        });
        await new Promise(resolve => provider.listen(0, '127.0.0.1', resolve));
        const endpoint = `http://127.0.0.1:${provider.address().port}`;
        let streamHeaders;
        let restoreData;
        try {
            await whodb.setupChatMock();
            await page.unroute('**/api/ai-chat/stream');
            await page.route('**/api/ai-chat/stream', async route => {
                streamHeaders = route.request().headers();
                const body = route.request().postDataJSON();
                // Point only this test's model call at the local provider.
                await route.continue({ postData: JSON.stringify({ ...body, providerId: '', endpoint }) });
            });
            await whodb.gotoChat();

            const ask = async prompt => {
                const response = page.waitForResponse('**/api/ai-chat/stream');
                await whodb.sendChatMessage(prompt);
                const stream = await response;
                expect(stream.ok()).toBe(true);
            };
            const observe = async query => {
                const response = await page.request.post('/api/query', {
                    headers: { 'X-CSRF-Token': streamHeaders['x-csrf-token'] ?? '' },
                    data: {
                        operationName: 'ObserveProtectedChat',
                        query: 'query ObserveProtectedChat($query: String!) { RunSourceQuery(query: $query) { Rows } }',
                        variables: { query },
                    },
                });
                expect(response.ok()).toBe(true);
                const body = await response.json();
                expect(body.errors).toBeUndefined();
                return body.data.RunSourceQuery.Rows;
            };

            await ask('Read the first user');
            await whodb.verifyChatSQLResult({ rowCount: 1 });
            await expect(page.locator('table').last()).toContainText(db.testTable.firstName);
            await expect(page.getByTestId('confirmation-message')).toHaveCount(0);

            // Use an isolated table: the shared users fixture has foreign keys
            // and indexed-column updates exercise unrelated DuckDB limitations.
            const tableName = `protected_chat_${Date.now()}`;
            const table = `${db.sql.schemaPrefix}${tableName}`;
            const exists = `SELECT count(*) FROM information_schema.tables WHERE table_name = '${tableName}'`;
            restoreData = () => observe(`DROP TABLE IF EXISTS ${table}`);
            modelSQL = `CREATE TABLE ${table} AS SELECT 731 AS guarded_value`;
            await ask('Create a table containing the requested value');
            const confirmation = page.getByTestId('confirmation-message');
            await expect(confirmation).toBeVisible();
            expect(await observe(exists)).toEqual([['0']]);
            const confirmed = page.waitForResponse(response => {
                if (!response.url().endsWith('/api/query')) return false;
                return response.request().postDataJSON()?.operationName === 'ExecuteConfirmedSQL';
            });
            await confirmation.getByRole('button', { name: 'Yes', exact: true }).click();
            const confirmedBody = await (await confirmed).json();
            expect(confirmedBody.errors).toBeUndefined();
            expect(confirmedBody.data.ExecuteConfirmedSQL.Type).not.toBe('error');
            await expect(confirmation).toHaveCount(0);
            expect(await observe(exists)).toEqual([['1']]);

            // A new protected read must still work after the approved write.
            modelSQL = `SELECT guarded_value FROM ${table}`;
            await ask('Read the new table');
            await whodb.verifyChatSQLResult({ rowCount: 1, columns: ['guarded_value'] });
            await expect(page.locator('table').last()).toContainText('731');
            expect(await observe(modelSQL)).toEqual([['731']]);
            await expect(page.getByTestId('confirmation-message')).toHaveCount(0);
            await expect(page.getByTestId('error-message')).toHaveCount(0);
            expect(providerCalls).toBe(3);
        } finally {
            try {
                if (restoreData) await restoreData();
            } finally {
                provider.closeAllConnections();
                await new Promise(resolve => provider.close(resolve));
            }
        }
    });
}, { features: ['chat', 'scratchpad'], databases: ['duckdb'] });
