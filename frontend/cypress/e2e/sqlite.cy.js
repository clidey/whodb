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

describe('Sqlite3 E2E test', () => {
    it('runs full Sqlite3 E2E flow', () => {
        // login and setup
        cy.login('Sqlite3', undefined, undefined, undefined, 'e2e_test.db');

        // 1) Lists tables
        cy.getTables().then(storageUnitNames => {
            expect(storageUnitNames).to.be.an('array');
            expect(storageUnitNames).to.deep.equal([
                "order_items",
                "orders",
                "payments",
                "products",
                "test_casting",
                "users"
            ]);
        });

        // 2) Explore users table metadata
        cy.explore("users");
        cy.getExploreFields().then(fields => {
            expect(fields.some(([k, v]) => k === "Type" && v === "table")).to.be.true;
            expect(fields.some(([k]) => k === "Count")).to.be.true;
            const expectedColumns = [
                ["id", "INTEGER"],
                ["username", "TEXT"],
                ["email", "TEXT"],
                ["password", "TEXT"],
                ["created_at", "DATETIME"]
            ];
            expectedColumns.forEach(([col, type]) => {
                expect(fields.some(([k, v]) => k === col && v === type)).to.be.true;
            });
        });

        // 3) Data: verify table data
        cy.data("users");
        cy.sortBy(0);
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal([
                "",
                "id",
                "username",
                "email",
                "password",
                "created_at"
            ]);
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
                ['', '1', 'john_doe', 'john@example.com', 'securepassword1'],
                ['', '2', 'jane_smith', 'jane@example.com', 'securepassword2'],
                ['', '3', 'admin_user', 'admin@example.com', 'adminpass1']
            ]);
        });

        // 4) Respects page size pagination
        cy.setTablePageSize(1);
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
                ['', '1', 'john_doe', 'john@example.com', 'securepassword1'],
            ]);
        });

        // 5) Applies where condition id=3 and clears it
        cy.setTablePageSize(10);
        cy.whereTable([['id', '=', '3']]);
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
                ['', '3', 'admin_user', 'admin@example.com', 'adminpass1']
            ]);
        });
        cy.clearWhereConditions();
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(3);
        });

        // 6) Edit row: save, revert, and cancel
        cy.setTablePageSize(2);
        cy.submitTable();
        cy.updateRow(1, 1, 'jane_smith1', false);
        cy.getTableData().then(({rows}) => {
            expect(rows[1][2]).to.equal('jane_smith1');
        });
        cy.updateRow(1, 1, 'jane_smith', false);
        cy.getTableData().then(({rows}) => {
            expect(rows[1][2]).to.equal('jane_smith');
        });
        cy.updateRow(1, 1, 'jane_smith_temp');
        cy.getTableData().then(({rows}) => {
            expect(rows[1][2]).to.equal('jane_smith');
        });

        // 7) Search highlights multiple matches in sequence
        cy.searchTable('john');
        cy.getHighlightedCell().first().should('have.text', 'john_doe');
        cy.searchTable('john');
        cy.getHighlightedCell().first().should('have.text', 'john@example.com');

        // 8) Graph topology and node fields
        cy.goto("graph");
        cy.getGraph().then(graph => {
            const expectedGraph = {
                "users": ["orders"],
                "orders": ["order_items", "payments"],
                "order_items": [],
                "products": ["order_items"],
                "payments": [],
            };
            Object.keys(expectedGraph).forEach(key => {
                expect(graph).to.have.property(key);
                expect(graph[key].sort()).to.deep.equal(expectedGraph[key].sort());
            });
        });
        cy.getGraphNode("users").then(fields => {
            expect(fields.some(([k, v]) => k === "Type" && v === "table")).to.be.true;
            expect(fields.some(([k]) => k === "Count")).to.be.true;
            const expectedColumns = [
                ["id", "INTEGER"],
                ["username", "TEXT"],
                ["email", "TEXT"],
                ["password", "TEXT"],
                ["created_at", "DATETIME"]
            ];
            expectedColumns.forEach(([col, type]) => {
                expect(fields.some(([k, v]) => k === col && v === type)).to.be.true;
            });
        });
        cy.goto('graph');
        cy.get('.react-flow__node', {timeout: 10000}).should('be.visible');
        cy.get('[data-testid="graph-layout-button"]').click();

        cy.get('[data-testid="rf__node-users"] [data-testid="data-button"]').click({force: true});
        cy.url().should('include', '/storage-unit/explore');
        cy.contains('Total Count:').should('be.visible');
        cy.get('[data-testid="table-search"]').should('be.visible');

        // 9) Scratchpad page runs queries and manages cells
        cy.goto("scratchpad");
        cy.addScratchpadPage();
        cy.getScratchpadPages().then(pages => {
            expect(pages).to.deep.equal(["Page 1", "Page 2"]);
        });
        cy.deleteScratchpadPage(0);
        cy.getScratchpadPages().then(pages => {
            expect(pages).to.deep.equal(["Page 1", "Page 2"]);
        });
        cy.deleteScratchpadPage(0, false);
        cy.getScratchpadPages().then(pages => {
            expect(pages).to.deep.equal(["Page 2"]);
        });

        cy.writeCode(0, "SELECT * FROM users1;");
        cy.runCode(0);
        cy.getCellError(0).then(err => expect(err).to.equal('no such table: users1'));

        cy.writeCode(0, "SELECT * FROM users ORDER BY id;");
        cy.runCode(0);
        cy.getCellQueryOutput(0).then(({rows, columns}) => {
            expect(columns).to.deep.equal([
                "",
                "id",
                "username",
                "email",
                "password",
                "created_at"
            ]);
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
                ['', '1', 'john_doe', 'john@example.com', 'securepassword1'],
                ['', '2', 'jane_smith', 'jane@example.com', 'securepassword2'],
                ['', '3', 'admin_user', 'admin@example.com', 'adminpass1']
            ]);
        });

        cy.writeCode(0, "UPDATE users SET username='john_doe1' WHERE id=1");
        cy.runCode(0);
        cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));

        cy.writeCode(0, "UPDATE users SET username='john_doe' WHERE id=1");
        cy.runCode(0);
        cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));

        cy.addCell(0);
        cy.writeCode(1, "SELECT * FROM users WHERE id=1;");
        cy.runCode(1);
        cy.getCellQueryOutput(1).then(({rows, columns}) => {
            expect(columns).to.deep.equal([
                "",
                "id",
                "username",
                "email",
                "password",
                "created_at"
            ]);
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
                ['', '1', 'john_doe', 'john@example.com', 'securepassword1'],
            ]);
        });

        cy.removeCell(0);
        cy.getCellQueryOutput(0).then(({rows, columns}) => {
            expect(columns).to.deep.equal([
                "",
                "id",
                "username",
                "email",
                "password",
                "created_at"
            ]);
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
                ['', '1', 'john_doe', 'john@example.com', 'securepassword1'],
            ]);
        });

        // 9.5) Test Query History functionality
        // First, run a few more queries to build up history
        cy.writeCode(0, 'SELECT COUNT(*) as user_count FROM users;');
        cy.runCode(0);
        cy.getCellQueryOutput(0).then(({rows}) => {
            expect(rows[0][1]).to.equal('3'); // Should have 3 users
        });

        cy.writeCode(0, 'SELECT username, email FROM users WHERE id > 1;');
        cy.runCode(0);
        cy.getCellQueryOutput(0).then(({rows}) => {
            expect(rows.length).to.equal(2); // Should have 2 users with id > 1
        });

        // Test Query History - Copy functionality
        cy.openQueryHistory();
        cy.getQueryHistoryItems().then(items => {
            expect(items.length).to.be.greaterThan(0);
            // Verify the most recent queries are in history
            expect(items[0]).to.include('SELECT username, email FROM users WHERE id > 1');
            expect(items[1]).to.include('SELECT COUNT(*) as user_count FROM users');
        });

        // Test copy functionality
        cy.copyQueryFromHistory(0);
        cy.closeQueryHistory();

        // Test Query History - Clone to editor functionality
        cy.openQueryHistory();
        cy.cloneQueryToEditor(1, 0); // Clone the COUNT query
        
        // Run the cloned query to verify it works
        cy.runCode(0);
        cy.getCellQueryOutput(0).then(({rows}) => {
            expect(rows[0][1]).to.equal('3');
        });

        // Test Query History - Execute from history functionality
        cy.openQueryHistory();
        cy.executeQueryFromHistory(0); // Execute the username/email query from history
        cy.closeQueryHistory();

        // Verify the query executed and results are shown
        cy.getCellQueryOutput(0).then(({rows}) => {
            expect(rows.length).to.equal(2);
            const usernames = rows.map(row => row[1]);
            expect(usernames).to.include('jane_smith');
            expect(usernames).to.include('admin_user');
        });

        // 10) Manage where conditions (edit and sheet)
        cy.setWhereConditionMode('sheet');
        cy.data('users');

        cy.whereTable([
            ['id', '=', '1'],
            ['username', '=', 'john_doe'],
        ]);
        cy.submitTable();

        // Verify conditions
        cy.getConditionCount().should('equal', 2);
        cy.verifyCondition(0, 'id = 1');
        cy.verifyCondition(1, 'username = john_doe');
        cy.getTableData().then(({rows}) => {
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([['', '1', 'john_doe', 'john@example.com', 'securepassword1']]);
        });

        // Try to edit conditions
        cy.getWhereConditionMode().then(mode => {
            if (mode === 'popover') {
                cy.clickConditionToEdit(0);
                cy.updateConditionValue('2');
                cy.get('[data-testid="cancel-button"]').click();
                cy.verifyCondition(0, 'id = 1');

                // Edit and save
                cy.clickConditionToEdit(0);
                cy.updateConditionValue('2');
                cy.get('[data-testid="update-condition-button"]').click();
                cy.submitTable();
                cy.assertNoDataAvailable();
                cy.verifyCondition(0, 'id = 2');
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
                    expect(rows[0][2]).to.equal('jane_smith');
                } else {
                    // In sheet mode, conditions weren't changed
                    expect(rows[0][2]).to.equal('john_doe');
                }
            });
        });

        // Clear ALL conditions before proceeding to avoid duplicates
        cy.clearWhereConditions();
        cy.wait(500); // Wait for conditions to clear

        // Verify all conditions are cleared
        cy.getConditionCount().should('equal', 0);

        // Add conditions for testing the more button
        cy.whereTable([
            ['id', '=', '3'],  // This will show admin_user
            ['username', '=', 'admin_user'],
            ['email', '!=', 'jane@example.com']
        ]);

        // With MAX_VISIBLE_CONDITIONS = 2, we should see 2 badges and "+1 more"
        cy.getWhereConditionMode().then(mode => {
            if (mode === 'popover') {
                // Should show first 2 conditions as badges
                cy.getConditionCount().should('equal', 3);
                cy.verifyCondition(0, 'id = 3');
                cy.verifyCondition(1, 'username = admin_user');

                // Check for more conditions button (3rd condition is hidden)
                cy.checkMoreConditionsButton('+1 more');

                // Click to open sheet with all conditions
                cy.clickMoreConditions();

                // In the sheet, keep only the first condition (id = 3)
                cy.removeConditionsInSheet(true);
                cy.saveSheetChanges();

                // After closing sheet, should have only 1 condition
                cy.getConditionCount().should('equal', 1);
                cy.verifyCondition(0, 'id = 3');
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
                    // After removing 2 conditions, only id=3 remains
                    expect(rows[0][2]).to.equal('admin_user');
                } else {
                    // In sheet mode, all 3 conditions are active, should still show admin_user
                    expect(rows[0][2]).to.equal('admin_user');
                }
            });
        });

        cy.clearWhereConditions();
        cy.submitTable();

        // 11) Export data (csv/excel) and selected rows
        cy.data('users');
        cy.intercept('POST', '/api/export').as('export');

        cy.contains('button', 'Export all').click();

        cy.contains('h2', 'Export Data').should('be.visible');
        cy.get('[role="dialog"]').within(() => {
            cy.contains('label', 'Format').parent().find('[role="combobox"]').first().should('contain.text', 'CSV');
            cy.contains('label', 'Delimiter').parent().find('[role="combobox"]').first().should('contain.text', 'Comma');
        });
        cy.get('[role="dialog"]').within(() => {
            cy.contains('button', 'Export').click();
        });
        cy.wait('@export').then(({request, response}) => {
            expect(response?.statusCode, 'CSV export status').to.equal(200);
            const headers = response?.headers || {};
            const cd = headers['content-disposition'] || headers['Content-Disposition'];
            const ct = headers['content-type'] || headers['Content-Type'];
            expect(cd, 'Content-Disposition for CSV').to.be.a('string');
            expect(cd).to.match(/filename="?users\.csv"?/i);
            if (ct) {
                expect(ct.toLowerCase()).to.match(/text\/csv|application\/octet-stream/);
            }
            expect(request.body.schema).to.equal('');
            expect(request.body.storageUnit).to.equal('users');
            expect(request.body.format).to.equal('csv');
            expect(request.body.delimiter).to.equal(',');
        });

        cy.get('body').type('{esc}');

        cy.get('[role="dialog"]').should('not.exist');

        cy.contains('button', 'Export all').click();

        cy.contains('h2', 'Export Data').should('be.visible');
        cy.get('[role="dialog"]').within(() => {
            cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
        });
        cy.get('[role="listbox"]').should('be.visible');
        cy.get('[role="option"]').contains('Excel').click({force: true});
        cy.get('[role="listbox"]').should('not.exist');
        cy.contains('Excel XLSX format').should('be.visible');
        cy.get('[role="dialog"]').within(() => {
            cy.contains('button', 'Export').click();
        });
        cy.wait('@export').then(({response}) => {
            expect(response?.statusCode, 'Excel export status').to.equal(200);
            const headers = response?.headers || {};
            const cd = headers['content-disposition'] || headers['Content-Disposition'];
            const ct = headers['content-type'] || headers['Content-Type'];
            expect(cd, 'Content-Disposition for Excel').to.be.a('string');
            expect(cd).to.match(/filename="users\.xlsx"?/i);
            if (ct) {
                expect(ct.toLowerCase()).to.match(/application\/vnd\.openxmlformats-officedocument\.spreadsheetml\.sheet|application\/octet-stream/);
            }
        });

        cy.get('body').type('{esc}');

        cy.get('[role="dialog"]').should('not.exist');

        cy.get('table tbody tr').first().rightclick({force: true});
        cy.contains('div,button,span', 'Select Row').click({force: true});

        cy.get('button').contains('Export').filter(':visible').last().click();

        cy.get('[role="dialog"]').should('be.visible');

        cy.contains('You are about to export 1 selected rows.').should('be.visible');

        cy.get('[role="dialog"]').within(() => {
            cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
        });
        cy.get('[role="listbox"]').should('be.visible');
        cy.get('[role="option"]').first().click({force: true});

        cy.get('[role="dialog"]').within(() => {
            cy.contains('label', 'Delimiter').should('be.visible');
            cy.contains('label', 'Delimiter').parent().within(() => {
                cy.get('[role="combobox"]').eq(-1).click();
            });
        });

        cy.get('[role="listbox"]').should('be.visible');

        cy.get('[role="option"]').eq(-1).click({force: true});
        cy.get('[role="listbox"]').should('not.exist');
        cy.get('[role="dialog"]').within(() => {
            cy.contains('label', 'Delimiter').parent().find('[role="combobox"]').eq(-1).invoke('text').should('include', '|');
        });

        cy.get('[role="dialog"]').within(() => {
            cy.contains('button', 'Export').click();
        });
        cy.wait('@export').then(({request, response}) => {
            expect(response?.statusCode, 'Selected CSV export status').to.equal(200);
            const headers = response?.headers || {};
            const cd = headers['content-disposition'] || headers['Content-Disposition'];
            const ct = headers['content-type'] || headers['Content-Type'];
            expect(cd, 'Content-Disposition for selected CSV').to.be.a('string');
            expect(cd).to.match(/filename="?users\.csv"?/i);
            if (ct) {
                expect(ct.toLowerCase()).to.match(/text\/csv|application\/octet-stream/);
            }
            expect(request.body.delimiter).to.equal('|');
            expect(request.body.selectedRows).to.exist;
            expect(Array.isArray(request.body.selectedRows)).to.be.true;
            expect(request.body.selectedRows.length).to.be.greaterThan(0);
        });

        cy.get('[role="dialog"]').should('not.exist');

        // 12) Test type casting behavior (related to issue #613)
        cy.data('test_casting');

        // Test adding a row with various numeric types that require casting
        cy.addRow({
            bigint_col: '5000000000',  // Large number as string
            integer_col: '42',          // Regular integer as string
            smallint_col: '256',        // Small integer as string
            numeric_col: '9876.54',     // Decimal as string
            description: 'Test casting from strings'
        });

        // Verify the row was added - should now have 4 rows
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(4, 'Should have 4 rows after first addition');
        });

        // Verify the data was inserted correctly
        cy.sortBy(0); // Sort by id to get consistent ordering
        cy.getTableData().then(({columns, rows}) => {
            expect(rows.length).to.equal(4, 'Should still have 4 rows');
            // Find the row we just added (should be the last one with id=4)
            const addedRow = rows[rows.length - 1];
            expect(addedRow[1]).to.match(/^\d+$/); // id should be a number
            expect(addedRow[2]).to.equal('5000000000');
            expect(addedRow[3]).to.equal('42');
            expect(addedRow[4]).to.equal('256');
            expect(addedRow[5]).to.equal('9876.54');
            expect(addedRow[6]).to.equal('Test casting from strings');
        });

        // Test editing a row with type casting
        cy.updateRow(1, 1, '7500000000', false); // Update bigint_col
        cy.getTableData().then(({rows}) => {
            expect(rows[1][2]).to.equal('7500000000');
        });

        // Restore original value
        cy.updateRow(1, 1, '1000000', false);

        // Verify we still have 4 rows before adding the second test row
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(4, 'Should have 4 rows before second addition');
        });

        // Test edge cases: zero and negative numbers, and verify the row was added successfully
        cy.getTableData().then(({rows}) => {
            const initialRowCount = rows.length;
            cy.addRow({
                bigint_col: '0',
                integer_col: '-42',
                smallint_col: '-100',
                numeric_col: '-1234.56',
                description: 'Zero and negative values'
            });

            cy.getTableData().then(({rows: newRows}) => {
                expect(newRows.length).to.equal(initialRowCount + 1, 'Should have one new row');
                const lastRow = newRows[newRows.length - 1];
                expect(lastRow[2]).to.equal('0');
                expect(lastRow[3]).to.equal('-42');
                expect(lastRow[4]).to.equal('-100');
                expect(lastRow[5]).to.equal('-1234.56');
                expect(lastRow[6]).to.equal('Zero and negative values');
            });
        });

        // Clean up: delete the test rows we added
        cy.getTableData().then(({rows}) => {
            const initialRowCount = rows.length;
            cy.deleteRow(initialRowCount - 1); // Delete the last row
            cy.getTableData().its('rows.length').should('eq', initialRowCount - 1);
        });

        cy.getTableData().then(({rows}) => {
            const initialRowCount = rows.length;
            cy.deleteRow(initialRowCount - 1); // Delete the last row again
            cy.getTableData().its('rows.length').should('eq', initialRowCount - 1);
        });

        // Verify cleanup
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(3); // Should be back to original 3 rows
        });

        // 13) Open scratchpad drawer from Explore and run query
        cy.data('users');
        cy.get('[data-testid="scratchpad-button"]').click();
        cy.contains('h2', 'Scratchpad').should('be.visible');

        cy.get('[data-testid="code-editor"]').should('exist');
        cy.get('[data-testid="code-editor"]').should('contain', 'SELECT * FROM users');

        cy.get('[data-testid="run-submit-button"]').filter(':contains("Run")').first().click();

        cy.get('[role="dialog"] table', {timeout: 500}).should('be.visible');
        cy.get('[role="dialog"] table thead th').should('contain', 'id');
        cy.get('[role="dialog"] table thead th').should('contain', 'username');
        cy.get('[role="dialog"] table tbody tr').should('have.length.at.least', 1);

        cy.get('body').type('{esc}');
        cy.get('[data-testid="table-search"]').should('be.visible');

        // 14) Open Mock Data sheet, enforce limits, and show overwrite confirmation
        cy.data('users');
        cy.selectMockData();

        cy.contains('div', 'Mock Data for users').should('be.visible');
        cy.contains('Note').should('be.visible');

        cy.contains('label', 'Number of Rows').parent().find('input').as('rowsInput');
        cy.get('@rowsInput').clear().type('300');
        cy.get('@rowsInput').invoke('val').then((val) => {
            expect(parseInt(val, 10)).to.be.equal(200);
        });

        cy.contains('label', 'Number of Rows').parent().find('input').as('rowsInput');
        cy.get('@rowsInput').clear().type('50');
        cy.contains('button', 'Generate').click();

        cy.contains(/Total Count:\s*\d+/).should(($el) => {
            const text = $el.text();
            const match = text.match(/Total Count:\s*(\d+)/);
            const count = match ? parseInt(match[1], 10) : 0;
            // expect(count).to.be.at.least(50); // TODO: fix this when we can sync the Total Count on update
        });

        cy.data('users');
        cy.selectMockData();
        cy.get('@rowsInput').clear().type('10')
        cy.contains('label', 'Data Handling').parent().find('[role="combobox"]').eq(-1).click();
        cy.contains('[role="option"]', 'Overwrite existing data').click();
        cy.contains('button', 'Generate').click();
        cy.contains('button', 'Yes, Overwrite').should('be.visible').click();

        cy.contains(/Total Count:\s*\d+/).should(($el) => {
            const text = $el.text();
            const match = text.match(/Total Count:\s*(\d+)/);
            const count = match ? parseInt(match[1], 10) : 0;
            // expect(count).to.be.equal(10); // TODO: fix this when we can sync the Total Count on update
        });
        cy.get('body').type('{esc}');

        // 15) Mock data on a table that does not support it
        cy.data('orders');
        cy.selectMockData();

        // Wait for any toasts to clear

        cy.contains('button', 'Generate').click();
        cy.contains('Mock data generation is not allowed for this table').should('exist');

        cy.get('body').type('{esc}');
        cy.logout()
    });
});
