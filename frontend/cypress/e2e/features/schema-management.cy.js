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

import { forEachDatabase, loginToDatabase } from '../../support/test-runner';


/**
 * Schema/Table Management Tests
 *
 * Tests for creating and managing storage units (tables, collections, etc.)
 * including field definition, data type selection, and primary key configuration.
 */
describe('Schema Management', () => {
    describe('Create Storage Unit Visibility', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('shows create storage unit card for SQL databases', () => {
                    cy.visit('/storage-unit');

                    // Wait for storage unit cards to load
                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    // Create card should be visible
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .should('be.visible');

                    // Should have create button
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .find('button')
                        .contains(/create/i)
                        .should('be.visible');
                });

                it('can open create storage unit form', () => {
                    cy.visit('/storage-unit');

                    // Wait for storage unit cards to load
                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    // Click create button to expand the form
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .find('button')
                        .contains(/create/i)
                        .click();

                    // Wait for form to expand
                    cy.wait(500);

                    // Form fields should be visible (form elements are in a sibling container, not inside create-storage-unit-card)
                    // Name input should be visible
                    cy.get('input[placeholder*="name" i]')
                        .should('be.visible');

                    // Field cards should be visible
                    cy.get('[data-testid="create-field-card"]')
                        .should('have.length.at.least', 1);

                    // Add field button should be visible
                    cy.get('[data-testid="add-field-button"]')
                        .should('be.visible');

                    // Submit button should be visible
                    cy.get('[data-testid="submit-button"]')
                        .should('be.visible');
                });
            });
        });
    });

    describe('Add Columns with Types', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('can add columns with types during table creation', () => {
                    cy.visit('/storage-unit');

                    // Wait for storage unit cards to load
                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    // Open create form (opens Sheet/drawer via portal)
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .find('button')
                        .contains(/create/i)
                        .click();

                    // Wait for Sheet to open
                    cy.wait(500);

                    // Initially should have one field
                    cy.get('[data-testid="create-field-card"]')
                        .should('have.length', 1);

                    // Click add field button and wait for DOM update
                    // Use scrollIntoView since button may be below viewport in scrollable drawer
                    cy.get('[data-testid="add-field-button"]')
                        .scrollIntoView()
                        .click();
                    cy.get('[data-testid="create-field-card"]')
                        .should('have.length', 2);

                    // Add another field
                    cy.get('[data-testid="add-field-button"]')
                        .scrollIntoView()
                        .click();
                    cy.get('[data-testid="create-field-card"]')
                        .should('have.length', 3);

                    // Each field should have name input and type selector
                    cy.get('[data-testid="create-field-card"]').each(($field) => {
                        // Field name input exists
                        cy.wrap($field).find('input').should('exist');
                        // Field type selector button exists
                        cy.wrap($field).find('button[data-testid^="field-type-"]').should('exist');
                    });
                });

                it('can remove added fields', () => {
                    cy.visit('/storage-unit');

                    // Wait for storage unit cards to load
                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    // Open create form (opens Sheet/drawer via portal)
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .find('button')
                        .contains(/create/i)
                        .click();

                    // Wait for Sheet to open
                    cy.wait(500);

                    // Add two more fields (total of 3) - wait for each to be added
                    // Use scrollIntoView since button may be below viewport in scrollable drawer
                    cy.get('[data-testid="add-field-button"]').scrollIntoView().click();
                    cy.get('[data-testid="create-field-card"]').should('have.length', 2);

                    cy.get('[data-testid="add-field-button"]').scrollIntoView().click();
                    cy.get('[data-testid="create-field-card"]').should('have.length', 3);

                    // Remove field buttons should be visible (when there's more than one field)
                    cy.get('[data-testid="remove-field-button"]')
                        .should('have.length', 3);

                    // Remove the last field
                    cy.get('[data-testid="remove-field-button"]').last().click();
                    cy.get('[data-testid="create-field-card"]').should('have.length', 2);

                    // Remove another field
                    cy.get('[data-testid="remove-field-button"]').last().click();
                    cy.get('[data-testid="create-field-card"]').should('have.length', 1);

                    // Remove button should still be visible but clicking on the last field should not remove it
                    // (minimum 1 field required based on the handleRemove logic in storage-unit.tsx)
                });
            });
        });
    });

    describe('Set Primary Key', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('can set primary key during table creation', () => {
                    cy.visit('/storage-unit');

                    // Wait for storage unit cards to load
                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    // Open create form
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .find('button')
                        .contains(/create/i)
                        .click();

                    cy.wait(500);

                    // Check if database supports modifiers (Primary, Nullable)
                    cy.get('[data-testid="create-field-card"]').first().then(($field) => {
                        if ($field.find('label:contains("Primary")').length > 0) {
                            // Database supports modifiers
                            cy.get('[data-testid="create-field-card"]').first().within(() => {
                                // Primary key checkbox should be visible
                                cy.contains('label', /primary/i)
                                    .should('be.visible');

                                // Nullable checkbox should be visible
                                cy.contains('label', /nullable/i)
                                    .should('be.visible');

                                // Check the Primary checkbox
                                cy.contains('label', /primary/i)
                                    .prev('button')
                                    .click();
                            });

                            // Verify checkbox is checked (has data-state="checked")
                            cy.get('[data-testid="create-field-card"]').first()
                                .contains('label', /primary/i)
                                .prev('button')
                                .should('have.attr', 'data-state', 'checked');
                        } else {
                            // Database doesn't support modifiers (skip test)
                            cy.log('Database does not support field modifiers');
                        }
                    });
                });
            });
        });
    });

    describe('Create New Table', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                const uniqueTableName = `test_table_${Date.now()}`;
                afterEach(() => {
                    // Clean up: Delete the created table if it exists
                    // Use scratchpad to drop the table
                    cy.visit('/scratchpad');

                    // Wait for scratchpad page to load
                    cy.get('[data-testid="raw-execute-page"]', { timeout: 10000 })
                        .should('be.visible');

                    // Wait for the editor cell to be ready
                    cy.get('[data-testid="cell-0"]', { timeout: 10000 })
                        .should('exist');

                    // Type DROP TABLE query using the cell's textarea/editor
                    const dropQuery = `DROP TABLE IF EXISTS ${uniqueTableName};`;
                    cy.get('[data-testid="cell-0"]')
                        .find('textarea, [contenteditable="true"], .cm-content')
                        .first()
                        .click()
                        .clear()
                        .type(dropQuery, { delay: 0 });

                    // Execute the query
                    cy.get('[data-testid="query-cell-button"]').click();

                    // Wait a bit for the query to complete
                    cy.wait(1000);
                });

                it('successfully creates a new table', () => {
                    cy.visit('/storage-unit');

                    // Wait for storage unit cards to load
                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    // Open create form
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .find('button')
                        .contains(/create/i)
                        .click();

                    cy.wait(500);

                    // Enter table name (form elements are in sibling container)
                    cy.get('input[placeholder*="name" i]')
                        .first()
                        .clear()
                        .type(uniqueTableName);

                    // Configure first field: id (integer, primary key)
                    cy.get('[data-testid="create-field-card"]').first().within(() => {
                        // Field name (first input in field card)
                        cy.get('input').first()
                            .clear()
                            .type('id');

                        // Field type
                        cy.get('button[data-testid^="field-type-"]').click();
                    });

                    // Select INTEGER or INT type
                    cy.get('[role="option"]').then(($options) => {
                        const intOption = $options.filter((i, el) => {
                            const text = el.textContent.toLowerCase();
                            return text === 'integer' || text === 'int' || text.includes('int');
                        });

                        if (intOption.length > 0) {
                            cy.wrap(intOption.first()).click();
                        } else {
                            // Fallback: click first option
                            cy.get('[role="option"]').first().click();
                        }
                    });

                    // Set as primary key if supported
                    cy.get('[data-testid="create-field-card"]').first().then(($field) => {
                        if ($field.find('label:contains("Primary")').length > 0) {
                            cy.wrap($field).within(() => {
                                cy.contains('label', /primary/i)
                                    .prev('button')
                                    .click();
                            });
                        }
                    });

                    // Add second field: name (text/varchar)
                    cy.get('[data-testid="add-field-button"]').click();

                    cy.get('[data-testid="create-field-card"]').eq(1).within(() => {
                        // Field name (first input in field card)
                        cy.get('input').first()
                            .clear()
                            .type('name');

                        // Field type
                        cy.get('button[data-testid^="field-type-"]').click();
                    });

                    // Select TEXT or VARCHAR type
                    cy.get('[role="option"]').then(($options) => {
                        const textOption = $options.filter((i, el) => {
                            const text = el.textContent.toLowerCase();
                            return text === 'text' || text === 'varchar' || text.includes('char');
                        });

                        if (textOption.length > 0) {
                            cy.wrap(textOption.first()).click();
                        } else {
                            // Fallback: click second option
                            cy.get('[role="option"]').eq(1).click();
                        }
                    });

                    // Submit the form
                    cy.get('[data-testid="submit-button"]').click();

                    // Wait for success toast
                    cy.contains(/success/i, { timeout: 10000 }).should('be.visible');

                    // Wait for form to close
                    cy.wait(1000);
                });

                it('new table appears in storage unit list after creation', () => {
                    // First create the table
                    cy.visit('/storage-unit');

                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    cy.get('[data-testid="create-storage-unit-card"]')
                        .find('button')
                        .contains(/create/i)
                        .click();

                    cy.wait(500);

                    // Enter table name (form elements are in sibling container)
                    cy.get('input[placeholder*="name" i]')
                        .first()
                        .clear()
                        .type(uniqueTableName);

                    // Configure first field (first input in field card)
                    cy.get('[data-testid="create-field-card"]').first().within(() => {
                        cy.get('input').first()
                            .clear()
                            .type('id');

                        cy.get('button[data-testid^="field-type-"]').click();
                    });

                    cy.get('[role="option"]').first().click();

                    // Submit
                    cy.get('[data-testid="submit-button"]').click();

                    // Wait for creation to complete (drawer closes or page updates)
                    cy.wait(3000);

                    // Refresh the page to ensure table list is updated
                    cy.visit('/storage-unit');

                    // Wait for cards to load
                    cy.get('[data-testid="storage-unit-card"]', { timeout: 15000 })
                        .should('have.length.at.least', 1);

                    // Verify new table appears in the list (use contains for partial match)
                    cy.contains('[data-testid="storage-unit-card"]', uniqueTableName, { timeout: 10000 })
                        .should('exist');
                });
            });
        });
    });

    describe('Hide Create for Key-Value Databases', () => {
        forEachDatabase('keyvalue', (db) => {
            describe(`${db.type}`, () => {
                it('hides create option for key-value databases', () => {
                    cy.visit('/storage-unit');

                    // Wait for page to load
                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    // Create storage unit card should NOT be visible for Redis
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .should('not.be.visible');

                    // Verify it's in the DOM but hidden (CSS hidden)
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .should('exist')
                        .and('not.be.visible');
                });
            });
        });
    });

    describe('Form Validation', () => {
        forEachDatabase('sql', (db) => {
            describe(`${db.type}`, () => {
                it('prevents submission without table name', () => {
                    cy.visit('/storage-unit');

                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    // Open create form
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .find('button')
                        .contains(/create/i)
                        .click();

                    cy.wait(500);

                    // Configure field but leave table name empty
                    cy.get('[data-testid="create-field-card"]').first().within(() => {
                        cy.get('input').first()
                            .clear()
                            .type('test_field');

                        cy.get('button[data-testid^="field-type-"]').click();
                    });

                    cy.get('[role="option"]').first().click();

                    // Submit without entering table name
                    cy.get('[data-testid="submit-button"]').click();

                    // Form should still be open (validation prevented submission)
                    // The drawer/sheet should still be visible
                    cy.wait(1000);
                    cy.get('[data-testid="create-field-card"]').should('exist');
                });

                it('prevents submission with empty field name', () => {
                    cy.visit('/storage-unit');

                    cy.get('[data-testid="storage-unit-card-list"]', { timeout: 15000 })
                        .should('be.visible');

                    // Open create form
                    cy.get('[data-testid="create-storage-unit-card"]')
                        .find('button')
                        .contains(/create/i)
                        .click();

                    cy.wait(500);

                    // Enter table name
                    cy.get('input[placeholder*="name" i]')
                        .first()
                        .clear()
                        .type('test_table');

                    // Leave field name empty but select type
                    cy.get('[data-testid="create-field-card"]').first().within(() => {
                        cy.get('button[data-testid^="field-type-"]').click();
                    });

                    cy.get('[role="option"]').first().click();

                    // Submit with empty field name
                    cy.get('[data-testid="submit-button"]').click();

                    // Form should still be open (validation prevented submission)
                    cy.wait(1000);
                    cy.get('[data-testid="create-field-card"]').should('exist');
                });
            });
        });
    });
});
