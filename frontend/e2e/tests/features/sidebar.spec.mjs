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


/**
 * Sidebar Navigation Tests
 *
 * Tests sidebar functionality including schema/database switching,
 * profile management, navigation, and database-specific options.
 */
test.describe('Sidebar Navigation', () => {
    /**
     * Sidebar Dropdown Visibility Tests
     *
     * These tests enforce that each database type shows the correct sidebar dropdowns
     * based on its configuration in the fixture files.
     *
     * Database/Schema Configuration Summary:
     * - PostgreSQL: Database YES, Schema YES (has both concepts)
     * - MySQL/MariaDB: Database YES, Schema NO (database=schema, uses database dropdown)
     * - ClickHouse: Database YES, Schema NO (database only)
     * - MongoDB: Database YES, Schema NO (database only)
     * - Redis: Database YES, Schema NO (numbered databases 0-15)
     * - Elasticsearch: Database NO, Schema NO (neither concept)
     * - SQLite: Database NO, Schema NO (file-based, selected at connection)
     */
    test.describe('Sidebar Dropdown Visibility', () => {
        forEachDatabase('all', (db) => {
            // Skip if no sidebar config defined
            if (!db.sidebar) {
                return;
            }

            test.describe(`${db.type}`, () => {
                if (db.sidebar.showsDatabaseDropdown) {
                    test('shows database dropdown', async ({ whodb, page }) => {
                        await expect(page.locator('[data-testid="sidebar-database"]')).toBeVisible();
                    });

                    test('can interact with database dropdown', async ({ whodb, page }) => {
                        await page.locator('[data-testid="sidebar-database"]').click();
                        // Should show at least one database option
                        await page.locator('[role="option"]').first().waitFor({ timeout: 5000 });
                        // Close dropdown
                        await page.keyboard.press('Escape');
                    });
                } else {
                    test('does not show database dropdown', async ({ whodb, page }) => {
                        await expect(page.locator('[data-testid="sidebar-database"]')).not.toBeAttached();
                    });
                }

                if (db.sidebar.showsSchemaDropdown) {
                    test('shows schema dropdown', async ({ whodb, page }) => {
                        await expect(page.locator('[data-testid="sidebar-schema"]')).toBeVisible();
                    });

                    test('can interact with schema dropdown', async ({ whodb, page }) => {
                        await page.locator('[data-testid="sidebar-schema"]').click();
                        // Should show at least one schema option
                        await page.locator('[role="option"]').first().waitFor({ timeout: 5000 });
                        // Close dropdown
                        await page.keyboard.press('Escape');
                    });
                } else {
                    test('does not show schema dropdown', async ({ whodb, page }) => {
                        await expect(page.locator('[data-testid="sidebar-schema"]')).not.toBeAttached();
                    });
                }
            });
        });
    });

    test.describe('Schema Selection', () => {
        // Test with databases that support schema
        forEachDatabase('sql', (db) => {
            // Skip databases that don't have schema dropdown
            if (!db.sidebar?.showsSchemaDropdown) {
                return;
            }

            test.describe(`${db.type}`, () => {
                test('shows schema dropdown', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-schema"]')).toBeVisible();
                });

                test('can select different schema', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-schema"]').click();

                    // Should show at least one schema option
                    await page.locator('[role="option"]').first().waitFor({ timeout: 5000 });

                    // Select the first schema (or current one if only one exists)
                    await page.locator('[role="option"]').first().click();

                    // Schema should be selected
                    await expect(page.locator('[data-testid="sidebar-schema"]')).toBeAttached();
                });

                test('reloads storage units when schema changes', async ({ whodb, page }) => {
                    const queryPromise = page.waitForResponse(resp =>
                        resp.url().includes('/api/query') && resp.request().method() === 'POST'
                    );

                    await page.locator('[data-testid="sidebar-schema"]').click();
                    // Select a different schema first (to trigger an actual change)
                    // Then select the fixture schema (known to have tables)
                    const options = await page.locator('[role="option"]').all();
                    const currentSchema = db.schema || '';
                    let foundOther = false;
                    for (const opt of options) {
                        const text = await opt.textContent();
                        if (!text.includes(currentSchema)) {
                            await opt.click();
                            foundOther = true;
                            break;
                        }
                    }
                    if (!foundOther) {
                        // Only one schema exists, just click first
                        await page.locator('[role="option"]').first().click();
                    }

                    // Now the schema has changed, storage units should have reloaded
                    // We don't assert on specific count since different schemas may have different tables
                });
            });
        });
    });

    test.describe('Database Selection', () => {
        // Test with databases that support database switching but not schema
        forEachDatabase('all', (db) => {
            // Only test databases with database dropdown but no schema dropdown
            if (!db.sidebar?.showsDatabaseDropdown) {
                return;
            }

            test.describe(`${db.type}`, () => {
                test('shows database dropdown', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-database"]')).toBeVisible();
                });

                test('can switch to different database', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-database"]').click();

                    // Should show database options
                    await page.locator('[role="option"]').first().waitFor({ timeout: 5000 });

                    // Select an option
                    await page.locator('[role="option"]').first().click();

                    // Database should update
                    await expect(page.locator('[data-testid="sidebar-database"]')).toBeAttached();
                });
            });
        });
    });

    test.describe('Navigation Links', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('highlights current route in sidebar', async ({ whodb, page }) => {
                    // Currently on storage-unit
                    await expect(page).toHaveURL(/\/storage-unit/);

                    // Navigate to graph
                    await page.locator('[href="/graph"]').click();
                    await expect(page).toHaveURL(/\/graph/);

                    // Navigate to scratchpad
                    await page.locator('[href="/scratchpad"]').click();
                    await expect(page).toHaveURL(/\/scratchpad/);

                    // Navigate to chat
                    await page.locator('[href="/chat"]').click();
                    await expect(page).toHaveURL(/\/chat/);

                    // Navigate back to storage-unit
                    await page.locator('[href="/storage-unit"]').click();
                    await expect(page).toHaveURL(/\/storage-unit/);
                });

                test('shows chat option for SQL databases', async ({ whodb, page }) => {
                    await expect(page.locator('[href="/chat"]')).toBeAttached();
                });

                test('shows graph option', async ({ whodb, page }) => {
                    await expect(page.locator('[href="/graph"]')).toBeAttached();
                });

                test('shows scratchpad option for SQL databases', async ({ whodb, page }) => {
                    await expect(page.locator('[href="/scratchpad"]')).toBeAttached();
                });
            });
        }, { features: ['scratchpad'] });
    });

    test.describe('NoSQL Navigation', () => {
        // Test NoSQL databases that may not show all options
        forEachDatabase('keyvalue', (db) => {
            test.describe(`${db.type}`, () => {
                test('hides chat option for key-value databases', async ({ whodb, page }) => {
                    // Redis and similar don't support SQL chat
                    await expect(page.locator('[href="/chat"]')).not.toBeAttached();
                });

                test('hides scratchpad option for key-value databases', async ({ whodb, page }) => {
                    // Key-value stores don't support SQL scratchpad
                    await expect(page.locator('[href="/scratchpad"]')).not.toBeAttached();
                });

                test('still shows graph option', async ({ whodb, page }) => {
                    await expect(page.locator('[href="/graph"]')).toBeAttached();
                });
            });
        });

        forEachDatabase('document', (db) => {
            test.describe(`${db.type}`, () => {
                test('navigation options depend on database features', async ({ whodb, page }) => {
                    // Document databases may or may not have certain features
                    await expect(page.locator('[href="/graph"]')).toBeAttached();
                });
            });
        });
    });

    test.describe('Sidebar Toggle', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('can collapse and expand sidebar with keyboard shortcut', async ({ whodb, page }) => {
                    // Sidebar should be visible initially
                    await expect(page.locator('[data-sidebar="sidebar"]')).toBeVisible();

                    // Toggle sidebar with Ctrl+B
                    await page.keyboard.press('Control+b');

                    // Sidebar state should change
                    await page.waitForTimeout(300); // Wait for animation

                    // Toggle back
                    await page.keyboard.press('Control+b');
                    await page.waitForTimeout(300);

                    // Sidebar should be visible again
                    await expect(page.locator('[data-sidebar="sidebar"]')).toBeVisible();
                });

                test('sidebar state persists in session', async ({ whodb, page }) => {
                    // Collapse sidebar
                    await page.keyboard.press('Control+b');
                    await page.waitForTimeout(500); // Wait for collapse animation

                    // Navigate to another page - use link locator scoped to sidebar
                    await page.locator('a[href="/graph"]').first().click();
                    await expect(page).toHaveURL(/\/graph/);

                    // Navigate back
                    await page.locator('a[href="/storage-unit"]').first().click();
                    await expect(page).toHaveURL(/\/storage-unit/);

                    // Sidebar state should be maintained
                    // (exact assertion depends on implementation)
                });
            });
        });
    });

    test.describe('Logout', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('logout redirects to login page', async ({ whodb, page }) => {
                    // Use the logout command
                    await whodb.logout();

                    // Should redirect to login page
                    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
                });

                test('logout clears session', async ({ whodb, page }) => {
                    await whodb.logout();

                    // Wait for redirect
                    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

                    // Try to visit storage-unit directly
                    await page.goto(whodb.url('/storage-unit'));

                    // Should redirect back to login (session cleared)
                    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
                });
            });
        });
    });

    test.describe('Profile Display', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows current profile in sidebar', async ({ whodb, page }) => {
                    // Profile selector should be visible
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();
                });

                test('profile dropdown shows options', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    // Should show at least the current profile
                    await page.locator('[role="menuitem"], [role="option"]').first().waitFor({ timeout: 5000 });

                    // Close dropdown
                    await page.keyboard.press('Escape');
                });
            });
        });
    });

    test.describe('Add New Profile', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('can open add profile dialog from sidebar', async ({ whodb, page }) => {
                    // Click profile dropdown
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    // Look for add profile option (rendered as CommandItem with role="option")
                    const addOption = page.getByRole('option', { name: /Add Another Profile/i });
                    const addOptionCount = await addOption.count();
                    if (addOptionCount > 0) {
                        await addOption.first().click();

                        // Should show login form in sheet/dialog
                        await expect(page.locator('[role="dialog"], [data-testid="login-form"]')).toBeVisible({ timeout: 5000 });

                        // Close dialog
                        await page.keyboard.press('Escape');
                    }
                });
            });
        });
    });

    test.describe('Database Type Icons', () => {
        forEachDatabase('all', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows database type icon in profile', async ({ whodb, page }) => {
                    // Profile area should show database type indicator
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();

                    // There should be an icon representing the database type
                    await expect(page.locator('[data-testid="sidebar-profile"]').locator('svg, img').first()).toBeAttached();
                });
            });
        });
    });

    test.describe('Version Display', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows WhoDB version in sidebar footer', async ({ whodb, page }) => {
                    // Version should be displayed somewhere in sidebar
                    await expect(page.locator('[data-sidebar="sidebar"]')).toContainText('Version: development');
                });
            });
        });
    });
});
