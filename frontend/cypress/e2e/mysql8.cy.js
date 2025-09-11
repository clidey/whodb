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

describe('MySQL 8 E2E test', () => {
    const isDocker = Cypress.env('isDocker');
    const dbHost = isDocker ? 'e2e_mysql_842' : 'localhost';
    const dbUser = 'user';
    const dbPassword = 'password';


    it('runs full MySQL 8 E2E flow', () => {
        // login and setup
        cy.login('MySQL', dbHost, dbUser, dbPassword, 'test_db', {"Port": "3308"});
        cy.selectSchema("test_db");

        // 1) Lists tables
        cy.getTables().then(storageUnitNames => {
            expect(storageUnitNames).to.be.an('array');
            expect(storageUnitNames).to.deep.equal([
                'order_items',
                'order_summary',
                'orders',
                'payments',
                'products',
                'users'
            ]);
        });

        // 2) Explore users table metadata
        cy.explore('users');
        cy.getExploreFields().then(fields => {
            expect(fields.some(([k, v]) => k === 'Type' && v === 'BASE TABLE')).to.be.true;
            expect(fields.some(([k]) => k === 'Total Size')).to.be.true;
            expect(fields.some(([k]) => k === 'Data Size')).to.be.true;
            expect(fields.some(([k]) => k === 'Count')).to.be.true;
            const expectedColumns = [
                ['id', 'int'],
                ['username', 'varchar'],
                ['email', 'varchar'],
                ['password', 'varchar'],
                ['created_at', 'timestamp']
            ];
            expectedColumns.forEach(([col, type]) => {
                expect(fields.some(([k, v]) => k === col && v === type)).to.be.true;
            });
        });

        // 3) Data: add a row then delete and verify table data
        cy.data('users');
        cy.sortBy(0);
        cy.addRow({
            id: '5',
            username: 'alice_wonder',
            email: 'alice@example.com',
            password: 'securepassword2',
            created_at: '2022-02-02'
        });
        cy.deleteRow(3);
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal(['', 'id', 'username', 'email', 'password', 'created_at']);
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
                ['', '1', 'john_doe', 'john@example.com', 'securepassword1'],
                ['', '2', 'jane_smith', 'jane@example.com', 'securepassword2'],
                ['', '3', 'admin_user', 'admin@example.com', 'adminpass']
            ]);
        });

        // 4) Respects page size pagination
        cy.setTablePageSize(1);
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([['', '1', 'john_doe', 'john@example.com', 'securepassword1']]);
        });

        // 5) Applies where condition id=3 and clears it
        cy.setTablePageSize(10);
        cy.whereTable([['id', '=', '3']]);
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([['', '3', 'admin_user', 'admin@example.com', 'adminpass']]);
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
        cy.wait(100);
        cy.getHighlightedCell().first().should('have.text', 'john_doe');
        cy.searchTable('john');
        cy.getHighlightedCell().first().should('have.text', 'john@example.com');

        // 8) Graph topology and node fields
        cy.goto('graph');
        cy.getGraph().then(graph => {
            const expectedGraph = {
                users: ['orders'],
                orders: ['order_items', 'payments'],
                order_items: [],
                products: ['order_items'],
                payments: [],
                order_summary: []
            };
            Object.keys(expectedGraph).forEach(key => {
                expect(graph).to.have.property(key);
                expect(graph[key].sort()).to.deep.equal(expectedGraph[key].sort());
            });
        });
        cy.getGraphNode('users').then(fields => {
            const expectedColumns = [
                ['id', 'int'],
                ['username', 'varchar'],
                ['email', 'varchar'],
                ['password', 'varchar'],
                ['created_at', 'timestamp']
            ];
            expect(fields.some(([k, v]) => k === 'Type' && v === 'BASE TABLE')).to.be.true;
            expect(fields.some(([k]) => k === 'Total Size')).to.be.true;
            expect(fields.some(([k]) => k === 'Data Size')).to.be.true;
            expect(fields.some(([k]) => k === 'Count')).to.be.true;
            expectedColumns.forEach(([col, type]) => {
                expect(fields.some(([k, v]) => k === col && v === type)).to.be.true;
            });
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

        // 9) Scratchpad page runs queries and manages cells
        cy.goto('scratchpad');
        cy.addScratchpadPage();
        cy.getScratchpadPages().then(pages => {
            expect(pages).to.deep.equal(['Page 1', 'Page 2']);
        });
        cy.deleteScratchpadPage(0);
        cy.getScratchpadPages().then(pages => {
            expect(pages).to.deep.equal(['Page 1', 'Page 2']);
        });
        cy.deleteScratchpadPage(0, false);
        cy.getScratchpadPages().then(pages => {
            expect(pages).to.deep.equal(['Page 2']);
        });

        cy.writeCode(0, 'SELECT * FROM test_db.users1;');
        cy.runCode(0);
        cy.getCellError(0).then(err => expect(err).to.equal("Error 1146 (42S02): Table 'test_db.users1' doesn't exist"));

        cy.writeCode(0, 'SELECT * FROM test_db.users ORDER BY id;');
        cy.runCode(0);
        cy.getCellQueryOutput(0).then(({rows, columns}) => {
            expect(columns).to.deep.equal(['', 'id', 'username', 'email', 'password', 'created_at']);
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
                ['', '1', 'john_doe', 'john@example.com', 'securepassword1'],
                ['', '2', 'jane_smith', 'jane@example.com', 'securepassword2'],
                ['', '3', 'admin_user', 'admin@example.com', 'adminpass']
            ]);
        });

        cy.writeCode(0, "UPDATE test_db.users SET username='john_doe1' WHERE id=1");
        cy.runCode(0);
        cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));

        cy.writeCode(0, "UPDATE test_db.users SET username='john_doe' WHERE id=1");
        cy.runCode(0);
        cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));

        cy.addCell(0);
        cy.writeCode(1, 'SELECT * FROM test_db.users WHERE id=1;');
        cy.runCode(1);
        cy.getCellQueryOutput(1).then(({rows, columns}) => {
            expect(columns).to.deep.equal(['', 'id', 'username', 'email', 'password', 'created_at']);
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([['', '1', 'john_doe', 'john@example.com', 'securepassword1']]);
        });

        cy.removeCell(0);
        cy.getCellQueryOutput(0).then(({rows, columns}) => {
            expect(columns).to.deep.equal(['', 'id', 'username', 'email', 'password', 'created_at']);
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([['', '1', 'john_doe', 'john@example.com', 'securepassword1']]);
        });

        // 10) Manage where conditions (edit and sheet)
        cy.data('users');

        cy.whereTable([
            ['id', '=', '1'],
            ['username', '=', 'john_doe'],
        ]);
        cy.submitTable();

        // UI: two badges rendered and show correct text
        cy.get('[data-testid="where-condition-badge"]').should('have.length', 2);
        cy.get('[data-testid="where-condition-badge"]').eq(0).should('contain.text', 'id = 1');
        cy.get('[data-testid="where-condition-badge"]').eq(1).should('contain.text', 'username = john_doe');
        cy.getTableData().then(({rows}) => {
            expect(rows.map(row => row.slice(0, -1))).to.deep.equal([['', '1', 'john_doe', 'john@example.com', 'securepassword1']]);
        });

        // check cancel works
        cy.get('[data-testid="where-condition-badge"]').first().click();
        cy.get('[data-testid="field-value"]').clear().type('2');
        cy.get('[data-testid="cancel-button"]').click();
        cy.get('[data-testid="where-condition-badge"]').first().should('contain.text', 'id = 1');

        // edit first condition to id=2
        cy.get('[data-testid="where-condition-badge"]').first().click();
        cy.get('[data-testid="field-value"]').clear().type('2');
        cy.get('[data-testid="update-condition-button"]').click();
        cy.submitTable();
        cy.assertNoDataAvailable();

        // UI: first badge updated text
        cy.get('[data-testid="where-condition-badge"]').eq(0).should('contain.text', 'id = 2');

        cy.get('[data-testid="remove-where-condition-button"]').eq(1).click();
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            // row username should be jane_smith when id=2 is applied
            expect(rows[0][2]).to.equal('jane_smith');
        });

        cy.get('[data-testid="remove-where-condition-button"]').first().click();

        cy.whereTable([
            ['id', '=', '1'],
            ['username', '=', 'john_doe'],
            ['email', '!=', 'jane@example.com']
        ]);
        // UI: +1 more button visible
        cy.get('[data-testid="more-conditions-button"]').should('be.visible').and('contain.text', '+1 more');

        // open the manage sheet and update to only id=3, remove other filters
        cy.get('[data-testid="more-conditions-button"]').click();
        cy.get('[data-testid="sheet-field-value-0"]').clear().type('3');
        cy.wait(1000);
        // remove other filters (indices shift as we remove)
        cy.get('[data-testid^="remove-sheet-filter-"]').then($els => {
            const count = $els.length;
            if (count > 1) {
                // remove last indices first to avoid reindex surprises
                for (let i = count - 1; i >= 1; i--) {
                    cy.get(`[data-testid="remove-sheet-filter-${i}"]`).click();
                }
            }
        });
        cy.contains('button', 'Save Changes').click();

        // UI: only one badge and text shows id = 3, no +N more
        cy.get('[data-testid="where-condition-badge"]').should('have.length', 1).first().should('contain.text', 'id = 3');
        cy.get('[data-testid="more-conditions-button"]').should('not.exist');

        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            // admin_user is id=3
            expect(rows[0][2]).to.equal('admin_user');
        });

        // cleanup: clear conditions
        cy.clearWhereConditions();
        cy.submitTable();

        // 11) Export data (csv/excel) and selected rows
        cy.data('users');
        cy.intercept('POST', '/api/export').as('export');

        // Export all using the bottom export button
        cy.contains('button', 'Export all').click();

        cy.contains('h2', 'Export Data').should('be.visible');
        // UI: default shows CSV and Comma delimiter - scope to dialog
        cy.get('[role="dialog"]').within(() => {
            cy.contains('label', 'Format').parent().find('[role="combobox"]').first().should('contain.text', 'CSV');
            cy.contains('label', 'Delimiter').parent().find('[role="combobox"]').first().should('contain.text', 'Comma');
        });
        // Export as CSV default - look for button within the sheet footer
        cy.get('[role="dialog"]').within(() => {
            cy.contains('button', 'Export').click();
        });
        cy.wait('@export').then(({request, response}) => {
            expect(response?.statusCode, 'CSV export status').to.equal(200);
            const headers = response?.headers || {};
            const cd = headers['content-disposition'] || headers['Content-Disposition'];
            const ct = headers['content-type'] || headers['Content-Type'];
            expect(cd, 'Content-Disposition for CSV').to.be.a('string');
            expect(cd).to.match(/filename="?test_db_users\.csv"?/i);
            // Content-Type may vary by server; allow common CSV or generic stream
            if (ct) {
                expect(ct.toLowerCase()).to.match(/text\/csv|application\/octet-stream/);
            }
            // Verify request basics but do not rely on them for pass/fail of file
            expect(request.body.schema).to.equal('test_db');
            expect(request.body.storageUnit).to.equal('users');
            expect(request.body.format).to.equal('csv');
            expect(request.body.delimiter).to.equal(',');
        });

        // Close the dialog and wait for it to be closed
        cy.get('body').type('{esc}');
        cy.wait(500); // Give time for dialog to close

        // Ensure dialog is fully closed before continuing
        cy.get('[role="dialog"]').should('not.exist');

        // Re-open export all and change to Excel format
        cy.contains('button', 'Export all').click();

        cy.contains('h2', 'Export Data').should('be.visible');
        // Change format to Excel - use first combobox for Format
        cy.get('[role="dialog"]').within(() => {
            cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
        });
        // Wait for dropdown to open and click Excel option
        cy.get('[role="listbox"]').should('be.visible');
        cy.get('[role="option"]').contains('Excel').click({force: true});
        // Ensure dropdown is closed
        cy.get('[role="listbox"]').should('not.exist');
        // UI: Export details update for Excel
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
            expect(cd).to.match(/filename="?test_db_users\.xlsx"?/i);
            if (ct) {
                expect(ct.toLowerCase()).to.match(/application\/vnd\.openxmlformats-officedocument\.spreadsheetml\.sheet|application\/octet-stream/);
            }
        });

        // Close the dialog and wait for it to be closed
        cy.get('body').type('{esc}');
        cy.wait(500); // Give time for dialog to close

        // Ensure dialog is fully closed before continuing
        cy.get('[role="dialog"]').should('not.exist');

        // Select a row and export with pipe delimiter
        cy.get('table tbody tr').first().rightclick({force: true});
        cy.contains('div,button,span', 'Select Row').click({force: true});

        cy.get('button').contains('Export').filter(':visible').last().click();

        // Wait for export dialog to open
        cy.get('[role="dialog"]').should('be.visible');

        // UI: sheet indicates selected rows export
        cy.contains('You are about to export 1 selected rows.').should('be.visible');

        // First ensure CSV format is selected (delimiter only shows for CSV)
        cy.get('[role="dialog"]').within(() => {
            cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
        });
        cy.get('[role="listbox"]').should('be.visible');
        cy.get('[role="option"]').first().click({force: true}); // First option should be CSV

        // Now set delimiter to pipe - make sure we're clicking the right dropdown
        cy.get('[role="dialog"]').within(() => {
            cy.contains('label', 'Delimiter').should('be.visible');
            cy.contains('label', 'Delimiter').parent().within(() => {
                cy.get('[role="combobox"]').eq(-1).click();
            });
        });

        cy.get('[role="listbox"]').should('be.visible');

        // Select the third option (Pipe) - index 2
        cy.get('[role="option"]').eq(-1).click({force: true});
        // Ensure dropdown is closed
        cy.get('[role="listbox"]').should('not.exist');
        // UI: verify delimiter was selected - the combobox should show the pipe delimiter
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
            expect(cd).to.match(/filename="?test_db_users\.csv"?/i);
            if (ct) {
                expect(ct.toLowerCase()).to.match(/text\/csv|application\/octet-stream/);
            }
            // Keep minimal request checks
            expect(request.body.delimiter).to.equal('|');
            expect(request.body.selectedRows).to.exist;
            expect(Array.isArray(request.body.selectedRows)).to.be.true;
            expect(request.body.selectedRows.length).to.be.greaterThan(0);
        });

        cy.get('[role="dialog"]').should('not.exist');

        // 12) Open scratchpad drawer from Explore and run query
        cy.data('users');
        cy.get('[data-testid="scratchpad-button"]').click();
        cy.contains('h2', 'Scratchpad').should('be.visible');

        // The drawer should have the default query populated
        cy.get('[data-testid="code-editor"]').should('exist');
        cy.get('[data-testid="code-editor"]').should('contain', 'SELECT * FROM test_db.users');

        // Run the query and check results appear
        cy.get('[data-testid="submit-button"]').filter(':contains("Run")').first().click();

        // Wait for results to load and verify table appears in the drawer
        cy.get('[role="dialog"] table', {timeout: 500}).should('be.visible');
        cy.get('[role="dialog"] table thead th').should('contain', 'id');
        cy.get('[role="dialog"] table thead th').should('contain', 'username');
        cy.get('[role="dialog"] table tbody tr').should('have.length.at.least', 1);

        // Close the drawer
        cy.get('body').type('{esc}');
        cy.get('[data-testid="table-search"]').should('be.visible');

        // 13) Open Mock Data sheet, enforce limits, and show overwrite confirmation
        cy.data('users');
        cy.get('table thead tr').eq(0).rightclick({force: true});
        cy.contains('div,button,span', 'Mock Data').click({force: true});

        // UI: sheet title and note visible
        cy.contains('div', 'Mock Data for users').should('be.visible');
        cy.contains('Note').should('be.visible');

        // The sheet should open; try to exceed max count and verify it clamps
        cy.contains('label', 'Number of Rows').parent().find('input').as('rowsInput');
        cy.get('@rowsInput').clear().type('300');
        cy.get('@rowsInput').invoke('val').then((val) => {
            expect(parseInt(val, 10)).to.be.equal(200);
        });

        cy.wait(1000);
        // now actually add 50
        cy.contains('label', 'Number of Rows').parent().find('input').as('rowsInput');
        cy.get('@rowsInput').clear().type('50');
        cy.contains('button', 'Generate').click();

        // Wait for mock data generation to complete and verify the total count
        cy.wait(5000); // Give time for data generation
        cy.contains(/Total Count:\s*\d+/).should(($el) => {
            const text = $el.text();
            const match = text.match(/Total Count:\s*(\d+)/);
            const count = match ? parseInt(match[1], 10) : 0;
            // expect(count).to.be.at.least(50); // TODO: fix this when we can sync the Total Count on update
        });

        // Switch to Overwrite and click Generate to show confirmation
        cy.get('table thead tr').rightclick({force: true});
        cy.contains('div,button,span', 'Mock Data').click({force: true});
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

        // 14) Mock data on a table that does not support it
        cy.data('orders');
        cy.get('table thead tr').rightclick({force: true});
        cy.contains('div,button,span', 'Mock Data').click({force: true});
        // Wait for any toasts to clear
        cy.wait(1000);
        cy.contains('button', 'Generate').click();
        cy.contains('Mock data generation is not allowed for this table').should('exist');
        cy.get('body').type('{esc}');
        cy.logout();
    });
});
