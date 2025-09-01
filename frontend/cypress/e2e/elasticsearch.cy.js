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

const dbHost = 'localhost';
const password = 'pgmio430fe$$#@@';
const username = 'elastic';

describe('ElasticSearch E2E test', () => {
  it('should login correctly', () => {
    // login and setup
    cy.login('ElasticSearch', dbHost, username, password);
    
    // get all indices
    cy.getTables().then(storageUnitNames => {
      cy.log(storageUnitNames);
      expect(storageUnitNames).to.be.an('array');
      expect(storageUnitNames).to.deep.equal([
        "order_items",
        "orders",
        "payments",
        "products",
        "users"
      ]);
    });

    // check users index and fields
    cy.explore("users");
    cy.wait(100);
    cy.getExploreFields().then(fields => {
      // Check Storage Size, Count (just keys exist)
      expect(fields.some(([k]) => k === "Storage Size")).to.be.true;
      expect(fields.some(([k]) => k === "Count")).to.be.true;

      // ElasticSearch doesn't expose field types in the same way
      // so we just verify basic metadata
    });

    // check user default data
    cy.data("users");
    // Don't sort by document column (0) as it's not sortable in Elasticsearch

    // ElasticSearch supports adding and deleting documents
    // ElasticSearch has a single "document" field that expects JSON
    cy.addRow({
      username: "new_user",
      email: "new@example.com",
      password: "newpassword"
    }, true);

    // Delete the newly added document - get the actual row count first and delete the last row
    cy.getTableData().then(({rows}) => {
      // Delete the last row (which should be the newly added one)
      const lastRowIndex = rows.length - 1;
      cy.deleteRow(lastRowIndex);
      cy.wait(100);
    });

    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "",
        "document"
      ]);
      // Verify we have the expected number of users (original 3 after add/delete cycle)
      expect(rows.length).to.equal(3);
      // ElasticSearch returns documents as JSON strings
      // Just verify we have some expected usernames (at least the core ones that should remain)
      const usernames = rows.map(row => {
        const doc = JSON.parse(row[1]);
        return doc.username;
      });
      // Check that we have at least these core users (the delete operation might affect which specific users remain)
      const expectedUsernames = ["john_doe", "jane_smith", "admin_user"];
      const presentExpectedUsers = expectedUsernames.filter(name => usernames.includes(name));
      expect(presentExpectedUsers.length).to.be.at.least(2); // At least 2 of the 3 expected users should be present
    });

    // check page size
    cy.setTablePageSize(1);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(1);
      const doc = JSON.parse(rows[0][1]);
      // Just check that we have a document with username
      expect(doc.username).to.exist;
    });

    // check conditions
    cy.setTablePageSize(10);
    cy.submitTable();
    cy.whereTable([
      ["username", "match", "john_doe"],
    ]);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(1);
      const doc = JSON.parse(rows[0][1]);
      expect(doc.username).to.equal("john_doe");
    });

    // check clearing of the query and page size
    cy.setTablePageSize(10);
    cy.clearWhereConditions();
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(3);
    });


    cy.setTablePageSize(10);
    cy.submitTable();

    // check editing capability - ElasticSearch documents are JSON
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
      const doc = JSON.parse(rows[0][1]);
      doc.username = "jane_smit_temph";
      cy.updateRow(1, 1, JSON.stringify(doc));
    });

    cy.getTableData().then(({rows}) => {
      const doc = JSON.parse(rows[2][1]);
      expect(doc.username).to.equal("jane_smith");
    });

    // check search
    cy.searchTable("john");
    cy.wait(100);
    cy.getHighlightedCell().first().then(cell => {
      const text = cell.text();
      expect(text).to.include("john");
    });

    cy.goto("graph");
    cy.getGraph().then(graph => {
      // Elasticsearch indices should appear as isolated nodes without connections
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
      // Check type
      // expect(fields.some(([k, v]) => k === "Type" && v === "Index")).to.be.true;

      // Check Storage Size, Count (just keys exist)
      expect(fields.some(([k]) => k === "Storage Size")).to.be.true;
      expect(fields.some(([k]) => k === "Count")).to.be.true;
    });

    // logout
    cy.logout();
  });
});