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

import {forEachDatabase, getSqlQuery} from '../../support/test-runner';

describe('Query History', () => {

    // SQL Databases with scratchpad support
    forEachDatabase('sql', (db) => {
        it('stores executed queries in history', () => {
            cy.goto('scratchpad');

            // Execute a query
            const query = getSqlQuery(db, 'selectAllUsers');
            cy.writeCode(0, query);
            cy.runCode(0);

            // Open history
            cy.openQueryHistory(0);

            cy.getQueryHistoryItems().then(items => {
                expect(items.length).to.be.greaterThan(0);
                expect(items[0]).to.contain('SELECT');
            });

            cy.closeQueryHistory();
        });

        it('clones query from history to editor', () => {
            cy.goto('scratchpad');

            // Execute first query
            const query1 = getSqlQuery(db, 'selectAllUsers');
            cy.writeCode(0, query1);
            cy.runCode(0);

            // Execute second query
            cy.addCell(0);
            const query2 = getSqlQuery(db, 'countUsers');
            cy.writeCode(1, query2);
            cy.runCode(1);

            // Open history and clone first query
            cy.openQueryHistory(1);
            cy.cloneQueryToEditor(0, 1);

            // Verify cloned
            cy.verifyQueryInEditor(1, 'COUNT');
        });

        it('copies query to clipboard', () => {
            cy.goto('scratchpad');

            const query = getSqlQuery(db, 'selectAllUsers');
            cy.writeCode(0, query);
            cy.runCode(0);

            cy.openQueryHistory(0);
            cy.copyQueryFromHistory(0);
            cy.closeQueryHistory();
        });

        it('executes query directly from history', () => {
            cy.goto('scratchpad');

            const query = getSqlQuery(db, 'selectAllUsers');
            cy.writeCode(0, query);
            cy.runCode(0);

            // Clear editor
            cy.writeCode(0, '-- cleared');

            cy.openQueryHistory(0);
            cy.executeQueryFromHistory(0);
            cy.closeQueryHistory();

            // Verify results appeared
            cy.getCellQueryOutput(0).then(({rows}) => {
                expect(rows.length).to.be.greaterThan(0);
            });
        });
    }, {features: ['queryHistory', 'scratchpad']});

});
