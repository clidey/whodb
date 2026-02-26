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

/** Methods for where-condition management */
export const whereMethods = {
    /**
     * Add where conditions to the table
     * @param {Array<[string, string, string]>} fieldArray
     */
    async whereTable(fieldArray) {
        await this.page.locator('[data-testid="where-button"]').click();

        await Promise.race([
            this.page.locator('[data-testid="field-key"]').waitFor({ timeout: TIMEOUT.ELEMENT }).catch(() => {}),
            this.page.locator('[data-testid*="sheet-field"]').first().waitFor({ timeout: TIMEOUT.ELEMENT }).catch(() => {}),
        ]);

        const isSheetMode =
            (await this.page.locator('[role="dialog"]').count()) > 0 &&
            (await this.page.locator('[data-testid*="sheet-field"]').count()) > 0;
        const isPopoverMode = (await this.page.locator('[data-testid="field-key"]').count()) > 0;

        console.log(`Where condition mode detected: ${isSheetMode ? "sheet" : isPopoverMode ? "popover" : "unknown"}`);

        for (const [key, operator, value] of fieldArray) {
            console.log(`Adding condition: ${key} ${operator} ${value}`);

            if (isSheetMode) {
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

                await this.page.waitForTimeout(100);
                await this.page.locator('[role="dialog"] [data-testid="add-conditions-button"]').click();
            } else {
                await this.page.locator('[data-testid="field-key"]').first().click();
                await this.page.locator(`[data-value="${key}"]`).click();

                await this.page.locator('[data-testid="field-operator"]').first().click();
                await this.page.locator(`[data-value="${operator}"]`).click();

                await this.page.locator('[data-testid="field-value"]').first().fill(value);

                await this.page.waitForTimeout(100);
                if ((await this.page.locator('[data-testid="add-condition-button"]').count()) > 0) {
                    await this.page.locator('[data-testid="add-condition-button"]').click();
                } else {
                    await this.page.getByRole("button", { name: "Add" }).click();
                }
            }

            await this.page.waitForTimeout(200);
        }

        if (isSheetMode) {
            if ((await this.page.locator('[role="dialog"] button[aria-label="Close"]').count()) > 0) {
                await this.page.locator('[role="dialog"] button[aria-label="Close"]').click();
            } else {
                await this.page.keyboard.press("Escape");
            }
            await this.page.waitForTimeout(100);
            await this.page.locator('[role="dialog"]').waitFor({ state: "hidden", timeout: TIMEOUT.ELEMENT });
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.ELEMENT });
            await this.page.waitForTimeout(300);
        } else {
            if ((await this.page.locator('[data-testid="cancel-button"]').count()) > 0) {
                await this.page.locator('[data-testid="cancel-button"]').click();
            } else {
                await this.page.mouse.click(0, 0);
            }
        }
        await this.page.waitForTimeout(500);
    },

    /**
     * Check if we're in sheet mode or popover mode
     * @returns {Promise<string>}
     */
    async getWhereConditionMode() {
        const hasSheetFields = (await this.page.locator('[data-testid*="sheet-field"]').count()) > 0;
        const hasPopoverBadges = (await this.page.locator('[data-testid="where-condition-badge"]').count()) > 0;
        const hasFieldKey = (await this.page.locator('[data-testid="field-key"]').count()) > 0;

        if (hasSheetFields || (!hasPopoverBadges && !hasFieldKey)) {
            return "sheet";
        }
        return "popover";
    },

    /**
     * Get condition count
     * @returns {Promise<number>}
     */
    async getConditionCount() {
        const whereContainer = this.page.locator("[data-condition-count]");
        const count = await whereContainer.count();
        if (count > 0) {
            const attr = await whereContainer.getAttribute("data-condition-count");
            if (attr !== null) return parseInt(attr);
        }
        return 0;
    },

    /**
     * Get all conditions as array of {key, operator, value} objects
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
    },

    /**
     * Verify condition by index using data attributes
     */
    async verifyCondition(index, expectedKey, expectedOperator, expectedValue) {
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

        const el = this.page.locator('[data-testid="where-condition"]').nth(index);
        expect(await el.getAttribute("data-condition-key")).toBe(expectedKey);
        expect(await el.getAttribute("data-condition-operator")).toBe(expectedOperator);
        expect(await el.getAttribute("data-condition-value")).toBe(expectedValue);
    },

    /**
     * Click on a condition to edit it
     * @param {number} index
     */
    async clickConditionToEdit(index) {
        const mode = await this.getWhereConditionMode();
        if (mode === "popover") {
            await this.page.locator('[data-testid="where-condition-badge"]').nth(index).click();
            await this.page.locator('[data-testid="update-condition-button"]').waitFor({ timeout: TIMEOUT.ELEMENT });
        } else {
            console.log("Sheet mode: Cannot click individual conditions - need to open sheet");
        }
    },

    /**
     * Remove a specific condition
     * @param {number} index
     */
    async removeCondition(index) {
        const mode = await this.getWhereConditionMode();
        if (mode === "popover") {
            await this.page.locator('[data-testid="remove-where-condition-button"]').nth(index).click();
        } else {
            await this.page.locator('[data-testid="where-button"]').click();
            await this.page.waitForTimeout(300);

            const deleteBtn = this.page.locator(`[data-testid="delete-existing-filter-${index}"]`);
            await deleteBtn.waitFor({ state: "visible", timeout: TIMEOUT.ELEMENT });
            await deleteBtn.click();
            await this.page.waitForTimeout(200);

            await this.page.keyboard.press("Escape");
            await this.page.waitForTimeout(100);
            const dialog = this.page.locator('[role="dialog"]');
            if (await dialog.count() > 0) {
                await dialog.waitFor({ state: "hidden", timeout: TIMEOUT.ELEMENT });
            }
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.ELEMENT });
            await this.page.waitForTimeout(300);
        }
    },

    /**
     * Update field value in edit mode
     * @param {string} value
     */
    async updateConditionValue(value) {
        const mode = await this.getWhereConditionMode();
        if (mode === "popover") {
            await this.page.locator('[data-testid="field-value"]').fill(value);
        } else {
            await this.page.locator('[data-testid="sheet-field-value-0"]').fill(value);
        }
    },

    /**
     * Check for more conditions button
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
    },

    /**
     * Click more conditions button
     */
    async clickMoreConditions() {
        const mode = await this.getWhereConditionMode();
        if (mode === "popover") {
            const moreButtonCount = await this.page.locator('[data-testid="more-conditions-button"]').count();
            if (moreButtonCount > 0) {
                await this.page.locator('[data-testid="more-conditions-button"]').click();
                await this.page.waitForTimeout(500);

                const hasDialog = (await this.page.locator('[role="dialog"]').count()) > 0;
                if (!hasDialog) {
                    console.log("Expanded conditions in place");
                } else {
                    console.log("Opened conditions sheet");
                }
            } else {
                console.log("No more conditions button found");
            }
        } else {
            await this.page.locator('[data-testid="where-button"]').click();
        }
    },

    /**
     * Save changes in a sheet
     */
    async saveSheetChanges() {
        await this.page.locator('[role="dialog"]').locator("button", { hasText: /^(Add|Update|Add to Page|Add Condition|Save Changes)$/ }).click();
    },

    /**
     * Remove conditions in sheet
     * @param {boolean} keepFirst
     */
    async removeConditionsInSheet(keepFirst = true) {
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
    },

    /**
     * Clear all where conditions
     */
    async clearWhereConditions() {
        const visibleBadges = await this.page.locator('[data-testid="where-condition-badge"]').count();

        if (visibleBadges > 0) {
            while (true) {
                const remaining = await this.page.locator('[data-testid="remove-where-condition-button"]').count();
                if (remaining === 0) break;
                await this.page.locator('[data-testid="remove-where-condition-button"]').first().click({ force: true });
                await this.page.waitForTimeout(100);
            }
        } else {
            const buttonText = await this.page.locator('[data-testid="where-button"]').innerText();
            if (buttonText.trim() === "Add") {
                return;
            }

            await this.page.locator('[data-testid="where-button"]').click();
            await this.page.waitForTimeout(500);

            while (true) {
                const remaining = await this.page.locator('[data-testid^="delete-existing-filter-"]').count();
                if (remaining === 0) break;
                await this.page.locator('[data-testid="delete-existing-filter-0"]').click();
                await this.page.waitForTimeout(100);
            }

            await this.page.keyboard.press("Escape");
            await this.page.waitForTimeout(100);
            await this.page.locator('[role="dialog"]').waitFor({ state: "hidden", timeout: TIMEOUT.ELEMENT });
            await expect(this.page.locator("body")).not.toHaveAttribute("data-scroll-locked", /.+/, { timeout: TIMEOUT.ELEMENT });
            await this.page.waitForTimeout(300);
        }
    },

    /**
     * Set the where condition mode via localStorage and reload
     * @param {string} mode
     */
    async setWhereConditionMode(mode) {
        await this.page.evaluate((m) => {
            const settingsKey = "persist:settings";
            try {
                const currentSettings = JSON.parse(localStorage.getItem(settingsKey) || "{}");
                currentSettings.whereConditionMode = `"${m}"`;
                localStorage.setItem(settingsKey, JSON.stringify(currentSettings));
            } catch (e) {
                localStorage.setItem(
                    settingsKey,
                    JSON.stringify({ whereConditionMode: `"${m}"`, _persist: '{"version":-1,"rehydrated":true}' })
                );
            }
        }, mode);
        await this.page.reload();
    },
};
