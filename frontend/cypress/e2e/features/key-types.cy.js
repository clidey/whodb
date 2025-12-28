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
import {verifyColumnsForType} from '../../support/categories/keyvalue';

/**
 * Key Types Test Suite for Key-Value Databases
 *
 * Tests that Redis correctly handles different key types (string, hash, list,
 * set, zset) with proper column structures and operations where supported.
 *
 * This is analogous to data-types.cy.js for SQL databases, but adapted for
 * the schema-less nature of key-value stores where "types" are determined by
 * the Redis data structure used for each key.
 */
describe('Key Types Operations', () => {

    forEachDatabase('keyvalue', (db) => {
        const keyTypeTests = db.keyTypeTests;

        if (!keyTypeTests) {
            it.skip('keyTypeTests config missing in fixture', () => {
            });
            return;
        }

        Object.entries(keyTypeTests).forEach(([keyType, testConfig]) => {
            describe(`Type: ${testConfig.typeName} (${keyType})`, () => {
                const {testKey, expectedColumns, testData, supportsUpdate} = testConfig;

                it('COLUMNS - displays correct column structure', () => {
                    cy.data(testKey);

                    cy.getTableData().then(({columns}) => {
                        // Use the existing verifyColumnsForType helper
                        verifyColumnsForType(columns, keyType);
                        // Also verify against fixture-defined columns
                        expect(columns, `${testConfig.typeName} key should have correct columns`).to.deep.equal(expectedColumns);
                    });
                });

                if (keyType === 'string') {
                    it('DATA - displays string value correctly', () => {
                        cy.data(testKey);

                        cy.getTableData().then(({rows}) => {
                            expect(rows.length).to.equal(1);
                            expect(rows[0][testData.valueColumnIndex]).to.equal(testData.originalValue);
                        });
                    });

                    if (supportsUpdate) {
                        it('UPDATE - modifies string value', () => {
                            cy.data(testKey);

                            // For strings, there's only one row with value column
                            cy.updateRow(0, 0, testData.updateValue, false);

                            cy.getTableData().then(({rows}) => {
                                expect(rows[0][testData.valueColumnIndex]).to.equal(testData.updateValue);

                                // Revert
                                cy.updateRow(0, 0, testData.originalValue, false);

                                cy.getTableData().then(({rows: revertedRows}) => {
                                    expect(revertedRows[0][testData.valueColumnIndex]).to.equal(testData.originalValue);
                                });
                            });
                        });
                    }
                }

                if (keyType === 'hash') {
                    it('DATA - displays hash fields correctly', () => {
                        cy.data(testKey);

                        cy.getTableData().then(({rows}) => {
                            expect(rows.length).to.be.greaterThan(0);

                            // Verify the test field exists with expected value
                            const targetRow = rows.find(r => r[testData.fieldColumnIndex] === testData.testField);
                            expect(targetRow, `Hash should contain field: ${testData.testField}`).to.exist;
                            expect(targetRow[testData.valueColumnIndex]).to.equal(testData.originalValue);
                        });
                    });

                    if (supportsUpdate) {
                        it('UPDATE - modifies hash field value', () => {
                            cy.data(testKey);

                            cy.getTableData().then(({rows}) => {
                                const rowIndex = rows.findIndex(r => r[testData.fieldColumnIndex] === testData.testField);
                                expect(rowIndex, `Row with field ${testData.testField} should exist`).to.be.greaterThan(-1);

                                // Edit the hash field - columnIndex 1 triggers edit, value goes to column 2
                                cy.updateRow(rowIndex, 1, testData.updateValue, false);

                                cy.getTableData().then(({rows: updatedRows}) => {
                                    expect(updatedRows[rowIndex][testData.valueColumnIndex]).to.equal(testData.updateValue);

                                    // Revert
                                    cy.updateRow(rowIndex, 1, testData.originalValue, false);

                                    cy.getTableData().then(({rows: revertedRows}) => {
                                        expect(revertedRows[rowIndex][testData.valueColumnIndex]).to.equal(testData.originalValue);
                                    });
                                });
                            });
                        });
                    }
                }

                if (keyType === 'list') {
                    it('DATA - displays list entries with indices', () => {
                        cy.data(testKey);

                        cy.getTableData().then(({rows}) => {
                            expect(rows.length).to.be.greaterThan(0);

                            // Verify index column contains numeric indices
                            rows.forEach((row, i) => {
                                const indexValue = row[testData.indexColumnIndex];
                                expect(indexValue).to.equal(String(i));
                            });
                        });
                    });

                    if (supportsUpdate) {
                        it('UPDATE - modifies list entry at index', () => {
                            cy.data(testKey);

                            cy.getTableData().then(({rows}) => {
                                const originalValue = rows[testData.testIndex][testData.valueColumnIndex];

                                // Edit list item - columnIndex 1 triggers edit for the index row
                                cy.updateRow(testData.testIndex, 1, 'keytype_test_value', false);

                                cy.getTableData().then(({rows: updatedRows}) => {
                                    expect(updatedRows[testData.testIndex][testData.valueColumnIndex]).to.equal('keytype_test_value');

                                    // Revert
                                    cy.updateRow(testData.testIndex, 1, originalValue, false);

                                    cy.getTableData().then(({rows: revertedRows}) => {
                                        expect(revertedRows[testData.testIndex][testData.valueColumnIndex]).to.equal(originalValue);
                                    });
                                });
                            });
                        });
                    }
                }

                if (keyType === 'set') {
                    it('DATA - displays set members correctly', () => {
                        cy.data(testKey);

                        cy.getTableData().then(({rows}) => {
                            expect(rows.length).to.be.greaterThan(0);

                            // Verify expected members are present
                            if (testData.expectedMembers) {
                                const actualMembers = rows.map(r => r[testData.valueColumnIndex]);
                                testData.expectedMembers.forEach(member => {
                                    expect(actualMembers, `Set should contain member: ${member}`).to.include(member);
                                });
                            }
                        });
                    });

                    // Sets don't support UPDATE - members can only be added/removed
                }

                if (keyType === 'zset') {
                    it('DATA - displays sorted set with members and scores', () => {
                        cy.data(testKey);

                        cy.getTableData().then(({rows}) => {
                            expect(rows.length).to.be.greaterThan(0);

                            // Verify each row has index, member, and score columns
                            rows.forEach((row, i) => {
                                const index = row[testData.indexColumnIndex];
                                const member = row[testData.memberColumnIndex];
                                const score = row[testData.scoreColumnIndex];

                                expect(index).to.equal(String(i));
                                expect(member).to.be.a('string').and.not.be.empty;
                                expect(score).to.match(/^-?\d+(\.\d+)?$/);
                            });
                        });
                    });

                    // Sorted sets don't support UPDATE through WhoDB's interface
                }
            });
        });
    });
});
