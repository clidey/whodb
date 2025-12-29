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

import {forEachDatabase} from '../../support/test-runner';

// Helper to assert a setting value in localStorage (redux-persist format)
const assertPersistedSetting = (key, expectedValue) => {
    cy.window().then((win) => {
        const persistedSettings = win.localStorage.getItem('persist:settings');
        expect(persistedSettings).to.not.be.null;
        const parsed = JSON.parse(persistedSettings);
        // Redux-persist double-encodes values as JSON strings
        const value = parsed[key] ? JSON.parse(parsed[key]) : null;
        expect(value).to.equal(expectedValue);
    });
};

describe('Settings', () => {

    // Run settings tests for one database only (settings are global)
    forEachDatabase('sql', (db) => {
        // Only run for first SQL database (postgres) to avoid redundant tests
        if (db.type !== 'Postgres') {
            return;
        }

        beforeEach(() => {
            cy.goto('settings');
        });

        describe('Font Size', () => {
            it('can change font size to small', () => {
                // First set to medium to ensure we're changing from a known state
                cy.get('#font-size').click();
                cy.get('[data-value="medium"]').click();
                assertPersistedSetting('fontSize', 'medium');

                // Now change to small
                cy.get('#font-size').click();
                cy.get('[data-value="small"]').click();
                cy.get('#font-size').should('contain', 'Small');
                assertPersistedSetting('fontSize', 'small');
            });

            it('can change font size to medium', () => {
                // First set to small to ensure we're changing from a known state
                cy.get('#font-size').click();
                cy.get('[data-value="small"]').click();
                assertPersistedSetting('fontSize', 'small');

                // Now change to medium
                cy.get('#font-size').click();
                cy.get('[data-value="medium"]').click();
                cy.get('#font-size').should('contain', 'Medium');
                assertPersistedSetting('fontSize', 'medium');
            });

            it('can change font size to large', () => {
                // First set to medium to ensure we're changing from a known state
                cy.get('#font-size').click();
                cy.get('[data-value="medium"]').click();
                assertPersistedSetting('fontSize', 'medium');

                // Now change to large
                cy.get('#font-size').click();
                cy.get('[data-value="large"]').click();
                cy.get('#font-size').should('contain', 'Large');
                assertPersistedSetting('fontSize', 'large');
            });
        });

        describe('Border Radius', () => {
            it('can change border radius to none', () => {
                // First set to medium to ensure we're changing from a known state
                cy.get('#border-radius').click();
                cy.get('[data-value="medium"]').click();
                assertPersistedSetting('borderRadius', 'medium');

                // Now change to none
                cy.get('#border-radius').click();
                cy.get('[data-value="none"]').click();
                cy.get('#border-radius').should('contain', 'None');
                assertPersistedSetting('borderRadius', 'none');
            });

            it('can change border radius to small', () => {
                // First set to none to ensure we're changing from a known state
                cy.get('#border-radius').click();
                cy.get('[data-value="none"]').click();
                assertPersistedSetting('borderRadius', 'none');

                // Now change to small
                cy.get('#border-radius').click();
                cy.get('[data-value="small"]').click();
                cy.get('#border-radius').should('contain', 'Small');
                assertPersistedSetting('borderRadius', 'small');
            });

            it('can change border radius to medium', () => {
                // First set to small to ensure we're changing from a known state
                cy.get('#border-radius').click();
                cy.get('[data-value="small"]').click();
                assertPersistedSetting('borderRadius', 'small');

                // Now change to medium
                cy.get('#border-radius').click();
                cy.get('[data-value="medium"]').click();
                cy.get('#border-radius').should('contain', 'Medium');
                assertPersistedSetting('borderRadius', 'medium');
            });

            it('can change border radius to large', () => {
                // First set to medium to ensure we're changing from a known state
                cy.get('#border-radius').click();
                cy.get('[data-value="medium"]').click();
                assertPersistedSetting('borderRadius', 'medium');

                // Now change to large
                cy.get('#border-radius').click();
                cy.get('[data-value="large"]').click();
                cy.get('#border-radius').should('contain', 'Large');
                assertPersistedSetting('borderRadius', 'large');
            });
        });

        describe('Spacing', () => {
            it('can change spacing to compact', () => {
                // First set to comfortable to ensure we're changing from a known state
                cy.get('#spacing').click();
                cy.get('[data-value="comfortable"]').click();
                assertPersistedSetting('spacing', 'comfortable');

                // Now change to compact
                cy.get('#spacing').click();
                cy.get('[data-value="compact"]').click();
                cy.get('#spacing').should('contain', 'Compact');
                assertPersistedSetting('spacing', 'compact');
            });

            it('can change spacing to comfortable', () => {
                // First set to compact to ensure we're changing from a known state
                cy.get('#spacing').click();
                cy.get('[data-value="compact"]').click();
                assertPersistedSetting('spacing', 'compact');

                // Now change to comfortable
                cy.get('#spacing').click();
                cy.get('[data-value="comfortable"]').click();
                cy.get('#spacing').should('contain', 'Comfortable');
                assertPersistedSetting('spacing', 'comfortable');
            });

            it('can change spacing to spacious', () => {
                // First set to comfortable to ensure we're changing from a known state
                cy.get('#spacing').click();
                cy.get('[data-value="comfortable"]').click();
                assertPersistedSetting('spacing', 'comfortable');

                // Now change to spacious
                cy.get('#spacing').click();
                cy.get('[data-value="spacious"]').click();
                cy.get('#spacing').should('contain', 'Spacious');
                assertPersistedSetting('spacing', 'spacious');
            });
        });

        describe('Storage Unit View', () => {
            it('can change view to list', () => {
                // First set to card to ensure we're changing from a known state
                cy.get('#storage-unit-view').click();
                cy.get('[data-value="card"]').click();
                assertPersistedSetting('storageUnitView', 'card');

                // Now change to list
                cy.get('#storage-unit-view').click();
                cy.get('[data-value="list"]').click();
                cy.get('#storage-unit-view').should('contain', 'List');
                assertPersistedSetting('storageUnitView', 'list');
            });

            it('can change view to card', () => {
                // First set to list to ensure we're changing from a known state
                cy.get('#storage-unit-view').click();
                cy.get('[data-value="list"]').click();
                assertPersistedSetting('storageUnitView', 'list');

                // Now change to card
                cy.get('#storage-unit-view').click();
                cy.get('[data-value="card"]').click();
                cy.get('#storage-unit-view').should('contain', 'Card');
                assertPersistedSetting('storageUnitView', 'card');
            });

            it('persists card view when navigating to tables', () => {
                // First set to list to ensure we're changing from a known state
                cy.get('#storage-unit-view').click();
                cy.get('[data-value="list"]').click();
                assertPersistedSetting('storageUnitView', 'list');

                // Now change to card
                cy.get('#storage-unit-view').click();
                cy.get('[data-value="card"]').click();
                assertPersistedSetting('storageUnitView', 'card');

                // Navigate to storage units page
                cy.goto('storage-unit');
                cy.get('[data-testid="storage-unit-card"]', {timeout: 10000}).should('exist');

                // Should see card view (grid layout with cards)
                cy.get('[data-testid="storage-unit-card"]').should('exist');
            });

            it('persists list view when navigating to tables', () => {
                // First set to card to ensure we're changing from a known state
                cy.get('#storage-unit-view').click();
                cy.get('[data-value="card"]').click();
                assertPersistedSetting('storageUnitView', 'card');

                // Now change to list
                cy.get('#storage-unit-view').click();
                cy.get('[data-value="list"]').click();
                assertPersistedSetting('storageUnitView', 'list');

                // Navigate to storage units page
                cy.goto('storage-unit');
                cy.get('[data-testid="scratchpad-button"]', {timeout: 10000}).should('be.visible');

                // Should see list view (table)
                cy.get('table').should('exist');
            });
        });

        describe('Where Condition Mode', () => {
            it('can change where condition mode to popover', () => {
                // First set to sheet to ensure we're changing from a known state
                cy.get('#where-condition-mode').click();
                cy.get('[data-value="sheet"]').click();
                assertPersistedSetting('whereConditionMode', 'sheet');

                // Now change to popover
                cy.get('#where-condition-mode').click();
                cy.get('[data-value="popover"]').click();
                cy.get('#where-condition-mode').should('contain', 'Popover');
                assertPersistedSetting('whereConditionMode', 'popover');
            });

            it('can change where condition mode to sheet', () => {
                // First set to popover to ensure we're changing from a known state
                cy.get('#where-condition-mode').click();
                cy.get('[data-value="popover"]').click();
                assertPersistedSetting('whereConditionMode', 'popover');

                // Now change to sheet
                cy.get('#where-condition-mode').click();
                cy.get('[data-value="sheet"]').click();
                cy.get('#where-condition-mode').should('contain', 'Sheet');
                assertPersistedSetting('whereConditionMode', 'sheet');
            });
        });

        describe('Default Page Size', () => {
            it('can change default page size', () => {
                // First set to 100 to ensure we're changing from a known state
                cy.get('#default-page-size').click();
                cy.get('[data-value="100"]').click();
                assertPersistedSetting('defaultPageSize', 100);

                // Now change to 25
                cy.get('#default-page-size').click();
                cy.get('[data-value="25"]').click();
                cy.get('#default-page-size').should('contain', '25');
                assertPersistedSetting('defaultPageSize', 25);
            });

            it('can set custom page size', () => {
                // First set to 50 to ensure we're changing from a known state
                cy.get('#default-page-size').click();
                cy.get('[data-value="50"]').click();
                assertPersistedSetting('defaultPageSize', 50);

                // Now change to custom
                cy.get('#default-page-size').click();
                cy.get('[data-value="custom"]').click();

                // Enter custom value and press Enter to trigger save
                cy.get('input[type="number"]').clear().type('75{enter}');

                // Verify custom input is displayed and persisted
                cy.get('input[type="number"]').should('have.value', '75');
                // Wait for redux-persist to save
                cy.wait(100);
                assertPersistedSetting('defaultPageSize', 75);
            });
        });

        describe('Telemetry Toggle', () => {
            it('can toggle telemetry off and persists to localStorage', () => {
                const telemetryButton = 'button[role="switch"]';

                // Check if telemetry toggle exists (CE only - not available in EE)
                cy.get('body').then($body => {
                    if ($body.find(telemetryButton).length === 0) {
                        // Telemetry toggle not present - skip test
                        cy.log('Telemetry toggle not available (EE mode) - skipping test');
                        return;
                    }

                    // First ensure telemetry is ON so we can test turning it OFF
                    cy.get(telemetryButton).first().invoke('attr', 'data-state').then((initialState) => {
                        if (initialState === 'unchecked') {
                            // Enable first
                            cy.get(telemetryButton).first().trigger('click');
                            cy.get(telemetryButton).first().should('have.attr', 'data-state', 'checked');
                            // Wait for state to persist
                            cy.wait(200);
                        }

                        // Verify telemetry is now enabled (before state)
                        cy.window().its('localStorage')
                            .invoke('getItem', 'whodb.analytics.consent')
                            .should('equal', 'granted');
                        assertPersistedSetting('metricsEnabled', true);

                        // Toggle telemetry OFF
                        cy.get(telemetryButton).first().trigger('click');
                        cy.get(telemetryButton).first().should('have.attr', 'data-state', 'unchecked');

                        // Wait for state to persist
                        cy.wait(200);

                        // Verify both localStorage keys are updated (after state)
                        cy.window().its('localStorage')
                            .invoke('getItem', 'whodb.analytics.consent')
                            .should('equal', 'denied');
                        assertPersistedSetting('metricsEnabled', false);
                    });
                });
            });
        });
    });

    // Theme toggle tests - run for any database since it's global
    forEachDatabase('sql', (db) => {
        // Only run for first SQL database to avoid redundant tests
        if (db.type !== 'Postgres') {
            return;
        }

        describe('Theme Toggle', () => {
            it('has theme toggle button visible on storage unit page', () => {
                cy.goto('storage-unit');
                cy.get('[data-testid="scratchpad-button"]', {timeout: 10000}).should('be.visible');

                // Mode toggle should be visible
                cy.get('[data-testid="mode-toggle"]').should('be.visible');
                cy.get('[data-testid="mode-toggle"] button').should('exist');
            });

            it('can toggle theme mode', () => {
                cy.goto('storage-unit');
                cy.get('[data-testid="scratchpad-button"]', {timeout: 10000}).should('be.visible');

                // Click the mode toggle button - use first() in case multiple found, scrollIntoView for visibility
                cy.get('[data-testid="mode-toggle"] button').first().scrollIntoView().click();

                // A dropdown menu should appear with theme options
                cy.get('[role="menu"]').should('be.visible');

                // Should have exactly 3 options: Light, Dark, System
                cy.get('[role="menuitem"]').should('have.length', 3);

                // Verify all options exist
                cy.get('[role="menuitem"]').contains(/light/i).should('be.visible');
                cy.get('[role="menuitem"]').contains(/dark/i).should('be.visible');
                cy.get('[role="menuitem"]').contains(/system/i).should('be.visible');

                // Close the menu by pressing Escape
                cy.get('body').type('{esc}');
            });

            it('can switch to light mode', () => {
                cy.goto('storage-unit');
                cy.get('[data-testid="scratchpad-button"]', {timeout: 10000}).should('be.visible');

                // Click the mode toggle
                cy.get('[data-testid="mode-toggle"] button').first().scrollIntoView().click();

                // Click light mode option
                cy.get('[role="menuitem"]').contains(/light/i).click();

                // Verify theme changed (html element should have class)
                cy.get('html').should('have.class', 'light');
            });

            it('can switch to dark mode', () => {
                cy.goto('storage-unit');
                cy.get('[data-testid="scratchpad-button"]', {timeout: 10000}).should('be.visible');

                // Click the mode toggle
                cy.get('[data-testid="mode-toggle"] button').first().scrollIntoView().click();

                // Click dark mode option
                cy.get('[role="menuitem"]').contains(/dark/i).click();

                // Verify theme changed
                cy.get('html').should('have.class', 'dark');
            });

            it('theme persists after navigation', () => {
                cy.goto('storage-unit');
                cy.get('[data-testid="scratchpad-button"]', {timeout: 10000}).should('be.visible');

                // Switch to light mode
                cy.get('[data-testid="mode-toggle"] button').first().scrollIntoView().click();
                cy.get('[role="menuitem"]').contains(/light/i).click();

                // Navigate to settings
                cy.goto('settings');

                // Theme should still be light
                cy.get('html').should('have.class', 'light');

                // Switch back to dark for other tests
                cy.get('[data-testid="mode-toggle"] button').first().scrollIntoView().click();
                cy.get('[role="menuitem"]').contains(/dark/i).click();
            });
        });
    });

});
