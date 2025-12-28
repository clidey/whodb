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

import {forEachDatabase} from '../../support/test-runner';
import {filterSessionKeys} from '../../support/categories/keyvalue';

describe('Storage Unit Listing', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        it('lists expected tables', () => {
            cy.getTables().then(tables => {
                expect(tables).to.be.an('array');
                // Filter out test artifacts (tables like test_table_12345 from mock data tests)
                const filteredTables = tables.filter(t => !t.match(/^test_table_\d+/));
                expect(filteredTables).to.deep.equal(db.expectedTables);
            });
        });
    });

    // Document Databases (MongoDB, Elasticsearch)
    forEachDatabase('document', (db) => {
        it('lists expected collections/indices', () => {
            cy.getTables().then(items => {
                expect(items).to.be.an('array');
                // Create a copy to avoid mutating the config
                let expected = [...(db.expectedIndices || db.expectedTables)];
                // Some databases may include system.views
                if (db.includesSystemViews) {
                    expected.push('system.views');
                    expected.sort();
                }
                expect(items).to.deep.equal(expected);
            });
        });
    });

    // Key-Value Databases (Redis)
    forEachDatabase('keyvalue', (db) => {
        it('lists expected keys', () => {
            cy.getTables().then(keys => {
                expect(keys).to.be.an('array');
                const filteredKeys = filterSessionKeys(keys);
                expect(filteredKeys.length).to.be.at.least(db.expectedKeys.length);
                db.expectedKeys.forEach(key => {
                    expect(filteredKeys).to.include(key);
                });
            });
        });
    });

});
