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

/**
 * Extract text from an element, converting HTML to plain text
 * @param {import("@playwright/test").Locator} locator
 * @returns {Promise<string>}
 */
async function extractText(locator) {
    const html = await locator.innerHTML();
    return html
        .replace(/<br\s*\/?>/g, "\n")
        .replace(/<\/(p|div|li|h[1-6])>/g, "\n")
        .replace(/&nbsp;/g, " ")
        .replace(/<[^>]*>/g, "")
        .trim();
}

/** Methods for scratchpad cells and pages */
export const scratchpadMethods = {
    /**
     * Add a cell after the specified index
     * @param {number} afterIndex
     */
    async addCell(afterIndex) {
        await this.page
            .locator(`[role="tabpanel"][data-state="active"] [data-testid="cell-${afterIndex}"] [data-testid="add-cell-button"]`)
            .click();
    },

    /**
     * Remove a cell at the specified index
     * @param {number} index
     */
    async removeCell(index) {
        await this.page
            .locator(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="delete-cell-button"]`)
            .click();
    },

    /**
     * Write code into a cell's CodeMirror editor
     * @param {number} index
     * @param {string} text
     */
    async writeCode(index, text) {
        const selector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] .cm-content`;
        const editor = this.page.locator(selector);
        await editor.scrollIntoViewIfNeeded();
        await editor.waitFor({ state: "visible" });
        await editor.click();
        await editor.clear();
        await editor.fill(text);
        await editor.blur();
        await this.page.waitForTimeout(100);
    },

    /**
     * Run code in a specific cell
     * @param {number} index
     */
    async runCode(index) {
        const buttonSelector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="query-cell-button"]`;
        await this.page.locator(buttonSelector).first().click({ force: true });

        const cellLocator = this.page.locator(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"]`);
        await cellLocator
            .locator('[data-testid="cell-query-output"], [data-testid="cell-action-output"], [data-testid="cell-error"]')
            .first()
            .waitFor({ timeout: 5000 });
    },

    /**
     * Get cell query output as {columns, rows}
     * @param {number} index
     * @returns {Promise<{columns: string[], rows: string[][]}>}
     */
    async getCellQueryOutput(index) {
        const tableLocator = this.page.locator(
            `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="cell-query-output"] table`
        );
        await tableLocator.waitFor({ timeout: 10000 });

        return await tableLocator.evaluate((table) => {
            const columns = Array.from(table.querySelectorAll("th")).map((el) => el.innerText.trim());
            const rows = Array.from(table.querySelectorAll("tbody tr")).map((row) => {
                return Array.from(row.querySelectorAll("td")).map((cell) => cell.innerText.trim());
            });
            return { columns, rows };
        });
    },

    /**
     * Get cell action output text
     * @param {number} index
     * @returns {Promise<string>}
     */
    async getCellActionOutput(index) {
        const el = this.page.locator(
            `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="cell-action-output"]`
        );
        await el.waitFor({ timeout: 10000 });
        return await extractText(el);
    },

    /**
     * Get cell error text
     * @param {number} index
     * @returns {Promise<string>}
     */
    async getCellError(index) {
        const el = this.page.locator(
            `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="cell-error"]`
        );
        await el.waitFor({ timeout: 10000 });
        await el.waitFor({ state: "visible" });
        const text = await el.innerText();
        return text.replace(/^Error\s*/i, "").trim();
    },

    /**
     * Add a new scratchpad page
     */
    async addScratchpadPage() {
        await this.page.locator('[data-testid="add-page-button"]').click();
    },

    /**
     * Get scratchpad page names
     * @returns {Promise<string[]>}
     */
    async getScratchpadPages() {
        const els = this.page.locator('[data-testid="page-tabs"] [data-testid*="page-tab"]');
        const count = await els.count();
        const result = [];
        for (let i = 0; i < count; i++) {
            const text = (await els.nth(i).innerText()).trim();
            if (text.length > 0) {
                result.push(text);
            }
        }
        return result;
    },

    /**
     * Delete a scratchpad page
     * @param {number} index
     * @param {boolean} cancel
     */
    async deleteScratchpadPage(index, cancel = true) {
        await this.page.locator(`[data-testid="delete-page-tab-${index}"]`).click();

        if (cancel) {
            await this.page.locator('[data-testid="delete-page-button-cancel"]').click();
        } else {
            await this.page.locator('[data-testid="delete-page-button-confirm"]').click();
        }
    },
};
