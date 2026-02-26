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

/** Methods for row-level operations: add, delete, update, context menu */
export const rowsMethods = {
    /**
     * Add a row to the table
     * @param {Object|string} data
     * @param {boolean} isSingleInput
     */
    async addRow(data, isSingleInput = false) {
        await this.page.locator('[data-testid="add-row-button"]').click();

        if (isSingleInput) {
            const jsonString = typeof data === "string" ? data : JSON.stringify(data, null, 2);
            const editorContainer = this.page.locator('[data-testid="add-row-field-document"] .cm-editor');
            await editorContainer.waitFor({ timeout: TIMEOUT.ELEMENT });
            await editorContainer.locator('[role="textbox"]').fill(jsonString);
        } else {
            for (const [key, value] of Object.entries(data)) {
                await this.page.locator(`[data-testid="add-row-field-${key}"] input`).clear();
                await this.page.locator(`[data-testid="add-row-field-${key}"] input`).fill(value);
            }
        }

        await this.page.locator('[data-testid="submit-add-row-button"]').click();
        await this.page.locator('[data-testid="submit-add-row-button"]').waitFor({ state: "hidden", timeout: TIMEOUT.ACTION });
        await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.ELEMENT });
        await this.page.waitForTimeout(500);
        await this.page.locator("table tbody").waitFor({ state: "visible" });
    },

    /**
     * Waits for a row containing a specific value in a given column
     * @param {number} columnIndex
     * @param {string} expectedValue
     * @param {Object} options
     * @returns {Promise<number>}
     */
    async waitForRowValue(columnIndex, expectedValue, options = {}) {
        const timeout = options.timeout || 10000;
        const expectedStr = String(expectedValue).trim();

        await expect(async () => {
            const rows = await this.page.locator("table tbody tr").all();
            let found = false;
            for (const row of rows) {
                const cell = row.locator("td").nth(columnIndex);
                const cellText = (await cell.innerText()).trim();
                if (cellText === expectedStr) {
                    found = true;
                    break;
                }
            }
            expect(found).toBe(true);
        }).toPass({ timeout });

        const rows = await this.page.locator("table tbody tr").all();
        for (let i = 0; i < rows.length; i++) {
            const cell = rows[i].locator("td").nth(columnIndex);
            const cellText = (await cell.innerText()).trim();
            if (cellText === expectedStr) {
                return i;
            }
        }
        return -1;
    },

    /**
     * Waits for a row containing a specific value anywhere in the row
     * @param {string} expectedValue
     * @param {Object} options
     * @returns {Promise<number>}
     */
    async waitForRowContaining(expectedValue, options = {}) {
        const timeout = options.timeout || 10000;
        const caseSensitive = options.caseSensitive || false;
        const searchStr = caseSensitive ? String(expectedValue) : String(expectedValue).toLowerCase();

        await expect(async () => {
            const rows = await this.page.locator("table tbody tr").all();
            let found = false;
            for (const row of rows) {
                const rowText = caseSensitive ? await row.innerText() : (await row.innerText()).toLowerCase();
                if (rowText.includes(searchStr)) {
                    found = true;
                    break;
                }
            }
            expect(found).toBe(true);
        }).toPass({ timeout });

        const rows = await this.page.locator("table tbody tr").all();
        for (let i = 0; i < rows.length; i++) {
            const rowText = caseSensitive ? await rows[i].innerText() : (await rows[i].innerText()).toLowerCase();
            if (rowText.includes(searchStr)) {
                return i;
            }
        }
        return -1;
    },

    /**
     * Open context menu on a table row with retry logic
     * @param {number} rowIndex
     * @param {number} maxRetries
     */
    async openContextMenu(rowIndex, maxRetries = 3) {
        for (let attempt = 1; attempt <= maxRetries; attempt++) {
            const targetRow = this.page.locator("table tbody tr").nth(rowIndex);
            await targetRow.scrollIntoViewIfNeeded();
            await this.page.waitForTimeout(200);
            await targetRow.click({ button: "right", force: true });
            await this.page.waitForTimeout(300);

            const menuExists =
                (await this.page.locator('[data-testid="context-menu-edit-row"], [data-testid="context-menu-more-actions"]').count()) > 0;

            if (menuExists) {
                return;
            }

            if (attempt < maxRetries) {
                await this.page.mouse.click(0, 0);
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
     */
    async deleteRow(rowIndex) {
        const initialRowCount = await this.page.locator("table tbody tr").count();
        expect(await this.page.locator("table tbody tr").count()).toBeGreaterThan(rowIndex);

        await this.openContextMenu(rowIndex);

        const deleteBtn = this.page.locator('[data-testid="context-menu-delete-row"]');
        await deleteBtn.scrollIntoViewIfNeeded();
        await deleteBtn.waitFor({ timeout: TIMEOUT.ELEMENT });
        await deleteBtn.click({ force: true });

        await expect(this.page.locator("table tbody tr")).toHaveCount(initialRowCount - 1, { timeout: TIMEOUT.ACTION });
    },

    /**
     * Update a row via context menu
     * @param {number} rowIndex
     * @param {number} columnIndex
     * @param {string} text
     * @param {boolean} cancel
     */
    async updateRow(rowIndex, columnIndex, text, cancel = true) {
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
            await this.page.keyboard.press("Escape");
            await this.page.getByText("Edit Row").first().waitFor({ state: "hidden" });
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.ELEMENT });
        } else {
            await this.page.locator('[data-testid="update-button"]').click();
            await this.page.locator('[data-testid="update-button"]').waitFor({ state: "hidden" });
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.ELEMENT });
        }
    },
};
