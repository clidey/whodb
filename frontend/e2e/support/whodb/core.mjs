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

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";

// ============================================================================
// Platform-Aware Keyboard Shortcuts
// ============================================================================

/**
 * Detect if the test is running on macOS
 */
export const isMac = process.platform === "darwin";

/**
 * Platform-aware keyboard shortcuts object
 */
export const platformKeys = {
    navMod: isMac ? "Control" : "Alt",
    cmdMod: isMac ? "Meta" : "Control",
};

function getNavShortcut(num) {
    return `${platformKeys.navMod}+${num}`;
}

function getCmdShortcut(key) {
    return `${platformKeys.cmdMod}+${key}`;
}

/** Methods assigned to WhoDB.prototype for core navigation, auth, and keyboard shortcuts */
export const coreMethods = {
    /**
     * Resolve a relative path against the base URL
     * @param {string} path
     * @returns {string}
     */
    url(path) {
        return `${BASE_URL}${path.startsWith("/") ? path : "/" + path}`;
    },

    /**
     * Type a platform-aware navigation shortcut (Alt/Ctrl + Number)
     * @param {number} num
     */
    async typeNavShortcut(num) {
        await this.page.keyboard.press(getNavShortcut(num));
    },

    /**
     * Type a platform-aware command shortcut (Cmd/Ctrl + Key)
     * @param {string} key
     * @param {Object} options
     */
    async typeCmdShortcut(key, options = {}) {
        let combo = platformKeys.cmdMod;
        if (options.shift) {
            combo += "+Shift";
        }
        combo += `+${key}`;
        await this.page.keyboard.press(combo);
    },

    /**
     * Navigate to a route
     * @param {string} route
     */
    async goto(route) {
        await this.page.goto(this.url(`/${route}`));
    },

    /**
     * Login to the application
     */
    async login(databaseType, hostname, username, password, database, advanced = {}, schema = null) {
        const currentUrl = this.page.url();
        if (currentUrl.startsWith(BASE_URL)) {
            await this.page.context().clearCookies();
            await this.page.evaluate(() => {
                localStorage.clear();
                localStorage.setItem("whodb.analytics.consent", "denied");
                sessionStorage.clear();
            });
        }
        await this.page.goto(this.url("/login"));

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

        await this.page.locator('[data-testid="sidebar-profile"]').waitFor({ timeout: 30000 });

        await this.page.evaluate(() => {
            const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
            settings.storageUnitView = '"card"';
            localStorage.setItem("persist:settings", JSON.stringify(settings));
        });
    },

    /**
     * Set advanced options (no-op placeholder)
     */
    async setAdvanced(type, value) {
        // No-op
    },

    /**
     * Select a database from sidebar dropdown
     * @param {string} value
     */
    async selectDatabase(value) {
        await this.page.locator('[data-testid="sidebar-database"]').click();
        await this.page.locator(`[data-value="${value}"]`).click();
    },

    /**
     * Select a schema from sidebar dropdown
     * @param {string} value
     */
    async selectSchema(value) {
        await this.page.locator('[data-testid="sidebar-schema"]').click();
        await this.page.locator(`[data-value="${value}"]`).click();
        await this.page.locator('[data-testid="storage-unit-card"]').first().waitFor({ state: "visible", timeout: 15000 });
    },

    /**
     * Logout from the application
     */
    async logout() {
        const hasSidebar =
            (await this.page.locator('[data-sidebar="sidebar"]').count()) > 0 ||
            (await this.page.locator('[data-sidebar="trigger"]').count()) > 0;

        if (!hasSidebar) {
            console.log("No sidebar found - skipping logout (may not be logged in)");
            return;
        }

        const sidebarTriggerVisible = await this.page.locator('[data-sidebar="trigger"]').filter({ visible: true }).count();
        const bodyText = await this.page.locator("body").innerText();
        if (sidebarTriggerVisible > 0 && !bodyText.includes("Logout Profile")) {
            await this.page.locator('[data-sidebar="trigger"]').first().click({ force: true });
            await this.page.waitForTimeout(300);
        }

        const updatedBodyText = await this.page.locator("body").innerText();
        if (updatedBodyText.includes("Logout Profile")) {
            await this.page.getByText("Logout Profile").click({ force: true });
        } else {
            await this.page
                .locator('[data-sidebar="sidebar"]')
                .first()
                .locator('li[data-sidebar="menu-item"]')
                .last()
                .locator("div.cursor-pointer")
                .first()
                .click({ force: true });
        }

        await this.page.waitForURL(/\/login/, { timeout: 10000 }).catch(() => {});
    },
};
