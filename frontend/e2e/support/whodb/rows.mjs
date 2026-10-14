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

import {expect} from "@playwright/test";
import {TIMEOUT} from "../helpers/test-utils.mjs";

async function waitForStableLocator(locator) {
    await locator.evaluate((element) => {
        return new Promise((resolve) => {
            let lastRect = "";
            let stableFrames = 0;

            const check = () => {
                const rect = element.getBoundingClientRect();
                const currentRect = `${rect.x}:${rect.y}:${rect.width}:${rect.height}`;

                if (currentRect === lastRect) {
                    stableFrames += 1;
                } else {
                    stableFrames = 0;
                    lastRect = currentRect;
                }

                if (stableFrames >= 3) {
                    resolve();
                    return;
                }

                requestAnimationFrame(check);
            };

            requestAnimationFrame(check);
        });
    });
}

/** Methods for row-level operations: add, delete, update, context menu */
export const rowsMethods = {
    /**
     * Add a row to the table
     * @param {Object|string} data
     * @param {boolean} isSingleInput
     */
    async addRow(data, isSingleInput = false) {
        // The sheet fetches column constraints after opening; wait for them so identity columns
        // are already disabled before we decide which fields to fill.
        const constraintsLoaded = this.page.waitForResponse(
            (response) => response.url().includes("/api/query")
                && response.request().postDataJSON?.()?.operationName === "SourceFieldConstraints",
            { timeout: TIMEOUT.ACTION },
        ).catch(() => null);
        await this.page.locator('[data-testid="add-row-button"]').click();
        await constraintsLoaded;

        if (isSingleInput) {
            const jsonString = typeof data === "string" ? data : JSON.stringify(data, null, 2);
            const editorContainer = this.page.locator('[data-testid="add-row-field-document"] .cm-editor');
            await editorContainer.waitFor({ timeout: TIMEOUT.ELEMENT });
            await editorContainer.locator('[role="textbox"]').fill(jsonString);
        } else {
            for (const [key, value] of Object.entries(data)) {
                const input = this.page.locator(`[data-testid="add-row-field-${key}"] input`);
                // Identity columns are auto-generated and their inputs are disabled.
                await input.waitFor({ state: "attached", timeout: TIMEOUT.ELEMENT });
                if (await input.isDisabled()) {
                    continue;
                }
                await input.clear();
                await input.fill(value);
            }
        }

        const submitAddRowButton = this.page.locator('[data-testid="submit-add-row-button"]');
        await submitAddRowButton.waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
        await expect(submitAddRowButton).toBeEnabled({ timeout: TIMEOUT.ACTION });
        await waitForStableLocator(submitAddRowButton);
        await submitAddRowButton.click();
        await submitAddRowButton.waitFor({ state: "hidden", timeout: TIMEOUT.SLOW });
        await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.SLOW });
        await this.waitForDataTable();
    },

    /**
     * Waits for a row containing a specific value in a given column
     * @param {number} columnIndex
     * @param {string} expectedValue
     * @param {Object} options
     * @returns {Promise<number>}
     */
    async waitForRowValue(columnIndex, expectedValue, options = {}) {
        const timeout = options.timeout || TIMEOUT.SLOW;
        const expectedStr = String(expectedValue).trim();
        let foundIndex = -1;

        await expect(async () => {
            const { rows } = await this.getTableData();
            foundIndex = -1;
            for (let i = 0; i < rows.length; i++) {
                const cellText = String(rows[i][columnIndex] ?? "").trim();
                if (cellText === expectedStr) {
                    foundIndex = i;
                    break;
                }
            }
            expect(foundIndex).not.toEqual(-1);
        }).toPass({ timeout });

        return foundIndex;
    },

    /**
     * Waits for a row containing a specific value anywhere in the row
     * @param {string} expectedValue
     * @param {Object} options
     * @returns {Promise<number>}
     */
    async waitForRowContaining(expectedValue, options = {}) {
        const timeout = options.timeout || TIMEOUT.SLOW;
        const caseSensitive = options.caseSensitive || false;
        const searchStr = caseSensitive ? String(expectedValue) : String(expectedValue).toLowerCase();
        let foundIndex = -1;

        await expect(async () => {
            const { rows } = await this.getTableData();
            foundIndex = -1;
            for (let i = 0; i < rows.length; i++) {
                const rowText = rows[i].join(" ");
                const comparableRowText = caseSensitive ? rowText : rowText.toLowerCase();
                if (comparableRowText.includes(searchStr)) {
                    foundIndex = i;
                    break;
                }
            }
            expect(foundIndex).not.toEqual(-1);
        }).toPass({ timeout });

        return foundIndex;
    },

    /**
     * Open context menu on a table row with retry logic
     * @param {number} rowIndex
     * @param {number} maxRetries
     */
    async openContextMenu(rowIndex, maxRetries = 3) {
        for (let attempt = 1; attempt <= maxRetries; attempt++) {
            const targetRow = this.dataRow(rowIndex);
            await targetRow.scrollIntoViewIfNeeded();
            await targetRow.waitFor({ state: "visible" });
            await targetRow.click({ button: "right", force: true });

            const menuExists =
                (await this.page.locator('[data-testid="context-menu-edit-row"], [data-testid="context-menu-more-actions"]').count()) > 0;

            if (menuExists) {
                return;
            }

            if (attempt < maxRetries) {
                await this.page.mouse.click(0, 0);
                // Required: retry loop stabilization between attempts
                await this.page.waitForTimeout(100);
            } else {
                await this.page
                    .locator('[data-testid="context-menu-edit-row"], [data-testid="context-menu-more-actions"]')
                    .waitFor({ timeout: TIMEOUT.ELEMENT });
            }
        }
    },

    /**
     * Delete a row by index
     * @param {number} rowIndex
     * @param {{ waitForRowCount?: boolean }} [options]
     */
    async deleteRow(rowIndex, { waitForRowCount = true } = {}) {
        const initialRowCount = await this.dataRows().count();
        expect(initialRowCount).toBeGreaterThan(rowIndex);

        await this.openContextMenu(rowIndex);

        const deleteBtn = this.page.locator('[data-testid="context-menu-delete-row"]');
        await deleteBtn.scrollIntoViewIfNeeded();
        await deleteBtn.waitFor({ timeout: TIMEOUT.ELEMENT });
        await deleteBtn.click({ force: true });

        const confirmDeleteBtn = this.page.locator('[data-testid="confirm-delete-row-button"]');
        await confirmDeleteBtn.waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
        await confirmDeleteBtn.click();
        await confirmDeleteBtn.waitFor({ state: "hidden", timeout: TIMEOUT.ACTION });
        await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.ELEMENT });

        if (waitForRowCount) {
            await expect(async () => {
                const { rows } = await this.getTableData();
                expect(rows.length).toEqual(initialRowCount - 1);
            }).toPass({ timeout: TIMEOUT.SLOW });
        }
    },

    /**
     * Update a row via context menu
     * @param {number} rowIndex
     * @param {number} columnIndex
     * @param {string} text
     * @param {boolean} cancel
     */
    async updateRow(rowIndex, columnIndex, text, cancel = true) {
        // Required: state stabilization before opening context menu
        await this.page.waitForTimeout(500);
        await this.openContextMenu(rowIndex);

        const editBtn = this.page.locator('[data-testid="context-menu-edit-row"]');
        await editBtn.scrollIntoViewIfNeeded();
        await editBtn.waitFor({ timeout: TIMEOUT.ELEMENT });
        await editBtn.click({ force: true });

        const standardField = this.page.locator(`[data-testid="editable-field-${columnIndex}"]`);
        if ((await standardField.count()) > 0) {
            await standardField.clear();
            await standardField.fill(text);
        } else {
            const targetElement = await this.page.evaluate(() => {
                const elements = document.querySelectorAll('textarea, input[type="text"]');
                for (const el of elements) {
                    const value = el.value;
                    if (value === "" || value.startsWith("{") || value.startsWith("[")) {
                        return true;
                    }
                }
                return false;
            });

            if (targetElement) {
                const textareas = this.page.locator('textarea, input[type="text"]');
                const count = await textareas.count();
                let filled = false;
                for (let i = 0; i < count; i++) {
                    const el = textareas.nth(i);
                    const value = await el.inputValue();
                    if (value === "" || value.startsWith("{") || value.startsWith("[")) {
                        await el.clear();
                        await el.fill(text);
                        filled = true;
                        break;
                    }
                }
                if (!filled) {
                    await textareas.first().clear();
                    await textareas.first().fill(text);
                }
            } else {
                await this.page.locator('textarea, input[type="text"]').first().clear();
                await this.page.locator('textarea, input[type="text"]').first().fill(text);
            }
        }

        if (cancel) {
            await this.page.locator('[data-testid="cancel-edit-row"]').click();
            await this.page.getByText("Edit Row").first().waitFor({ state: "hidden" });
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.ELEMENT });
        } else {
            await this.page.locator('[data-testid="update-button"]').click();
            await this.page.locator('[data-testid="update-button"]').waitFor({ state: "hidden", timeout: TIMEOUT.SLOW });
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.SLOW });
            await this.waitForDataTable();
        }
    },
};
