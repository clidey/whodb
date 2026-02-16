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

const targetDb = process.env.DATABASE;
const shouldRun = !targetDb || targetDb.toLowerCase() === 'postgres';

/**
 * Browser Storage Tests
 *
 * Tests localStorage, sessionStorage, and Redux persist functionality
 * including state persistence across page reloads, logout behavior,
 * and storage key management.
 */
const describeOrSkip = shouldRun ? test.describe : test.describe.skip;

describeOrSkip('Browser Storage', () => {
    const db = getDatabaseConfig('postgres');

    test.beforeEach(async ({ whodb, page }) => {
        const conn = db.connection;
        await whodb.login(
            db.uiType || db.type,
            conn.host ?? undefined,
            conn.user ?? undefined,
            conn.password ?? undefined,
            conn.database ?? undefined,
            conn.advanced || {}
        );
    });

    test.afterEach(async ({ whodb, page }) => {
        await whodb.logout();
    });

    test.describe('Redux State Persistence', () => {
        test('persists auth state across page reload', async ({ whodb, page }) => {
            // Verify we're logged in
            await expect(page).toHaveURL(/\/storage-unit/);

            // Check that auth state exists in localStorage
            const authData = await page.evaluate(() => localStorage.getItem('persist:auth'));
            expect(authData).not.toBeNull();

            const parsed = JSON.parse(authData);
            expect(parsed.status).toBeDefined();
            expect(JSON.parse(parsed.status)).toEqual('logged-in');
            expect(parsed.profiles).toBeDefined();
            expect(parsed.current).toBeDefined();

            // Reload the page
            await page.reload();

            // Should still be logged in without redirecting to login
            await expect(page).toHaveURL(/\/storage-unit/, { timeout: 10000 });
            await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();
        });

        test('persists database state across page reload', async ({ whodb, page }) => {
            // Check that database state exists in localStorage
            const databaseData = await page.evaluate(() => localStorage.getItem('persist:database'));
            expect(databaseData).not.toBeNull();

            const parsed = JSON.parse(databaseData);
            expect(parsed).toHaveProperty('_persist');

            // Reload the page
            await page.reload();

            // Database state should still be available
            await expect(page.locator('[data-testid="sidebar-profile"]')).toBeAttached();
        });

        test('persists settings state across page reload', async ({ whodb, page }) => {
            // Navigate to storage-unit and set a preference
            await page.goto(whodb.url('/storage-unit'));

            // Change storage unit view to list via localStorage
            await page.evaluate(() => {
                const settingsData = localStorage.getItem('persist:settings');
                const parsed = JSON.parse(settingsData || '{}');
                parsed.storageUnitView = '"list"';
                localStorage.setItem('persist:settings', JSON.stringify(parsed));
            });

            // Reload the page
            await page.reload();

            // Setting should persist
            const storageUnitView = await page.evaluate(() => {
                const settingsData = localStorage.getItem('persist:settings');
                const parsed = JSON.parse(settingsData);
                return JSON.parse(parsed.storageUnitView);
            });
            expect(storageUnitView).toEqual('list');
        });

        test('persists all Redux slices with correct keys', async ({ whodb, page }) => {
            const result = await page.evaluate(() => {
                // Check that all expected persist keys exist
                const expectedKeys = [
                    'persist:auth',
                    'persist:database',
                    'persist:settings',
                    'persist:houdini',
                    'persist:aiModels',
                    'persist:scratchpad',
                    'persist:tour',
                    'persist:databaseMetadata'
                ];

                const results = {};
                expectedKeys.forEach(key => {
                    const data = localStorage.getItem(key);
                    results[key] = {
                        exists: data !== null,
                        hasPersist: false
                    };
                    if (data !== null) {
                        try {
                            const parsed = JSON.parse(data);
                            results[key].hasPersist = parsed._persist !== undefined;
                        } catch {
                            // ignore
                        }
                    }
                });
                return results;
            });

            for (const [key, value] of Object.entries(result)) {
                expect(value.exists, `${key} should exist in localStorage`).toBeTruthy();
                expect(value.hasPersist, `${key} should have _persist`).toBeTruthy();
            }
        });

        test('maintains Redux state structure after reload', async ({ whodb, page }) => {
            // Capture state before reload
            const beforeReload = await page.evaluate(() => ({
                auth: localStorage.getItem('persist:auth'),
                database: localStorage.getItem('persist:database')
            }));

            // Reload the page
            await page.reload();

            // Verify state structure is maintained
            const afterReload = await page.evaluate(() => ({
                auth: localStorage.getItem('persist:auth'),
                database: localStorage.getItem('persist:database')
            }));

            expect(JSON.parse(afterReload.auth)).toEqual(JSON.parse(beforeReload.auth));
            expect(JSON.parse(afterReload.database)).toEqual(JSON.parse(beforeReload.database));
        });
    });

    test.describe('Settings Persistence', () => {
        test('persists storage unit view preference', async ({ whodb, page }) => {
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
                settings.storageUnitView = '"list"';
                localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            await page.reload();

            const view = await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings'));
                return JSON.parse(settings.storageUnitView);
            });
            expect(view).toEqual('list');
        });

        test('persists font size preference', async ({ whodb, page }) => {
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
                settings.fontSize = '"large"';
                localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            await page.reload();

            const fontSize = await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings'));
                return JSON.parse(settings.fontSize);
            });
            expect(fontSize).toEqual('large');
        });

        test('persists border radius preference', async ({ whodb, page }) => {
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
                settings.borderRadius = '"none"';
                localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            await page.reload();

            const borderRadius = await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings'));
                return JSON.parse(settings.borderRadius);
            });
            expect(borderRadius).toEqual('none');
        });

        test('persists spacing preference', async ({ whodb, page }) => {
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
                settings.spacing = '"spacious"';
                localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            await page.reload();

            const spacing = await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings'));
                return JSON.parse(settings.spacing);
            });
            expect(spacing).toEqual('spacious');
        });

        test('persists where condition mode preference', async ({ whodb, page }) => {
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
                settings.whereConditionMode = '"sheet"';
                localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            await page.reload();

            const whereConditionMode = await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings'));
                return JSON.parse(settings.whereConditionMode);
            });
            expect(whereConditionMode).toEqual('sheet');
        });

        test('persists default page size preference', async ({ whodb, page }) => {
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
                settings.defaultPageSize = '50';
                localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            await page.reload();

            const defaultPageSize = await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings'));
                return JSON.parse(settings.defaultPageSize);
            });
            expect(defaultPageSize).toEqual(50);
        });

        test('persists language preference', async ({ whodb, page }) => {
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
                settings.language = '"es"';
                localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            await page.reload();

            const language = await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings'));
                return JSON.parse(settings.language);
            });
            expect(language).toEqual('es');
        });

        test('persists metrics enabled preference', async ({ whodb, page }) => {
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
                settings.metricsEnabled = 'false';
                localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            await page.reload();

            const metricsEnabled = await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings'));
                return JSON.parse(settings.metricsEnabled);
            });
            expect(metricsEnabled).toEqual(false);
        });
    });

    test.describe('Logout Storage Cleanup', () => {
        test('clears Redux auth state on logout', async ({ whodb, page }) => {
            // Verify auth state exists before logout
            const statusBefore = await page.evaluate(() => {
                const authData = localStorage.getItem('persist:auth');
                const parsed = JSON.parse(authData);
                return JSON.parse(parsed.status);
            });
            expect(statusBefore).toEqual('logged-in');

            // Logout
            await whodb.logout();
            await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

            // Auth state should be cleared (status = unauthorized, profiles empty)
            const result = await page.evaluate(() => {
                const authData = localStorage.getItem('persist:auth');
                if (authData) {
                    const parsed = JSON.parse(authData);
                    return {
                        status: JSON.parse(parsed.status),
                        profileCount: JSON.parse(parsed.profiles).length
                    };
                }
                return null;
            });
            if (result) {
                expect(result.status).toEqual('unauthorized');
                expect(result.profileCount).toEqual(0);
            }
        });

        test('preserves settings after logout', async ({ whodb, page }) => {
            // Set a custom setting
            await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
                settings.storageUnitView = '"list"';
                localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            // Logout
            await whodb.logout();
            await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

            // Settings should still exist
            const view = await page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem('persist:settings'));
                return JSON.parse(settings.storageUnitView);
            });
            expect(view).toEqual('list');
        });

        test('preserves first login flag after logout', async ({ whodb, page }) => {
            // First login flag should be set after initial login
            const hasLoggedIn = await page.evaluate(() => localStorage.getItem('whodb_has_logged_in'));
            expect(hasLoggedIn).toEqual('true');

            // Logout
            await whodb.logout();
            await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

            // First login flag should persist
            const hasLoggedInAfter = await page.evaluate(() => localStorage.getItem('whodb_has_logged_in'));
            expect(hasLoggedInAfter).toEqual('true');
        });

        test('clears current profile but preserves persist keys', async ({ whodb, page }) => {
            // Logout
            await whodb.logout();
            await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

            // Redux persist keys should still exist but with cleared data
            const result = await page.evaluate(() => {
                const authData = localStorage.getItem('persist:auth');
                if (authData) {
                    const parsed = JSON.parse(authData);
                    return { hasPersist: parsed._persist !== undefined };
                }
                return { hasPersist: false };
            });
            expect(result.hasPersist).toBeTruthy();
        });

        test('prevents access to protected routes after logout', async ({ whodb, page }) => {
            // Logout
            await whodb.logout();
            await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

            // Try to visit a protected route
            await page.goto(whodb.url('/storage-unit'));

            // Should redirect back to login
            await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
        });
    });

    test.describe('First Login Flag', () => {
        test('sets first login flag on initial login', async ({ whodb, page }) => {
            await clearBrowserState(page);

            // Clear the first login flag specifically
            await page.evaluate(() => {
                localStorage.removeItem('whodb_has_logged_in');
            });

            // Login
            const conn = db.connection;
            await whodb.login(
                db.uiType || db.type,
                conn.host ?? undefined,
                conn.user ?? undefined,
                conn.password ?? undefined,
                conn.database ?? undefined,
                conn.advanced || {}
            );

            // First login flag should be set
            const hasLoggedIn = await page.evaluate(() => localStorage.getItem('whodb_has_logged_in'));
            expect(hasLoggedIn).toEqual('true');
        });

        test('first login flag persists across page reloads', async ({ whodb, page }) => {
            await clearBrowserState(page);

            const conn = db.connection;
            await whodb.login(
                db.uiType || db.type,
                conn.host ?? undefined,
                conn.user ?? undefined,
                conn.password ?? undefined,
                conn.database ?? undefined,
                conn.advanced || {}
            );

            // Verify flag is set
            const hasLoggedIn = await page.evaluate(() => localStorage.getItem('whodb_has_logged_in'));
            expect(hasLoggedIn).toEqual('true');

            // Reload the page
            await page.reload();

            // Flag should still be set
            const hasLoggedInAfter = await page.evaluate(() => localStorage.getItem('whodb_has_logged_in'));
            expect(hasLoggedInAfter).toEqual('true');
        });

        test('first login flag persists after logout', async ({ whodb, page }) => {
            await clearBrowserState(page);

            const conn = db.connection;
            await whodb.login(
                db.uiType || db.type,
                conn.host ?? undefined,
                conn.user ?? undefined,
                conn.password ?? undefined,
                conn.database ?? undefined,
                conn.advanced || {}
            );

            // Logout
            await whodb.logout();
            await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

            // Flag should still be set
            const hasLoggedIn = await page.evaluate(() => localStorage.getItem('whodb_has_logged_in'));
            expect(hasLoggedIn).toEqual('true');
        });
    });

    test.describe('Analytics Consent', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
        });

        test('analytics consent is set to denied by clearBrowserState', async ({ whodb, page }) => {
            const consent = await page.evaluate(() => localStorage.getItem('whodb.analytics.consent'));
            expect(consent).toEqual('denied');
        });

        test('analytics consent persists across page navigation', async ({ whodb, page }) => {
            const conn = db.connection;
            await whodb.login(
                db.uiType || db.type,
                conn.host ?? undefined,
                conn.user ?? undefined,
                conn.password ?? undefined,
                conn.database ?? undefined,
                conn.advanced || {}
            );

            const consent = await page.evaluate(() => localStorage.getItem('whodb.analytics.consent'));
            expect(consent).toEqual('denied');

            // Navigate to another page
            await page.goto(whodb.url('/graph'));

            const consentAfter = await page.evaluate(() => localStorage.getItem('whodb.analytics.consent'));
            expect(consentAfter).toEqual('denied');
        });
    });

    test.describe('Sidebar State Persistence', () => {
        test('sidebar toggle state persists during navigation', async ({ whodb, page }) => {
            // Sidebar should be visible initially
            await expect(page.locator('[data-sidebar="sidebar"]')).toBeVisible();

            // Toggle sidebar
            await page.keyboard.press('Control+b');
            await page.waitForTimeout(300);

            // Navigate to graph
            await page.goto(whodb.url('/graph'));
            await page.waitForTimeout(500);

            // Navigate back to storage-unit
            await page.goto(whodb.url('/storage-unit'));
            await page.waitForTimeout(500);

            // Sidebar state is maintained through navigation
            await expect(page.locator('[data-sidebar="sidebar"]')).toBeAttached();
        });

        test('sidebar can be toggled multiple times', async ({ whodb, page }) => {
            // Initial state - visible
            await expect(page.locator('[data-sidebar="sidebar"]')).toBeVisible();

            // Toggle off
            await page.keyboard.press('Control+b');
            await page.waitForTimeout(300);

            // Toggle on
            await page.keyboard.press('Control+b');
            await page.waitForTimeout(300);

            // Should be visible again
            await expect(page.locator('[data-sidebar="sidebar"]')).toBeVisible();
        });
    });

    test.describe('Storage Size and Limits', () => {
        test('localStorage contains expected data after login', async ({ whodb, page }) => {
            const result = await page.evaluate(() => {
                const storageKeys = Object.keys(localStorage);
                const persistKeys = storageKeys.filter(key => key.startsWith('persist:'));
                return {
                    persistKeyCount: persistKeys.length,
                    hasAnalyticsConsent: storageKeys.includes('whodb.analytics.consent'),
                    hasFirstLoginFlag: storageKeys.includes('whodb_has_logged_in')
                };
            });

            // Should have multiple persist keys
            expect(result.persistKeyCount).toBeGreaterThan(5);

            // Should have first login flag
            expect(result.hasFirstLoginFlag).toBeTruthy();
        });

        test('localStorage data is valid JSON', async ({ whodb, page }) => {
            const invalidKeys = await page.evaluate(() => {
                const storageKeys = Object.keys(localStorage);
                const invalid = [];
                storageKeys.forEach(key => {
                    if (key.startsWith('persist:')) {
                        const data = localStorage.getItem(key);
                        try {
                            JSON.parse(data);
                        } catch {
                            invalid.push(key);
                        }
                    }
                });
                return invalid;
            });

            expect(invalidKeys).toEqual([]);
        });

        test('sessionStorage is not used for persistence', async ({ whodb, page }) => {
            const persistKeyCount = await page.evaluate(() => {
                const sessionKeys = Object.keys(sessionStorage);
                return sessionKeys.filter(key => key.startsWith('persist:')).length;
            });

            // Should have no persist keys in sessionStorage
            expect(persistKeyCount).toEqual(0);
        });
    });

    test.describe('Profile Management Storage', () => {
        test('stores profile information in auth state', async ({ whodb, page }) => {
            const result = await page.evaluate(() => {
                const authData = JSON.parse(localStorage.getItem('persist:auth'));
                const profiles = JSON.parse(authData.profiles);
                if (profiles.length > 0) {
                    const profile = profiles[0];
                    return {
                        count: profiles.length,
                        hasId: profile.Id !== undefined,
                        hasType: profile.Type !== undefined,
                        hasHostname: profile.Hostname !== undefined
                    };
                }
                return { count: 0 };
            });

            // Should have at least one profile
            expect(result.count).toBeGreaterThan(0);

            // Each profile should have required fields
            expect(result.hasId).toBeTruthy();
            expect(result.hasType).toBeTruthy();
            expect(result.hasHostname).toBeTruthy();
        });

        test('stores current profile in auth state', async ({ whodb, page }) => {
            const result = await page.evaluate(() => {
                const authData = JSON.parse(localStorage.getItem('persist:auth'));
                const current = JSON.parse(authData.current);
                return {
                    isNotNull: current !== null,
                    hasId: current?.Id !== undefined,
                    hasType: current?.Type !== undefined
                };
            });

            // Current profile should exist
            expect(result.isNotNull).toBeTruthy();
            expect(result.hasId).toBeTruthy();
            expect(result.hasType).toBeTruthy();
        });

        test('profile data persists after navigation', async ({ whodb, page }) => {
            // Get the current profile ID
            const profileId = await page.evaluate(() => {
                const authData = JSON.parse(localStorage.getItem('persist:auth'));
                const current = JSON.parse(authData.current);
                return current.Id;
            });

            // Navigate to another page
            await page.goto(whodb.url('/graph'));
            await page.waitForTimeout(500);

            // Profile should remain the same
            const profileIdAfter = await page.evaluate(() => {
                const authData = JSON.parse(localStorage.getItem('persist:auth'));
                const current = JSON.parse(authData.current);
                return current.Id;
            });
            expect(profileIdAfter).toEqual(profileId);
        });
    });

    test.describe('Corrupted Data Handling', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
        });

        test('handles corrupted Redux persist data gracefully', async ({ whodb, page }) => {
            // Set corrupted data
            await page.evaluate(() => {
                localStorage.setItem('persist:settings', 'invalid json{');
            });

            // Login should still work (Redux persist will reset corrupted data)
            const conn = db.connection;
            await whodb.login(
                db.uiType || db.type,
                conn.host ?? undefined,
                conn.user ?? undefined,
                conn.password ?? undefined,
                conn.database ?? undefined,
                conn.advanced || {}
            );

            // Should be logged in successfully
            await expect(page).toHaveURL(/\/storage-unit/);
        });

        test('clears corrupted scratchpad data on load', async ({ whodb, page }) => {
            // Set corrupted scratchpad data with invalid dates
            await page.evaluate(() => {
                const corruptedData = {
                    cells: {
                        'cell-1': {
                            history: [
                                { date: 'invalid-date', query: 'SELECT 1' }
                            ]
                        }
                    },
                    _persist: '{"version":-1,"rehydrated":true}'
                };
                localStorage.setItem('persist:scratchpad', JSON.stringify(corruptedData));
            });

            // Visit the app - should handle corrupted data
            await page.goto(whodb.url('/login'));

            // Should not crash
            await expect(page.locator('[data-testid="database-type-select"]')).toBeAttached();
        });
    });
});
