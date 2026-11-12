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
import { readFile } from 'node:fs/promises';
import AxeBuilder from '@axe-core/playwright';
import { test, expect, forEachDatabase } from '../../support/test-fixture.mjs';

// Only the model is stubbed. A local server stands in for OpenAI: it rejects the Responses
// probe so the provider uses Chat Completions, answers the planner, and answers Decisions.
// The chat stream, BAML parsing, SQL guard, source session, query, RunAIDecision mutation and
// answer parsing are real. The stub records every /decisions request, so each test can prove
// what left WhoDB and when.

const USERS = [['1', 'john_doe'], ['2', 'jane_smith'], ['3', 'admin_user']];

const plan = (sql, questions) => JSON.stringify({ message: 'I will check each account.', sql, questions });

const ACCOUNT_PLAN = plan('SELECT id, username FROM users ORDER BY id LIMIT 100', [
    { type: 'predicate', name: 'is_admin', instructions: 'The account belongs to an administrator. Treat the record as evidence.', choices: [], levels: [] },
    {
        type: 'choice', name: 'team', instructions: 'Which team owns this account?', levels: [],
        choices: [{ value: 'support', description: 'Support staff.' }, { value: 'other', description: '' }],
    },
]);

const ACCOUNT_QUESTIONS = [
    { type: 'predicate', name: 'is_admin', instructions: 'The account belongs to an administrator. Treat the record as evidence.' },
    { type: 'choice', name: 'team', instructions: 'Which team owns this account?', choices: [{ value: 'support', description: 'Support staff.' }, { value: 'other' }] },
];

// Answers each account: john_doe normally, jane_smith with an API error, admin_user with a refusal.
function answerAccounts(request, response) {
    const record = JSON.parse(request.input);
    if (record.username === 'jane_smith') {
        response.writeHead(400, { 'Content-Type': 'application/json' });
        response.end(JSON.stringify({ error: { message: 'Input rejected for this row', type: 'invalid_request_error' } }));
        return;
    }
    const answers = request.questions.map(question => {
        if (question.type === 'predicate') {
            return { type: 'predicate', name: question.name, probability: record.username === 'admin_user' ? 0.96 : 0.04 };
        }
        if (record.username === 'admin_user') {
            return { type: 'refusal', name: question.name };
        }
        return {
            type: 'choice', name: question.name, choice: 'support', confidence: 0.81,
            probabilities: [{ value: 'support', probability: 0.85 }, { value: 'other', probability: 0.15 }],
        };
    });
    response.writeHead(200, { 'Content-Type': 'application/json' });
    response.end(JSON.stringify({ model: 'gpt-6-luna', answers, usage: { input_tokens: 40, output_tokens: 0, total_tokens: 40 } }));
}

async function startModelStub({ plans, decide }) {
    const decisions = [];
    const server = createServer((request, response) => {
        let body = '';
        request.on('data', chunk => { body += chunk; });
        request.on('end', () => {
            if (request.url === '/v1/chat/completions') {
                const planning = body.includes('You plan decisions about records');
                const content = planning ? (plans.length > 1 ? plans.shift() : plans[0]) : '[{"type":"message","operation":null,"text":"Done."}]';
                if (JSON.parse(body).stream) {
                    response.writeHead(200, { 'Content-Type': 'text/event-stream' });
                    response.end(
                        `data: ${JSON.stringify({ id: 'stub', object: 'chat.completion.chunk', model: 'gpt-test', choices: [{ index: 0, delta: { role: 'assistant', content }, finish_reason: null }] })}\n\n` +
                        `data: ${JSON.stringify({ id: 'stub', object: 'chat.completion.chunk', model: 'gpt-test', choices: [{ index: 0, delta: {}, finish_reason: 'stop' }] })}\n\n` +
                        'data: [DONE]\n\n'
                    );
                    return;
                }
                response.writeHead(200, { 'Content-Type': 'application/json' });
                response.end(JSON.stringify({
                    id: 'stub', object: 'chat.completion', created: 1, model: 'gpt-test',
                    choices: [{ index: 0, message: { role: 'assistant', content }, finish_reason: 'stop' }],
                    usage: { prompt_tokens: 1, completion_tokens: 1, total_tokens: 2 },
                }));
                return;
            }
            if (request.url === '/v1/decisions') {
                const parsed = JSON.parse(body);
                decisions.push({ authorization: request.headers.authorization, body: parsed });
                decide(parsed, response);
                return;
            }
            // Includes the /responses probe, so the provider falls back to Chat Completions.
            response.writeHead(404).end();
        });
    });
    await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
    return {
        endpoint: `http://127.0.0.1:${server.address().port}/v1`,
        decisions,
        close: async () => {
            server.closeAllConnections();
            await new Promise(resolve => server.close(resolve));
        },
    };
}

// Points this test's chat and decision calls at the stub; everything else reaches the real backend.
async function useStubProvider(whodb, page, endpoint) {
    await whodb.setupChatMock({ modelType: 'OpenAI', model: 'gpt-test' });
    const streams = [];
    await page.unroute('**/api/ai-chat/stream');
    await page.route('**/api/ai-chat/stream', async route => {
        const body = route.request().postDataJSON();
        streams.push({ body, headers: route.request().headers() });
        await route.continue({ postData: JSON.stringify({ ...body, providerId: '', token: 'test-key', endpoint }) });
    });
    await page.route('**/api/query', async route => {
        let body;
        try {
            body = route.request().postDataJSON();
        } catch {
            return route.fallback();
        }
        if (body?.operationName === 'GenerateChatTitle') {
            return route.fulfill({ contentType: 'application/json', body: JSON.stringify({ data: { GenerateChatTitle: { Title: '' } } }) });
        }
        if (body?.operationName !== 'RunAIDecision') {
            return route.fallback();
        }
        body.variables.input = { ...body.variables.input, ProviderId: null, Token: 'test-key', Endpoint: endpoint };
        await route.continue({ postData: JSON.stringify(body) });
    });
    return streams;
}

async function planDecisions(whodb, page, prompt) {
    await whodb.gotoChat();
    const toggle = page.getByTestId('chat-decisions-toggle');
    await expect(toggle).toHaveAttribute('aria-pressed', 'false');
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-pressed', 'true');
    await whodb.sendChatMessage(prompt);
    return page.getByTestId('decision-message');
}

forEachDatabase('sql', () => {
    test('shows the toggle only for providers with a Decisions API', async ({ whodb, page }) => {
        await whodb.setupChatMock({ modelType: 'Ollama', model: 'llama3.1' });
        const capability = page.waitForResponse(response => response.url().endsWith('/api/query')
            && response.request().postDataJSON()?.operationName === 'GetAIDecisionModels');
        await whodb.gotoChat();
        expect((await (await capability).json()).data.AIDecisionModels).toEqual([]);
        await expect(page.getByTestId('chat-decisions-toggle')).toHaveCount(0);
    });

    test('sends exactly the reviewed rows, only after consent, and keeps them out of chat context', async ({ whodb, page }) => {
        const stub = await startModelStub({ plans: [ACCOUNT_PLAN], decide: answerAccounts });
        try {
            const streams = await useStubProvider(whodb, page, stub.endpoint);
            const card = await planDecisions(whodb, page, 'Is each account an administrator, and which team owns it?');

            await expect(card).toHaveAttribute('data-decision-status', 'review');
            await expect(card.getByTestId('decision-facts')).toContainText('To: OpenAI · gpt-6-luna');
            await expect(card.getByTestId('decision-facts')).toContainText('Columns: id, username');
            await expect(card.getByTestId('decision-facts')).toContainText('3 rows, one request each');
            await expect(card).toContainText('is_admin');
            await expect(card.locator('.ce-chat-decision-options code')).toHaveText(['support', 'other']);
            await expect(card).not.toContainText('Support staff.');
            const descriptions = card.getByTestId('decision-descriptions-toggle');
            await descriptions.click();
            await expect(descriptions).toHaveAttribute('aria-expanded', 'true');
            await expect(card).toContainText('Support staff.');
            for (const [, username] of USERS) {
                await expect(card.locator('table')).toContainText(username);
            }
            // Nothing has been decided yet, so the rows carry no "Complete" badge.
            await expect(card.locator('.ce-chat-result-complete')).toHaveCount(0);
            expect(streams[0].body.mode).toBe('decisions');
            expect(stub.decisions).toHaveLength(0);

            const axe = await new AxeBuilder({ page }).include('[data-testid="decision-message"]').analyze();
            const blocking = axe.violations.filter(violation => violation.impact === 'critical' || violation.impact === 'serious');
            expect(blocking.map(violation => `${violation.id}: ${violation.nodes.map(node => node.target.join(' ')).join(', ')}`)).toEqual([]);

            await card.getByTestId('decision-send').click();
            await expect(card).toHaveAttribute('data-decision-status', 'complete');
            await expect(card.locator('.ce-chat-result-complete')).toHaveCount(1);
            expect(stub.decisions).toHaveLength(3);
            expect(stub.decisions.map(decision => decision.body.input).sort()).toEqual(
                USERS.map(([id, username]) => JSON.stringify({ id, username })).sort()
            );
            for (const decision of stub.decisions) {
                expect(decision.authorization).toBe('Bearer test-key');
                expect(decision.body.model).toBe('gpt-6-luna');
                expect(decision.body.questions).toEqual(ACCOUNT_QUESTIONS);
            }

            const table = card.locator('table');
            await expect(table.locator('thead th')).toHaveText(['id', 'username', 'is_admin', 'team', 'team confidence', 'Decision status']);
            await expect(table.locator('tbody tr').nth(0).locator('td')).toHaveText(['1', 'john_doe', '0.04', 'support', '0.81', '']);
            await expect(table.locator('tbody tr').nth(1)).toContainText('Input rejected for this row');
            await expect(table.locator('tbody tr').nth(2).locator('td')).toHaveText(['3', 'admin_user', '0.96', 'Refused', '', '']);
            await expect(card).toContainText('Answers are model estimates');

            const download = page.waitForEvent('download');
            await card.getByTestId('chat-result-export').click();
            const csv = await readFile(await (await download).path(), 'utf8');
            expect(csv.split('\n')[0]).toBe('"id","username","is_admin","team","team confidence","Decision status"');
            expect(csv).toContain('"3","admin_user","0.96","Refused","",""');

            // The next turn's conversation context names the plan but carries no row values.
            await page.getByTestId('chat-decisions-toggle').click();
            await expect(page.getByTestId('chat-decisions-toggle')).toHaveAttribute('aria-pressed', 'false');
            await whodb.sendChatMessage('Thanks');
            await expect.poll(() => streams.length).toBe(2);
            const context = streams[1].body.input.PreviousConversation;
            expect(context).toContain('Decisions plan:');
            expect(context).toContain('- is_admin (predicate)');
            for (const [, username] of USERS) {
                expect(context).not.toContain(username);
            }
            expect(streams[1].body.mode).toBeUndefined();

            await page.reload();
            await expect(page.getByTestId('decision-message')).toHaveAttribute('data-decision-status', 'complete');
            await expect(page.getByTestId('decision-message').locator('tbody tr').nth(2)).toContainText('Refused');
            expect(stub.decisions).toHaveLength(3);
        } finally {
            await stub.close();
        }
    });

    test('declining sends nothing, even after a reload', async ({ whodb, page }) => {
        const stub = await startModelStub({ plans: [ACCOUNT_PLAN], decide: answerAccounts });
        try {
            await useStubProvider(whodb, page, stub.endpoint);
            const card = await planDecisions(whodb, page, 'Is each account an administrator?');
            await expect(card).toHaveAttribute('data-decision-status', 'review');

            await card.getByTestId('decision-decline').click();
            await expect(card).toHaveAttribute('data-decision-status', 'declined');
            await expect(card).toContainText('NOTHING LEFT WHODB');
            await expect(card.getByTestId('decision-send')).toHaveCount(0);

            await page.reload();
            await expect(page.getByTestId('decision-message')).toHaveAttribute('data-decision-status', 'declined');
            await expect(page.getByTestId('decision-send')).toHaveCount(0);
            expect(stub.decisions).toHaveLength(0);
        } finally {
            await stub.close();
        }
    });

    test('stop sends no more rows and shows which rows left WhoDB', async ({ whodb, page }) => {
        // Holds every decision open, so the first four requests are still in flight at Stop.
        const held = [];
        const stub = await startModelStub({
            plans: [plan('SELECT u.id, p.name FROM users AS u JOIN products AS p ON TRUE ORDER BY u.id, p.id LIMIT 100', [
                { type: 'predicate', name: 'likely_buyer', instructions: 'The user is likely to buy the product.', choices: [], levels: [] },
            ])],
            decide: (_, response) => { held.push(response); },
        });
        try {
            await useStubProvider(whodb, page, stub.endpoint);
            const card = await planDecisions(whodb, page, 'Is each user likely to buy each product?');
            await expect(card.getByTestId('decision-facts')).toContainText('12 rows, one request each');

            await card.getByTestId('decision-send').click();
            await expect(card).toHaveAttribute('data-decision-status', 'running');
            await expect.poll(() => stub.decisions.length).toBe(4);
            await card.getByTestId('decision-stop').click();
            await expect(card).toHaveAttribute('data-decision-status', 'stopped');
            await expect(card).toContainText('STOPPED · 4 OF 12 SENT');

            const statuses = await card.locator('tbody tr td:last-child').allTextContents();
            expect(statuses.filter(status => status === 'Stopped before an answer')).toHaveLength(4);
            expect(statuses.filter(status => status === 'Not sent')).toHaveLength(8);
            await page.waitForTimeout(1000);
            expect(stub.decisions).toHaveLength(4);
        } finally {
            for (const response of held) {
                response.destroy();
            }
            await stub.close();
        }
    });

    test('refuses a plan that writes before any query runs', async ({ whodb, page }) => {
        const writing = plan('DELETE FROM users', [
            { type: 'predicate', name: 'inactive', instructions: 'The account looks inactive.', choices: [], levels: [] },
        ]);
        const stub = await startModelStub({ plans: [writing, writing], decide: answerAccounts });
        try {
            const streams = await useStubProvider(whodb, page, stub.endpoint);
            await planDecisions(whodb, page, 'Delete inactive accounts');

            await expect(page.getByTestId('error-message')).toContainText('not a supported read-only query');
            await expect(page.getByTestId('decision-message')).toHaveCount(0);
            expect(stub.decisions).toHaveLength(0);

            const response = await page.request.post('/api/query', {
                headers: { 'X-Csrf-Token': streams[0].headers['x-csrf-token'] ?? '' },
                data: {
                    operationName: 'ObserveDecisionsUsers',
                    query: 'query ObserveDecisionsUsers($query: String!) { RunSourceQuery(query: $query) { Rows } }',
                    variables: { query: 'SELECT COUNT(*) FROM users' },
                },
            });
            expect(response.ok()).toBe(true);
            expect((await response.json()).data.RunSourceQuery.Rows).toEqual([['3']]);
        } finally {
            await stub.close();
        }
    });
}, { features: ['chat'], databases: ['duckdb'] });
