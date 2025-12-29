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

import {getDatabaseConfig, loginToDatabase} from '../../support/test-runner';
import {clearBrowserState} from '../../support/helpers/animation';

const targetDb = Cypress.env('database');
const shouldRun = !targetDb || targetDb.toLowerCase() === 'postgres';

/**
 * Browser Storage Tests
 *
 * Tests localStorage, sessionStorage, and Redux persist functionality
 * including state persistence across page reloads, logout behavior,
 * and storage key management.
 */
(shouldRun ? describe : describe.skip)('Browser Storage', () => {
    const db = getDatabaseConfig('postgres');

    beforeEach(() => {
        loginToDatabase(db);
    });

    afterEach(() => {
        cy.logout();
    });

    describe('Redux State Persistence', () => {
        it('persists auth state across page reload', () => {
            // Verify we're logged in
            cy.url().should('include', '/storage-unit');

            // Check that auth state exists in localStorage
            cy.window().then((win) => {
                const authData = win.localStorage.getItem('persist:auth');
                expect(authData).to.not.be.null;

                const parsed = JSON.parse(authData);
                expect(parsed.status).to.exist;
                expect(JSON.parse(parsed.status)).to.equal('logged-in');
                expect(parsed.profiles).to.exist;
                expect(parsed.current).to.exist;
            });

            // Reload the page
            cy.reload();

            // Should still be logged in without redirecting to login
            cy.url({timeout: 10000}).should('include', '/storage-unit');
            cy.get('[data-testid="sidebar-profile"]').should('exist');
        });

        it('persists database state across page reload', () => {
            // Check that database state exists in localStorage
            cy.window().then((win) => {
                const databaseData = win.localStorage.getItem('persist:database');
                expect(databaseData).to.not.be.null;

                const parsed = JSON.parse(databaseData);
                expect(parsed).to.have.property('_persist');
            });

            // Reload the page
            cy.reload();

            // Database state should still be available
            cy.get('[data-testid="sidebar-profile"]').should('exist');
        });

        it('persists settings state across page reload', () => {
            // Navigate to storage-unit and set a preference
            cy.visit('/storage-unit');

            // Change storage unit view to list via localStorage
            cy.window().then((win) => {
                const settingsData = win.localStorage.getItem('persist:settings');
                const parsed = JSON.parse(settingsData || '{}');
                parsed.storageUnitView = '"list"';
                win.localStorage.setItem('persist:settings', JSON.stringify(parsed));
            });

            // Reload the page
            cy.reload();

            // Setting should persist
            cy.window().then((win) => {
                const settingsData = win.localStorage.getItem('persist:settings');
                const parsed = JSON.parse(settingsData);
                expect(JSON.parse(parsed.storageUnitView)).to.equal('list');
            });
        });

        it('persists all Redux slices with correct keys', () => {
            cy.window().then((win) => {
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

                expectedKeys.forEach(key => {
                    const data = win.localStorage.getItem(key);
                    expect(data, `${key} should exist in localStorage`).to.not.be.null;

                    // Each should have _persist property
                    const parsed = JSON.parse(data);
                    expect(parsed._persist, `${key} should have _persist`).to.exist;
                });
            });
        });

        it('maintains Redux state structure after reload', () => {
            let beforeReloadAuth;
            let beforeReloadDatabase;

            // Capture state before reload
            cy.window().then((win) => {
                beforeReloadAuth = win.localStorage.getItem('persist:auth');
                beforeReloadDatabase = win.localStorage.getItem('persist:database');
            });

            // Reload the page
            cy.reload();

            // Verify state structure is maintained
            cy.window().then((win) => {
                const afterReloadAuth = win.localStorage.getItem('persist:auth');
                const afterReloadDatabase = win.localStorage.getItem('persist:database');

                expect(afterReloadAuth).to.equal(beforeReloadAuth);
                expect(afterReloadDatabase).to.equal(beforeReloadDatabase);
            });
        });
    });

    describe('Settings Persistence', () => {
        it('persists storage unit view preference', () => {
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
                settings.storageUnitView = '"list"';
                win.localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            cy.reload();

            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings'));
                expect(JSON.parse(settings.storageUnitView)).to.equal('list');
            });
        });

        it('persists font size preference', () => {
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
                settings.fontSize = '"large"';
                win.localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            cy.reload();

            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings'));
                expect(JSON.parse(settings.fontSize)).to.equal('large');
            });
        });

        it('persists border radius preference', () => {
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
                settings.borderRadius = '"none"';
                win.localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            cy.reload();

            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings'));
                expect(JSON.parse(settings.borderRadius)).to.equal('none');
            });
        });

        it('persists spacing preference', () => {
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
                settings.spacing = '"spacious"';
                win.localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            cy.reload();

            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings'));
                expect(JSON.parse(settings.spacing)).to.equal('spacious');
            });
        });

        it('persists where condition mode preference', () => {
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
                settings.whereConditionMode = '"sheet"';
                win.localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            cy.reload();

            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings'));
                expect(JSON.parse(settings.whereConditionMode)).to.equal('sheet');
            });
        });

        it('persists default page size preference', () => {
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
                settings.defaultPageSize = '50';
                win.localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            cy.reload();

            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings'));
                expect(JSON.parse(settings.defaultPageSize)).to.equal(50);
            });
        });

        it('persists language preference', () => {
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
                settings.language = '"es"';
                win.localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            cy.reload();

            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings'));
                expect(JSON.parse(settings.language)).to.equal('es');
            });
        });

        it('persists metrics enabled preference', () => {
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
                settings.metricsEnabled = 'false';
                win.localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            cy.reload();

            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings'));
                expect(JSON.parse(settings.metricsEnabled)).to.equal(false);
            });
        });
    });

    describe('Logout Storage Cleanup', () => {
        it('clears Redux auth state on logout', () => {
            // Verify auth state exists before logout
            cy.window().then((win) => {
                const authData = win.localStorage.getItem('persist:auth');
                const parsed = JSON.parse(authData);
                expect(JSON.parse(parsed.status)).to.equal('logged-in');
            });

            // Logout
            cy.logout();
            cy.url({timeout: 10000}).should('include', '/login');

            // Auth state should be cleared (status = unauthorized, profiles empty)
            cy.window().then((win) => {
                const authData = win.localStorage.getItem('persist:auth');
                if (authData) {
                    const parsed = JSON.parse(authData);
                    expect(JSON.parse(parsed.status)).to.equal('unauthorized');
                    const profiles = JSON.parse(parsed.profiles);
                    expect(profiles).to.have.length(0);
                }
            });
        });

        it('preserves settings after logout', () => {
            // Set a custom setting
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
                settings.storageUnitView = '"list"';
                win.localStorage.setItem('persist:settings', JSON.stringify(settings));
            });

            // Logout
            cy.logout();
            cy.url({timeout: 10000}).should('include', '/login');

            // Settings should still exist
            cy.window().then((win) => {
                const settings = JSON.parse(win.localStorage.getItem('persist:settings'));
                expect(JSON.parse(settings.storageUnitView)).to.equal('list');
            });
        });

        it('preserves first login flag after logout', () => {
            // First login flag should be set after initial login
            cy.window().then((win) => {
                const hasLoggedIn = win.localStorage.getItem('whodb_has_logged_in');
                expect(hasLoggedIn).to.equal('true');
            });

            // Logout
            cy.logout();
            cy.url({timeout: 10000}).should('include', '/login');

            // First login flag should persist
            cy.window().then((win) => {
                const hasLoggedIn = win.localStorage.getItem('whodb_has_logged_in');
                expect(hasLoggedIn).to.equal('true');
            });
        });

        it('clears current profile but preserves persist keys', () => {
            // Logout
            cy.logout();
            cy.url({timeout: 10000}).should('include', '/login');

            // Redux persist keys should still exist but with cleared data
            cy.window().then((win) => {
                const authData = win.localStorage.getItem('persist:auth');
                expect(authData).to.not.be.null;

                const parsed = JSON.parse(authData);
                expect(parsed._persist).to.exist;
            });
        });

        it('prevents access to protected routes after logout', () => {
            // Logout
            cy.logout();
            cy.url({timeout: 10000}).should('include', '/login');

            // Try to visit a protected route
            cy.visit('/storage-unit');

            // Should redirect back to login
            cy.url({timeout: 10000}).should('include', '/login');
        });
    });

    describe('First Login Flag', () => {
        it('sets first login flag on initial login', () => {
            clearBrowserState();

            // Clear the first login flag specifically
            cy.window().then((win) => {
                win.localStorage.removeItem('whodb_has_logged_in');
            });

            // Login
            loginToDatabase(db);

            // First login flag should be set
            cy.window().then((win) => {
                const hasLoggedIn = win.localStorage.getItem('whodb_has_logged_in');
                expect(hasLoggedIn).to.equal('true');
            });
        });

        it('first login flag persists across page reloads', () => {
            clearBrowserState();
            loginToDatabase(db);

            // Verify flag is set
            cy.window().then((win) => {
                const hasLoggedIn = win.localStorage.getItem('whodb_has_logged_in');
                expect(hasLoggedIn).to.equal('true');
            });

            // Reload the page
            cy.reload();

            // Flag should still be set
            cy.window().then((win) => {
                const hasLoggedIn = win.localStorage.getItem('whodb_has_logged_in');
                expect(hasLoggedIn).to.equal('true');
            });
        });

        it('first login flag persists after logout', () => {
            clearBrowserState();
            loginToDatabase(db);

            // Logout
            cy.logout();
            cy.url({timeout: 10000}).should('include', '/login');

            // Flag should still be set
            cy.window().then((win) => {
                const hasLoggedIn = win.localStorage.getItem('whodb_has_logged_in');
                expect(hasLoggedIn).to.equal('true');
            });
        });
    });

    describe('Analytics Consent', () => {
        beforeEach(() => {
            clearBrowserState();
        });

        it('analytics consent is set to denied by clearBrowserState', () => {
            cy.window().then((win) => {
                const consent = win.localStorage.getItem('whodb.analytics.consent');
                expect(consent).to.equal('denied');
            });
        });

        it('analytics consent persists across page navigation', () => {
            loginToDatabase(db);

            cy.window().then((win) => {
                const consent = win.localStorage.getItem('whodb.analytics.consent');
                expect(consent).to.equal('denied');
            });

            // Navigate to another page
            cy.visit('/graph');

            cy.window().then((win) => {
                const consent = win.localStorage.getItem('whodb.analytics.consent');
                expect(consent).to.equal('denied');
            });
        });
    });

    describe('Sidebar State Persistence', () => {
        it('sidebar toggle state persists during navigation', () => {
            // Sidebar should be visible initially
            cy.get('[data-sidebar="sidebar"]').should('be.visible');

            // Toggle sidebar
            cy.get('body').type('{ctrl}b');
            cy.wait(300);

            // Navigate to graph
            cy.visit('/graph');
            cy.wait(500);

            // Navigate back to storage-unit
            cy.visit('/storage-unit');
            cy.wait(500);

            // Sidebar state is maintained through navigation
            cy.get('[data-sidebar="sidebar"]').should('exist');
        });

        it('sidebar can be toggled multiple times', () => {
            // Initial state - visible
            cy.get('[data-sidebar="sidebar"]').should('be.visible');

            // Toggle off
            cy.get('body').type('{ctrl}b');
            cy.wait(300);

            // Toggle on
            cy.get('body').type('{ctrl}b');
            cy.wait(300);

            // Should be visible again
            cy.get('[data-sidebar="sidebar"]').should('be.visible');
        });
    });

    describe('Storage Size and Limits', () => {
        it('localStorage contains expected data after login', () => {
            cy.window().then((win) => {
                const storageKeys = Object.keys(win.localStorage);

                // Should have multiple persist keys
                const persistKeys = storageKeys.filter(key => key.startsWith('persist:'));
                expect(persistKeys.length).to.be.greaterThan(5);

                // Should have analytics consent
                expect(storageKeys).to.include('whodb.analytics.consent');

                // Should have first login flag
                expect(storageKeys).to.include('whodb_has_logged_in');
            });
        });

        it('localStorage data is valid JSON', () => {
            cy.window().then((win) => {
                const storageKeys = Object.keys(win.localStorage);

                storageKeys.forEach(key => {
                    if (key.startsWith('persist:')) {
                        const data = win.localStorage.getItem(key);
                        expect(() => JSON.parse(data), `${key} should be valid JSON`).to.not.throw();
                    }
                });
            });
        });

        it('sessionStorage is not used for persistence', () => {
            cy.window().then((win) => {
                // WhoDB uses localStorage for Redux persist, not sessionStorage
                const sessionKeys = Object.keys(win.sessionStorage);
                const persistKeys = sessionKeys.filter(key => key.startsWith('persist:'));

                // Should have no persist keys in sessionStorage
                expect(persistKeys.length).to.equal(0);
            });
        });
    });

    describe('Profile Management Storage', () => {
        it('stores profile information in auth state', () => {
            cy.window().then((win) => {
                const authData = JSON.parse(win.localStorage.getItem('persist:auth'));
                const profiles = JSON.parse(authData.profiles);

                // Should have at least one profile
                expect(profiles.length).to.be.greaterThan(0);

                // Each profile should have required fields
                const profile = profiles[0];
                expect(profile).to.have.property('Id');
                expect(profile).to.have.property('Type');
                expect(profile).to.have.property('Hostname');
            });
        });

        it('stores current profile in auth state', () => {
            cy.window().then((win) => {
                const authData = JSON.parse(win.localStorage.getItem('persist:auth'));
                const current = JSON.parse(authData.current);

                // Current profile should exist
                expect(current).to.not.be.null;
                expect(current).to.have.property('Id');
                expect(current).to.have.property('Type');
            });
        });

        it('profile data persists after navigation', () => {
            let profileId;

            // Get the current profile ID
            cy.window().then((win) => {
                const authData = JSON.parse(win.localStorage.getItem('persist:auth'));
                const current = JSON.parse(authData.current);
                profileId = current.Id;
            });

            // Navigate to another page
            cy.visit('/graph');
            cy.wait(500);

            // Profile should remain the same
            cy.window().then((win) => {
                const authData = JSON.parse(win.localStorage.getItem('persist:auth'));
                const current = JSON.parse(authData.current);
                expect(current.Id).to.equal(profileId);
            });
        });
    });

    describe('Corrupted Data Handling', () => {
        beforeEach(() => {
            clearBrowserState();
        });

        it('handles corrupted Redux persist data gracefully', () => {
            // Set corrupted data
            cy.window().then((win) => {
                win.localStorage.setItem('persist:settings', 'invalid json{');
            });

            // Login should still work (Redux persist will reset corrupted data)
            loginToDatabase(db);

            // Should be logged in successfully
            cy.url().should('include', '/storage-unit');
        });

        it('clears corrupted scratchpad data on load', () => {
            // Set corrupted scratchpad data with invalid dates
            cy.window().then((win) => {
                const corruptedData = {
                    cells: {
                        'cell-1': {
                            history: [
                                {date: 'invalid-date', query: 'SELECT 1'}
                            ]
                        }
                    },
                    _persist: '{"version":-1,"rehydrated":true}'
                };
                win.localStorage.setItem('persist:scratchpad', JSON.stringify(corruptedData));
            });

            // Visit the app - should handle corrupted data
            cy.visit('/login');

            // Should not crash
            cy.get('[data-testid="database-type-select"]').should('exist');
        });
    });
});
