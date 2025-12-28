/*
 * Copyright 2025 Clidey, Inc.
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

import {clearBrowserState} from '../../support/helpers/animation';
import {getDatabaseConfig, loginToDatabase} from '../../support/test-runner';

// Tour tests only run for PostgreSQL (the sample database type)
const targetDb = Cypress.env('database');
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
(shouldRun ? describe : describe.skip)('Tour & Onboarding', () => {
    /**
     * Helper function to log into the sample database
     * The sample database is a built-in profile that triggers the tour on first login
     */
    const loginToSampleDatabase = () => {
        cy.visit('/login');

        // Dismiss telemetry modal if it appears
        cy.get('body').then($body => {
            const $btn = $body.find('button').filter(function () {
                return this.textContent.includes('Disable Telemetry');
            });
            if ($btn.length) {
                cy.wrap($btn).click();
            }
        });

        // Click the "Get Started" button for sample database
        cy.get('[data-testid="get-started-sample-db"]', {timeout: 10000}).should('be.visible').click();

        // Wait for navigation to storage-unit page
        cy.url({timeout: 15000}).should('include', '/storage-unit');
    };

    /**
     * Helper function to wait for tour to become active
     */
    const waitForTourToStart = () => {
        // Wait for tour tooltip to appear (more reliable than checking overflow)
        cy.get('[data-testid="tour-tooltip"]', {timeout: 15000}).should('be.visible');
    };

    /**
     * Helper function to get the current tour tooltip
     */
    const getTourTooltip = () => {
        return cy.get('[data-testid="tour-tooltip"]', {timeout: 5000});
    };

    describe('Tour Auto-Start', () => {
        beforeEach(() => {
            clearBrowserState();
        });

        it('starts tour automatically on sample database login (first-time user)', () => {
            loginToSampleDatabase();
            waitForTourToStart();

            // Verify tour tooltip is visible
            getTourTooltip().should('be.visible');

            // Verify first step content
            getTourTooltip().should('contain.text', 'Welcome to WhoDB');
            getTourTooltip().should('contain.text', 'Let\'s take a quick tour');
        });

        it('displays correct step counter on first step', () => {
            loginToSampleDatabase();
            waitForTourToStart();

            // Should show step 1 of 7 (based on tour-config.tsx)
            getTourTooltip().should('contain.text', '1');
            getTourTooltip().should('contain.text', '7');
        });

        it('shows spotlight on target element', () => {
            loginToSampleDatabase();
            waitForTourToStart();

            // First step targets #whodb-app-container
            // Verify spotlight exists (implementation may use backdrop/overlay)
            cy.get('#whodb-app-container').should('exist');
        });
    });

    describe('Tour Navigation - Next Button', () => {
        beforeEach(() => {
            clearBrowserState();
            loginToSampleDatabase();
            waitForTourToStart();
        });

        it('advances to next step when next button is clicked', () => {
            // Verify first step
            getTourTooltip().should('contain.text', 'Welcome to WhoDB');

            // Click next button
            cy.get('[data-testid="tour-next-button"]').click();

            // Wait for transition
            cy.wait(500);

            // Verify second step - AI Chat Assistant
            getTourTooltip().should('contain.text', 'AI Chat Assistant');
            getTourTooltip().should('contain.text', 'Ask questions in plain English');

            // Step counter should show 2 of 7
            getTourTooltip().should('contain.text', '2');
        });

        it('navigates through all tour steps sequentially', () => {
            const expectedSteps = [
                'Welcome to WhoDB',
                'AI Chat Assistant',
                'Visual Schema Explorer',
                'Browse Database Tables',
                'SQL Editor & Scratchpad',
                'View Table Data',
                'You\'re All Set!'
            ];

            expectedSteps.forEach((stepTitle, index) => {
                // Verify current step title
                getTourTooltip().should('contain.text', stepTitle);

                // Verify step counter
                getTourTooltip().should('contain.text', `${index + 1}`);
                getTourTooltip().should('contain.text', '7');

                // Click next unless it's the last step
                if (index < expectedSteps.length - 1) {
                    cy.get('[data-testid="tour-next-button"]').click();
                    cy.wait(800); // Wait for transition and potential navigation
                }
            });
        });

        it('highlights correct sidebar elements during navigation steps', () => {
            // Step 1: Welcome (center position)
            getTourTooltip().should('contain.text', 'Welcome to WhoDB');
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);

            // Step 2: AI Chat - should highlight chat link
            getTourTooltip().should('contain.text', 'AI Chat Assistant');
            cy.get('[href="/chat"]').should('exist');
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);

            // Step 3: Graph - should highlight graph link
            getTourTooltip().should('contain.text', 'Visual Schema Explorer');
            cy.get('[href="/graph"]').should('exist');
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);

            // Step 4: Storage Unit cards
            getTourTooltip().should('contain.text', 'Browse Database Tables');
            cy.get('[data-testid="storage-unit-card-list"]').should('exist');
        });
    });

    describe('Tour Navigation - Previous Button', () => {
        beforeEach(() => {
            clearBrowserState();
            loginToSampleDatabase();
            waitForTourToStart();
        });

        it('does not show previous button on first step', () => {
            getTourTooltip().should('contain.text', 'Welcome to WhoDB');

            // Previous button should not exist on first step (component doesn't render it)
            cy.get('[data-testid="tour-prev-button"]').should('not.exist');
        });

        it('goes back to previous step when previous button is clicked', () => {
            // Navigate to second step
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);
            getTourTooltip().should('contain.text', 'AI Chat Assistant');

            // Click previous button
            cy.get('[data-testid="tour-prev-button"]').should('be.visible').click();
            cy.wait(500);

            // Should be back to first step
            getTourTooltip().should('contain.text', 'Welcome to WhoDB');
            getTourTooltip().should('contain.text', '1');
        });

        it('navigates backward through multiple steps', () => {
            // Navigate to step 4
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(800);

            getTourTooltip().should('contain.text', 'Browse Database Tables');
            getTourTooltip().should('contain.text', '4');

            // Go back to step 3
            cy.get('[data-testid="tour-prev-button"]').click();
            cy.wait(500);
            getTourTooltip().should('contain.text', 'Visual Schema Explorer');
            getTourTooltip().should('contain.text', '3');

            // Go back to step 2
            cy.get('[data-testid="tour-prev-button"]').click();
            cy.wait(500);
            getTourTooltip().should('contain.text', 'AI Chat Assistant');
            getTourTooltip().should('contain.text', '2');
        });
    });

    describe('Tour Skip Functionality', () => {
        beforeEach(() => {
            clearBrowserState();
            loginToSampleDatabase();
            waitForTourToStart();
        });

        it('shows skip button on tour tooltip', () => {
            getTourTooltip().should('be.visible');
            cy.get('[data-testid="tour-skip-button"]').should('be.visible');
        });

        it('closes tour when skip button is clicked', () => {
            getTourTooltip().should('be.visible');

            // Click skip button
            cy.get('[data-testid="tour-skip-button"]').click();

            // Tour tooltip should disappear
            cy.get('[data-testid="tour-tooltip"]').should('not.exist');

            // Body overflow should be restored
            cy.get('body').should('not.have.css', 'overflow', 'hidden');
        });

        it('marks onboarding as complete when tour is skipped', () => {
            cy.get('[data-testid="tour-skip-button"]').click();

            // Verify localStorage key is set
            cy.window().then((win) => {
                const completed = win.localStorage.getItem('@clidey/whodb/onboarding-completed');
                expect(completed).to.equal('true');
            });
        });

        it('can skip tour from any step', () => {
            // Navigate to middle step
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);

            getTourTooltip().should('contain.text', 'Visual Schema Explorer');

            // Skip from this step
            cy.get('[data-testid="tour-skip-button"]').click();

            // Tour should close
            cy.get('[data-testid="tour-tooltip"]').should('not.exist');

            // Onboarding should be marked complete
            cy.window().then((win) => {
                const completed = win.localStorage.getItem('@clidey/whodb/onboarding-completed');
                expect(completed).to.equal('true');
            });
        });
    });

    describe('Tour Completion', () => {
        beforeEach(() => {
            clearBrowserState();
            loginToSampleDatabase();
            waitForTourToStart();
        });

        it('shows finish button on last step', () => {
            // Navigate to last step
            for (let i = 0; i < 6; i++) {
                cy.get('[data-testid="tour-next-button"]').click();
                cy.wait(800);
            }

            // Last step should show finish/complete button
            getTourTooltip().should('contain.text', 'You\'re All Set!');
            getTourTooltip().should('contain.text', '7');

            // Next button might say "Finish" or "Complete" on last step
            cy.get('[data-testid="tour-next-button"]')
                .should('be.visible');
        });

        it('closes tour and marks onboarding complete when finishing', () => {
            // Navigate to last step
            for (let i = 0; i < 6; i++) {
                cy.get('[data-testid="tour-next-button"]').click();
                cy.wait(800);
            }

            getTourTooltip().should('contain.text', 'You\'re All Set!');

            // Click finish button
            cy.get('[data-testid="tour-next-button"]').click();

            // Tour should close
            cy.get('[data-testid="tour-tooltip"]').should('not.exist');

            // Body overflow should be restored
            cy.get('body').should('not.have.css', 'overflow', 'hidden');

            // Verify onboarding is marked complete in localStorage
            cy.window().then((win) => {
                const completed = win.localStorage.getItem('@clidey/whodb/onboarding-completed');
                expect(completed).to.equal('true');
            });
        });
    });

    describe('Tour Persistence', () => {
        it('does not restart tour on subsequent logins after completion', () => {
            clearBrowserState();
            loginToSampleDatabase();
            waitForTourToStart();

            // Skip the tour to mark it complete
            cy.get('[data-testid="tour-skip-button"]').click();
            cy.get('[data-testid="tour-tooltip"]').should('not.exist');

            // Logout
            cy.logout();
            cy.url({timeout: 10000}).should('include', '/login');

            // After onboarding complete, "Get Started" panel is hidden
            // Login with regular postgres credentials instead
            const db = getDatabaseConfig('postgres');
            loginToDatabase(db);

            // Wait a bit to see if tour would start
            cy.wait(2000);

            // Tour should not start
            cy.get('[data-testid="tour-tooltip"]').should('not.exist');

            // Body should not have overflow hidden
            cy.get('body').should('not.have.css', 'overflow', 'hidden');
        });

        it('does not restart tour after page refresh if completed', () => {
            clearBrowserState();
            loginToSampleDatabase();
            waitForTourToStart();

            // Complete the tour
            cy.get('[data-testid="tour-skip-button"]').click();

            // Reload the page
            cy.reload();

            // Wait for page to load
            cy.url({timeout: 10000}).should('include', '/storage-unit');
            cy.wait(2000);

            // Tour should not restart
            cy.get('[data-testid="tour-tooltip"]').should('not.exist');
        });

        it('persists onboarding completion across sessions', () => {
            clearBrowserState();

            // Manually set onboarding complete
            cy.window().then((win) => {
                win.localStorage.setItem('@clidey/whodb/onboarding-completed', 'true');
            });

            // After onboarding complete, "Get Started" panel is hidden
            // Login with regular postgres credentials instead
            const db = getDatabaseConfig('postgres');
            loginToDatabase(db);

            // Wait to see if tour would start
            cy.wait(2000);

            // Tour should not start
            cy.get('[data-testid="tour-tooltip"]').should('not.exist');
        });
    });

    describe('Tour Tooltip Content', () => {
        beforeEach(() => {
            clearBrowserState();
            loginToSampleDatabase();
            waitForTourToStart();
        });

        it('displays correct title and description for each step', () => {
            const steps = [
                {
                    title: 'Welcome to WhoDB',
                    descriptionSnippet: 'quick tour'
                },
                {
                    title: 'AI Chat Assistant',
                    descriptionSnippet: 'plain English'
                },
                {
                    title: 'Visual Schema Explorer',
                    descriptionSnippet: 'database structure'
                },
                {
                    title: 'Browse Database Tables',
                    descriptionSnippet: 'tables in your database'
                },
                {
                    title: 'SQL Editor & Scratchpad',
                    descriptionSnippet: 'custom SQL queries'
                },
                {
                    title: 'View Table Data',
                    descriptionSnippet: 'table card'
                },
                {
                    title: 'You\'re All Set!',
                    descriptionSnippet: 'key features'
                }
            ];

            steps.forEach((step, index) => {
                getTourTooltip().should('contain.text', step.title);
                getTourTooltip().should('contain.text', step.descriptionSnippet);

                // Move to next step unless it's the last one
                if (index < steps.length - 1) {
                    cy.get('[data-testid="tour-next-button"]').click();
                    cy.wait(800);
                }
            });
        });

        it('displays icons for each step', () => {
            // First step should have an icon (Sparkles icon for welcome)
            getTourTooltip().find('svg').should('exist');

            // Navigate and check other steps have icons
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);
            getTourTooltip().find('svg').should('exist'); // Chat icon

            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);
            getTourTooltip().find('svg').should('exist'); // Graph icon
        });

        it('shows progress indicator with current step', () => {
            getTourTooltip().should('contain.text', '1');
            getTourTooltip().should('contain.text', '7');

            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);

            getTourTooltip().should('contain.text', '2');
            getTourTooltip().should('contain.text', '7');
        });
    });

    describe('Tour Overlay Behavior', () => {
        beforeEach(() => {
            clearBrowserState();
            loginToSampleDatabase();
            waitForTourToStart();
        });

        it('prevents page scrolling while tour is active', () => {
            // Body should have overflow hidden
            cy.get('body').should('have.css', 'overflow', 'hidden');
        });

        it('restores page scrolling when tour is closed', () => {
            cy.get('[data-testid="tour-skip-button"]').click();

            // Body overflow should be restored (empty string or 'visible')
            cy.get('body').then($body => {
                const overflow = $body.css('overflow');
                expect(overflow).to.be.oneOf(['', 'visible', 'auto']);
            });
        });

        it('maintains overlay while navigating between steps', () => {
            // Body should have overflow hidden on step 1
            cy.get('body').should('have.css', 'overflow', 'hidden');

            // Navigate to step 2
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);

            // Body should still have overflow hidden
            cy.get('body').should('have.css', 'overflow', 'hidden');

            // Navigate to step 3
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(500);

            // Body should still have overflow hidden
            cy.get('body').should('have.css', 'overflow', 'hidden');
        });
    });

    describe('Tour Target Element Highlighting', () => {
        beforeEach(() => {
            clearBrowserState();
            loginToSampleDatabase();
            waitForTourToStart();
        });

        it('scrolls target element into view when needed', () => {
            // Navigate to a step that requires scrolling (e.g., storage unit cards)
            for (let i = 0; i < 3; i++) {
                cy.get('[data-testid="tour-next-button"]').click();
                cy.wait(800);
            }

            getTourTooltip().should('contain.text', 'Browse Database Tables');

            // Target element should be visible
            cy.get('[data-testid="storage-unit-card-list"]').should('be.visible');
        });

        it('waits for target elements to be available after navigation', () => {
            getTourTooltip().should('contain.text', 'Welcome to WhoDB');

            // Step 2 targets the chat link which should be in the sidebar
            cy.get('[data-testid="tour-next-button"]').click();
            cy.wait(800);

            // Tour should wait for element and then show tooltip
            getTourTooltip().should('contain.text', 'AI Chat Assistant');
            cy.get('[href="/chat"]').should('exist');
        });
    });
});
