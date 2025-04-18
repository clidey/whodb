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
const dbPassword = 'password';

describe('Clickhouse E2E test', () => {
  it('should login correctly', () => {
    // login and setup
    cy.login('ClickHouse', 'localhost', 'user', 'password', 'test_db');
    cy.selectDatabase("test_db");
    
    // get all tables
    cy.getTables().then(storageUnitNames => {
      expect(storageUnitNames).to.be.an('array');
      const expectedStorageUnits = [
        "order_items",
        "order_summary",
        "orders",
        "payments",
        "products",
        "users"
      ];
      expectedStorageUnits.forEach(item => {
        expect(storageUnitNames).contain(item);
      });
    });

    // check users table and fields
    cy.explore("users");
    cy.getExploreFields().then(text => {
      const textLines = text.split("\n");
    
      const expectedPatterns = [
        /^users$/,
        /^Type: MergeTree$/,
        /^Total Size: .+$/, // Ignores actual size value
        /^Count: .+$/,      // Ignores actual count value
        /^id: UInt32$/,
        /^username: String$/,
        /^email: String$/,
        /^password: String$/,
        /^created_at: DateTime$/
      ];
      expectedPatterns.forEach(pattern => {
        expect(textLines.some(line => pattern.test(line))).to.be.true;
      });
    });

    // check user default data
    cy.data("users");
    cy.sortBy(0);
    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "#",
        "id [UInt32]",
        "username [String]",
        "email [String]",
        "password [String]",
        "created_at [DateTime]"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1",
        ],
        [
            "2",
            "2",
            "jane_smith",
            "jane@example.com",
            "securepassword2",
        ],
        [
            "3",
            "3",
            "admin_user",
            "admin@example.com",
            "adminpass",
        ]
      ]);
    });

    // check total count

    // check page size
    cy.setTablePageSize(1);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
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
          "1",
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
    
    // todo: [NOT PASSING - FIX] check pagination on the bottom
    // cy.getPageNumbers().then(pageNumbers => expect(pageNumbers).to.deep.equal(['1']));
    
    // check editing capability
    cy.setTablePageSize(2);
    cy.submitTable();

    // test saving
    cy.updateRow(1, 2, "jane_smith1", false);
    cy.getTableData().then(({ rows }) => {
      expect(rows.slice(1).map(row => row.slice(0, -1))).to.deep.equal([
        [
          "",
          "2",
          "jane_smith1",
          "jane@example.com",
          "securepassword2",
        ]
      ]);
    });
    cy.updateRow(1, 2, "jane_smith", false);
    cy.getTableData().then(({ rows }) => {
      expect(rows.slice(1).map(row => row.slice(0, -1))).to.deep.equal([
        [
          "",
          "2",
          "jane_smith",
          "jane@example.com",
          "securepassword2",
        ]
      ]);
    });

    cy.updateRow(1, 2, "jane_smith");
    cy.getTableData().then(({ rows }) => {
      expect(rows.slice(1).map(row => row.slice(0, -1))).to.deep.equal([
        [
          "",
          "2",
          "jane_smith",
          "jane@example.com",
          "securepassword2",
        ]
      ]);
    });

    // check search
    cy.searchTable("john");
    cy.wait(250);
    cy.getHighlightedRows().then(rows => {
      expect(rows.length).to.equal(1);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1"
        ]
      ]);
    });

    // check graph
    // cy.goto("graph");
    // cy.getGraph().then(graph => {
    //   const expectedGraph = {
    //     "users": [],
    //     "orders": [],
    //     "order_items": [],
    //     "products": [],
    //     "payments": [],
    //     "order_summary": []
    //   };
    
    //   Object.keys(expectedGraph).forEach(key => {
    //     expect(graph).to.have.property(key);
    //     expect(graph[key].sort()).to.deep.equal(expectedGraph[key].sort());
    //   });
    // });
    // cy.getGraphNode().then(text => {
    //   const textLines = text.split("\n");
    //   const expectedPatterns = [
    //     /^users$/,
    //     /^Type: MergeTree$/,
    //     /^Total Size: .+$/,
    //     /^Count: .+$/,     
    //     /^id: UInt32$/,
    //     /^username: String$/,
    //     /^email: String$/,
    //     /^password: String$/,
    //     /^created_at: DateTime$/
    //   ];
    //   expectedPatterns.forEach(pattern => {
    //     expect(textLines.some(line => pattern.test(line))).to.be.true;
    //   });
    // });

    // check sql query in scratchpad
    cy.goto("scratchpad");
    cy.writeCode(0, "SELECT * FROM test_db.users1;");
    cy.runCode(0);
    cy.getCellError(0).then(err => expect(err).to.equal("code: 60, message: Unknown table expression identifier 'test_db.users1' in scope SELECT * FROM test_db.users1"));
    
    cy.writeCode(0, "SELECT * FROM test_db.users ORDER BY id;");
    cy.runCode(0);
    cy.getCellQueryOutput(0).then(({ rows, columns }) => {
      expect(columns).to.deep.equal([
        "#",
        "id [UInt32]",
        "username [String]",
        "email [String]",
        "password [String]",
        "created_at [DateTime]"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1",
        ],
        [
            "2",
            "2",
            "jane_smith",
            "jane@example.com",
            "securepassword2",
        ],
        [
            "3",
            "3",
            "admin_user",
            "admin@example.com",
            "adminpass",
        ]
      ]);
    });

    // add cell
    cy.addCell(0);
    cy.writeCode(1, "SELECT * FROM test_db.users WHERE id=1;");
    cy.runCode(1);
    cy.getCellQueryOutput(1).then(({ rows, columns }) => {
      expect(columns).to.deep.equal([
        "#",
        "id [UInt32]",
        "username [String]",
        "email [String]",
        "password [String]",
        "created_at [DateTime]"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
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
        "#",
        "id [UInt32]",
        "username [String]",
        "email [String]",
        "password [String]",
        "created_at [DateTime]"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
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
