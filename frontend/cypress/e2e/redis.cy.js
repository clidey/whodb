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

describe('Redis E2E test', () => {
    const isDocker = Cypress.env('isDocker');
    const dbHost = isDocker ? 'e2e_redis' : 'localhost';
    const dbPassword = 'password';


    it('runs full Redis E2E flow', () => {
        // login and setup
        cy.login('Redis', dbHost, undefined, dbPassword, undefined);

        // 1) Lists keys
        cy.getTables().then(storageUnitNames => {
            expect(storageUnitNames).to.be.an('array');
            const expectedKeys = [
                "bestsellers", "cart:user:1", "category:accessories", "category:computers",
                "category:electronics", "inventory:product:1", "inventory:product:2",
                "inventory:product:3", "inventory:product:4", "inventory:product:5",
                "order:1", "order:1:items", "order:2", "order:2:items", "order:3",
                "order:3:items", "order:4", "order:4:items", "order:5", "order:5:items",
                "order_item:1", "order_item:10", "order_item:11", "order_item:2",
                "order_item:3", "order_item:4", "order_item:5", "order_item:6",
                "order_item:7", "order_item:8", "order_item:9", "orders:recent",
                "payment:1", "payment:2", "payment:3", "payments:by_date", "product:1",
                "product:1:views", "product:2", "product:2:views", "product:3",
                "product:3:views", "product:4", "product:4:views", "product:5",
                "product:5:views", "products:by_price", "search:products", "user:1",
                "user:1:orders", "user:2", "user:2:orders", "user:3", "user:3:orders",
                "user:4", "user:4:orders", "user:5",
            ];
            const filteredKeys = storageUnitNames.filter(key => !key.startsWith('session:'));
            expect(filteredKeys.length).to.be.at.least(expectedKeys.length);
            expectedKeys.forEach(key => {
                expect(filteredKeys).to.include(key);
            });
        });

        // 2) Explore user:1 hash metadata
        cy.explore("user:1");
        cy.getExploreFields().then(fields => {
            expect(fields.some(([k, v]) => k === "Type" && v === "hash")).to.be.true;
            expect(fields.some(([k]) => k === "Size")).to.be.true;
        });

        // 3) Data: check different key types and delete a field from a hash
        cy.data("user:2");
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal(["", "field", "value"]);
            expect(rows.length).to.equal(5);
            expect(rows[0]).to.deep.equal(["", "created_at", "2023-02-20T14:45:00Z"]);
        });
        cy.deleteRow(2);
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(4);
        });

        cy.data("orders:recent");
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal(["", "index", "value"]);
            expect(rows.length).to.be.gt(0);
        });

        cy.data("category:electronics");
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal(["", "index", "value"]);
            const members = rows.map(row => row[2]);
            expect(members).to.include.members(["1", "2", "3", "4", "5"]);
        });

        cy.data("products:by_price");
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal(["", "index", "member", "score"]);
            expect(rows.length).to.be.gt(0);
        });

        cy.data("inventory:product:1");
        cy.getTableData().then(({columns, rows}) => {
            expect(columns).to.deep.equal(["", "value"]);
            expect(rows[0][1]).to.equal("50");
        });

        // 4) Respects page size pagination
        cy.setTablePageSize(2);
        cy.submitTable();
        cy.getTableData().then(({rows}) => {
            expect(rows.length).to.equal(1);
        });

        // 5) Edit value in hash: save, revert, and cancel
        cy.data("user:1");
        cy.updateRow(4, 1, "johndoe_updated", false);
        cy.getTableData().then(({rows}) => {
            expect(rows[4][2]).to.equal("johndoe_updated");
        });
        cy.updateRow(4, 1, "johndoe", false);
        cy.getTableData().then(({rows}) => {
            expect(rows[4][2]).to.equal("johndoe");
        });
        cy.updateRow(4, 1, "johndoe100");
        cy.getTableData().then(({rows}) => {
            expect(rows[4][2]).to.equal("johndoe");
        });

        // 6) Search highlights multiple matches in sequence
        cy.searchTable("john");
        cy.getHighlightedCell().first().should('contain.text', 'john');

        // logout
        cy.logout();
    });
});