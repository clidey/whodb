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
import {getDocumentId, parseDocument} from '../../support/categories/document';

/**
 * Get the operator for a given database
 * @param {Object} db - Database configuration
 * @param {string} operatorKey - Operator key (e.g., 'equals', 'notEquals')
 * @returns {string} Operator string for this database
 */
function getOperator(db, operatorKey) {
    if (db.whereOperators && db.whereOperators[operatorKey]) {
        return db.whereOperators[operatorKey];
    }
    // Default fallbacks
    const defaults = {
        equals: '=',
        notEquals: '!=',
        greaterThan: '>',
        lessThan: '<',
    };
    return defaults[operatorKey] || '=';
}

describe('Where Conditions', () => {

    // SQL Databases
    forEachDatabase('sql', (db) => {
        if (!hasFeature(db, 'whereConditions')) {
            return;
        }

        const testTable = db.testTable || {
            name: 'users',
            idField: 'id',
            identifierField: 'username',
            firstName: 'john_doe',
            identifierColIndex: 1
        };
        const tableName = testTable.name;
        const idField = testTable.idField || 'id';
        const nameField = testTable.identifierField || 'username';
        const firstName = testTable.firstName || 'john_doe';
        const colIndex = testTable.identifierColIndex || 1;
        const eq = getOperator(db, 'equals');

        const whereConfig = testTable.whereConditions || {};
        const testId1 = whereConfig.testId1 || '1';
        const testId2 = whereConfig.testId2 || '2';
        const testId3 = whereConfig.testId3 || '3';

        it('applies where condition and filters data', () => {
            cy.data(tableName);

            cy.whereTable([[idField, eq, testId3]]);
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.equal(1);
                expect(rows[0][1]).to.equal(testId3); // id column
            });

            // Clear conditions
            cy.clearWhereConditions();
            cy.submitTable();

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.be.greaterThan(1);
            });
        });

        it('applies multiple conditions', () => {
            cy.setWhereConditionMode('sheet');
            cy.data(tableName);

            cy.whereTable([
                [idField, eq, testId1],
                [nameField, eq, firstName],
            ]);
            cy.submitTable();

            cy.getConditionCount().should('equal', 2);

            cy.getTableData().then(({rows}) => {
                expect(rows.length).to.equal(1);
                expect(rows[0][colIndex + 1]).to.equal(firstName);
            });

            cy.clearWhereConditions();
            cy.submitTable();
        });

        it('edits existing condition', () => {
            cy.data(tableName);

            cy.whereTable([[idField, eq, testId1]]);
            cy.submitTable();

            cy.getWhereConditionMode().then(mode => {
                if (mode === 'popover') {
                    cy.clickConditionToEdit(0);
                    cy.updateConditionValue(testId2);
                    cy.get('[data-testid="update-condition-button"]').click();
                    cy.submitTable();

                    cy.verifyCondition(0, `${idField} ${eq} ${testId2}`);

                    cy.getTableData().then(({rows}) => {
                        expect(rows[0][1]).to.equal(testId2);
                    });
                }
            });

            cy.clearWhereConditions();
            cy.submitTable();
        });

        it('removes individual condition', () => {
            cy.data(tableName);

            cy.whereTable([
                [idField, eq, testId1],
                [nameField, eq, firstName],
            ]);
            cy.submitTable();

            cy.getConditionCount().should('equal', 2);

            cy.removeCondition(1);
            cy.submitTable();

            cy.getConditionCount().should('equal', 1);

            cy.clearWhereConditions();
            cy.submitTable();
        });

        it('shows more conditions button when exceeding visible limit', () => {
            cy.data(tableName);

            // Clear any existing conditions first
            cy.clearWhereConditions();
            cy.wait(500);

            const neq = getOperator(db, 'notEquals');

            // Get config-driven values for third condition
            const whereConfig = testTable.whereConditions || {};
            const thirdCol = whereConfig.thirdColumn || 'email';
            const thirdVal = whereConfig.thirdValue || 'jane@example.com';
            const expectedName = whereConfig.expectedValue || 'admin_user';

            // Add 3 conditions - should show "+1 more" button
            cy.whereTable([
                [idField, eq, '3'],
                [nameField, eq, expectedName],
                [thirdCol, neq, thirdVal],
            ]);

            cy.getWhereConditionMode().then(mode => {
                if (mode === 'popover') {
                    // Should show first 2 conditions as badges
                    cy.getConditionCount().should('equal', 3);
                    cy.verifyCondition(0, `${idField} ${eq} 3`);
                    cy.verifyCondition(1, `${nameField} ${eq} ${expectedName}`);

                    // Check for more conditions button
                    cy.checkMoreConditionsButton('+1 more');

                    // Click to open sheet with all conditions
                    cy.clickMoreConditions();

                    // Remove conditions in sheet
                    cy.removeConditionsInSheet(true);
                    cy.saveSheetChanges();

                    // After closing sheet, should have only 1 condition
                    cy.getConditionCount().should('equal', 1);
                    cy.verifyCondition(0, `${idField} ${eq} 3`);
                } else {
                    // In sheet mode, just verify count
                    cy.getConditionCount().should('equal', 3);
                }
            });

            cy.submitTable();
            cy.getTableData().then(({rows}) => {
                expect(rows[0][colIndex + 1]).to.equal(expectedName);
            });

            cy.clearWhereConditions();
            cy.submitTable();
        });

        it('cancels condition edit', () => {
            cy.data(tableName);

            cy.whereTable([[idField, eq, '1']]);
            cy.submitTable();

            cy.getWhereConditionMode().then(mode => {
                if (mode === 'popover') {
                    cy.clickConditionToEdit(0);
                    cy.updateConditionValue('2');
                    cy.get('[data-testid="cancel-button"]').click();

                    // Condition should remain unchanged
                    cy.verifyCondition(0, `${idField} ${eq} 1`);
                }
            });

            cy.clearWhereConditions();
            cy.submitTable();
        });
    });

    // Document Databases
    forEachDatabase('document', (db) => {
        if (!hasFeature(db, 'whereConditions')) {
            return;
        }

        const eq = getOperator(db, 'equals');

        it('applies where condition and filters documents', () => {
            cy.data('users');
            cy.sortBy(0);

            const refreshDelay = db.indexRefreshDelay || 0;

            cy.getTableData().then(({rows}) => {
                // Use _id field for filtering (works for both Elasticsearch and MongoDB)
                const firstDocId = getDocumentId(rows[0]);

                cy.whereTable([['_id', eq, firstDocId]]);
                cy.submitTable();

                // Wait for query to process
                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                }

                // Use Cypress retry to wait for filtered results
                cy.get('table:visible tbody tr', {timeout: 10000}).should('have.length', 1);

                cy.getTableData().then(({rows: filteredRows}) => {
                    expect(filteredRows.length).to.equal(1);
                    expect(getDocumentId(filteredRows[0])).to.equal(firstDocId);
                });

                cy.clearWhereConditions();
                cy.submitTable();

                // Wait for data to reload after clearing conditions
                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                }

                cy.getTableData().then(({rows: clearedRows}) => {
                    expect(clearedRows.length).to.be.greaterThan(1);
                });
            });
        });

        it('applies multiple conditions on documents', () => {
            cy.setWhereConditionMode('sheet');
            cy.data('users');
            cy.sortBy(0);

            const refreshDelay = db.indexRefreshDelay || 0;

            cy.getTableData().then(({rows}) => {
                // Use _id for exact matching (works for both Elasticsearch and MongoDB)
                const firstDocId = getDocumentId(rows[0]);
                const firstDoc = parseDocument(rows[0]);

                cy.whereTable([
                    ['_id', eq, firstDocId],
                    ['username', eq, firstDoc.username],
                ]);
                cy.submitTable();

                // Wait for query to process
                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                }

                cy.getConditionCount().should('equal', 2);

                // Wait for filtered results
                cy.get('table:visible tbody tr', {timeout: 10000}).should('have.length.at.least', 1);

                cy.getTableData().then(({rows: filtered}) => {
                    const doc = parseDocument(filtered[0]);
                    expect(doc.username).to.equal(firstDoc.username);
                });

                cy.clearWhereConditions();
                cy.submitTable();

                // Wait for data to reload after clearing
                if (refreshDelay > 0) {
                    cy.wait(refreshDelay);
                }
            });
        });
    });

});
