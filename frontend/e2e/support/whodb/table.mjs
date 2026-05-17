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

export const DATA_CONTAINER_SELECTOR = 'table, [role="grid"]';
export const DATA_ROW_SELECTOR = 'table tbody tr, [role="grid"] [role="row"]:has([role="gridcell"])';
export const DATA_CELL_SELECTOR = 'td, [role="gridcell"]';

/** Methods for table/storage-unit navigation, data extraction, sorting, pagination */
export const tableMethods = {
    /**
     * Get the visible data table/grid container.
     * @returns {import("@playwright/test").Locator}
     */
    dataContainer() {
        return this.page.locator(DATA_CONTAINER_SELECTOR).filter({ visible: true }).first();
    },

    /**
     * Get rendered data rows from either native table or ARIA grid markup.
     * @returns {import("@playwright/test").Locator}
     */
    dataRows() {
        return this.page.locator(DATA_ROW_SELECTOR);
    },

    /**
     * Get one rendered data row by index.
     * @param {number} rowIndex
     * @returns {import("@playwright/test").Locator}
     */
    dataRow(rowIndex) {
        return this.dataRows().nth(rowIndex);
    },

    /**
     * Get one rendered data cell by row and cell index.
     * @param {number} rowIndex
     * @param {number} cellIndex
     * @returns {import("@playwright/test").Locator}
     */
    dataCell(rowIndex, cellIndex) {
        return this.dataRow(rowIndex).locator(DATA_CELL_SELECTOR).nth(cellIndex);
    },

    /**
     * Wait for the data table/grid to render.
     * @param {{ waitForRows?: boolean, timeout?: number, rowTimeout?: number }} [options]
     */
    async waitForDataTable({ waitForRows = true, timeout = TIMEOUT.SLOW, rowTimeout = TIMEOUT.SLOW } = {}) {
        await this.dataContainer().waitFor({ state: "visible", timeout });
        if (waitForRows) {
            await this.dataRows().first().waitFor({ state: "visible", timeout: rowTimeout });
        }
    },

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
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: TIMEOUT.NAVIGATION });

        const card = this.page.locator(`[data-testid="storage-unit-card"][data-table-name="${tableName}"]`).first();
        await card.waitFor({ state: "visible", timeout: TIMEOUT.NAVIGATION });
        const exploreBtn = card.locator('[data-testid="explore-button"]');
        await exploreBtn.scrollIntoViewIfNeeded();
        await exploreBtn.waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
        await exploreBtn.click({ force: true });
    },

    /**
     * Get explore fields as array of [key, value] pairs
     * @param {{ expectedKeys?: string[] }} [options]
     * @returns {Promise<Array<[string, string]>>}
     */
    async getExploreFields({ expectedKeys = [] } = {}) {
        await this.page.locator('[data-testid="explore-fields"]').waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
        await this.page.locator('[data-testid="explore-fields"] [data-field-key]').first().waitFor({ timeout: TIMEOUT.ACTION });

        if (expectedKeys.length > 0) {
            await expect(async () => {
                const actualKeys = await this.page.evaluate(() => {
                    return Array.from(document.querySelectorAll('[data-testid="explore-fields"] [data-field-key]'))
                        .map((field) => field.getAttribute("data-field-key"))
                        .filter(Boolean);
                });
                for (const expectedKey of expectedKeys) {
                    expect(actualKeys).toContain(expectedKey);
                }
            }).toPass({ timeout: TIMEOUT.SLOW });
        }

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
        for (let attempt = 0; attempt < 2; attempt++) {
            await this.page.evaluate(() => {
                const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
                settings.storageUnitView = '"card"';
                localStorage.setItem("persist:settings", JSON.stringify(settings));
            });

            await this.page.evaluate((url) => { window.location.href = url; }, this.url("/storage-unit"));
            await this.page.waitForURL(/\/storage-unit/, { timeout: TIMEOUT.NAVIGATION });
            await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: TIMEOUT.NAVIGATION });

            const card = this.page.locator(`[data-testid="storage-unit-card"][data-table-name="${tableName}"]`).first();
            await card.waitFor({ state: "visible", timeout: TIMEOUT.NAVIGATION });
            const dataBtn = card.locator('[data-testid="data-button"]').first();
            await dataBtn.scrollIntoViewIfNeeded();
            await dataBtn.waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
            await dataBtn.click({ force: true });

            await this.page.waitForURL(/\/storage-unit\/explore/, { timeout: TIMEOUT.ACTION });
            const dataTableResult = this.waitForDataTable({ waitForRows })
                .then(() => ({ status: "ready" }))
                .catch((error) => ({ status: "error", error }));
            const errorBoundaryResult = this.page.getByRole("heading", { name: /whoops, something went wrong/i })
                .waitFor({ state: "visible", timeout: TIMEOUT.ELEMENT })
                .then(() => ({ status: "error-boundary" }))
                .catch(() => ({ status: "not-visible" }));
            const result = await Promise.race([dataTableResult, errorBoundaryResult]);

            if (result.status === "ready") {
                return;
            }
            if (attempt === 0 && result.status === "error-boundary") {
                continue;
            }
            if (result.status === "error") {
                throw result.error;
            }
            await this.waitForDataTable({ waitForRows });
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
        await expect(this.page.getByText(/No data available/i)).toBeVisible({ timeout: TIMEOUT.ACTION });
    },

    /**
     * Get table data as {columns, rows} from the first visible table
     * @returns {Promise<{columns: string[], rows: string[][]}>}
     */
    async getTableData() {
        await this.page.waitForFunction(() => {
            return !!document.querySelector("table tbody tr")
                || !!document.querySelector('[role="grid"] [role="row"] [role="gridcell"]');
        }, { timeout: TIMEOUT.SLOW });

        return await this.page.evaluate(() => {
            const table = Array.from(document.querySelectorAll("table"))
                .find((el) => el.querySelector("tbody tr"));
            if (table) {
                const columns = Array.from(table.querySelectorAll("th")).map((el) => el.innerText.trim());
                const rows = Array.from(table.querySelectorAll("tbody tr")).map((row) => {
                    return Array.from(row.querySelectorAll("td")).map((cell) => cell.innerText.trim());
                });

                return { columns, rows };
            }

            const grid = Array.from(document.querySelectorAll('[role="grid"]'))
                .find((el) => el.querySelector('[role="row"] [role="gridcell"]'));
            if (!grid) return { columns: [], rows: [] };

            const headerRow = Array.from(grid.querySelectorAll('[role="row"]'))
                .find((row) => row.querySelector('[role="columnheader"]'));
            const columns = headerRow
                ? Array.from(headerRow.querySelectorAll('[role="columnheader"]')).map((el) => el.innerText.trim())
                : [];
            const rows = Array.from(grid.querySelectorAll('[role="row"]'))
                .filter((row) => row.querySelector('[role="gridcell"]'))
                .map((row) => {
                    return Array.from(row.querySelectorAll('[role="gridcell"]')).map((cell) => cell.innerText.trim());
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

        const submitBtn = this.page.locator('[data-testid="submit-button"]');
        await submitBtn.waitFor({ timeout: TIMEOUT.ELEMENT });
        await submitBtn.click();
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
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: TIMEOUT.NAVIGATION });

        const elements = this.page.locator('[data-testid="storage-unit-name"]');
        const count = await elements.count();
        const result = [];
        for (let i = 0; i < count; i++) {
            result.push(await elements.nth(i).innerText());
        }
        return result;
    },
};
