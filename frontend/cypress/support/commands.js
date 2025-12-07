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

function extractText(chain) {
    return chain.invoke("html").then((html) => {
        return html.replace(/<br\s*\/?>/g, '\n') // Replace <br> with new lines
                   .replace(/<\/(p|div|li|h[1-6])>/g, '\n') // New line after block elements
                   .replace(/&nbsp;/g, ' ') // Replace non-breaking spaces
                   .replace(/<[^>]*>/g, '')
                   .trim(); // Remove remaining HTML tags
    });
}

Cypress.Commands.add("goto", (route) => {
    cy.visit(`/${route}`);
});

Cypress.Commands.add('login', (databaseType, hostname, username, password, database, advanced={}) => {
    cy.visit('/login');

    // Poll for telemetry modal and dismiss if it appears (handles async React rendering)
    const tryDismissTelemetry = (attemptsLeft = 5) => {
        cy.get('body').then($body => {
            const $btn = $body.find('button').filter(function() {
                return this.textContent.includes('Disable Telemetry');
            });
            if ($btn.length) {
                cy.wrap($btn).click();
            } else if (attemptsLeft > 1) {
                cy.wait(300).then(() => tryDismissTelemetry(attemptsLeft - 1));
            }
        });
    };
    tryDismissTelemetry();

    if (databaseType) {
        cy.get('[data-testid="database-type-select"]').click();
        cy.get(`[data-value="${databaseType}"]`).click();
    }

    if (hostname !== undefined && hostname !== null) {
        cy.get('[data-testid="hostname"]').clear();
        if (hostname !== '') {
            cy.get('[data-testid="hostname"]').type(hostname);
        }
    }

    // Only interact with username field if explicitly provided
    if (username !== undefined) {
        cy.get('[data-testid="username"]').clear();
        if (username !== null && username !== '') {
            cy.get('[data-testid="username"]').type(username);
        }
    }

    // Only interact with password field if explicitly provided
    if (password !== undefined) {
        cy.get('[data-testid="password"]').clear();
        if (password !== null && password !== '') {
            cy.get('[data-testid="password"]').type(password, {log: false});
        }
    }

    // Handle database field based on type
    if (database !== undefined) {
        if (databaseType === "Sqlite3") {
            cy.get('[data-testid="database"]').click();
            cy.get(`[data-value="${database}"]`).click();
        } else {
            cy.get('[data-testid="database"]').clear();
            if (database !== null && database !== '') {
                cy.get('[data-testid="database"]').type(database);
            }
        }
    }

    // Handle advanced options
    if (Object.keys(advanced).length > 0) {
        cy.get('[data-testid="advanced-button"]').click();
        for (const [key, value] of Object.entries(advanced)) {
            cy.get(`[data-testid="${key}-input"]`).clear();
            if (value !== '') {
                cy.get(`[data-testid="${key}-input"]`).type(value);
            }
        }
    }

    // Intercept login API call right before clicking to avoid catching other requests
    cy.intercept('POST', '**/api/query').as('loginQuery');

    cy.get('[data-testid="login-button"]').click();

    cy.wait('@loginQuery', {timeout: 60000}).then((interception) => {
        // Log if there was an error in the response for debugging
        if (interception.response?.body?.errors) {
            cy.log('Login API returned errors:', JSON.stringify(interception.response.body.errors));
        }
    });

    // Wait for successful login - sidebar should appear after navigation
    cy.get('[data-testid="sidebar-database"], [data-testid="sidebar-schema"]', {timeout: 30000})
        .should('exist');
});

Cypress.Commands.add('setAdvanced', (type, value) => {
});

Cypress.Commands.add("selectDatabase", (value) => {
    cy.get('[data-testid="sidebar-database"]').click().get(`[data-value="${value}"]`).click();
});

Cypress.Commands.add("selectSchema", (value) => {
    cy.get('[data-testid="sidebar-schema"]').click().get(`[data-value="${value}"]`).click();
});

Cypress.Commands.add('explore', (tableName) => {
    // Ensure card view is set for consistent test behavior
    cy.window().then(win => {
        const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
        settings.storageUnitView = '"card"';
        win.localStorage.setItem('persist:settings', JSON.stringify(settings));
    });

    // Visit the storage-unit page (will use card view)
    cy.visit('/storage-unit');
    // Wait for cards to load
    cy.get('[data-testid="storage-unit-card"]', {timeout: 15000})
        .should('have.length.at.least', 1);

    // Find the card containing the exact table name by iterating through cards
    return cy.get('[data-testid="storage-unit-card"]', {timeout: 10000}).then($cards => {
        let targetCard = null;
        $cards.each((_, card) => {
            const nameEl = Cypress.$(card).find('[data-testid="storage-unit-name"]');
            if (nameEl.length && nameEl.text().trim() === tableName) {
                targetCard = card;
                return false; // break the loop
            }
        });

        if (!targetCard) {
            throw new Error(`Could not find storage unit card with name: ${tableName}`);
        }

        // Click the explore button within the found card
        cy.wrap(targetCard)
            .find('[data-testid="explore-button"]')
            .scrollIntoView()
            .should('be.visible')
            .click({force: true});
    });
});

Cypress.Commands.add('getExploreFields', () => {
    // Returns a list of [key, value] arrays from the explore fields panel
    return cy.document().then((doc) => {
        const result = [];
        const rows = doc.querySelectorAll('[data-testid="explore-fields"] p');
        rows.forEach(row => {
            const spans = row.querySelectorAll('span');
            if (spans.length >= 2) {
                const key = spans[0].textContent.trim();
                const value = spans[1].textContent.trim();
                result.push([key, value]);
            }
        });
        return result;
    });
});

Cypress.Commands.add('data', (tableName) => {
    // Ensure card view is set for consistent test behavior
    cy.window().then(win => {
        const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
        settings.storageUnitView = '"card"';
        win.localStorage.setItem('persist:settings', JSON.stringify(settings));
    });

    // Visit the storage-unit page (will use card view)
    cy.visit('/storage-unit');
    // Wait for cards to load
    cy.get('[data-testid="storage-unit-card"]', {timeout: 15000})
        .should('have.length.at.least', 1);

    // Find the card containing the exact table name by iterating through cards
    cy.get('[data-testid="storage-unit-card"]', {timeout: 10000}).then($cards => {
        let targetCard = null;
        $cards.each((_, card) => {
            const nameEl = Cypress.$(card).find('[data-testid="storage-unit-name"]');
            if (nameEl.length && nameEl.text().trim() === tableName) {
                targetCard = card;
                return false; // break the loop
            }
        });

        if (!targetCard) {
            throw new Error(`Could not find storage unit card with name: ${tableName}`);
        }

        // Click the data button within the found card
        cy.wrap(targetCard)
            .find('[data-testid="data-button"]')
            .first()
            .scrollIntoView()
            .should('be.visible')
            .click({force: true});
    });

    // Wait for URL to change to explore page
    cy.url().should('include', '/storage-unit/explore');
    // Wait for the page to stabilize - ensure we're not on the list page anymore
    // The list view has a hidden table, so we must check cards are gone
    cy.get('[data-testid="storage-unit-card"]', {timeout: 5000}).should('not.exist');
    // Wait for a VISIBLE table (not the hidden list view table)
    cy.get('table:visible', {timeout: 10000}).should('exist');
    return cy.get('table:visible tbody tr', {timeout: 15000}).should('have.length.at.least', 1);
});

Cypress.Commands.add('sortBy', (index) => {
    return cy.get('th').eq(index + 1).click();
});

Cypress.Commands.add('assertNoDataAvailable', () => {
    // Assert the empty-state text is visible (retries until timeout)
    cy.contains(/No data available/i, {timeout: 10000}).should('be.visible');
});


Cypress.Commands.add('getTableData', () => {
    // First wait for a VISIBLE table to exist (not hidden list view tables)
    return cy.get('table:visible', {timeout: 10000}).should('exist').then(() => {
        // Wait for at least one table row to be present with proper scoping
        return cy.get('table:visible tbody tr', {timeout: 10000})
            .then(() => {
                // Additional wait to ensure data is fully rendered
                cy.wait(100);

                // Now get the visible table and extract data
                return cy.get('table:visible').first().then($table => {
                    const columns = Cypress.$.makeArray($table.find('th'))
                        .map(el => el.innerText.trim());

                    const rows = Cypress.$.makeArray($table.find('tbody tr')).map(row => {
                        const cells = Cypress.$(row).find('td');
                        return Cypress.$.makeArray(cells).map(cell => cell.innerText.trim());
                    });

                    return { columns, rows };
                });
            });
    });
});

Cypress.Commands.add("setTablePageSize", (pageSize) => {
    cy.get('[data-testid="table-page-size"]').click();
    cy.get(`[role="option"][data-value="${pageSize}"]`).click();
});

Cypress.Commands.add("getTablePageSize", () => {
    return cy.get('[data-testid="table-page-size"]').then(($el) => {
        return $el.innerText.trim();
    });
});

Cypress.Commands.add("submitTable", () => {
    cy.get('body').then($body => {
        const dialogVisible = $body.find('[role="dialog"]:visible').length > 0;
        if (dialogVisible) {
            cy.get('[role="dialog"]:visible').within(() => {
                cy.get('[data-testid="add-conditions-button"]').click();
                cy.wait(300);
            });
        }
    });
    cy.get('[data-testid="submit-button"]:visible').then($btn => {
        if ($btn.length > 0) {
            cy.wrap($btn).click({force: true});
            cy.wait(200);
        }
    });
});

Cypress.Commands.add("whereTable", (fieldArray) => {
    cy.get('[data-testid="where-button"]').click();

    // Wait for the dialog/sheet to be visible
    cy.wait(500);

    // Detect which mode we're in by checking what's visible
    cy.get('body').then($body => {
        const isSheetMode = $body.find('[role="dialog"]').length > 0 &&
                           $body.find('[data-testid*="sheet-field"]').length > 0;
        const isPopoverMode = $body.find('[data-testid="field-key"]').length > 0;

        cy.log(`Where condition mode detected: ${isSheetMode ? 'sheet' : isPopoverMode ? 'popover' : 'unknown'}`);

        fieldArray.forEach(([key, operator, value]) => {
            cy.log(`Adding condition: ${key} ${operator} ${value}`);

            if (isSheetMode) {
                // Sheet mode - always uses index 0 for new conditions
                // Try both with and without index for compatibility
                cy.get('body').then($body => {
                    if ($body.find('[data-testid="sheet-field-key-0"]').length > 0) {
                        cy.get('[data-testid="sheet-field-key-0"]').click();
                    } else if ($body.find('[data-testid="sheet-field-key"]').length > 0) {
                        cy.get('[data-testid="sheet-field-key"]').click();
                    }
                });
                cy.get(`[data-value="${key}"]`).click();

                cy.get('body').then($body => {
                    if ($body.find('[data-testid="sheet-field-operator-0"]').length > 0) {
                        cy.get('[data-testid="sheet-field-operator-0"]').click();
                    } else if ($body.find('[data-testid="sheet-field-operator"]').length > 0) {
                        cy.get('[data-testid="sheet-field-operator"]').click();
                    }
                });
                cy.get(`[data-value="${operator}"]`).click();

                cy.get('body').then($body => {
                    if ($body.find('[data-testid="sheet-field-value-0"]').length > 0) {
                        cy.get('[data-testid="sheet-field-value-0"]').clear().type(value);
                    } else if ($body.find('[data-testid="sheet-field-value"]').length > 0) {
                        cy.get('[data-testid="sheet-field-value"]').clear().type(value);
                    }
                });

                // In sheet mode, add button is inside the dialog
                cy.wait(100);
                cy.get('[role="dialog"]').within(() => {
                    cy.get('[data-testid="add-conditions-button"]').click();
                });
            } else {
                // Popover mode - uses non-indexed test IDs
                cy.get('[data-testid="field-key"]').first().click();
                cy.get(`[data-value="${key}"]`).click();

                cy.get('[data-testid="field-operator"]').first().click();
                cy.get(`[data-value="${operator}"]`).click();

                cy.get('[data-testid="field-value"]').first().clear().type(value);

                // In popover mode, try multiple selectors for add button
                cy.wait(100);
                cy.get('body').then($body => {
                    if ($body.find('[data-testid="add-condition-button"]').length > 0) {
                        cy.get('[data-testid="add-condition-button"]').click();
                    } else {
                        // Fallback to finding button by text
                        cy.get('button').contains('Add').click();
                    }
                });
            }

            // Wait for the condition to be added
            cy.wait(200);
        });

        // Close the dialog/popover
        if (isSheetMode) {
            // Sheet mode - the sheet doesn't auto-close after adding, so we need to close it
            // First check if there's a close button or use Escape
            cy.get('body').then($body => {
                // Try to find a close button in the sheet
                if ($body.find('[role="dialog"] button[aria-label="Close"]').length > 0) {
                    cy.get('[role="dialog"] button[aria-label="Close"]').click();
                } else {
                    // Fall back to Escape key
                    cy.get('body').type('{esc}');
                }
            });
            // Wait a moment for close animation to start
            cy.wait(100);
            // Wait for the sheet to fully close by checking that the dialog is gone
            cy.get('[role="dialog"]', { timeout: 5000 }).should('not.exist');
            // Also ensure body no longer has scroll lock
            cy.get('body', { timeout: 5000 }).should('not.have.attr', 'data-scroll-locked');
            // Additional wait to ensure DOM updates are complete
            cy.wait(300);
        } else {
            // Popover mode - might have a cancel button or close on click outside
            cy.get('body').then($body => {
                if ($body.find('[data-testid="cancel-button"]').length > 0) {
                    cy.get('[data-testid="cancel-button"]').click();
                } else {
                    // Click outside to close popover
                    cy.get('body').click(0, 0);
                }
            });
        }
        cy.wait(500);
    });
});

// Helper command to check if we're in sheet mode or popover mode
Cypress.Commands.add("getWhereConditionMode", () => {
    return cy.get('body').then($body => {
        const hasSheetFields = $body.find('[data-testid*="sheet-field"]').length > 0;
        const hasPopoverBadges = $body.find('[data-testid="where-condition-badge"]').length > 0;
        const hasFieldKey = $body.find('[data-testid="field-key"]').length > 0;

        if (hasSheetFields || (!hasPopoverBadges && !hasFieldKey)) {
            return 'sheet';
        }
        return 'popover';
    });
});

// Helper to get condition count
Cypress.Commands.add("getConditionCount", () => {
    return cy.get('body').then($body => {
        // In sheet mode, parse from button text
        const whereButton = $body.find('[data-testid="where-button"]');
        if (whereButton.length > 0) {
            const text = whereButton.text();
            const match = text.match(/(\d+)\s+Condition/);
            if (match) return parseInt(match[1]);
        }

        // In popover mode, count badges + hidden conditions from "+N more" button
        let count = 0;
        const badges = $body.find('[data-testid="where-condition-badge"]');
        count += badges.length;

        // Check for "+N more" button which indicates hidden conditions
        const moreButton = $body.find('[data-testid="more-conditions-button"]');
        if (moreButton.length > 0) {
            const moreText = moreButton.text();
            const moreMatch = moreText.match(/\+(\d+)/);
            if (moreMatch) {
                count += parseInt(moreMatch[1]);
            }
        }

        return count;
    });
});

// Helper to verify condition text
Cypress.Commands.add("verifyCondition", (index, expectedText) => {
    cy.getWhereConditionMode().then(mode => {
        if (mode === 'popover') {
            cy.get('[data-testid="where-condition-badge"]').eq(index).should('contain.text', expectedText);
        } else {
            // In sheet mode, would need to open sheet to see conditions
            cy.log(`Sheet mode: Skipping condition text verification for "${expectedText}"`);
        }
    });
});

// Helper to click on a condition to edit it
Cypress.Commands.add("clickConditionToEdit", (index) => {
    cy.getWhereConditionMode().then(mode => {
        if (mode === 'popover') {
            cy.get('[data-testid="where-condition-badge"]').eq(index).click();
        } else {
            cy.log('Sheet mode: Cannot click individual conditions - need to open sheet');
        }
    });
});

// Helper to remove a specific condition
Cypress.Commands.add("removeCondition", (index) => {
    cy.getWhereConditionMode().then(mode => {
        if (mode === 'popover') {
            cy.get('[data-testid="remove-where-condition-button"]').eq(index).click();
        } else {
            cy.log('Sheet mode: Need to open sheet to remove specific conditions');
        }
    });
});

// Helper to update field value in edit mode
Cypress.Commands.add("updateConditionValue", (value) => {
    cy.getWhereConditionMode().then(mode => {
        if (mode === 'popover') {
            cy.get('[data-testid="field-value"]').clear().type(value);
        } else {
            cy.get('[data-testid="sheet-field-value-0"]').clear().type(value);
        }
    });
});

// Helper to check for more conditions button
Cypress.Commands.add("checkMoreConditionsButton", (expectedText) => {
    cy.getWhereConditionMode().then(mode => {
        if (mode === 'popover') {
            cy.get('[data-testid="more-conditions-button"]').should('be.visible').and('contain.text', expectedText);
        } else {
            cy.log('Sheet mode: No more-conditions button - all managed in sheet');
        }
    });
});

// Helper to click more conditions button
Cypress.Commands.add("clickMoreConditions", () => {
    cy.getWhereConditionMode().then(mode => {
        if (mode === 'popover') {
            // First check if it's a button that expands conditions in place
            cy.get('body').then($body => {
                const moreButton = $body.find('[data-testid="more-conditions-button"]');
                if (moreButton.length > 0) {
                    // Click the more conditions button
                    cy.get('[data-testid="more-conditions-button"]').click();

                    // Wait to see if it opens a sheet or expands in place
                    cy.wait(500);

                    // Check if a dialog opened
                    cy.get('body').then($body => {
                        const hasDialog = $body.find('[role="dialog"]').length > 0;
                        if (!hasDialog) {
                            // No dialog means it expanded in place, we're done
                            cy.log('Expanded conditions in place');
                        } else {
                            cy.log('Opened conditions sheet');
                        }
                    });
                } else {
                    cy.log('No more conditions button found');
                }
            });
        } else {
            // In sheet mode, open the where button sheet
            cy.get('[data-testid="where-button"]').click();
        }
    });
});

// Helper to save changes in a sheet
Cypress.Commands.add("saveSheetChanges", () => {
    // Click save button - matches various button texts used in different sheets
    cy.get('[role="dialog"]').within(() => {
        cy.contains('button', /^(Add|Update|Add to Page|Add Condition|Save Changes)$/).click();
    });
});

// Helper to remove conditions in sheet
Cypress.Commands.add("removeConditionsInSheet", (keepFirst = true) => {
    cy.get('body').then($body => {
        // Try both possible selectors
        let removeButtons = $body.find('[data-testid^="delete-existing-filter-"]');
        let selectorPrefix = 'delete-existing-filter-';

        if (removeButtons.length === 0) {
            removeButtons = $body.find('[data-testid^="remove-sheet-filter-"]');
            selectorPrefix = 'remove-sheet-filter-';
        }

        const count = removeButtons.length;
        const startIndex = keepFirst ? count - 1 : count - 1;
        const endIndex = keepFirst ? 1 : 0;

        for (let i = startIndex; i >= endIndex; i--) {
            cy.get(`[data-testid="${selectorPrefix}${i}"]`).click();
        }
    });
});

Cypress.Commands.add("clearWhereConditions", () => {
    // First check if we're in popover mode by looking for visible badges
    cy.get('body').then($body => {
        const visibleBadges = $body.find('[data-testid="where-condition-badge"]').length;

        if (visibleBadges > 0) {
            // Popover mode - badges are always visible, remove them directly
            // Recursively remove badges until none are left
            function removeBadges() {
                cy.get('body').then($b => {
                    const remaining = $b.find('[data-testid="remove-where-condition-button"]').length;
                    if (remaining > 0) {
                        cy.get('[data-testid="remove-where-condition-button"]').first().click({ force: true });
                        cy.wait(100);
                        removeBadges();
                    }
                });
            }
            removeBadges();
        } else {
            // Sheet mode - check button text to see if there are conditions
            cy.get('[data-testid="where-button"]').then($button => {
                const buttonText = $button.text();
                // In sheet mode, button shows "N Condition(s)" or "10+ Conditions" when there are conditions
                // Only "Add" means no conditions
                if (buttonText.trim() === 'Add') {
                    return;
                }

                // Click where button to open sheet
                cy.get('[data-testid="where-button"]').click();
                cy.wait(500);

                // Delete all existing filters by clicking index 0 repeatedly
                function deleteAllFilters() {
                    cy.get('body').then($b => {
                        const remaining = $b.find('[data-testid^="delete-existing-filter-"]').length;
                        if (remaining > 0) {
                            cy.get('[data-testid="delete-existing-filter-0"]').click();
                            cy.wait(100);
                            deleteAllFilters();
                        }
                    });
                }
                deleteAllFilters();

                // Close sheet
                cy.get('body').type('{esc}');
                // Wait a moment for close animation to start
                cy.wait(100);
                // Wait for the sheet to fully close
                cy.get('[role="dialog"]', { timeout: 5000 }).should('not.exist');
                cy.get('body', { timeout: 5000 }).should('not.have.attr', 'data-scroll-locked');
                cy.wait(300);
            });
        }
    });
});

Cypress.Commands.add("setWhereConditionMode", (mode) => {
    // Set the where condition mode via localStorage and reload to ensure consistency
    cy.window().then((win) => {
        const settingsKey = 'persist:settings';
        try {
            const currentSettings = JSON.parse(win.localStorage.getItem(settingsKey) || '{}');
            currentSettings.whereConditionMode = `"${mode}"`; // The value needs to be a JSON string
            win.localStorage.setItem(settingsKey, JSON.stringify(currentSettings));
        } catch (e) {
            win.localStorage.setItem(settingsKey, JSON.stringify({ whereConditionMode: `"${mode}"`, _persist: '{"version":-1,"rehydrated":true}' }));
        }
    });
    // Reload the page to make sure the setting is applied
    cy.reload();
});

Cypress.Commands.add("getHighlightedCell", (options = {}) => {
    return cy.get('td.table-search-highlight', options);
});

Cypress.Commands.add("getHighlightedRows", () => {
    return cy.get('tr:has(td.table-search-highlight)').then(($rows) => {
        const rows = [];
        $rows.each((index, row) => {
            const cells = Cypress.$(row).find('td').toArray()
                .map(cell => cell.innerText.trim());
            rows.push(cells);
        });
        return rows;
    });
});

Cypress.Commands.add("addRow", (data, isSingleInput = false) => {
    cy.get('[data-testid="add-row-button"]').click();

    // Check if we have a single "document" field (for Elasticsearch/document databases)
    cy.get('body').then($body => {
        if (isSingleInput) {
            // Document database - single text box for JSON
            const jsonString = typeof data === 'string' ? data : JSON.stringify(data, null, 2);
            cy.get('[data-testid="add-row-field-document"] input, [data-testid="add-row-field-document"] textarea').first()
                .clear()
                .type(jsonString, {parseSpecialCharSequences: false});
        } else {
            // Traditional database - multiple fields
            for (const [key, value] of Object.entries(data)) {
                cy.get(`[data-testid="add-row-field-${key}"] input`).clear().type(value);
            }
        }
    });

    cy.get('[data-testid="submit-add-row-button"]').click();

    // Wait for the operation to complete - dialog should close on success
    // Use longer timeout to account for slow database operations
    cy.get('[data-testid="submit-add-row-button"]', {timeout: 10000}).should('not.exist');

    // Ensure body no longer has scroll lock
    cy.get('body', { timeout: 5000 }).should('not.have.attr', 'data-scroll-locked');

    // Wait for GraphQL mutation to complete and UI to update
    cy.wait(500);

    // Ensure table is visible
    cy.get('table tbody').should('be.visible');
});

Cypress.Commands.add("openContextMenu", (rowIndex, maxRetries = 3) => {
    const attemptContextMenu = (attempt) => {
        // Get a fresh reference to the row
        cy.get('table tbody tr').eq(rowIndex).as('targetRow');

        // Scroll into view and wait for it to stabilize (virtualization may need time)
        cy.get('@targetRow').scrollIntoView();
        cy.wait(200);

        // Right-click to open context menu (force: true handles visibility issues with virtualization)
        cy.get('@targetRow').rightclick({ force: true });

        // Wait for context menu to render
        cy.wait(300);

        // Check if context menu appeared
        cy.get('body').then($body => {
            const menuExists = $body.find('[data-testid="context-menu-edit-row"], [data-testid="context-menu-more-actions"]').length > 0;

            if (!menuExists && attempt < maxRetries) {
                // Close any partial menu state by clicking elsewhere
                cy.get('body').click(0, 0);
                cy.wait(100);
                // Retry
                attemptContextMenu(attempt + 1);
            } else if (!menuExists) {
                // Final attempt failed, let Cypress assertion handle it
                cy.get('[data-testid="context-menu-edit-row"], [data-testid="context-menu-more-actions"]', { timeout: 5000 })
                    .should('exist');
            }
        });
    };

    attemptContextMenu(1);
});

Cypress.Commands.add("deleteRow", (rowIndex) => {
    // Get initial row count to verify deletion
    cy.get('table tbody tr').its('length').then(initialRowCount => {
        // Ensure the target row exists before interacting
        cy.get('table tbody tr').should('have.length.greaterThan', rowIndex);

        // Use the helper to open context menu with retry logic
        cy.openContextMenu(rowIndex);

        // Use scrollIntoView and force click to handle overflow issues with wide tables
        cy.get('[data-testid="context-menu-more-actions"]', {timeout: 5000})
            .scrollIntoView()
            .should('exist')
            .click({force: true});
        cy.get('[data-testid="context-menu-delete-row"]', {timeout: 5000})
            .scrollIntoView()
            .should('exist')
            .click({force: true});

        // Wait for the row to be removed by checking the row count
        // Use a timeout to handle async deletion
        cy.get('table tbody tr', { timeout: 10000 }).should('have.length', initialRowCount - 1);
    });
});

Cypress.Commands.add("updateRow", (rowIndex, columnIndex, text, cancel = true) => {
    // Wait for table to stabilize
    cy.wait(500);

    // Use the helper to open context menu with retry logic
    cy.openContextMenu(rowIndex);

    // Wait for the menu to be visible, then click the "Edit row" item
    // Use scrollIntoView and force click to handle overflow issues with wide tables
    cy.get('[data-testid="context-menu-edit-row"]', {timeout: 5000})
        .scrollIntoView()
        .should('exist')
        .click({force: true});

    // Try to find the standard editable field first
    cy.get('body').then(($body) => {
        if ($body.find(`[data-testid="editable-field-${columnIndex}"]`).length > 0) {
            // Standard field-based editing (SQL databases)
            cy.get(`[data-testid="editable-field-${columnIndex}"]`)
                .should('exist')
                .clear()
                .type(text, {force: true, parseSpecialCharSequences: false});
        } else {
            // Document-based editing (MongoDB, Elasticsearch)
            // Look for a textarea or input that contains the JSON document
            cy.get('textarea, input[type="text"]').then(($elements) => {
                // Find the element that contains JSON-like content or is empty and ready for input
                const targetElement = $elements.filter((index, el) => {
                    const value = el.value;
                    return value === '' || value.startsWith('{') || value.startsWith('[');
                })[0];

                if (targetElement) {
                    cy.wrap(targetElement)
                        .clear()
                        .type(text, {force: true, parseSpecialCharSequences: false});
                } else {
                    // Fallback: use the first textarea or text input
                    cy.get('textarea, input[type="text"]').first()
                        .clear()
                        .type(text, {force: true, parseSpecialCharSequences: false});
                }
            });
        }
    });

    // Click cancel (escape key) or update as requested
    if (cancel) {
        // Close the sheet by pressing Escape
        cy.get('body').type('{esc}');
        // Wait for the sheet to disappear
        cy.contains('Edit Row').should('not.exist');
        // Ensure body no longer has scroll lock
        cy.get('body', { timeout: 5000 }).should('not.have.attr', 'data-scroll-locked');
    } else {
        cy.get(`[data-testid="update-button"]`).click();
        // Wait for the update to complete by asserting the sheet is gone.
        cy.get(`[data-testid="update-button"]`).should('not.exist');
        // Ensure body no longer has scroll lock
        cy.get('body', { timeout: 5000 }).should('not.have.attr', 'data-scroll-locked');
    }
});

Cypress.Commands.add("getPageNumbers", () => {
    return cy.get('[data-testid="table-page-number"]').then(($els) => {
        return $els.toArray().map(el => el.innerText.trim());
    });
});

Cypress.Commands.add("searchTable", (search) => {
    // Break up the chain to avoid element detachment after clear
    cy.get('[data-testid="table-search"]').clear();
    cy.get('[data-testid="table-search"]').type(`${search}{enter}`);
    // Wait for search to process and highlight to appear
    cy.wait(300);
});

Cypress.Commands.add("getGraph", () => {
    // Wait for the graph to be fully loaded - nodes should exist and be visible
    cy.get('.react-flow__node', {timeout: 10000}).should('be.visible');

    // Add a small wait to ensure layout has completed (React Flow layout takes time)
    cy.wait(400); // Slightly more than the 300ms layout timeout

    return cy.get('.react-flow__node').then(($nodeEls) => {
        const nodes = $nodeEls.toArray().map(el => el.getAttribute("data-id"));

        // Check if edges exist, if not, return graph with empty connections
        return cy.get('body').then(($body) => {
            if ($body.find('.react-flow__edge').length > 0) {
                return cy.get('.react-flow__edge').then(($edgeEls) => {
                    const edges = $edgeEls.toArray().map(el => {
                        // Edge ID format is either:
                        // - "source->target" (old format without handles)
                        // - "source-sourceCol-target-targetCol" (new format with column info)
                        const edgeId = el.getAttribute("data-testid").slice("rf__edge-".length);

                        let source, target;
                        if (edgeId.includes("->")) {
                            // Old format: "source->target"
                            [source, target] = edgeId.split("->");
                        } else {
                            // New format: "source-sourceCol-target-targetCol"
                            // Extract source and target by finding the node that matches
                            const parts = edgeId.split("-");
                            // Try to match against known nodes
                            for (let i = 1; i < parts.length; i++) {
                                const possibleSource = parts.slice(0, i).join("-");
                                if (nodes.includes(possibleSource)) {
                                    source = possibleSource;
                                    // Target is the rest after skipping the source column
                                    const remaining = parts.slice(i + 1);
                                    for (let j = 1; j <= remaining.length; j++) {
                                        const possibleTarget = remaining.slice(0, j).join("-");
                                        if (nodes.includes(possibleTarget)) {
                                            target = possibleTarget;
                                            break;
                                        }
                                    }
                                    break;
                                }
                            }

                            // Fallback: if we can't parse it, skip this edge
                            if (!source || !target) {
                                return null;
                            }
                        }

                        return {source, target};
                    }).filter(edge => edge !== null);

                    const graph = {};
                    nodes.forEach(node => {
                        const targets = edges
                            .filter(edge => edge.source === node)
                            .map(edge => edge.target);
                        // Deduplicate targets (multiple FK columns can create duplicate edges)
                        graph[node] = [...new Set(targets)];
                    });
                    return graph;
                });
            } else {
                // No edges exist, create graph with empty connections
                const graph = {};
                nodes.forEach(node => {
                    graph[node] = [];
                });
                return graph;
            }
        });
    });
});

Cypress.Commands.add("getGraphNode", (nodeId) => {
    // Returns the key-value data from the "users" node in the graph, as an array of [key, value] pairs
    return cy.document().then((doc) => {
        const el = doc.querySelector(`[data-testid="rf__node-${nodeId}"]`);
        if (!el) return [];
        const result = [];
        // Find all <p> elements inside the node (like in getExploreFields)
        const rows = el.querySelectorAll('p');
        rows.forEach(row => {
            const spans = row.querySelectorAll('span');
            if (spans.length >= 2) {
                const key = spans[0].textContent.trim();
                const value = spans[1].textContent.trim();
                result.push([key, value]);
            }
        });
        return result;
    });
});

Cypress.Commands.add("addCell", (afterIndex) => {
    return cy.get(`[role="tabpanel"][data-state="active"] [data-testid="cell-${afterIndex}"] [data-testid="add-cell-button"]`).click();
});

Cypress.Commands.add("removeCell", (index) => {
    return cy.get(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="delete-cell-button"]`).click();
});

Cypress.Commands.add("writeCode", (index, text) => {
    const selector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] .cm-content`;

    // Click to focus, clear, and then type.
    // Using {force: true} for type to handle cases where the editor might be partially obscured.
    cy.get(selector)
        .scrollIntoView()
        .should('be.visible')
        .click()
        .clear()
        .type(text, { parseSpecialCharSequences: false, force: true });

    // Blur to ensure state updates
    cy.get(selector).blur();

    // Small wait to ensure React state has fully updated
    cy.wait(100);
});

Cypress.Commands.add("runCode", (index) => {
    const buttonSelector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="query-cell-button"]`;

    // The button is only visible on hover. Forcing the click is the standard
    // and most reliable way to handle this scenario in Cypress tests.
    // Using .first() to prevent errors when multiple buttons are found.
    cy.get(buttonSelector).first().click({ force: true });

    // Just wait for the query results to appear
    // Don't check for loading spinner since queries execute very quickly on localhost
    cy.get(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"]`).within(() => {
        cy.get('[data-testid="cell-query-output"], [data-testid="cell-action-output"], [data-testid="cell-error"]', { timeout: 5000 })
            .should('exist');
    });
});

Cypress.Commands.add("getCellQueryOutput", (index) => {
    return cy.get(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="cell-query-output"] table`, {timeout: 10000}).then($table => {
        const columns = Cypress.$.makeArray($table.find('th'))
            .map(el => el.innerText.trim());

        const rows = Cypress.$.makeArray($table.find('tbody tr')).map(row => {
            return Cypress.$.makeArray(Cypress.$(row).find('td'))
                .map(cell => cell.innerText.trim());
        });
        return { columns, rows };
    });
});

Cypress.Commands.add("getCellActionOutput", (index) => {
    return cy.get(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="cell-action-output"]`, {timeout: 10000})
        .should('exist')
        .then($el => extractText(cy.wrap($el)));
});

Cypress.Commands.add("getCellError", (index) => {
    // Wait for the error element to appear
    return cy.get(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="cell-error"]`, {timeout: 10000})
        .should('exist')
        .should('be.visible')
        .invoke('text')
        .then((text) => {
            // Remove "Error" prefix from AlertTitle
            return text.replace(/^Error\s*/i, '').trim();
        });
});

Cypress.Commands.add('logout', () => {
    cy.get('body').then($body => {
        // Check if we're on a page with sidebar (i.e., logged in)
        const hasSidebar = $body.find('[data-sidebar="sidebar"]').length > 0 ||
            $body.find('[data-sidebar="trigger"]').length > 0;

        if (!hasSidebar) {
            // Not logged in or on login page - nothing to logout from
            cy.log('No sidebar found - skipping logout (may not be logged in)');
            return;
        }

        // Check if the sidebar trigger button exists and is visible (indicates sidebar is closed)
        const sidebarTrigger = $body.find('[data-sidebar="trigger"]:visible');
        if (sidebarTrigger.length > 0 && !$body.text().includes('Logout Profile')) {
            // Sidebar is closed, open it first
            cy.get('[data-sidebar="trigger"]').first().click();
            cy.wait(300); // Wait for sidebar animation
        }

        // Now the sidebar should be open, click logout
        cy.get('body').then($body => {
            if ($body.text().includes('Logout Profile')) {
                // Sidebar is expanded, click on the text
                cy.contains('Logout Profile').click({force: true});
            } else {
                // Fallback: try to find the logout button in the sidebar
                cy.get('[data-sidebar="sidebar"]').first().within(() => {
                    cy.get('li[data-sidebar="menu-item"]').last().within(() => {
                        cy.get('div.cursor-pointer').first().click({force: true});
                    });
                });
            }
        });
    });
});

Cypress.Commands.add('getTables', () => {
    // Ensure card view is set for consistent test behavior
    cy.window().then(win => {
        const settings = JSON.parse(win.localStorage.getItem('persist:settings') || '{}');
        settings.storageUnitView = '"card"';
        win.localStorage.setItem('persist:settings', JSON.stringify(settings));
    });

    cy.visit('/storage-unit');
    // Wait for cards to load
    cy.get('[data-testid="storage-unit-card"]', {timeout: 15000})
        .should('have.length.at.least', 1);

    return cy.get('[data-testid="storage-unit-name"]')
        .then($elements => {
            return Cypress.$.makeArray($elements).map(el => el.innerText);
        });
});

Cypress.Commands.add('addScratchpadPage', () => {
    cy.get('[data-testid="add-page-button"]').click();
});

Cypress.Commands.add('getScratchpadPages', () => {
    return cy.get('[data-testid="page-tabs"] [data-testid*="page-tab"]').then(($els) => {
        return $els.toArray().map(el => el.innerText.trim()).filter(el => el.length > 0);
    });
});

Cypress.Commands.add('deleteScratchpadPage', (index, cancel = true) => {
    // Click the delete button on the specific page tab
    cy.get(`[data-testid="delete-page-tab-${index}"]`).click();
    
    // Handle the confirmation dialog
    if (cancel) {
        cy.get('[data-testid="delete-page-button-cancel"]').click();
    } else {
        cy.get('[data-testid="delete-page-button-confirm"]').click();
    }
});

Cypress.Commands.add('dismissContextMenu', () => {
    cy.get('body').then($body => {
        const contextMenus = $body.find('[role="menu"]:visible');
        if (contextMenus.length > 0) {
            cy.get('body').click(0, 0);
            cy.wait(100);
        }
    });
});

Cypress.Commands.add('selectMockData', () => {
    // Right-click the table header to open the context menu
    // Target the table header row with cursor-context-menu class
    cy.get('table thead tr.cursor-context-menu').first().rightclick({ force: true });
    // Wait for context menu to appear
    cy.wait(200);
    // Click the "Mock Data" item using scrollIntoView and force click to handle overflow issues
    cy.get('[data-testid="context-menu-mock-data"]', {timeout: 5000})
        .scrollIntoView()
        .should('exist')
        .click({force: true});
});

// Query History Commands
Cypress.Commands.add('openQueryHistory', (index = 0) => {
    // Click the options menu (three dots) in the scratchpad cell.
    // Use .first() to ensure we only click one, even if the selector finds multiple.
    cy.get(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="icon-button"]`).first().click();

    // Click on "Query History" option, scoped to the newly visible menu to be specific.
    cy.get('[role="menu"]').should('be.visible').within(() => {
        cy.get('[role="menuitem"]').contains('Query History').click();
    });

    // Wait a bit for animations
    cy.wait(500);

    // Wait for the history dialog to open and verify the title is present.
    cy.get('[role="dialog"], .bg-background[data-state="open"]').should('be.visible');
    cy.contains('Query History').should('be.visible');
});

Cypress.Commands.add('getQueryHistoryItems', () => {
    // Returns an array of query history items with their text content.
    // Uses a reliable selector based on the component's actual HTML structure.
    return cy.get('[role="dialog"] [data-slot="card"]', { timeout: 10000 }).then($items => {
        const items = [];
        $items.each((index, item) => {
            // Extract the query text directly from the code block for accuracy.
            const queryText = Cypress.$(item).find('pre code').text().trim();
            items.push(queryText);
        });
        return items;
    });
});

Cypress.Commands.add('copyQueryFromHistory', (index = 0) => {
    // Stub the clipboard to bypass browser permission prompts.
    // The stub must return a resolved promise to prevent application errors.
    cy.window().then((win) => {
        cy.stub(win.navigator.clipboard, 'writeText').resolves().as('copy');
    });

    // Find the history item and the text to be copied
    cy.get('[role="dialog"] [data-slot="card"]').eq(index).within(($card) => {
        const textToCopy = $card.find('pre code').text().trim();
        
        // Click the copy button
        cy.get('[data-testid="copy-to-clipboard-button"]').click();

        // Verify the stub was called with the correct text. This is the
        // most reliable way to confirm the copy action in a test.
        cy.get('@copy').should('have.been.calledOnceWith', textToCopy);
    });
});

Cypress.Commands.add('cloneQueryToEditor', (historyIndex = 0, targetCellIndex = 0) => {
    // Find the history item card, store its text in an alias, and click the clone button.
    cy.get('[role="dialog"] [data-slot="card"]').eq(historyIndex).within(($card) => {
        const expectedText = $card.find('pre code').text().trim();
        cy.wrap(expectedText).as('expectedQueryText');
        cy.get('[data-testid="clone-to-editor-button"]').click();
    });

    // Wait a bit for the click handler to process
    cy.wait(500);

    // Wait for the sheet to close - it should no longer be visible
    // The sheet uses data-state attribute for open/closed state
    cy.get('body').then($body => {
        // Check if any sheet/dialog elements are still visible
        const dialogElements = $body.find('[role="dialog"]:visible');
        if (dialogElements.length > 0) {
            // If dialog still visible, wait for it to close
            cy.get('[role="dialog"]', { timeout: 10000 }).should('not.be.visible');
        }
    });

    // Use the alias to get the stored text and then assert.
    cy.get('@expectedQueryText').then(expectedText => {
        const editorSelector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${targetCellIndex}"] [data-testid="code-editor"] .cm-content`;
        cy.get(editorSelector, { timeout: 10000 }).should('contain.text', expectedText);

        // Add a small wait to ensure React state has updated after the code change
        cy.wait(500);
    });
});

Cypress.Commands.add('executeQueryFromHistory', (index = 0) => {
    // Click the run button for the specified history item
    cy.get('[role="dialog"] [data-slot="card"]').eq(index)
        .within(() => {
            cy.get('[data-testid="run-history-button"]').click();
        });

    // The dialog stays open after executing - this is the expected behavior
    // No need to wait since queries execute quickly on localhost
});

Cypress.Commands.add('closeQueryHistory', () => {
    // Close the query history dialog
    cy.get('body').type('{esc}');

    // Wait for the dialog to fully close
    cy.get('body').then($body => {
        const dialogElements = $body.find('[role="dialog"]');
        if (dialogElements.length > 0) {
            // If dialog exists, ensure it's not visible
            cy.get('[role="dialog"]').should('not.be.visible');
        }
    });

    // Ensure body no longer has scroll lock
    cy.get('body', { timeout: 5000 }).should('not.have.attr', 'data-scroll-locked');

    // Additional wait for animation completion
    cy.wait(300);
});

Cypress.Commands.add('verifyQueryInEditor', (index, expectedQuery) => {
    // Verify that the editor in the specified cell contains the expected query text
    const selector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="code-editor"] .cm-content`;
    cy.get(selector).should('contain.text', expectedQuery);
});

Cypress.Commands.add('enableAutocomplete', () => {
    // Enables the SQL autocomplete for the duration of the test
    Cypress.env('disableAutocomplete', false);
});

Cypress.Commands.add('disableAutocomplete', () => {
    // Disables the SQL autocomplete for the duration of the test
    Cypress.env('disableAutocomplete', true);
});

// ============================================================================
// Chat Commands
// ============================================================================

// Test-scoped chat response storage using Cypress.env (cleared between tests)
// Using Cypress.env instead of module-level global variable for test isolation
const CHAT_RESPONSE_KEY = '__chatMockResponses__';

/**
 * Sets up a mock AI provider for chat testing
 * This creates a comprehensive intercept that handles all GraphQL operations for chat
 * @param {Object} options - Configuration options
 * @param {string} options.modelType - The model type (e.g., 'Ollama', 'OpenAI')
 * @param {string} options.model - The specific model name (e.g., 'llama3.1')
 * @param {string} options.providerId - Optional provider ID
 */
Cypress.Commands.add('setupChatMock', ({ modelType = 'Ollama', model = 'llama3.1', providerId = 'test-provider' } = {}) => {
    // Reset chat response using test-scoped storage
    Cypress.env(CHAT_RESPONSE_KEY, null);

    // Single comprehensive intercept that handles all chat-related GraphQL operations
    // Note: The actual endpoint is /api/query, not /graphql
    cy.intercept('POST', '**/api/query', (req) => {
        const operation = req.body.operationName;
        console.log('[CYPRESS] Intercepted GraphQL operation:', operation);

        if (operation === 'GetAIProviders') {
            console.log('[CYPRESS] Handling GetAIProviders');
            req.reply({
                data: {
                    AIProviders: [{
                        Type: modelType,
                        ProviderId: providerId,
                        IsEnvironmentDefined: false
                    }]
                }
            });
            return;
        }

        if (operation === 'GetAIModels') {
            console.log('[CYPRESS] Handling GetAIModels, returning:', [model]);
            req.reply({
                data: {
                    AIModel: [model]
                }
            });
            return;
        }

        if (operation === 'GetAIChat') {
            console.log('[CYPRESS] Intercepted GetAIChat');
            console.log('[CYPRESS] Request body:', JSON.stringify(req.body, null, 2));

            // Use the stored response from test-scoped storage
            const storedResponse = Cypress.env(CHAT_RESPONSE_KEY);
            console.log('[CYPRESS] storedResponse:', JSON.stringify(storedResponse, null, 2));
            const responseData = storedResponse || [];

            if (responseData.length === 0) {
                console.warn('[CYPRESS] WARNING: No chat response configured! Sending empty array.');
            }

            const chatMessages = responseData.map(response => {
                const msg = {
                    Type: response.type || 'text',
                    Text: response.text || '',
                    __typename: 'AIChatMessage',
                    Result: response.result || null
                };

                return msg;
            });

            console.log('[CYPRESS] Mapped chat messages:', JSON.stringify(chatMessages, null, 2));

            // Clear the response BEFORE replying using test-scoped storage
            const responseCopy = [...chatMessages];
            Cypress.env(CHAT_RESPONSE_KEY, null);

            // Reply immediately with GraphQL format
            req.reply({
                statusCode: 200,
                headers: {
                    'content-type': 'application/json'
                },
                body: {
                    data: {
                        AIChat: responseCopy
                    }
                }
            });

            console.log('[CYPRESS] Response sent successfully');
            return;
        }

        // Let other GraphQL operations pass through
        req.continue();
    }).as('graphqlMock');
});

/**
 * Mocks a chat response with specific content
 * Must be called after setupChatMock
 * @param {Array<Object>} responses - Array of chat message responses
 * Each response can have:
 * - type: 'message', 'text', 'sql:get', 'sql:insert', 'sql:update', 'sql:delete', 'error', 'sql:pie-chart', 'sql:line-chart'
 * - text: The message or SQL query text
 * - result: Optional result object with Columns and Rows for SQL queries
 */
Cypress.Commands.add('mockChatResponse', (responses) => {
    // Store the responses using test-scoped storage for the intercept to use
    console.log('[CYPRESS] mockChatResponse called with:', responses);
    Cypress.env(CHAT_RESPONSE_KEY, responses);
    console.log('[CYPRESS] chatMockResponses now set to:', Cypress.env(CHAT_RESPONSE_KEY));
});

/**
 * Navigates to the chat page
 * Expects setupChatMock to be called first with providerId and model values
 */
Cypress.Commands.add('gotoChat', () => {
    cy.visit('/chat');

    // Wait for the AI provider section to be loaded
    cy.get('[data-testid="ai-provider"]', { timeout: 10000 }).should('exist');

    // Wait a bit for initial GraphQL requests to complete
    cy.wait(1000);

    // Wait for the AI provider dropdown to be visible
    cy.get('[data-testid="ai-provider-select"]', { timeout: 10000 }).should('be.visible');

    // Check the button text to determine if we need to select
    cy.get('[data-testid="ai-provider-select"]').invoke('text').then((buttonText) => {
        // If "Select Model Type" is shown, we need to click and select
        if (buttonText.includes('Select Model Type') || buttonText.trim() === '') {
            cy.log('Selecting AI provider from dropdown');

            // Click the provider dropdown to open it
            cy.get('[data-testid="ai-provider-select"]').click();

            // Wait for the dropdown options to appear
            cy.get('[role="option"]', { timeout: 5000 }).should('be.visible');

            // Select the first option (Ollama from our mock)
            cy.get('[role="option"]').first().click();

            // Wait for models to load after provider selection
            cy.wait(1500);

            // Verify provider was selected by checking button text changed
            cy.get('[data-testid="ai-provider-select"]', { timeout: 5000 })
                .invoke('text')
                .should('not.include', 'Select Model Type');
        } else {
            cy.log('AI provider already selected: ' + buttonText);
        }
    });

    // Wait for the model dropdown to be visible and enabled
    cy.get('[data-testid="ai-model-select"]', { timeout: 10000 })
        .should('be.visible')
        .should('not.be.disabled');

    // Check if model needs to be selected
    cy.get('[data-testid="ai-model-select"]').invoke('text').then((buttonText) => {
        // If "Select Model" is shown, we need to click and select
        if (buttonText.includes('Select Model') || buttonText.trim() === '') {
            cy.log('Selecting AI model from dropdown');

            // Click the model dropdown to open it
            cy.get('[data-testid="ai-model-select"]').click();

            // Wait for model options to appear
            cy.get('[role="option"]', { timeout: 5000 }).should('be.visible');

            // Select the first model option (llama3.1 from our mock)
            cy.get('[role="option"]').first().click();

            // Wait for selection to complete and state to update
            cy.wait(1000);

            // Verify model was selected by checking button text changed
            cy.get('[data-testid="ai-model-select"]', { timeout: 5000 })
                .invoke('text')
                .should('not.include', 'Select Model');
        } else {
            cy.log('AI model already selected: ' + buttonText);
        }
    });

    // Ensure chat input is enabled and ready
    cy.get('[data-testid="chat-input"]', { timeout: 10000 })
        .should('be.visible')
        .should('not.be.disabled');

    // Verify input is empty (no autofill or stale state)
    cy.get('[data-testid="chat-input"]').should('have.value', '');

    // Additional wait to ensure Redux state has fully propagated
    cy.wait(1000);
});

/**
 * Sends a chat message
 * @param {string} message - The message to send
 */
Cypress.Commands.add('sendChatMessage', (message) => {
    // Get the chat input and ensure it's ready
    cy.get('[data-testid="chat-input"]')
        .should('be.visible')
        .should('not.be.disabled');

    // Clear the input - use force:true to handle any autofill issues
    cy.get('[data-testid="chat-input"]').clear({ force: true });

    // Verify it's actually cleared
    cy.get('[data-testid="chat-input"]').should('have.value', '');

    // Type the message
    cy.get('[data-testid="chat-input"]').type(message);

    // Wait a moment for React state to update after typing
    cy.wait(300);

    // Verify the send button is enabled before clicking
    // The button becomes enabled when query.trim().length > 0 and model is selected
    cy.get('[data-testid="icon-button"]').last()
        .should('be.visible')
        .should('not.be.disabled', { timeout: 5000 })
        .click();

    // Small wait for the request to be initiated
    cy.wait(200);
});

/**
 * Verifies a user chat message appears in the conversation
 * @param {string} expectedMessage - The expected message text
 */
Cypress.Commands.add('verifyChatUserMessage', (expectedMessage) => {
    cy.get('[data-input-message="user"]').last().should('contain.text', expectedMessage);
});

/**
 * Verifies a system chat message appears in the conversation
 * @param {string} expectedMessage - The expected message text
 */
Cypress.Commands.add('verifyChatSystemMessage', (expectedMessage) => {
    cy.get('[data-input-message="system"]').last().should('contain.text', expectedMessage);
});

/**
 * Verifies a SQL query result is displayed in the chat
 * @param {Object} options - Verification options
 * @param {Array<string>} options.columns - Expected column names
 * @param {number} options.rowCount - Expected number of rows (optional)
 */
Cypress.Commands.add('verifyChatSQLResult', ({ columns, rowCount }) => {
    // Wait for the table to appear
    cy.get('table', { timeout: 10000 }).should('be.visible').last().within(() => {
        // Verify columns
        if (columns) {
            columns.forEach(column => {
                cy.get('thead th').should('contain.text', column);
            });
        }

        // Verify row count if specified
        if (rowCount !== undefined) {
            cy.get('tbody tr').should('have.length', rowCount);
        }
    });
});

/**
 * Verifies an error message appears in the chat
 * @param {string} errorText - Expected error text (can be partial, case-insensitive)
 */
Cypress.Commands.add('verifyChatError', (errorText) => {
    cy.get('[data-testid="error-state"]', { timeout: 10000 })
        .should('be.visible')
        .invoke('text')
        .should('match', new RegExp(errorText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i'));
});

/**
 * Verifies an action executed message appears
 */
Cypress.Commands.add('verifyChatActionExecuted', () => {
    cy.contains('Action Executed', { timeout: 10000 }).should('be.visible');
});

/**
 * Gets all chat messages
 * @returns {Array<Object>} Array of message objects with type and text
 */
Cypress.Commands.add('getChatMessages', () => {
    return cy.document().then(doc => {
        const messages = [];
        const messageElements = doc.querySelectorAll('[data-input-message]');
        messageElements.forEach(el => {
            messages.push({
                type: el.getAttribute('data-input-message'),
                text: el.textContent.trim()
            });
        });
        return messages;
    });
});

/**
 * Clears the chat history
 */
Cypress.Commands.add('clearChat', () => {
    cy.get('[data-testid="chat-new-chat"]').click();

    // Wait for messages to be cleared
    cy.get('[data-input-message]').should('not.exist');

    // Ensure the input field is not disabled and is ready for interaction
    cy.get('[data-testid="chat-input"]')
        .should('be.visible')
        .should('not.be.disabled')
        .should('have.value', ''); // Should be empty after clear

    // Additional wait to ensure React state has settled
    cy.wait(300);
});

/**
 * Toggles between SQL and table view in chat
 */
Cypress.Commands.add('toggleChatSQLView', () => {
    // Find the last table preview group and click its toggle button
    cy.get('.group\\/table-preview').last().within(() => {
        cy.get('[data-testid="icon-button"]').first().click({ force: true });
    });
    cy.wait(300);
});

/**
 * Verifies SQL code is displayed in the chat
 * @param {string} expectedSQL - The expected SQL query (can be partial)
 */
Cypress.Commands.add('verifyChatSQL', (expectedSQL) => {
    cy.get('[data-testid="code-editor"]').last().should('contain.text', expectedSQL);
});

/**
 * Opens the move to scratchpad dialog from the last chat result
 */
Cypress.Commands.add('openMoveToScratchpad', () => {
    cy.get('.group\\/table-preview').last().within(() => {
        cy.get('[title="Move to Scratchpad"]').click({ force: true });
    });
    cy.contains('h2', 'Move to Scratchpad', { timeout: 5000 }).should('be.visible');
});

/**
 * Confirms moving a query to scratchpad
 * @param {Object} options - Options for moving to scratchpad
 * @param {string} options.pageOption - 'new' or page ID
 * @param {string} options.newPageName - Name for new page (if pageOption is 'new')
 */
Cypress.Commands.add('confirmMoveToScratchpad', ({ pageOption = 'new', newPageName = '' } = {}) => {
    if (pageOption !== 'new') {
        // Select existing page
        cy.get('[role="dialog"]').within(() => {
            cy.get('[role="combobox"]').click();
        });
        cy.get('[role="listbox"]').within(() => {
            cy.get(`[value="${pageOption}"]`).click();
        });
    } else if (newPageName) {
        // Enter new page name (placeholder has capital letters: "Enter Page Name")
        cy.get('[role="dialog"]').within(() => {
            cy.get('input[placeholder="Enter Page Name"]').clear().type(newPageName);
        });
    }

    // Click the Move to Scratchpad button
    cy.get('[role="dialog"]').within(() => {
        cy.contains('button', 'Move to Scratchpad').click();
    });

    // Wait for navigation and verify we're on scratchpad
    cy.url({ timeout: 10000 }).should('include', '/scratchpad');
});

/**
 * Navigates chat history using arrow keys
 * @param {string} direction - 'up' or 'down'
 */
Cypress.Commands.add('navigateChatHistory', (direction = 'up') => {
    const key = direction === 'up' ? '{upArrow}' : '{downArrow}';
    cy.get('[data-testid="chat-input"]').focus().type(key);
    cy.wait(200);
});

/**
 * Gets the current value in the chat input
 * @returns {string} The current input value
 */
Cypress.Commands.add('getChatInputValue', () => {
    return cy.get('[data-testid="chat-input"]').invoke('val');
});

/**
 * Verifies the chat is empty (no messages)
 */
Cypress.Commands.add('verifyChatEmpty', () => {
    cy.get('[data-input-message]').should('not.exist');
});

/**
 * Waits for chat response to complete
 */
Cypress.Commands.add('waitForChatResponse', () => {
    // First, wait for the user message to appear (confirming send worked)
    cy.get('[data-input-message="user"]', { timeout: 5000 }).should('exist');

    // Wait for loading indicator to disappear if it appears
    cy.get('body').then($body => {
        if ($body.find('[data-testid="loading"]').length > 0) {
            cy.get('[data-testid="loading"]', { timeout: 10000 }).should('not.exist');
        }
    });

    // Wait for either a system response, SQL result table, or error state to appear
    cy.get('body', { timeout: 10000 }).should($body => {
        const hasSystemMessage = $body.find('[data-input-message="system"]').length > 0;
        const hasErrorState = $body.find('[data-testid="error-state"]').length > 0;
        // SQL results show as tables in the chat area
        const hasSQLResult = $body.find('[data-testid="chat-sql-result"] table, [data-testid="sql-result-table"]').length > 0;
        // Also check for any table that appeared after the user message (fallback)
        const hasAnyResultTable = $body.find('table').length > 0;
        expect(hasSystemMessage || hasErrorState || hasSQLResult || hasAnyResultTable,
            'Expected system message, SQL result, or error state').to.be.true;
    });

    // Additional wait for UI to fully render the response
    cy.wait(500);
});

// ============================================================================
// Screenshot Highlighting Utilities
// ============================================================================

/**
 * Highlights an element with a rounded border overlay for screenshots
 * @param {string} selector - The CSS selector or test-id of the element to highlight
 * @param {Object} options - Styling options for the highlight
 * @param {string} options.borderColor - Border color (default: '#ff0000')
 * @param {string} options.borderWidth - Border width (default: '2px')
 * @param {string} options.borderRadius - Border radius (default: '8px')
 * @param {string} options.padding - Extra padding around the element (default: '4px')
 * @param {boolean} options.shadow - Whether to add a shadow (default: false)
 * @returns {Cypress.Chainable} Chainable for further commands
 */
Cypress.Commands.add('highlightElement', (selector, {
    borderColor = '#ca6f1e',
    borderWidth = '2px',
    borderRadius = '8px',
    padding = '4px',
    shadow = true
} = {}) => {
    cy.get(selector).scrollIntoView().should('be.visible').then($el => {
        const rect = $el[0].getBoundingClientRect();
        cy.document().then(doc => {
            const overlay = doc.createElement('div');
            const paddingPx = parseInt(padding);

            overlay.style.position = 'fixed';
            overlay.style.top = `${rect.top - paddingPx}px`;
            overlay.style.left = `${rect.left - paddingPx}px`;
            overlay.style.width = `${rect.width + paddingPx * 2}px`;
            overlay.style.height = `${rect.height + paddingPx * 2}px`;
            overlay.style.border = `${borderWidth} solid ${borderColor}`;
            overlay.style.borderRadius = borderRadius;
            overlay.style.pointerEvents = 'none';
            overlay.style.zIndex = '9999';

            if (shadow) {
                overlay.style.boxShadow = `0 0 0 4px rgba(202, 111, 30, 0.1)`;
            }

            overlay.setAttribute('data-testid', 'cypress-highlight-overlay');
            doc.body.appendChild(overlay);
        });
    });
});

/**
 * Removes all highlight overlays from the page
 */
Cypress.Commands.add('removeHighlights', () => {
    cy.document().then(doc => {
        const overlays = doc.querySelectorAll('[data-testid="cypress-highlight-overlay"]');
        overlays.forEach(overlay => overlay.remove());
    });
});

/**
 * Highlights an element and takes a screenshot, then removes the highlight
 * @param {string} selector - The CSS selector or test-id of the element to highlight
 * @param {string} screenshotName - Name for the screenshot file
 * @param {Object} highlightOptions - Options for the highlight (see highlightElement)
 * @param {Object} screenshotOptions - Options for the screenshot (Cypress screenshot options)
 */
Cypress.Commands.add('screenshotWithHighlight', (selector, screenshotName, highlightOptions = {}, screenshotOptions = {}) => {
    cy.highlightElement(selector, highlightOptions);
    cy.wait(300);
    cy.screenshot(screenshotName, { overwrite: true, ...screenshotOptions });
    cy.removeHighlights();
});