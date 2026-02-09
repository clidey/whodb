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


/**
 * Schema/Table Management Tests
 *
 * Tests for creating and managing storage units (tables, collections, etc.)
 * including field definition, data type selection, and primary key configuration.
 */
test.describe('Schema Management', () => {
    test.describe('Create Storage Unit Visibility', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows create storage unit card for SQL databases', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    // Wait for storage unit cards to load
                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Create card should be visible
                    await expect(page.locator('[data-testid="create-storage-unit-card"]')).toBeVisible();

                    // Should have create button
                    await expect(
                        page.locator('[data-testid="create-storage-unit-card"]')
                            .locator('button')
                            .filter({ hasText: /create/i })
                    ).toBeVisible();
                });

                test('can open create storage unit form', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    // Wait for storage unit cards to load
                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Click create button to expand the form
                    await page.locator('[data-testid="create-storage-unit-card"]')
                        .locator('button')
                        .filter({ hasText: /create/i })
                        .click();

                    // Wait for form to expand
                    await page.waitForTimeout(500);

                    // Form fields should be visible (form elements are in a sibling container, not inside create-storage-unit-card)
                    // Name input should be visible
                    await expect(page.locator('input[placeholder*="name" i]')).toBeVisible();

                    // Field cards should be visible
                    const fieldCards = page.locator('[data-testid="create-field-card"]');
                    const fieldCount = await fieldCards.count();
                    expect(fieldCount).toBeGreaterThanOrEqual(1);

                    // Add field button should be visible
                    await expect(page.locator('[data-testid="add-field-button"]')).toBeVisible();

                    // Submit button should be visible
                    await expect(page.locator('[data-testid="submit-button"]')).toBeVisible();
                });
            });
        });
    });

    test.describe('Add Columns with Types', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('can add columns with types during table creation', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    // Wait for storage unit cards to load
                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Open create form (opens Sheet/drawer via portal)
                    await page.locator('[data-testid="create-storage-unit-card"]')
                        .locator('button')
                        .filter({ hasText: /create/i })
                        .click();

                    // Wait for Sheet to open
                    await page.waitForTimeout(500);

                    // Initially should have one field
                    await expect(page.locator('[data-testid="create-field-card"]')).toHaveCount(1);

                    // Click add field button and wait for DOM update
                    // Use scrollIntoView since button may be below viewport in scrollable drawer
                    await page.locator('[data-testid="add-field-button"]').scrollIntoViewIfNeeded();
                    await page.locator('[data-testid="add-field-button"]').click();
                    await expect(page.locator('[data-testid="create-field-card"]')).toHaveCount(2);

                    // Add another field
                    await page.locator('[data-testid="add-field-button"]').scrollIntoViewIfNeeded();
                    await page.locator('[data-testid="add-field-button"]').click();
                    await expect(page.locator('[data-testid="create-field-card"]')).toHaveCount(3);

                    // Each field should have name input and type selector
                    const fieldCards = page.locator('[data-testid="create-field-card"]');
                    const count = await fieldCards.count();
                    for (let i = 0; i < count; i++) {
                        const field = fieldCards.nth(i);
                        // Field name input exists
                        await expect(field.locator('input')).toBeAttached();
                        // Field type selector button exists
                        await expect(field.locator('button[data-testid^="field-type-"]')).toBeAttached();
                    }
                });

                test('can remove added fields', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    // Wait for storage unit cards to load
                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Open create form (opens Sheet/drawer via portal)
                    await page.locator('[data-testid="create-storage-unit-card"]')
                        .locator('button')
                        .filter({ hasText: /create/i })
                        .click();

                    // Wait for Sheet to open
                    await page.waitForTimeout(500);

                    // Add two more fields (total of 3) - wait for each to be added
                    // Use scrollIntoView since button may be below viewport in scrollable drawer
                    await page.locator('[data-testid="add-field-button"]').scrollIntoViewIfNeeded();
                    await page.locator('[data-testid="add-field-button"]').click();
                    await expect(page.locator('[data-testid="create-field-card"]')).toHaveCount(2);

                    await page.locator('[data-testid="add-field-button"]').scrollIntoViewIfNeeded();
                    await page.locator('[data-testid="add-field-button"]').click();
                    await expect(page.locator('[data-testid="create-field-card"]')).toHaveCount(3);

                    // Remove field buttons should be visible (when there's more than one field)
                    await expect(page.locator('[data-testid="remove-field-button"]')).toHaveCount(3);

                    // Remove the last field
                    await page.locator('[data-testid="remove-field-button"]').last().click();
                    await expect(page.locator('[data-testid="create-field-card"]')).toHaveCount(2);

                    // Remove another field
                    await page.locator('[data-testid="remove-field-button"]').last().click();
                    await expect(page.locator('[data-testid="create-field-card"]')).toHaveCount(1);

                    // Remove button should still be visible but clicking on the last field should not remove it
                    // (minimum 1 field required based on the handleRemove logic in storage-unit.tsx)
                });
            });
        });
    });

    test.describe('Set Primary Key', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('can set primary key during table creation', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    // Wait for storage unit cards to load
                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Open create form
                    await page.locator('[data-testid="create-storage-unit-card"]')
                        .locator('button')
                        .filter({ hasText: /create/i })
                        .click();

                    await page.waitForTimeout(500);

                    // Check if database supports modifiers (Primary, Nullable)
                    const firstField = page.locator('[data-testid="create-field-card"]').first();
                    const hasPrimaryLabel = await firstField.locator('label').filter({ hasText: /primary/i }).count();

                    if (hasPrimaryLabel > 0) {
                        // Database supports modifiers
                        // Primary key checkbox should be visible
                        await expect(firstField.locator('label').filter({ hasText: /primary/i })).toBeVisible();

                        // Nullable checkbox should be visible
                        await expect(firstField.locator('label').filter({ hasText: /nullable/i })).toBeVisible();

                        // Check the Primary checkbox (click the button preceding the label)
                        const primaryLabel = firstField.locator('label').filter({ hasText: /primary/i });
                        const primaryButton = primaryLabel.locator('xpath=preceding-sibling::button[1]');
                        await primaryButton.click();

                        // Verify checkbox is checked (has data-state="checked")
                        await expect(primaryButton).toHaveAttribute('data-state', 'checked');
                    }
                    // If no primary label, database doesn't support modifiers - skip silently
                });
            });
        });
    });

    test.describe('Create New Table', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                const uniqueTableName = `test_table_${Date.now()}`;

                test.afterEach(async ({ whodb, page }) => {
                    // Clean up: Delete the created table if it exists
                    // Use scratchpad to drop the table
                    await page.goto(whodb.url('/scratchpad'));

                    // Wait for scratchpad page to load
                    await expect(page.locator('[data-testid="raw-execute-page"]')).toBeVisible({ timeout: 10000 });

                    // Wait for the editor cell to be ready
                    await page.locator('[data-testid="cell-0"]').waitFor({ timeout: 10000 });

                    // Type DROP TABLE query using the cell's textarea/editor
                    const dropQuery = `DROP TABLE IF EXISTS ${uniqueTableName};`;
                    const editorLocator = page.locator('[data-testid="cell-0"]')
                        .locator('textarea, [contenteditable="true"], .cm-content')
                        .first();
                    await editorLocator.click();
                    await editorLocator.fill('');
                    await editorLocator.fill(dropQuery);

                    // Execute the query
                    await page.locator('[data-testid="query-cell-button"]').click();

                    // Wait a bit for the query to complete
                    await page.waitForTimeout(1000);
                });

                test('successfully creates a new table', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    // Wait for storage unit cards to load
                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Open create form
                    await page.locator('[data-testid="create-storage-unit-card"]')
                        .locator('button')
                        .filter({ hasText: /create/i })
                        .click();

                    await page.waitForTimeout(500);

                    // Enter table name (form elements are in sibling container)
                    await page.locator('input[placeholder*="name" i]').first().fill('');
                    await page.locator('input[placeholder*="name" i]').first().fill(uniqueTableName);

                    // Configure first field: id (integer, primary key)
                    const firstField = page.locator('[data-testid="create-field-card"]').first();
                    await firstField.locator('input').first().fill('');
                    await firstField.locator('input').first().fill('id');

                    // Field type
                    await firstField.locator('button[data-testid^="field-type-"]').click();

                    // Select INTEGER or INT type
                    const options = page.locator('[role="option"]');
                    const optionCount = await options.count();
                    let foundInt = false;
                    for (let i = 0; i < optionCount; i++) {
                        const text = (await options.nth(i).textContent()).toLowerCase();
                        if (text === 'integer' || text === 'int' || text.includes('int')) {
                            await options.nth(i).click();
                            foundInt = true;
                            break;
                        }
                    }
                    if (!foundInt) {
                        // Fallback: click first option
                        await options.first().click();
                    }

                    // Set as primary key if supported
                    const hasPrimaryLabel = await firstField.locator('label').filter({ hasText: /primary/i }).count();
                    if (hasPrimaryLabel > 0) {
                        const primaryLabel = firstField.locator('label').filter({ hasText: /primary/i });
                        const primaryButton = primaryLabel.locator('xpath=preceding-sibling::button[1]');
                        await primaryButton.click();
                    }

                    // Add second field: name (text/varchar)
                    await page.locator('[data-testid="add-field-button"]').click();

                    const secondField = page.locator('[data-testid="create-field-card"]').nth(1);
                    await secondField.locator('input').first().fill('');
                    await secondField.locator('input').first().fill('name');

                    // Field type
                    await secondField.locator('button[data-testid^="field-type-"]').click();

                    // Select TEXT or VARCHAR type
                    const typeOptions = page.locator('[role="option"]');
                    const typeOptionCount = await typeOptions.count();
                    let foundText = false;
                    for (let i = 0; i < typeOptionCount; i++) {
                        const text = (await typeOptions.nth(i).textContent()).toLowerCase();
                        if (text === 'text' || text === 'varchar' || text.includes('char')) {
                            await typeOptions.nth(i).click();
                            foundText = true;
                            break;
                        }
                    }
                    if (!foundText) {
                        // Fallback: click second option
                        await typeOptions.nth(1).click();
                    }

                    // Submit the form
                    await page.locator('[data-testid="submit-button"]').click();

                    // Wait for success toast
                    await expect(page.getByText(/success/i)).toBeVisible({ timeout: 10000 });

                    // Wait for form to close
                    await page.waitForTimeout(1000);
                });

                test('new table appears in storage unit list after creation', async ({ whodb, page }) => {
                    // First create the table
                    await page.goto(whodb.url('/storage-unit'));

                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    await page.locator('[data-testid="create-storage-unit-card"]')
                        .locator('button')
                        .filter({ hasText: /create/i })
                        .click();

                    await page.waitForTimeout(500);

                    // Enter table name (form elements are in sibling container)
                    await page.locator('input[placeholder*="name" i]').first().fill('');
                    await page.locator('input[placeholder*="name" i]').first().fill(uniqueTableName);

                    // Configure first field (first input in field card)
                    const firstField = page.locator('[data-testid="create-field-card"]').first();
                    await firstField.locator('input').first().fill('');
                    await firstField.locator('input').first().fill('id');

                    await firstField.locator('button[data-testid^="field-type-"]').click();

                    await page.locator('[role="option"]').first().click();

                    // Submit
                    await page.locator('[data-testid="submit-button"]').click();

                    // Wait for creation to complete (drawer closes or page updates)
                    await page.waitForTimeout(3000);

                    // Refresh the page to ensure table list is updated
                    await page.goto(whodb.url('/storage-unit'));

                    // Wait for cards to load
                    await page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

                    // Verify new table appears in the list (use text match for partial match)
                    await expect(
                        page.locator('[data-testid="storage-unit-card"]').filter({ hasText: uniqueTableName })
                    ).toBeAttached({ timeout: 10000 });
                });
            });
        });
    });

    test.describe('Hide Create for Key-Value Databases', () => {
        forEachDatabase('keyvalue', (db) => {
            test.describe(`${db.type}`, () => {
                test('hides create option for key-value databases', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    // Wait for page to load
                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Create storage unit card should NOT be visible for Redis
                    await expect(page.locator('[data-testid="create-storage-unit-card"]')).not.toBeVisible();

                    // Verify it's in the DOM but hidden (CSS hidden)
                    await expect(page.locator('[data-testid="create-storage-unit-card"]')).toBeAttached();
                    await expect(page.locator('[data-testid="create-storage-unit-card"]')).not.toBeVisible();
                });
            });
        });
    });

    test.describe('Form Validation', () => {
        forEachDatabase('sql', (db) => {
            test.describe(`${db.type}`, () => {
                test('prevents submission without table name', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Open create form
                    await page.locator('[data-testid="create-storage-unit-card"]')
                        .locator('button')
                        .filter({ hasText: /create/i })
                        .click();

                    await page.waitForTimeout(500);

                    // Configure field but leave table name empty
                    const firstField = page.locator('[data-testid="create-field-card"]').first();
                    await firstField.locator('input').first().fill('');
                    await firstField.locator('input').first().fill('test_field');

                    await firstField.locator('button[data-testid^="field-type-"]').click();

                    await page.locator('[role="option"]').first().click();

                    // Submit without entering table name
                    await page.locator('[data-testid="submit-button"]').click();

                    // Form should still be open (validation prevented submission)
                    // The drawer/sheet should still be visible
                    await page.waitForTimeout(1000);
                    await expect(page.locator('[data-testid="create-field-card"]')).toBeAttached();
                });

                test('prevents submission with empty field name', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Open create form
                    await page.locator('[data-testid="create-storage-unit-card"]')
                        .locator('button')
                        .filter({ hasText: /create/i })
                        .click();

                    await page.waitForTimeout(500);

                    // Enter table name
                    await page.locator('input[placeholder*="name" i]').first().fill('');
                    await page.locator('input[placeholder*="name" i]').first().fill('test_table');

                    // Leave field name empty but select type
                    const firstField = page.locator('[data-testid="create-field-card"]').first();
                    await firstField.locator('button[data-testid^="field-type-"]').click();

                    await page.locator('[role="option"]').first().click();

                    // Submit with empty field name
                    await page.locator('[data-testid="submit-button"]').click();

                    // Form should still be open (validation prevented submission)
                    await page.waitForTimeout(1000);
                    await expect(page.locator('[data-testid="create-field-card"]')).toBeAttached();
                });
            });
        });
    });
});
