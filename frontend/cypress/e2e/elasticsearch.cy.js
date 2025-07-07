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
    cy.getExploreFields().then(text => {
      const textLines = text.split("\n");
    
      const expectedPatterns = [
        /^users$/,
        /^Storage Size: \d+$/,
        /^Count: \d+$/,
      ];
      expectedPatterns.forEach(pattern => {
        expect(textLines.some(line => pattern.test(line))).to.be.true;
      });
    });

    // check user default data
    cy.data("users");
    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "#",
        "document [Document]",
      ]);
      expect(rows.length).to.equal(3);
      expect(rows[0]).to.deep.equal([
        "1",
        "{\"_id\":\"1\",\"created_at\":\"2024-01-01T12:00:00\",\"email\":\"john@example.com\",\"password\":\"securepassword1\",\"username\":\"john_doe\"}"
    ]);
      expect(rows[1]).to.deep.equal([
        "2",
        "{\"_id\":\"2\",\"created_at\":\"2024-01-02T12:00:00\",\"email\":\"jane@example.com\",\"password\":\"securepassword2\",\"username\":\"jane_smith\"}"
    ]);
      expect(rows[2]).to.deep.equal([
        "3",
        "{\"_id\":\"3\",\"created_at\":\"2024-01-03T12:00:00\",\"email\":\"admin@example.com\",\"password\":\"adminpass\",\"username\":\"admin_user\"}"
    ]);
    });

    // check page size
    cy.setTablePageSize(1);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(1);
      expect(rows[0]).to.deep.equal([
        "1",
        "{\"_id\":\"1\",\"created_at\":\"2024-01-01T12:00:00\",\"email\":\"john@example.com\",\"password\":\"securepassword1\",\"username\":\"john_doe\"}"
    ]);
    });

    // check conditions
    cy.whereTable([
      ["username", "match", "john_doe"],
    ]);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(1);
      expect(rows[0]).to.deep.equal([
        "1",
        "{\"_id\":\"1\",\"created_at\":\"2024-01-01T12:00:00\",\"email\":\"john@example.com\",\"password\":\"securepassword1\",\"username\":\"john_doe\"}"
    ]);
    });

    // check clearing of the query and page size
    cy.setTablePageSize(10);
    cy.clearWhereConditions();
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(1);
    });

    // check editing capability
    // cy.updateRow(0, 6, "155", false);
    // cy.getTableData().then(({ rows }) => {
    //   expect(rows[0][6]).to.equal("155");
    // });
    
    // revert the change
    // cy.updateRow(0, 6, "150", false);
    // cy.getTableData().then(({ rows }) => {
    //   expect(rows[0][6]).to.equal("150");
    // });

    // save the change
    // cy.updateRow(0, 6, "150");
    // cy.getTableData().then(({ rows }) => {
    //   expect(rows[0][6]).to.equal("150");
    // });

    // check search
    // cy.searchTable("laptop");
    // cy.wait(250);
    // cy.getHighlightedRows().then(rows => {
    //   expect(rows[0]).to.equal("Laptop");
    // });

    // check graph - Elasticsearch doesn't have traditional foreign key relationships
    cy.goto("graph");
    cy.getGraph().then(graph => {
      // Elasticsearch indices should appear as isolated nodes without connections
      const expectedIndices = ["users", "products", "orders", "order_items", "payments"];
      expectedIndices.forEach(index => {
        expect(graph).to.have.property(index);
      });
    });

    // check elasticsearch queries in scratchpad
    // cy.goto("scratchpad");
    
    // For Elasticsearch, we need to select the index context first
    // Let's start with users index
    // cy.data("users");
    // cy.goto("scratchpad");
    
    // test successful query - match all in users
    // cy.writeCode(0, '{"query": {"match_all": {}}}');
    // cy.runCode(0);
    // cy.getCellQueryOutput(0).then(({ rows, columns }) => {
    //   expect(columns).to.deep.equal([
    //     "#",
    //     "_id",
    //     "_score",
    //     "created_at [date]",
    //     "email [keyword]",
    //     "password [text]",
    //     "username [keyword]"
    //   ]);
    //   expect(rows.length).to.equal(3);
    // });

    // test term query
    // cy.writeCode(0, '{"query": {"term": {"username": "john_doe"}}}');
    // cy.runCode(0);
    // cy.getCellQueryOutput(0).then(({ rows }) => {
    //   expect(rows.length).to.equal(1);
    //   expect(rows[0][6]).to.equal("john_doe");
    // });

    // Switch to products index
    // cy.data("products");
    // cy.goto("scratchpad");

    // test range query on products
    // cy.writeCode(0, '{"query": {"range": {"price": {"gte": 200, "lte": 1000}}}}');
    // cy.runCode(0);
    // cy.getCellQueryOutput(0).then(({ rows }) => {
    //   expect(rows.length).to.equal(1);
    //   expect(rows[0][5]).to.equal("Smartphone");
    //   expect(rows[0][6]).to.equal("800");
    // });

    // add cell for aggregation query
    // cy.addCell(0);
    // cy.writeCode(1, '{"aggs": {"avg_price": {"avg": {"field": "price"}}}, "size": 0}');
    // cy.runCode(1);
    // cy.getCellQueryOutput(1).then(({ rows, columns }) => {
    //   // Aggregation results show differently
    //   expect(columns).to.include("#");
    //   expect(rows.length).to.be.greaterThan(0);
    // });

    // test update by query
    // cy.writeCode(1, '{"script": {"source": "ctx._source.stock_quantity += 5"}, "query": {"term": {"name": "Laptop"}}}');
    // cy.runCode(1);
    // cy.getCellActionOutput(1).then(output => expect(output).to.equal('Action Executed'));

    // verify the update worked
    // cy.writeCode(1, '{"query": {"term": {"name": "Laptop"}}}');
    // cy.runCode(1);
    // cy.getCellQueryOutput(1).then(({ rows }) => {
    //   expect(rows.length).to.equal(1);
    //   expect(rows[0][7]).to.equal("15"); // Was 10, added 5
    // });

    // revert the change
    // cy.writeCode(1, '{"script": {"source": "ctx._source.stock_quantity -= 5"}, "query": {"term": {"name": "Laptop"}}}');
    // cy.runCode(1);
    // cy.getCellActionOutput(1).then(output => expect(output).to.equal('Action Executed'));

    // remove first cell
    // cy.removeCell(0);

    // ensure the first cell has the second cell data
    // cy.getCellQueryOutput(0).then(({ rows }) => {
    //   expect(rows.length).to.equal(1);
    //   expect(rows[0][7]).to.equal("10"); // Back to original
    // });

    // logout
    cy.logout();
  });
});