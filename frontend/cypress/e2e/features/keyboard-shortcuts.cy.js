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

        const tableName = 'products';

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

            it('clears row focus with ESC', () => {
                cy.data(tableName);

                // Focus a row using arrow key
                cy.get('body').type('{downarrow}');

                // Verify row is focused
                cy.get('table tbody tr[data-focused="true"]').should('exist');

                // Press ESC to clear focus
                cy.get('body').type('{esc}');

                // Focus should be cleared
                cy.get('table tbody tr[data-focused="true"]').should('not.exist');
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

        describe('Arrow Key Navigation', () => {
            it('ArrowDown focuses first row when no row is focused', () => {
                cy.data(tableName);

                // Ensure no row is focused initially
                cy.get('table tbody tr[data-focused="true"]').should('not.exist');

                // Press ArrowDown
                cy.get('body').type('{downarrow}');

                // First row should be focused
                cy.get('table tbody tr').first().should('have.attr', 'data-focused', 'true');
            });

            it('ArrowDown moves focus to next row', () => {
                cy.data(tableName);

                // Focus first row
                cy.get('body').type('{downarrow}');
                cy.get('table tbody tr').first().should('have.attr', 'data-focused', 'true');

                // Press ArrowDown again
                cy.get('body').type('{downarrow}');

                // Second row should be focused
                cy.get('table tbody tr').eq(1).should('have.attr', 'data-focused', 'true');
                cy.get('table tbody tr').first().should('not.have.attr', 'data-focused', 'true');
            });

            it('ArrowUp focuses last row when no row is focused', () => {
                cy.data(tableName);

                // Ensure no row is focused initially
                cy.get('table tbody tr[data-focused="true"]').should('not.exist');

                // Press ArrowUp
                cy.get('body').type('{uparrow}');

                // Last row should be focused
                cy.get('table tbody tr').last().should('have.attr', 'data-focused', 'true');
            });

            it('ArrowUp moves focus to previous row', () => {
                cy.data(tableName);

                // Focus second row
                cy.get('body').type('{downarrow}{downarrow}');
                cy.get('table tbody tr').eq(1).should('have.attr', 'data-focused', 'true');

                // Press ArrowUp
                cy.get('body').type('{uparrow}');

                // First row should be focused
                cy.get('table tbody tr').first().should('have.attr', 'data-focused', 'true');
            });

            it('Home key focuses first row', () => {
                cy.data(tableName);

                // Focus a middle row first
                cy.get('body').type('{downarrow}{downarrow}{downarrow}');

                // Press Home
                cy.get('body').type('{home}');

                // First row should be focused
                cy.get('table tbody tr').first().should('have.attr', 'data-focused', 'true');
            });

            it('End key focuses last row', () => {
                cy.data(tableName);

                // Focus first row
                cy.get('body').type('{downarrow}');

                // Press End
                cy.get('body').type('{end}');

                // Last row should be focused
                cy.get('table tbody tr').last().should('have.attr', 'data-focused', 'true');
            });

            it('clicking a row sets focus to that row', () => {
                cy.data(tableName);

                // Click on a cell in the second row (not the checkbox area)
                cy.get('table tbody tr').eq(1).find('td').eq(1).click();

                // Wait for React state update
                cy.wait(100);

                // Second row should be focused
                cy.get('table tbody tr').eq(1).should('have.attr', 'data-focused', 'true');
            });
        });

        describe('Row Selection with Space', () => {
            it('Space toggles selection of focused row', () => {
                cy.data(tableName);

                // Focus first row
                cy.get('body').type('{downarrow}');

                // Verify row is not selected (Radix checkbox uses data-state)
                cy.get('table tbody tr').first().find('[data-slot="checkbox"]').should('have.attr', 'data-state', 'unchecked');

                // Press Space to select
                cy.get('body').type(' ');

                // Row should be selected
                cy.get('table tbody tr').first().find('[data-slot="checkbox"]').should('have.attr', 'data-state', 'checked');

                // Press Space again to deselect
                cy.get('body').type(' ');

                // Row should be deselected
                cy.get('table tbody tr').first().find('[data-slot="checkbox"]').should('have.attr', 'data-state', 'unchecked');
            });
        });

        describe('Shift+Arrow Multi-Select', () => {
            it('Shift+ArrowDown extends selection', () => {
                cy.data(tableName);

                // Focus first row
                cy.get('body').type('{downarrow}');

                // Shift+ArrowDown to extend selection
                cy.get('body').type('{shift}{downarrow}');

                // Both first and second rows should be selected (Radix checkbox uses data-state)
                cy.get('table tbody tr').eq(0).find('[data-slot="checkbox"]').should('have.attr', 'data-state', 'checked');
                cy.get('table tbody tr').eq(1).find('[data-slot="checkbox"]').should('have.attr', 'data-state', 'checked');
            });

            it('Shift+ArrowUp extends selection upward', () => {
                cy.data(tableName);

                // Focus third row
                cy.get('body').type('{downarrow}{downarrow}{downarrow}');

                // Shift+ArrowUp twice to extend selection
                cy.get('body').type('{shift}{uparrow}{shift}{uparrow}');

                // Rows 1, 2, and 3 should be selected (Radix checkbox uses data-state)
                cy.get('table tbody tr').eq(0).find('[data-slot="checkbox"]').should('have.attr', 'data-state', 'checked');
                cy.get('table tbody tr').eq(1).find('[data-slot="checkbox"]').should('have.attr', 'data-state', 'checked');
                cy.get('table tbody tr').eq(2).find('[data-slot="checkbox"]').should('have.attr', 'data-state', 'checked');
            });
        });

        describe('Enter Key - Edit Row', () => {
            it('Enter opens edit dialog for focused row', () => {
                cy.data(tableName);

                // Focus first row
                cy.get('body').type('{downarrow}');

                // Press Enter to edit
                cy.get('body').type('{enter}');

                // Edit dialog should open
                cy.contains('Edit Row').should('be.visible');

                // Close with ESC
                cy.get('body').type('{esc}');
            });
        });

        describe('Global Table Shortcuts (Ctrl/Cmd)', () => {
            it('Ctrl+M opens Mock Data sheet', () => {
                cy.data(tableName);

                // Press Ctrl+M - events are on window, so use body
                cy.get('body').type('{ctrl}m');

                // Mock Data sheet should open
                cy.contains('Mock Data').should('be.visible');

                // Close it
                cy.get('body').type('{esc}');
            });

            it('Ctrl+A selects all visible rows', () => {
                cy.data(tableName);

                // Ensure no rows are selected initially (Radix checkbox uses data-state)
                cy.get('table tbody tr [data-slot="checkbox"][data-state="checked"]').should('not.exist');

                // Press Ctrl+A
                cy.get('body').type('{ctrl}a');

                // All row checkboxes should be checked (Radix checkbox uses data-state)
                cy.get('table tbody tr [data-slot="checkbox"]').each(($checkbox) => {
                    cy.wrap($checkbox).should('have.attr', 'data-state', 'checked');
                });

                // Press Ctrl+A again to deselect
                cy.get('body').type('{ctrl}a');

                // All row checkboxes should be unchecked (Radix checkbox uses data-state)
                cy.get('table tbody tr [data-slot="checkbox"]').each(($checkbox) => {
                    cy.wrap($checkbox).should('have.attr', 'data-state', 'unchecked');
                });
            });

            it('Ctrl+Shift+E opens Export dialog', () => {
                cy.data(tableName);

                // Press Ctrl+Shift+E
                cy.get('body').type('{ctrl}{shift}e');

                // Export dialog should open
                cy.contains('Export').should('be.visible');

                // Close it
                cy.get('body').type('{esc}');
            });

            it('Ctrl+E edits focused row', () => {
                cy.data(tableName);

                // Focus first row
                cy.get('body').type('{downarrow}');

                // Press Ctrl+E
                cy.get('body').type('{ctrl}e');

                // Edit dialog should open
                cy.contains('Edit Row').should('be.visible');

                // Close it
                cy.get('body').type('{esc}');
            });

            it('Ctrl+R refreshes the table', () => {
                cy.data(tableName);

                // Press Ctrl+R to refresh
                cy.get('body').type('{ctrl}r');

                // Table should still have rows after refresh
                cy.get('table tbody tr').should('have.length.at.least', 1);
            });
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
                // The menu should contain shortcut indicators for shortcuts
                cy.get('[role="menu"]').then($menu => {
                    // Verify menu items exist with shortcut labels
                    expect($menu.text()).to.include('Edit');
                    expect($menu.text()).to.include('Enter'); // Shortcut for Edit
                    expect($menu.text()).to.include('Space'); // Shortcut for Select
                });
            });
        });

        describe('ARIA Accessibility', () => {
            it('table has correct ARIA attributes', () => {
                cy.data(tableName);

                // Table should have grid role
                cy.get('table').should('have.attr', 'role', 'grid');

                // Table should have aria-multiselectable
                cy.get('table').should('have.attr', 'aria-multiselectable', 'true');
            });

            it('rows have correct ARIA attributes', () => {
                cy.data(tableName);

                // Focus a row
                cy.get('body').type('{downarrow}');

                // Focused row should have correct attributes
                cy.get('table tbody tr[data-focused="true"]')
                    .should('have.attr', 'role', 'row')
                    .and('have.attr', 'tabindex', '0');

                // Non-focused rows should have tabindex -1 (only check if multiple rows exist)
                cy.get('table tbody tr').then($rows => {
                    if ($rows.length > 1) {
                        cy.get('table tbody tr:not([data-focused="true"])')
                            .first()
                            .should('have.attr', 'tabindex', '-1');
                    }
                });
            });

            it('selected rows have aria-selected attribute', () => {
                cy.data(tableName);

                // Focus and select first row
                cy.get('body').type('{downarrow}');
                cy.get('body').type(' ');

                // Row should have aria-selected
                cy.get('table tbody tr').first().should('have.attr', 'aria-selected', 'true');
            });
        });

        describe('Focus Reset Behavior', () => {
            it('focus resets when changing pages', () => {
                cy.data(tableName);

                // Set page size to a small value to see pagination (2 is available in E2E mode)
                cy.setTablePageSize(2);
                cy.get('[data-testid="submit-button"]').click();
                cy.get('[data-testid="table-page-number"]', { timeout: 10000 }).should('exist');

                // Focus a row
                cy.get('body').type('{downarrow}');
                cy.get('table tbody tr[data-focused="true"]').should('exist');

                // Go to next page (if available)
                cy.get('body').then($body => {
                    if ($body.find('[data-testid="table-page-number"]').length > 1) {
                        cy.get('[data-testid="table-page-number"]').eq(1).click();

                        // Focus should be reset
                        cy.get('table tbody tr[data-focused="true"]').should('not.exist');
                    }
                });
            });
        });

        describe('Pagination Shortcuts', () => {
            it('Ctrl+ArrowRight goes to next page', () => {
                cy.data(tableName);

                // Set page size to a small value to see pagination
                cy.setTablePageSize(2);
                cy.get('[data-testid="submit-button"]').click();
                cy.get('[data-testid="table-page-number"]', { timeout: 10000 }).should('exist');

                // Check we're on page 1
                cy.get('[data-testid="table-page-number"]').first().should('have.attr', 'data-active', 'true');

                // Press Ctrl+ArrowRight to go to next page
                cy.get('body').type('{ctrl}{rightarrow}');

                // Should now be on page 2
                cy.get('[data-testid="table-page-number"]').eq(1).should('have.attr', 'data-active', 'true');
            });

            it('Ctrl+ArrowLeft goes to previous page', () => {
                cy.data(tableName);

                // Set page size to a small value to see pagination
                cy.setTablePageSize(2);
                cy.get('[data-testid="submit-button"]').click();
                cy.get('[data-testid="table-page-number"]', { timeout: 10000 }).should('exist');

                // Go to page 2 first
                cy.get('[data-testid="table-page-number"]').eq(1).click();
                cy.get('[data-testid="table-page-number"]').eq(1).should('have.attr', 'data-active', 'true');

                // Press Ctrl+ArrowLeft to go to previous page
                cy.get('body').type('{ctrl}{leftarrow}');

                // Should now be on page 1
                cy.get('[data-testid="table-page-number"]').first().should('have.attr', 'data-active', 'true');
            });

            it('Ctrl+ArrowLeft does nothing on first page', () => {
                cy.data(tableName);

                // Set page size to a small value to see pagination
                cy.setTablePageSize(2);
                cy.get('[data-testid="submit-button"]').click();
                cy.get('[data-testid="table-page-number"]', { timeout: 10000 }).should('exist');

                // Ensure we're on page 1
                cy.get('[data-testid="table-page-number"]').first().should('have.attr', 'data-active', 'true');

                // Press Ctrl+ArrowLeft - should stay on page 1
                cy.get('body').type('{ctrl}{leftarrow}');

                // Should still be on page 1
                cy.get('[data-testid="table-page-number"]').first().should('have.attr', 'data-active', 'true');
            });
        });

        describe('Keyboard Shortcuts Help Modal', () => {
            it('shortcuts button exists in page header', () => {
                cy.data(tableName);

                // Shortcuts button should be visible in the header
                cy.contains('button', 'Shortcuts').should('be.visible');
            });

            it('clicking shortcuts button opens the modal', () => {
                cy.data(tableName);

                // Click the shortcuts button
                cy.contains('button', 'Shortcuts').click();

                // Modal should open with title
                cy.contains('Keyboard Shortcuts').should('be.visible');

                // Close with ESC
                cy.get('body').type('{esc}');
                cy.contains('Keyboard Shortcuts').should('not.exist');
            });

            it('pressing ? key opens the shortcuts modal', () => {
                cy.data(tableName);

                // Press ? key (Shift+/)
                cy.get('body').type('?');

                // Modal should open
                cy.contains('Keyboard Shortcuts').should('be.visible');

                // Verify the modal contains shortcut categories (use exist instead of visible for scrollable content)
                cy.contains('Global').should('exist');
                cy.contains('Table Navigation').should('exist');

                // Close with ESC
                cy.get('body').type('{esc}');
            });
        });

        describe('Sidebar Navigation Shortcuts', () => {
            it('Ctrl+B toggles sidebar', () => {
                cy.data(tableName);

                // Get initial sidebar state
                cy.get('[data-sidebar="sidebar"]').should('exist');

                // Press Ctrl+B to toggle sidebar
                cy.get('body').type('{ctrl}b');

                // Wait for animation
                cy.wait(300);

                // Press Ctrl+B again to toggle back
                cy.get('body').type('{ctrl}b');

                // Sidebar should be visible again
                cy.wait(300);
                cy.get('[data-sidebar="sidebar"]').should('exist');
            });

            it('Alt+1 navigates to first view', () => {
                cy.data(tableName);

                // Press Alt+1 to go to first view (Chat for SQL databases)
                cy.get('body').type('{alt}1');

                // Should navigate to chat page
                cy.url().should('include', '/chat');
            });

            it('Alt+2 navigates to second view', () => {
                cy.data(tableName);

                // Press Alt+2 to go to second view (Storage Units/Tables)
                cy.get('body').type('{alt}2');

                // Should navigate to storage-unit page
                cy.url().should('include', '/storage-unit');
            });

            it('Alt+3 navigates to third view', () => {
                cy.data(tableName);

                // Press Alt+3 to go to third view (Graph)
                cy.get('body').type('{alt}3');

                // Should navigate to graph page
                cy.url().should('include', '/graph');
            });

            it('Alt+4 navigates to fourth view (Scratchpad)', () => {
                cy.data(tableName);

                // Press Alt+4 to go to fourth view (Scratchpad)
                cy.get('body').type('{alt}4');

                // Should navigate to scratchpad page
                cy.url().should('include', '/scratchpad');
            });
        });

        // Deletion tests are placed at the end because they modify data and could affect other tests
        describe('Ctrl+Delete/Backspace - Delete Row', () => {
            it('Delete key without Ctrl does not delete row', () => {
                cy.data(tableName);

                // Get initial row count
                cy.get('table tbody tr').its('length').then(initialCount => {
                    // Focus first row
                    cy.get('body').type('{downarrow}');

                    // Press Delete without Ctrl
                    cy.get('body').type('{del}');

                    // Row count should remain the same
                    cy.get('table tbody tr').should('have.length', initialCount);
                });
            });

            it('Ctrl+Delete triggers delete for focused row', () => {
                cy.data(tableName);

                // Get initial row count
                cy.get('table tbody tr').its('length').then(initialCount => {
                    // Focus first row
                    cy.get('body').type('{downarrow}');

                    // Press Ctrl+Delete
                    cy.get('body').type('{ctrl}{del}');

                    // Row should be deleted (count decreased)
                    cy.get('table tbody tr', {timeout: 10000}).should('have.length', initialCount - 1);
                });
            });

            it('Ctrl+Backspace triggers delete for focused row', () => {
                cy.data(tableName);

                // Get initial row count
                cy.get('table tbody tr').its('length').then(initialCount => {
                    // Focus first row
                    cy.get('body').type('{downarrow}');

                    // Press Ctrl+Backspace
                    cy.get('body').type('{ctrl}{backspace}');

                    // Row should be deleted (count decreased)
                    cy.get('table tbody tr', {timeout: 10000}).should('have.length', initialCount - 1);
                });
            });
        });
    });

});
