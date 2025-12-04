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

describe('Data Export', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        if (!hasFeature(db, 'export')) {
            return;
        }

        describe('Export All', () => {
            it('exports table data as CSV with default comma delimiter', () => {
                cy.data('users');
                cy.intercept('POST', '/api/export').as('export');

                // Use Export All button
                cy.contains('button', 'Export All').click();
                cy.contains('h2', 'Export Data').should('be.visible');

                // Verify default format is CSV with comma delimiter
                cy.get('[role="dialog"]').within(() => {
                    cy.contains('label', 'Format').parent().find('[role="combobox"]').first().should('contain.text', 'CSV');
                    cy.contains('label', 'Delimiter').parent().find('[role="combobox"]').first().should('contain.text', 'Comma');
                });

                // Export
                cy.get('[role="dialog"]').within(() => {
                    cy.contains('button', 'Export').click();
                });

                cy.wait('@export').then(({response}) => {
                    expect(response?.statusCode).to.equal(200);
                    const headers = response?.headers || {};
                    const cd = headers['content-disposition'] || headers['Content-Disposition'];
                    expect(cd).to.be.a('string');
                    expect(cd).to.match(/\.csv/i);
                });

                cy.get('body').type('{esc}');
                cy.get('[role="dialog"]').should('not.exist');
            });

            it('exports table data as Excel', () => {
                cy.data('users');
                cy.intercept('POST', '/api/export').as('export');

                cy.contains('button', 'Export All').click();
                cy.contains('h2', 'Export Data').should('be.visible');

                // Change format to Excel
                cy.get('[role="dialog"]').within(() => {
                    cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
                });
                cy.get('[role="listbox"]').should('be.visible');
                cy.get('[role="option"]').contains('Excel').click({force: true});
                cy.get('[role="listbox"]').should('not.exist');

                // Verify Excel description shows (locale uses capital F in Format)
                cy.contains('Excel XLSX Format').should('be.visible');

                cy.get('[role="dialog"]').within(() => {
                    cy.contains('button', 'Export').click();
                });

                cy.wait('@export').then(({response}) => {
                    expect(response?.statusCode).to.equal(200);
                    const headers = response?.headers || {};
                    const cd = headers['content-disposition'] || headers['Content-Disposition'];
                    expect(cd).to.be.a('string');
                    expect(cd).to.match(/\.xlsx/i);
                });

                cy.get('body').type('{esc}');
                cy.get('[role="dialog"]').should('not.exist');
            });
        });

        describe('Export Selected Rows', () => {
            it('exports selected rows with pipe delimiter', () => {
                cy.data('users');
                cy.intercept('POST', '/api/export').as('export');

                // Wait for table to stabilize after data load
                cy.get('table tbody tr').should('have.length.at.least', 1);
                cy.wait(1000); // Wait for any re-renders to complete

                // Select a row via context menu - click on first visible data cell
                cy.get('table tbody tr').first().find('td').eq(1).as('targetCell');
                cy.get('@targetCell').scrollIntoView();
                cy.get('@targetCell').rightclick({position: 'left'});
                cy.wait(500); // Wait for context menu animation
                cy.contains('Select Row').should('be.visible').click({force: true});
                cy.wait(300); // Wait for selection to register

                // Verify row was selected - button should change to "Export 1 Selected"
                cy.contains('button', 'Export 1 Selected').should('be.visible').click();

                cy.get('[role="dialog"]').should('be.visible');
                // Note: UI shows {count} with braces due to translation format
                cy.contains('You are about to export {1} selected rows.').should('be.visible');

                // Ensure CSV format is selected
                cy.get('[role="dialog"]').within(() => {
                    cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
                });
                cy.get('[role="listbox"]').should('be.visible');
                cy.get('[role="option"]').first().click({force: true});

                // Change delimiter to pipe
                cy.get('[role="dialog"]').within(() => {
                    cy.contains('label', 'Delimiter').parent().within(() => {
                        cy.get('[role="combobox"]').eq(-1).click();
                    });
                });
                cy.get('[role="listbox"]').should('be.visible');
                cy.get('[role="option"]').eq(2).click({force: true});
                cy.get('[role="listbox"]').should('not.exist');

                // Verify pipe delimiter selected
                cy.get('[role="dialog"]').within(() => {
                    cy.contains('label', 'Delimiter').parent().find('[role="combobox"]').eq(-1).invoke('text').should('include', '|');
                });

                cy.get('[role="dialog"]').within(() => {
                    cy.contains('button', 'Export').click();
                });

                cy.wait('@export').then(({request, response}) => {
                    expect(response?.statusCode).to.equal(200);
                    expect(request.body.delimiter).to.equal('|');
                    expect(request.body.selectedRows).to.exist;
                    expect(Array.isArray(request.body.selectedRows)).to.be.true;
                    expect(request.body.selectedRows.length).to.be.greaterThan(0);
                });

                cy.get('[role="dialog"]').should('not.exist');
            });
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        if (!hasFeature(db, 'export')) {
            return;
        }

        it('exports collection/index data as JSON', () => {
            cy.data('users');
            cy.intercept('POST', '/api/export').as('export');

            cy.contains('button', 'Export All').click();
            cy.contains('h2', 'Export Data').should('be.visible');

            // Select JSON format
            cy.get('[role="dialog"]').within(() => {
                cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
            });
            cy.get('[role="listbox"]').should('be.visible');
            cy.get('[role="option"]').contains('JSON').click({force: true});

            cy.get('[role="dialog"]').within(() => {
                cy.contains('button', 'Export').click();
            });

            cy.wait('@export').then(({response}) => {
                expect(response?.statusCode).to.equal(200);
            });

            cy.get('body').type('{esc}');
            cy.get('[role="dialog"]').should('not.exist');
        });
    });

});
