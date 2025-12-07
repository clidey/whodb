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

import {forEachDatabase, getTableConfig, hasFeature} from '../../support/test-runner';

describe('Graph Visualization', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        if (!hasFeature(db, 'graph')) {
            return;
        }

        const testTable = db.testTable || {name: 'users'};
        const tableName = testTable.name;

        it('displays graph with expected topology', () => {
            cy.goto('graph');

            cy.getGraph().then(graph => {
                if (db.graph && db.graph.expectedNodes) {
                    Object.keys(db.graph.expectedNodes).forEach(node => {
                        expect(graph, `Graph should have node: ${node}`).to.have.property(node);
                        expect(graph[node].sort()).to.deep.equal(
                            db.graph.expectedNodes[node].sort()
                        );
                    });
                }
            });
        });

        it('shows table metadata in graph nodes', () => {
            cy.goto('graph');

            // Wait for graph to render and layout
            cy.get('.react-flow__node', {timeout: 10000}).should('be.visible');
            cy.get('[data-testid="graph-layout-button"]').click();
            cy.wait(1000); // Wait for layout animation

            // Wait for the specific node to exist
            cy.get(`[data-testid="rf__node-${tableName}"]`, {timeout: 10000}).should('exist');

            cy.getGraphNode(tableName).then(fields => {
                const tableConfig = getTableConfig(db, tableName);
                if (tableConfig && tableConfig.metadata) {
                    // Graph nodes only show metadata (Type, Size), not column types
                    if (tableConfig.metadata.type) {
                        expect(fields.some(([k, v]) => k === 'Type' && v === tableConfig.metadata.type),
                            `Should have Type: ${tableConfig.metadata.type}`).to.be.true;
                    }
                    if (tableConfig.metadata.hasSize) {
                        expect(fields.some(([k]) => k === 'Total Size' || k === 'Data Size'),
                            'Should have size info').to.be.true;
                    }
                }
            });
        });

        it('can navigate from graph node to data view', () => {
            cy.goto('graph');

            cy.get('.react-flow__node', {timeout: 10000}).should('be.visible');
            cy.get('[data-testid="graph-layout-button"]').click();

            cy.get(`[data-testid="rf__node-${tableName}"] [data-testid="data-button"]`).click({force: true});

            cy.url().should('include', '/storage-unit/explore');
            cy.contains('Total Count:').should('be.visible');
        });
    });

    // Document Databases (MongoDB has graph support)
    forEachDatabase('document', (db) => {
        if (!hasFeature(db, 'graph')) {
            return;
        }

        it('displays graph with expected topology', () => {
            cy.goto('graph');

            cy.getGraph().then(graph => {
                if (db.graph && db.graph.expectedNodes) {
                    Object.keys(db.graph.expectedNodes).forEach(node => {
                        expect(graph).to.have.property(node);
                        expect(graph[node].sort()).to.deep.equal(
                            db.graph.expectedNodes[node].sort()
                        );
                    });
                }
            });
        });

        it('shows collection metadata in graph nodes', () => {
            cy.goto('graph');

            // Wait for graph to render and layout
            cy.get('.react-flow__node', {timeout: 10000}).should('be.visible');
            cy.wait(500); // Wait for layout to stabilize

            // Wait for the specific node to exist
            cy.get('[data-testid="rf__node-users"]', {timeout: 10000}).should('exist');

            cy.getGraphNode('users').then(fields => {
                const tableConfig = getTableConfig(db, 'users');
                if (tableConfig && tableConfig.metadata) {
                    expect(fields.some(([k, v]) => k === 'Type' && v === tableConfig.metadata.type)).to.be.true;
                }
            });
        });
    });

});
