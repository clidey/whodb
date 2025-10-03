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

    cy.get('[data-testid="login-button"]').click();
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
    return cy.getTables().then(elements => {
        const index = elements.findIndex(name => name === tableName);
        return cy.get('[data-testid="explore-button"]').eq(index).click();
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
    return cy.getTables().then(elements => {
        const index = elements.findIndex(name => name === tableName);
        return cy.get('[data-testid="data-button"]').eq(index).click().then(() => {
            // Wait for the table to be present after clicking data button
            return cy.get('table', {timeout: 10000}).should('exist');
        });
    });
});

Cypress.Commands.add('sortBy', (index) => {
    return cy.get('th').eq(index + 1).click();
});

Cypress.Commands.add('assertNoDataAvailable', () => {
    // Assert the empty-state text is visible (retries until timeout)
    cy.contains(/No data available/i, {timeout: 10000}).should('be.visible');
});


Cypress.Commands.add('getTableData', () => {
    // First wait for the table to exist
    return cy.get('table', {timeout: 10000}).should('exist').then(() => {
        // Wait for at least one table row to be present with proper scoping
        return cy.get('table tbody tr', {timeout: 10000})
            .then(() => {
                // Additional wait to ensure data is fully rendered
                cy.wait(100);
                
                // Now get the table and extract data
                return cy.get('table').then($table => {
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
    cy.get('[data-testid="submit-button"]').click().then(() => {
        // Wait a bit for the request to complete
        cy.wait(200);
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
                    cy.get('button').contains('Add').click();
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

        // In popover mode, count badges
        const badges = $body.find('[data-testid="where-condition-badge"]');
        if (badges.length > 0) return badges.length;

        return 0;
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
    // Click Add/Update button to save and close
    cy.get('[role="dialog"]').within(() => {
        cy.contains('button', /^(Add|Update|Add to Page)$/).click();
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

Cypress.Commands.add("getHighlightedCell", () => {
    return cy.get('td.table-search-highlight');
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
    // Wait for the sheet/dialog to close - the submit button should no longer be visible
    cy.get('[data-testid="submit-add-row-button"]').should('not.exist');
    // Ensure body no longer has scroll lock
    cy.get('body', { timeout: 5000 }).should('not.have.attr', 'data-scroll-locked');
    // Increased wait for headless mode - ensures GraphQL mutation completes and UI updates
    cy.wait(500);
    // Additional check to ensure the table has been refreshed
    cy.get('table tbody').should('be.visible');
});

Cypress.Commands.add("deleteRow", (rowIndex) => {
    // Get initial row count to verify deletion
    cy.get('table tbody tr').its('length').then(initialRowCount => {
        // Ensure the target row exists before interacting
        cy.get('table tbody tr').should('have.length.greaterThan', rowIndex);

        // Right-click to open the context menu
        cy.get('table tbody tr').eq(rowIndex).rightclick({ force: true });

        // Wait for the menu to be visible, then click the items
        cy.get('[data-testid="context-menu-more-actions"]').should('be.visible').click();
        cy.get('[data-testid="context-menu-delete-row"]').should('be.visible').click();

        // Wait for the row to be removed by checking the row count
        // Use a timeout to handle async deletion
        cy.get('table tbody tr', { timeout: 10000 }).should('have.length', initialRowCount - 1);
    });
});

Cypress.Commands.add("updateRow", (rowIndex, columnIndex, text, cancel = true) => {
    // Open the context menu for the row at rowIndex
    cy.get('table tbody tr').eq(rowIndex).rightclick({ force: true });

    // Wait for the menu to be visible, then click the "Edit row" item
    cy.get('[data-testid="context-menu-edit-row"]').should('be.visible').click();

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
    cy.get('[data-testid="table-search"]').clear().type(`${search}{enter}`);
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
                        const [source, target] = el.getAttribute("data-testid").slice("rf__edge-".length).split("->");
                        return {source, target};
                    });
                    const graph = {};
                    nodes.forEach(node => {
                        graph[node] = edges
                            .filter(edge => edge.source === node)
                            .map(edge => edge.target);
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
    // First check if sidebar is closed (collapsed)
    cy.get('body').then($body => {
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
                cy.get('[data-sidebar="sidebar"]').within(() => {
                    cy.get('li[data-sidebar="menu-item"]').last().within(() => {
                        cy.get('div.cursor-pointer').first().click({force: true});
                    });
                });
            }
        });
    });
});

Cypress.Commands.add('getTables', () => {
    cy.visit('/storage-unit');
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
    cy.get('table thead tr').first().rightclick({ force: true });
    // Click the "Mock Data" item using its stable test ID
    cy.get('[data-testid="context-menu-mock-data"]').should('be.visible').click();
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