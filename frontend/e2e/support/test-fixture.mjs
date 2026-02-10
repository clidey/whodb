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
 *   - `forEachDatabase()`: Iterates databases with auto-login/logout
 *
 * Dual mode:
 *   - When CDP_ENDPOINT is set → connects to an existing browser via CDP
 *   - Otherwise → launches standalone Chromium (default)
 *
 * Usage:
 *   import { test, expect, forEachDatabase } from '../support/test-fixture.mjs';
 *
 *   test.describe('My Feature', () => {
 *     forEachDatabase('sql', (db) => {
 *       test('does something', async ({ whodb }) => {
 *         const tables = await whodb.getTables();
 *         expect(tables).toContain('users');
 *       });
 *     });
 *   });
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

const NYC_OUTPUT_DIR = path.resolve(process.cwd(), ".nyc_output");

const CDP_ENDPOINT = process.env.CDP_ENDPOINT;

/**
 * Mock AI provider/model calls to prevent backend Ollama timeouts.
 * No Ollama is running in the test environment, so these calls would
 * spam timeout errors in the backend logs for ~5s each.
 *
 * Chat tests that call setupChatMock() override this with their own
 * mock data (Playwright routes are LIFO - last registered runs first).
 */
async function mockAIProviders(page) {
  await page.route("**/api/query", async (route) => {
    const postData = route.request().postDataJSON?.();
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
 * vite-plugin-istanbul instruments the app when NODE_ENV=test,
 * exposing window.__coverage__. We grab it and write to .nyc_output/
 * so `nyc report` works exactly as it did with Cypress.
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
        // Don't close - the CDP browser is managed externally.
      },

      whodb: async ({ page }, use, testInfo) => {
        await mockAIProviders(page);
        await use(new WhoDB(page));
        await collectCoverage(page, testInfo);
      },
    })
  : base.extend({
      // Standalone mode: use Playwright's default browser lifecycle (fast, shared browser).
      whodb: async ({ page }, use, testInfo) => {
        await mockAIProviders(page);
        await use(new WhoDB(page));
        await collectCoverage(page, testInfo);
      },
    });

export { expect };

/**
 * Replacement for Cypress's forEachDatabase().
 * Same API - test authors don't need to change their test structure.
 *
 * @param {string} categoryFilter - 'sql', 'document', 'keyvalue', or 'all'
 * @param {Function} testFn - (db) => { test('...', async ({whodb}) => { ... }) }
 * @param {Object} options
 * @param {boolean} options.login - Auto-login before each test (default: true)
 * @param {boolean} options.logout - Auto-logout after each test (default: true)
 * @param {boolean} options.navigateToStorageUnit - Navigate to storage-unit after login (default: true)
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
    test.describe(`[${dbConfig.type}]`, () => {
      if (login) {
        test.beforeEach(async ({ whodb, page }) => {
          const conn = dbConfig.connection;
          await whodb.login(
            dbConfig.uiType || dbConfig.type,
            conn.host ?? undefined,
            conn.user ?? undefined,
            conn.password ?? undefined,
            conn.database ?? undefined,
            conn.advanced || {},
            dbConfig.schema ?? null
          );
          if (dbConfig.schema && dbConfig.sidebar?.showsSchemaDropdown) {
            await whodb.selectSchema(dbConfig.schema);
          }
          if (navigateToStorageUnit) {
            // After login, the app navigates to /storage-unit automatically.
            // If selectSchema was called, it already waited for storage-unit-cards.
            // Only navigate if we're not already there.
            const hasCards = await page.locator('[data-testid="storage-unit-card"]').count();
            if (hasCards === 0) {
              await page.goto(whodb.url("/storage-unit"));
              await page
                .locator('[data-testid="storage-unit-card"]')
                .first()
                .waitFor({ timeout: 15000 });
            }
          }
        });
      }

      if (logout) {
        test.afterEach(async ({ whodb }) => {
          await whodb.logout();
        });
      }

      testFn(dbConfig);
    });
  }
}

/**
 * Login to database using WhoDB helper (standalone usage outside forEachDatabase).
 * Equivalent to Cypress's loginToDatabase() from test-runner.js.
 * @param {WhoDB} whodb - WhoDB helper instance
 * @param {Object} dbConfig - Database configuration
 * @param {Object} options
 * @param {boolean} options.visitStorageUnit - Navigate to storage-unit after login (default: true)
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

/**
 * Skip test if feature is not supported.
 * @param {Object} dbConfig - Database configuration
 * @param {string} feature - Feature name
 * @returns {boolean} true if feature is missing (should skip)
 */
export function skipIfNoFeature(dbConfig, feature) {
  return !hasFeature(dbConfig, feature);
}

/** Conditional test - only runs if condition is met */
export function conditionalTest(condition, name, fn) {
  if (condition) {
    test(name, fn);
  } else {
    test.skip(name, fn);
  }
}

/** Conditional describe - only runs if condition is met */
export function conditionalDescribe(condition, name, fn) {
  if (condition) {
    test.describe(name, fn);
  } else {
    test.describe.skip(name, fn);
  }
}
