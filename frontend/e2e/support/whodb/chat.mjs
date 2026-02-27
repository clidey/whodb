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

/** Methods for AI chat: mocking, navigation, message verification */
export const chatMethods = {
    /**
     * Sets up a mock for the Version query
     * @param {string} version
     */
    async mockVersion(version = "v1.1.1") {
        await this.page.route("**/api/query", async (route) => {
            const request = route.request();
            let postData;
            try {
                postData = request.postDataJSON();
            } catch {
                return route.fallback();
            }
            if (postData?.operationName === "GetVersion") {
                await route.fulfill({
                    contentType: "application/json",
                    body: JSON.stringify({
                        data: { Version: version },
                    }),
                });
            } else {
                await route.fallback();
            }
        });
    },

    /**
     * Sets up a mock AI provider for chat testing
     */
    async setupChatMock({ modelType = "Ollama", model = "llama3.1", providerId = "test-provider" } = {}) {
        this._chatMockResponses = null;

        await this.page.route("**/api/query", async (route) => {
            const request = route.request();
            let postData;
            try {
                postData = request.postDataJSON();
            } catch {
                return route.fallback();
            }
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
                        data: { AIModel: [model] },
                    }),
                });
                return;
            }

            await route.fallback();
        });

        await this.page.route("**/api/ai-chat/stream", async (route) => {
            console.log("[PLAYWRIGHT] Intercepted streaming chat request");

            const responseData = this._chatMockResponses || [];
            console.log("[PLAYWRIGHT] storedResponse:", JSON.stringify(responseData, null, 2));

            if (responseData.length === 0) {
                console.warn("[PLAYWRIGHT] WARNING: No chat response configured! Sending empty response.");
            }

            let sseData = "";

            for (const response of responseData) {
                const type = response.type || "text";
                const text = response.text || "";
                const result = response.result || null;

                if (type === "text" || type === "message") {
                    sseData += `event: chunk\n`;
                    sseData += `data: ${JSON.stringify({ type: "text", text })}\n\n`;
                }

                if (type.startsWith("sql:")) {
                    sseData += `event: message\n`;
                    sseData += `data: ${JSON.stringify({ Type: type, Text: text, Result: result })}\n\n`;
                }

                if (type === "error") {
                    sseData += `event: message\n`;
                    sseData += `data: ${JSON.stringify({ Type: "error", Text: text, Result: result })}\n\n`;
                }
            }

            sseData += `event: done\n`;
            sseData += `data: {}\n\n`;

            console.log("[PLAYWRIGHT] Sending SSE response:", sseData);

            this._chatMockResponses = null;

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
    },

    /**
     * Mocks a chat response with specific content
     * @param {Array<Object>} responses
     */
    async mockChatResponse(responses) {
        console.log("[PLAYWRIGHT] mockChatResponse called with:", responses);
        this._chatMockResponses = responses;
        console.log("[PLAYWRIGHT] chatMockResponses now set to:", this._chatMockResponses);
    },

    /**
     * Navigates to the chat page
     */
    async gotoChat() {
        await this.page.goto(this.url("/chat"));
        await this.page.locator('[data-testid="ai-provider"]').waitFor({ timeout: TIMEOUT.ACTION });
        // Required: provider component initialization after route load
        await this.page.waitForTimeout(1000);
        await this.page.locator('[data-testid="ai-provider-select"]').waitFor({ state: "visible", timeout: TIMEOUT.ACTION });

        const buttonText = await this.page.locator('[data-testid="ai-provider-select"]').innerText();

        if (buttonText.includes("Select Model Type") || buttonText.trim() === "") {
            console.log("Selecting AI provider from dropdown");
            await this.page.locator('[data-testid="ai-provider-select"]').click();
            await this.page.locator('[role="option"]').first().waitFor({ state: "visible", timeout: TIMEOUT.ELEMENT });
            await this.page.locator('[role="option"]').first().click();
            // Required: dropdown state propagation after provider selection
            await this.page.waitForTimeout(1500);
            await expect(this.page.locator('[data-testid="ai-provider-select"]')).not.toContainText("Select Model Type", {
                timeout: TIMEOUT.ELEMENT,
            });
        } else {
            console.log("AI provider already selected: " + buttonText);
        }

        await this.page.locator('[data-testid="ai-model-select"]').waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
        await expect(this.page.locator('[data-testid="ai-model-select"]')).toBeEnabled();

        const modelButtonText = await this.page.locator('[data-testid="ai-model-select"]').innerText();

        if (modelButtonText.includes("Select Model") || modelButtonText.trim() === "") {
            console.log("Selecting AI model from dropdown");
            await this.page.locator('[data-testid="ai-model-select"]').click();
            await this.page.locator('[role="option"]').first().waitFor({ state: "visible", timeout: TIMEOUT.ELEMENT });
            await this.page.locator('[role="option"]').first().click();
            // Required: dropdown state propagation after model selection
            await this.page.waitForTimeout(1000);
            await expect(this.page.locator('[data-testid="ai-model-select"]')).not.toContainText("Select Model", {
                timeout: TIMEOUT.ELEMENT,
            });
        } else {
            console.log("AI model already selected: " + modelButtonText);
        }

        await this.page.locator('[data-testid="chat-input"]').waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
        await expect(this.page.locator('[data-testid="chat-input"]')).toBeEnabled();
        await expect(this.page.locator('[data-testid="chat-input"]')).toHaveValue("");
    },

    /**
     * Sends a chat message
     * @param {string} message
     */
    async sendChatMessage(message) {
        await this.page.locator('[data-testid="chat-input"]').waitFor({ state: "visible" });
        await expect(this.page.locator('[data-testid="chat-input"]')).toBeEnabled();
        await this.page.locator('[data-testid="chat-input"]').clear({ force: true });
        await expect(this.page.locator('[data-testid="chat-input"]')).toHaveValue("");
        await this.page.locator('[data-testid="chat-input"]').fill(message);
        await expect(this.page.locator('[data-testid="chat-input"]')).toHaveValue(message);

        const sendBtn = this.page.locator('[data-testid="icon-button"]').last();
        await sendBtn.waitFor({ state: "visible" });
        await expect(sendBtn).toBeEnabled({ timeout: TIMEOUT.ELEMENT });
        await sendBtn.click();
    },

    async verifyChatUserMessage(expectedMessage) {
        await expect(this.page.locator('[data-input-message="user"]').last()).toContainText(expectedMessage);
    },

    async verifyChatSystemMessage(expectedMessage) {
        await expect(this.page.locator('[data-input-message="system"]').last()).toContainText(expectedMessage);
    },

    async verifyChatSQLResult({ columns, rowCount }) {
        const table = this.page.locator("table").last();
        await table.waitFor({ state: "visible", timeout: TIMEOUT.ACTION });

        if (columns) {
            for (const column of columns) {
                await expect(table.locator("thead th")).toContainText([column]);
            }
        }

        if (rowCount !== undefined) {
            await expect(table.locator("tbody tr")).toHaveCount(rowCount);
        }
    },

    async verifyChatError(errorText) {
        const errorState = this.page.locator('[data-testid="error-state"]');
        await errorState.waitFor({ state: "visible", timeout: TIMEOUT.ACTION });
        const text = await errorState.innerText();
        expect(text.toLowerCase()).toContain(errorText.toLowerCase());
    },

    async verifyChatActionExecuted() {
        await expect(this.page.getByText("Action Executed")).toBeVisible({ timeout: TIMEOUT.ACTION });
    },

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
    },

    async clearChat() {
        await this.page.locator('[data-testid="chat-new-chat"]').click();
        await this.page.locator("[data-input-message]").waitFor({ state: "hidden" });
        await this.page.locator('[data-testid="chat-input"]').waitFor({ state: "visible" });
        await expect(this.page.locator('[data-testid="chat-input"]')).toBeEnabled();
        await expect(this.page.locator('[data-testid="chat-input"]')).toHaveValue("");
    },

    async toggleChatSQLView() {
        const group = this.page.locator(".group\\/table-preview").last();
        await group.hover();
        // Required: CSS opacity transition on group-hover reveal
        await this.page.waitForTimeout(200);
        await group.locator('[data-testid="icon-button"]').first().click();
        await this.page.locator('[data-testid="toggle-view-option"]').click();
    },

    async verifyChatSQL(expectedSQL) {
        const actualText = await this.page.locator('[data-testid="code-editor"]').last().innerText();
        const normalizedActual = actualText
            .replace(/^\d+>/gm, "")
            .replace(/\s+/g, " ")
            .trim();
        const normalizedExpected = expectedSQL
            .replace(/\s+/g, " ")
            .trim();
        expect(normalizedActual).toContain(normalizedExpected);
    },

    async openMoveToScratchpad() {
        const group = this.page.locator(".group\\/table-preview").last();
        await group.hover();
        // Required: CSS opacity transition on group-hover reveal
        await this.page.waitForTimeout(200);
        await group.locator('[data-testid="icon-button"]').first().click();
        await this.page.locator('[data-testid="move-to-scratchpad-option"]').click();
        await expect(this.page.locator("h2").filter({ hasText: "Move to Scratchpad" })).toBeVisible({ timeout: TIMEOUT.ELEMENT });
    },

    async confirmMoveToScratchpad({ pageOption = "new", newPageName = "" } = {}) {
        if (pageOption !== "new") {
            await this.page.locator('[role="dialog"] [role="combobox"]').click();
            await this.page.locator(`[role="listbox"] [value="${pageOption}"]`).click();
        } else if (newPageName) {
            await this.page.locator('[role="dialog"] input[placeholder="Enter Page Name"]').clear();
            await this.page.locator('[role="dialog"] input[placeholder="Enter Page Name"]').fill(newPageName);
        }

        await this.page.locator('[role="dialog"]').getByRole("button", { name: "Move to Scratchpad" }).click();
        await this.page.waitForURL(/\/scratchpad/, { timeout: TIMEOUT.ACTION });
    },

    async navigateChatHistory(direction = "up") {
        const key = direction === "up" ? "ArrowUp" : "ArrowDown";
        await this.page.locator('[data-testid="chat-input"]').focus();
        await this.page.keyboard.press(key);
    },

    async getChatInputValue() {
        return await this.page.locator('[data-testid="chat-input"]').inputValue();
    },

    async verifyChatEmpty() {
        await this.page.locator("[data-input-message]").waitFor({ state: "hidden" });
    },

    async waitForChatResponse() {
        await this.page.locator('[data-input-message="user"]').first().waitFor({ timeout: TIMEOUT.ELEMENT });

        const loadingCount = await this.page.locator('[data-testid="loading"]').count();
        if (loadingCount > 0) {
            await this.page.locator('[data-testid="loading"]').waitFor({ state: "hidden", timeout: TIMEOUT.ACTION });
        }

        await expect(async () => {
            const hasSystemMessage = (await this.page.locator('[data-input-message="system"]').count()) > 0;
            const hasErrorState = (await this.page.locator('[data-testid="error-state"]').count()) > 0;
            const hasSQLResult =
                (await this.page.locator('[data-testid="chat-sql-result"] table, [data-testid="sql-result-table"]').count()) > 0;
            const hasAnyResultTable = (await this.page.locator("table").count()) > 0;
            expect(hasSystemMessage || hasErrorState || hasSQLResult || hasAnyResultTable).toBe(true);
        }).toPass({ timeout: TIMEOUT.ACTION });

        // Required: SSE streaming responses render asynchronously via a queue
        await this.page.waitForTimeout(500);
    },
};
