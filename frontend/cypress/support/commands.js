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
    if (databaseType) cy.get('[data-testid="database-type-select"]').click().get(`[data-value="${databaseType}"]`).click();
    if (hostname) cy.get('[data-testid="hostname"]').clear().type(hostname);
    if (username) cy.get('[data-testid="username"]').clear().type(username);
    if (password) cy.get('[data-testid="password"]').clear().type(password, {log: false});
    if (databaseType !== "Sqlite3" && database) cy.get('[data-testid="database"]').clear().type(database);
    if (databaseType === "Sqlite3" && database) cy.get('[data-testid="database"]').click().get(`[data-value="${database}"]`).click();

    if (Object.keys(advanced).length > 0) {
        cy.get('[data-testid="advanced-button"]').click();
        for (const [key, value] of Object.entries(advanced)) {
            cy.get(`[data-testid="${key}-input"]`).clear().type(value);
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
    cy.get('[data-testid="where-button"]').click()
    cy.wait(100);
    for (const [key, operator, value] of fieldArray) {
        cy.log(key, operator, value);
        cy.get('[data-testid="field-key"]').click()
        cy.get(`[data-value="${key}"]`).click();
        cy.get('[data-testid="field-operator"]').click();
        cy.get(`[data-value="${operator}"]`).click();
        cy.get('[data-testid="field-value"]').clear().type(value);
        cy.wait(100);
        cy.get('[data-testid="add-condition-button"]').click();
    }
    cy.get('[data-testid="cancel-button"]').click();
});

Cypress.Commands.add("clearWhereConditions", () => {
    return cy.get('[data-testid="where-condition"]').each(($el) => {
        return cy.wrap($el)
            .scrollIntoView()
            .click()
            .within(() => {
                cy.get('[data-testid="remove-where-condition-button"]')
                    .click({ force: true });
            });
    });
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
    // Additional wait to ensure animations complete
    cy.wait(100);
});

Cypress.Commands.add("deleteRow", (rowIndex) => {
    // Wait a moment for any previous operations to complete
    cy.wait(100);

    // First check how many rows exist and ensure the target row exists
    cy.get('table tbody tr').should('have.length.greaterThan', rowIndex).then(() => {
        cy.get('table tbody tr')
            .eq(rowIndex)
            .rightclick({force: true});
        cy.get('[data-testid="context-menu-more-actions"]').click({force: true});
        cy.get('[data-testid="context-menu-delete-row"]').click({force: true});
        // Wait for the delete to process
        cy.wait(100);
    });
});

Cypress.Commands.add("updateRow", (rowIndex, columnIndex, text, cancel = true) => {
    // Wait a moment for any previous operations to complete
    cy.wait(100);
    // Open the context menu for the row at rowIndex, use force since dialogs might be animating
    cy.get('table tbody tr')
      .eq(rowIndex)
        .rightclick({force: true});

    // Click the "Edit row" context menu item
    cy.get('[data-testid="context-menu-edit-row"]').click({force: true});

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
        // Force wait since the sheet might have animation
        cy.wait(100);
    } else {
        cy.get(`[data-testid="update-button"]`).click();
        // Wait for the update to process and sheet to close
        cy.wait(100);
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
    return cy.get(`[data-testid="cell-${afterIndex}"] [data-testid="add-cell-button"]`).click();
});

Cypress.Commands.add("removeCell", (index) => {
    return cy.get(`[data-testid="cell-${index}"] [data-testid="delete-cell-button"]`).click();
});

Cypress.Commands.add("writeCode", (index, text) => {
    // Focus the editor
    cy.get('[data-testid="cell-' + index + '"] [data-testid="code-editor"]')
        .should('exist')
        .should('be.visible')
        .click();

    cy.wait(100);

    // Clear content
    cy.get('[data-testid="cell-' + index + '"] .cm-content')
        .type('{selectall}{backspace}', {delay: 0});

    // Paste the text directly to avoid intellisense
    cy.window().then(win => {
        const textarea = win.document.querySelector('[data-testid="cell-' + index + '"] .cm-content');
        if (textarea) {
            textarea.textContent = '';
            // Simulate paste event
            const pasteEvent = new win.ClipboardEvent('paste', {
                bubbles: true,
                cancelable: true,
                clipboardData: new DataTransfer()
            });
            pasteEvent.clipboardData.setData('text/plain', text);
            textarea.dispatchEvent(pasteEvent);
        }
    });

    // Blur to trigger state update
    cy.get('[data-testid="cell-' + index + '"] .cm-content').blur();

    // Re-focus
    cy.get('[data-testid="cell-' + index + '"] [data-testid="code-editor"]').click();

    cy.wait(100);
});

Cypress.Commands.add("runCode", (index) => {
    // Try the main run button first
    cy.get(`[data-testid="cell-${index}"] [data-testid="query-cell-button"]`)
        .should('exist')
        .then($button => {
            if ($button.is(':visible') && !$button.is(':disabled')) {
                cy.wrap($button).click();
            } else {
                // If main button not available, try the play button in the gutter
                cy.get(`[data-testid="cell-${index}"] .cm-play-button`)
                    .first()
                    .click();
            }
        });

    // Wait for the query to execute
    cy.wait(100);
});

Cypress.Commands.add("getCellQueryOutput", (index) => {
    return cy.get(`[data-testid="cell-${index}"] [data-testid="cell-query-output"] table`, {timeout: 10000}).then($table => {
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
    return cy.get(`[data-testid="cell-${index}"] [data-testid="cell-action-output"]`, {timeout: 10000})
        .should('exist')
        .then($el => extractText(cy.wrap($el)));
});

Cypress.Commands.add("getCellError", (index) => {
    // Wait for the error element to appear
    return cy.get(`[data-testid="cell-${index}"] [data-testid="cell-error"]`, {timeout: 10000})
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
        return $els.toArray().map(el => el.innerText.trim());
    });
});

Cypress.Commands.add('deleteScratchpadPage', (index, cancel = true) => {
    cy.get(`[data-testid="page-tab-${index}"]`).click();
    cy.get('[data-testid="delete-page-button"]').click();
    if (cancel) {
        cy.get('[data-testid="delete-page-button-cancel"]').click();
    } else {
        cy.get('[data-testid="delete-page-button-confirm"]').click();
    }
});