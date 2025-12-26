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
        const unsupportedTable = db.mockData.unsupportedTable;

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

        it('prevents mock data on tables with foreign keys', () => {
            cy.data(unsupportedTable);

            cy.selectMockData();

            cy.generateMockData();

            // Should show error about foreign key constraint
            cy.contains('Mock Data Generation Is Not Allowed for This Table').should('exist');
        });
    }, { features: ['mockData'] });

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
