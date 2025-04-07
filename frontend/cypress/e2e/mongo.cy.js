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

describe('MongoDB E2E test', () => {
  it('should login correctly', () => {
    // login and setup
    cy.login('MongoDB', 'localhost', 'user', 'password');
    cy.selectSchema("test_db");
    
    // get all çollections
    cy.getTables().then(storageUnitNames => {
      expect(storageUnitNames).to.be.an('array');
      expect(storageUnitNames).to.deep.equal([
        "order_items",
        "order_summary",
        "orders",
        "payments",
        "products",
        "system.views",
        "users",
      ]);
    });

    // check users table and fields
    cy.explore("users");
    cy.getExploreFields().then(text => {
      const textLines = text.split("\n");
    
      const expectedPatterns = [
        /^users$/,
        /^Type: Collection$/,
        /^Storage Size: .+$/, // Ignores actual size value
        /^Count: .+$/,      // Ignores actual count value
      ];
      expectedPatterns.forEach(pattern => {
        expect(textLines.some(line => pattern.test(line))).to.be.true;
      });
    });

    // check user default data
    cy.data("users");
    cy.sortBy(0);
    
    const expectedData = [
      {
        _id: undefined, // only used to update and not needed
        email: "john@example.com",
        password: "securepassword1",
        username: "john_doe",
      },
      {
        _id: undefined, // only used to update and not needed
        email: "jane@example.com",
        password: "securepassword2",
        username: "jane_smith",
      },
      {
        _id: undefined, // only used to update and not needed
        email: "admin@example.com",
        password: "adminpass",
        username: "admin_user",
      }
    ];
    
    function validateRow(row, expected, expectedIndex) {
      const [rowIndex, rawJson] = row;
      const json = JSON.parse(rawJson);
      if (expectedData[expectedIndex-1]._id == null) {
        expectedData[expectedIndex-1]._id = json["_id"];
      }
      expect(rowIndex).to.equal(expectedIndex.toString());
      expect(json.email).to.equal(expected.email);
      expect(json.password).to.equal(expected.password);
      expect(json.username).to.equal(expected.username);
    }

    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "#",
        "document [Document]"
      ]);
    
      rows.forEach((row, index) => {
        const expected = expectedData[index];
        validateRow(row, expected, index + 1);
      });
    
      // Now that all _id values are captured, use them dynamically
      cy.setTablePageSize(1);
      cy.submitTable();
      cy.getTableData().then(({ rows }) => {
        validateRow(rows[0], expectedData[0], 1);
      });
    
      cy.whereTable([
        ["_id", "eq", expectedData[0]._id],
      ]);
      cy.submitTable();
      cy.getTableData().then(({ rows }) => {
        validateRow(rows[0], expectedData[0], 1);
      });
    
      cy.setTablePageSize(10);
      cy.clearWhereConditions();
      cy.submitTable();
      cy.getTableData().then(({ rows }) => {
        expect(rows.length).to.equal(3);
        rows.forEach((row, index) => {
          validateRow(row, expectedData[index], index + 1);
        });
      });
    
      // Editing check
      cy.setTablePageSize(2);
      cy.submitTable();
    
      cy.updateRow(1, 1, JSON.stringify({
        _id: expectedData[1]._id,
        created_at: "2025-02-22T10:42:06.577Z",
        email: expectedData[1].email,
        password: expectedData[1].password,
        username: "jane_smith1"
      }), false);
    
      cy.getTableData().then(({ rows }) => {
        const updated = { ...expectedData[1], username: "jane_smith1" };
        const row = rows[1];
        const [_, rawJson] = row;
        const json = JSON.parse(rawJson);
        expect(json.username).to.equal(updated.username);
      });
    
      cy.updateRow(1, 1, JSON.stringify(expectedData[1]), false);
      cy.getTableData().then(({ rows }) => {
        const row = rows[1];
        const [_, rawJson] = row;
        const json = JSON.parse(rawJson);
        expect(json.username).to.equal(expectedData[1].username);
      });
    
      cy.updateRow(1, 1, JSON.stringify({
        ...expectedData[1],
        username: "jane_smith1"
      }));
      cy.getTableData().then(({ rows }) => {
        const row = rows[1];
        const [_, rawJson] = row;
        const json = JSON.parse(rawJson);
        // Even though we updated, we are expecting the username to still be "jane_smith"
        expect(json.username).to.equal(expectedData[1].username);
      });
    
      // Search
      cy.searchTable("john");
      cy.wait(250);
      cy.getHighlightedRows().then(rows => {
        expect(rows.length).to.equal(1);
        validateRow(rows[0], expectedData[0], 1);
      });
    
      // Graph
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
    
      cy.getGraphNode().then(text => {
        const textLines = text.split("\n");
        const expectedPatterns = [
          /^users$/,
          /^Type: Collection$/,
          /^Storage Size: .+$/,
          /^Count: .+$/
        ];
        expectedPatterns.forEach(pattern => {
          expect(textLines.some(line => pattern.test(line))).to.be.true;
        });
      });
    
      // Logout
      cy.logout();
    });
  });
});
