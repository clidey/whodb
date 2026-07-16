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
import { getUniqueTestId, waitForMutation } from '../../support/helpers/test-utils.mjs';

function matchesTypeLabel(text, label) {
    if (text === label) {
        return true;
    }
    if (label === 'int') {
        return /^int\d*$/.test(text);
    }
    return false;
}

async function selectFieldType(page, typeLabels) {
    const options = page.locator('[role="option"]');
    await options.first().waitFor({ state: 'visible', timeout: 10000 });
    const optionCount = await options.count();
    for (let i = 0; i < optionCount; i++) {
        const text = ((await options.nth(i).textContent()) ?? '').trim().toLowerCase();
        if (typeLabels.some(label => matchesTypeLabel(text, label))) {
            await options.nth(i).click();
            return;
        }
    }

    throw new Error(`No field type option found for: ${typeLabels.join(', ')}`);
}

function schemaQualifiedTableName(db, tableName) {
    return `${db.sql?.schemaPrefix ?? ''}${tableName}`;
}

function schemaFieldTypes(db, key, defaults) {
    return db.schemaManagement?.fieldTypes?.[key] ?? defaults;
}

function waitForCreateSourceObjectFromDefinition(page) {
    const responsePromise = page.waitForResponse(
        resp =>
            resp.url().includes('/api/query') &&
            resp.request().postDataJSON?.()?.operationName === 'CreateSourceObjectFromDefinition',
        { timeout: 30000 }
    );

    return async () => {
        const response = await responsePromise;
        const result = await response.json();
        expect(result.errors, 'CreateSourceObjectFromDefinition mutation should succeed').toBeUndefined();
        return {
            result,
            variables: response.request().postDataJSON?.()?.variables ?? {},
            headers: response.request().headers(),
        };
    };
}

function createdObjectRef(parentRef, objectName) {
    return {
        Kind: 'Table',
        Path: [...(parentRef?.Path ?? []), objectName],
    };
}

async function querySourceFieldConstraints(page, ref, requestHeaders = {}) {
    const authorization = requestHeaders.authorization ?? requestHeaders.Authorization;
    const result = await page.evaluate(async ({ ref, authorization }) => {
        const headers = { 'content-type': 'application/json' };
        if (authorization) {
            headers.authorization = authorization;
        }
        // Session-cookie auth requires the CSRF token (double-submit) on POST
        // requests. Mirror the app's Apollo client by echoing the readable
        // whodb_csrf cookie back in the X-CSRF-Token header.
        const csrfMatch = document.cookie.split('; ').find(row => row.startsWith('whodb_csrf='));
        if (csrfMatch) {
            headers['X-CSRF-Token'] = decodeURIComponent(csrfMatch.slice('whodb_csrf='.length));
        }
        const response = await fetch(`${window.location.origin}/api/query`, {
            method: 'POST',
            headers,
            credentials: 'include',
            body: JSON.stringify({
                operationName: 'SourceFieldConstraints',
                variables: { ref },
                query: `query SourceFieldConstraints($ref: SourceObjectRefInput!) {
                    SourceFieldConstraints(ref: $ref) {
                        Name
                        Type
                        Nullable
                        Primary
                        Unique
                        Identity
                        DefaultValue
                        AllowedValues
                        CheckMin
                        CheckMax
                        ForeignKey {
                            Table
                            Column
                        }
                        Length
                        Precision
                        Scale
                    }
                }`,
            }),
        });
        const text = await response.text();
        try {
            return JSON.parse(text);
        } catch {
            throw new Error(`SourceFieldConstraints HTTP ${response.status}: ${text}`);
        }
    }, { ref, authorization });

    expect(result.errors, 'SourceFieldConstraints query should succeed').toBeUndefined();
    return result.data.SourceFieldConstraints;
}

async function setModifierIfPresent(field, labelPattern) {
    await expandFieldOptionsIfPresent(field);
    const label = field.locator('label').filter({ hasText: labelPattern }).first();
    if (await label.count() === 0) {
        return false;
    }
    const button = label.locator('xpath=preceding-sibling::button[1]');
    await button.click();
    await expect(button).toHaveAttribute('data-state', 'checked');
    return true;
}

async function expandFieldOptionsIfPresent(field) {
    const trigger = field.locator('[data-testid^="field-options-trigger-"]').first();
    if (await trigger.count() === 0) {
        return;
    }
    if (await trigger.getAttribute('data-state') !== 'open') {
        await trigger.click();
    }
}

/**
 * Schema/Table Management Tests
 *
 * Tests for creating and managing storage units (tables, collections, etc.)
 * including field definition, data type selection, and primary key configuration.
 */
test.describe('Schema Management', () => {
    test.describe('Create Storage Unit Visibility', () => {
        forEachDatabase('sql', (db) => {
            if (db.features?.schemaManagement === false) {
                return;
            }

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
                    // Already on /storage-unit from beforeEach (storageState has card view)
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
                    await expect(page.locator('input[placeholder*="name" i]').first()).toBeVisible();

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
            if (db.features?.schemaManagement === false) {
                return;
            }

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
                        await expect(field.locator('input').first()).toBeAttached();
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
            if (db.features?.schemaManagement === false) {
                return;
            }

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
                    await expandFieldOptionsIfPresent(firstField);
                    const hasPrimaryLabel = await firstField.locator('label').filter({ hasText: /primary/i }).count();

                    if (hasPrimaryLabel > 0) {
                        await expect(firstField.locator('label').filter({ hasText: /primary/i })).toBeVisible();
                        await expect(firstField.locator('label').filter({ hasText: /nullable|null/i })).toBeVisible();
                        const uniqueLabel = firstField.locator('label').filter({ hasText: /unique/i });
                        if (await uniqueLabel.count() > 0) {
                            await expect(uniqueLabel).toBeVisible();
                        }
                        const identityLabel = firstField.locator('label').filter({ hasText: /identity|auto_increment|autoincrement|nextval/i });
                        if (await identityLabel.count() > 0) {
                            await expect(identityLabel).toBeVisible();
                        }
                        await expect(firstField.locator('input[placeholder*="default" i]')).toBeVisible();

                        const primaryLabel = firstField.locator('label').filter({ hasText: /primary/i });
                        const primaryButton = primaryLabel.locator('xpath=preceding-sibling::button[1]');
                        await primaryButton.click();

                        await expect(primaryButton).toHaveAttribute('data-state', 'checked');
                    }
                    // If no primary label, database doesn't support modifiers - skip silently
                });
            });
        });
    });

    test.describe('Create New Table', () => {
        forEachDatabase('sql', (db) => {
            if (db.features?.schemaManagement === false) {
                return;
            }

            test.describe(`${db.type}`, () => {
                const createdTableNames = new Set();

                test.afterEach(async ({ whodb, page }) => {
                    if (createdTableNames.size === 0) {
                        return;
                    }

                    // Clean up: Delete the created table if it exists
                    // Use scratchpad to drop the table
                    await page.goto(whodb.url('/scratchpad'));

                    // Wait for scratchpad page to load
                    await expect(page.locator('[data-testid="raw-execute-page"]')).toBeVisible({ timeout: 10000 });

                    // Wait for the editor cell to be ready
                    await page.locator('[data-testid="cell-0"]').waitFor({ timeout: 10000 });

                    const editorLocator = page.locator('[data-testid="cell-0"]')
                        .locator('textarea, [contenteditable="true"], .cm-content')
                        .first();

                    for (const tableName of createdTableNames) {
                        const dropQuery = `DROP TABLE IF EXISTS ${schemaQualifiedTableName(db, tableName)};`;
                        await editorLocator.click();
                        await editorLocator.fill('');
                        await editorLocator.fill(dropQuery);
                        await whodb.runCode(0);
                    }

                    createdTableNames.clear();
                });

                test('successfully creates a new table', async ({ whodb, page }) => {
                    const uniqueTableName = `test_table_${getUniqueTestId()}`;
                    createdTableNames.add(uniqueTableName);

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

                    await selectFieldType(page, schemaFieldTypes(db, 'integer', ['integer', 'int']));
                    const primarySelected = await setModifierIfPresent(firstField, /primary/i);

                    // Add second field: name (text/varchar)
                    await page.locator('[data-testid="add-field-button"]').click();

                    const secondField = page.locator('[data-testid="create-field-card"]').nth(1);
                    await secondField.locator('input').first().fill('');
                    await secondField.locator('input').first().fill('name');

                    // Field type
                    await secondField.locator('button[data-testid^="field-type-"]').click();

                    await selectFieldType(page, schemaFieldTypes(db, 'string', ['varchar', 'string', 'text']));

                    const verifyCreate = waitForCreateSourceObjectFromDefinition(page);

                    // Submit the form
                    await page.locator('[data-testid="submit-button"]').click();
                    const createRequest = await verifyCreate();

                    await page.goto(whodb.url('/storage-unit'));
                    await page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

                    await expect(
                        page.locator('[data-testid="storage-unit-card"]').filter({ hasText: uniqueTableName })
                    ).toBeAttached({ timeout: 10000 });

                    const constraints = await querySourceFieldConstraints(
                        page,
                        createdObjectRef(createRequest.variables.parent, uniqueTableName),
                        createRequest.headers
                    );
                    const idConstraints = constraints.find(field => field.Name.toLowerCase() === 'id');
                    const nameConstraints = constraints.find(field => field.Name.toLowerCase() === 'name');

                    expect(idConstraints, 'created id field constraints should be returned').toBeTruthy();
                    expect(nameConstraints, 'created name field constraints should be returned').toBeTruthy();
                    expect(idConstraints.Type, 'id field type should be returned').toBeTruthy();
                    expect(nameConstraints.Type, 'name field type should be returned').toBeTruthy();
                    if (primarySelected) {
                        expect(idConstraints.Primary, 'id primary key constraint should round-trip').toBe(true);
                    }
                });

                test('new table appears in storage unit list after creation', async ({ whodb, page }) => {
                    const uniqueTableName = `test_table_${getUniqueTestId()}`;
                    createdTableNames.add(uniqueTableName);

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

                    await selectFieldType(page, schemaFieldTypes(db, 'integer', ['integer', 'int']));

                    const verifyCreate = waitForMutation(page, 'CreateSourceObjectFromDefinition');

                    // Submit
                    await page.locator('[data-testid="submit-button"]').click();
                    await verifyCreate();

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

    test.describe('Create for Key-Value Databases', () => {
        forEachDatabase('keyvalue', (db) => {
            test.describe(`${db.type}`, () => {
                test('shows create option for key-value databases', async ({ whodb, page }) => {
                    await page.goto(whodb.url('/storage-unit'));

                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Create storage unit card should be visible and have a create button
                    await expect(page.locator('[data-testid="create-storage-unit-card"]')).toBeAttached();
                    await expect(
                        page.locator('[data-testid="create-storage-unit-card"]')
                            .locator('button')
                            .filter({ hasText: /create/i })
                    ).toBeAttached();
                });

                test('creates a new key', async ({ whodb, page }) => {
                    const uniqueKey = `e2e_test_key_${Date.now()}`;

                    await page.goto(whodb.url('/storage-unit'));
                    await expect(page.locator('[data-testid="storage-unit-card-list"]')).toBeVisible({ timeout: 15000 });

                    // Open create form
                    await page.locator('[data-testid="create-storage-unit-card"]')
                        .locator('button')
                        .filter({ hasText: /create/i })
                        .click();

                    await page.waitForTimeout(500);

                    // Enter key name
                    await page.locator('input[placeholder*="name" i]').first().fill(uniqueKey);

                    // Submit
                    await page.locator('[data-testid="submit-button"]').click();

                    await page.waitForTimeout(2000);

                    // Refresh and verify new key appears
                    await page.goto(whodb.url('/storage-unit'));
                    await page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

                    await expect(
                        page.locator('[data-testid="storage-unit-card"]').filter({ hasText: uniqueKey })
                    ).toBeAttached({ timeout: 10000 });
                });
            });
        });
    });

    test.describe('Form Validation', () => {
        forEachDatabase('sql', (db) => {
            if (db.features?.schemaManagement === false) {
                return;
            }

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
