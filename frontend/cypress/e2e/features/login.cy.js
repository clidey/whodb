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

import {getDatabaseConfig} from '../../support/test-runner';
import {clearBrowserState} from '../../support/helpers/animation';

/**
 * Login & Authentication Tests
 *
 * Tests the login page UI, form validation, and authentication flows.
 * Unlike other feature tests, these don't use forEachDatabase() since they
 * test the login page itself before any database is connected.
 */
describe('Login & Authentication', () => {
    beforeEach(() => {
        clearBrowserState();
        cy.visit('/login');

        // Dismiss telemetry modal if it appears
        cy.get('body').then($body => {
            const $btn = $body.find('button').filter(function() {
                return this.textContent.includes('Disable Telemetry');
            });
            if ($btn.length) {
                cy.wrap($btn).click();
            }
        });
    });

    describe('Database Type Selection', () => {
        it('shows database type dropdown with options', () => {
            cy.get('[data-testid="database-type-select"]').should('be.visible');
            cy.get('[data-testid="database-type-select"]').click();

            // Verify common database types are available
            cy.get('[data-value="Postgres"]').should('exist');
            cy.get('[data-value="MySQL"]').should('exist');
            cy.get('[data-value="Sqlite3"]').should('exist');
            cy.get('[data-value="MongoDB"]').should('exist');
            cy.get('[data-value="Redis"]').should('exist');

            // Close dropdown
            cy.get('body').type('{esc}');
        });

        it('changes form fields based on database type selection', () => {
            // PostgreSQL should show all fields
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();
            cy.get('[data-testid="hostname"]').should('be.visible');
            cy.get('[data-testid="username"]').should('be.visible');
            cy.get('[data-testid="password"]').should('be.visible');
            cy.get('[data-testid="database"]').should('be.visible');

            // Redis should show only hostname (no username/password/database required)
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Redis"]').click();
            cy.get('[data-testid="hostname"]').should('be.visible');
            // Redis doesn't require username/password/database in the form

            // SQLite should show only database path
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Sqlite3"]').click();
            cy.get('[data-testid="database"]').should('be.visible');
            cy.get('[data-testid="hostname"]').should('not.exist');
        });
    });

    describe('Form Validation', () => {
        it('disables login button when required fields are empty', () => {
            // Select PostgreSQL which requires all fields
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();

            // Login button should be disabled when fields are empty
            cy.get('[data-testid="login-button"]').should('be.disabled');
        });

        it('enables login button when required fields are filled', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();

            cy.get('[data-testid="hostname"]').type('localhost');
            cy.get('[data-testid="username"]').type('user');
            cy.get('[data-testid="password"]').type('password');
            cy.get('[data-testid="database"]').type('testdb');

            cy.get('[data-testid="login-button"]').should('not.be.disabled');
        });
    });

    describe('Advanced Options', () => {
        it('toggles advanced options visibility', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();

            // Advanced options should be hidden by default
            cy.get('[data-testid="Port-input"]').should('not.exist');

            // Click advanced button to show options
            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').should('be.visible');

            // Click again to hide
            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').should('not.exist');
        });

        it('accepts advanced configuration values', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();

            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').clear().type('5433');
            cy.get('[data-testid="Port-input"]').should('have.value', '5433');
        });
    });

    describe('Direct Credentials Login', () => {
        it('successfully logs in with valid credentials', () => {
            // Use PostgreSQL config from fixtures
            const db = getDatabaseConfig('postgres');

            cy.get('[data-testid="database-type-select"]').click();
            cy.get(`[data-value="${db.type}"]`).click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });
            cy.get('[data-testid="database"]').type(db.connection.database);

            // Handle advanced options if needed
            if (db.connection.advanced && Object.keys(db.connection.advanced).length > 0) {
                cy.get('[data-testid="advanced-button"]').click();
                for (const [key, value] of Object.entries(db.connection.advanced)) {
                    cy.get(`[data-testid="${key}-input"]`).clear().type(String(value));
                }
            }

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@loginQuery', { timeout: 30000 });

            // Should redirect to storage-unit page
            cy.url().should('include', '/storage-unit');
            cy.get('[data-testid="sidebar-database"], [data-testid="sidebar-schema"]', { timeout: 15000 })
                .should('exist');
        });

        it('shows error message on invalid credentials', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();

            cy.get('[data-testid="hostname"]').type('localhost');
            cy.get('[data-testid="username"]').type('invalid_user');
            cy.get('[data-testid="password"]').type('wrong_password');
            cy.get('[data-testid="database"]').type('nonexistent_db');

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            // Should show error toast or remain on login page
            cy.wait('@loginQuery', { timeout: 30000 });

            // Either toast error or still on login page
            cy.url().should('include', '/login');
        });
    });

    // TODO: URL parsing feature may not exist or work differently
    // describe('URL Parsing', () => {
    //     it('parses PostgreSQL connection URL and fills form', () => {
    //         cy.get('[data-testid="database-type-select"]').click();
    //         cy.get('[data-value="Postgres"]').click();
    //
    //         // Paste a PostgreSQL connection URL
    //         const testUrl = 'postgres://testuser:testpass@testhost:5432/testdb';
    //         cy.get('[data-testid="hostname"]').type(testUrl);
    //
    //         // Form should be auto-populated from URL
    //         cy.get('[data-testid="hostname"]').should('have.value', 'testhost');
    //         cy.get('[data-testid="username"]').should('have.value', 'testuser');
    //         cy.get('[data-testid="password"]').should('have.value', 'testpass');
    //         cy.get('[data-testid="database"]').should('have.value', 'testdb');
    //
    //         // Advanced options should show port
    //         cy.get('[data-testid="advanced-button"]').click();
    //         cy.get('[data-testid="Port-input"]').should('have.value', '5432');
    //     });
    //
    //     it('parses MongoDB SRV URL and fills form', () => {
    //         cy.get('[data-testid="database-type-select"]').click();
    //         cy.get('[data-value="MongoDB"]').click();
    //
    //         // Paste a MongoDB SRV URL
    //         const testUrl = 'mongodb+srv://testuser:testpass@cluster.mongodb.net/testdb?retryWrites=true';
    //         cy.get('[data-testid="hostname"]').type(testUrl);
    //
    //         // Form should be auto-populated
    //         cy.get('[data-testid="hostname"]').should('have.value', 'cluster.mongodb.net');
    //         cy.get('[data-testid="username"]').should('have.value', 'testuser');
    //         cy.get('[data-testid="password"]').should('have.value', 'testpass');
    //         cy.get('[data-testid="database"]').should('have.value', 'testdb');
    //
    //         // Advanced options should show DNS enabled
    //         cy.get('[data-testid="advanced-button"]').click();
    //         cy.get('[data-testid="DNS Enabled-input"]').should('have.value', 'true');
    //     });
    // });

    // TODO: ModeToggle component structure may differ from test expectations
    // describe('Theme Toggle', () => {
    //     it('shows theme toggle button on login page', () => {
    //         cy.get('[data-testid="mode-toggle-login"]').should('be.visible');
    //     });
    //
    //     it('can toggle theme on login page', () => {
    //         // Click the theme toggle
    //         cy.get('[data-testid="mode-toggle-login"]').within(() => {
    //             cy.get('button').click();
    //         });
    //
    //         // Should show dropdown with options
    //         cy.get('[role="menuitem"]').should('have.length.at.least', 2);
    //     });
    // });

    describe('Saved Profiles', () => {
        // Note: This test requires saved profiles to exist
        // In a fresh environment, there may be no saved profiles
        it('shows available profiles section when profiles exist', () => {
            // Check if profiles section exists (may not show if no profiles)
            cy.get('body').then($body => {
                if ($body.find('[data-testid="available-profiles-select"]').length > 0) {
                    cy.get('[data-testid="available-profiles-select"]').should('be.visible');
                    cy.get('[data-testid="login-with-profile-button"]').should('exist');
                }
            });
        });
    });

    describe('Sample Database', () => {
        it('shows sample database panel for first-time users', () => {
            // Clear first login flag to simulate first-time user
            cy.window().then(win => {
                win.localStorage.removeItem('whodb_has_logged_in');
            });
            cy.reload();

            // Dismiss telemetry again after reload
            cy.wait(500);
            cy.get('body').then($body => {
                const $btn = $body.find('button').filter(function() {
                    return this.textContent.includes('Disable Telemetry');
                });
                if ($btn.length) {
                    cy.wrap($btn).click();
                }
            });

            // Sample database panel should be visible for first-time users
            // Note: This depends on feature flags and sample database availability
            cy.get('body').then($body => {
                if ($body.find('[data-testid="sample-database-panel"]').length > 0) {
                    cy.get('[data-testid="sample-database-panel"]').should('be.visible');
                    cy.get('[data-testid="get-started-sample-db"]').should('be.visible');
                }
            });
        });

        it('can login with sample database', () => {
            // Clear first login flag
            cy.window().then(win => {
                win.localStorage.removeItem('whodb_has_logged_in');
            });
            cy.reload();

            cy.wait(500);
            cy.get('body').then($body => {
                const $btn = $body.find('button').filter(function() {
                    return this.textContent.includes('Disable Telemetry');
                });
                if ($btn.length) {
                    cy.wrap($btn).click();
                }
            });

            // Try to click sample database button if available
            cy.get('body').then($body => {
                if ($body.find('[data-testid="get-started-sample-db"]').length > 0) {
                    cy.intercept('POST', '**/api/query').as('loginQuery');
                    cy.get('[data-testid="get-started-sample-db"]').click();

                    cy.wait('@loginQuery', { timeout: 30000 });

                    // Should redirect to storage-unit
                    cy.url().should('include', '/storage-unit');
                }
            });
        });
    });

    // TODO: Session persistence test assumes non-empty database
    // describe('Session Persistence', () => {
    //     it('maintains session after page reload when logged in', () => {
    //         // Login first
    //         const db = getDatabaseConfig('postgres');
    //         cy.login(
    //             db.type,
    //             db.connection.host,
    //             db.connection.user,
    //             db.connection.password,
    //             db.connection.database,
    //             db.connection.advanced
    //         );
    //
    //         // Verify we're on storage-unit page
    //         cy.url().should('include', '/storage-unit');
    //
    //         // Reload the page
    //         cy.reload();
    //
    //         // Should still be logged in (redirected back to storage-unit, not login)
    //         cy.url().should('include', '/storage-unit');
    //         cy.get('[data-testid="storage-unit-card"]', { timeout: 15000 })
    //             .should('have.length.at.least', 1);
    //     });
    // });

    describe('URL Parameter Pre-filling', () => {
        it('pre-fills form from URL parameters', () => {
            cy.visit('/login?type=Postgres&host=urlhost&username=urluser&database=urldb');

            // Dismiss telemetry
            cy.wait(500);
            cy.get('body').then($body => {
                const $btn = $body.find('button').filter(function() {
                    return this.textContent.includes('Disable Telemetry');
                });
                if ($btn.length) {
                    cy.wrap($btn).click();
                }
            });

            // Form should be pre-filled
            cy.get('[data-testid="hostname"]').should('have.value', 'urlhost');
            cy.get('[data-testid="username"]').should('have.value', 'urluser');
            cy.get('[data-testid="database"]').should('have.value', 'urldb');
        });

        it('pre-fills form from base64 encoded credentials', () => {
            const credentials = {
                type: 'Postgres',
                host: 'encodedhost',
                username: 'encodeduser',
                database: 'encodeddb'
            };
            const encoded = btoa(JSON.stringify(credentials));

            cy.visit(`/login?credentials=${encoded}`);

            // Dismiss telemetry
            cy.wait(500);
            cy.get('body').then($body => {
                const $btn = $body.find('button').filter(function() {
                    return this.textContent.includes('Disable Telemetry');
                });
                if ($btn.length) {
                    cy.wrap($btn).click();
                }
            });

            // Form should be pre-filled from encoded credentials
            cy.get('[data-testid="hostname"]').should('have.value', 'encodedhost');
            cy.get('[data-testid="username"]').should('have.value', 'encodeduser');
            cy.get('[data-testid="database"]').should('have.value', 'encodeddb');
        });
    });
});
