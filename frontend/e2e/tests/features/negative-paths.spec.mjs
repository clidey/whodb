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
import { getSqlQuery } from '../../support/database-config.mjs';
import { clearBrowserState } from '../../support/helpers/animation.mjs';

const REPRESENTATIVE_DATABASES = ['postgres', 'mongodb', 'redis', 'memcached'];

async function dismissTelemetry(page) {
    const disableBtn = page.locator('button').filter({ hasText: 'Disable Telemetry' });
    if (await disableBtn.count() > 0) {
        await disableBtn.click();
    }
}

async function attemptLoginWithUnreachablePort(page, whodb, db) {
    const conn = db.connection;

    await clearBrowserState(page);
    await page.goto(whodb.url('/login'));
    await dismissTelemetry(page);

    await page.locator('[data-testid="database-type-select"]').click();
    await page.locator(`[data-value="${db.uiType || db.type}"]`).click();

    if (conn.host !== undefined && conn.host !== null) {
        await page.locator('[data-testid="hostname"]').clear();
        await page.locator('[data-testid="hostname"]').fill(conn.host);
    }
    if (conn.user !== undefined && conn.user !== null) {
        await page.locator('[data-testid="username"]').clear();
        await page.locator('[data-testid="username"]').fill(conn.user);
    }
    if (conn.password !== undefined && conn.password !== null) {
        await page.locator('[data-testid="password"]').clear();
        await page.locator('[data-testid="password"]').fill(conn.password);
    }
    if (conn.database !== undefined && conn.database !== null) {
        await page.locator('[data-testid="database"]').clear();
        await page.locator('[data-testid="database"]').fill(conn.database);
    }

    await page.locator('[data-testid="port"]').clear();
    await page.locator('[data-testid="port"]').fill('1');

    await page.locator('[data-testid="login-button"]').click();
    await expect(page.getByText(/Login Failed/i)).toBeVisible({ timeout: 45_000 });
    await expect(page).toHaveURL(/\/login/);
}

test.describe('Negative Path Contracts', () => {
    forEachDatabase('sql', (db) => {
        test('invalid scratchpad query shows an error and remains recoverable', async ({ whodb }) => {
            await whodb.goto('scratchpad');

            await whodb.writeCode(0, getSqlQuery(db, 'invalidQuery'));
            await whodb.runCode(0);
            const error = await whodb.getCellError(0);
            expect(error.length).toBeGreaterThan(0);

            await whodb.writeCode(0, getSqlQuery(db, 'countUsers'));
            await whodb.runCode(0);
            const { rows } = await whodb.getCellQueryOutput(0);
            expect(rows.length).toBeGreaterThan(0);
        });
    }, { features: ['scratchpad'] });

    forEachDatabase('all', (db) => {
        test('unreachable port fails login without leaving login page', async ({ whodb, page }) => {
            await attemptLoginWithUnreachablePort(page, whodb, db);
        });
    }, {
        login: false,
        logout: false,
        databases: REPRESENTATIVE_DATABASES,
    });
});
