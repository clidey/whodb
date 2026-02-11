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

import { readFileSync } from 'fs';
import { test, expect } from '../../support/test-fixture.mjs';
import { getDatabaseConfig } from '../../support/database-config.mjs';
import { clearBrowserState } from '../../support/helpers/animation.mjs';

/**
 * Maps container certificate paths to host filesystem paths for testing.
 * Container paths like /app/certs/... map to ../dev/certs/... on the host.
 */
function containerPathToHostPath(containerPath) {
    if (!containerPath) return null;
    return containerPath.replace('/app/certs/', '../dev/certs/');
}

/**
 * SSL Configuration Tests
 *
 * Tests the SSL configuration UI in the login form's advanced options.
 * Verifies mode selection, certificate inputs, and conditional display.
 * Also includes integration tests for actual SSL connections when SSL databases are available.
 */
test.describe('SSL Configuration', () => {
    test.beforeEach(async ({ whodb, page }) => {
        await clearBrowserState(page);
        await page.goto(whodb.url('/login'));

        // Dismiss telemetry modal if it appears
        const disableBtn = page.locator('button').filter({ hasText: 'Disable Telemetry' });
        if (await disableBtn.count() > 0) {
            await disableBtn.click();
        }
    });

    test.describe('SSL Mode Dropdown', () => {
        test('shows SSL mode dropdown for PostgreSQL in advanced options', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();

            // Open advanced options
            await page.locator('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            await expect(page.locator('[data-testid="ssl-mode-select"]')).toBeVisible();
        });

        test('shows SSL mode dropdown for MySQL in advanced options', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="MySQL"]').click();

            // Open advanced options
            await page.locator('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            await expect(page.locator('[data-testid="ssl-mode-select"]')).toBeVisible();
        });

        test('shows SSL mode dropdown for ClickHouse in advanced options', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="ClickHouse"]').click();

            // Open advanced options
            await page.locator('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            await expect(page.locator('[data-testid="ssl-mode-select"]')).toBeVisible();
        });

        test('shows SSL mode dropdown for MongoDB in advanced options', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="MongoDB"]').click();

            // Open advanced options
            await page.locator('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            await expect(page.locator('[data-testid="ssl-mode-select"]')).toBeVisible();
        });

        test('shows SSL mode dropdown for Redis in advanced options', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Redis"]').click();

            // Open advanced options
            await page.locator('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            await expect(page.locator('[data-testid="ssl-mode-select"]')).toBeVisible();
        });

        test('shows SSL mode dropdown for Elasticsearch in advanced options', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="ElasticSearch"]').click();

            // Open advanced options
            await page.locator('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            await expect(page.locator('[data-testid="ssl-mode-select"]')).toBeVisible();
        });

        test('does NOT show SSL mode dropdown for SQLite', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Sqlite3"]').click();

            // Advanced button should be hidden for SQLite (no SSL/advanced options)
            await expect(page.locator('[data-testid="advanced-button"]')).not.toBeVisible();

            // SSL mode dropdown should NOT exist (SQLite doesn't support SSL)
            await expect(page.locator('[data-testid="ssl-mode-select"]')).not.toBeAttached();
        });
    });

    test.describe('SSL Mode Selection', () => {
        test.beforeEach(async ({ whodb, page }) => {
            // Setup PostgreSQL for SSL mode tests
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();
            await page.locator('[data-testid="advanced-button"]').click();
        });

        test('can select different SSL modes from dropdown', async ({ whodb, page }) => {
            // Click the SSL mode dropdown
            await page.locator('[data-testid="ssl-mode-select"]').click();

            // Should show available modes
            await expect(page.locator('[data-value="disabled"]')).toBeAttached();
            await expect(page.locator('[data-value="required"]')).toBeAttached();
            await expect(page.locator('[data-value="verify-ca"]')).toBeAttached();
            await expect(page.locator('[data-value="verify-identity"]')).toBeAttached();

            // Select required mode
            await page.locator('[data-value="required"]').click();

            // Dropdown should close and show selected value
            await expect(page.locator('[data-testid="ssl-mode-select"]')).toContainText('Required');
        });

        test('shows CA certificate input when verify-ca mode is selected', async ({ whodb, page }) => {
            // Initially, certificate inputs should not be visible
            await expect(page.locator('[data-testid="ssl-ca-certificate-choose-file"]')).not.toBeAttached();

            // Select verify-ca mode
            await page.locator('[data-testid="ssl-mode-select"]').click();
            await page.locator('[data-value="verify-ca"]').click();

            // CA certificate file picker should now be visible
            await expect(page.locator('[data-testid="ssl-ca-certificate-choose-file"]')).toBeVisible();
        });

        test('shows all certificate inputs when verify-identity mode is selected', async ({ whodb, page }) => {
            // Select verify-identity mode
            await page.locator('[data-testid="ssl-mode-select"]').click();
            await page.locator('[data-value="verify-identity"]').click();

            // CA certificate file picker should be visible
            await expect(page.locator('[data-testid="ssl-ca-certificate-choose-file"]')).toBeVisible();

            // Client certificate file pickers should be visible (optional)
            await expect(page.locator('[data-testid="ssl-client-certificate-choose-file"]')).toBeVisible();
            await expect(page.locator('[data-testid="ssl-client-private-key-choose-file"]')).toBeVisible();

            // Server name override should be visible
            await expect(page.locator('[data-testid="ssl-server-name-input"]')).toBeVisible();
        });

        test('hides certificate inputs when switching to disabled mode', async ({ whodb, page }) => {
            // First select verify-ca to show certificate inputs
            await page.locator('[data-testid="ssl-mode-select"]').click();
            await page.locator('[data-value="verify-ca"]').click();
            await expect(page.locator('[data-testid="ssl-ca-certificate-choose-file"]')).toBeVisible();

            // Now switch to disabled
            await page.locator('[data-testid="ssl-mode-select"]').click();
            await page.locator('[data-value="disabled"]').click();

            // Certificate inputs should be hidden
            await expect(page.locator('[data-testid="ssl-ca-certificate-choose-file"]')).not.toBeAttached();
        });
    });

    test.describe('Certificate Input Modes', () => {
        test.beforeEach(async ({ whodb, page }) => {
            // Setup PostgreSQL with verify-ca mode
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();
            await page.locator('[data-testid="advanced-button"]').click();
            await page.locator('[data-testid="ssl-mode-select"]').click();
            await page.locator('[data-value="verify-ca"]').click();
        });

        test('defaults to file picker mode for certificates', async ({ whodb, page }) => {
            // File picker button should be visible by default
            await expect(page.locator('[data-testid="ssl-ca-certificate-choose-file"]')).toBeVisible();

            // Content textarea should not be visible
            await expect(page.locator('[data-testid="ssl-ca-certificate-content"]')).not.toBeAttached();
        });

        test('can toggle to content (paste PEM) mode', async ({ whodb, page }) => {
            // Find and click the toggle button for CA certificate
            await page.getByRole('button', { name: 'Paste PEM' }).first().click();

            // Content textarea should now be visible
            await expect(page.locator('[data-testid="ssl-ca-certificate-content"]')).toBeVisible();

            // File picker should be hidden
            await expect(page.locator('[data-testid="ssl-ca-certificate-choose-file"]')).not.toBeAttached();
        });

        test('can toggle back to file picker mode', async ({ whodb, page }) => {
            // Toggle to content mode
            await page.getByRole('button', { name: 'Paste PEM' }).first().click();
            await expect(page.locator('[data-testid="ssl-ca-certificate-content"]')).toBeVisible();

            // Toggle back to file picker mode
            await page.getByRole('button', { name: 'Choose File' }).first().click();

            // File picker should be visible again
            await expect(page.locator('[data-testid="ssl-ca-certificate-choose-file"]')).toBeVisible();
            await expect(page.locator('[data-testid="ssl-ca-certificate-content"]')).not.toBeAttached();
        });

        test('can paste PEM content', async ({ whodb, page }) => {
            // Toggle to content mode
            await page.getByRole('button', { name: 'Paste PEM' }).first().click();

            const testPEM = '-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----';

            await page.locator('[data-testid="ssl-ca-certificate-content"]').fill(testPEM);
            await expect(page.locator('[data-testid="ssl-ca-certificate-content"]')).toHaveValue(testPEM);
        });
    });

    test.describe('SSL Mode Description', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();
            await page.locator('[data-testid="advanced-button"]').click();
        });

        test('shows description text for selected SSL mode', async ({ whodb, page }) => {
            // Select verify-ca mode
            await page.locator('[data-testid="ssl-mode-select"]').click();
            await page.locator('[data-value="verify-ca"]').click();

            // Should show description text
            await expect(page.getByText('Verify server certificate')).toBeVisible();
        });
    });

    test.describe('Database-Specific SSL Modes', () => {
        test('shows Preferred mode for MySQL but not PostgreSQL', async ({ whodb, page }) => {
            // MySQL should have Preferred mode
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="MySQL"]').click();
            await page.locator('[data-testid="advanced-button"]').click();
            await page.locator('[data-testid="ssl-mode-select"]').click();
            await expect(page.locator('[data-value="preferred"]')).toBeAttached();
            await page.keyboard.press('Escape');

            // PostgreSQL should NOT have Preferred mode
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();
            await page.locator('[data-testid="ssl-mode-select"]').click();
            await expect(page.locator('[data-value="preferred"]')).not.toBeAttached();
        });

        test('shows Enabled/Insecure modes for ClickHouse', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="ClickHouse"]').click();
            await page.locator('[data-testid="advanced-button"]').click();
            await page.locator('[data-testid="ssl-mode-select"]').click();

            // ClickHouse uses enabled/insecure instead of verify-ca/verify-identity
            await expect(page.locator('[data-value="enabled"]')).toBeAttached();
            await expect(page.locator('[data-value="insecure"]')).toBeAttached();
            await expect(page.locator('[data-value="verify-ca"]')).not.toBeAttached();
        });
    });

    test.describe('Form Persistence', () => {
        test('maintains SSL configuration when toggling advanced options', async ({ whodb, page }) => {
            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();

            // Open advanced, configure SSL
            await page.locator('[data-testid="advanced-button"]').click();
            await page.locator('[data-testid="ssl-mode-select"]').click();
            await page.locator('[data-value="verify-ca"]').click();

            // Switch to paste mode and enter certificate content
            await page.getByRole('button', { name: 'Paste PEM' }).first().click();
            await page.locator('[data-testid="ssl-ca-certificate-content"]').fill('-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----');

            // Close and reopen advanced options
            await page.locator('[data-testid="advanced-button"]').click();
            await expect(page.locator('[data-testid="ssl-mode-select"]')).not.toBeAttached();

            await page.locator('[data-testid="advanced-button"]').click();

            // SSL mode should be preserved (content may reset due to state management)
            await expect(page.locator('[data-testid="ssl-mode-select"]')).toContainText('Verify CA');
        });
    });
});

/**
 * SSL Integration Tests
 *
 * These tests require SSL-enabled databases to be running.
 * Run with: docker-compose --profile ssl up
 *
 * Tests actual SSL connections through the backend to verify the full stack works.
 * SSL config is now part of main fixtures under the "ssl" key.
 */
test.describe('SSL Integration Tests', () => {
    test.beforeEach(async ({ whodb, page }) => {
        await clearBrowserState(page);
    });

    /**
     * Helper to login with SSL using whodb.login() and proper credential handling.
     * Uses SSL-specific credentials (user/password) when available in db.ssl,
     * otherwise falls back to db.connection credentials.
     */
    async function loginWithSSL(whodb, db, sslMode) {
        const ssl = db.ssl;
        const conn = db.connection;

        // Use SSL-specific credentials if provided, otherwise fall back to connection credentials
        // Pass undefined instead of null to skip fields that don't exist (Redis has no user/database)
        const sslUser = (ssl.user ?? conn.user) ?? undefined;
        const sslPassword = ssl.password ?? conn.password;
        const database = conn.database ?? undefined;

        const hostPath = containerPathToHostPath(ssl.caCertPath);

        let caCertContent = undefined;
        if (hostPath) {
            try {
                caCertContent = readFileSync(hostPath, 'utf-8');
            } catch {
                // cert file not available
            }
        }

        await whodb.login(
            db.type,
            conn.host,
            sslUser,
            sslPassword,
            database,
            {
                Port: String(ssl.port),
                ssl: { mode: sslMode, ...(caCertContent ? { caCertContent } : {}) }
            }
        );
    }

    // SSL mode mapping: databases use different modes for certificate verification
    const sslModeMap = {
        postgres: 'verify-ca',
        mysql: 'verify-ca',
        mariadb: 'verify-ca',
        mongodb: 'enabled',
        redis: 'enabled',
        clickhouse: 'enabled',
        elasticsearch: 'enabled'
    };

    // Test SSL connections for each database type
    ['postgres', 'mysql', 'mariadb', 'mongodb', 'redis', 'clickhouse', 'elasticsearch'].forEach((dbName) => {
        test.describe(`${dbName.charAt(0).toUpperCase() + dbName.slice(1)} SSL Connection`, () => {
            test(`connects to ${dbName} with SSL ${sslModeMap[dbName]} mode`, async ({ whodb, page }) => {
                let db;
                try {
                    db = getDatabaseConfig(dbName);
                } catch {
                    // Database config not available
                    return;
                }
                if (!db.ssl) {
                    // Skipping: SSL config not available
                    return;
                }

                await loginWithSSL(whodb, db, sslModeMap[dbName]);
            });
        });
    });

    test.describe('SSL Status Badge', () => {
        test('shows SSL badge in sidebar when connected with SSL', async ({ whodb, page }) => {
            let db;
            try {
                db = getDatabaseConfig('postgres');
            } catch {
                return;
            }
            if (!db.ssl) {
                // Skipping: postgres SSL config not available
                return;
            }

            const hostPath = containerPathToHostPath(db.ssl.caCertPath);
            let caCertContent = undefined;
            if (hostPath) {
                try {
                    caCertContent = readFileSync(hostPath, 'utf-8');
                } catch {
                    // cert file not available
                }
            }

            await whodb.login(
                db.type,
                db.connection.host,
                db.connection.user,
                db.connection.password,
                db.connection.database,
                {
                    Port: String(db.ssl.port),
                    ssl: { mode: 'verify-ca', ...(caCertContent ? { caCertContent } : {}) }
                }
            );

            // SSL shield badge should appear in the profile selector when SSL is enabled
            const sidebarProfile = page.locator('[data-testid="sidebar-profile"]');
            await sidebarProfile.waitFor({ timeout: 15000 });
            await expect(sidebarProfile.locator('[data-testid="ssl-badge"]')).toBeAttached();
        });

        test('does not show SSL badge when connected without SSL', async ({ whodb, page }) => {
            const db = getDatabaseConfig('postgres');

            await whodb.login(
                db.type,
                db.connection.host,
                db.connection.user,
                db.connection.password,
                db.connection.database
            );

            // Without SSL, there should be no SSL badge
            const sidebarProfile = page.locator('[data-testid="sidebar-profile"]');
            await sidebarProfile.waitFor({ timeout: 15000 });
            await expect(sidebarProfile.locator('[data-testid="ssl-badge"]')).not.toBeAttached();
        });
    });

    test.describe('SSL Connection Failure Cases', () => {
        test('fails to connect with wrong SSL mode', async ({ whodb, page }) => {
            let db;
            try {
                db = getDatabaseConfig('postgres');
            } catch {
                return;
            }
            if (!db.ssl) {
                // Skipping: postgres SSL config not available
                return;
            }

            // This test verifies graceful error handling when SSL is misconfigured.
            // We use the SSL port but wrong credentials and no SSL config.
            // Note: whodb.login() cannot be used here as it expects success.
            await page.goto(whodb.url('/login'));

            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator('[data-value="Postgres"]').click();

            await page.locator('[data-testid="hostname"]').fill('localhost');
            await page.locator('[data-testid="username"]').fill('wrong_user');
            await page.locator('[data-testid="password"]').fill('wrong_password');
            await page.locator('[data-testid="database"]').fill('test_db');

            await page.locator('[data-testid="advanced-button"]').click();
            await page.locator('[data-testid="Port-input"]').fill('');
            await page.locator('[data-testid="Port-input"]').fill(String(db.ssl.port));

            // Don't configure SSL - connection should fail
            const loginResponsePromise = page.waitForResponse(
                resp => resp.url().includes('/api/query') && resp.request().method() === 'POST',
                { timeout: 30000 }
            );
            await page.locator('[data-testid="login-button"]').click();

            // Wait for the request to complete - we just want to ensure no crash
            await loginResponsePromise;

            // Should remain on login page (connection failed)
            await expect(page).toHaveURL(/\/login/);
        });
    });
});
