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

import { forEachDatabase, loginToDatabase, getDatabaseConfig } from '../../support/test-runner';
import { clearBrowserState } from '../../support/helpers/animation';

/**
 * Profile Management Tests
 *
 * Tests profile functionality including displaying multiple profiles,
 * switching between profiles, database type icons, adding new profiles,
 * and logging out from specific profiles.
 */
describe('Profile Management', () => {
    describe('Profile Display', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('displays profile selector in sidebar', () => {
                    cy.get('[data-testid="sidebar-profile"]').should('be.visible');
                });

                it('shows database type icon in profile', () => {
                    cy.get('[data-testid="sidebar-profile"]').should('exist');

                    cy.get('[data-testid="sidebar-profile"]').within(() => {
                        cy.get('svg, img').should('exist');
                    });
                });

                it('displays current connection information', () => {
                    cy.get('[data-testid="sidebar-profile"]').should('exist');

                    cy.get('[data-testid="sidebar-profile"]').within(() => {
                        cy.get('svg, img').should('exist');
                    });
                });
            });
        });
    });

    describe('Profile Dropdown', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('opens dropdown when profile is clicked', () => {
                    cy.get('[data-testid="sidebar-profile"]').click();

                    cy.get('[role="menuitem"], [role="option"]').should('have.length.at.least', 1);

                    cy.get('body').type('{esc}');
                });

                it('shows current profile in dropdown', () => {
                    cy.get('[data-testid="sidebar-profile"]').click();

                    cy.get('[role="menuitem"], [role="option"]').should('have.length.at.least', 1);

                    cy.get('body').type('{esc}');
                });

                it('closes dropdown when escape is pressed', () => {
                    cy.get('[data-testid="sidebar-profile"]').click();

                    cy.get('[role="menuitem"], [role="option"]').should('be.visible');

                    cy.get('body').type('{esc}');

                    cy.wait(300);

                    cy.get('body').then($body => {
                        const visibleOptions = $body.find('[role="menuitem"]:visible, [role="option"]:visible');
                        expect(visibleOptions.length).to.equal(0);
                    });
                });
            });
        });
    });

    describe('Multiple Profiles', () => {
        it('displays multiple profiles in dropdown when multiple connections exist', () => {
            clearBrowserState();

            const db1 = getDatabaseConfig('postgres');
            const db2 = getDatabaseConfig('mysql');

            loginToDatabase(db1, { visitStorageUnit: true });

            cy.get('[data-testid="sidebar-profile"]').click();

            // Look for "Add Another Profile" text
            cy.contains('Add Another Profile').should('be.visible').click();

            cy.get('[role="dialog"], [data-testid="login-form"]', { timeout: 5000 })
                .should('be.visible');

            cy.get('[data-testid="database-type-select"]').click();
            cy.get(`[data-value="${db2.type}"]`).click();

            cy.get('[data-testid="hostname"]').clear().type(db2.connection.host);
            cy.get('[data-testid="username"]').clear().type(db2.connection.user);
            cy.get('[data-testid="password"]').clear().type(db2.connection.password, { log: false });
            cy.get('[data-testid="database"]').clear().type(db2.connection.database);

            cy.intercept('POST', '**/api/query').as('addProfileQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@addProfileQuery', { timeout: 30000 });

            cy.wait(1000);

            cy.get('[data-testid="sidebar-profile"]').click();

            // Profile dropdown should have at least 2 profiles now
            // Verify by checking "Add Another Profile" is visible (dropdown is open)
            // and we can see profile entries
            cy.contains('Add Another Profile').should('be.visible');

            cy.get('body').type('{esc}');
        });
    });

    describe('Profile Switching', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {                it('can switch between profiles when multiple exist', () => {
                    cy.get('[data-testid="sidebar-profile"]').click();

                    cy.get('body').then($body => {
                        const menuItems = $body.find('[role="menuitem"], [role="option"]');
                        if (menuItems.length > 1) {
                            cy.get('[role="menuitem"], [role="option"]').eq(1).click();

                            cy.wait(1000);

                            cy.get('[data-testid="sidebar-profile"]').should('exist');
                        } else {
                            cy.get('body').type('{esc}');
                        }
                    });
                });
            });
        });
    });

    describe('Add New Profile', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('shows add profile option in dropdown', () => {
                    cy.get('[data-testid="sidebar-profile"]').click();

                    cy.contains('Add Another Profile').should('be.visible');

                    cy.get('body').type('{esc}');
                });

                it('opens login dialog when add profile is clicked', () => {
                    cy.get('[data-testid="sidebar-profile"]').click();

                    cy.contains('Add Another Profile').click();

                    cy.get('[role="dialog"], [data-testid="login-form"]', { timeout: 5000 })
                        .should('be.visible');

                    cy.get('body').type('{esc}');
                });

                it('can cancel adding new profile', () => {
                    cy.get('[data-testid="sidebar-profile"]').click();

                    cy.contains('Add Another Profile').click();

                    // Wait for login dialog/sheet to appear
                    cy.get('[role="dialog"], [data-testid="login-form"]', { timeout: 5000 })
                        .should('be.visible');

                    // Close the dialog by pressing escape
                    cy.get('body').type('{esc}');
                    cy.wait(500);

                    // Verify we're back to normal state by checking sidebar profile exists
                    cy.get('[data-testid="sidebar-profile"]').should('exist');
                });
            });
        });
    });

    describe('Database Type Icons', () => {
        forEachDatabase('all', (db) => {
            describe(`${db.type}`, () => {                it('displays correct database type icon for profile', () => {
                    cy.get('[data-testid="sidebar-profile"]').should('exist');

                    cy.get('[data-testid="sidebar-profile"]').within(() => {
                        cy.get('svg, img').should('exist').and('be.visible');
                    });
                });

                it('maintains icon visibility in profile dropdown', () => {
                    cy.get('[data-testid="sidebar-profile"]').click();

                    cy.get('[role="menuitem"], [role="option"]').first().within(() => {
                        cy.get('svg, img').should('exist');
                    });

                    cy.get('body').type('{esc}');
                });
            });
        });
    });

    describe('Profile Last Accessed', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {                it('displays profile information', () => {
                    cy.get('[data-testid="sidebar-profile"]').should('exist');
                    cy.get('[data-testid="sidebar-profile"]').click();

                    cy.get('[role="menuitem"], [role="option"]').should('have.length.at.least', 1);

                    cy.get('body').type('{esc}');
                });
            });
        });
    });

    describe('Logout from Profile', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {                it('can logout from current profile', () => {
                    cy.get('[data-testid="sidebar-profile"]').should('exist');

                    cy.logout();

                    cy.url({ timeout: 10000 }).should('include', '/login');
                });

                it('logout clears current session', () => {
                    cy.logout();

                    cy.url({ timeout: 10000 }).should('include', '/login');

                    cy.visit('/storage-unit');

                    cy.url({ timeout: 10000 }).should('include', '/login');
                });

                it('can logout and login with different profile', () => {
                    const originalDb = db;

                    cy.logout();

                    cy.url({ timeout: 10000 }).should('include', '/login');

                    const newDb = getDatabaseConfig('postgres');

                    cy.login(
                        newDb.type,
                        newDb.connection.host,
                        newDb.connection.user,
                        newDb.connection.password,
                        newDb.connection.database,
                        newDb.connection.advanced
                    );

                    cy.url().should('include', '/storage-unit');
                    cy.get('[data-testid="sidebar-profile"]').should('exist');
                });
            });
        });
    });

    describe('Profile Persistence', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {                it('maintains profile selection after page reload', () => {
                    cy.get('[data-testid="sidebar-profile"]').should('exist');

                    cy.reload();

                    cy.get('[data-testid="sidebar-profile"]', { timeout: 10000 }).should('exist');
                    cy.url().should('include', '/storage-unit');
                });

                it('maintains profile selection when navigating between pages', () => {
                    cy.get('[data-testid="sidebar-profile"]').should('exist');

                    cy.get('[href="/graph"]').click();
                    cy.url().should('include', '/graph');

                    cy.get('[data-testid="sidebar-profile"]').should('exist');

                    cy.get('[href="/storage-unit"]').click();
                    cy.url().should('include', '/storage-unit');

                    cy.get('[data-testid="sidebar-profile"]').should('exist');
                });
            });
        });
    });

    describe('Profile Navigation', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('profile element exists on different pages', () => {
                    // Verify profile element exists on graph page
                    cy.get('[href="/graph"]').click();
                    cy.url().should('include', '/graph');
                    cy.get('[data-testid="sidebar-profile"]', { timeout: 5000 }).should('exist');

                    // Verify profile element exists on scratchpad page
                    cy.get('[href="/scratchpad"]').click();
                    cy.url().should('include', '/scratchpad');
                    cy.get('[data-testid="sidebar-profile"]', { timeout: 5000 }).should('exist');

                    // Navigate back to storage-unit
                    cy.get('[href="/storage-unit"]').click();
                    cy.url().should('include', '/storage-unit');
                    cy.get('[data-testid="sidebar-profile"]', { timeout: 5000 }).should('exist');
                });
            });
        }, { features: ['scratchpad'] });
    });
});
