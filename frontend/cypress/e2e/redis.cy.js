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
const dbPassword = 'password';

describe('Redis E2E test', () => {
  it('should login correctly', () => {
    // login and setup
    cy.login('Redis', dbHost, '', dbPassword, '');
    
    // get all keys/patterns
    cy.getTables().then(storageUnitNames => {
      cy.log(storageUnitNames);
      expect(storageUnitNames).to.be.an('array');
      // Redis returns keys in different order, so check for inclusion
      const expectedKeys = [
        "bestsellers",
        "cart:user:1",
        "category:accessories",
        "category:computers",
        "category:electronics",
        "inventory:product:1",
        "inventory:product:2", 
        "inventory:product:3",
        "inventory:product:4",
        "inventory:product:5",
        "order:1",
        "order:1:items",
        "order:2",
        "order:2:items",
        "order:3",
        "order:3:items",
        "order:4",
        "order:4:items",
        "order:5",
        "order:5:items",
        "order_item:1",
        "order_item:10",
        "order_item:11",
        "order_item:2",
        "order_item:3",
        "order_item:4",
        "order_item:5",
        "order_item:6",
        "order_item:7",
        "order_item:8",
        "order_item:9",
        "orders:recent",
        "payment:1",
        "payment:2",
        "payment:3",
        "payments:by_date",
        "product:1",
        "product:1:views",
        "product:2",
        "product:2:views",
        "product:3",
        "product:3:views",
        "product:4",
        "product:4:views",
        "product:5",
        "product:5:views",
        "products:by_price",
        "search:products",
        "user:1",
        "user:1:orders",
        "user:2",
        "user:2:orders",
        "user:3",
        "user:3:orders",
        "user:4",
        "user:4:orders",
        "user:5",
      ];
      
      // Filter out session keys as they expire
      const filteredKeys = storageUnitNames.filter(key => !key.startsWith('session:'));
      expect(filteredKeys.length).to.be.at.least(expectedKeys.length);
      expectedKeys.forEach(key => {
        expect(filteredKeys).to.include(key);
      });
    });

    // check user:1 hash
    cy.explore("user:1");
    cy.getExploreFields().then(text => {
      const textLines = text.split("\n");
    
      const expectedPatterns = [
        /^user:1$/,
        /^Type: hash$/,
        /^Size: 5$/,
      ];
      expectedPatterns.forEach(pattern => {
        expect(textLines.some(line => pattern.test(line))).to.be.true;
      });
    });

    // check user data
    cy.data("user:1");
    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "#",
        "field [string]",
        "value [string]"
      ]);
      expect(rows.map(row => row)).to.deep.equal([
        ["1", "created_at", "2023-01-15T10:30:00Z"],
        ["2", "email", "john@example.com"],
        ["3", "id", "1"],
        ["4", "password", "hashed_password_1"],
        ["5", "username", "johndoe"]
      ]);
    });

    // check list data (orders:recent)
    cy.data("orders:recent");
    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "#",
        "index [string]",
        "value [string]"
      ]);
      expect(rows.map(row => row)).to.deep.equal([
        ["1", "0", "5"],
        ["2", "1", "4"],
        ["3", "2", "3"],
        ["4", "3", "2"],
        ["5", "4", "1"]
      ]);
    });

    // check set data (category:electronics)
    cy.data("category:electronics");
    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "#",
        "index [string]",
        "value [string]"
      ]);
      // Sets don't have guaranteed order, so just check members exist
      const members = rows.map(row => row[2]);
      expect(members).to.include.members(["1", "2", "3", "4", "5"]);
    });

    // check sorted set data (products:by_price)
    cy.data("products:by_price");
    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "#",
        "index [string]",
        "member [string]",
        "score [string]"
      ]);
      expect(rows.map(row => row)).to.deep.equal([
        ["1", "0", "2", "29.99"],
        ["2", "1", "3", "79.99"],
        ["3", "2", "5", "199.99"],
        ["4", "3", "4", "399.99"],
        ["5", "4", "1", "999.99"]
      ]);
    });

    // check string data (inventory:product:1)
    cy.data("inventory:product:1");
    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "#",
        "value [string]"
      ]);
      expect(rows.map(row => row)).to.deep.equal([
        ["1", "50"]
      ]);
    });

    // check pagination
    cy.setTablePageSize(2);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(1); // String type only has 1 row
    });

    // check conditions/filters
    cy.data("user:1");
    // todo: implement whereTable for redis
    // cy.whereTable([
    //   ["field", "=", "username"],
    // ]);
    // cy.submitTable();
    // cy.getTableData().then(({ rows }) => {
    //   expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
    //     ["1", "username", "johndoe"]
    //   ]);
    // });

    // // clear conditions
    // cy.clearWhereConditions();
    // cy.submitTable();
    // cy.getTableData().then(({ rows }) => {
    //   expect(rows.length).to.equal(5);
    // });

    // check editing capability for hash
    cy.updateRow(4, 2, "johndoe_updated", false);
    cy.getTableData().then(({ rows }) => {
      expect(rows[4]).to.deep.equal([
        "", "username", "johndoe_updated"
      ]);
    });
    
    // revert the change
    cy.updateRow(4, 2, "johndoe", false);
    cy.getTableData().then(({ rows }) => {
      expect(rows[4]).to.deep.equal([
        "", "username", "johndoe"
      ]);
    });

    // save the change
    cy.updateRow(4, 2, "johndoe");
    cy.getTableData().then(({ rows }) => {
      expect(rows[4]).to.deep.equal([
        "", "username", "johndoe"
      ]);
    });

    // check search
    cy.searchTable("john");
    cy.wait(250);
    cy.getHighlightedRows().then(rows => {
      expect(rows.length).to.be.at.least(1);
      const userRow = rows.find(row => row[1] === "email");
      expect(userRow).to.exist;
      expect(userRow).to.deep.equal([
        "2", "email", "john@example.com"
      ]);
    });

    // todo: for graph it should try to find the text that it is not supported

    // for redis it should show the text that it is not supported
    // cy.goto("scratchpad");

    // logout
    cy.logout();
  });
});