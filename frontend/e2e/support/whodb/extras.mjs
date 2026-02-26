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

/** Methods for graph, export, import, mock data, query history, context menu, screenshots */
export const extrasMethods = {
    // ── Graph ─────────────────────────────────────────────────────────────

    async getGraph() {
        await this.page.locator(".react-flow__node").first().waitFor({ state: "visible", timeout: TIMEOUT.ACTION });

        await this.page.waitForFunction(() => {
            const container = document.querySelector(".react-flow");
            return container && !container.classList.contains("laying-out");
        }, { timeout: TIMEOUT.ELEMENT }).catch(() => {});

        await this.page.waitForTimeout(600);

        return await this.page.evaluate(() => {
            const nodeEls = document.querySelectorAll(".react-flow__node");
            const nodes = Array.from(nodeEls).map((el) => el.getAttribute("data-id"));

            const edgeEls = document.querySelectorAll(".react-flow__edge-path");
            const edges = Array.from(edgeEls)
                .map((el) => {
                    const source = el.getAttribute("data-edge-source");
                    const target = el.getAttribute("data-edge-target");
                    if (!source || !target) return null;
                    return { source, target };
                })
                .filter((edge) => edge !== null);

            const graph = {};
            nodes.forEach((node) => {
                const targets = edges.filter((edge) => edge.source === node).map((edge) => edge.target);
                graph[node] = [...new Set(targets)];
            });
            return graph;
        });
    },

    async getGraphNode(nodeId) {
        return await this.page.evaluate((nId) => {
            const el = document.querySelector(`[data-testid="rf__node-${nId}"]`);
            if (!el) return [];
            const result = [];
            const fields = el.querySelectorAll("[data-field-key]");
            fields.forEach((field) => {
                const key = field.getAttribute("data-field-key");
                const value = field.getAttribute("data-field-value");
                if (key && value) {
                    result.push([key, value]);
                }
            });
            return result;
        }, nodeId);
    },

    // ── Context Menu ──────────────────────────────────────────────────────

    async selectMockData() {
        await this.page.locator("table thead tr.cursor-context-menu").first().click({ button: "right", force: true });
        await this.page.waitForTimeout(200);
        const mockDataItem = this.page.locator('[data-testid="context-menu-mock-data"]');
        await mockDataItem.scrollIntoViewIfNeeded();
        await mockDataItem.waitFor({ timeout: TIMEOUT.ELEMENT });
        await mockDataItem.click({ force: true });
    },

    // ── Export ─────────────────────────────────────────────────────────────

    async selectExportFormat(format) {
        await this.page.locator('[data-testid="export-format-select"]').click();
        await this.page.locator(`[data-value="${format}"]`).click();
    },

    async selectExportDelimiter(delimiter) {
        await this.page.locator('[data-testid="export-delimiter-select"]').click();
        await this.page.locator(`[data-value="${delimiter}"]`).click();
    },

    async confirmExport() {
        await this.page.locator('[data-testid="export-confirm-button"]').click();
    },

    // ── Import ────────────────────────────────────────────────────────────

    async openImport(tableName) {
        await this.data(tableName);
        const importBtn = this.page.locator('[data-testid="import-button"]');
        await importBtn.waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
        await importBtn.click();
        await this.page.locator('[data-testid="import-dialog"]').waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
    },

    async selectImportMode(mode) {
        await this.page.locator('[data-testid="import-mode-select"]').click();
        await this.page.locator(`[data-value="${mode}"]`).click();
    },

    async uploadDataFile(filePath) {
        await this.page.locator('[data-testid="import-data-file-input"]').setInputFiles(filePath);
    },

    async uploadSqlFile(filePath) {
        await this.page.locator('[data-testid="import-sql-file-input"]').setInputFiles(filePath);
    },

    async waitForPreview() {
        const loading = this.page.locator('[data-testid="import-preview-loading"]');
        if (await loading.count() > 0) {
            await loading.waitFor({ state: "hidden", timeout: TIMEOUT.SHORT }).catch(() => {});
        }
        await this.page.locator('[data-testid="import-preview-section"]').waitFor({ timeout: TIMEOUT.NAVIGATION });
    },

    async getPreviewData() {
        await this.page.locator('[data-testid="import-preview-table"]').waitFor({ timeout: TIMEOUT.ACTION });

        return await this.page.locator('[data-testid="import-preview-table"]').evaluate((table) => {
            const columns = Array.from(table.querySelectorAll("thead th")).map((el) => el.innerText.trim());
            const rows = Array.from(table.querySelectorAll("tbody tr")).map((row) => {
                return Array.from(row.querySelectorAll("td")).map((cell) => cell.innerText.trim());
            });
            return { columns, rows };
        });
    },

    async confirmImportData() {
        const btn = this.page.locator('[data-testid="import-submit-button"]');
        await expect(btn).toBeEnabled();
        await btn.click();
    },

    async confirmSqlImport() {
        await this.page.locator('[data-testid="import-sql-confirm-checkbox"]').click({ force: true });
        const btn = this.page.locator('[data-testid="import-submit-button"]');
        await expect(btn).toBeEnabled();
        await btn.click();
    },

    async selectImportDataMode(mode) {
        await this.page.locator('[data-testid="import-data-mode-select"]').click();
        await this.page.locator(`[data-value="${mode}"]`).click();
    },

    async typeSqlInEditor(sql) {
        const selector = '[data-testid="import-sql-editor"] .cm-content';
        const editor = this.page.locator(selector);
        await editor.scrollIntoViewIfNeeded();
        await editor.waitFor({ state: "visible" });
        await editor.click();
        await editor.clear();
        await editor.fill(sql);
        await editor.blur();
        await this.page.waitForTimeout(100);
    },

    // ── Mock Data ─────────────────────────────────────────────────────────

    async setMockDataRows(count) {
        await this.page.locator('[data-testid="mock-data-rows-input"]').clear();
        await this.page.locator('[data-testid="mock-data-rows-input"]').fill(count.toString());
    },

    async setMockDataHandling(handling) {
        await this.page.locator('[data-testid="mock-data-handling-select"]').click();
        await this.page.locator(`[data-value="${handling}"]`).click();
    },

    async generateMockData() {
        const btn = this.page.locator('[data-testid="mock-data-generate-button"]');
        await expect(btn).toBeEnabled({ timeout: TIMEOUT.SLOW });
        await btn.scrollIntoViewIfNeeded();
        await btn.click({ force: true });
    },

    async confirmMockDataOverwrite() {
        await this.page.locator('[data-testid="mock-data-overwrite-button"]').click();
    },

    // ── Query History ─────────────────────────────────────────────────────

    async openQueryHistory(index = 0) {
        await this.page
            .locator(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="icon-button"]`)
            .first()
            .click();

        const menu = this.page.locator('[role="menu"]');
        await menu.waitFor({ state: "visible" });
        await menu.locator('[role="menuitem"]').filter({ hasText: "Query History" }).click();

        await this.page.waitForTimeout(500);

        await this.page.locator('[role="dialog"], .bg-background[data-state="open"]').waitFor({ state: "visible" });
        await expect(this.page.getByText("Query History")).toBeVisible();
    },

    async getQueryHistoryItems() {
        await this.page.locator('[role="dialog"] [data-slot="card"]').first().waitFor({ timeout: TIMEOUT.ACTION });
        const items = this.page.locator('[role="dialog"] [data-slot="card"]');
        const count = await items.count();
        const result = [];
        for (let i = 0; i < count; i++) {
            const queryText = (await items.nth(i).locator("pre code").innerText()).trim();
            result.push(queryText);
        }
        return result;
    },

    async copyQueryFromHistory(index = 0) {
        const context = this.page.context();
        await context.grantPermissions(["clipboard-read", "clipboard-write"]);

        const card = this.page.locator('[role="dialog"] [data-slot="card"]').nth(index);
        const textToCopy = (await card.locator("pre code").innerText()).trim();

        await card.locator('[data-testid="copy-to-clipboard-button"]').click({ force: true });

        const clipboardText = await this.page.evaluate(() => navigator.clipboard.readText());
        expect(clipboardText).toBe(textToCopy);
    },

    async cloneQueryToEditor(historyIndex = 0, targetCellIndex = 0) {
        const card = this.page.locator('[role="dialog"] [data-slot="card"]').nth(historyIndex);
        const expectedText = (await card.locator("pre code").innerText()).trim();
        await card.locator('[data-testid="clone-to-editor-button"]').click();

        await this.page.waitForTimeout(500);

        const dialogCount = await this.page.locator('[role="dialog"]').filter({ visible: true }).count();
        if (dialogCount > 0) {
            await this.page.locator('[role="dialog"]').waitFor({ state: "hidden", timeout: TIMEOUT.ACTION });
        }

        const editorSelector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${targetCellIndex}"] [data-testid="code-editor"] .cm-content`;
        await expect(this.page.locator(editorSelector)).toContainText(expectedText, { timeout: TIMEOUT.ACTION });

        await this.page.waitForTimeout(500);
    },

    async executeQueryFromHistory(index = 0) {
        await this.page
            .locator('[role="dialog"] [data-slot="card"]')
            .nth(index)
            .locator('[data-testid="run-history-button"]')
            .click({ force: true });
    },

    async closeQueryHistory() {
        const closeBtn = this.page.locator('[role="dialog"] button').filter({ hasText: "Close" });
        if (await closeBtn.count() > 0) {
            await closeBtn.click();
        } else {
            await this.page.keyboard.press("Escape");
        }

        const dialogCount = await this.page.locator('[role="dialog"]').count();
        if (dialogCount > 0) {
            await this.page.locator('[role="dialog"]').waitFor({ state: "hidden", timeout: TIMEOUT.ACTION });
        }

        await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.ELEMENT });
        await this.page.waitForTimeout(300);
    },

    async verifyQueryInEditor(index, expectedQuery) {
        const selector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="code-editor"] .cm-content`;
        await expect(this.page.locator(selector)).toContainText(expectedQuery);
    },

    async enableAutocomplete() {
        await this.page.evaluate(() => {
            delete window.__E2E_DISABLE_AUTOCOMPLETE;
        });
    },

    async disableAutocomplete() {
        await this.page.evaluate(() => {
            window.__E2E_DISABLE_AUTOCOMPLETE = true;
        });
    },

    // ── Screenshot Highlighting ───────────────────────────────────────────

    async highlightElement(
        selector,
        { borderColor = "#ca6f1e", borderWidth = "2px", borderRadius = "8px", padding = "4px", shadow = true } = {}
    ) {
        const el = this.page.locator(selector).first();
        await el.scrollIntoViewIfNeeded();
        await el.waitFor({ state: "visible" });

        await this.page.evaluate(
            ({ sel, borderColor, borderWidth, borderRadius, padding, shadow }) => {
                const element = document.querySelector(sel);
                if (!element) return;
                const rect = element.getBoundingClientRect();
                const overlay = document.createElement("div");
                const paddingPx = parseInt(padding);

                overlay.style.position = "fixed";
                overlay.style.top = `${rect.top - paddingPx}px`;
                overlay.style.left = `${rect.left - paddingPx}px`;
                overlay.style.width = `${rect.width + paddingPx * 2}px`;
                overlay.style.height = `${rect.height + paddingPx * 2}px`;
                overlay.style.border = `${borderWidth} solid ${borderColor}`;
                overlay.style.borderRadius = borderRadius;
                overlay.style.pointerEvents = "none";
                overlay.style.zIndex = "9999";

                if (shadow) {
                    overlay.style.boxShadow = `0 0 0 4px rgba(202, 111, 30, 0.1)`;
                }

                overlay.setAttribute("data-testid", "e2e-highlight-overlay");
                document.body.appendChild(overlay);
            },
            { sel: selector, borderColor, borderWidth, borderRadius, padding, shadow }
        );
    },

    async removeHighlights() {
        await this.page.evaluate(() => {
            const overlays = document.querySelectorAll('[data-testid="e2e-highlight-overlay"]');
            overlays.forEach((overlay) => overlay.remove());
        });
    },

    async screenshotWithHighlight(selector, screenshotName, highlightOptions = {}, screenshotOptions = {}) {
        await this.highlightElement(selector, highlightOptions);
        await this.page.waitForTimeout(300);
        await this.page.screenshot({ path: `${screenshotName}.png`, ...screenshotOptions });
        await this.removeHighlights();
    },
};
