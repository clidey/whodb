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

import {forEachDatabase, hasFeature} from '../../support/test-runner';

describe('Mock Data Generation', () => {

    // SQL Databases with mock data support
    forEachDatabase('sql', (db) => {
        const supportedTable = db.mockData.supportedTable;
        const tableWithFKs = db.mockData.tableWithFKs;

        it('shows mock data generation UI', () => {
            cy.data(supportedTable);

            cy.selectMockData();

            // Verify dialog and note appeared
            cy.get('[data-testid="mock-data-sheet"]').should('be.visible');
            cy.contains('Note').should('be.visible');
            cy.get('[data-testid="mock-data-generate-button"]').should('be.visible');

            // Close dialog
            cy.get('body').type('{esc}');
        });

        it('enforces maximum row count limit', () => {
            cy.data(supportedTable);

            cy.selectMockData();

            // Try to exceed max count (should clamp to 200)
            cy.get('[data-testid="mock-data-rows-input"]').as('rowsInput');
            cy.get('@rowsInput').clear().type('300');
            cy.get('@rowsInput').invoke('val').then((val) => {
                expect(parseInt(val, 10)).to.be.equal(200);
            });

            cy.get('body').type('{esc}');
        });

        it('shows overwrite confirmation dialog', () => {
            cy.data(supportedTable);

            cy.selectMockData();

            // Set row count
            cy.setMockDataRows(10);

            // Switch to Overwrite mode
            cy.setMockDataHandling('overwrite');

            // Click Generate
            cy.generateMockData();

            // Should show confirmation
            cy.get('[data-testid="mock-data-overwrite-button"]').should('be.visible');

            // Cancel instead of confirming
            cy.get('body').type('{esc}');
        });

        it('generates mock data and adds rows to table', () => {
            cy.data(supportedTable);

            // Get initial row count
            cy.get('table tbody tr').its('length').then(initialCount => {
                cy.selectMockData();

                // Generate 5 rows (append mode - default)
                cy.setMockDataRows(5);
                cy.generateMockData();

                // Wait for success toast
                cy.contains('Successfully Generated', { timeout: 30000 }).should('be.visible');

                // Sheet should close after success
                cy.get('[data-testid="mock-data-sheet"]').should('not.exist');

                // Verify rows were added (may need to account for pagination)
                cy.get('table tbody tr').its('length').should('be.gte', initialCount);
            });
        });

        // Skip FK dependency test for databases without FK support (e.g., ClickHouse)
        if (tableWithFKs) {
            it('shows dependency preview for tables with foreign keys', () => {
                cy.data(tableWithFKs);

                cy.selectMockData();

                // Set row count to trigger dependency analysis
                cy.setMockDataRows(10);

                // Should show dependency preview with parent tables
                cy.contains('Tables to populate').should('be.visible');
                cy.contains('Total:').should('be.visible');

                // Generate button should be enabled (FK tables now supported)
                cy.get('[data-testid="mock-data-generate-button"]').should('not.be.disabled');

                cy.get('body').type('{esc}');
            });

            it('generates mock data for FK table and populates parent tables', () => {
                cy.data(tableWithFKs);

                cy.selectMockData();

                // Generate rows for FK table
                cy.setMockDataRows(5);
                cy.generateMockData();

                // Wait for success toast - FK generation may take longer
                cy.contains('Successfully Generated', { timeout: 60000 }).should('be.visible');

                // Sheet should close after success
                cy.get('[data-testid="mock-data-sheet"]').should('not.exist');

                // Verify the FK table has data
                cy.get('table tbody tr').should('have.length.gte', 1);
            });

            // Edge case: Low row count with FK tables (bug fix verification)
            // Previously failed with FK constraint error when generating < 5 rows
            it('generates low row count (1-3 rows) for FK table successfully', () => {
                cy.data(tableWithFKs);

                cy.selectMockData();

                // Generate only 2 rows - previously this would fail with FK constraint error
                cy.setMockDataRows(2);
                cy.generateMockData();

                // Should succeed without FK constraint errors
                cy.contains('Successfully Generated', { timeout: 60000 }).should('be.visible');

                // Sheet should close
                cy.get('[data-testid="mock-data-sheet"]').should('not.exist');

                // Table should have data
                cy.get('table tbody tr').should('have.length.gte', 1);
            });

            // Comprehensive FK verification: checks row counts and FK->PK relationships
            it('verifies correct row counts and FK references after generation', () => {
                const fkRelationship = db.mockData.fkRelationships?.[tableWithFKs];
                if (!fkRelationship) {
                    cy.log('Skipping: No FK relationship defined in fixture');
                    return;
                }

                const { parentTable, fkColumn, parentPkColumn } = fkRelationship;
                const rowsToGenerate = 5;
                let initialParentCount = 0;
                let expectedParentRows = 0;

                // Step 1: Get initial parent table row count
                cy.data(parentTable);
                cy.get('table tbody tr').then($rows => {
                    initialParentCount = $rows.length;
                    cy.log(`Initial parent table (${parentTable}) count: ${initialParentCount}`);
                });

                // Step 2: Navigate to FK table and start mock data generation
                cy.data(tableWithFKs);
                cy.selectMockData();
                cy.setMockDataRows(rowsToGenerate);

                // Step 3: Parse the dependency preview to get expected parent row count
                cy.contains('Tables to populate').should('be.visible');
                cy.get('[data-testid="mock-data-sheet"]').within(() => {
                    // Find the parent table row in the dependency preview
                    cy.contains(parentTable).parent().then($row => {
                        const rowText = $row.text();
                        // Extract the row count from text like "users: 1 row" or "users: 5 rows"
                        const match = rowText.match(/(\d+)\s*rows?/);
                        if (match) {
                            expectedParentRows = parseInt(match[1], 10);
                            cy.log(`Expected new parent rows: ${expectedParentRows}`);
                        }
                    });
                });

                // Step 4: Generate mock data (overwrite mode to ensure clean state)
                cy.setMockDataHandling('overwrite');
                cy.generateMockData();
                cy.get('[data-testid="mock-data-overwrite-button"]').click();
                cy.contains('Successfully Generated', { timeout: 60000 }).should('be.visible');

                // Step 5: Verify parent table has the expected row count
                cy.data(parentTable);
                cy.get('table tbody tr').should($rows => {
                    const newCount = $rows.length;
                    cy.log(`New parent table count: ${newCount}, expected: ${initialParentCount + expectedParentRows}`);
                    // Parent should have at least the expected new rows
                    // (may have more if existing data wasn't cleared)
                    expect(newCount).to.be.gte(expectedParentRows);
                });

                // Step 6: Collect parent PKs
                const parentPKs = new Set();
                cy.get('table thead th').then($headers => {
                    // Find the PK column index
                    let pkColIndex = -1;
                    $headers.each((i, th) => {
                        if (Cypress.$(th).text().trim().toLowerCase() === parentPkColumn.toLowerCase()) {
                            pkColIndex = i;
                        }
                    });

                    if (pkColIndex === -1) {
                        cy.log(`Warning: Could not find PK column ${parentPkColumn}`);
                        return;
                    }

                    // Collect all PK values
                    cy.get('table tbody tr').each($row => {
                        const pkValue = Cypress.$($row).find('td').eq(pkColIndex).text().trim();
                        if (pkValue) {
                            parentPKs.add(pkValue);
                        }
                    }).then(() => {
                        cy.log(`Parent PKs collected: ${parentPKs.size} values`);
                        cy.wrap(Array.from(parentPKs)).as('parentPKs');
                    });
                });

                // Step 7: Navigate to FK table and verify FK values exist in parent PKs
                cy.data(tableWithFKs);
                cy.get('table tbody tr').should('have.length', rowsToGenerate);

                cy.get('@parentPKs').then(parentPKArray => {
                    const parentPKSet = new Set(parentPKArray);

                    cy.get('table thead th').then($headers => {
                        // Find the FK column index
                        let fkColIndex = -1;
                        $headers.each((i, th) => {
                            if (Cypress.$(th).text().trim().toLowerCase() === fkColumn.toLowerCase()) {
                                fkColIndex = i;
                            }
                        });

                        if (fkColIndex === -1) {
                            cy.log(`Warning: Could not find FK column ${fkColumn}`);
                            return;
                        }

                        // Verify each FK value exists in parent PKs
                        cy.get('table tbody tr').each($row => {
                            const fkValue = Cypress.$($row).find('td').eq(fkColIndex).text().trim();
                            if (fkValue && fkValue !== 'NULL' && fkValue !== '') {
                                expect(parentPKSet.has(fkValue),
                                    `FK value ${fkValue} should exist in parent PKs`).to.be.true;
                            }
                        });
                    });
                });
            });
        }
    }, {features: ['mockData']});

    // Edge case tests for specific bug fixes
    describe('Mock Data Edge Cases', () => {

        // SQLite-specific: dependency preview with empty schema (bug fix)
        forEachDatabase('sql', (db) => {
            const tableWithFKs = db.mockData?.tableWithFKs;

            if (!tableWithFKs) {
                return;
            }

            // Test specifically for SQLite (which has no schema)
            if (db.type === 'Sqlite3') {
                it('shows dependency preview for SQLite (no schema required)', () => {
                    cy.data(tableWithFKs);

                    cy.selectMockData();

                    // Set row count to trigger dependency analysis
                    cy.setMockDataRows(5);

                    // Should show dependency preview even without schema
                    // (Previously SQLite would not show this because schema was required)
                    cy.contains('Tables to populate').should('be.visible');
                    cy.contains('Total:').should('be.visible');

                    // Generate should work
                    cy.get('[data-testid="mock-data-generate-button"]').should('not.be.disabled');

                    cy.get('body').type('{esc}');
                });
            }
        }, {features: ['mockData']});
    });

    // Document Databases - mock data not supported (inverse: runs when feature is NOT present)
    forEachDatabase('document', (db) => {
        if (hasFeature(db, 'mockData')) {
            return; // Only run if mock data is NOT supported
        }

        it('shows not allowed message for document databases', () => {
            cy.data('orders');

            cy.selectMockData();

            cy.generateMockData();

            cy.contains('Mock Data Generation Is Not Allowed for This Table').should('exist');
        });
    });

});
