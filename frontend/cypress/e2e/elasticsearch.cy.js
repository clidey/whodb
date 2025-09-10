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

describe('ElasticSearch E2E test', () => {
    const isDocker = Cypress.env('isDocker');
    const dbHost = isDocker ? 'e2e_elasticsearch' : 'localhost';
    const password = 'pgmio430fe$$#@@';
    const username = 'elastic';

    it('runs full ElasticSearch E2E flow', () => {
        // login and setup
        cy.login('ElasticSearch', dbHost, username, password);

        // 1) Lists indices
        cy.getTables().then(storageUnitNames => {
            expect(storageUnitNames).to.be.an('array');
            expect(storageUnitNames).to.deep.equal([
                "order_items",
                "orders",
                "payments",
                "products",
                "users"
            ]);
        });

        // 2) Explore users index metadata
        cy.explore("users");
        cy.wait(100);
        cy.getExploreFields().then(fields => {
            expect(fields.some(([k]) => k === "Storage Size")).to.be.true;
            expect(fields.some(([k]) => k === "Count")).to.be.true;
        });

        // 3) Data: add a document then delete and verify index data
        cy.data("users");
        cy.addRow({
            username: "new_user",
            email: "new@example.com",
            password: "newpassword"
        }, true);
        cy.data("users");
        cy.getTableData().then(({rows}) => {
            const lastRowIndex = rows.length - 1;
            cy.deleteRow(lastRowIndex);
            cy.wait(100);
        });
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal(["", "document"]);
            expect(rows.length).to.equal(3);
            const usernames = rows.map(row => JSON.parse(row[1]).username);
            expect(usernames).to.include.members(["john_doe", "jane_smith", "admin_user"]);
        });

        // 4) Respects page size pagination
        cy.setTablePageSize(1);
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(1);
        });

        // 5) Applies where condition and clears it
        cy.setTablePageSize(10);
        cy.whereTable([["username", "match", "john_doe"]]);
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(1);
            const doc = JSON.parse(rows[0][1]);
            expect(doc.username).to.equal("john_doe");
        });
        cy.clearWhereConditions();
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(3);
        });

        // 6) Edit document: save, revert, and cancel
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[1][1]);
            doc.username = "jane_smith1";
            cy.updateRow(1, 1, JSON.stringify(doc), false);
        });
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[2][1]);
            expect(doc.username).to.equal("jane_smith1");
            doc.username = "jane_smith";
            cy.updateRow(1, 1, JSON.stringify(doc), false);
        });
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[2][1]);
            expect(doc.username).to.equal("jane_smith");
        });
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[2][1]);
            doc.username = "jane_smith_temp";
            cy.updateRow(1, 1, JSON.stringify(doc));
        });
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[2][1]);
            expect(doc.username).to.equal("jane_smith");
        });

        // 7) Search highlights multiple matches in sequence
        cy.searchTable("john");
        cy.wait(100);
        cy.getHighlightedCell().first().should('contain.text', 'john');

        // 8) Graph topology and node fields
        cy.goto("graph");
        cy.getGraph().then(graph => {
            const expectedGraph = {
                "users": ["orders"],
                "orders": ["order_items", "payments", "users"],
                "order_items": ["orders", "products"],
                "products": ["order_items"],
                "payments": ["orders"]
            };
            Object.keys(expectedGraph).forEach(key => {
                expect(graph).to.have.property(key);
                expect(graph[key].sort()).to.deep.equal(expectedGraph[key].sort());
            });
        });
        cy.getGraphNode("users").then(fields => {
            expect(fields.some(([k]) => k === "Storage Size")).to.be.true;
            expect(fields.some(([k]) => k === "Count")).to.be.true;
        });
        cy.goto('graph');
        cy.get('.react-flow__node', {timeout: 10000}).should('be.visible');
        cy.get('[role="tab"]').first().click();
        cy.get('button').filter(':visible').then($buttons => {
            cy.wrap($buttons[1]).click();
        });

        cy.get('[data-testid="rf__node-users"] [data-testid="data-button"]').click({force: true});
        cy.url().should('include', '/storage-unit/explore');
        cy.contains('Total Count:').should('be.visible');
        cy.get('[data-testid="table-search"]').should('be.visible');

        // 9) Manage where conditions (edit and sheet)
        cy.data('users');
        cy.whereTable([
            ['username', 'match', 'john_doe'],
            ['email', 'match', 'john@example.com'],
        ]);
        cy.submitTable();

        cy.get('[data-testid="where-condition-badge"]').should('have.length', 2);
        cy.get('[data-testid="where-condition-badge"]').eq(0).should('contain.text', 'username match john_doe');
        cy.get('[data-testid="where-condition-badge"]').eq(1).should('contain.text', 'email match john@example.com');
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[0][1]);
            expect(doc.username).to.equal("john_doe");
        });

        cy.get('[data-testid="where-condition-badge"]').first().click();
        cy.get('[data-testid="field-value"]').clear().type('jane_smith');
        cy.get('[data-testid="cancel-button"]').click();
        cy.get('[data-testid="where-condition-badge"]').first().should('contain.text', 'username match john_doe');

        cy.get('[data-testid="where-condition-badge"]').first().click();
        cy.get('[data-testid="field-value"]').clear().type('jane_smith');
        cy.get('[data-testid="update-condition-button"]').click();
        cy.submitTable();
        cy.assertNoDataAvailable();

        cy.get('[data-testid="where-condition-badge"]').eq(0).should('contain.text', 'username match jane_smith');

        cy.get('[data-testid="remove-where-condition-button"]').eq(1).click();
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[0][1]);
            expect(doc.username).to.equal('jane_smith');
        });

        cy.get('[data-testid="remove-where-condition-button"]').first().click();

        cy.whereTable([
            ['username', 'match', 'john_doe'],
            ['email', 'match', 'john@example.com'],
            ['_id', 'exists', '1']
        ]);
        cy.get('[data-testid="more-conditions-button"]').should('be.visible').and('contain.text', '+1 more');

        cy.get('[data-testid="more-conditions-button"]').click();
        cy.get('[data-testid="sheet-field-value-0"]').clear().type('john_doe');
        cy.wait(1000);
        cy.get('[data-testid^="remove-sheet-filter-"]').then($els => {
            const count = $els.length;
            if (count > 1) {
                for (let i = count - 1; i >= 1; i--) {
                    cy.get(`[data-testid="remove-sheet-filter-${i}"]`).click();
                }
            }
        });
        cy.contains('button', 'Save Changes').click();

        cy.get('[data-testid="where-condition-badge"]').should('have.length', 1).first().should('contain.text', 'username match john_doe');
        cy.get('[data-testid="more-conditions-button"]').should('not.exist');

        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[0][1]);
            expect(doc.username).to.equal('john_doe');
        });

        cy.clearWhereConditions();
        cy.submitTable();

        // 10) Mock data on a table that does not support it
        cy.data('orders');
        cy.get('table thead tr').rightclick({force: true});
        cy.contains('div,button,span', 'Mock Data').click({force: true});

        // Wait for any toasts to clear
        cy.wait(1000);

        cy.contains('button', 'Generate').click();
        // Check for toast notification (may be partially covered but should exist)
        cy.contains('Mock data generation is not allowed for this table').should('exist');

        cy.logout();
    });
});