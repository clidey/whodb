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

describe('MongoDB E2E test', () => {
    const isDocker = Cypress.env('isDocker');
    const dbHost = isDocker ? 'e2e_mongo' : 'localhost';
    const dbUser = 'user';
    const dbPassword = 'password';
    const dbName = 'test_db';


    it('runs full MongoDB E2E flow', () => {
        // login and setup
        cy.login('MongoDB', dbHost, dbUser, dbPassword, dbName);

        // 1) Lists collections
        cy.getTables().then(storageUnitNames => {
            expect(storageUnitNames).to.be.an('array');
            expect(storageUnitNames).to.deep.equal([
                "order_items",
                "order_summary",
                "orders",
                "payments",
                "products",
                "system.views",
                "users"
            ]);
        });

        // 2) Explore users collection metadata
        cy.explore("users");
        cy.getExploreFields().then(fields => {
            const arr = Array.isArray(fields) ? fields : (typeof fields === "string" ? fields.split("\n").map(line => {
                const idx = line.indexOf(": ");
                if (idx === -1) return [line, ""];
                return [line.slice(0, idx), line.slice(idx + 2)];
            }) : []);
            expect(arr.some(([k, v]) => k === "Type" && v === "Collection")).to.be.true;
            expect(arr.some(([k]) => k === "Storage Size")).to.be.true;
            expect(arr.some(([k]) => k === "Count")).to.be.true;
        });

        // 3) Data: verify collection data
        cy.data("users");
        cy.sortBy(0);
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal(["", "document"]);
            const expectedRows = [
                {email: "john@example.com", password: "securepassword1", username: "john_doe"},
                {email: "jane@example.com", password: "securepassword2", username: "jane_smith"},
                {email: "admin@example.com", password: "adminpass", username: "admin_user"}
            ];
            rows.forEach((row, idx) => {
                const json = JSON.parse(row[1]);
                expect(json.email).to.equal(expectedRows[idx].email);
                expect(json.password).to.equal(expectedRows[idx].password);
                expect(json.username).to.equal(expectedRows[idx].username);
            });
        });

        // 4) Respects page size pagination
        cy.setTablePageSize(1);
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(1);
        });

        // 5) Applies where condition and clears it
        cy.setTablePageSize(10);
        cy.getTableData().then(({rows}) => {
            const firstDocId = JSON.parse(rows[0][1])._id;
            cy.whereTable([["_id", "eq", firstDocId]]);
            cy.submitTable();
            cy.getTableData().then(({rows: filteredRows}) => {
                expect(filteredRows.length).to.equal(1);
                expect(JSON.parse(filteredRows[0][1])._id).to.equal(firstDocId);
            });
            cy.clearWhereConditions();
            cy.submitTable();
            cy.getTableData().then(({rows: clearedRows}) => {
                expect(clearedRows.length).to.equal(3);
            });
        });


        // 6) Edit document: save, revert, and cancel
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[1][1]);
            doc.username = "jane_smith1";
            cy.updateRow(1, 1, JSON.stringify(doc), false);
        });
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[1][1]);
            expect(doc.username).to.equal("jane_smith1");
            doc.username = "jane_smith";
            cy.updateRow(1, 1, JSON.stringify(doc), false);
        });
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[1][1]);
            expect(doc.username).to.equal("jane_smith");
        });
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[1][1]);
            doc.username = "jane_smit_temph";
            cy.updateRow(1, 1, JSON.stringify(doc));
        });
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[1][1]);
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
                "orders": ["order_items", "payments"],
                "order_items": [],
                "products": ["order_items"],
                "payments": [],
                "order_summary": []
            };
            Object.keys(expectedGraph).forEach(key => {
                expect(graph).to.have.property(key);
                expect(graph[key].sort()).to.deep.equal(expectedGraph[key].sort());
            });
        });
        cy.getGraphNode("users").then(fields => {
            expect(fields.some(([k, v]) => k === "Type" && v === "Collection")).to.be.true;
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
        cy.getTableData().then(({rows}) => {
            const firstDocId = JSON.parse(rows[0][1])._id;
            const secondDocId = JSON.parse(rows[1][1])._id;
            const thirdDocId = JSON.parse(rows[2][1])._id;

            cy.whereTable([
                ['_id', 'eq', firstDocId],
                ['username', 'eq', 'john_doe'],
            ]);
            cy.submitTable();

            cy.get('[data-testid="where-condition-badge"]').should('have.length', 2);
            cy.get('[data-testid="where-condition-badge"]').eq(0).should('contain.text', `_id eq ${firstDocId}`);
            cy.get('[data-testid="where-condition-badge"]').eq(1).should('contain.text', 'username eq john_doe');
            cy.getTableData().then(({rows}) => {
                expect(JSON.parse(rows[0][1]).username).to.equal("john_doe");
            });

            cy.get('[data-testid="where-condition-badge"]').first().click();
            cy.get('[data-testid="field-value"]').clear().type(secondDocId);
            cy.get('[data-testid="cancel-button"]').click();
            cy.get('[data-testid="where-condition-badge"]').first().should('contain.text', `_id eq ${firstDocId}`);

            cy.get('[data-testid="where-condition-badge"]').first().click();
            cy.get('[data-testid="field-value"]').clear().type(secondDocId);
            cy.get('[data-testid="update-condition-button"]').click();
            cy.submitTable();
            cy.assertNoDataAvailable();

            cy.get('[data-testid="where-condition-badge"]').eq(0).should('contain.text', `_id eq ${secondDocId}`);

            cy.get('[data-testid="remove-where-condition-button"]').eq(1).click();
            cy.submitTable();
            cy.getTableData().then(({rows}) => {
                expect(JSON.parse(rows[0][1]).username).to.equal('jane_smith');
            });

            cy.get('[data-testid="remove-where-condition-button"]').first().click();

            cy.whereTable([
                ['_id', 'eq', firstDocId],
                ['username', 'eq', 'john_doe'],
                ['email', 'ne', 'jane@example.com']
            ]);
            cy.get('[data-testid="more-conditions-button"]').should('be.visible').and('contain.text', '+1 more');

            cy.get('[data-testid="more-conditions-button"]').click();
            cy.get('[data-testid="sheet-field-value-0"]').clear().type(thirdDocId);
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

            cy.get('[data-testid="where-condition-badge"]').should('have.length', 1).first().should('contain.text', `_id eq ${thirdDocId}`);
            cy.get('[data-testid="more-conditions-button"]').should('not.exist');

            cy.submitTable();
            cy.getTableData().then(({rows}) => {
                expect(JSON.parse(rows[0][1]).username).to.equal('admin_user');
            });

            cy.clearWhereConditions();
            cy.submitTable();
        });

        // 10) Mock data on a table that does not support it
        cy.data('orders');
        cy.get('table thead tr').eq(0).rightclick({force: true});
        cy.contains('div,button,span', 'Mock Data').click({force: true});
        // Wait for any toasts to clear
        cy.wait(1000);
        cy.contains('button', 'Generate').click();
        cy.contains('Mock data generation is not allowed for this table').should('exist');

        cy.logout();
    });
});
