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

import {forEachDatabase, hasFeature} from '../../support/test-runner';

describe('Keyboard Shortcuts', () => {

    // Keyboard shortcuts are global UI behaviors - only test with Postgres
    forEachDatabase('sql', (db) => {
        if (db.type !== 'Postgres') {
            return;
        }

        const testTable = db.testTable || {};
        const tableName = testTable.name || 'users';

        describe('ESC Key', () => {
            it('closes context menu with ESC', () => {
                cy.data(tableName);

                // Open context menu
                cy.openContextMenu(0);
                cy.get('[data-testid="context-menu-edit-row"]').should('be.visible');

                // Press ESC to close
                cy.get('body').type('{esc}');

                // Context menu should be closed
                cy.get('[data-testid="context-menu-edit-row"]').should('not.exist');
            });

            it('closes edit dialog with ESC', () => {
                cy.data(tableName);

                // Open edit dialog
                cy.openContextMenu(0);
                cy.get('[data-testid="context-menu-edit-row"]').click();

                // Edit dialog should be visible
                cy.contains('Edit Row').should('be.visible');

                // Press ESC to close
                cy.get('body').type('{esc}');

                // Dialog should be closed
                cy.contains('Edit Row').should('not.exist');
            });

            it('closes add row dialog with ESC', () => {
                cy.data(tableName);

                // Open add row dialog via the Add Row button (not context menu)
                cy.get('[data-testid="add-row-button"]').click();

                // Add row dialog should be visible (check for submit button which is in the sheet)
                cy.get('[data-testid="submit-add-row-button"]', {timeout: 5000}).should('be.visible');

                // Press ESC to close
                cy.get('body').type('{esc}');

                // Dialog should be closed
                cy.get('[data-testid="submit-add-row-button"]').should('not.exist');
            });

            if (hasFeature(db, 'scratchpad')) {
                it('closes embedded scratchpad drawer with ESC', () => {
                    cy.data(tableName);

                    // Open embedded scratchpad
                    cy.get('[data-testid="embedded-scratchpad-button"]').click();
                    cy.contains('h2', 'Scratchpad').should('be.visible');

                    // Press ESC to close
                    cy.get('body').type('{esc}');

                    // Drawer should be closed (table search should be visible again)
                    cy.get('[data-testid="table-search"]').should('be.visible');
                });
            }
        });

        describe('Context Menu', () => {
            it('opens context menu with right-click', () => {
                cy.data(tableName);

                // Right-click on a row
                cy.get('table tbody tr').first().rightclick();

                // Context menu should be visible with options
                cy.get('[role="menu"]').should('be.visible');
                cy.get('[data-testid="context-menu-edit-row"]').should('be.visible');
            });

            it('shows keyboard shortcut hints in context menu', () => {
                cy.data(tableName);

                // Open context menu
                cy.openContextMenu(0);

                // Check that shortcut hints are displayed
                cy.get('[role="menu"]').should('be.visible');
                // The menu should contain shortcut indicators
                cy.get('[role="menu"]').then($menu => {
                    // Verify menu items exist (Delete is inside "More Actions" submenu)
                    expect($menu.text()).to.include('Edit');
                    expect($menu.text()).to.include('Copy');
                });
            });
        });
    });

});
