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
        cy.get('[data-testid="graph-layout-button"]').click();

        cy.get('[data-testid="rf__node-users"] [data-testid="data-button"]').click({force: true});
        cy.url().should('include', '/storage-unit/explore');
        cy.contains('Total Count:').should('be.visible');
        cy.get('[data-testid="table-search"]').should('be.visible');

        // 9) Manage where conditions (edit and sheet)
        cy.setWhereConditionMode('sheet');
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

            cy.getConditionCount().should('equal', 2);
            cy.verifyCondition(0, `_id eq ${firstDocId}`);
            cy.verifyCondition(1, 'username eq john_doe');
            cy.getTableData().then(({rows}) => {
                expect(JSON.parse(rows[0][1]).username).to.equal("john_doe");
            });

            // Try to edit conditions
            cy.getWhereConditionMode().then(mode => {
                if (mode === 'popover') {
                    cy.clickConditionToEdit(0);
                    cy.updateConditionValue(secondDocId);
                    cy.get('[data-testid="cancel-button"]').click();
                    cy.verifyCondition(0, `_id eq ${firstDocId}`);

                    cy.clickConditionToEdit(0);
                    cy.updateConditionValue(secondDocId);
                    cy.get('[data-testid="update-condition-button"]').click();
                    cy.submitTable();
                    cy.assertNoDataAvailable();
                    cy.verifyCondition(0, `_id eq ${secondDocId}`);
                } else {
                    cy.log('Sheet mode: Skipping inline edit tests');
                }
            });

            // Remove conditions
            cy.removeCondition(1);
            cy.submitTable();
            cy.getWhereConditionMode().then(mode => {
                cy.getTableData().then(({rows}) => {
                    if (mode === 'popover') {
                        expect(JSON.parse(rows[0][1]).username).to.equal('jane_smith');
                    } else {
                        // In sheet mode, conditions weren't changed
                        expect(JSON.parse(rows[0][1]).username).to.equal('john_doe');
                    }
                });
            });

            // Clear ALL conditions before proceeding to avoid duplicates
            cy.clearWhereConditions();
            cy.wait(500); // Wait for conditions to clear

            // Verify all conditions are cleared
            cy.getConditionCount().should('equal', 0);

            // Add conditions for testing the more button (3 conditions)
            cy.whereTable([
                ['_id', 'eq', thirdDocId],
                ['username', 'eq', 'admin_user'],
                ['email', 'ne', 'jane@example.com']
            ]);

            // With MAX_VISIBLE_CONDITIONS = 2, we should see 2 badges and "+1 more"
            cy.getWhereConditionMode().then(mode => {
                if (mode === 'popover') {
                    // Should show first 2 conditions as badges
                    cy.getConditionCount().should('equal', 3);
                    cy.verifyCondition(0, `_id eq ${thirdDocId}`);
                    cy.verifyCondition(1, 'username eq admin_user');

                    // Check for more conditions button
                    cy.checkMoreConditionsButton('+1 more');

                    // Click to open sheet with all conditions
                    cy.clickMoreConditions();

                    // In the sheet, keep only the first condition
                    cy.removeConditionsInSheet(true);
                    cy.saveSheetChanges();

                    // After closing sheet, should have only 1 condition
                    cy.getConditionCount().should('equal', 1);
                    cy.verifyCondition(0, `_id eq ${thirdDocId}`);
                } else {
                    // In sheet mode, just verify count
                    cy.getConditionCount().should('equal', 3);
                    cy.log('Sheet mode: All 3 conditions active');
                }
            });

            cy.submitTable();
            cy.getTableData().then(({rows}) => {
                cy.getWhereConditionMode().then(mode => {
                    if (mode === 'popover') {
                        // After removing 2 conditions, only _id=thirdDocId remains
                        expect(JSON.parse(rows[0][1]).username).to.equal('admin_user');
                    } else {
                        // In sheet mode, all 3 conditions are active
                        expect(JSON.parse(rows[0][1]).username).to.equal('admin_user');
                    }
                });
            });

            cy.clearWhereConditions();
            cy.submitTable();
        });

        // 10) Mock data on a table that does not support it
        cy.data('orders');
        cy.selectMockData();
        // Wait for any toasts to clear
        cy.contains('button', 'Generate').click();
        cy.contains('Mock data generation is not allowed for this table').should('exist');

        cy.logout();
    });
});
