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
    cy.visit(`http://localhost:3000/${route}`);
});

Cypress.Commands.add('login', (databaseType, hostname, username, password, database, advanced={}) => {
    cy.visit('http://localhost:3000/login');
    if (databaseType) cy.get('[data-testid="database-type-select"]').click().get(`[data-value="${databaseType}"]`).click();
    if (hostname) cy.get('[data-testid="hostname"]').clear().type(hostname);
    if (username) cy.get('[data-testid="username"]').clear().type(username);
    if (password) cy.get('[data-testid="password"]').clear().type(password, { log: false });
    if (databaseType !== "Sqlite3" && database) cy.get('[data-testid="database"]').clear().type(database);
    if (databaseType === "Sqlite3" && database) cy.get('[data-testid="database"]').click().get(`[data-value="${database}"]`).click();

    if (Object.keys(advanced).length > 0) {
        cy.get('[data-testid="advanced-button"]').click();
        for (const [key, value] of Object.entries(advanced)) {
            cy.get(`[data-testid="${key}-input"] input`).clear().type(value);
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
            return cy.get('table', { timeout: 10000 }).should('exist');
        });
    });
});

Cypress.Commands.add('sortBy', (index) => {
    return cy.get('th').eq(index+1).click();
});

Cypress.Commands.add('getTableData', () => {
    // First wait for the table to exist
    return cy.get('table', { timeout: 10000 }).should('exist').then(() => {
        // Wait for at least one table row to be present with proper scoping
        return cy.get('table tbody tr', { timeout: 10000 })
            .should('have.length.greaterThan', 0)
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

Cypress.Commands.add("submitTable", (pageSize) => {
    cy.get('[data-testid="submit-button"]').click().then(() => {
        // Wait a bit for the request to complete
        cy.wait(500);
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

Cypress.Commands.add("addRow", (data) => {
    cy.get('[data-testid="add-row-button"]').click();

    for (const [key, value] of Object.entries(data)) {
        cy.get(`[data-testid="add-row-field-${key}"] input`).clear().type(value);
    }

    cy.get('[data-testid="submit-add-row-button"]').click();
});

Cypress.Commands.add("deleteRow", (rowIndex) => {
    cy.get('table tbody tr')
      .eq(rowIndex)
      .rightclick();
    cy.get('[data-testid="context-menu-more-actions"]').click();
    cy.get('[data-testid="context-menu-delete-row"]').click();
});

Cypress.Commands.add("updateRow", (rowIndex, columnIndex, text, cancel = true) => {
    // Open the context menu for the row at rowIndex
    cy.get('table tbody tr')
      .eq(rowIndex)
      .rightclick();

    // Click the "Edit row" context menu item
    cy.get('[data-testid="context-menu-edit-row"]').click();

    // Wait for the editable row to appear, using the row index in the test id
    cy.get(`[data-testid="editable-field-${columnIndex}"]`).should('exist');

    // Find the correct input for the column using the row and column index
    cy.get(`[data-testid="editable-field-${columnIndex}"] input`)
        .should('exist')
        .clear()
        .type(text, { force: true });

    // Click cancel or update as requested
    if (cancel) {
        return cy.get('[role="dialog"] > button').click();
    }
    cy.get(`[data-testid="update-button"]`).click();
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
    return cy.get('.react-flow__node').then(($nodeEls) => {
        const nodes = $nodeEls.toArray().map(el => el.getAttribute("data-id"));

        return cy.get('.react-flow__edge').then(($edgeEls) => {            
            const edges = $edgeEls.toArray().map(el => {
                const [source, target] = el.getAttribute("data-testid").slice("rf__edge-".length).split("->");
                return { source, target };
            });
            const graph = {};
            nodes.forEach(node => {
                graph[node] = edges
                    .filter(edge => edge.source === node)
                    .map(edge => edge.target);
            });

            return graph;
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
    cy.get(`[data-testid="cell-${index}"] [data-testid="code-editor"]`)
        .should('exist') // Wait for the code editor to exist in the DOM
        .should('be.visible') // Ensure it is visible before interacting
        .then(($editor) => {
            $editor.click();
            const editorElement = $editor[0].querySelector('.cm-content');
            if (editorElement) {
                editorElement.textContent = text;
                editorElement.dispatchEvent(new Event('input', { bubbles: true }));
            } else {
                throw new Error("Editor not found!");
            }
        });
});

Cypress.Commands.add("runCode", (index) => {
    return cy.get(`[data-testid="cell-${index}"] [data-testid="query-cell-button"]`).click();
});

Cypress.Commands.add("getCellQueryOutput", (index) => {
    return cy.get(`[data-testid="cell-${index}"] [data-testid="cell-query-output"] table`, { timeout: 10000 }).then($table => {
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
    return cy.get(`[data-testid="cell-${index}"] [data-testid="cell-action-output"]`, { timeout: 10000 })
        .should('exist')
        .then($el => extractText(cy.wrap($el)));
});

Cypress.Commands.add("getCellError", (index) => {
    // Wait for error element to appear after query execution
    // The error is in an Alert component, we need to extract just the AlertDescription text
    return cy.get(`[data-testid="cell-${index}"] [data-testid="cell-error"]`, { timeout: 10000 })
        .should('exist')
        .find('[role="alert"]') // Find the Alert component
        .invoke("text")
        .then((text) => {
            // The Alert contains both AlertTitle ("Error") and AlertDescription (actual error message)
            // Remove the "Error" prefix if it exists
            return text.replace(/^Error\s*/i, '').trim();
        });
});

Cypress.Commands.add('logout', () => {
    cy.get('[data-testid="logout"]').click();
});

Cypress.Commands.add('getTables', () => {
    cy.visit('http://localhost:3000/storage-unit');
    return cy.get('[data-testid="storage-unit-name"]')
        .then($elements => {
            return Cypress.$.makeArray($elements).map(el => el.innerText);
        });
});

Cypress.Commands.add('addScratchadPage', () => {
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