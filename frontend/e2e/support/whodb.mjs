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

import fs from "fs";
import path from "path";
import { expect } from "@playwright/test";

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";

// Session cache directory — stores login state per database to avoid re-authenticating
const SESSION_CACHE_DIR = path.resolve(process.cwd(), "e2e", ".session-cache");

/**
 * Get the cached session file path for a database connection.
 * Matches Cypress's cy.session() pattern with cacheAcrossSpecs.
 */
function getSessionPath(databaseType, hostname, database, schema) {
    const key = [databaseType, hostname || "default", database || "default", schema || "default"].join("-");
    return path.join(SESSION_CACHE_DIR, `${key}.json`);
}

// ============================================================================
// Platform-Aware Keyboard Shortcuts
// ============================================================================
// The app uses different modifier keys based on platform:
// - Mac: Ctrl+Number for sidebar nav, Cmd+K for command palette, Cmd+X for other shortcuts
// - Windows/Linux: Alt+Number for sidebar nav, Ctrl+K for command palette, Ctrl+X for other shortcuts

/**
 * Detect if the test is running on macOS
 * Uses process.platform which returns 'darwin' for macOS
 */
const isMac = process.platform === "darwin";

/**
 * Platform-aware keyboard shortcuts object
 * Use these instead of hardcoded modifiers to ensure tests work cross-platform
 */
const platformKeys = {
    // Navigation shortcuts (sidebar navigation)
    // Mac: Ctrl+Number, Windows/Linux: Alt+Number
    navMod: isMac ? "Control" : "Alt",

    // Command/Control modifier for general shortcuts
    // Mac: Cmd (meta), Windows/Linux: Ctrl
    cmdMod: isMac ? "Meta" : "Control",
};

/**
 * Get the navigation key combo for a number (1-9)
 * @param {number} num - The number key (1-9)
 * @returns {string} The Playwright key sequence
 */
function getNavShortcut(num) {
    return `${platformKeys.navMod}+${num}`;
}

/**
 * Get the command/control key combo for a letter
 * @param {string} key - The letter key
 * @returns {string} The Playwright key sequence
 */
function getCmdShortcut(key) {
    return `${platformKeys.cmdMod}+${key}`;
}

/**
 * Extract text from an element, converting HTML to plain text
 * @param {import("@playwright/test").Locator} locator
 * @returns {Promise<string>}
 */
async function extractText(locator) {
    const html = await locator.innerHTML();
    return html
        .replace(/<br\s*\/?>/g, "\n") // Replace <br> with new lines
        .replace(/<\/(p|div|li|h[1-6])>/g, "\n") // New line after block elements
        .replace(/&nbsp;/g, " ") // Replace non-breaking spaces
        .replace(/<[^>]*>/g, "") // Remove remaining HTML tags
        .trim();
}

export { platformKeys, isMac };

export class WhoDB {
    /**
     * @param {import("@playwright/test").Page} page
     */
    constructor(page) {
        this.page = page;
        /** @type {Array<Object>|null} Test-scoped chat response storage */
        this._chatMockResponses = null;
    }

    /**
     * Resolve a relative path against the base URL
     * @param {string} path
     * @returns {string}
     */
    url(path) {
        return `${BASE_URL}${path.startsWith("/") ? path : "/" + path}`;
    }

    // ============================================================================
    // Platform-Aware Keyboard Shortcuts
    // ============================================================================

    /**
     * Type a platform-aware navigation shortcut (Alt/Ctrl + Number)
     * @param {number} num - The number key (1-4)
     */
    async typeNavShortcut(num) {
        await this.page.keyboard.press(getNavShortcut(num));
    }

    /**
     * Type a platform-aware command shortcut (Cmd/Ctrl + Key)
     * @param {string} key - The key to combine with Cmd/Ctrl
     * @param {Object} options - Additional options like shift
     */
    async typeCmdShortcut(key, options = {}) {
        let combo = platformKeys.cmdMod;
        if (options.shift) {
            combo += "+Shift";
        }
        combo += `+${key}`;
        await this.page.keyboard.press(combo);
    }

    // ============================================================================
    // Auth & Navigation
    // ============================================================================

    /**
     * Navigate to a route
     * @param {string} route
     */
    async goto(route) {
        await this.page.goto(this.url(`/${route}`));
    }

    /**
     * Login to the application
     * @param {string} databaseType
     * @param {string} hostname
     * @param {string} username
     * @param {string} password
     * @param {string} database
     * @param {Object} advanced
     */
    async login(databaseType, hostname, username, password, database, advanced = {}, schema = null) {
        // Try to restore cached session first (matches Cypress cy.session() with cacheAcrossSpecs)
        const sessionFile = getSessionPath(databaseType, hostname, database, schema);
        if (fs.existsSync(sessionFile)) {
            try {
                const cached = JSON.parse(fs.readFileSync(sessionFile, "utf-8"));
                // Restore localStorage and cookies
                await this.page.goto(this.url("/login"));
                await this.page.evaluate((storage) => {
                    for (const [k, v] of Object.entries(storage)) {
                        localStorage.setItem(k, v);
                    }
                }, cached.localStorage);
                await this.page.context().addCookies(cached.cookies);

                // Validate the session is still valid
                await this.page.goto(this.url("/storage-unit"), { waitUntil: "domcontentloaded" });
                const valid = await this.page.locator('[data-testid="sidebar-profile"]')
                    .waitFor({ timeout: 5000 })
                    .then(() => true)
                    .catch(() => false);

                if (valid) {
                    return; // Session restored successfully
                }
                // Session invalid — fall through to full login
            } catch {
                // Cache corrupted — fall through to full login
            }
        }

        // Full login flow
        await this.page.goto(this.url("/login"));

        // Clear stale session state from prior test's logout
        await this.page.evaluate(() => {
            localStorage.clear();
            localStorage.setItem("whodb.analytics.consent", "denied");
            sessionStorage.clear();
        });
        await this.page.context().clearCookies();
        await this.page.goto(this.url("/login"));

        // Poll for telemetry modal and dismiss if it appears (handles async React rendering)
        for (let attempt = 0; attempt < 5; attempt++) {
            const btn = this.page.locator("button").filter({ hasText: "Disable Telemetry" });
            const count = await btn.count();
            if (count > 0) {
                await btn.click();
                break;
            }
            await this.page.waitForTimeout(300);
        }

        if (databaseType) {
            await this.page.locator('[data-testid="database-type-select"]').click();
            await this.page.locator(`[data-value="${databaseType}"]`).click();
        }

        if (hostname !== undefined && hostname !== null) {
            await this.page.locator('[data-testid="hostname"]').clear();
            if (hostname !== "") {
                await this.page.locator('[data-testid="hostname"]').fill(hostname);
            }
        }

        if (username !== undefined) {
            await this.page.locator('[data-testid="username"]').clear();
            if (username != null && username !== "") {
                await this.page.locator('[data-testid="username"]').fill(username);
            }
        }

        if (password !== undefined) {
            await this.page.locator('[data-testid="password"]').clear();
            if (password != null && password !== "") {
                await this.page.locator('[data-testid="password"]').fill(password);
            }
        }

        if (database !== undefined) {
            if (databaseType === "Sqlite3") {
                await expect(this.page.locator('[data-testid="database"]')).toBeEnabled({ timeout: 10000 });
                await this.page.locator('[data-testid="database"]').click();
                await this.page.locator('[role="option"]').first().waitFor({ timeout: 10000 });
                await this.page.locator(`[data-value="${database}"]`).click();
            } else {
                await this.page.locator('[data-testid="database"]').clear();
                if (database !== null && database !== "") {
                    await this.page.locator('[data-testid="database"]').fill(database);
                }
            }
        }

        // Handle advanced options (including SSL)
        const ssl = advanced.ssl || {};
        const advancedFields = { ...advanced };
        delete advancedFields.ssl;

        const hasAdvancedOptions = Object.keys(advancedFields).length > 0 || Object.keys(ssl).length > 0;

        if (hasAdvancedOptions) {
            await this.page.locator('[data-testid="advanced-button"]').click();

            for (const [key, value] of Object.entries(advancedFields)) {
                await this.page.locator(`[data-testid="${key}-input"]`).clear();
                if (value != null && value !== "") {
                    await this.page.locator(`[data-testid="${key}-input"]`).fill(value);
                }
            }

            if (ssl.mode) {
                await this.page.locator('[data-testid="ssl-mode-select"]').click();
                await this.page.locator(`[data-value="${ssl.mode}"]`).click();
            }

            if (ssl.caCertContent) {
                await this.page.getByRole("button", { name: "Paste PEM" }).first().click();
                await this.page.locator('[data-testid="ssl-ca-certificate-content"]').fill(ssl.caCertContent);
            }
        }

        // Wait for login API response
        const loginResponsePromise = this.page.waitForResponse(
            (response) => response.url().includes("/api/query") && response.request().method() === "POST",
            { timeout: 60000 }
        );

        await this.page.locator('[data-testid="login-button"]').click();

        const loginResponse = await loginResponsePromise;
        const body = await loginResponse.json();
        if (body?.errors) {
            console.log("Login API returned errors:", JSON.stringify(body.errors));
        }

        // Wait for successful login
        await this.page.locator('[data-testid="sidebar-profile"]').waitFor({ timeout: 30000 });

        // Cache the session for reuse across tests (matches Cypress cacheAcrossSpecs)
        try {
            if (!fs.existsSync(SESSION_CACHE_DIR)) {
                fs.mkdirSync(SESSION_CACHE_DIR, { recursive: true });
            }
            const localStorage = await this.page.evaluate(() => {
                const items = {};
                for (let i = 0; i < window.localStorage.length; i++) {
                    const key = window.localStorage.key(i);
                    items[key] = window.localStorage.getItem(key);
                }
                return items;
            });
            const cookies = await this.page.context().cookies();
            fs.writeFileSync(sessionFile, JSON.stringify({ localStorage, cookies }));
        } catch {
            // Non-critical — caching failure doesn't affect test
        }
    }

    /**
     * Set advanced options (no-op placeholder)
     * @param {string} type
     * @param {string} value
     */
    async setAdvanced(type, value) {
        // No-op - matches Cypress implementation
    }

    /**
     * Select a database from sidebar dropdown
     * @param {string} value
     */
    async selectDatabase(value) {
        await this.page.locator('[data-testid="sidebar-database"]').click();
        await this.page.locator(`[data-value="${value}"]`).click();
    }

    /**
     * Select a schema from sidebar dropdown
     * @param {string} value
     */
    async selectSchema(value) {
        await this.page.locator('[data-testid="sidebar-schema"]').click();
        await this.page.locator(`[data-value="${value}"]`).click();
        // Wait for the app to reload storage units after schema change
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ state: "visible", timeout: 15000 });
    }

    // ============================================================================
    // Tables & Storage Units
    // ============================================================================

    /**
     * Navigate to storage unit page and click explore on a specific table
     * @param {string} tableName
     */
    async explore(tableName) {
        // Ensure card view is set for consistent test behavior
        await this.page.evaluate(() => {
            const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
            settings.storageUnitView = '"card"';
            localStorage.setItem("persist:settings", JSON.stringify(settings));
        });

        // Visit the storage-unit page (will use card view)
        await this.page.goto(this.url("/storage-unit"));
        // Wait for cards to load
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

        // Find the card by data-table-name attribute for reliable selection
        const card = this.page.locator(`[data-testid="storage-unit-card"][data-table-name="${tableName}"]`).first();
        const exploreBtn = card.locator('[data-testid="explore-button"]');
        await exploreBtn.scrollIntoViewIfNeeded();
        await exploreBtn.waitFor({ state: "visible", timeout: 10000 });
        await exploreBtn.click({ force: true });
    }

    /**
     * Get explore fields as array of [key, value] pairs
     * @returns {Promise<Array<[string, string]>>}
     */
    async getExploreFields() {
        // Wait for the explore-fields panel to be visible (Sheet needs to open)
        await this.page.locator('[data-testid="explore-fields"]').waitFor({ state: "visible", timeout: 10000 });

        // Wait for columns section to appear - it has an h3 header that renders when columns are loaded
        // The columns are fetched via async API call (fetchColumnsBatch) on page load
        await this.page.locator('[data-testid="explore-fields"] h3').waitFor({ timeout: 10000 });

        // Returns a list of [key, value] arrays from the explore fields panel
        // Uses data-field-key and data-field-value attributes for reliable extraction
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
    }

    /**
     * Navigate to storage unit page and click data button for a specific table
     * @param {string} tableName
     */
    async data(tableName) {
        // Ensure card view is set for consistent test behavior
        await this.page.evaluate(() => {
            const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
            settings.storageUnitView = '"card"';
            localStorage.setItem("persist:settings", JSON.stringify(settings));
        });

        // Visit the storage-unit page (will use card view)
        await this.page.goto(this.url("/storage-unit"));
        // Wait for cards to load
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

        // Find the card by data-table-name attribute for reliable selection
        const card = this.page.locator(`[data-testid="storage-unit-card"][data-table-name="${tableName}"]`).first();
        const dataBtn = card.locator('[data-testid="data-button"]').first();
        await dataBtn.scrollIntoViewIfNeeded();
        await dataBtn.waitFor({ state: "visible", timeout: 10000 });
        await dataBtn.click({ force: true });

        // Wait for URL to change to explore page
        await this.page.waitForURL(/\/storage-unit\/explore/, { timeout: 10000 });
        // Wait for the page to stabilize - ensure we're not on the list page anymore
        // The list view has a hidden table, so we must check cards are gone
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ state: "hidden", timeout: 5000 });
        // Wait for a VISIBLE table (not the hidden list view table)
        await this.page.locator("table").filter({ visible: true }).waitFor({ timeout: 10000 });
        await this.page.locator("table").filter({ visible: true }).locator("tbody tr").first().waitFor({ timeout: 15000 });
    }

    /**
     * Sort table by column name or index
     * @param {string|number} columnNameOrIndex
     */
    async sortBy(columnNameOrIndex) {
        // Support both column name (string) and index (number)
        if (typeof columnNameOrIndex === "string") {
            await this.page.locator(`[data-column-name="${columnNameOrIndex}"]`).click();
        } else {
            await this.page.locator("th").nth(columnNameOrIndex + 1).click();
        }
    }

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
    }

    /**
     * Assert that "No data available" text is visible
     */
    async assertNoDataAvailable() {
        await expect(this.page.getByText(/No data available/i)).toBeVisible({ timeout: 10000 });
    }

    /**
     * Get table data as {columns, rows} from the first visible table
     * @returns {Promise<{columns: string[], rows: string[][]}>}
     */
    async getTableData() {
        // First wait for a VISIBLE table to exist (not hidden list view tables)
        await this.page.locator("table").filter({ visible: true }).waitFor({ timeout: 10000 });

        // Wait for at least one table row to be present
        await this.page.locator("table").filter({ visible: true }).locator("tbody tr").first().waitFor({ timeout: 10000 });

        // Now get the visible table and extract data
        return await this.page.evaluate(() => {
            const table = document.querySelector("table");
            if (!table) return { columns: [], rows: [] };

            const columns = Array.from(table.querySelectorAll("th")).map((el) => el.innerText.trim());

            const rows = Array.from(table.querySelectorAll("tbody tr")).map((row) => {
                return Array.from(row.querySelectorAll("td")).map((cell) => cell.innerText.trim());
            });

            return { columns, rows };
        });
    }

    /**
     * Set the table page size via dropdown
     * @param {string|number} pageSize
     */
    async setTablePageSize(pageSize) {
        await this.page.locator('[data-testid="table-page-size"]').click();
        await this.page.locator(`[role="option"][data-value="${pageSize}"]`).click();
    }

    /**
     * Get the current table page size text
     * @returns {Promise<string>}
     */
    async getTablePageSize() {
        return (await this.page.locator('[data-testid="table-page-size"]').innerText()).trim();
    }

    /**
     * Submit the table (handles both dialog and standalone submit button)
     */
    async submitTable() {
        const dialogVisible = await this.page.locator('[role="dialog"]').filter({ visible: true }).count();
        if (dialogVisible > 0) {
            await this.page.locator('[role="dialog"]').filter({ visible: true }).locator('[data-testid="add-conditions-button"]').click();
            await this.page.waitForTimeout(300);
        }
        const submitBtn = this.page.locator('[data-testid="submit-button"]').filter({ visible: true });
        const btnCount = await submitBtn.count();
        if (btnCount > 0) {
            await submitBtn.click({ force: true });
            await this.page.waitForTimeout(200);
        }
    }

    // ============================================================================
    // Where Conditions
    // ============================================================================

    /**
     * Add where conditions to the table
     * @param {Array<[string, string, string]>} fieldArray - Array of [key, operator, value] tuples
     */
    async whereTable(fieldArray) {
        await this.page.locator('[data-testid="where-button"]').click();

        // Wait for the dialog/sheet to be visible
        await this.page.waitForTimeout(500);

        // Detect which mode we're in by checking what's visible
        const isSheetMode =
            (await this.page.locator('[role="dialog"]').count()) > 0 &&
            (await this.page.locator('[data-testid*="sheet-field"]').count()) > 0;
        const isPopoverMode = (await this.page.locator('[data-testid="field-key"]').count()) > 0;

        console.log(`Where condition mode detected: ${isSheetMode ? "sheet" : isPopoverMode ? "popover" : "unknown"}`);

        for (const [key, operator, value] of fieldArray) {
            console.log(`Adding condition: ${key} ${operator} ${value}`);

            if (isSheetMode) {
                // Sheet mode - always uses index 0 for new conditions
                // Try both with and without index for compatibility
                if ((await this.page.locator('[data-testid="sheet-field-key-0"]').count()) > 0) {
                    await this.page.locator('[data-testid="sheet-field-key-0"]').click();
                } else if ((await this.page.locator('[data-testid="sheet-field-key"]').count()) > 0) {
                    await this.page.locator('[data-testid="sheet-field-key"]').click();
                }
                await this.page.locator(`[data-value="${key}"]`).click();

                if ((await this.page.locator('[data-testid="sheet-field-operator-0"]').count()) > 0) {
                    await this.page.locator('[data-testid="sheet-field-operator-0"]').click();
                } else if ((await this.page.locator('[data-testid="sheet-field-operator"]').count()) > 0) {
                    await this.page.locator('[data-testid="sheet-field-operator"]').click();
                }
                await this.page.locator(`[data-value="${operator}"]`).click();

                if ((await this.page.locator('[data-testid="sheet-field-value-0"]').count()) > 0) {
                    await this.page.locator('[data-testid="sheet-field-value-0"]').clear();
                    await this.page.locator('[data-testid="sheet-field-value-0"]').fill(value);
                } else if ((await this.page.locator('[data-testid="sheet-field-value"]').count()) > 0) {
                    await this.page.locator('[data-testid="sheet-field-value"]').clear();
                    await this.page.locator('[data-testid="sheet-field-value"]').fill(value);
                }

                // In sheet mode, add button is inside the dialog
                await this.page.waitForTimeout(100);
                await this.page.locator('[role="dialog"] [data-testid="add-conditions-button"]').click();
            } else {
                // Popover mode - uses non-indexed test IDs
                await this.page.locator('[data-testid="field-key"]').first().click();
                await this.page.locator(`[data-value="${key}"]`).click();

                await this.page.locator('[data-testid="field-operator"]').first().click();
                await this.page.locator(`[data-value="${operator}"]`).click();

                await this.page.locator('[data-testid="field-value"]').first().clear();
                await this.page.locator('[data-testid="field-value"]').first().fill(value);

                // In popover mode, try multiple selectors for add button
                await this.page.waitForTimeout(100);
                if ((await this.page.locator('[data-testid="add-condition-button"]').count()) > 0) {
                    await this.page.locator('[data-testid="add-condition-button"]').click();
                } else {
                    // Fallback to finding button by text
                    await this.page.getByRole("button", { name: "Add" }).click();
                }
            }

            // Wait for the condition to be added
            await this.page.waitForTimeout(200);
        }

        // Close the dialog/popover
        if (isSheetMode) {
            // Sheet mode - the sheet doesn't auto-close after adding, so we need to close it
            // First check if there's a close button or use Escape
            if ((await this.page.locator('[role="dialog"] button[aria-label="Close"]').count()) > 0) {
                await this.page.locator('[role="dialog"] button[aria-label="Close"]').click();
            } else {
                // Fall back to Escape key
                await this.page.keyboard.press("Escape");
            }
            // Wait a moment for close animation to start
            await this.page.waitForTimeout(100);
            // Wait for the sheet to fully close by checking that the dialog is gone
            await this.page.locator('[role="dialog"]').waitFor({ state: "hidden", timeout: 5000 });
            // Also ensure body no longer has scroll lock
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: 5000 });
            // Additional wait to ensure DOM updates are complete
            await this.page.waitForTimeout(300);
        } else {
            // Popover mode - might have a cancel button or close on click outside
            if ((await this.page.locator('[data-testid="cancel-button"]').count()) > 0) {
                await this.page.locator('[data-testid="cancel-button"]').click();
            } else {
                // Click outside to close popover
                await this.page.mouse.click(0, 0);
            }
        }
        await this.page.waitForTimeout(500);
    }

    /**
     * Helper command to check if we're in sheet mode or popover mode
     * @returns {Promise<string>} 'sheet' or 'popover'
     */
    async getWhereConditionMode() {
        const hasSheetFields = (await this.page.locator('[data-testid*="sheet-field"]').count()) > 0;
        const hasPopoverBadges = (await this.page.locator('[data-testid="where-condition-badge"]').count()) > 0;
        const hasFieldKey = (await this.page.locator('[data-testid="field-key"]').count()) > 0;

        if (hasSheetFields || (!hasPopoverBadges && !hasFieldKey)) {
            return "sheet";
        }
        return "popover";
    }

    /**
     * Helper to get condition count
     * @returns {Promise<number>}
     */
    async getConditionCount() {
        // Use data-condition-count attribute for reliable count
        const whereContainer = this.page.locator("[data-condition-count]");
        const count = await whereContainer.count();
        if (count > 0) {
            const attr = await whereContainer.getAttribute("data-condition-count");
            if (attr !== null) return parseInt(attr);
        }
        return 0;
    }

    /**
     * Helper to get all conditions as array of {key, operator, value} objects
     * @returns {Promise<Array<{key: string, operator: string, value: string}>>}
     */
    async getConditions() {
        return await this.page.evaluate(() => {
            const conditions = [];
            const conditionEls = document.querySelectorAll('[data-testid="where-condition"]');
            conditionEls.forEach((el) => {
                const key = el.getAttribute("data-condition-key");
                const operator = el.getAttribute("data-condition-operator");
                const value = el.getAttribute("data-condition-value");
                if (key && operator && value) {
                    conditions.push({ key, operator, value });
                }
            });
            return conditions;
        });
    }

    /**
     * Helper to verify condition by index using data attributes
     * @param {number} index
     * @param {string} expectedKey
     * @param {string} [expectedOperator]
     * @param {string} [expectedValue]
     */
    async verifyCondition(index, expectedKey, expectedOperator, expectedValue) {
        // If called with just 2 args (index, expectedText), fall back to text matching
        if (expectedOperator === undefined && expectedValue === undefined) {
            const expectedText = expectedKey;
            const mode = await this.getWhereConditionMode();
            if (mode === "popover") {
                await expect(this.page.locator('[data-testid="where-condition-badge"]').nth(index)).toContainText(expectedText);
            } else {
                console.log(`Sheet mode: Skipping condition text verification for "${expectedText}"`);
            }
            return;
        }

        // Use data attributes for reliable verification
        const el = this.page.locator('[data-testid="where-condition"]').nth(index);
        expect(await el.getAttribute("data-condition-key")).toBe(expectedKey);
        expect(await el.getAttribute("data-condition-operator")).toBe(expectedOperator);
        expect(await el.getAttribute("data-condition-value")).toBe(expectedValue);
    }

    /**
     * Helper to click on a condition to edit it
     * @param {number} index
     */
    async clickConditionToEdit(index) {
        const mode = await this.getWhereConditionMode();
        if (mode === "popover") {
            await this.page.locator('[data-testid="where-condition-badge"]').nth(index).click();
        } else {
            console.log("Sheet mode: Cannot click individual conditions - need to open sheet");
        }
    }

    /**
     * Helper to remove a specific condition
     * @param {number} index
     */
    async removeCondition(index) {
        const mode = await this.getWhereConditionMode();
        if (mode === "popover") {
            await this.page.locator('[data-testid="remove-where-condition-button"]').nth(index).click();
        } else {
            console.log("Sheet mode: Need to open sheet to remove specific conditions");
        }
    }

    /**
     * Helper to update field value in edit mode
     * @param {string} value
     */
    async updateConditionValue(value) {
        const mode = await this.getWhereConditionMode();
        if (mode === "popover") {
            await this.page.locator('[data-testid="field-value"]').clear();
            await this.page.locator('[data-testid="field-value"]').fill(value);
        } else {
            await this.page.locator('[data-testid="sheet-field-value-0"]').clear();
            await this.page.locator('[data-testid="sheet-field-value-0"]').fill(value);
        }
    }

    /**
     * Helper to check for more conditions button
     * @param {string} expectedText
     */
    async checkMoreConditionsButton(expectedText) {
        const mode = await this.getWhereConditionMode();
        if (mode === "popover") {
            await expect(this.page.locator('[data-testid="more-conditions-button"]')).toBeVisible();
            await expect(this.page.locator('[data-testid="more-conditions-button"]')).toContainText(expectedText);
        } else {
            console.log("Sheet mode: No more-conditions button - all managed in sheet");
        }
    }

    /**
     * Helper to click more conditions button
     */
    async clickMoreConditions() {
        const mode = await this.getWhereConditionMode();
        if (mode === "popover") {
            // First check if more conditions button exists
            const moreButtonCount = await this.page.locator('[data-testid="more-conditions-button"]').count();
            if (moreButtonCount > 0) {
                // Click the more conditions button
                await this.page.locator('[data-testid="more-conditions-button"]').click();

                // Wait to see if it opens a sheet or expands in place
                await this.page.waitForTimeout(500);

                // Check if a dialog opened
                const hasDialog = (await this.page.locator('[role="dialog"]').count()) > 0;
                if (!hasDialog) {
                    // No dialog means it expanded in place, we're done
                    console.log("Expanded conditions in place");
                } else {
                    console.log("Opened conditions sheet");
                }
            } else {
                console.log("No more conditions button found");
            }
        } else {
            // In sheet mode, open the where button sheet
            await this.page.locator('[data-testid="where-button"]').click();
        }
    }

    /**
     * Helper to save changes in a sheet
     */
    async saveSheetChanges() {
        // Click save button - matches various button texts used in different sheets
        await this.page.locator('[role="dialog"]').locator("button", { hasText: /^(Add|Update|Add to Page|Add Condition|Save Changes)$/ }).click();
    }

    /**
     * Helper to remove conditions in sheet
     * @param {boolean} keepFirst
     */
    async removeConditionsInSheet(keepFirst = true) {
        // Try both possible selectors
        let removeButtons = this.page.locator('[data-testid^="delete-existing-filter-"]');
        let selectorPrefix = "delete-existing-filter-";

        let count = await removeButtons.count();
        if (count === 0) {
            removeButtons = this.page.locator('[data-testid^="remove-sheet-filter-"]');
            selectorPrefix = "remove-sheet-filter-";
            count = await removeButtons.count();
        }

        const endIndex = keepFirst ? 1 : 0;

        for (let i = count - 1; i >= endIndex; i--) {
            await this.page.locator(`[data-testid="${selectorPrefix}${i}"]`).click();
        }
    }

    /**
     * Clear all where conditions
     */
    async clearWhereConditions() {
        // First check if we're in popover mode by looking for visible badges
        const visibleBadges = await this.page.locator('[data-testid="where-condition-badge"]').count();

        if (visibleBadges > 0) {
            // Popover mode - badges are always visible, remove them directly
            // Recursively remove badges until none are left
            while (true) {
                const remaining = await this.page.locator('[data-testid="remove-where-condition-button"]').count();
                if (remaining === 0) break;
                await this.page.locator('[data-testid="remove-where-condition-button"]').first().click({ force: true });
                await this.page.waitForTimeout(100);
            }
        } else {
            // Sheet mode - check button text to see if there are conditions
            const buttonText = await this.page.locator('[data-testid="where-button"]').innerText();
            // In sheet mode, button shows "N Condition(s)" or "10+ Conditions" when there are conditions
            // Only "Add" means no conditions
            if (buttonText.trim() === "Add") {
                return;
            }

            // Click where button to open sheet
            await this.page.locator('[data-testid="where-button"]').click();
            await this.page.waitForTimeout(500);

            // Delete all existing filters by clicking index 0 repeatedly
            while (true) {
                const remaining = await this.page.locator('[data-testid^="delete-existing-filter-"]').count();
                if (remaining === 0) break;
                await this.page.locator('[data-testid="delete-existing-filter-0"]').click();
                await this.page.waitForTimeout(100);
            }

            // Close sheet
            await this.page.keyboard.press("Escape");
            // Wait a moment for close animation to start
            await this.page.waitForTimeout(100);
            // Wait for the sheet to fully close
            await this.page.locator('[role="dialog"]').waitFor({ state: "hidden", timeout: 5000 });
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: 5000 });
            await this.page.waitForTimeout(300);
        }
    }

    /**
     * Set the where condition mode via localStorage and reload to ensure consistency
     * @param {string} mode
     */
    async setWhereConditionMode(mode) {
        await this.page.evaluate((m) => {
            const settingsKey = "persist:settings";
            try {
                const currentSettings = JSON.parse(localStorage.getItem(settingsKey) || "{}");
                currentSettings.whereConditionMode = `"${m}"`; // The value needs to be a JSON string
                localStorage.setItem(settingsKey, JSON.stringify(currentSettings));
            } catch (e) {
                localStorage.setItem(
                    settingsKey,
                    JSON.stringify({ whereConditionMode: `"${m}"`, _persist: '{"version":-1,"rehydrated":true}' })
                );
            }
        }, mode);
        // Reload the page to make sure the setting is applied
        await this.page.reload();
    }

    // ============================================================================
    // Table Search & Highlighting
    // ============================================================================

    /**
     * Get highlighted table cells
     * @param {Object} options - Locator options (e.g. timeout)
     * @returns {import("@playwright/test").Locator}
     */
    getHighlightedCell(options = {}) {
        return this.page.locator("td.table-search-highlight");
    }

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
    }

    // ============================================================================
    // Rows
    // ============================================================================

    /**
     * Add a row to the table
     * @param {Object|string} data - Row data as key-value pairs or JSON string for document DBs
     * @param {boolean} isSingleInput - Whether this is a document database with a single input
     */
    async addRow(data, isSingleInput = false) {
        await this.page.locator('[data-testid="add-row-button"]').click();

        if (isSingleInput) {
            // Document database - single text box for JSON
            const jsonString = typeof data === "string" ? data : JSON.stringify(data, null, 2);
            const field = this.page
                .locator('[data-testid="add-row-field-document"] input, [data-testid="add-row-field-document"] textarea')
                .first();
            await field.clear();
            await field.fill(jsonString);
        } else {
            // Traditional database - multiple fields
            for (const [key, value] of Object.entries(data)) {
                await this.page.locator(`[data-testid="add-row-field-${key}"] input`).clear();
                await this.page.locator(`[data-testid="add-row-field-${key}"] input`).fill(value);
            }
        }

        await this.page.locator('[data-testid="submit-add-row-button"]').click();

        // Wait for the operation to complete - dialog should close on success
        // Use longer timeout to account for slow database operations
        await this.page.locator('[data-testid="submit-add-row-button"]').waitFor({ state: "hidden", timeout: 10000 });

        // Ensure body no longer has scroll lock
        await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: 5000 });

        // Wait for GraphQL mutation to complete and UI to re-fetch/render
        // This cannot be assertion-based because table already has rows
        await this.page.waitForTimeout(500);

        // Ensure table is visible
        await this.page.locator("table tbody").waitFor({ state: "visible" });
    }

    /**
     * Waits for a row containing a specific value in a given column to appear in the table.
     * Uses polling to retry until found or timeout.
     * @param {number} columnIndex - The column index (1-based, accounting for checkbox column)
     * @param {string} expectedValue - The value to search for
     * @param {Object} options - Optional config: { timeout: 10000 }
     * @returns {Promise<number>} - The row index where the value was found
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

        // Return the row index for use in subsequent operations
        const rows = await this.page.locator("table tbody tr").all();
        for (let i = 0; i < rows.length; i++) {
            const cell = rows[i].locator("td").nth(columnIndex);
            const cellText = (await cell.innerText()).trim();
            if (cellText === expectedStr) {
                return i;
            }
        }
        return -1;
    }

    /**
     * Waits for a row containing a specific value (anywhere in the row) to appear in the table.
     * Uses polling to retry until found or timeout.
     * @param {string} expectedValue - The value to search for (case-insensitive partial match)
     * @param {Object} options - Optional config: { timeout: 10000, caseSensitive: false }
     * @returns {Promise<number>} - The row index where the value was found
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

        // Return the row index for use in subsequent operations
        const rows = await this.page.locator("table tbody tr").all();
        for (let i = 0; i < rows.length; i++) {
            const rowText = caseSensitive ? await rows[i].innerText() : (await rows[i].innerText()).toLowerCase();
            if (rowText.includes(searchStr)) {
                return i;
            }
        }
        return -1;
    }

    /**
     * Open context menu on a table row with retry logic
     * @param {number} rowIndex
     * @param {number} maxRetries
     */
    async openContextMenu(rowIndex, maxRetries = 3) {
        for (let attempt = 1; attempt <= maxRetries; attempt++) {
            const targetRow = this.page.locator("table tbody tr").nth(rowIndex);

            // Scroll into view and wait for it to stabilize (virtualization may need time)
            await targetRow.scrollIntoViewIfNeeded();
            await this.page.waitForTimeout(200);

            // Right-click to open context menu (force: true handles visibility issues with virtualization)
            await targetRow.click({ button: "right", force: true });

            // Wait for context menu to render
            await this.page.waitForTimeout(300);

            // Check if context menu appeared
            const menuExists =
                (await this.page.locator('[data-testid="context-menu-edit-row"], [data-testid="context-menu-more-actions"]').count()) > 0;

            if (menuExists) {
                return;
            }

            if (attempt < maxRetries) {
                // Close any partial menu state by clicking elsewhere
                await this.page.mouse.click(0, 0);
                await this.page.waitForTimeout(100);
            } else {
                // Final attempt failed, let assertion handle it
                await this.page
                    .locator('[data-testid="context-menu-edit-row"], [data-testid="context-menu-more-actions"]')
                    .waitFor({ timeout: 5000 });
            }
        }
    }

    /**
     * Delete a row by index
     * @param {number} rowIndex
     */
    async deleteRow(rowIndex) {
        // Get initial row count to verify deletion
        const initialRowCount = await this.page.locator("table tbody tr").count();

        // Ensure the target row exists before interacting
        expect(await this.page.locator("table tbody tr").count()).toBeGreaterThan(rowIndex);

        // Use the helper to open context menu with retry logic
        await this.openContextMenu(rowIndex);

        // Click delete directly - no longer in a submenu
        const deleteBtn = this.page.locator('[data-testid="context-menu-delete-row"]');
        await deleteBtn.scrollIntoViewIfNeeded();
        await deleteBtn.waitFor({ timeout: 5000 });
        await deleteBtn.click({ force: true });

        // Wait for the row to be removed by checking the row count
        await expect(this.page.locator("table tbody tr")).toHaveCount(initialRowCount - 1, { timeout: 10000 });
    }

    /**
     * Update a row via context menu
     * @param {number} rowIndex
     * @param {number} columnIndex
     * @param {string} text
     * @param {boolean} cancel - If true, cancel the edit; if false, submit the update
     */
    async updateRow(rowIndex, columnIndex, text, cancel = true) {
        // Wait for table to stabilize before interacting
        await this.page.waitForTimeout(500);

        // Use the helper to open context menu with retry logic
        await this.openContextMenu(rowIndex);

        // Wait for the menu to be visible, then click the "Edit row" item
        const editBtn = this.page.locator('[data-testid="context-menu-edit-row"]');
        await editBtn.scrollIntoViewIfNeeded();
        await editBtn.waitFor({ timeout: 5000 });
        await editBtn.click({ force: true });

        // Try to find the standard editable field first
        const standardField = this.page.locator(`[data-testid="editable-field-${columnIndex}"]`);
        if ((await standardField.count()) > 0) {
            // Standard field-based editing (SQL databases)
            await standardField.clear();
            await standardField.fill(text);
        } else {
            // Document-based editing (MongoDB, Elasticsearch)
            // Look for a textarea or input that contains JSON-like content or is empty
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
                // Find the matching element and interact with it
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
                    // Fallback: use the first textarea or text input
                    await textareas.first().clear();
                    await textareas.first().fill(text);
                }
            } else {
                // Fallback: use the first textarea or text input
                await this.page.locator('textarea, input[type="text"]').first().clear();
                await this.page.locator('textarea, input[type="text"]').first().fill(text);
            }
        }

        // Click cancel (escape key) or update as requested
        if (cancel) {
            // Close the sheet by pressing Escape
            await this.page.keyboard.press("Escape");
            // Wait for the sheet to disappear
            await this.page.getByText("Edit Row").waitFor({ state: "hidden" });
            // Ensure body no longer has scroll lock
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: 5000 });
        } else {
            await this.page.locator('[data-testid="update-button"]').click();
            // Wait for the update to complete by asserting the sheet is gone.
            await this.page.locator('[data-testid="update-button"]').waitFor({ state: "hidden" });
            // Ensure body no longer has scroll lock
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: 5000 });
        }
    }

    // ============================================================================
    // Pagination & Search
    // ============================================================================

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
    }

    /**
     * Search the table
     * @param {string} search
     */
    async searchTable(search) {
        // Break up the chain to avoid element detachment after clear
        await this.page.locator('[data-testid="table-search"]').clear();
        // Use pressSequentially (not fill) to trigger input events that activate search highlighting
        await this.page.locator('[data-testid="table-search"]').pressSequentially(search);
        await this.page.locator('[data-testid="table-search"]').press("Enter");
        // Search completes when the input value is stable (no loading)
        await expect(this.page.locator('[data-testid="table-search"]')).toHaveValue(search);
    }

    // ============================================================================
    // Graph
    // ============================================================================

    /**
     * Get graph nodes and edges
     * @returns {Promise<Object>} Graph as { nodeName: [connectedNodeNames] }
     */
    async getGraph() {
        // Wait for the graph to be fully loaded - nodes should exist and be visible
        await this.page.locator(".react-flow__node").first().waitFor({ state: "visible", timeout: 10000 });

        // Add a small wait to ensure layout has completed (React Flow layout takes time)
        await this.page.waitForTimeout(400); // Slightly more than the 300ms layout timeout

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
                // Deduplicate targets (multiple FK columns can create duplicate edges)
                graph[node] = [...new Set(targets)];
            });
            return graph;
        });
    }

    /**
     * Get graph node data as array of [key, value] pairs
     * @param {string} nodeId
     * @returns {Promise<Array<[string, string]>>}
     */
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
    }

    // ============================================================================
    // Scratchpad / Code Cells
    // ============================================================================

    /**
     * Add a cell after the specified index
     * @param {number} afterIndex
     */
    async addCell(afterIndex) {
        await this.page
            .locator(`[role="tabpanel"][data-state="active"] [data-testid="cell-${afterIndex}"] [data-testid="add-cell-button"]`)
            .click();
    }

    /**
     * Remove a cell at the specified index
     * @param {number} index
     */
    async removeCell(index) {
        await this.page
            .locator(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="delete-cell-button"]`)
            .click();
    }

    /**
     * Write code into a cell's CodeMirror editor
     * @param {number} index
     * @param {string} text
     */
    async writeCode(index, text) {
        const selector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] .cm-content`;

        // Click to focus, clear, and then type.
        const editor = this.page.locator(selector);
        await editor.scrollIntoViewIfNeeded();
        await editor.waitFor({ state: "visible" });
        await editor.click();
        await editor.clear();
        await editor.fill(text);

        // Blur to ensure state updates
        await editor.blur();

        // Small wait to ensure React state has fully updated
        await this.page.waitForTimeout(100);
    }

    /**
     * Run code in a specific cell
     * @param {number} index
     */
    async runCode(index) {
        const buttonSelector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="query-cell-button"]`;

        // The button is only visible on hover. Forcing the click is the standard
        // and most reliable way to handle this scenario.
        // Using .first() to prevent errors when multiple buttons are found.
        await this.page.locator(buttonSelector).first().click({ force: true });

        // Just wait for the query results to appear
        // Don't check for loading spinner since queries execute very quickly on localhost
        const cellLocator = this.page.locator(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"]`);
        await cellLocator
            .locator('[data-testid="cell-query-output"], [data-testid="cell-action-output"], [data-testid="cell-error"]')
            .first()
            .waitFor({ timeout: 5000 });
    }

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
    }

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
    }

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
        // Remove "Error" prefix from AlertTitle
        return text.replace(/^Error\s*/i, "").trim();
    }

    // ============================================================================
    // Logout
    // ============================================================================

    /**
     * Logout from the application
     */
    async logout() {
        // Check if we're on a page with sidebar (i.e., logged in)
        const hasSidebar =
            (await this.page.locator('[data-sidebar="sidebar"]').count()) > 0 ||
            (await this.page.locator('[data-sidebar="trigger"]').count()) > 0;

        if (!hasSidebar) {
            // Not logged in or on login page - nothing to logout from
            console.log("No sidebar found - skipping logout (may not be logged in)");
            return;
        }

        // Check if the sidebar trigger button exists and is visible (indicates sidebar is closed)
        const sidebarTriggerVisible = await this.page.locator('[data-sidebar="trigger"]').filter({ visible: true }).count();
        const bodyText = await this.page.locator("body").innerText();
        if (sidebarTriggerVisible > 0 && !bodyText.includes("Logout Profile")) {
            // Sidebar is closed, open it first (use force: true in case of overlays)
            await this.page.locator('[data-sidebar="trigger"]').first().click({ force: true });
            await this.page.waitForTimeout(300); // Wait for sidebar animation
        }

        // Now the sidebar should be open, click logout
        const updatedBodyText = await this.page.locator("body").innerText();
        if (updatedBodyText.includes("Logout Profile")) {
            // Sidebar is expanded, click on the text
            await this.page.getByText("Logout Profile").click({ force: true });
        } else {
            // Fallback: try to find the logout button in the sidebar
            await this.page
                .locator('[data-sidebar="sidebar"]')
                .first()
                .locator('li[data-sidebar="menu-item"]')
                .last()
                .locator("div.cursor-pointer")
                .first()
                .click({ force: true });
        }
    }

    // ============================================================================
    // Tables List
    // ============================================================================

    /**
     * Get all table names from the storage unit page
     * @returns {Promise<string[]>}
     */
    async getTables() {
        // Ensure card view is set for consistent test behavior
        await this.page.evaluate(() => {
            const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
            settings.storageUnitView = '"card"';
            localStorage.setItem("persist:settings", JSON.stringify(settings));
        });

        await this.page.goto(this.url("/storage-unit"));
        // Wait for cards to load
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ timeout: 15000 });

        const elements = this.page.locator('[data-testid="storage-unit-name"]');
        const count = await elements.count();
        const result = [];
        for (let i = 0; i < count; i++) {
            result.push(await elements.nth(i).innerText());
        }
        return result;
    }

    // ============================================================================
    // Scratchpad Pages
    // ============================================================================

    /**
     * Add a new scratchpad page
     */
    async addScratchpadPage() {
        await this.page.locator('[data-testid="add-page-button"]').click();
    }

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
    }

    /**
     * Delete a scratchpad page
     * @param {number} index
     * @param {boolean} cancel - If true, cancel the deletion
     */
    async deleteScratchpadPage(index, cancel = true) {
        // Click the delete button on the specific page tab
        await this.page.locator(`[data-testid="delete-page-tab-${index}"]`).click();

        // Handle the confirmation dialog
        if (cancel) {
            await this.page.locator('[data-testid="delete-page-button-cancel"]').click();
        } else {
            await this.page.locator('[data-testid="delete-page-button-confirm"]').click();
        }
    }

    // ============================================================================
    // Context Menu
    // ============================================================================

    /**
     * Dismiss any visible context menu
     */
    async dismissContextMenu() {
        const contextMenus = await this.page.locator('[role="menu"]').filter({ visible: true }).count();
        if (contextMenus > 0) {
            await this.page.mouse.click(0, 0);
            await this.page.waitForTimeout(100);
        }
    }

    /**
     * Select "Mock Data" from the table header context menu
     */
    async selectMockData() {
        // Right-click the table header to open the context menu
        // Target the table header row with cursor-context-menu class
        await this.page.locator("table thead tr.cursor-context-menu").first().click({ button: "right", force: true });
        // Wait for context menu to appear
        await this.page.waitForTimeout(200);
        // Click the "Mock Data" item using scrollIntoView and force click to handle overflow issues
        const mockDataItem = this.page.locator('[data-testid="context-menu-mock-data"]');
        await mockDataItem.scrollIntoViewIfNeeded();
        await mockDataItem.waitFor({ timeout: 5000 });
        await mockDataItem.click({ force: true });
    }

    // ============================================================================
    // Export Dialog Commands
    // ============================================================================

    /**
     * Select export format from dropdown
     * @param {string} format
     */
    async selectExportFormat(format) {
        await this.page.locator('[data-testid="export-format-select"]').click();
        await this.page.locator(`[data-value="${format}"]`).click();
    }

    /**
     * Select export delimiter from dropdown
     * @param {string} delimiter
     */
    async selectExportDelimiter(delimiter) {
        await this.page.locator('[data-testid="export-delimiter-select"]').click();
        await this.page.locator(`[data-value="${delimiter}"]`).click();
    }

    /**
     * Confirm the export
     */
    async confirmExport() {
        await this.page.locator('[data-testid="export-confirm-button"]').click();
    }

    // ============================================================================
    // Mock Data Dialog Commands
    // ============================================================================

    /**
     * Set the number of mock data rows
     * @param {number} count
     */
    async setMockDataRows(count) {
        await this.page.locator('[data-testid="mock-data-rows-input"]').clear();
        await this.page.locator('[data-testid="mock-data-rows-input"]').fill(count.toString());
    }

    /**
     * Set mock data handling mode
     * @param {string} handling
     */
    async setMockDataHandling(handling) {
        await this.page.locator('[data-testid="mock-data-handling-select"]').click();
        await this.page.locator(`[data-value="${handling}"]`).click();
    }

    /**
     * Generate mock data
     */
    async generateMockData() {
        await this.page.locator('[data-testid="mock-data-generate-button"]').click();
    }

    /**
     * Confirm mock data overwrite
     */
    async confirmMockDataOverwrite() {
        await this.page.locator('[data-testid="mock-data-overwrite-button"]').click();
    }

    // ============================================================================
    // Query History Commands
    // ============================================================================

    /**
     * Open query history for a specific cell
     * @param {number} index - Cell index (default: 0)
     */
    async openQueryHistory(index = 0) {
        // Click the options menu (three dots) in the scratchpad cell.
        // Use .first() to ensure we only click one, even if the selector finds multiple.
        await this.page
            .locator(`[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="icon-button"]`)
            .first()
            .click();

        // Click on "Query History" option, scoped to the newly visible menu to be specific.
        const menu = this.page.locator('[role="menu"]');
        await menu.waitFor({ state: "visible" });
        await menu.locator('[role="menuitem"]').filter({ hasText: "Query History" }).click();

        // Wait a bit for animations
        await this.page.waitForTimeout(500);

        // Wait for the history dialog to open and verify the title is present.
        await this.page.locator('[role="dialog"], .bg-background[data-state="open"]').waitFor({ state: "visible" });
        await expect(this.page.getByText("Query History")).toBeVisible();
    }

    /**
     * Get query history items as array of query text strings
     * @returns {Promise<string[]>}
     */
    async getQueryHistoryItems() {
        await this.page.locator('[role="dialog"] [data-slot="card"]').first().waitFor({ timeout: 10000 });
        const items = this.page.locator('[role="dialog"] [data-slot="card"]');
        const count = await items.count();
        const result = [];
        for (let i = 0; i < count; i++) {
            const queryText = (await items.nth(i).locator("pre code").innerText()).trim();
            result.push(queryText);
        }
        return result;
    }

    /**
     * Copy a query from history (verifies clipboard interaction)
     * @param {number} index - History item index (default: 0)
     */
    async copyQueryFromHistory(index = 0) {
        // Grant clipboard permissions for the test
        const context = this.page.context();
        await context.grantPermissions(["clipboard-read", "clipboard-write"]);

        const card = this.page.locator('[role="dialog"] [data-slot="card"]').nth(index);
        const textToCopy = (await card.locator("pre code").innerText()).trim();

        // Click the copy button
        await card.locator('[data-testid="copy-to-clipboard-button"]').click();

        // Verify clipboard content
        const clipboardText = await this.page.evaluate(() => navigator.clipboard.readText());
        expect(clipboardText).toBe(textToCopy);
    }

    /**
     * Clone a query from history to the editor
     * @param {number} historyIndex - History item index (default: 0)
     * @param {number} targetCellIndex - Target cell index (default: 0)
     */
    async cloneQueryToEditor(historyIndex = 0, targetCellIndex = 0) {
        const card = this.page.locator('[role="dialog"] [data-slot="card"]').nth(historyIndex);
        const expectedText = (await card.locator("pre code").innerText()).trim();
        await card.locator('[data-testid="clone-to-editor-button"]').click();

        // Wait a bit for the click handler to process
        await this.page.waitForTimeout(500);

        // Wait for the sheet to close - it should no longer be visible
        const dialogCount = await this.page.locator('[role="dialog"]').filter({ visible: true }).count();
        if (dialogCount > 0) {
            await this.page.locator('[role="dialog"]').waitFor({ state: "hidden", timeout: 10000 });
        }

        // Verify the editor contains the expected text
        const editorSelector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${targetCellIndex}"] [data-testid="code-editor"] .cm-content`;
        await expect(this.page.locator(editorSelector)).toContainText(expectedText, { timeout: 10000 });

        // Add a small wait to ensure React state has updated after the code change
        await this.page.waitForTimeout(500);
    }

    /**
     * Execute a query from history
     * @param {number} index - History item index (default: 0)
     */
    async executeQueryFromHistory(index = 0) {
        // Click the run button for the specified history item
        await this.page
            .locator('[role="dialog"] [data-slot="card"]')
            .nth(index)
            .locator('[data-testid="run-history-button"]')
            .click();

        // The dialog stays open after executing - this is the expected behavior
        // No need to wait since queries execute quickly on localhost
    }

    /**
     * Close the query history dialog
     */
    async closeQueryHistory() {
        // Close the query history dialog
        await this.page.keyboard.press("Escape");

        // Wait for the dialog to fully close
        const dialogCount = await this.page.locator('[role="dialog"]').count();
        if (dialogCount > 0) {
            await this.page.locator('[role="dialog"]').waitFor({ state: "hidden" });
        }

        // Ensure body no longer has scroll lock
        await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: 5000 });

        // Additional wait for animation completion
        await this.page.waitForTimeout(300);
    }

    /**
     * Verify that the editor in the specified cell contains the expected query text
     * @param {number} index
     * @param {string} expectedQuery
     */
    async verifyQueryInEditor(index, expectedQuery) {
        const selector = `[role="tabpanel"][data-state="active"] [data-testid="cell-${index}"] [data-testid="code-editor"] .cm-content`;
        await expect(this.page.locator(selector)).toContainText(expectedQuery);
    }

    /**
     * Enables the SQL autocomplete for the duration of the test
     */
    async enableAutocomplete() {
        await this.page.evaluate(() => {
            delete window.__E2E_DISABLE_AUTOCOMPLETE;
        });
    }

    /**
     * Disables the SQL autocomplete for the duration of the test
     */
    async disableAutocomplete() {
        await this.page.evaluate(() => {
            window.__E2E_DISABLE_AUTOCOMPLETE = true;
        });
    }

    // ============================================================================
    // Chat Commands
    // ============================================================================

    /**
     * Sets up a mock for the Version query to return a consistent version string
     * @param {string} version - The version string to return (default: 'v1.1.1')
     */
    async mockVersion(version = "v1.1.1") {
        await this.page.route("**/api/query", async (route) => {
            const request = route.request();
            const postData = request.postDataJSON();
            if (postData?.operationName === "GetVersion") {
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        data: {
                            Version: version,
                        },
                    }),
                });
            } else {
                await route.fallback();
            }
        });
    }

    /**
     * Sets up a mock AI provider for chat testing
     * This creates a comprehensive route handler that handles all GraphQL operations for chat
     * @param {Object} options - Configuration options
     * @param {string} options.modelType - The model type (e.g., 'Ollama', 'OpenAI')
     * @param {string} options.model - The specific model name (e.g., 'llama3.1')
     * @param {string} options.providerId - Optional provider ID
     */
    async setupChatMock({ modelType = "Ollama", model = "llama3.1", providerId = "test-provider" } = {}) {
        // Reset chat response
        this._chatMockResponses = null;

        // Route handler for GraphQL operations for provider and model info
        await this.page.route("**/api/query", async (route) => {
            const request = route.request();
            const postData = request.postDataJSON();
            const operation = postData?.operationName;
            console.log("[PLAYWRIGHT] Intercepted GraphQL operation:", operation);

            if (operation === "GetAIProviders") {
                console.log("[PLAYWRIGHT] Handling GetAIProviders");
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        data: {
                            AIProviders: [
                                {
                                    Type: modelType,
                                    Name: modelType,
                                    ProviderId: providerId,
                                    IsEnvironmentDefined: false,
                                },
                            ],
                        },
                    }),
                });
                return;
            }

            if (operation === "GetAIModels") {
                console.log("[PLAYWRIGHT] Handling GetAIModels, returning:", [model]);
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        data: {
                            AIModel: [model],
                        },
                    }),
                });
                return;
            }

            // Let other GraphQL operations pass through
            await route.fallback();
        });

        // Route handler for the streaming chat endpoint
        await this.page.route("**/api/ai-chat/stream", async (route) => {
            console.log("[PLAYWRIGHT] Intercepted streaming chat request");

            // Use the stored response
            const responseData = this._chatMockResponses || [];
            console.log("[PLAYWRIGHT] storedResponse:", JSON.stringify(responseData, null, 2));

            if (responseData.length === 0) {
                console.warn("[PLAYWRIGHT] WARNING: No chat response configured! Sending empty response.");
            }

            // Build SSE response
            let sseData = "";

            for (const response of responseData) {
                const type = response.type || "text";
                const text = response.text || "";
                const result = response.result || null;

                // Send text messages as streaming chunks
                if (type === "text" || type === "message") {
                    sseData += `event: chunk\n`;
                    sseData += `data: ${JSON.stringify({ type: "text", text })}\n\n`;
                }

                // Send SQL results as complete messages
                if (type.startsWith("sql:")) {
                    sseData += `event: message\n`;
                    sseData += `data: ${JSON.stringify({ Type: type, Text: text, Result: result })}\n\n`;
                }

                // Send errors as complete messages (will appear in chat history)
                if (type === "error") {
                    sseData += `event: message\n`;
                    sseData += `data: ${JSON.stringify({ Type: "error", Text: text, Result: result })}\n\n`;
                }
            }

            // Send done event
            sseData += `event: done\n`;
            sseData += `data: {}\n\n`;

            console.log("[PLAYWRIGHT] Sending SSE response:", sseData);

            // Clear the response after using it
            this._chatMockResponses = null;

            // Reply with SSE format
            await route.fulfill({
                status: 200,
                headers: {
                    "content-type": "text/event-stream",
                    "cache-control": "no-cache",
                    connection: "keep-alive",
                },
                body: sseData,
            });

            console.log("[PLAYWRIGHT] Response sent successfully");
        });
    }

    /**
     * Mocks a chat response with specific content
     * Must be called after setupChatMock
     * @param {Array<Object>} responses - Array of chat message responses
     * Each response can have:
     * - type: 'message', 'text', 'sql:get', 'sql:insert', 'sql:update', 'sql:delete', 'error', 'sql:pie-chart', 'sql:line-chart'
     * - text: The message or SQL query text
     * - result: Optional result object with Columns and Rows for SQL queries
     */
    async mockChatResponse(responses) {
        console.log("[PLAYWRIGHT] mockChatResponse called with:", responses);
        this._chatMockResponses = responses;
        console.log("[PLAYWRIGHT] chatMockResponses now set to:", this._chatMockResponses);
    }

    /**
     * Navigates to the chat page
     * Expects setupChatMock to be called first with providerId and model values
     */
    async gotoChat() {
        await this.page.goto(this.url("/chat"));

        // Wait for the AI provider section to be loaded
        await this.page.locator('[data-testid="ai-provider"]').waitFor({ timeout: 10000 });

        // Wait a bit for initial GraphQL requests to complete
        await this.page.waitForTimeout(1000);

        // Wait for the AI provider dropdown to be visible
        await this.page.locator('[data-testid="ai-provider-select"]').waitFor({ state: "visible", timeout: 10000 });

        // Check the button text to determine if we need to select
        const buttonText = await this.page.locator('[data-testid="ai-provider-select"]').innerText();

        // If "Select Model Type" is shown, we need to click and select
        if (buttonText.includes("Select Model Type") || buttonText.trim() === "") {
            console.log("Selecting AI provider from dropdown");

            // Click the provider dropdown to open it
            await this.page.locator('[data-testid="ai-provider-select"]').click();

            // Wait for the dropdown options to appear
            await this.page.locator('[role="option"]').first().waitFor({ state: "visible", timeout: 5000 });

            // Select the first option (Ollama from our mock)
            await this.page.locator('[role="option"]').first().click();

            // Wait for models to load after provider selection
            await this.page.waitForTimeout(1500);

            // Verify provider was selected by checking button text changed
            await expect(this.page.locator('[data-testid="ai-provider-select"]')).not.toContainText("Select Model Type", {
                timeout: 5000,
            });
        } else {
            console.log("AI provider already selected: " + buttonText);
        }

        // Wait for the model dropdown to be visible and enabled
        await this.page.locator('[data-testid="ai-model-select"]').waitFor({ state: "visible", timeout: 10000 });
        await expect(this.page.locator('[data-testid="ai-model-select"]')).toBeEnabled();

        // Check if model needs to be selected
        const modelButtonText = await this.page.locator('[data-testid="ai-model-select"]').innerText();

        // If "Select Model" is shown, we need to click and select
        if (modelButtonText.includes("Select Model") || modelButtonText.trim() === "") {
            console.log("Selecting AI model from dropdown");

            // Click the model dropdown to open it
            await this.page.locator('[data-testid="ai-model-select"]').click();

            // Wait for model options to appear
            await this.page.locator('[role="option"]').first().waitFor({ state: "visible", timeout: 5000 });

            // Select the first model option (llama3.1 from our mock)
            await this.page.locator('[role="option"]').first().click();

            // Wait for selection to complete and state to update
            await this.page.waitForTimeout(1000);

            // Verify model was selected by checking button text changed
            await expect(this.page.locator('[data-testid="ai-model-select"]')).not.toContainText("Select Model", {
                timeout: 5000,
            });
        } else {
            console.log("AI model already selected: " + modelButtonText);
        }

        // Ensure chat input is enabled and ready
        await this.page.locator('[data-testid="chat-input"]').waitFor({ state: "visible", timeout: 10000 });
        await expect(this.page.locator('[data-testid="chat-input"]')).toBeEnabled();

        // Verify input is empty (no autofill or stale state)
        await expect(this.page.locator('[data-testid="chat-input"]')).toHaveValue("");

        // Additional wait to ensure Redux state has fully propagated
        await this.page.waitForTimeout(1000);
    }

    /**
     * Sends a chat message
     * @param {string} message - The message to send
     */
    async sendChatMessage(message) {
        // Get the chat input and ensure it's ready
        await this.page.locator('[data-testid="chat-input"]').waitFor({ state: "visible" });
        await expect(this.page.locator('[data-testid="chat-input"]')).toBeEnabled();

        // Clear the input - use force:true to handle any autofill issues
        await this.page.locator('[data-testid="chat-input"]').clear({ force: true });

        // Verify it's actually cleared
        await expect(this.page.locator('[data-testid="chat-input"]')).toHaveValue("");

        // Type the message
        await this.page.locator('[data-testid="chat-input"]').fill(message);

        // Wait a moment for React state to update after typing
        await this.page.waitForTimeout(300);

        // Verify the send button is enabled before clicking
        // The button becomes enabled when query.trim().length > 0 and model is selected
        const sendBtn = this.page.locator('[data-testid="icon-button"]').last();
        await sendBtn.waitFor({ state: "visible" });
        await expect(sendBtn).toBeEnabled({ timeout: 5000 });
        await sendBtn.click();

        // Small wait for the request to be initiated
        await this.page.waitForTimeout(200);
    }

    /**
     * Verifies a user chat message appears in the conversation
     * @param {string} expectedMessage - The expected message text
     */
    async verifyChatUserMessage(expectedMessage) {
        await expect(this.page.locator('[data-input-message="user"]').last()).toContainText(expectedMessage);
    }

    /**
     * Verifies a system chat message appears in the conversation
     * @param {string} expectedMessage - The expected message text
     */
    async verifyChatSystemMessage(expectedMessage) {
        await expect(this.page.locator('[data-input-message="system"]').last()).toContainText(expectedMessage);
    }

    /**
     * Verifies a SQL query result is displayed in the chat
     * @param {Object} options - Verification options
     * @param {Array<string>} options.columns - Expected column names
     * @param {number} options.rowCount - Expected number of rows (optional)
     */
    async verifyChatSQLResult({ columns, rowCount }) {
        // Wait for the table to appear
        const table = this.page.locator("table").last();
        await table.waitFor({ state: "visible", timeout: 10000 });

        // Verify columns
        if (columns) {
            for (const column of columns) {
                await expect(table.locator("thead th")).toContainText([column]);
            }
        }

        // Verify row count if specified
        if (rowCount !== undefined) {
            await expect(table.locator("tbody tr")).toHaveCount(rowCount);
        }
    }

    /**
     * Verifies an error message appears in the chat
     * @param {string} errorText - Expected error text (can be partial, case-insensitive)
     */
    async verifyChatError(errorText) {
        const errorState = this.page.locator('[data-testid="error-state"]');
        await errorState.waitFor({ state: "visible", timeout: 10000 });
        const text = await errorState.innerText();
        expect(text.toLowerCase()).toContain(errorText.toLowerCase());
    }

    /**
     * Verifies an action executed message appears
     */
    async verifyChatActionExecuted() {
        await expect(this.page.getByText("Action Executed")).toBeVisible({ timeout: 10000 });
    }

    /**
     * Gets all chat messages
     * @returns {Promise<Array<{type: string, text: string}>>} Array of message objects with type and text
     */
    async getChatMessages() {
        return await this.page.evaluate(() => {
            const messages = [];
            const messageElements = document.querySelectorAll("[data-input-message]");
            messageElements.forEach((el) => {
                messages.push({
                    type: el.getAttribute("data-input-message"),
                    text: el.textContent.trim(),
                });
            });
            return messages;
        });
    }

    /**
     * Clears the chat history
     */
    async clearChat() {
        await this.page.locator('[data-testid="chat-new-chat"]').click();

        // Wait for messages to be cleared
        await this.page.locator("[data-input-message]").waitFor({ state: "hidden" });

        // Ensure the input field is not disabled and is ready for interaction
        await this.page.locator('[data-testid="chat-input"]').waitFor({ state: "visible" });
        await expect(this.page.locator('[data-testid="chat-input"]')).toBeEnabled();
        await expect(this.page.locator('[data-testid="chat-input"]')).toHaveValue(""); // Should be empty after clear

        // Additional wait to ensure React state has settled
        await this.page.waitForTimeout(300);
    }

    /**
     * Toggles between SQL and table view in chat
     */
    async toggleChatSQLView() {
        // Find the last table preview group and click its toggle button
        const group = this.page.locator(".group\\/table-preview").last();
        await group.locator('[data-testid="icon-button"]').first().click({ force: true });
        await this.page.waitForTimeout(300);
    }

    /**
     * Verifies SQL code is displayed in the chat
     * @param {string} expectedSQL - The expected SQL query (can be partial)
     */
    async verifyChatSQL(expectedSQL) {
        const actualText = await this.page.locator('[data-testid="code-editor"]').last().innerText();
        // Remove line number prefixes (e.g., "1>", "912>") and normalize whitespace
        const normalizedActual = actualText
            .replace(/^\d+>/gm, "") // Remove line number prefixes
            .replace(/\s+/g, " ") // Normalize whitespace
            .trim();
        const normalizedExpected = expectedSQL
            .replace(/\s+/g, " ") // Normalize whitespace
            .trim();
        expect(normalizedActual).toContain(normalizedExpected);
    }

    /**
     * Opens the move to scratchpad dialog from the last chat result
     */
    async openMoveToScratchpad() {
        const group = this.page.locator(".group\\/table-preview").last();
        // The button uses aria-label instead of title for accessibility
        await group.locator('[aria-label="Move to Scratchpad"]').click({ force: true });
        await expect(this.page.locator("h2").filter({ hasText: "Move to Scratchpad" })).toBeVisible({ timeout: 5000 });
    }

    /**
     * Confirms moving a query to scratchpad
     * @param {Object} options - Options for moving to scratchpad
     * @param {string} options.pageOption - 'new' or page ID
     * @param {string} options.newPageName - Name for new page (if pageOption is 'new')
     */
    async confirmMoveToScratchpad({ pageOption = "new", newPageName = "" } = {}) {
        if (pageOption !== "new") {
            // Select existing page
            await this.page.locator('[role="dialog"] [role="combobox"]').click();
            await this.page.locator(`[role="listbox"] [value="${pageOption}"]`).click();
        } else if (newPageName) {
            // Enter new page name (placeholder has capital letters: "Enter Page Name")
            await this.page.locator('[role="dialog"] input[placeholder="Enter Page Name"]').clear();
            await this.page.locator('[role="dialog"] input[placeholder="Enter Page Name"]').fill(newPageName);
        }

        // Click the Move to Scratchpad button
        await this.page.locator('[role="dialog"]').getByRole("button", { name: "Move to Scratchpad" }).click();

        // Wait for navigation and verify we're on scratchpad
        await this.page.waitForURL(/\/scratchpad/, { timeout: 10000 });
    }

    /**
     * Navigates chat history using arrow keys
     * @param {string} direction - 'up' or 'down'
     */
    async navigateChatHistory(direction = "up") {
        const key = direction === "up" ? "ArrowUp" : "ArrowDown";
        await this.page.locator('[data-testid="chat-input"]').focus();
        await this.page.keyboard.press(key);
        await this.page.waitForTimeout(200);
    }

    /**
     * Gets the current value in the chat input
     * @returns {Promise<string>} The current input value
     */
    async getChatInputValue() {
        return await this.page.locator('[data-testid="chat-input"]').inputValue();
    }

    /**
     * Verifies the chat is empty (no messages)
     */
    async verifyChatEmpty() {
        await this.page.locator("[data-input-message]").waitFor({ state: "hidden" });
    }

    /**
     * Waits for chat response to complete
     */
    async waitForChatResponse() {
        // First, wait for the user message to appear (confirming send worked)
        await this.page.locator('[data-input-message="user"]').first().waitFor({ timeout: 5000 });

        // Wait for loading indicator to disappear if it appears
        const loadingCount = await this.page.locator('[data-testid="loading"]').count();
        if (loadingCount > 0) {
            await this.page.locator('[data-testid="loading"]').waitFor({ state: "hidden", timeout: 10000 });
        }

        // Wait for either a system response, SQL result table, or error state to appear
        await expect(async () => {
            const hasSystemMessage = (await this.page.locator('[data-input-message="system"]').count()) > 0;
            const hasErrorState = (await this.page.locator('[data-testid="error-state"]').count()) > 0;
            const hasSQLResult =
                (await this.page.locator('[data-testid="chat-sql-result"] table, [data-testid="sql-result-table"]').count()) > 0;
            const hasAnyResultTable = (await this.page.locator("table").count()) > 0;
            expect(hasSystemMessage || hasErrorState || hasSQLResult || hasAnyResultTable).toBe(true);
        }).toPass({ timeout: 10000 });

        // Additional wait for UI to fully render the response
        await this.page.waitForTimeout(500);
    }

    // ============================================================================
    // Screenshot Highlighting Utilities
    // ============================================================================

    /**
     * Highlights an element with a rounded border overlay for screenshots
     * @param {string} selector - The CSS selector or test-id of the element to highlight
     * @param {Object} options - Styling options for the highlight
     * @param {string} options.borderColor - Border color (default: '#ca6f1e')
     * @param {string} options.borderWidth - Border width (default: '2px')
     * @param {string} options.borderRadius - Border radius (default: '8px')
     * @param {string} options.padding - Extra padding around the element (default: '4px')
     * @param {boolean} options.shadow - Whether to add a shadow (default: true)
     */
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

                overlay.setAttribute("data-testid", "cypress-highlight-overlay");
                document.body.appendChild(overlay);
            },
            { sel: selector, borderColor, borderWidth, borderRadius, padding, shadow }
        );
    }

    /**
     * Removes all highlight overlays from the page
     */
    async removeHighlights() {
        await this.page.evaluate(() => {
            const overlays = document.querySelectorAll('[data-testid="cypress-highlight-overlay"]');
            overlays.forEach((overlay) => overlay.remove());
        });
    }

    /**
     * Highlights an element and takes a screenshot, then removes the highlight
     * @param {string} selector - The CSS selector or test-id of the element to highlight
     * @param {string} screenshotName - Name for the screenshot file
     * @param {Object} highlightOptions - Options for the highlight (see highlightElement)
     * @param {Object} screenshotOptions - Options for the screenshot (Playwright screenshot options)
     */
    async screenshotWithHighlight(selector, screenshotName, highlightOptions = {}, screenshotOptions = {}) {
        await this.highlightElement(selector, highlightOptions);
        await this.page.waitForTimeout(300);
        await this.page.screenshot({ path: `${screenshotName}.png`, ...screenshotOptions });
        await this.removeHighlights();
    }
}
