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

// Helper to assert a setting value in localStorage (redux-persist format)
const assertPersistedSetting = async (page, key, expectedValue) => {
    const value = await page.evaluate(([k]) => {
        const persistedSettings = localStorage.getItem('persist:settings');
        if (!persistedSettings) return null;
        const parsed = JSON.parse(persistedSettings);
        // Redux-persist double-encodes values as JSON strings
        return parsed[k] ? JSON.parse(parsed[k]) : null;
    }, [key]);
    expect(value).toEqual(expectedValue);
};

test.describe('Settings', () => {

    // Run settings tests for one database only (settings are global)
    forEachDatabase('sql', (db) => {
        // Only run for first SQL database (postgres) to avoid redundant tests
        if (db.type !== 'Postgres') {
            return;
        }

        test.beforeEach(async ({ whodb, page }) => {
            await whodb.goto('settings');
        });

        test.describe('Font Size', () => {
            test('can change font size to small', async ({ whodb, page }) => {
                // First set to medium to ensure we're changing from a known state
                await page.locator('#font-size').click();
                await page.locator('[data-value="medium"]').click();
                await assertPersistedSetting(page, 'fontSize', 'medium');

                // Now change to small
                await page.locator('#font-size').click();
                await page.locator('[data-value="small"]').click();
                await expect(page.locator('#font-size')).toContainText('Small');
                await assertPersistedSetting(page, 'fontSize', 'small');
            });

            test('can change font size to medium', async ({ whodb, page }) => {
                // First set to small to ensure we're changing from a known state
                await page.locator('#font-size').click();
                await page.locator('[data-value="small"]').click();
                await assertPersistedSetting(page, 'fontSize', 'small');

                // Now change to medium
                await page.locator('#font-size').click();
                await page.locator('[data-value="medium"]').click();
                await expect(page.locator('#font-size')).toContainText('Medium');
                await assertPersistedSetting(page, 'fontSize', 'medium');
            });

            test('can change font size to large', async ({ whodb, page }) => {
                // First set to medium to ensure we're changing from a known state
                await page.locator('#font-size').click();
                await page.locator('[data-value="medium"]').click();
                await assertPersistedSetting(page, 'fontSize', 'medium');

                // Now change to large
                await page.locator('#font-size').click();
                await page.locator('[data-value="large"]').click();
                await expect(page.locator('#font-size')).toContainText('Large');
                await assertPersistedSetting(page, 'fontSize', 'large');
            });
        });

        test.describe('Border Radius', () => {
            test('can change border radius to none', async ({ whodb, page }) => {
                // First set to medium to ensure we're changing from a known state
                await page.locator('#border-radius').click();
                await page.locator('[data-value="medium"]').click();
                await assertPersistedSetting(page, 'borderRadius', 'medium');

                // Now change to none
                await page.locator('#border-radius').click();
                await page.locator('[data-value="none"]').click();
                await expect(page.locator('#border-radius')).toContainText('None');
                await assertPersistedSetting(page, 'borderRadius', 'none');
            });

            test('can change border radius to small', async ({ whodb, page }) => {
                // First set to none to ensure we're changing from a known state
                await page.locator('#border-radius').click();
                await page.locator('[data-value="none"]').click();
                await assertPersistedSetting(page, 'borderRadius', 'none');

                // Now change to small
                await page.locator('#border-radius').click();
                await page.locator('[data-value="small"]').click();
                await expect(page.locator('#border-radius')).toContainText('Small');
                await assertPersistedSetting(page, 'borderRadius', 'small');
            });

            test('can change border radius to medium', async ({ whodb, page }) => {
                // First set to small to ensure we're changing from a known state
                await page.locator('#border-radius').click();
                await page.locator('[data-value="small"]').click();
                await assertPersistedSetting(page, 'borderRadius', 'small');

                // Now change to medium
                await page.locator('#border-radius').click();
                await page.locator('[data-value="medium"]').click();
                await expect(page.locator('#border-radius')).toContainText('Medium');
                await assertPersistedSetting(page, 'borderRadius', 'medium');
            });

            test('can change border radius to large', async ({ whodb, page }) => {
                // First set to medium to ensure we're changing from a known state
                await page.locator('#border-radius').click();
                await page.locator('[data-value="medium"]').click();
                await assertPersistedSetting(page, 'borderRadius', 'medium');

                // Now change to large
                await page.locator('#border-radius').click();
                await page.locator('[data-value="large"]').click();
                await expect(page.locator('#border-radius')).toContainText('Large');
                await assertPersistedSetting(page, 'borderRadius', 'large');
            });
        });

        test.describe('Spacing', () => {
            test('can change spacing to compact', async ({ whodb, page }) => {
                // First set to comfortable to ensure we're changing from a known state
                await page.locator('#spacing').click();
                await page.locator('[data-value="comfortable"]').click();
                await assertPersistedSetting(page, 'spacing', 'comfortable');

                // Now change to compact
                await page.locator('#spacing').click();
                await page.locator('[data-value="compact"]').click();
                await expect(page.locator('#spacing')).toContainText('Compact');
                await assertPersistedSetting(page, 'spacing', 'compact');
            });

            test('can change spacing to comfortable', async ({ whodb, page }) => {
                // First set to compact to ensure we're changing from a known state
                await page.locator('#spacing').click();
                await page.locator('[data-value="compact"]').click();
                await assertPersistedSetting(page, 'spacing', 'compact');

                // Now change to comfortable
                await page.locator('#spacing').click();
                await page.locator('[data-value="comfortable"]').click();
                await expect(page.locator('#spacing')).toContainText('Comfortable');
                await assertPersistedSetting(page, 'spacing', 'comfortable');
            });

            test('can change spacing to spacious', async ({ whodb, page }) => {
                // First set to comfortable to ensure we're changing from a known state
                await page.locator('#spacing').click();
                await page.locator('[data-value="comfortable"]').click();
                await assertPersistedSetting(page, 'spacing', 'comfortable');

                // Now change to spacious
                await page.locator('#spacing').click();
                await page.locator('[data-value="spacious"]').click();
                await expect(page.locator('#spacing')).toContainText('Spacious');
                await assertPersistedSetting(page, 'spacing', 'spacious');
            });
        });

        test.describe('Storage Unit View', () => {
            test('can change view to list', async ({ whodb, page }) => {
                // First set to card to ensure we're changing from a known state
                await page.locator('#storage-unit-view').click();
                await page.locator('[data-value="card"]').click();
                await assertPersistedSetting(page, 'storageUnitView', 'card');

                // Now change to list
                await page.locator('#storage-unit-view').click();
                await page.locator('[data-value="list"]').click();
                await expect(page.locator('#storage-unit-view')).toContainText('List');
                await assertPersistedSetting(page, 'storageUnitView', 'list');
            });

            test('can change view to card', async ({ whodb, page }) => {
                // First set to list to ensure we're changing from a known state
                await page.locator('#storage-unit-view').click();
                await page.locator('[data-value="list"]').click();
                await assertPersistedSetting(page, 'storageUnitView', 'list');

                // Now change to card
                await page.locator('#storage-unit-view').click();
                await page.locator('[data-value="card"]').click();
                await expect(page.locator('#storage-unit-view')).toContainText('Card');
                await assertPersistedSetting(page, 'storageUnitView', 'card');
            });

            test('persists card view when navigating to tables', async ({ whodb, page }) => {
                // First set to list to ensure we're changing from a known state
                await page.locator('#storage-unit-view').click();
                await page.locator('[data-value="list"]').click();
                await assertPersistedSetting(page, 'storageUnitView', 'list');

                // Now change to card
                await page.locator('#storage-unit-view').click();
                await page.locator('[data-value="card"]').click();
                await assertPersistedSetting(page, 'storageUnitView', 'card');

                // Navigate to storage units page
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 10000 });

                // Should see card view (grid layout with cards)
                await expect(page.locator('[data-testid="storage-unit-card"]')).toBeTruthy();
            });

            test('persists list view when navigating to tables', async ({ whodb, page }) => {
                // First set to card to ensure we're changing from a known state
                await page.locator('#storage-unit-view').click();
                await page.locator('[data-value="card"]').click();
                await assertPersistedSetting(page, 'storageUnitView', 'card');

                // Now change to list
                await page.locator('#storage-unit-view').click();
                await page.locator('[data-value="list"]').click();
                await assertPersistedSetting(page, 'storageUnitView', 'list');

                // Navigate to storage units page
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="scratchpad-button"]').waitFor({ state: 'visible', timeout: 10000 });

                // Should see list view (table)
                await expect(page.locator('table')).toBeTruthy();
            });
        });

        test.describe('Where Condition Mode', () => {
            test('can change where condition mode to popover', async ({ whodb, page }) => {
                // First set to sheet to ensure we're changing from a known state
                await page.locator('#where-condition-mode').click();
                await page.locator('[data-value="sheet"]').click();
                await assertPersistedSetting(page, 'whereConditionMode', 'sheet');

                // Now change to popover
                await page.locator('#where-condition-mode').click();
                await page.locator('[data-value="popover"]').click();
                await expect(page.locator('#where-condition-mode')).toContainText('Popover');
                await assertPersistedSetting(page, 'whereConditionMode', 'popover');
            });

            test('can change where condition mode to sheet', async ({ whodb, page }) => {
                // First set to popover to ensure we're changing from a known state
                await page.locator('#where-condition-mode').click();
                await page.locator('[data-value="popover"]').click();
                await assertPersistedSetting(page, 'whereConditionMode', 'popover');

                // Now change to sheet
                await page.locator('#where-condition-mode').click();
                await page.locator('[data-value="sheet"]').click();
                await expect(page.locator('#where-condition-mode')).toContainText('Sheet');
                await assertPersistedSetting(page, 'whereConditionMode', 'sheet');
            });
        });

        test.describe('Default Page Size', () => {
            test('can change default page size', async ({ whodb, page }) => {
                // First set to 100 to ensure we're changing from a known state
                await page.locator('#default-page-size').click();
                await page.locator('[data-value="100"]').click();
                await assertPersistedSetting(page, 'defaultPageSize', 100);

                // Now change to 25
                await page.locator('#default-page-size').click();
                await page.locator('[data-value="25"]').click();
                await expect(page.locator('#default-page-size')).toContainText('25');
                await assertPersistedSetting(page, 'defaultPageSize', 25);
            });

            test('can set custom page size', async ({ whodb, page }) => {
                // First set to 50 to ensure we're changing from a known state
                await page.locator('#default-page-size').click();
                await page.locator('[data-value="50"]').click();
                await assertPersistedSetting(page, 'defaultPageSize', 50);

                // Now change to custom
                await page.locator('#default-page-size').click();
                await page.locator('[data-value="custom"]').click();

                // Enter custom value and press Enter to trigger save
                await page.locator('input[type="number"]').clear();
                await page.locator('input[type="number"]').fill('75');
                await page.locator('input[type="number"]').press('Enter');

                // Verify custom input is displayed and persisted
                await expect(page.locator('input[type="number"]')).toHaveValue('75');
                // Wait for redux-persist to save
                await page.waitForTimeout(100);
                await assertPersistedSetting(page, 'defaultPageSize', 75);
            });
        });

        test.describe('Telemetry Toggle', () => {
            test('can toggle telemetry off and persists to localStorage', async ({ whodb, page }) => {
                const telemetryButton = 'button[role="switch"]';

                // Check if telemetry toggle exists (CE only - not available in EE)
                const toggleCount = await page.locator(telemetryButton).count();
                if (toggleCount === 0) {
                    // Telemetry toggle not present - skip test
                    test.skip();
                    return;
                }

                // First ensure telemetry is ON so we can test turning it OFF
                const initialState = await page.locator(telemetryButton).first().getAttribute('data-state');
                if (initialState === 'unchecked') {
                    // Enable first
                    await page.locator(telemetryButton).first().dispatchEvent('click');
                    await expect(page.locator(telemetryButton).first()).toHaveAttribute('data-state', 'checked');
                    // Wait for state to persist
                    await page.waitForTimeout(200);
                }

                // Verify telemetry is now enabled (before state)
                const consentBefore = await page.evaluate(() => localStorage.getItem('whodb.analytics.consent'));
                expect(consentBefore).toEqual('granted');
                await assertPersistedSetting(page, 'metricsEnabled', true);

                // Toggle telemetry OFF
                await page.locator(telemetryButton).first().dispatchEvent('click');
                await expect(page.locator(telemetryButton).first()).toHaveAttribute('data-state', 'unchecked');

                // Wait for state to persist
                await page.waitForTimeout(200);

                // Verify both localStorage keys are updated (after state)
                const consentAfter = await page.evaluate(() => localStorage.getItem('whodb.analytics.consent'));
                expect(consentAfter).toEqual('denied');
                await assertPersistedSetting(page, 'metricsEnabled', false);
            });
        });
    });

    // Schema terminology tests - run for databases where the setting is relevant
    // (databases that use "database" instead of "schema", e.g., MySQL, MariaDB, ClickHouse)
    forEachDatabase('all', (db) => {
        if (!db.sidebar?.showsDatabaseDropdown || db.sidebar?.showsSchemaDropdown !== false) {
            return;
        }

        test.describe('Schema Terminology', () => {
            test.beforeEach(async ({ whodb }) => {
                await whodb.goto('settings');
            });

            test('can change terminology to schema', async ({ whodb, page }) => {
                // Set to schema
                await page.locator('#database-schema-terminology').click();
                await page.locator('[data-value="schema"]').click();
                await expect(page.locator('#database-schema-terminology')).toContainText('Schema');
                await assertPersistedSetting(page, 'databaseSchemaTerminology', 'schema');

                // Navigate to storage-unit and verify sidebar dropdown reflects the change
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 10000 });

                const sidebarText = await page.locator('[data-testid="sidebar-database-label"]').textContent();
                expect(sidebarText.toLowerCase()).toContain('schema');
            });

            test('can change terminology to database', async ({ whodb, page }) => {
                // Set to database
                await page.locator('#database-schema-terminology').click();
                await page.locator('[data-value="database"]').click();
                await expect(page.locator('#database-schema-terminology')).toContainText('Database');
                await assertPersistedSetting(page, 'databaseSchemaTerminology', 'database');

                // Navigate to storage-unit and verify sidebar dropdown reflects the change
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 10000 });

                const sidebarText = await page.locator('[data-testid="sidebar-database-label"]').textContent();
                expect(sidebarText.toLowerCase()).toContain('database');
            });

            test('terminology persists after navigation', async ({ whodb, page }) => {
                // Set to schema
                await page.locator('#database-schema-terminology').click();
                await page.locator('[data-value="schema"]').click();
                await assertPersistedSetting(page, 'databaseSchemaTerminology', 'schema');

                // Navigate away
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 10000 });

                // Come back to settings
                await whodb.goto('settings');

                // Verify the select still shows "schema"
                await expect(page.locator('#database-schema-terminology')).toContainText('Schema');

                // Reset to database for other tests
                await page.locator('#database-schema-terminology').click();
                await page.locator('[data-value="database"]').click();
            });
        });
    });

    // Theme toggle tests - run for any database since it's global
    forEachDatabase('sql', (db) => {
        // Only run for first SQL database to avoid redundant tests
        if (db.type !== 'Postgres') {
            return;
        }

        test.describe('Theme Toggle', () => {
            test('has theme toggle button visible on storage unit page', async ({ whodb, page }) => {
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="scratchpad-button"]').waitFor({ state: 'visible', timeout: 10000 });

                // Mode toggle should be visible
                await expect(page.locator('[data-testid="mode-toggle"]')).toBeVisible();
                await expect(page.locator('[data-testid="mode-toggle"] button')).toBeTruthy();
            });

            test('can toggle theme mode', async ({ whodb, page }) => {
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="scratchpad-button"]').waitFor({ state: 'visible', timeout: 10000 });

                // Click the mode toggle button - use first() in case multiple found, scrollIntoView for visibility
                await page.locator('[data-testid="mode-toggle"] button').first().scrollIntoViewIfNeeded();
                await page.locator('[data-testid="mode-toggle"] button').first().click();

                // A dropdown menu should appear with theme options
                await expect(page.locator('[role="menu"]')).toBeVisible();

                // Should have exactly 3 options: Light, Dark, System
                await expect(page.locator('[role="menuitem"]')).toHaveCount(3);

                // Verify all options exist
                await expect(page.locator('[role="menuitem"]').filter({ hasText: /light/i })).toBeVisible();
                await expect(page.locator('[role="menuitem"]').filter({ hasText: /dark/i })).toBeVisible();
                await expect(page.locator('[role="menuitem"]').filter({ hasText: /system/i })).toBeVisible();

                // Close the menu by pressing Escape
                await page.keyboard.press('Escape');
            });

            test('can switch to light mode', async ({ whodb, page }) => {
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="scratchpad-button"]').waitFor({ state: 'visible', timeout: 10000 });

                // Click the mode toggle
                await page.locator('[data-testid="mode-toggle"] button').first().scrollIntoViewIfNeeded();
                await page.locator('[data-testid="mode-toggle"] button').first().click();

                // Click light mode option
                await page.locator('[role="menuitem"]').filter({ hasText: /light/i }).click();

                // Verify theme changed (html element should have class)
                await expect(page.locator('html')).toHaveClass(/light/);
            });

            test('can switch to dark mode', async ({ whodb, page }) => {
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="scratchpad-button"]').waitFor({ state: 'visible', timeout: 10000 });

                // Click the mode toggle
                await page.locator('[data-testid="mode-toggle"] button').first().scrollIntoViewIfNeeded();
                await page.locator('[data-testid="mode-toggle"] button').first().click();

                // Click dark mode option
                await page.locator('[role="menuitem"]').filter({ hasText: /dark/i }).click();

                // Verify theme changed
                await expect(page.locator('html')).toHaveClass(/dark/);
            });

            test('theme persists after navigation', async ({ whodb, page }) => {
                await whodb.goto('storage-unit');
                await page.locator('[data-testid="scratchpad-button"]').waitFor({ state: 'visible', timeout: 10000 });

                // Switch to light mode
                await page.locator('[data-testid="mode-toggle"] button').first().scrollIntoViewIfNeeded();
                await page.locator('[data-testid="mode-toggle"] button').first().click();
                await page.locator('[role="menuitem"]').filter({ hasText: /light/i }).click();

                // Navigate to settings
                await whodb.goto('settings');

                // Theme should still be light
                await expect(page.locator('html')).toHaveClass(/light/);

                // Switch back to dark for other tests
                await page.locator('[data-testid="mode-toggle"] button').first().scrollIntoViewIfNeeded();
                await page.locator('[data-testid="mode-toggle"] button').first().click();
                await page.locator('[role="menuitem"]').filter({ hasText: /dark/i }).click();
            });
        });
    });

});
