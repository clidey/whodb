/**
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

const dbHost = 'localhost';
const dbUser = 'user';
const dbPassword = 'jio53$*(@nfe)';

describe('Postgres E2E test', () => {
  it('should login correctly', () => {
    // login and setup
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'postgres');
    cy.selectDatabase("test_db");
    cy.selectSchema("test_schema");
    
    // get all tables
    cy.getTables().then(storageUnitNames => {
      cy.log(storageUnitNames);
      expect(storageUnitNames).to.be.an('array');
      expect(storageUnitNames).to.deep.equal([
        "order_items",
        "order_summary",
        "orders",
        "payments",
        "products",
        "users"
      ]);
    });

    // check users table and fields
    cy.explore("users");
    cy.getExploreFields().then(fields => {
      // Check type
      expect(fields.some(([k, v]) => k === "Type" && v === "BASE TABLE")).to.be.true;

      // Check Total Size, Data Size, Count (just keys exist)
      expect(fields.some(([k]) => k === "Total Size")).to.be.true;
      expect(fields.some(([k]) => k === "Data Size")).to.be.true;
      expect(fields.some(([k]) => k === "Count")).to.be.true;

      // Check columns and types
      const expectedColumns = [
        ["id", "integer"],
        ["username", "character varying"],
        ["email", "character varying"],
        ["password", "character varying"],
        ["created_at", "timestamp without time zone"]
      ];
      expectedColumns.forEach(([col, type]) => {
        expect(fields.some(([k, v]) => k === col && v === type)).to.be.true;
      });
    });

    // check user default data
    cy.data("users");
    cy.sortBy(0);

    cy.addRow({
      id: "5",
      username: "alice_wonder",
      email: "alice@example.com",
      password: "securepassword2",
      created_at: "2022-02-02"
    });

    cy.deleteRow(3);

    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "",
        "id",
        "username",
        "email",
        "password",
        "created_at"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1"
        ],
        [
            "",
            "2",
            "jane_smith",
            "jane@example.com",
            "securepassword2"
        ],
        [
            "",
            "3",
            "admin_user",
            "admin@example.com",
            "adminpass"
        ]
      ]);
    });

    // check page size
    cy.setTablePageSize(1);

    // check total count

    // check page size
    cy.setTablePageSize(1);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1",
        ],
      ]);
    });

    // check conditions
    // todo: check all types
    cy.whereTable([
      ["id", "=", "3"],
    ]);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
          "",
          "3",
          "admin_user",
          "admin@example.com",
          "adminpass",
        ]
      ]);
    });

    // check clearing of the query and page size
    cy.setTablePageSize(10);
    cy.clearWhereConditions();
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(3);
    });
    

    cy.setTablePageSize(2);
    cy.submitTable();
    
    // check editing capability
    // First, update and save the change
    cy.updateRow(1, 1, "jane_smith1", false);
    cy.wait(1000);
    cy.getTableData().then(({ rows }) => {
      // Just check that the update was applied to the second row
      expect(rows[1][2]).to.equal("jane_smith1");
    });
    
    // Revert the change back
    cy.updateRow(1, 1, "jane_smith", false);
    cy.wait(1000);
    cy.getTableData().then(({ rows }) => {
      // Check that the update was reverted
      expect(rows[1][2]).to.equal("jane_smith");
    });

    // Test canceling an edit (should keep original value)
    cy.updateRow(1, 1, "jane_smith_temp");
    cy.wait(1000);
    cy.getTableData().then(({ rows }) => {
      // Check that canceling preserves the original value
      expect(rows[1][2]).to.equal("jane_smith");
    });

    // check search
    cy.searchTable("john");
    cy.wait(250);
    cy.getHighlightedCell().first().should('have.text', 'john_doe');
    cy.searchTable("john");
    cy.wait(250);
    cy.getHighlightedCell().first().should('have.text', 'john@example.com');

    // check graph
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
      // Check type
      expect(fields.some(([k, v]) => k === "Type" && v === "BASE TABLE")).to.be.true;

      // Check Total Size, Data Size, Count (just keys exist)
      expect(fields.some(([k]) => k === "Total Size")).to.be.true;
      expect(fields.some(([k]) => k === "Data Size")).to.be.true;
      expect(fields.some(([k]) => k === "Count")).to.be.true;

      // Check columns and types
      const expectedColumns = [
        ["id", "integer"],
        ["username", "character varying"],
        ["email", "character varying"],
        ["password", "character varying"],
        ["created_at", "timestamp without time zone"]
      ];
      expectedColumns.forEach(([col, type]) => {
        expect(fields.some(([k, v]) => k === col && v === type)).to.be.true;
      });
    });

    // check sql query in scratchpad
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

    cy.writeCode(0, "SELECT * FROM test_schema.users1;");
    cy.runCode(0);
    cy.getCellError(0).then(err => expect(err).to.equal('ERROR: relation "test_schema.users1" does not exist (SQLSTATE 42P01)'));

    cy.writeCode(0, "SELECT * FROM test_schema.users ORDER BY id;");
    cy.runCode(0);
    cy.getCellQueryOutput(0).then(({ rows, columns }) => {
      expect(columns).to.deep.equal([
        "",
        "id",
        "username",
        "email",
        "password",
        "created_at"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1",
        ],
        [
            "",
            "2",
            "jane_smith",
            "jane@example.com",
            "securepassword2",
        ],
        [
            "",
            "3",
            "admin_user",
            "admin@example.com",
            "adminpass",
        ]
      ]);
    });

    cy.writeCode(0, "UPDATE test_schema.users SET username='john_doe1' WHERE id=1");
    cy.runCode(0);
    cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));

    cy.writeCode(0, "UPDATE test_schema.users SET username='john_doe' WHERE id=1");
    cy.runCode(0);
    cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));

    // add cell
    cy.addCell(0);
    cy.writeCode(1, "SELECT * FROM test_schema.users WHERE id=1;");
    cy.runCode(1);
    cy.getCellQueryOutput(1).then(({ rows, columns }) => {
      expect(columns).to.deep.equal([
        "",
        "id",
        "username",
        "email",
        "password",
        "created_at"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1",
        ]
      ]);
    });

    // remove first cell
    cy.removeCell(0);

    // ensure the first cell has the second cell data
    cy.getCellQueryOutput(0).then(({ rows, columns }) => {
      expect(columns).to.deep.equal([
        "",
        "id",
        "username",
        "email",
        "password",
        "created_at"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1",
        ]
      ]);
    });

    // logout
    cy.logout();
  });
});
