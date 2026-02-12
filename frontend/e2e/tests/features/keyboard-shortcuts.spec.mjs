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

import { test, expect, forEachDatabase } from '../../support/test-fixture.mjs';
import { hasFeature } from '../../support/database-config.mjs';

test.describe('Keyboard Shortcuts', () => {

    // Keyboard shortcuts are global UI behaviors - only test with Postgres
    forEachDatabase('sql', (db) => {
        if (db.type !== 'Postgres') {
            return;
        }

        const tableName = 'products';

        test.describe('ESC Key', () => {
            test('closes context menu with ESC', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open context menu
                await whodb.openContextMenu(0);
                await expect(page.locator('[data-testid="context-menu-edit-row"]')).toBeVisible();

                // Press ESC to close
                await page.keyboard.press('Escape');

                // Context menu should be closed
                await expect(page.locator('[data-testid="context-menu-edit-row"]')).not.toBeAttached();
            });

            test('closes edit dialog with ESC', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open edit dialog
                await whodb.openContextMenu(0);
                await page.locator('[data-testid="context-menu-edit-row"]').click();

                // Edit dialog should be visible
                await expect(page.locator('[data-testid="edit-row-dialog"]')).toBeVisible();

                // Press ESC to close
                await page.keyboard.press('Escape');

                // Dialog should be closed
                await expect(page.locator('[data-testid="edit-row-dialog"]')).not.toBeAttached();
            });

            test('closes add row dialog with ESC', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open add row dialog via the Add Row button (not context menu)
                await page.locator('[data-testid="add-row-button"]').click();

                // Add row dialog should be visible (check for submit button which is in the sheet)
                await expect(page.locator('[data-testid="submit-add-row-button"]')).toBeVisible({ timeout: 5000 });

                // Press ESC to close
                await page.keyboard.press('Escape');

                // Dialog should be closed
                await expect(page.locator('[data-testid="submit-add-row-button"]')).not.toBeAttached();
            });

            test('clears row focus with ESC', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus a row using arrow key
                await page.keyboard.press('ArrowDown');

                // Verify row is focused
                await expect(page.locator('table tbody tr[data-focused="true"]')).toBeAttached();

                // Press ESC to clear focus
                await page.keyboard.press('Escape');

                // Focus should be cleared
                await expect(page.locator('table tbody tr[data-focused="true"]')).not.toBeAttached();
            });

            if (hasFeature(db, 'scratchpad')) {
                test('closes embedded scratchpad drawer with ESC', async ({ whodb, page }) => {
                    await whodb.data(tableName);

                    // Open embedded scratchpad
                    await page.locator('[data-testid="embedded-scratchpad-button"]').click();
                    await expect(page.locator('[data-testid="scratchpad-drawer"]')).toBeVisible();

                    // Press ESC to close
                    await page.keyboard.press('Escape');

                    // Drawer should be closed (table search should be visible again)
                    await expect(page.locator('[data-testid="table-search"]')).toBeVisible();
                });
            }
        });

        test.describe('Arrow Key Navigation', () => {
            test('ArrowDown focuses first row when no row is focused', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Ensure no row is focused initially
                await expect(page.locator('table tbody tr[data-focused="true"]')).not.toBeAttached();

                // Press ArrowDown
                await page.keyboard.press('ArrowDown');

                // First row should be focused
                await expect(page.locator('table tbody tr').first()).toHaveAttribute('data-focused', 'true');
            });

            test('ArrowDown moves focus to next row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus first row
                await page.keyboard.press('ArrowDown');
                await expect(page.locator('table tbody tr').first()).toHaveAttribute('data-focused', 'true');

                // Press ArrowDown again
                await page.keyboard.press('ArrowDown');

                // Second row should be focused
                await expect(page.locator('table tbody tr').nth(1)).toHaveAttribute('data-focused', 'true');
                await expect(page.locator('table tbody tr').first()).not.toHaveAttribute('data-focused', 'true');
            });

            test('ArrowUp focuses last row when no row is focused', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Ensure no row is focused initially
                await expect(page.locator('table tbody tr[data-focused="true"]')).not.toBeAttached();

                // Press ArrowUp
                await page.keyboard.press('ArrowUp');

                // Last row should be focused
                await expect(page.locator('table tbody tr').last()).toHaveAttribute('data-focused', 'true');
            });

            test('ArrowUp moves focus to previous row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus second row
                await page.keyboard.press('ArrowDown');
                await page.keyboard.press('ArrowDown');
                await expect(page.locator('table tbody tr').nth(1)).toHaveAttribute('data-focused', 'true');

                // Press ArrowUp
                await page.keyboard.press('ArrowUp');

                // First row should be focused
                await expect(page.locator('table tbody tr').first()).toHaveAttribute('data-focused', 'true');
            });

            test('Home key focuses first row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus a middle row first
                await page.keyboard.press('ArrowDown');
                await page.keyboard.press('ArrowDown');
                await page.keyboard.press('ArrowDown');

                // Press Home
                await page.keyboard.press('Home');

                // First row should be focused
                await expect(page.locator('table tbody tr').first()).toHaveAttribute('data-focused', 'true');
            });

            test('End key focuses last row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus first row
                await page.keyboard.press('ArrowDown');

                // Press End
                await page.keyboard.press('End');

                // Last row should be focused
                await expect(page.locator('table tbody tr').last()).toHaveAttribute('data-focused', 'true');
            });

            test('clicking a row sets focus to that row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Click on a cell in the second row (not the checkbox area)
                await page.locator('table tbody tr').nth(1).locator('td').nth(1).click();

                // Wait for React state update
                await page.waitForTimeout(100);

                // Second row should be focused
                await expect(page.locator('table tbody tr').nth(1)).toHaveAttribute('data-focused', 'true');
            });
        });

        test.describe('Row Selection with Space', () => {
            test('Space toggles selection of focused row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus first row
                await page.keyboard.press('ArrowDown');

                // Verify row is not selected (Radix checkbox uses data-state)
                await expect(page.locator('table tbody tr').first().locator('[data-slot="checkbox"]')).toHaveAttribute('data-state', 'unchecked');

                // Press Space to select
                await page.keyboard.press('Space');

                // Row should be selected
                await expect(page.locator('table tbody tr').first().locator('[data-slot="checkbox"]')).toHaveAttribute('data-state', 'checked');

                // Press Space again to deselect
                await page.keyboard.press('Space');

                // Row should be deselected
                await expect(page.locator('table tbody tr').first().locator('[data-slot="checkbox"]')).toHaveAttribute('data-state', 'unchecked');
            });
        });

        test.describe('Shift+Arrow Multi-Select', () => {
            test('Shift+ArrowDown extends selection', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus first row
                await page.keyboard.press('ArrowDown');

                // Shift+ArrowDown to extend selection
                await page.keyboard.press('Shift+ArrowDown');

                // Both first and second rows should be selected (Radix checkbox uses data-state)
                await expect(page.locator('table tbody tr').nth(0).locator('[data-slot="checkbox"]')).toHaveAttribute('data-state', 'checked');
                await expect(page.locator('table tbody tr').nth(1).locator('[data-slot="checkbox"]')).toHaveAttribute('data-state', 'checked');
            });

            test('Shift+ArrowUp extends selection upward', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus third row
                await page.keyboard.press('ArrowDown');
                await page.keyboard.press('ArrowDown');
                await page.keyboard.press('ArrowDown');

                // Shift+ArrowUp twice to extend selection
                await page.keyboard.press('Shift+ArrowUp');
                await page.keyboard.press('Shift+ArrowUp');

                // Rows 1, 2, and 3 should be selected (Radix checkbox uses data-state)
                await expect(page.locator('table tbody tr').nth(0).locator('[data-slot="checkbox"]')).toHaveAttribute('data-state', 'checked');
                await expect(page.locator('table tbody tr').nth(1).locator('[data-slot="checkbox"]')).toHaveAttribute('data-state', 'checked');
                await expect(page.locator('table tbody tr').nth(2).locator('[data-slot="checkbox"]')).toHaveAttribute('data-state', 'checked');
            });
        });

        test.describe('Enter Key - Edit Row', () => {
            test('Enter opens edit dialog for focused row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus first row
                await page.keyboard.press('ArrowDown');

                // Press Enter to edit
                await page.keyboard.press('Enter');

                // Edit dialog should open
                await expect(page.locator('[data-testid="edit-row-dialog"]')).toBeVisible();

                // Close with ESC
                await page.keyboard.press('Escape');
            });
        });

        test.describe('Global Table Shortcuts (Ctrl/Cmd)', () => {
            test('Cmd/Ctrl+M opens Mock Data sheet', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Press Cmd+M (Mac) or Ctrl+M (Win/Linux)
                await whodb.typeCmdShortcut('m');

                // Mock Data sheet should open
                await expect(page.locator('[data-testid="mock-data-sheet"]')).toBeVisible();

                // Close it
                await page.keyboard.press('Escape');
            });

            test('Cmd/Ctrl+A selects all visible rows', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Ensure no rows are selected initially (Radix checkbox uses data-state)
                await expect(page.locator('table tbody tr [data-slot="checkbox"][data-state="checked"]')).not.toBeAttached();

                // Press Cmd+A (Mac) or Ctrl+A (Win/Linux)
                await whodb.typeCmdShortcut('a');

                // All row checkboxes should be checked (Radix checkbox uses data-state)
                const checkboxes = page.locator('table tbody tr [data-slot="checkbox"]');
                const count = await checkboxes.count();
                for (let i = 0; i < count; i++) {
                    await expect(checkboxes.nth(i)).toHaveAttribute('data-state', 'checked');
                }

                // Press Cmd/Ctrl+A again to deselect
                await whodb.typeCmdShortcut('a');

                // All row checkboxes should be unchecked (Radix checkbox uses data-state)
                const checkboxes2 = page.locator('table tbody tr [data-slot="checkbox"]');
                const count2 = await checkboxes2.count();
                for (let i = 0; i < count2; i++) {
                    await expect(checkboxes2.nth(i)).toHaveAttribute('data-state', 'unchecked');
                }
            });

            test('Cmd/Ctrl+Shift+E opens Export dialog', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Press Cmd+Shift+E (Mac) or Ctrl+Shift+E (Win/Linux)
                await whodb.typeCmdShortcut('e', { shift: true });

                // Export dialog should open
                await expect(page.locator('[data-testid="export-dialog"]')).toBeVisible();

                // Close it
                await page.keyboard.press('Escape');
            });

            test('Cmd/Ctrl+E edits focused row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus first row
                await page.keyboard.press('ArrowDown');

                // Press Cmd+E (Mac) or Ctrl+E (Win/Linux)
                await whodb.typeCmdShortcut('e');

                // Edit dialog should open
                await expect(page.locator('[data-testid="edit-row-dialog"]')).toBeVisible();

                // Close it
                await page.keyboard.press('Escape');
            });

            test('Cmd/Ctrl+R refreshes the table', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Get initial row count
                const initialCount = await page.locator('table tbody tr').count();

                // Dispatch the keyboard event via JS to avoid triggering browser refresh
                // (Control+R on Linux refreshes the page natively before the app can handle it)
                await page.evaluate((mod) => {
                    document.dispatchEvent(new KeyboardEvent('keydown', {
                        key: 'r', code: 'KeyR', [mod]: true, bubbles: true
                    }));
                }, process.platform === 'darwin' ? 'metaKey' : 'ctrlKey');

                // Table should still have rows after refresh
                await page.waitForTimeout(2000);
                const rowCount = await page.locator('table tbody tr').count();
                expect(rowCount).toBeGreaterThanOrEqual(1);
            });
        });

        test.describe('Context Menu', () => {
            test('opens context menu with right-click', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Right-click on a row
                await page.locator('table tbody tr').first().click({ button: 'right' });

                // Context menu should be visible with options
                await expect(page.locator('[role="menu"]')).toBeVisible();
                await expect(page.locator('[data-testid="context-menu-edit-row"]')).toBeVisible();
            });

            test('shows keyboard shortcut hints in context menu', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open context menu
                await whodb.openContextMenu(0);

                // Check that shortcut hints are displayed
                await expect(page.locator('[role="menu"]')).toBeVisible();
                // The menu should contain shortcut indicators for shortcuts
                const menuText = await page.locator('[role="menu"]').textContent();
                // Verify menu items exist with shortcut labels
                expect(menuText).toContain('Edit');
                expect(menuText).toContain('Enter'); // Shortcut for Edit
                expect(menuText).toContain('Space'); // Shortcut for Select
            });
        });

        test.describe('ARIA Accessibility', () => {
            test('table has correct ARIA attributes', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Table should have grid role
                await expect(page.locator('table')).toHaveAttribute('role', 'grid');

                // Table should have aria-multiselectable
                await expect(page.locator('table')).toHaveAttribute('aria-multiselectable', 'true');
            });

            test('rows have correct ARIA attributes', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus a row
                await page.keyboard.press('ArrowDown');

                // Focused row should have correct attributes
                const focusedRow = page.locator('table tbody tr[data-focused="true"]');
                await expect(focusedRow).toHaveAttribute('role', 'row');
                await expect(focusedRow).toHaveAttribute('tabindex', '0');

                // Non-focused rows should have tabindex -1 (only check if multiple rows exist)
                const rowCount = await page.locator('table tbody tr').count();
                if (rowCount > 1) {
                    await expect(page.locator('table tbody tr:not([data-focused="true"])').first()).toHaveAttribute('tabindex', '-1');
                }
            });

            test('selected rows have aria-selected attribute', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus and select first row
                await page.keyboard.press('ArrowDown');
                await page.keyboard.press('Space');

                // Row should have aria-selected
                await expect(page.locator('table tbody tr').first()).toHaveAttribute('aria-selected', 'true');
            });
        });

        test.describe('Focus Reset Behavior', () => {
            test('focus resets when changing pages', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Set page size to a small value to see pagination (2 is available in E2E mode)
                await whodb.setTablePageSize(2);
                await page.locator('[data-testid="submit-button"]').click();
                await expect(page.locator('[data-testid="table-page-number"]').first()).toBeAttached({ timeout: 10000 });

                // Focus a row
                await page.keyboard.press('ArrowDown');
                await expect(page.locator('table tbody tr[data-focused="true"]')).toBeAttached();

                // Go to next page (if available)
                const pageNumbers = await page.locator('[data-testid="table-page-number"]').count();
                if (pageNumbers > 1) {
                    await page.locator('[data-testid="table-page-number"]').nth(1).click();

                    // Focus should be reset
                    await expect(page.locator('table tbody tr[data-focused="true"]')).not.toBeAttached();
                }
            });
        });

        test.describe('Pagination Shortcuts', () => {
            test('Cmd/Ctrl+ArrowRight goes to next page', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Set page size to a small value to see pagination
                await whodb.setTablePageSize(2);
                await page.locator('[data-testid="submit-button"]').click();
                await expect(page.locator('[data-testid="table-page-number"]').first()).toBeAttached({ timeout: 10000 });

                // Check we're on page 1
                await expect(page.locator('[data-testid="table-page-number"]').first()).toHaveAttribute('data-active', 'true');

                // Press Cmd+ArrowRight (Mac) or Ctrl+ArrowRight (Win/Linux) to go to next page
                await whodb.typeCmdShortcut('ArrowRight');

                // Should now be on page 2
                await expect(page.locator('[data-testid="table-page-number"]').nth(1)).toHaveAttribute('data-active', 'true');
            });

            test('Cmd/Ctrl+ArrowLeft goes to previous page', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Set page size to a small value to see pagination
                await whodb.setTablePageSize(2);
                await page.locator('[data-testid="submit-button"]').click();
                await expect(page.locator('[data-testid="table-page-number"]').first()).toBeAttached({ timeout: 10000 });

                // Go to page 2 first
                await page.locator('[data-testid="table-page-number"]').nth(1).click();
                await expect(page.locator('[data-testid="table-page-number"]').nth(1)).toHaveAttribute('data-active', 'true');

                // Press Cmd+ArrowLeft (Mac) or Ctrl+ArrowLeft (Win/Linux) to go to previous page
                await whodb.typeCmdShortcut('ArrowLeft');

                // Should now be on page 1
                await expect(page.locator('[data-testid="table-page-number"]').first()).toHaveAttribute('data-active', 'true');
            });

            test('Cmd/Ctrl+ArrowLeft does nothing on first page', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Set page size to a small value to see pagination
                await whodb.setTablePageSize(2);
                await page.locator('[data-testid="submit-button"]').click();
                await expect(page.locator('[data-testid="table-page-number"]').first()).toBeAttached({ timeout: 10000 });

                // Ensure we're on page 1
                await expect(page.locator('[data-testid="table-page-number"]').first()).toHaveAttribute('data-active', 'true');

                // Press Cmd/Ctrl+ArrowLeft - should stay on page 1
                await whodb.typeCmdShortcut('ArrowLeft');

                // Should still be on page 1
                await expect(page.locator('[data-testid="table-page-number"]').first()).toHaveAttribute('data-active', 'true');
            });
        });

        test.describe('Keyboard Shortcuts Help Modal', () => {
            test('pressing ? key opens the shortcuts modal', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Press ? key (Shift+/)
                await page.keyboard.press('Shift+/');

                // Modal should open
                await expect(page.locator('[data-testid="shortcuts-modal"]')).toBeVisible();

                // Verify the modal contains shortcut categories
                await expect(page.locator('[data-testid="shortcuts-category-global"]')).toBeAttached();
                await expect(page.locator('[data-testid="shortcuts-category-navigation"]')).toBeAttached();

                // Close with ESC
                await page.keyboard.press('Escape');
            });
        });

        test.describe('Sidebar Navigation Shortcuts', () => {
            test('Ctrl+B toggles sidebar', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Get initial sidebar state
                await expect(page.locator('[data-sidebar="sidebar"]').first()).toBeAttached();

                // Press Cmd/Ctrl+B to toggle sidebar
                await whodb.typeCmdShortcut('b');

                // Wait for animation
                await page.waitForTimeout(300);

                // Press Cmd/Ctrl+B again to toggle back
                await whodb.typeCmdShortcut('b');

                // Sidebar should be visible again
                await page.waitForTimeout(300);
                await expect(page.locator('[data-sidebar="sidebar"]').first()).toBeAttached();
            });

            test('navigates to first view with number shortcut', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Press Ctrl+1 (Mac) or Alt+1 (Win/Linux) to go to first view (Chat for SQL databases)
                await whodb.typeNavShortcut(1);

                // Should navigate to chat page
                await expect(page).toHaveURL(/\/chat/);
            });

            test('navigates to second view with number shortcut', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Press Ctrl+2 (Mac) or Alt+2 (Win/Linux) to go to second view (Storage Units/Tables)
                await whodb.typeNavShortcut(2);

                // Should navigate to storage-unit page
                await expect(page).toHaveURL(/\/storage-unit/);
            });

            test('navigates to third view with number shortcut', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Press Ctrl+3 (Mac) or Alt+3 (Win/Linux) to go to third view (Graph)
                await whodb.typeNavShortcut(3);

                // Should navigate to graph page
                await expect(page).toHaveURL(/\/graph/);
            });

            test('navigates to fourth view (Scratchpad) with number shortcut', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Press Ctrl+4 (Mac) or Alt+4 (Win/Linux) to go to fourth view (Scratchpad)
                await whodb.typeNavShortcut(4);

                // Should navigate to scratchpad page
                await expect(page).toHaveURL(/\/scratchpad/);
            });
        });

        test.describe('Command Palette (Cmd/Ctrl+K)', () => {
            test('Cmd/Ctrl+K opens command palette', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Press Cmd+K (Mac) or Ctrl+K (Win/Linux)
                await whodb.typeCmdShortcut('k');

                // Command palette should open
                await expect(page.locator('[data-testid="command-palette"]')).toBeVisible();

                // Should have search input
                await expect(page.locator('[data-testid="command-palette-input"]')).toBeVisible();

                // Close with ESC
                await page.keyboard.press('Escape');
                await expect(page.locator('[data-testid="command-palette"]')).not.toBeAttached();
            });

            test('Command palette trigger button opens palette', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Click the trigger button
                await page.locator('[data-testid="command-palette-trigger"]').click();

                // Command palette should open
                await expect(page.locator('[data-testid="command-palette"]')).toBeVisible();

                // Close with ESC
                await page.keyboard.press('Escape');
            });

            test('Command palette shows navigation options', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open command palette
                await whodb.typeCmdShortcut('k');

                // Should show navigation commands
                await expect(page.locator('[data-testid="command-nav-chat"]')).toBeVisible();
                await expect(page.locator('[data-testid="command-nav-storage-units"]')).toBeVisible();
                await expect(page.locator('[data-testid="command-nav-graph"]')).toBeVisible();
                await expect(page.locator('[data-testid="command-nav-scratchpad"]')).toBeVisible();

                // Close
                await page.keyboard.press('Escape');
            });

            test('Command palette shows action options', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open command palette
                await whodb.typeCmdShortcut('k');

                // Should show action commands
                await expect(page.locator('[data-testid="command-action-refresh"]')).toBeVisible();
                await expect(page.locator('[data-testid="command-action-export"]')).toBeVisible();
                await expect(page.locator('[data-testid="command-action-toggle-sidebar"]')).toBeVisible();
                await expect(page.locator('[data-testid="command-action-disconnect"]')).toBeVisible();

                // Close
                await page.keyboard.press('Escape');
            });

            test('Command palette navigates to Chat when selected', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open command palette
                await whodb.typeCmdShortcut('k');

                // Click on Chat navigation
                await page.locator('[data-testid="command-nav-chat"]').click();

                // Should navigate to chat
                await expect(page).toHaveURL(/\/chat/);
            });

            test('Command palette search filters results', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open command palette
                await whodb.typeCmdShortcut('k');

                // Type to search
                await page.locator('[data-testid="command-palette-input"]').fill('graph');

                // Should filter to show only graph
                await expect(page.locator('[data-testid="command-nav-graph"]')).toBeAttached();
                await expect(page.locator('[data-testid="command-nav-chat"]')).not.toBeAttached();

                // Close
                await page.keyboard.press('Escape');
            });
        });

        test.describe('PageUp/PageDown Navigation', () => {
            test('PageDown jumps down by visible rows', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus first row
                await page.keyboard.press('ArrowDown');

                // Get the initial focused row
                await expect(page.locator('table tbody tr').first()).toHaveAttribute('data-focused', 'true');

                // Press PageDown
                await page.keyboard.press('PageDown');

                // First row should no longer be focused (focus moved down)
                await expect(page.locator('table tbody tr').first()).not.toHaveAttribute('data-focused', 'true');
            });

            test('PageUp jumps up by visible rows', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus last row using End key
                await page.keyboard.press('ArrowDown');
                await page.keyboard.press('End');

                // Press PageUp
                await page.keyboard.press('PageUp');

                // Last row should no longer be focused (focus moved up)
                await expect(page.locator('table tbody tr').last()).not.toHaveAttribute('data-focused', 'true');
            });
        });

        test.describe('Query Editor Shortcuts', () => {
            test('Ctrl+Enter executes query in scratchpad', async ({ whodb, page }) => {
                await whodb.goto('scratchpad');

                // Write a simple query
                await whodb.writeCode(0, 'SELECT 1 as test');

                // Focus the editor and press Ctrl+Enter
                const editorSelector = '[role="tabpanel"][data-state="active"] [data-testid="cell-0"] .cm-content';
                await page.locator(editorSelector).click();
                await page.keyboard.press('Control+Enter');

                // Query should execute and show results
                await expect(
                    page.locator('[role="tabpanel"][data-state="active"] [data-testid="cell-0"]')
                        .locator('[data-testid="cell-query-output"], [data-testid="cell-action-output"], [data-testid="cell-error"]')
                ).toBeAttached({ timeout: 5000 });
            });
        });

        test.describe('Column Header Keyboard Sorting', () => {
            test('column headers are focusable', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus the column header
                await page.locator('[data-testid="column-header-id"]').focus();

                // Should have focus
                await expect(page.locator('[data-testid="column-header-id"]')).toBeFocused();
            });

            test('Enter key on focused column header triggers sort', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus and press Enter using trigger (more reliable than type)
                await page.locator('[data-testid="column-header-id"]').focus();
                await page.locator('[data-testid="column-header-id"]').dispatchEvent('keydown', { key: 'Enter' });

                // Wait for sort to be applied and data to refresh
                await page.waitForTimeout(500);

                // Column should now show sort indicator
                await expect(page.locator('[data-testid="column-header-id"]').locator('[data-testid="sort-indicator"]')).toBeAttached();
            });

            test('Space key on focused column header triggers sort', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Focus and press Space using trigger
                await page.locator('[data-testid="column-header-name"]').focus();
                await page.locator('[data-testid="column-header-name"]').dispatchEvent('keydown', { key: ' ' });

                // Wait for sort to be applied and data to refresh
                await page.waitForTimeout(500);

                // Column should now show sort indicator
                await expect(page.locator('[data-testid="column-header-name"]').locator('[data-testid="sort-indicator"]')).toBeAttached();
            });

            test('clicking column header sorts data ascending', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Click to sort by id ascending
                await page.locator('[data-testid="column-header-id"]').click();
                await page.waitForTimeout(500);

                // Column should show sort indicator
                await expect(page.locator('[data-testid="column-header-id"]').locator('[data-testid="sort-indicator"]')).toBeAttached();

                // Verify data is sorted ascending - collect all id values and check order
                const rows = page.locator('table tbody tr');
                const rowCount = await rows.count();
                const ids = [];
                for (let i = 0; i < rowCount; i++) {
                    const idCell = await rows.nth(i).locator('td').nth(1).textContent();
                    const trimmed = idCell.trim();
                    if (trimmed) ids.push(parseInt(trimmed, 10));
                }

                // Check ascending order
                for (let i = 1; i < ids.length; i++) {
                    expect(ids[i]).toBeGreaterThanOrEqual(ids[i - 1]);
                }
            });

            test('clicking column header twice sorts data descending', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Click twice to sort by id descending
                await page.locator('[data-testid="column-header-id"]').click();
                await page.waitForTimeout(500);
                await page.locator('[data-testid="column-header-id"]').click();
                await page.waitForTimeout(500);

                // Column should show sort indicator
                await expect(page.locator('[data-testid="column-header-id"]').locator('[data-testid="sort-indicator"]')).toBeAttached();

                // Verify data is sorted descending
                const rows = page.locator('table tbody tr');
                const rowCount = await rows.count();
                const ids = [];
                for (let i = 0; i < rowCount; i++) {
                    const idCell = await rows.nth(i).locator('td').nth(1).textContent();
                    const trimmed = idCell.trim();
                    if (trimmed) ids.push(parseInt(trimmed, 10));
                }

                // Check descending order
                for (let i = 1; i < ids.length; i++) {
                    expect(ids[i]).toBeLessThanOrEqual(ids[i - 1]);
                }
            });

            test('multiple sorts can be applied by clicking different columns', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Sort by id first
                await page.locator('[data-testid="column-header-id"]').click();
                await page.waitForTimeout(500);

                // Sort by name second
                await page.locator('[data-testid="column-header-name"]').click();
                await page.waitForTimeout(500);

                // Both columns should show sort indicators
                await expect(page.locator('[data-testid="column-header-id"]').locator('[data-testid="sort-indicator"]')).toBeAttached();
                await expect(page.locator('[data-testid="column-header-name"]').locator('[data-testid="sort-indicator"]')).toBeAttached();
            });

            test('clicking column three times cycles through asc, desc, and removes sort', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Get original order before any sorting
                const getIds = async () => {
                    const rows = page.locator('table tbody tr');
                    const rowCount = await rows.count();
                    const ids = [];
                    for (let i = 0; i < rowCount; i++) {
                        const idCell = await rows.nth(i).locator('td').nth(1).textContent();
                        const trimmed = idCell.trim();
                        if (trimmed) ids.push(parseInt(trimmed, 10));
                    }
                    return ids;
                };

                // First click: ascending sort
                await page.locator('[data-testid="column-header-id"]').click();
                await page.waitForTimeout(500);
                await expect(page.locator('[data-testid="column-header-id"]').locator('[data-testid="sort-indicator"]')).toBeAttached();

                // Verify ascending
                let ids = await getIds();
                for (let i = 1; i < ids.length; i++) {
                    expect(ids[i]).toBeGreaterThanOrEqual(ids[i - 1]);
                }

                // Second click: descending sort
                await page.locator('[data-testid="column-header-id"]').click();
                await page.waitForTimeout(500);
                await expect(page.locator('[data-testid="column-header-id"]').locator('[data-testid="sort-indicator"]')).toBeAttached();

                // Verify descending
                ids = await getIds();
                for (let i = 1; i < ids.length; i++) {
                    expect(ids[i]).toBeLessThanOrEqual(ids[i - 1]);
                }

                // Third click: remove sort
                await page.locator('[data-testid="column-header-id"]').click();
                await page.waitForTimeout(500);

                // Sort indicator should be removed
                await expect(page.locator('[data-testid="column-header-id"]').locator('[data-testid="sort-indicator"]')).not.toBeAttached();
            });
        });

        test.describe('Command Palette Sort Commands', () => {
            test('command palette shows sort options when on table view', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open command palette
                await whodb.typeCmdShortcut('k');
                // Wait for command palette to receive columns from the table
                await page.waitForTimeout(300);

                // Should show sort commands with column options (scroll into view as they may be below fold)
                await page.locator('[data-testid="command-sort-id"]').scrollIntoViewIfNeeded();
                await expect(page.locator('[data-testid="command-sort-id"]')).toBeVisible();
                await page.locator('[data-testid="command-sort-name"]').scrollIntoViewIfNeeded();
                await expect(page.locator('[data-testid="command-sort-name"]')).toBeVisible();

                // Close
                await page.keyboard.press('Escape');
            });

            test('selecting sort command from palette sorts the column', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open command palette
                await whodb.typeCmdShortcut('k');
                // Wait for command palette to receive columns from the table
                await page.waitForTimeout(300);

                // Click on sort by id (scroll into view first as it may be below fold)
                await page.locator('[data-testid="command-sort-id"]').scrollIntoViewIfNeeded();
                await page.locator('[data-testid="command-sort-id"]').click();

                // Wait for sort to be applied
                await page.waitForTimeout(500);

                // Column should now show sort indicator
                await expect(page.locator('[data-testid="column-header-id"]').locator('[data-testid="sort-indicator"]')).toBeAttached();
            });

            test('sort commands can be searched in command palette', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Open command palette
                await whodb.typeCmdShortcut('k');
                // Wait for command palette to receive columns from the table
                await page.waitForTimeout(300);

                // Type to search for sort
                await page.locator('[data-testid="command-palette-input"]').fill('sort by name');

                // Should filter to show sort by name
                await expect(page.locator('[data-testid="command-sort-name"]')).toBeAttached();
                await expect(page.locator('[data-testid="command-sort-id"]')).not.toBeAttached();

                // Close
                await page.keyboard.press('Escape');
            });
        });

        // Deletion tests are placed at the end because they modify data and could affect other tests
        test.describe('Cmd/Ctrl+Delete/Backspace - Delete Row', () => {
            test('Delete key without Cmd/Ctrl does not delete row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Get initial row count
                const initialCount = await page.locator('table tbody tr').count();

                // Focus first row
                await page.keyboard.press('ArrowDown');

                // Press Delete without Cmd/Ctrl
                await page.keyboard.press('Delete');

                // Row count should remain the same
                await expect(page.locator('table tbody tr')).toHaveCount(initialCount);
            });

            test('Cmd/Ctrl+Delete triggers delete for focused row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Get initial row count
                const initialCount = await page.locator('table tbody tr').count();

                // Focus first row
                await page.keyboard.press('ArrowDown');

                // Press Cmd+Delete (Mac) or Ctrl+Delete (Win/Linux)
                await whodb.typeCmdShortcut('Delete');

                // Row should be deleted (count decreased)
                await expect(page.locator('table tbody tr')).toHaveCount(initialCount - 1, { timeout: 10000 });
            });

            test('Cmd/Ctrl+Backspace triggers delete for focused row', async ({ whodb, page }) => {
                await whodb.data(tableName);

                // Get initial row count
                const initialCount = await page.locator('table tbody tr').count();

                // Focus first row
                await page.keyboard.press('ArrowDown');

                // Press Cmd+Backspace (Mac) or Ctrl+Backspace (Win/Linux)
                await whodb.typeCmdShortcut('Backspace');

                // Row should be deleted (count decreased)
                await expect(page.locator('table tbody tr')).toHaveCount(initialCount - 1, { timeout: 10000 });
            });
        });
    });

});
