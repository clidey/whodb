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

// Tour tests only run for PostgreSQL (the sample database type)
const targetDb = process.env.DATABASE;
const shouldRun = !targetDb || targetDb.toLowerCase() === 'postgres';

/**
 * Tour & Onboarding Tests
 *
 * Tests the tour functionality that guides first-time users through WhoDB features.
 * The tour automatically starts when logging into the sample database for the first time.
 *
 * NOTE: These tests only run for PostgreSQL (the sample database type).
 * When running tests for other databases, these tests are skipped.
 */
const describeOrSkip = shouldRun ? test.describe : test.describe.skip;

describeOrSkip('Tour & Onboarding', () => {
    /**
     * Helper function to log into the sample database
     * The sample database is a built-in profile that triggers the tour on first login
     */
    const loginToSampleDatabase = async (whodb, page) => {
        await page.goto(whodb.url('/login'));

        // Dismiss telemetry modal if it appears
        const disableBtn = page.locator('button').filter({ hasText: 'Disable Telemetry' });
        if (await disableBtn.count() > 0) {
            await disableBtn.click();
        }

        // Click the "Get Started" button for sample database
        await page.locator('[data-testid="get-started-sample-db"]').waitFor({ timeout: 10000 });
        await expect(page.locator('[data-testid="get-started-sample-db"]')).toBeVisible();
        await page.locator('[data-testid="get-started-sample-db"]').click();

        // Wait for navigation to storage-unit page
        await expect(page).toHaveURL(/\/storage-unit/, { timeout: 15000 });
    };

    /**
     * Helper function to wait for tour to become active
     */
    const waitForTourToStart = async (page) => {
        // Wait for tour tooltip to appear (more reliable than checking overflow)
        await expect(page.locator('[data-testid="tour-tooltip"]')).toBeVisible({ timeout: 15000 });
    };

    const waitForTourCompletion = async (page) => {
        await expect(page.locator('[data-testid="tour-tooltip"]')).not.toBeAttached();
        await expect.poll(
            () => page.evaluate(() => localStorage.getItem('@clidey/whodb/onboarding-completed')),
            { timeout: 5000 }
        ).toEqual('true');
    };

    /**
     * Helper function to get the current tour tooltip
     */
    const getTourTooltip = (page) => {
        return page.locator('[data-testid="tour-tooltip"]');
    };

    test.describe('Tour Auto-Start', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
        });

        test('starts tour automatically on sample database login (first-time user)', async ({ whodb, page }) => {
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);

            // Verify tour tooltip is visible
            await expect(getTourTooltip(page)).toBeVisible();

            // Verify first step content
            await expect(getTourTooltip(page)).toContainText('Welcome to WhoDB');
            await expect(getTourTooltip(page)).toContainText('Take a quick tour');
        });

        test('displays correct step counter on first step', async ({ whodb, page }) => {
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);

            // Should show step 1 of 6 (based on tour-config.tsx)
            await expect(getTourTooltip(page)).toContainText('1');
            await expect(getTourTooltip(page)).toContainText('6');
        });

        test('shows spotlight on target element', async ({ whodb, page }) => {
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);

            // First step targets #whodb-app-container
            // Verify spotlight exists (implementation may use backdrop/overlay)
            await expect(page.locator('#whodb-app-container')).toBeAttached();
        });
    });

    test.describe('Tour Navigation - Next Button', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);
        });

        test('advances to next step when next button is clicked', async ({ whodb, page }) => {
            // Verify first step
            await expect(getTourTooltip(page)).toContainText('Welcome to WhoDB');

            // Click next button
            await page.locator('[data-testid="tour-next-button"]').click();

            // Wait for transition
            await page.waitForTimeout(500);

            // Verify second step - Chat
            await expect(getTourTooltip(page)).toContainText('Ask in plain English');
            await expect(getTourTooltip(page)).toContainText('top products this month');

            // Step counter should show 2 of 6
            await expect(getTourTooltip(page)).toContainText('2');
        });

        test('navigates through all tour steps sequentially', async ({ whodb, page }) => {
            const expectedSteps = [
                'Welcome to WhoDB',
                'Ask in plain English',
                'Explore your schema',
                'Browse tables',
                'Write SQL in Scratchpad',
                'You are all set'
            ];

            for (let index = 0; index < expectedSteps.length; index++) {
                const stepTitle = expectedSteps[index];
                // Verify current step title
                await expect(getTourTooltip(page)).toContainText(stepTitle);

                // Verify step counter
                await expect(getTourTooltip(page)).toContainText(`${index + 1}`);
                await expect(getTourTooltip(page)).toContainText('6');

                // Click next unless it's the last step
                if (index < expectedSteps.length - 1) {
                    await page.locator('[data-testid="tour-next-button"]').click();
                    await page.waitForTimeout(800); // Wait for transition and potential navigation
                }
            }
        });

        test('highlights correct sidebar elements during navigation steps', async ({ whodb, page }) => {
            // Step 1: Welcome (center position)
            await expect(getTourTooltip(page)).toContainText('Welcome to WhoDB');
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);

            // Step 2: AI Chat - should highlight chat link
            await expect(getTourTooltip(page)).toContainText('Ask in plain English');
            await expect(page.locator('[href="/chat"]')).toBeAttached();
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);

            // Step 3: Graph - should highlight graph link
            await expect(getTourTooltip(page)).toContainText('Explore your schema');
            await expect(page.locator('[href="/graph"]')).toBeAttached();
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);

            // Step 4: Storage Unit cards
            await expect(getTourTooltip(page)).toContainText('Browse tables');
            await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeAttached();
        });
    });

    test.describe('Tour Navigation - Previous Button', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);
        });

        test('does not show previous button on first step', async ({ whodb, page }) => {
            await expect(getTourTooltip(page)).toContainText('Welcome to WhoDB');

            // Previous button should not exist on first step (component doesn't render it)
            await expect(page.locator('[data-testid="tour-prev-button"]')).not.toBeAttached();
        });

        test('goes back to previous step when previous button is clicked', async ({ whodb, page }) => {
            // Navigate to second step
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);
            await expect(getTourTooltip(page)).toContainText('Ask in plain English');

            // Click previous button
            await expect(page.locator('[data-testid="tour-prev-button"]')).toBeVisible();
            await page.locator('[data-testid="tour-prev-button"]').click();
            await page.waitForTimeout(500);

            // Should be back to first step
            await expect(getTourTooltip(page)).toContainText('Welcome to WhoDB');
            await expect(getTourTooltip(page)).toContainText('1');
        });

        test('navigates backward through multiple steps', async ({ whodb, page }) => {
            // Navigate to step 4
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(800);

            await expect(getTourTooltip(page)).toContainText('Browse tables');
            await expect(getTourTooltip(page)).toContainText('4');

            // Go back to step 3
            await page.locator('[data-testid="tour-prev-button"]').click();
            await page.waitForTimeout(500);
            await expect(getTourTooltip(page)).toContainText('Explore your schema');
            await expect(getTourTooltip(page)).toContainText('3');

            // Go back to step 2
            await page.locator('[data-testid="tour-prev-button"]').click();
            await page.waitForTimeout(500);
            await expect(getTourTooltip(page)).toContainText('Ask in plain English');
            await expect(getTourTooltip(page)).toContainText('2');
        });
    });

    test.describe('Tour Skip Functionality', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);
        });

        test('shows skip button on tour tooltip', async ({ whodb, page }) => {
            await expect(getTourTooltip(page)).toBeVisible();
            await expect(page.locator('[data-testid="tour-skip-button"]')).toBeVisible();
        });

        test('closes tour when skip button is clicked', async ({ whodb, page }) => {
            await expect(getTourTooltip(page)).toBeVisible();

            // Click skip button
            await page.locator('[data-testid="tour-skip-button"]').click();

            // Tour tooltip should disappear
            await expect(page.locator('[data-testid="tour-tooltip"]')).not.toBeAttached();

            // Body overflow should be restored
            const overflow = await page.locator('body').evaluate(el => getComputedStyle(el).overflow);
            expect(overflow).not.toEqual('hidden');
        });

        test('marks onboarding as complete when tour is skipped', async ({ whodb, page }) => {
            await page.locator('[data-testid="tour-skip-button"]').click();

            // Verify localStorage key is set
            const completed = await page.evaluate(() => localStorage.getItem('@clidey/whodb/onboarding-completed'));
            expect(completed).toEqual('true');
        });

        test('can skip tour from any step', async ({ whodb, page }) => {
            // Navigate to middle step
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);

            await expect(getTourTooltip(page)).toContainText('Explore your schema');

            // Skip from this step
            await page.locator('[data-testid="tour-skip-button"]').click();

            // Tour should close
            await expect(page.locator('[data-testid="tour-tooltip"]')).not.toBeAttached();

            // Onboarding should be marked complete
            const completed = await page.evaluate(() => localStorage.getItem('@clidey/whodb/onboarding-completed'));
            expect(completed).toEqual('true');
        });
    });

    test.describe('Tour Completion', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);
        });

        test('shows finish button on last step', async ({ whodb, page }) => {
            // Navigate to last step
            for (let i = 0; i < 5; i++) {
                await page.locator('[data-testid="tour-next-button"]').click();
                await page.waitForTimeout(800);
            }

            // Last step should show finish/complete button
            await expect(getTourTooltip(page)).toContainText('You are all set');
            await expect(getTourTooltip(page)).toContainText('6');

            // Next button might say "Finish" or "Complete" on last step
            await expect(page.locator('[data-testid="tour-next-button"]')).toBeVisible();
        });

        test('closes tour and marks onboarding complete when finishing', async ({ whodb, page }) => {
            // Navigate to last step
            for (let i = 0; i < 5; i++) {
                await page.locator('[data-testid="tour-next-button"]').click();
                await page.waitForTimeout(800);
            }

            await expect(getTourTooltip(page)).toContainText('You are all set');

            // Click finish button
            await page.locator('[data-testid="tour-next-button"]').click();

            // Tour should close
            await expect(page.locator('[data-testid="tour-tooltip"]')).not.toBeAttached();

            // Body overflow should be restored
            const overflow = await page.locator('body').evaluate(el => getComputedStyle(el).overflow);
            expect(overflow).not.toEqual('hidden');

            // Verify onboarding is marked complete in localStorage
            const completed = await page.evaluate(() => localStorage.getItem('@clidey/whodb/onboarding-completed'));
            expect(completed).toEqual('true');
        });
    });

    test.describe('Tour Persistence', () => {
        test('does not restart tour on subsequent logins after completion', async ({ whodb, page }) => {
            await clearBrowserState(page);
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);

            // Skip the tour to mark it complete
            await page.locator('[data-testid="tour-skip-button"]').click();
            await expect(page.locator('[data-testid="tour-tooltip"]')).not.toBeAttached();

            // Logout
            await whodb.logout();
            await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

            // After onboarding complete, "Get Started" panel is hidden
            // Login with regular postgres credentials instead
            const db = getDatabaseConfig('postgres');
            const conn = db.connection;
            await whodb.login(
                db.uiType || db.type,
                conn.host ?? undefined,
                conn.user ?? undefined,
                conn.password ?? undefined,
                conn.database ?? undefined,
                conn.advanced || {}
            );

            // Wait a bit to see if tour would start
            await page.waitForTimeout(2000);

            // Tour should not start
            await expect(page.locator('[data-testid="tour-tooltip"]')).not.toBeAttached();

            // Body should not have overflow hidden
            const overflow = await page.locator('body').evaluate(el => getComputedStyle(el).overflow);
            expect(overflow).not.toEqual('hidden');
        });

        test('does not restart tour after page refresh if completed', async ({ whodb, page }) => {
            await clearBrowserState(page);
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);

            // Complete the tour
            await page.locator('[data-testid="tour-skip-button"]').click();
            await waitForTourCompletion(page);

            // Reload the page
            await page.reload();

            // Wait for page to load
            await expect(page).toHaveURL(/\/storage-unit/, { timeout: 10000 });
            await page.waitForTimeout(2000);

            // Tour should not restart
            await expect(page.locator('[data-testid="tour-tooltip"]')).not.toBeAttached();
        });

        test('persists onboarding completion across sessions', async ({ whodb, page }) => {
            // Navigate to app first so localStorage is accessible
            await page.goto(whodb.url('/login'));
            await clearBrowserState(page);

            // Manually set onboarding complete
            await page.evaluate(() => {
                localStorage.setItem('@clidey/whodb/onboarding-completed', 'true');
            });

            // After onboarding complete, "Get Started" panel is hidden
            // Login with regular postgres credentials instead
            const db = getDatabaseConfig('postgres');
            const conn = db.connection;
            await whodb.login(
                db.uiType || db.type,
                conn.host ?? undefined,
                conn.user ?? undefined,
                conn.password ?? undefined,
                conn.database ?? undefined,
                conn.advanced || {}
            );

            // Wait to see if tour would start
            await page.waitForTimeout(2000);

            // Tour should not start
            await expect(page.locator('[data-testid="tour-tooltip"]')).not.toBeAttached();
        });
    });

    test.describe('Tour Tooltip Content', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);
        });

        test('displays correct title and description for each step', async ({ whodb, page }) => {
            const steps = [
                {
                    title: 'Welcome to WhoDB',
                    descriptionSnippet: 'quick tour'
                },
                {
                    title: 'Ask in plain English',
                    descriptionSnippet: 'top products this month'
                },
                {
                    title: 'Explore your schema',
                    descriptionSnippet: 'relationships'
                },
                {
                    title: 'Browse tables',
                    descriptionSnippet: 'table card'
                },
                {
                    title: 'Write SQL in Scratchpad',
                    descriptionSnippet: 'autocomplete'
                },
                {
                    title: 'You are all set',
                    descriptionSnippet: 'sample database'
                }
            ];

            for (let index = 0; index < steps.length; index++) {
                const step = steps[index];
                await expect(getTourTooltip(page)).toContainText(step.title);
                await expect(getTourTooltip(page)).toContainText(step.descriptionSnippet);

                // Move to next step unless it's the last one
                if (index < steps.length - 1) {
                    await page.locator('[data-testid="tour-next-button"]').click();
                    await page.waitForTimeout(800);
                }
            }
        });

        test('displays progress dots for each step', async ({ whodb, page }) => {
            await expect(getTourTooltip(page).locator('.tour-tooltip-dots span')).toHaveCount(6);

            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);
            await expect(getTourTooltip(page).locator('.tour-tooltip-dots .is-current')).toHaveCount(1);

            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);
            await expect(getTourTooltip(page).locator('.tour-tooltip-dots span')).toHaveCount(6);
        });

        test('shows progress indicator with current step', async ({ whodb, page }) => {
            await expect(getTourTooltip(page)).toContainText('1');
            await expect(getTourTooltip(page)).toContainText('6');

            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);

            await expect(getTourTooltip(page)).toContainText('2');
            await expect(getTourTooltip(page)).toContainText('6');
        });
    });

    test.describe('Tour Overlay Behavior', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);
        });

        test('prevents page scrolling while tour is active', async ({ whodb, page }) => {
            // Body should have overflow hidden
            const overflow = await page.locator('body').evaluate(el => getComputedStyle(el).overflow);
            expect(overflow).toEqual('hidden');
        });

        test('restores page scrolling when tour is closed', async ({ whodb, page }) => {
            await page.locator('[data-testid="tour-skip-button"]').click();

            // Body overflow should be restored (empty string or 'visible')
            const overflow = await page.locator('body').evaluate(el => getComputedStyle(el).overflow);
            expect(['', 'visible', 'auto']).toContain(overflow);
        });

        test('maintains overlay while navigating between steps', async ({ whodb, page }) => {
            // Body should have overflow hidden on step 1
            let overflow = await page.locator('body').evaluate(el => getComputedStyle(el).overflow);
            expect(overflow).toEqual('hidden');

            // Navigate to step 2
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);

            // Body should still have overflow hidden
            overflow = await page.locator('body').evaluate(el => getComputedStyle(el).overflow);
            expect(overflow).toEqual('hidden');

            // Navigate to step 3
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(500);

            // Body should still have overflow hidden
            overflow = await page.locator('body').evaluate(el => getComputedStyle(el).overflow);
            expect(overflow).toEqual('hidden');
        });
    });

    test.describe('Tour Target Element Highlighting', () => {
        test.beforeEach(async ({ whodb, page }) => {
            await clearBrowserState(page);
            await loginToSampleDatabase(whodb, page);
            await waitForTourToStart(page);
        });

        test('scrolls target element into view when needed', async ({ whodb, page }) => {
            // Navigate to a step that requires scrolling (e.g., storage unit cards)
            for (let i = 0; i < 3; i++) {
                await page.locator('[data-testid="tour-next-button"]').click();
                await page.waitForTimeout(800);
            }

            await expect(getTourTooltip(page)).toContainText('Browse tables');

            // Target element should be visible
            await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible();
        });

        test('waits for target elements to be available after navigation', async ({ whodb, page }) => {
            await expect(getTourTooltip(page)).toContainText('Welcome to WhoDB');

            // Step 2 targets the chat link which should be in the sidebar
            await page.locator('[data-testid="tour-next-button"]').click();
            await page.waitForTimeout(800);

            // Tour should wait for element and then show tooltip
            await expect(getTourTooltip(page)).toContainText('Ask in plain English');
            await expect(page.locator('[href="/chat"]')).toBeAttached();
        });
    });
});
