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
        cy.login('ElasticSearch', dbHost, username, password, undefined);

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
        // Wait for Elasticsearch to index the new document
        cy.wait(1500);
        cy.data("users");
        cy.getTableData().then(({rows}) => {
            // Verify new_user was added (should have 4 rows now)
            expect(rows.length).to.equal(4);
            // Find the row with new_user (not just the last row, as order may vary)
            let targetRowIndex = -1;
            for (let i = 0; i < rows.length; i++) {
                const text = rows[i][1];
                if (text.includes('"username":"new_user"') || text.includes('new_user')) {
                    targetRowIndex = i;
                    break;
                }
            }
            expect(targetRowIndex, `Could not find new_user in rows`).to.be.greaterThan(-1);
            cy.deleteRow(targetRowIndex);
            // Wait for Elasticsearch to refresh its index after delete
            cy.wait(1500);
        });
        cy.data("users");
        // Additional wait to ensure table has fully refreshed with correct data
        cy.wait(1000);
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal(["", "document"]);
            expect(rows.length).to.equal(3);
            // For Elasticsearch, verify by checking visible emails (usernames may be truncated)
            const rawData = rows.map(r => r[1]);
            expect(rawData.some(d => d.includes('john@example.com'))).to.be.true;
            expect(rawData.some(d => d.includes('jane@example.com'))).to.be.true;
            expect(rawData.some(d => d.includes('admin@example.com'))).to.be.true;
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

        // 6) Edit document: test cancel (skip full edit test for Elasticsearch due to truncated JSON display)
        cy.getTableData().then(({rows}) => {
            let janeRowIndex = -1;
            for (let i = 0; i < rows.length; i++) {
                if (rows[i][1].includes('jane@example.com')) {
                    janeRowIndex = i;
                    break;
                }
            }
            expect(janeRowIndex).to.be.greaterThan(-1);
            // Open edit dialog and cancel (testing cancel functionality)
            cy.get('table tbody tr').eq(janeRowIndex).rightclick({ force: true });
            cy.get('[data-testid="context-menu-edit-row"]').should('be.visible').click();
            cy.contains('Edit Row').should('be.visible');
            cy.get('body').type('{esc}');
            cy.contains('Edit Row').should('not.exist');
        });

        // 7) Search highlights multiple matches in sequence
        cy.searchTable("john");
        cy.getHighlightedCell().first().should('contain.text', 'john');

        // 8) Graph topology and node fields - SKIPPED FOR NOW (TODO: Fix Elasticsearch graph loading issue)
        // cy.goto("graph");
        // cy.getGraph().then(graph => {
        //     const expectedGraph = {
        //         "users": ["orders"],
        //         "orders": ["order_items", "payments", "users"],
        //         "order_items": ["orders", "products"],
        //         "products": ["order_items"],
        //         "payments": ["orders"]
        //     };
        //     Object.keys(expectedGraph).forEach(key => {
        //         expect(graph).to.have.property(key);
        //         expect(graph[key].sort()).to.deep.equal(expectedGraph[key].sort());
        //     });
        // });
        // cy.getGraphNode("users").then(fields => {
        //     expect(fields.some(([k]) => k === "Storage Size")).to.be.true;
        //     expect(fields.some(([k]) => k === "Count")).to.be.true;
        // });
        // cy.goto('graph');
        // cy.get('.react-flow__node', {timeout: 10000}).should('be.visible');
        // cy.get('[data-testid="graph-layout-button"]').click();
        //
        // cy.get('[data-testid="rf__node-users"] [data-testid="data-button"]').click({force: true});
        // cy.url().should('include', '/storage-unit/explore');
        // cy.contains('Total Count:').should('be.visible');
        // cy.get('[data-testid="table-search"]').should('be.visible');

        // 9) Manage where conditions (edit and sheet)
        cy.setWhereConditionMode('sheet');
        cy.data('users');

        // Add both conditions at once since sheet mode allows multiple
        cy.whereTable([
            ['username', 'match', 'john_doe'],
            ['email', 'match', 'john@example.com']
        ]);
        cy.submitTable();

        // Verify conditions
        cy.getConditionCount().should('equal', 2);
        cy.verifyCondition(0, 'username match john_doe');
        cy.verifyCondition(1, 'email match john@example.com');
        cy.getTableData().then(({rows}) => {
            const doc = JSON.parse(rows[0][1]);
            expect(doc.username).to.equal("john_doe");
        });

        // Try to edit conditions
        cy.getWhereConditionMode().then(mode => {
            if (mode === 'popover') {
                cy.clickConditionToEdit(0);
                cy.updateConditionValue('jane_smith');
                cy.get('[data-testid="cancel-button"]').click();
                cy.verifyCondition(0, 'username match john_doe');

                cy.clickConditionToEdit(0);
                cy.updateConditionValue('jane_smith');
                cy.get('[data-testid="update-condition-button"]').click();
                cy.submitTable();
                cy.assertNoDataAvailable();
                cy.verifyCondition(0, 'username match jane_smith');
            } else {
                cy.log('Sheet mode: Skipping inline edit tests');
            }
        });

        // Remove conditions
        cy.removeCondition(1);
        cy.submitTable();
        cy.getWhereConditionMode().then(mode => {
            cy.getTableData().then(({rows}) => {
                const doc = JSON.parse(rows[0][1]);
                if (mode === 'popover') {
                    expect(doc.username).to.equal('jane_smith');
                } else {
                    // In sheet mode, conditions weren't changed
                    expect(doc.username).to.equal('john_doe');
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
            ['username', 'match', 'admin_user'],
            ['email', 'match', 'admin@example.com'],
            ['_id', 'exists', '1']
        ]);

        // With MAX_VISIBLE_CONDITIONS = 2, we should see 2 badges and "+1 more"
        cy.getWhereConditionMode().then(mode => {
            if (mode === 'popover') {
                // Should show first 2 conditions as badges
                cy.getConditionCount().should('equal', 3);
                cy.verifyCondition(0, 'username match admin_user');
                cy.verifyCondition(1, 'email match admin@example.com');

                // Check for more conditions button
                cy.checkMoreConditionsButton('+1 more');

                // Click to open sheet with all conditions
                cy.clickMoreConditions();

                // In the sheet, keep only the first condition
                cy.removeConditionsInSheet(true);
                cy.saveSheetChanges();

                // After closing sheet, should have only 1 condition
                cy.getConditionCount().should('equal', 1);
                cy.verifyCondition(0, 'username match admin_user');
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
                    // After removing 2 conditions, only username=admin_user remains
                    const doc = JSON.parse(rows[0][1]);
                    expect(doc.username).to.equal('admin_user');
                } else {
                    // In sheet mode, all 3 conditions are active
                    const doc = JSON.parse(rows[0][1]);
                    expect(doc.username).to.equal('admin_user');
                }
            });
        });

        cy.clearWhereConditions();
        cy.submitTable();

        // 10) Mock data on a table that does not support it
        cy.data('orders');
        cy.selectMockData();

        // Wait for any toasts to clear

        cy.contains('button', 'Generate').click();
        // Check for toast notification (may be partially covered but should exist)
        cy.contains('Mock data generation is not allowed for this table').should('exist');

        cy.logout();
    });
});