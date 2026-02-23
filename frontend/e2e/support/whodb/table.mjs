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

/** Methods for table/storage-unit navigation, data extraction, sorting, pagination */
export const tableMethods = {
    /**
     * Navigate to storage unit page and click explore on a specific table
     * @param {string} tableName
     */
    async explore(tableName) {
        await this.page.evaluate(() => {
            const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
            settings.storageUnitView = '"card"';
            localStorage.setItem("persist:settings", JSON.stringify(settings));
        });

        await this.page.goto(this.url("/storage-unit"));
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

        const card = this.page.locator(`[data-testid="storage-unit-card"][data-table-name="${tableName}"]`).first();
        const exploreBtn = card.locator('[data-testid="explore-button"]');
        await exploreBtn.scrollIntoViewIfNeeded();
        await exploreBtn.waitFor({ state: "visible", timeout: 10000 });
        await exploreBtn.click({ force: true });
    },

    /**
     * Get explore fields as array of [key, value] pairs
     * @returns {Promise<Array<[string, string]>>}
     */
    async getExploreFields() {
        await this.page.locator('[data-testid="explore-fields"]').waitFor({ state: "visible", timeout: 10000 });
        await this.page.locator('[data-testid="explore-fields"] h3').waitFor({ timeout: 10000 });

        return await this.page.evaluate(() => {
            const result = [];
            const fields = document.querySelectorAll('[data-testid="explore-fields"] [data-field-key]');
            fields.forEach((field) => {
                const key = field.getAttribute("data-field-key");
                const value = field.getAttribute("data-field-value");
                if (key && value) {
                    result.push([key, value]);
                }
            });
            return result;
        });
    },

    /**
     * Navigate to storage unit page and click data button for a specific table
     * @param {string} tableName
     * @param {{ waitForRows?: boolean }} [options]
     */
    async data(tableName, { waitForRows = true } = {}) {
        await this.page.evaluate(() => {
            const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
            settings.storageUnitView = '"card"';
            localStorage.setItem("persist:settings", JSON.stringify(settings));
        });

        await this.page.evaluate((url) => { window.location.href = url; }, this.url("/storage-unit"));
        await this.page.waitForURL(/\/storage-unit/, { timeout: 15000 });
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

        const card = this.page.locator(`[data-testid="storage-unit-card"][data-table-name="${tableName}"]`).first();
        const dataBtn = card.locator('[data-testid="data-button"]').first();
        await dataBtn.scrollIntoViewIfNeeded();
        await dataBtn.waitFor({ state: "visible", timeout: 10000 });
        await dataBtn.click({ force: true });

        await this.page.waitForURL(/\/storage-unit\/explore/, { timeout: 10000 });
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ state: "hidden", timeout: 5000 });
        await this.page.locator("table").filter({ visible: true }).waitFor({ timeout: 10000 });
        if (waitForRows) {
            await this.page.locator("table").filter({ visible: true }).locator("tbody tr").first().waitFor({ timeout: 30000 });
        }
    },

    /**
     * Sort table by column name or index
     * @param {string|number} columnNameOrIndex
     */
    async sortBy(columnNameOrIndex) {
        if (typeof columnNameOrIndex === "string") {
            await this.page.locator(`[data-column-name="${columnNameOrIndex}"]`).click();
        } else {
            await this.page.locator("th").nth(columnNameOrIndex + 1).click();
        }
    },

    /**
     * Get all table columns as array of {name, sortDirection} objects
     * @returns {Promise<Array<{name: string, sortDirection: string|null}>>}
     */
    async getTableColumns() {
        const headers = this.page.locator("[data-column-name]");
        const count = await headers.count();
        const result = [];
        for (let i = 0; i < count; i++) {
            const el = headers.nth(i);
            const name = await el.getAttribute("data-column-name");
            const sortDirection = await el.getAttribute("data-sort-direction");
            result.push({ name, sortDirection: sortDirection || null });
        }
        return result;
    },

    /**
     * Assert that "No data available" text is visible
     */
    async assertNoDataAvailable() {
        await expect(this.page.getByText(/No data available/i)).toBeVisible({ timeout: 10000 });
    },

    /**
     * Get table data as {columns, rows} from the first visible table
     * @returns {Promise<{columns: string[], rows: string[][]}>}
     */
    async getTableData() {
        await this.page.locator("table").filter({ visible: true }).waitFor({ timeout: 10000 });
        await this.page.locator("table").filter({ visible: true }).locator("tbody tr").first().waitFor({ timeout: 10000 });

        return await this.page.evaluate(() => {
            const table = document.querySelector("table");
            if (!table) return { columns: [], rows: [] };

            const columns = Array.from(table.querySelectorAll("th")).map((el) => el.innerText.trim());
            const rows = Array.from(table.querySelectorAll("tbody tr")).map((row) => {
                return Array.from(row.querySelectorAll("td")).map((cell) => cell.innerText.trim());
            });

            return { columns, rows };
        });
    },

    /**
     * Set the table page size via dropdown
     * @param {string|number} pageSize
     */
    async setTablePageSize(pageSize) {
        await this.page.locator('[data-testid="table-page-size"]').click();
        await this.page.locator(`[role="option"][data-value="${pageSize}"]`).click();
    },

    /**
     * Get the current table page size text
     * @returns {Promise<string>}
     */
    async getTablePageSize() {
        return (await this.page.locator('[data-testid="table-page-size"]').innerText()).trim();
    },

    /**
     * Submit the table
     */
    async submitTable() {
        await this.page.keyboard.press("Escape");
        await this.page.waitForTimeout(200);

        const submitBtn = this.page.locator('[data-testid="submit-button"]');
        await submitBtn.waitFor({ timeout: 5000 });
        await submitBtn.click();
        await this.page.waitForTimeout(200);
    },

    /**
     * Search the table
     * @param {string} search
     */
    async searchTable(search) {
        await this.page.locator('[data-testid="table-search"]').clear();
        await this.page.locator('[data-testid="table-search"]').pressSequentially(search);
        await this.page.locator('[data-testid="table-search"]').press("Enter");
        await expect(this.page.locator('[data-testid="table-search"]')).toHaveValue(search);
    },

    /**
     * Get page numbers as array of strings
     * @returns {Promise<string[]>}
     */
    async getPageNumbers() {
        const els = this.page.locator('[data-testid="table-page-number"]');
        const count = await els.count();
        const result = [];
        for (let i = 0; i < count; i++) {
            result.push((await els.nth(i).innerText()).trim());
        }
        return result;
    },

    /**
     * Get highlighted table cells
     * @returns {import("@playwright/test").Locator}
     */
    getHighlightedCell(options = {}) {
        return this.page.locator("td.table-search-highlight");
    },

    /**
     * Get highlighted rows as array of cell text arrays
     * @returns {Promise<string[][]>}
     */
    async getHighlightedRows() {
        return await this.page.evaluate(() => {
            const rows = [];
            const highlightedRows = document.querySelectorAll("tr:has(td.table-search-highlight)");
            highlightedRows.forEach((row) => {
                const cells = Array.from(row.querySelectorAll("td")).map((cell) => cell.innerText.trim());
                rows.push(cells);
            });
            return rows;
        });
    },

    /**
     * Get all table names from the storage unit page
     * @returns {Promise<string[]>}
     */
    async getTables() {
        await this.page.evaluate(() => {
            const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
            settings.storageUnitView = '"card"';
            localStorage.setItem("persist:settings", JSON.stringify(settings));
        });

        await this.page.goto(this.url("/storage-unit"));
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

        const elements = this.page.locator('[data-testid="storage-unit-name"]');
        const count = await elements.count();
        const result = [];
        for (let i = 0; i < count; i++) {
            result.push(await elements.nth(i).innerText());
        }
        return result;
    },
};
