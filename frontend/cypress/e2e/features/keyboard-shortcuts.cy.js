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
            it('Cmd/Ctrl+M opens Mock Data sheet', () => {
                cy.data(tableName);

                // Press Cmd+M (Mac) or Ctrl+M (Win/Linux)
                cy.typeCmdShortcut('m');

                // Mock Data sheet should open
                cy.contains('Mock Data').should('be.visible');

                // Close it
                cy.get('body').type('{esc}');
            });

            it('Cmd/Ctrl+A selects all visible rows', () => {
                cy.data(tableName);

                // Ensure no rows are selected initially (Radix checkbox uses data-state)
                cy.get('table tbody tr [data-slot="checkbox"][data-state="checked"]').should('not.exist');

                // Press Cmd+A (Mac) or Ctrl+A (Win/Linux)
                cy.typeCmdShortcut('a');

                // All row checkboxes should be checked (Radix checkbox uses data-state)
                cy.get('table tbody tr [data-slot="checkbox"]').each(($checkbox) => {
                    cy.wrap($checkbox).should('have.attr', 'data-state', 'checked');
                });

                // Press Cmd/Ctrl+A again to deselect
                cy.typeCmdShortcut('a');

                // All row checkboxes should be unchecked (Radix checkbox uses data-state)
                cy.get('table tbody tr [data-slot="checkbox"]').each(($checkbox) => {
                    cy.wrap($checkbox).should('have.attr', 'data-state', 'unchecked');
                });
            });

            it('Cmd/Ctrl+Shift+E opens Export dialog', () => {
                cy.data(tableName);

                // Press Cmd+Shift+E (Mac) or Ctrl+Shift+E (Win/Linux)
                cy.typeCmdShortcut('e', { shift: true });

                // Export dialog should open
                cy.contains('Export').should('be.visible');

                // Close it
                cy.get('body').type('{esc}');
            });

            it('Cmd/Ctrl+E edits focused row', () => {
                cy.data(tableName);

                // Focus first row
                cy.get('body').type('{downarrow}');

                // Press Cmd+E (Mac) or Ctrl+E (Win/Linux)
                cy.typeCmdShortcut('e');

                // Edit dialog should open
                cy.contains('Edit Row').should('be.visible');

                // Close it
                cy.get('body').type('{esc}');
            });

            it('Cmd/Ctrl+R refreshes the table', () => {
                cy.data(tableName);

                // Press Cmd+R (Mac) or Ctrl+R (Win/Linux)
                cy.typeCmdShortcut('r');

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
            it('Cmd/Ctrl+ArrowRight goes to next page', () => {
                cy.data(tableName);

                // Set page size to a small value to see pagination
                cy.setTablePageSize(2);
                cy.get('[data-testid="submit-button"]').click();
                cy.get('[data-testid="table-page-number"]', { timeout: 10000 }).should('exist');

                // Check we're on page 1
                cy.get('[data-testid="table-page-number"]').first().should('have.attr', 'data-active', 'true');

                // Press Cmd+ArrowRight (Mac) or Ctrl+ArrowRight (Win/Linux) to go to next page
                cy.typeCmdShortcut('{rightarrow}');

                // Should now be on page 2
                cy.get('[data-testid="table-page-number"]').eq(1).should('have.attr', 'data-active', 'true');
            });

            it('Cmd/Ctrl+ArrowLeft goes to previous page', () => {
                cy.data(tableName);

                // Set page size to a small value to see pagination
                cy.setTablePageSize(2);
                cy.get('[data-testid="submit-button"]').click();
                cy.get('[data-testid="table-page-number"]', { timeout: 10000 }).should('exist');

                // Go to page 2 first
                cy.get('[data-testid="table-page-number"]').eq(1).click();
                cy.get('[data-testid="table-page-number"]').eq(1).should('have.attr', 'data-active', 'true');

                // Press Cmd+ArrowLeft (Mac) or Ctrl+ArrowLeft (Win/Linux) to go to previous page
                cy.typeCmdShortcut('{leftarrow}');

                // Should now be on page 1
                cy.get('[data-testid="table-page-number"]').first().should('have.attr', 'data-active', 'true');
            });

            it('Cmd/Ctrl+ArrowLeft does nothing on first page', () => {
                cy.data(tableName);

                // Set page size to a small value to see pagination
                cy.setTablePageSize(2);
                cy.get('[data-testid="submit-button"]').click();
                cy.get('[data-testid="table-page-number"]', { timeout: 10000 }).should('exist');

                // Ensure we're on page 1
                cy.get('[data-testid="table-page-number"]').first().should('have.attr', 'data-active', 'true');

                // Press Cmd/Ctrl+ArrowLeft - should stay on page 1
                cy.typeCmdShortcut('{leftarrow}');

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

                // Press Cmd/Ctrl+B to toggle sidebar
                cy.typeCmdShortcut('b');

                // Wait for animation
                cy.wait(300);

                // Press Cmd/Ctrl+B again to toggle back
                cy.typeCmdShortcut('b');

                // Sidebar should be visible again
                cy.wait(300);
                cy.get('[data-sidebar="sidebar"]').should('exist');
            });

            it('navigates to first view with number shortcut', () => {
                cy.data(tableName);

                // Press Ctrl+1 (Mac) or Alt+1 (Win/Linux) to go to first view (Chat for SQL databases)
                cy.typeNavShortcut(1);

                // Should navigate to chat page
                cy.url().should('include', '/chat');
            });

            it('navigates to second view with number shortcut', () => {
                cy.data(tableName);

                // Press Ctrl+2 (Mac) or Alt+2 (Win/Linux) to go to second view (Storage Units/Tables)
                cy.typeNavShortcut(2);

                // Should navigate to storage-unit page
                cy.url().should('include', '/storage-unit');
            });

            it('navigates to third view with number shortcut', () => {
                cy.data(tableName);

                // Press Ctrl+3 (Mac) or Alt+3 (Win/Linux) to go to third view (Graph)
                cy.typeNavShortcut(3);

                // Should navigate to graph page
                cy.url().should('include', '/graph');
            });

            it('navigates to fourth view (Scratchpad) with number shortcut', () => {
                cy.data(tableName);

                // Press Ctrl+4 (Mac) or Alt+4 (Win/Linux) to go to fourth view (Scratchpad)
                cy.typeNavShortcut(4);

                // Should navigate to scratchpad page
                cy.url().should('include', '/scratchpad');
            });
        });

        describe('Command Palette (Cmd/Ctrl+K)', () => {
            it('Cmd/Ctrl+K opens command palette', () => {
                cy.data(tableName);

                // Press Cmd+K (Mac) or Ctrl+K (Win/Linux)
                cy.typeCmdShortcut('k');

                // Command palette should open
                cy.get('[data-testid="command-palette"]').should('be.visible');

                // Should have search input
                cy.get('[data-testid="command-palette-input"]').should('be.visible');

                // Close with ESC
                cy.get('body').type('{esc}');
                cy.get('[data-testid="command-palette"]').should('not.exist');
            });

            it('Command palette trigger button opens palette', () => {
                cy.data(tableName);

                // Click the trigger button
                cy.get('[data-testid="command-palette-trigger"]').click();

                // Command palette should open
                cy.get('[data-testid="command-palette"]').should('be.visible');

                // Close with ESC
                cy.get('body').type('{esc}');
            });

            it('Command palette shows navigation options', () => {
                cy.data(tableName);

                // Open command palette
                cy.typeCmdShortcut('k');

                // Should show navigation commands
                cy.get('[data-testid="command-nav-chat"]').should('exist');
                cy.get('[data-testid="command-nav-storage-units"]').should('exist');
                cy.get('[data-testid="command-nav-graph"]').should('exist');
                cy.get('[data-testid="command-nav-scratchpad"]').should('exist');

                // Close
                cy.get('body').type('{esc}');
            });

            it('Command palette shows action options', () => {
                cy.data(tableName);

                // Open command palette
                cy.typeCmdShortcut('k');

                // Should show action commands
                cy.get('[data-testid="command-action-refresh"]').should('exist');
                cy.get('[data-testid="command-action-export"]').should('exist');
                cy.get('[data-testid="command-action-toggle-sidebar"]').should('exist');
                cy.get('[data-testid="command-action-disconnect"]').should('exist');

                // Close
                cy.get('body').type('{esc}');
            });

            it('Command palette navigates to Chat when selected', () => {
                cy.data(tableName);

                // Open command palette
                cy.typeCmdShortcut('k');

                // Click on Chat navigation
                cy.get('[data-testid="command-nav-chat"]').click();

                // Should navigate to chat
                cy.url().should('include', '/chat');
            });

            it('Command palette search filters results', () => {
                cy.data(tableName);

                // Open command palette
                cy.typeCmdShortcut('k');

                // Type to search
                cy.get('[data-testid="command-palette-input"]').type('graph');

                // Should filter to show only graph
                cy.get('[data-testid="command-nav-graph"]').should('exist');
                cy.get('[data-testid="command-nav-chat"]').should('not.exist');

                // Close
                cy.get('body').type('{esc}');
            });
        });

        describe('PageUp/PageDown Navigation', () => {
            it('PageDown jumps down by visible rows', () => {
                cy.data(tableName);

                // Focus first row
                cy.get('body').type('{downarrow}');

                // Get the initial focused row
                cy.get('table tbody tr').first().should('have.attr', 'data-focused', 'true');

                // Press PageDown
                cy.get('body').type('{pagedown}');

                // First row should no longer be focused (focus moved down)
                cy.get('table tbody tr').first().should('not.have.attr', 'data-focused', 'true');
            });

            it('PageUp jumps up by visible rows', () => {
                cy.data(tableName);

                // Focus last row using End key
                cy.get('body').type('{downarrow}');
                cy.get('body').type('{end}');

                // Press PageUp
                cy.get('body').type('{pageup}');

                // Last row should no longer be focused (focus moved up)
                cy.get('table tbody tr').last().should('not.have.attr', 'data-focused', 'true');
            });
        });

        describe('Query Editor Shortcuts', () => {
            it('Ctrl+Enter executes query in scratchpad', () => {
                cy.goto('scratchpad');

                // Write a simple query
                cy.writeCode(0, 'SELECT 1 as test');

                // Focus the editor and press Ctrl+Enter
                const editorSelector = '[role="tabpanel"][data-state="active"] [data-testid="cell-0"] .cm-content';
                cy.get(editorSelector).click().type('{ctrl}{enter}');

                // Query should execute and show results
                cy.get('[role="tabpanel"][data-state="active"] [data-testid="cell-0"]').within(() => {
                    cy.get('[data-testid="cell-query-output"], [data-testid="cell-action-output"], [data-testid="cell-error"]', { timeout: 5000 })
                        .should('exist');
                });
            });
        });

        describe('Column Header Keyboard Sorting', () => {
            it('column headers are focusable', () => {
                cy.data(tableName);

                // Focus the column header
                cy.get('[data-testid="column-header-id"]').focus();

                // Should have focus
                cy.get('[data-testid="column-header-id"]').should('have.focus');
            });

            it('Enter key on focused column header triggers sort', () => {
                cy.data(tableName);

                // Focus and press Enter using trigger (more reliable than type)
                cy.get('[data-testid="column-header-id"]').focus();
                cy.get('[data-testid="column-header-id"]').trigger('keydown', { key: 'Enter' });

                // Wait for sort to be applied and data to refresh
                cy.wait(500);

                // Column should now show sort indicator
                cy.get('[data-testid="column-header-id"]').find('[data-testid="sort-indicator"]').should('exist');
            });

            it('Space key on focused column header triggers sort', () => {
                cy.data(tableName);

                // Focus and press Space using trigger
                cy.get('[data-testid="column-header-name"]').focus();
                cy.get('[data-testid="column-header-name"]').trigger('keydown', { key: ' ' });

                // Wait for sort to be applied and data to refresh
                cy.wait(500);

                // Column should now show sort indicator
                cy.get('[data-testid="column-header-name"]').find('[data-testid="sort-indicator"]').should('exist');
            });

            it('clicking column header sorts data ascending', () => {
                cy.data(tableName);

                // Click to sort by id ascending
                cy.get('[data-testid="column-header-id"]').click();
                cy.wait(500);

                // Column should show sort indicator
                cy.get('[data-testid="column-header-id"]').find('[data-testid="sort-indicator"]').should('exist');

                // Verify data is sorted ascending - collect all id values and check order
                cy.get('table tbody tr').then($rows => {
                    const ids = [];
                    $rows.each((_, row) => {
                        // Get the first data cell (after checkbox column)
                        const idCell = Cypress.$(row).find('td').eq(1).text().trim();
                        if (idCell) ids.push(parseInt(idCell, 10));
                    });

                    // Check ascending order
                    for (let i = 1; i < ids.length; i++) {
                        expect(ids[i]).to.be.gte(ids[i - 1]);
                    }
                });
            });

            it('clicking column header twice sorts data descending', () => {
                cy.data(tableName);

                // Click twice to sort by id descending
                cy.get('[data-testid="column-header-id"]').click();
                cy.wait(500);
                cy.get('[data-testid="column-header-id"]').click();
                cy.wait(500);

                // Column should show sort indicator
                cy.get('[data-testid="column-header-id"]').find('[data-testid="sort-indicator"]').should('exist');

                // Verify data is sorted descending
                cy.get('table tbody tr').then($rows => {
                    const ids = [];
                    $rows.each((_, row) => {
                        const idCell = Cypress.$(row).find('td').eq(1).text().trim();
                        if (idCell) ids.push(parseInt(idCell, 10));
                    });

                    // Check descending order
                    for (let i = 1; i < ids.length; i++) {
                        expect(ids[i]).to.be.lte(ids[i - 1]);
                    }
                });
            });

            it('multiple sorts can be applied by clicking different columns', () => {
                cy.data(tableName);

                // Sort by id first
                cy.get('[data-testid="column-header-id"]').click();
                cy.wait(500);

                // Sort by name second
                cy.get('[data-testid="column-header-name"]').click();
                cy.wait(500);

                // Both columns should show sort indicators
                cy.get('[data-testid="column-header-id"]').find('[data-testid="sort-indicator"]').should('exist');
                cy.get('[data-testid="column-header-name"]').find('[data-testid="sort-indicator"]').should('exist');
            });

            it('clicking column three times cycles through asc, desc, and removes sort', () => {
                cy.data(tableName);

                // Get original order before any sorting
                let originalIds = [];
                cy.get('table tbody tr').then($rows => {
                    $rows.each((_, row) => {
                        const idCell = Cypress.$(row).find('td').eq(1).text().trim();
                        if (idCell) originalIds.push(parseInt(idCell, 10));
                    });
                });

                // First click: ascending sort
                cy.get('[data-testid="column-header-id"]').click();
                cy.wait(500);
                cy.get('[data-testid="column-header-id"]').find('[data-testid="sort-indicator"]').should('exist');

                // Verify ascending
                cy.get('table tbody tr').then($rows => {
                    const ids = [];
                    $rows.each((_, row) => {
                        const idCell = Cypress.$(row).find('td').eq(1).text().trim();
                        if (idCell) ids.push(parseInt(idCell, 10));
                    });
                    for (let i = 1; i < ids.length; i++) {
                        expect(ids[i]).to.be.gte(ids[i - 1]);
                    }
                });

                // Second click: descending sort
                cy.get('[data-testid="column-header-id"]').click();
                cy.wait(500);
                cy.get('[data-testid="column-header-id"]').find('[data-testid="sort-indicator"]').should('exist');

                // Verify descending
                cy.get('table tbody tr').then($rows => {
                    const ids = [];
                    $rows.each((_, row) => {
                        const idCell = Cypress.$(row).find('td').eq(1).text().trim();
                        if (idCell) ids.push(parseInt(idCell, 10));
                    });
                    for (let i = 1; i < ids.length; i++) {
                        expect(ids[i]).to.be.lte(ids[i - 1]);
                    }
                });

                // Third click: remove sort
                cy.get('[data-testid="column-header-id"]').click();
                cy.wait(500);

                // Sort indicator should be removed
                cy.get('[data-testid="column-header-id"]').find('[data-testid="sort-indicator"]').should('not.exist');
            });
        });

        describe('Command Palette Sort Commands', () => {
            it('command palette shows sort options when on table view', () => {
                cy.data(tableName);

                // Open command palette
                cy.typeCmdShortcut('k');

                // Should show "Sort By" section with column options
                cy.contains('Sort By').should('exist');
                cy.get('[data-testid="command-sort-id"]').should('exist');
                cy.get('[data-testid="command-sort-name"]').should('exist');

                // Close
                cy.get('body').type('{esc}');
            });

            it('selecting sort command from palette sorts the column', () => {
                cy.data(tableName);

                // Open command palette
                cy.typeCmdShortcut('k');

                // Click on sort by id
                cy.get('[data-testid="command-sort-id"]').click();

                // Wait for sort to be applied
                cy.wait(500);

                // Column should now show sort indicator
                cy.get('[data-testid="column-header-id"]').find('[data-testid="sort-indicator"]').should('exist');
            });

            it('sort commands can be searched in command palette', () => {
                cy.data(tableName);

                // Open command palette
                cy.typeCmdShortcut('k');

                // Type to search for sort
                cy.get('[data-testid="command-palette-input"]').type('sort by name');

                // Should filter to show sort by name
                cy.get('[data-testid="command-sort-name"]').should('exist');
                cy.get('[data-testid="command-sort-id"]').should('not.exist');

                // Close
                cy.get('body').type('{esc}');
            });
        });

        // Deletion tests are placed at the end because they modify data and could affect other tests
        describe('Cmd/Ctrl+Delete/Backspace - Delete Row', () => {
            it('Delete key without Cmd/Ctrl does not delete row', () => {
                cy.data(tableName);

                // Get initial row count
                cy.get('table tbody tr').its('length').then(initialCount => {
                    // Focus first row
                    cy.get('body').type('{downarrow}');

                    // Press Delete without Cmd/Ctrl
                    cy.get('body').type('{del}');

                    // Row count should remain the same
                    cy.get('table tbody tr').should('have.length', initialCount);
                });
            });

            it('Cmd/Ctrl+Delete triggers delete for focused row', () => {
                cy.data(tableName);

                // Get initial row count
                cy.get('table tbody tr').its('length').then(initialCount => {
                    // Focus first row
                    cy.get('body').type('{downarrow}');

                    // Press Cmd+Delete (Mac) or Ctrl+Delete (Win/Linux)
                    cy.typeCmdShortcut('{del}');

                    // Row should be deleted (count decreased)
                    cy.get('table tbody tr', {timeout: 10000}).should('have.length', initialCount - 1);
                });
            });

            it('Cmd/Ctrl+Backspace triggers delete for focused row', () => {
                cy.data(tableName);

                // Get initial row count
                cy.get('table tbody tr').its('length').then(initialCount => {
                    // Focus first row
                    cy.get('body').type('{downarrow}');

                    // Press Cmd+Backspace (Mac) or Ctrl+Backspace (Win/Linux)
                    cy.typeCmdShortcut('{backspace}');

                    // Row should be deleted (count decreased)
                    cy.get('table tbody tr', {timeout: 10000}).should('have.length', initialCount - 1);
                });
            });
        });
    });

});
