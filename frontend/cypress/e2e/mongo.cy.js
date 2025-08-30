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
    cy.login('MongoDB', dbHost, dbUser, dbPassword);
    cy.selectDatabase("test_db");

    // get all collections
    cy.getTables().then(storageUnitNames => {
      cy.log(storageUnitNames);
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

    // check users collection and fields
    cy.explore("users");
    cy.getExploreFields().then(fields => {
      // For Mongo, fields may be a string or array of [key, value]
      const arr = Array.isArray(fields)
        ? fields
        : (typeof fields === "string"
            ? fields.split("\n").map(line => {
                const idx = line.indexOf(": ");
                if (idx === -1) return [line, ""];
                return [line.slice(0, idx), line.slice(idx + 2)];
              })
            : []);
      // Check type
      expect(arr.some(([k, v]) => k === "Type" && v === "Collection")).to.be.true;

      // Check Storage Size, Count (just keys exist)
      expect(arr.some(([k]) => k === "Storage Size")).to.be.true;
      expect(arr.some(([k]) => k === "Count")).to.be.true;

      // Check columns and types (Mongo doesn't have fixed columns, but check for sample document keys)
      // Not applicable for Mongo, so skip
    });

    // check user default data
    cy.data("users");
    cy.sortBy(0);

    // We'll use the same approach as Postgres: get table data, check columns and rows
    cy.getTableData().then(({ columns, rows }) => {
      console.log(columns);
      console.log(rows);
      expect(columns).to.deep.equal([
        "",
        "document"
      ]);
      // Save _id for each row for later use
      const expectedRows = [
        {
          email: "john@example.com",
          password: "securepassword1",
          username: "john_doe"
        },
        {
          email: "jane@example.com",
          password: "securepassword2",
          username: "jane_smith"
        },
        {
          email: "admin@example.com",
          password: "adminpass",
          username: "admin_user"
        }
      ];
      // Map to store _ids for later
      const rowIds = [];
      rows.forEach((row, idx) => {
        const [_, rawJson] = row;
        const json = JSON.parse(rawJson);
        rowIds.push(json._id);
        expect(json.email).to.equal(expectedRows[idx].email);
        expect(json.password).to.equal(expectedRows[idx].password);
        expect(json.username).to.equal(expectedRows[idx].username);
      });

      // Now that all _id values are captured, use them dynamically
      // check page size
      cy.setTablePageSize(1);
      cy.submitTable();
      cy.getTableData().then(({ rows }) => {
        const [_, rawJson] = rows[0];
        const json = JSON.parse(rawJson);
        expect(json.email).to.equal(expectedRows[0].email);
        expect(json.password).to.equal(expectedRows[0].password);
        expect(json.username).to.equal(expectedRows[0].username);
      });

      // check where condition by _id
      cy.whereTable([
        ["_id", "eq", rowIds[0]],
      ]);
      cy.submitTable();
      cy.getTableData().then(({ rows }) => {
        const [_, rawJson] = rows[0];
        const json = JSON.parse(rawJson);
        expect(json.email).to.equal(expectedRows[0].email);
        expect(json.password).to.equal(expectedRows[0].password);
        expect(json.username).to.equal(expectedRows[0].username);
      });

      // check clearing of the query and page size
      cy.setTablePageSize(10);
      cy.clearWhereConditions();
      cy.submitTable();
      cy.getTableData().then(({ rows }) => {
        expect(rows.length).to.equal(3);
        rows.forEach((row, idx) => {
          const [_, rawJson] = row;
          const json = JSON.parse(rawJson);
          expect(json.email).to.equal(expectedRows[idx].email);
          expect(json.password).to.equal(expectedRows[idx].password);
          expect(json.username).to.equal(expectedRows[idx].username);
        });
      });

      // check editing capability
      cy.setTablePageSize(2);
      cy.submitTable();

      // First, update and save the change
      const updatedJane = {
        _id: rowIds[1],
        email: expectedRows[1].email,
        password: expectedRows[1].password,
        username: "jane_smith1"
      };
      cy.updateRow(1, 0, JSON.stringify(updatedJane), false);
      cy.getTableData().then(({ rows }) => {
        const [_, rawJson] = rows[1];
        const json = JSON.parse(rawJson);
        expect(json.username).to.equal("jane_smith1");
      });

      // Revert the change back
      const revertedJane = {
        _id: rowIds[1],
        email: expectedRows[1].email,
        password: expectedRows[1].password,
        username: "jane_smith"
      };
      cy.updateRow(1, 0, JSON.stringify(revertedJane), false);
      cy.getTableData().then(({ rows }) => {
        const [_, rawJson] = rows[1];
        const json = JSON.parse(rawJson);
        expect(json.username).to.equal("jane_smith");
      });

      // Test canceling an edit (should keep original value)
      const tempJane = {
        _id: rowIds[1],
        email: expectedRows[1].email,
        password: expectedRows[1].password,
        username: "jane_smith_temp"
      };
      cy.updateRow(1, 0, JSON.stringify(tempJane));
      cy.wait(1000);
      cy.getTableData().then(({ rows }) => {
        const [_, rawJson] = rows[1];
        const json = JSON.parse(rawJson);
        expect(json.username).to.equal("jane_smith");
      });

      // Search
      cy.searchTable("john");
      cy.wait(250);
      cy.getHighlightedRows().then(rows => {
        expect(rows.length).to.equal(1);
        const [_, rawJson] = rows[0];
        const json = JSON.parse(rawJson);
        expect(json.email).to.equal("john@example.com");
        expect(json.username).to.equal("john_doe");
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

      cy.getGraphNode("users").then(fields => {
        // Check type
        expect(fields.some(([k, v]) => k === "Type" && v === "Collection")).to.be.true;

        // Check Storage Size, Count (just keys exist)
        expect(fields.some(([k]) => k === "Storage Size")).to.be.true;
        expect(fields.some(([k]) => k === "Count")).to.be.true;
      });

      // Logout
      cy.logout();
    });
  });
});
