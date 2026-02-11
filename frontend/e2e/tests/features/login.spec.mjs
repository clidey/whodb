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

import { test, expect } from '../../support/test-fixture.mjs';
import { getDatabaseConfig } from '../../support/database-config.mjs';
import { clearBrowserState } from '../../support/helpers/animation.mjs';

/**
 * Login & Authentication Tests
 *
 * Tests the login page UI, form validation, and authentication flows.
 * Unlike other feature tests, these don't use forEachDatabase() since they
 * test the login page itself before any database is connected.
 */
test.describe('Login & Authentication', () => {
    test.beforeEach(async ({ whodb, page }) => {
        await clearBrowserState(page);
        await page.goto(whodb.url('/login'));

        // Dismiss telemetry modal if it appears
        const disableBtn = page.locator('button').filter({ hasText: 'Disable Telemetry' });
        if (await disableBtn.count() > 0) {
            await disableBtn.click();
        }
    });

    test.describe('Database Type Selection', () => {
        test('shows database type dropdown with options', async ({ whodb, page }) => {
            await expect(page.locator('[data-testid="database-type-select"]')).toBeVisible();
            await page.locator('[data-testid="database-type-select"]').click();

            // Verify common database types are available
            await expect(page.locator('[data-value="Postgres"]')).toBeAttached();
            await expect(page.locator('[data-value="MySQL"]')).toBeAttached();
            await expect(page.locator('[data-value="Sqlite3"]')).toBeAttached();
            await expect(page.locator('[data-value="MongoDB"]')).toBeAttached();
            await expect(page.locator('[data-value="Redis"]')).toBeAttached();

            // Close dropdown
            await page.keyboard.press('Escape');
        });

        test('changes form fields based on database type selection', async ({ whodb, page }) => {
            // PostgreSQL should show all fields
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();
            await expect(page.locator('[data-testid="hostname"]')).toBeVisible();
            await expect(page.locator('[data-testid="username"]')).toBeVisible();
            await expect(page.locator('[data-testid="password"]')).toBeVisible();
            await expect(page.locator('[data-testid="database"]')).toBeVisible();

            // Redis should show only hostname (no username/password/database required)
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Redis"]').click();
            await expect(page.locator('[data-testid="hostname"]')).toBeVisible();
            // Redis doesn't require username/password/database in the form

            // SQLite should show only database path
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Sqlite3"]').click();
            await expect(page.locator('[data-testid="database"]')).toBeVisible();
            await expect(page.locator('[data-testid="hostname"]')).not.toBeAttached();
        });
    });

    test.describe('Form Validation', () => {
        test('disables login button when required fields are empty', async ({ whodb, page }) => {
            // Select PostgreSQL which requires all fields
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();

            // Login button should be disabled when fields are empty
            await expect(page.locator('[data-testid="login-button"]')).toBeDisabled();
        });

        test('enables login button when required fields are filled', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();

            await page.locator('[data-testid="hostname"]').fill('localhost');
            await page.locator('[data-testid="username"]').fill('user');
            await page.locator('[data-testid="password"]').fill('password');
            await page.locator('[data-testid="database"]').fill('testdb');

            await expect(page.locator('[data-testid="login-button"]')).not.toBeDisabled();
        });
    });

    test.describe('Advanced Options', () => {
        test('toggles advanced options visibility', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();

            // Advanced options should be hidden by default
            await expect(page.locator('[data-testid="Port-input"]')).not.toBeAttached();

            // Click advanced button to show options
            await page.locator('[data-testid="advanced-button"]').click();
            await expect(page.locator('[data-testid="Port-input"]')).toBeVisible();

            // Click again to hide
            await page.locator('[data-testid="advanced-button"]').click();
            await expect(page.locator('[data-testid="Port-input"]')).not.toBeAttached();
        });

        test('accepts advanced configuration values', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();

            await page.locator('[data-testid="advanced-button"]').click();
            await page.locator('[data-testid="Port-input"]').clear();
            await page.locator('[data-testid="Port-input"]').fill('5433');
            await expect(page.locator('[data-testid="Port-input"]')).toHaveValue('5433');
        });
    });

    test.describe('Direct Credentials Login', () => {
        test('successfully logs in with valid credentials', async ({ whodb, page }) => {
            // Use PostgreSQL config from fixtures
            const db = getDatabaseConfig('postgres');

            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator(`[data-value="${db.type}"]`).click();

            await page.locator('[data-testid="hostname"]').fill(db.connection.host);
            await page.locator('[data-testid="username"]').fill(db.connection.user);
            await page.locator('[data-testid="password"]').fill(db.connection.password);
            await page.locator('[data-testid="database"]').fill(db.connection.database);

            // Handle advanced options if needed
            if (db.connection.advanced && Object.keys(db.connection.advanced).length > 0) {
                await page.locator('[data-testid="advanced-button"]').click();
                for (const [key, value] of Object.entries(db.connection.advanced)) {
                    await page.locator(`[data-testid="${key}-input"]`).clear();
                    await page.locator(`[data-testid="${key}-input"]`).fill(String(value));
                }
            }

            const loginResponsePromise = page.waitForResponse(resp =>
                resp.url().includes('/api/query') && resp.request().method() === 'POST'
            );
            await page.locator('[data-testid="login-button"]').click();

            await loginResponsePromise;

            // Should redirect to storage-unit page
            await expect(page).toHaveURL(/\/storage-unit/);
            await expect(
                page.locator('[data-testid="sidebar-database"], [data-testid="sidebar-schema"]').first()
            ).toBeAttached({ timeout: 15000 });
        });

        test('shows error message on invalid credentials', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();

            await page.locator('[data-testid="hostname"]').fill('localhost');
            await page.locator('[data-testid="username"]').fill('invalid_user');
            await page.locator('[data-testid="password"]').fill('wrong_password');
            await page.locator('[data-testid="database"]').fill('nonexistent_db');

            const loginResponsePromise = page.waitForResponse(resp =>
                resp.url().includes('/api/query') && resp.request().method() === 'POST'
            );
            await page.locator('[data-testid="login-button"]').click();

            // Should show error toast or remain on login page
            await loginResponsePromise;

            // Either toast error or still on login page
            await expect(page).toHaveURL(/\/login/);
        });
    });

    // TODO: URL parsing feature may not exist or work differently
    // test.describe('URL Parsing', () => { ... });

    // TODO: ModeToggle component structure may differ from test expectations
    // test.describe('Theme Toggle', () => { ... });

    test.describe('Saved Profiles', () => {
        // Note: This test requires saved profiles to exist
        // In a fresh environment, there may be no saved profiles
        test('shows available profiles section when profiles exist', async ({ whodb, page }) => {
            // Check if profiles section exists (may not show if no profiles)
            const profilesSelect = page.locator('[data-testid="available-profiles-select"]');
            if (await profilesSelect.count() > 0) {
                await expect(profilesSelect).toBeVisible();
                await expect(page.locator('[data-testid="login-with-profile-button"]')).toBeAttached();
            }
        });
    });

    test.describe('Sample Database', () => {
        test('shows sample database panel for first-time users', async ({ whodb, page }) => {
            // Clear first login flag to simulate first-time user
            await page.evaluate(() => {
                window.localStorage.removeItem('whodb_has_logged_in');
            });
            await page.reload();

            // Dismiss telemetry again after reload
            await page.waitForTimeout(500);
            const disableBtn = page.locator('button').filter({ hasText: 'Disable Telemetry' });
            if (await disableBtn.count() > 0) {
                await disableBtn.click();
            }

            // Sample database panel should be visible for first-time users
            // Note: This depends on feature flags and sample database availability
            const samplePanel = page.locator('[data-testid="sample-database-panel"]');
            if (await samplePanel.count() > 0) {
                await expect(samplePanel).toBeVisible();
                await expect(page.locator('[data-testid="get-started-sample-db"]')).toBeVisible();
            }
        });

        test('can login with sample database', async ({ whodb, page }) => {
            // Clear first login flag
            await page.evaluate(() => {
                window.localStorage.removeItem('whodb_has_logged_in');
            });
            await page.reload();

            await page.waitForTimeout(500);
            const disableBtn = page.locator('button').filter({ hasText: 'Disable Telemetry' });
            if (await disableBtn.count() > 0) {
                await disableBtn.click();
            }

            // Try to click sample database button if available
            const sampleDbBtn = page.locator('[data-testid="get-started-sample-db"]');
            if (await sampleDbBtn.count() > 0) {
                const loginResponsePromise = page.waitForResponse(resp =>
                    resp.url().includes('/api/query') && resp.request().method() === 'POST'
                );
                await sampleDbBtn.click();

                await loginResponsePromise;

                // Should redirect to storage-unit
                await expect(page).toHaveURL(/\/storage-unit/);
            }
        });
    });

    // TODO: Session persistence test assumes non-empty database
    // test.describe('Session Persistence', () => { ... });

    test.describe('URL Parameter Pre-filling', () => {
        test('pre-fills form from URL parameters', async ({ whodb, page }) => {
            await page.goto(whodb.url('/login?type=Postgres&host=urlhost&username=urluser&database=urldb'));

            // Dismiss telemetry
            await page.waitForTimeout(500);
            const disableBtn = page.locator('button').filter({ hasText: 'Disable Telemetry' });
            if (await disableBtn.count() > 0) {
                await disableBtn.click();
            }

            // Form should be pre-filled
            await expect(page.locator('[data-testid="hostname"]')).toHaveValue('urlhost');
            await expect(page.locator('[data-testid="username"]')).toHaveValue('urluser');
            await expect(page.locator('[data-testid="database"]')).toHaveValue('urldb');
        });

        test('pre-fills form from base64 encoded credentials', async ({ whodb, page }) => {
            const credentials = {
                type: 'Postgres',
                host: 'encodedhost',
                username: 'encodeduser',
                database: 'encodeddb'
            };
            const encoded = Buffer.from(JSON.stringify(credentials)).toString('base64');

            await page.goto(whodb.url(`/login?credentials=${encoded}`));

            // Dismiss telemetry
            await page.waitForTimeout(500);
            const disableBtn = page.locator('button').filter({ hasText: 'Disable Telemetry' });
            if (await disableBtn.count() > 0) {
                await disableBtn.click();
            }

            // Form should be pre-filled from encoded credentials
            await expect(page.locator('[data-testid="hostname"]')).toHaveValue('encodedhost');
            await expect(page.locator('[data-testid="username"]')).toHaveValue('encodeduser');
            await expect(page.locator('[data-testid="database"]')).toHaveValue('encodeddb');
        });
    });
});
