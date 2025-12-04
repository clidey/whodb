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
        if (!hasFeature(db, 'mockData')) {
            return;
        }

        it('shows mock data generation UI', () => {
            cy.data('users');

            cy.selectMockData();

            // Verify dialog and note appeared
            cy.contains('div', 'Mock Data').should('be.visible');
            cy.contains('Note').should('be.visible');
            cy.contains('button', 'Generate').should('be.visible');

            // Close dialog
            cy.get('body').type('{esc}');
        });

        it('enforces maximum row count limit', () => {
            cy.data('users');

            cy.selectMockData();

            // Try to exceed max count (should clamp to 200)
            cy.contains('label', 'Number of Rows').parent().find('input').as('rowsInput');
            cy.get('@rowsInput').clear().type('300');
            cy.get('@rowsInput').invoke('val').then((val) => {
                expect(parseInt(val, 10)).to.be.equal(200);
            });

            cy.get('body').type('{esc}');
        });

        it('shows overwrite confirmation dialog', () => {
            cy.data('users');

            cy.selectMockData();

            // Set row count
            cy.contains('label', 'Number of Rows').parent().find('input').as('rowsInput');
            cy.get('@rowsInput').clear().type('10');

            // Switch to Overwrite mode
            cy.contains('label', 'Data Handling').parent().find('[role="combobox"]').eq(-1).click();
            cy.contains('[role="option"]', 'Overwrite existing data').click();

            // Click Generate
            cy.contains('button', 'Generate').click();

            // Should show confirmation
            cy.contains('button', 'Yes, Overwrite').should('be.visible');

            // Cancel instead of confirming
            cy.get('body').type('{esc}');
        });

        it('prevents mock data on tables with foreign keys', () => {
            cy.data('orders');

            cy.selectMockData();

            cy.contains('button', 'Generate').click();

            // Should show error about foreign key constraint
            cy.contains('Mock data generation is not allowed for this table').should('exist');
        });
    });

    // Document Databases - mock data not supported
    forEachDatabase('document', (db) => {
        if (hasFeature(db, 'mockData')) {
            return; // Only run if mock data is NOT supported
        }

        it('shows not allowed message for document databases', () => {
            cy.data('orders');

            cy.selectMockData();

            cy.contains('button', 'Generate').click();

            cy.contains('Mock data generation is not allowed for this table').should('exist');
        });
    });

});
