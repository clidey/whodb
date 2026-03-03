/*
 * Copyright 2026 Clidey, Inc.
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

import { test, expect, forEachDatabase } from '../../support/test-fixture.mjs';
import { getTableConfig } from '../../support/database-config.mjs';

test.describe('Graph Visualization', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        const testTable = db.testTable;
        const tableName = testTable.name;

        test('displays graph with expected topology', async ({ whodb, page }) => {
            await whodb.goto('graph');

            const graph = await whodb.getGraph();
            if (db.graph && db.graph.expectedNodes) {
                for (const node of Object.keys(db.graph.expectedNodes)) {
                    expect(graph, `Graph should have node: ${node}`).toHaveProperty(node);
                    expect(graph[node].sort()).toEqual(
                        db.graph.expectedNodes[node].sort()
                    );
                }
            }
        });

        test('shows table metadata in graph nodes', async ({ whodb, page }) => {
            await whodb.goto('graph');

            // Wait for graph to render and layout
            await page.locator('.react-flow__node').first().waitFor({ state: 'visible', timeout: 10000 });
            await page.locator('[data-testid="graph-layout-button"]').click();
            await page.waitForTimeout(1000); // Wait for layout animation

            // Wait for the specific node to exist
            await page.locator(`[data-testid="rf__node-${tableName}"]`).waitFor({ state: 'attached', timeout: 10000 });

            const fields = await whodb.getGraphNode(tableName);
            const tableConfig = getTableConfig(db, tableName);
            if (tableConfig && tableConfig.metadata) {
                // Graph nodes only show metadata (Type, Size), not column types
                if (tableConfig.metadata.type) {
                    expect(fields.some(([k, v]) => k === 'Type' && v === tableConfig.metadata.type),
                        `Should have Type: ${tableConfig.metadata.type}`).toBe(true);
                }
                if (tableConfig.metadata.hasSize) {
                    // Different databases use different size field names
                    const sizeFields = ['Total Size', 'Data Size', 'Size', 'Table Size', 'Segment Size'];
                    expect(fields.some(([k]) => sizeFields.some(sf => k.includes(sf) || k.toLowerCase().includes('size'))),
                        'Should have size info').toBe(true);
                }
            }
        });

        test('can navigate from graph node to data view', async ({ whodb, page }) => {
            await whodb.goto('graph');

            await page.locator('.react-flow__node').first().waitFor({ state: 'visible', timeout: 10000 });
            await page.locator('[data-testid="graph-layout-button"]').click();

            // Hover over the node first to reveal the data button, then click
            await page.locator(`[data-testid="rf__node-${tableName}"]`).hover();
            await page.waitForTimeout(300);
            await page.locator(`[data-testid="rf__node-${tableName}"] [data-testid="data-button"]`).click({ force: true });

            await expect(page).toHaveURL(/\/storage-unit\/explore/, { timeout: 15000 });
            await page.locator('table tbody tr').first().waitFor({ state: 'visible', timeout: 30000 });
        });
    }, { features: ['graph'] });

    // Document Databases (MongoDB has graph support)
    forEachDatabase('document', (db) => {
        test('displays graph with expected topology', async ({ whodb, page }) => {
            await whodb.goto('graph');

            const graph = await whodb.getGraph();
            if (db.graph && db.graph.expectedNodes) {
                for (const node of Object.keys(db.graph.expectedNodes)) {
                    expect(graph).toHaveProperty(node);
                    expect(graph[node].sort()).toEqual(
                        db.graph.expectedNodes[node].sort()
                    );
                }
            }
        });

        test('shows collection metadata in graph nodes', async ({ whodb, page }) => {
            await whodb.goto('graph');

            // Wait for graph to render and layout
            await page.locator('.react-flow__node').first().waitFor({ state: 'visible', timeout: 10000 });
            await page.waitForTimeout(500); // Wait for layout to stabilize

            // Wait for the specific node to exist
            await page.locator('[data-testid="rf__node-users"]').waitFor({ state: 'attached', timeout: 10000 });

            const fields = await whodb.getGraphNode('users');
            const tableConfig = getTableConfig(db, 'users');
            if (tableConfig && tableConfig.metadata) {
                expect(fields.some(([k, v]) => k === 'Type' && v === tableConfig.metadata.type)).toBe(true);
            }
        });
    }, { features: ['graph'] });

});
