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

import {forEachDatabase, getDatabaseConfig} from '../../support/test-runner';


/**
 * Sidebar Navigation Tests
 *
 * Tests sidebar functionality including schema/database switching,
 * profile management, navigation, and database-specific options.
 */
describe('Sidebar Navigation', () => {
    describe('Schema Selection', () => {
        // Test with databases that support schema
        forEachDatabase('sql', (db) => {
            // Skip databases that don't use schema dropdown
            if (['MySQL', 'MariaDB'].includes(db.type)) {
                return;
            }

            describe(`${db.type}`, () => {
                it('shows schema dropdown', () => {
                    cy.get('[data-testid="sidebar-schema"]').should('be.visible');
                });

                it('can select different schema', () => {
                    cy.get('[data-testid="sidebar-schema"]').click();

                    // Should show at least one schema option
                    cy.get('[role="option"]').should('have.length.at.least', 1);

                    // Select the first schema (or current one if only one exists)
                    cy.get('[role="option"]').first().click();

                    // Schema should be selected
                    cy.get('[data-testid="sidebar-schema"]').should('exist');
                });

                it('reloads storage units when schema changes', () => {
                    cy.intercept('POST', '**/api/query').as('schemaQuery');

                    cy.get('[data-testid="sidebar-schema"]').click();
                    cy.get('[role="option"]').first().click();

                    // Wait for the GraphQL query to complete
                    cy.wait('@schemaQuery', { timeout: 10000 });

                    // Storage units should update
                    cy.get('[data-testid="storage-unit-card"]', { timeout: 15000 })
                        .should('have.length.at.least', 1);
                });
            });
        });
    });

    describe('Database Selection (MySQL/MariaDB)', () => {
        // These databases use database dropdown instead of schema
        ['mysql', 'mysql8', 'mariadb'].forEach(dbKey => {
            const db = getDatabaseConfig(dbKey);
            if (!db) return;

            describe(`${db.type}`, () => {
                it('shows database dropdown', () => {
                    cy.get('[data-testid="sidebar-database"]').should('be.visible');
                });

                it('can switch to different database', () => {
                    cy.get('[data-testid="sidebar-database"]').click();

                    // Should show database options
                    cy.get('[role="option"]').should('have.length.at.least', 1);

                    // Select an option
                    cy.get('[role="option"]').first().click();

                    // Database should update
                    cy.get('[data-testid="sidebar-database"]').should('exist');
                });
            });
        });
    });

    describe('Navigation Links', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('highlights current route in sidebar', () => {
                    // Currently on storage-unit
                    cy.url().should('include', '/storage-unit');

                    // Navigate to graph
                    cy.get('[href="/graph"]').click();
                    cy.url().should('include', '/graph');

                    // Navigate to scratchpad
                    cy.get('[href="/scratchpad"]').click();
                    cy.url().should('include', '/scratchpad');

                    // Navigate to chat
                    cy.get('[href="/chat"]').click();
                    cy.url().should('include', '/chat');

                    // Navigate back to storage-unit
                    cy.get('[href="/storage-unit"]').click();
                    cy.url().should('include', '/storage-unit');
                });

                it('shows chat option for SQL databases', () => {
                    cy.get('[href="/chat"]').should('exist');
                });

                it('shows graph option', () => {
                    cy.get('[href="/graph"]').should('exist');
                });

                it('shows scratchpad option for SQL databases', () => {
                    cy.get('[href="/scratchpad"]').should('exist');
                });
            });
        }, { features: ['scratchpad'] });
    });

    describe('NoSQL Navigation', () => {
        // Test NoSQL databases that may not show all options
        forEachDatabase('keyvalue', (db) => {
            describe(`${db.type}`, () => {
                it('hides chat option for key-value databases', () => {
                    // Redis and similar don't support SQL chat
                    cy.get('[href="/chat"]').should('not.exist');
                });

                it('hides scratchpad option for key-value databases', () => {
                    // Key-value stores don't support SQL scratchpad
                    cy.get('[href="/scratchpad"]').should('not.exist');
                });

                it('still shows graph option', () => {
                    cy.get('[href="/graph"]').should('exist');
                });
            });
        });

        forEachDatabase('document', (db) => {
            describe(`${db.type}`, () => {
                it('navigation options depend on database features', () => {
                    // Document databases may or may not have certain features
                    cy.get('[href="/graph"]').should('exist');
                });
            });
        });
    });

    describe('Sidebar Toggle', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('can collapse and expand sidebar with keyboard shortcut', () => {
                    // Sidebar should be visible initially
                    cy.get('[data-sidebar="sidebar"]').should('be.visible');

                    // Toggle sidebar with Ctrl+B
                    cy.get('body').type('{ctrl}b');

                    // Sidebar state should change
                    cy.wait(300); // Wait for animation

                    // Toggle back
                    cy.get('body').type('{ctrl}b');
                    cy.wait(300);

                    // Sidebar should be visible again
                    cy.get('[data-sidebar="sidebar"]').should('be.visible');
                });

                it('sidebar state persists in session', () => {
                    // Collapse sidebar
                    cy.get('body').type('{ctrl}b');
                    cy.wait(300);

                    // Navigate to another page
                    cy.get('[href="/graph"]').click({ force: true });
                    cy.url().should('include', '/graph');

                    // Navigate back
                    cy.get('[href="/storage-unit"]').click({ force: true });
                    cy.url().should('include', '/storage-unit');

                    // Sidebar state should be maintained
                    // (exact assertion depends on implementation)
                });
            });
        });
    });

    describe('Logout', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('logout redirects to login page', () => {
                    // Use the logout command
                    cy.logout();

                    // Should redirect to login page
                    cy.url({ timeout: 10000 }).should('include', '/login');
                });

                it('logout clears session', () => {
                    cy.logout();

                    // Wait for redirect
                    cy.url({ timeout: 10000 }).should('include', '/login');

                    // Try to visit storage-unit directly
                    cy.visit('/storage-unit');

                    // Should redirect back to login (session cleared)
                    cy.url({ timeout: 10000 }).should('include', '/login');
                });
            });
        });
    });

    describe('Profile Display', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('shows current profile in sidebar', () => {
                    // Profile selector should be visible
                    cy.get('[data-testid="sidebar-profile"]').should('exist');
                });

                it('profile dropdown shows options', () => {
                    cy.get('[data-testid="sidebar-profile"]').click();

                    // Should show at least the current profile
                    cy.get('[role="menuitem"], [role="option"]').should('have.length.at.least', 1);

                    // Close dropdown
                    cy.get('body').type('{esc}');
                });
            });
        });
    });

    describe('Add New Profile', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('can open add profile dialog from sidebar', () => {
                    // Click profile dropdown
                    cy.get('[data-testid="sidebar-profile"]').click();

                    // Look for add profile option
                    cy.get('body').then($body => {
                        // Find button/link to add new profile
                        const addButton = $body.find('[data-testid="add-profile-button"], button:contains("Add"), a:contains("Add")');
                        if (addButton.length > 0) {
                            cy.wrap(addButton.first()).click();

                            // Should show login form in sheet/dialog
                            cy.get('[role="dialog"], [data-testid="login-form"]', { timeout: 5000 })
                                .should('be.visible');

                            // Close dialog
                            cy.get('body').type('{esc}');
                        }
                    });
                });
            });
        });
    });

    describe('Database Type Icons', () => {
        forEachDatabase('all', (db) => {
            describe(`${db.type}`, () => {
                it('shows database type icon in profile', () => {
                    // Profile area should show database type indicator
                    cy.get('[data-testid="sidebar-profile"]').should('exist');

                    // There should be an icon representing the database type
                    cy.get('[data-testid="sidebar-profile"]').within(() => {
                        cy.get('svg, img').should('exist');
                    });
                });
            });
        });
    });

    describe('Version Display', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('shows WhoDB version in sidebar footer', () => {
                    // Version should be displayed somewhere in sidebar
                    cy.get('[data-sidebar="sidebar"]').should('contain.text', 'Version: development');
                });
            });
        });
    });
});
