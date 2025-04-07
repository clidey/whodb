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
    if (databaseType) cy.get('[data-testid="database-type"]').trigger("mouseover").get(`[value="${databaseType}"]`).click();
    if (hostname) cy.get('[data-testid="hostname"] input').clear().type(hostname);
    if (username) cy.get('[data-testid="username"] input').clear().type(username);
    if (password) cy.get('[data-testid="password"] input').clear().type(password, { log: false });
    if (databaseType !== "Sqlite3" && database) cy.get('[data-testid="database"] input').clear().type(database);
    if (databaseType === "Sqlite3" && database) cy.get('[data-testid="database"]').trigger("mouseover").get(`[value="${database}"]`).click();

    if (Object.keys(advanced).length > 0) {
        cy.get('[data-testid="advanced-button"]').click();
        for (const [key, value] of Object.entries(advanced)) {
            cy.get(`[data-testid="${key}-input"] input`).clear().type(value);
        }
    }

    cy.get('[data-testid="submit-button"]').click();
});

Cypress.Commands.add('setAdvanced', (type, value) => {
});

Cypress.Commands.add("selectDatabase", (value) => {
    cy.get('[data-testid="sidebar-database"]').trigger("mouseover").get(`[value="${value}"]`).click();
});

Cypress.Commands.add("selectSchema", (value) => {
    cy.get('[data-testid="sidebar-schema"]').trigger("mouseover").get(`[value="${value}"]`).click();
});

Cypress.Commands.add('explore', (tableName) => {
    return cy.getTables().then(elements => {
        const index = elements.findIndex(name => name === tableName);
        return cy.get('[data-testid="explore-button"]').eq(index).click();
    });
});

Cypress.Commands.add('getExploreFields', () => {
    return extractText(cy.get('[data-testid="explore-fields"]'));
});

Cypress.Commands.add('data', (tableName) => {
    return cy.getTables().then(elements => {
        const index = elements.findIndex(name => name === tableName);
        return cy.get('[data-testid="data-button"]').eq(index).click();
    });
});

Cypress.Commands.add('sortBy', (index) => {
    return cy.get('[data-testid="table-header"]').eq(index+1).click();
});

Cypress.Commands.add('getTableData', () => {
    return cy.get('[data-testid="table"]').then($table => {
        const columns = Cypress.$.makeArray($table.find('[data-testid="table-header"]'))
            .map(el => el.innerText.trim());

        const rows = Cypress.$.makeArray($table.find('[data-testid="table-row"]')).map(row => {
            return Cypress.$.makeArray(Cypress.$(row).find('[data-testid="table-row-data"] .cell-data'))
                .map(cell => cell.innerText.trim());
        });

        return { columns, rows };
    });
});

Cypress.Commands.add("setTablePageSize", (pageSize) => {
    cy.get('[data-testid="table-page-size"] input').clear().type(pageSize);
});

Cypress.Commands.add("submitTable", (pageSize) => {
    cy.get('[data-testid="submit-button"]').click();
});

Cypress.Commands.add("whereTable", (fieldArray) => {
    cy.get('[data-testid="where-button"]').click();
    for (const [key, operator, value] of fieldArray) {
        cy.get('[data-testid="field-key"]').trigger("mouseover").get(`[value="${key}"]`).click();
        cy.get('[data-testid="field-operator"]').trigger("mouseover").get(`[value="${operator}"]`).click();
        cy.get('[data-testid="field-value"]').clear().type(value);
        cy.get('[data-testid="add-button"]').click();
    }
    cy.get('[data-testid="cancel-button"]').click();
});

Cypress.Commands.add("clearWhereConditions", () => {
    return cy.get('[data-testid="where-condition"]').each(($el) => {
        return cy.wrap($el)
            .scrollIntoView()
            .trigger("mouseover")
            .within(() => {
                cy.get('[data-testid="remove-where-condition-button"] button')
                    .click({ force: true });
            });
    });
});

Cypress.Commands.add("getHighlightedRows", () => {
    return cy.get('[data-testid="table"]').then($table => {
        return Cypress.$.makeArray($table.find('[class*="bg-yellow-100"][data-testid="table-row"]')).map(row => {
            return Cypress.$.makeArray(Cypress.$(row).find('[data-testid="table-row-data"] .cell-data'))
                .map(cell => cell.innerText.trim());
        });
    });
});

Cypress.Commands.add("updateRow", (rowIndex, columnIndex, text, cancel = true) => {
    cy.get('[data-testid="table"] [data-testid="table-row"]')
      .eq(rowIndex)
      .find('[data-testid="table-row-data"]') 
      .eq(columnIndex)
      .find('[data-testid="edit-button"]')
      .click({ force: true });

    cy.get('[data-testid="edit-dialog"] [data-testid="code-editor"]')
        .then(($editor) => {
            const editorElement = $editor[0].querySelector('.cm-content');
            if (editorElement) {
                editorElement.textContent = text;
                editorElement.dispatchEvent(new Event('input', { bubbles: true }));
            } else {
                throw new Error("Editor not found!");
            }
        });
    
    if (cancel) {
        return cy.get('[data-testid="cancel-update-button"]').click();
    }
    cy.get('[data-testid="update-button"]').click();
});


Cypress.Commands.add("getPageNumbers", () => {
    return cy.get('[data-testid="table-page-number"]').then(($els) => {
        return $els.toArray().map(el => el.innerText.trim());
    });
});

Cypress.Commands.add("searchTable", (search) => {
    cy.get('[data-testid="table-search"] input').clear().type(`${search}{enter}`);
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

Cypress.Commands.add("getGraphNode", () => {
    return extractText(cy.get('.react-flow__node[data-id="users"]'));
});

Cypress.Commands.add("addCell", (afterIndex) => {
    return cy.get(`[data-testid="cell-${afterIndex}"] [data-testid="add-button"]`).click();
});

Cypress.Commands.add("removeCell", (index) => {
    return cy.get(`[data-testid="cell-${index}"] [data-testid="delete-button"]`).click();
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
    return cy.get(`[data-testid="cell-${index}"] [data-testid="submit-button"]`).click();
});

Cypress.Commands.add("getCellQueryOutput", (index) => {
    return cy.get(`[data-testid="cell-${index}"] [data-testid="cell-query-output"] [data-testid="table"]`).then($table => {
        const columns = Cypress.$.makeArray($table.find('[data-testid="table-header"]'))
            .map(el => el.innerText.trim());

        const rows = Cypress.$.makeArray($table.find('[data-testid="table-row"]')).map(row => {
            return Cypress.$.makeArray(Cypress.$(row).find('[data-testid="table-row-data"] .cell-data'))
                .map(cell => cell.innerText.trim());
        });
        return { columns, rows };
    });
});

Cypress.Commands.add("getCellActionOutput", (index) => {
    return extractText(cy.get(`[data-testid="cell-${index}"] [data-testid="cell-action-output"]`));
});

Cypress.Commands.add("getCellError", (index) => {
    return extractText(cy.get(`[data-testid="cell-${index}"] [data-testid="cell-error"]`));
});

Cypress.Commands.add('logout', () => {
    cy.get('[data-testid="logout"] [data-testid="sidebar-button"]').click();
});

Cypress.Commands.add('getTables', () => {
    cy.visit('http://localhost:3000/storage-unit');
    return cy.get('[data-testid="storage-unit-name"]')
        .then($elements => {
            return Cypress.$.makeArray($elements).map(el => el.innerText);
        });
});