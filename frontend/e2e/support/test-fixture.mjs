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
 * Playwright test fixture for WhoDB.
 *
 * Provides:
 *   - `whodb`: WhoDB helper instance with all page commands
 *   - `forEachDatabase()`: Iterates databases with session persistence
 *
 * Session management:
 *   Tests with `login: true` (default) use Playwright's storageState to persist
 *   login sessions. A one-time beforeAll logs in and saves browser state to a file.
 *   Each test gets a fresh browser context pre-loaded with that state — no login
 *   form, no race conditions, no logout between tests.
 *
 *   Tests with `login: false` (login.spec, ssl-*.spec, error-handling.spec)
 *   manage their own login flow and don't use storageState.
 *
 * Dual mode:
 *   - When CDP_ENDPOINT is set → connects to an existing browser via CDP
 *   - Otherwise → launches standalone Chromium (default)
 */

import fs from "fs";
import path from "path";
import { test as base, expect, chromium } from "@playwright/test";
import { WhoDB } from "./whodb.mjs";
import {
  getDatabasesByCategory,
  getDatabaseId,
  hasFeature,
} from "./database-config.mjs";
import { VALID_FEATURES } from "./helpers/fixture-validator.mjs";

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
// Optional Wails bindings stub to exercise desktop-only code paths in Playwright.
const DESKTOP_STUB = ["true", "1"].includes((process.env.E2E_DESKTOP_STUB || "").toLowerCase());
const NYC_OUTPUT_DIR = path.resolve(process.cwd(), ".nyc_output");
const AUTH_DIR = path.resolve(process.cwd(), "e2e", ".auth");

const CDP_ENDPOINT = process.env.CDP_ENDPOINT;

/**
 * Mock AI provider/model calls to prevent backend Ollama timeouts.
 * Chat tests that call setupChatMock() override this with their own
 * mock data (Playwright routes are LIFO - last registered runs first).
 */
async function mockAIProviders(page) {
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

/**
 * Collect Istanbul code coverage from the browser after each test.
 */
async function collectCoverage(page, testInfo) {
  try {
    const coverage = await page.evaluate(() => window.__coverage__);
    if (coverage) {
      if (!fs.existsSync(NYC_OUTPUT_DIR)) {
        fs.mkdirSync(NYC_OUTPUT_DIR, { recursive: true });
      }
      const fileName = `coverage-${testInfo.testId}.json`;
      fs.writeFileSync(
        path.join(NYC_OUTPUT_DIR, fileName),
        JSON.stringify(coverage)
      );
    }
  } catch {
    // Page may have closed or navigated away — skip silently
  }
}

/**
 * Get the storageState file path for a database config.
 */
function getAuthFile(dbConfig) {
  const id = getDatabaseId(dbConfig);
  return path.join(AUTH_DIR, `${id}.json`);
}

export const test = CDP_ENDPOINT
  ? base.extend({
      // CDP mode: connect to an existing browser via Chrome DevTools Protocol.
      page: async ({}, use) => {
        const browser = await chromium.connectOverCDP(CDP_ENDPOINT);
        const contexts = browser.contexts();

        let whodbPage = null;
        for (const ctx of contexts) {
          for (const p of ctx.pages()) {
            const url = p.url();
            if (url.includes("whodb") || url.includes(":8080")) {
              whodbPage = p;
              break;
            }
          }
          if (whodbPage) break;
        }

        if (!whodbPage) {
          const ctx = contexts[0] || (await browser.newContext());
          whodbPage = await ctx.newPage();
        }

        await use(whodbPage);
      },

      whodb: async ({ page }, use, testInfo) => {
        await page.addInitScript((enableDesktopStub) => {
          window.__E2E_DISABLE_AUTOCOMPLETE = true;
          if (enableDesktopStub) {
            const go = window.go || {};
            go.common = go.common || {};
            go.main = go.main || {};
            go.common.App = go.common.App || { OpenURL: () => Promise.resolve() };
            go.main.App = go.main.App || { OpenURL: () => Promise.resolve() };
            window.go = go;
          }
        }, DESKTOP_STUB);
        await mockAIProviders(page);
        await use(new WhoDB(page));
        await collectCoverage(page, testInfo);
      },
    })
  : base.extend({
      whodb: async ({ page }, use, testInfo) => {
        await page.addInitScript((enableDesktopStub) => {
          window.__E2E_DISABLE_AUTOCOMPLETE = true;
          if (enableDesktopStub) {
            const go = window.go || {};
            go.common = go.common || {};
            go.main = go.main || {};
            go.common.App = go.common.App || { OpenURL: () => Promise.resolve() };
            go.main.App = go.main.App || { OpenURL: () => Promise.resolve() };
            window.go = go;
          }
        }, DESKTOP_STUB);
        await mockAIProviders(page);
        await use(new WhoDB(page));
        await collectCoverage(page, testInfo);
      },
    });

export { expect };

/**
 * Replacement for Cypress's forEachDatabase().
 *
 * When login=true (default), uses Playwright's storageState for session persistence:
 *   - beforeAll: Logs in once, saves browser state to e2e/.auth/{db}.json
 *   - test.use({ storageState }): Each test starts pre-authenticated
 *   - No login form interaction per test, no logout between tests
 *   This matches Cypress's cy.session({ cacheAcrossSpecs: true }).
 *
 * When login=false (login.spec, ssl-*.spec, error-handling.spec):
 *   - No session persistence, no auto-login/logout
 *   - Tests manage their own login flow
 *
 * @param {string} categoryFilter - 'sql', 'document', 'keyvalue', or 'all'
 * @param {Function} testFn - (db) => { test('...', async ({whodb}) => { ... }) }
 * @param {Object} options
 * @param {boolean} options.login - Use storageState session persistence (default: true)
 * @param {boolean} options.logout - Ignored when login=true (no logout needed with storageState)
 * @param {boolean} options.navigateToStorageUnit - Navigate to storage-unit before each test (default: true)
 * @param {string[]} options.features - Required features; databases without them are skipped
 */
export function forEachDatabase(categoryFilter, testFn, options = {}) {
  const {
    login = true,
    logout = true,
    navigateToStorageUnit = true,
    features = [],
  } = options;

  // Validate requested features
  for (const feature of features) {
    if (!VALID_FEATURES.includes(feature)) {
      throw new Error(
        `Unknown feature '${feature}' in forEachDatabase options. Valid features: ${VALID_FEATURES.join(", ")}`
      );
    }
  }

  const targetDb = process.env.DATABASE;
  const targetCategory = process.env.CATEGORY;

  let databases = getDatabasesByCategory(categoryFilter);

  // Filter by required features
  if (features.length > 0) {
    databases = databases.filter((db) =>
      features.every((f) => hasFeature(db, f))
    );
  }

  // If running specific database, filter to just that one
  if (targetDb) {
    databases = databases.filter((db) => {
      const id = getDatabaseId(db);
      return (
        id === targetDb.toLowerCase() ||
        db.type.toLowerCase() === targetDb.toLowerCase()
      );
    });
  }

  // If running specific category via env, skip non-matching blocks
  if (targetCategory && categoryFilter !== "all") {
    if (categoryFilter !== targetCategory) return;
  }

  if (databases.length === 0) return;

  for (const dbConfig of databases) {
    const authFile = getAuthFile(dbConfig);

    test.describe(`[${dbConfig.type}]`, () => {
      if (login) {
        // --- Session persistence mode (default) ---
        // Auth is handled by the "setup" project (auth.setup.mjs) which runs
        // BEFORE this project via Playwright's `dependencies` config.
        // The setup project saves storageState to e2e/.auth/{db}.json.
        // Each test gets a fresh browser context pre-loaded with that state.

        test.use({ storageState: authFile });

        test.beforeEach(async ({ whodb, page }) => {
          if (navigateToStorageUnit) {
            await page.goto(`${BASE_URL}/storage-unit`);
            await page
              .locator('[data-testid="storage-unit-card"]')
              .first()
              .waitFor({ timeout: 15000 });
          }
        });

      } else {
        // --- Manual login mode (login.spec, ssl-*.spec, error-handling.spec) ---
        // Tests handle their own login/logout. No storageState.

        if (logout) {
          test.afterEach(async ({ whodb }) => {
            await whodb.logout();
          });
        }
      }

      testFn(dbConfig);
    });
  }
}

/**
 * Login to database using WhoDB helper (standalone usage outside forEachDatabase).
 */
export async function loginToDatabase(whodb, dbConfig, options = {}) {
  const { visitStorageUnit = true } = options;
  const conn = dbConfig.connection;
  await whodb.login(
    dbConfig.uiType || dbConfig.type,
    conn.host ?? undefined,
    conn.user ?? undefined,
    conn.password ?? undefined,
    conn.database ?? undefined,
    conn.advanced || {}
  );
  if (dbConfig.schema && dbConfig.sidebar?.showsSchemaDropdown) {
    await whodb.selectSchema(dbConfig.schema);
  }
  if (visitStorageUnit) {
    await whodb.goto("/storage-unit");
    await whodb.page
      .locator('[data-testid="storage-unit-card"]')
      .first()
      .waitFor({ timeout: 15000 });
  }
}

export function skipIfNoFeature(dbConfig, feature) {
  return !hasFeature(dbConfig, feature);
}

export function conditionalTest(condition, name, fn) {
  if (condition) {
    test(name, fn);
  } else {
    test.skip(name, fn);
  }
}

export function conditionalDescribe(condition, name, fn) {
  if (condition) {
    test.describe(name, fn);
  } else {
    test.describe.skip(name, fn);
  }
}
