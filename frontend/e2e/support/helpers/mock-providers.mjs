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

/**
 * Mock AI provider/model calls to prevent backend Ollama timeouts.
 * Chat tests that call setupChatMock() override this with their own
 * mock data (Playwright routes are LIFO - last registered runs first).
 */
export async function mockAIProviders(page) {
  await page.route("**/api/query", async (route) => {
    // postDataJSON() throws on multipart form data (file uploads).
    // Skip non-JSON requests so they pass through to the server.
    let postData;
    try {
      postData = route.request().postDataJSON();
    } catch {
      return route.fallback();
    }
    const op = postData?.operationName;

    if (op === "GetAIProviders") {
      return route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ data: { AIProviders: [] } }),
      });
    }
    if (op === "GetAIModels") {
      return route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ data: { AIModel: [] } }),
      });
    }

    await route.fallback();
  });
}
