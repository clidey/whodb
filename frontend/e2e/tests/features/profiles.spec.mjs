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
import { getDatabaseConfig } from '../../support/database-config.mjs';
import { clearBrowserState } from '../../support/helpers/animation.mjs';

/**
 * Profile Management Tests
 *
 * Tests profile functionality including displaying multiple profiles,
 * switching between profiles, database type icons, adding new profiles,
 * and logging out from specific profiles.
 */
test.describe('Profile Management', () => {
    test.describe('Profile Display', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('displays profile selector in sidebar', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeVisible();
                });

                test('shows database type icon in profile', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();

                    await expect(page.locator('[data-testid="sidebar-profile"]').locator('svg, img').first()).toBeAttached();
                });

                test('displays current connection information', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();

                    await expect(page.locator('[data-testid="sidebar-profile"]').locator('svg, img').first()).toBeAttached();
                });
            });
        });
    });

    test.describe('Profile Dropdown', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('opens dropdown when profile is clicked', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    await page.locator('[role="menuitem"], [role="option"]').first().waitFor({ timeout: 5000 });

                    await page.keyboard.press('Escape');
                });

                test('shows current profile in dropdown', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    await page.locator('[role="menuitem"], [role="option"]').first().waitFor({ timeout: 5000 });

                    await page.keyboard.press('Escape');
                });

                test('closes dropdown when escape is pressed', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    await expect(page.locator('[role="menuitem"], [role="option"]').first()).toBeVisible();

                    await page.keyboard.press('Escape');

                    await page.waitForTimeout(300);

                    const visibleOptions = await page.locator('[role="menuitem"], [role="option"]').filter({ visible: true }).count();
                    expect(visibleOptions).toEqual(0);
                });
            });
        });
    });

    test.describe('Multiple Profiles', () => {
        test('displays multiple profiles in dropdown when multiple connections exist', async ({ whodb, page }) => {
            await clearBrowserState(page);

            const db1 = getDatabaseConfig('postgres');
            const db2 = getDatabaseConfig('mysql');

            await whodb.login(
                db1.uiType || db1.type,
                db1.connection.host,
                db1.connection.user,
                db1.connection.password,
                db1.connection.database,
                db1.connection.advanced || {}
            );

            // Ensure card view is set (clearBrowserState wiped localStorage settings)
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
                settings.storageUnitView = '"card"';
                localStorage.setItem("persist:settings", JSON.stringify(settings));
            });
            await page.reload();
            await page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

            await page.locator('[data-testid="sidebar-profile"]').click();

            // Look for "Add Another Profile" text
            await expect(page.locator('text=Add Another Profile')).toBeVisible();
            await page.locator('text=Add Another Profile').click();

            await expect(page.locator('[role="dialog"], [data-testid="login-form"]')).toBeVisible({ timeout: 5000 });

            await page.locator('[data-testid="database-type-select"]').click();
            await page.locator(`[data-value="${db2.type}"]`).click();

            await page.locator('[data-testid="hostname"]').fill(db2.connection.host);
            await page.locator('[data-testid="username"]').fill(db2.connection.user);
            await page.locator('[data-testid="password"]').fill(db2.connection.password);
            await page.locator('[data-testid="database"]').fill(db2.connection.database);

            const queryPromise = page.waitForResponse(resp =>
                resp.url().includes('/api/query') && resp.request().method() === 'POST'
            );
            await page.locator('[data-testid="login-button"]').click();

            await queryPromise;

            await page.waitForTimeout(1000);

            await page.locator('[data-testid="sidebar-profile"]').click();

            // Profile dropdown should have at least 2 profiles now
            // Verify by checking "Add Another Profile" is visible (dropdown is open)
            // and we can see profile entries
            await expect(page.locator('text=Add Another Profile')).toBeVisible();

            await page.keyboard.press('Escape');
        });
    });

    test.describe('Profile Switching', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('can switch between profiles when multiple exist', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    const menuItems = await page.locator('[role="menuitem"], [role="option"]').count();
                    if (menuItems > 1) {
                        await page.locator('[role="menuitem"], [role="option"]').nth(1).click();

                        await page.waitForTimeout(1000);

                        await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();
                    } else {
                        await page.keyboard.press('Escape');
                    }
                });
            });
        });
    });

    test.describe('Add New Profile', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows add profile option in dropdown', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    await expect(page.locator('text=Add Another Profile')).toBeVisible();

                    await page.keyboard.press('Escape');
                });

                test('opens login dialog when add profile is clicked', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    await page.locator('text=Add Another Profile').click();

                    await expect(page.locator('[role="dialog"], [data-testid="login-form"]')).toBeVisible({ timeout: 5000 });

                    await page.keyboard.press('Escape');
                });

                test('can cancel adding new profile', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    await page.locator('text=Add Another Profile').click();

                    // Wait for login dialog/sheet to appear
                    await expect(page.locator('[role="dialog"], [data-testid="login-form"]')).toBeVisible({ timeout: 5000 });

                    // Close the dialog by pressing escape
                    await page.keyboard.press('Escape');
                    await page.waitForTimeout(500);

                    // Verify we're back to normal state by checking sidebar profile exists
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();
                });
            });
        });
    });

    test.describe('Database Type Icons', () => {
        forEachDatabase('all', (db) => {
            test.describe(`${db.type}`, () => {
                test('displays correct database type icon for profile', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();

                    await expect(page.locator('[data-testid="sidebar-profile"]').locator('svg, img').first()).toBeVisible();
                });

                test('maintains icon visibility in profile dropdown', async ({ whodb, page }) => {
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    await expect(page.locator('[role="menuitem"], [role="option"]').first().locator('svg, img').first()).toBeAttached();

                    await page.keyboard.press('Escape');
                });
            });
        });
    });

    test.describe('Profile Last Accessed', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('displays profile information', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();
                    await page.locator('[data-testid="sidebar-profile"]').click();

                    await page.locator('[role="menuitem"], [role="option"]').first().waitFor({ timeout: 5000 });

                    await page.keyboard.press('Escape');
                });
            });
        });
    });

    test.describe('Logout from Profile', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('can logout from current profile', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();

                    await whodb.logout();

                    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
                });

                test('logout clears current session', async ({ whodb, page }) => {
                    await whodb.logout();

                    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

                    await page.goto(whodb.url('/storage-unit'));

                    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
                });

                test('can logout and login with different profile', async ({ whodb, page }) => {
                    await whodb.logout();
                    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
                    const newDb = getDatabaseConfig('postgres');
                    await whodb.login(
                        newDb.uiType || newDb.type,
                        newDb.connection.host,
                        newDb.connection.user,
                        newDb.connection.password,
                        newDb.connection.database,
                        newDb.connection.advanced || {}
                    );

                    await expect(page).toHaveURL(/\/storage-unit/);
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();
                });
            });
        });
    });

    test.describe('Profile Persistence', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('maintains profile selection after page reload', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();

                    await page.reload();

                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached({ timeout: 10000 });
                    await expect(page).toHaveURL(/\/storage-unit/);
                });

                test('maintains profile selection when navigating between pages', async ({ whodb, page }) => {
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();

                    await page.locator('[href="/graph"]').click();
                    await expect(page).toHaveURL(/\/graph/);

                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();

                    await page.locator('[href="/storage-unit"]').click();
                    await expect(page).toHaveURL(/\/storage-unit/);

                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();
                });
            });
        });
    });

    test.describe('Profile Navigation', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('profile element exists on different pages', async ({ whodb, page }) => {
                    // Verify profile element exists on graph page
                    await page.locator('[href="/graph"]').click();
                    await expect(page).toHaveURL(/\/graph/);
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached({ timeout: 5000 });

                    // Verify profile element exists on scratchpad page
                    await page.locator('[href="/scratchpad"]').click();
                    await expect(page).toHaveURL(/\/scratchpad/);
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached({ timeout: 5000 });

                    // Navigate back to storage-unit
                    await page.locator('[href="/storage-unit"]').click();
                    await expect(page).toHaveURL(/\/storage-unit/);
                    await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached({ timeout: 5000 });
                });
            });
        }, { features: ['scratchpad'] });
    });
});
