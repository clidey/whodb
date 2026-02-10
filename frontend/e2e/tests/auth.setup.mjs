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
 * Authentication setup project.
 *
 * Runs ONCE before any test project (via Playwright's `dependencies` config).
 * Logs into the target database and saves the browser state to a JSON file.
 * All test projects then load this file via `storageState`, starting
 * every test already authenticated — no login form, no race conditions.
 *
 * This follows Playwright's official recommendation:
 * https://playwright.dev/docs/auth#basic-shared-account-in-all-tests
 */

import fs from "fs";
import path from "path";
import { test as setup } from "@playwright/test";
import { WhoDB } from "../support/whodb.mjs";
import {
  getDatabasesByCategory,
  getDatabaseId,
} from "../support/database-config.mjs";

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const AUTH_DIR = path.resolve(process.cwd(), "e2e", ".auth");

// Mock AI providers during login to avoid backend Ollama timeouts
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

// Get the target database from env (set by run-e2e.sh per database)
const targetDb = process.env.DATABASE;
const allDatabases = getDatabasesByCategory("all");
const databases = targetDb
  ? allDatabases.filter(
      (db) =>
        getDatabaseId(db) === targetDb.toLowerCase() ||
        db.type.toLowerCase() === targetDb.toLowerCase()
    )
  : allDatabases;

// Create one setup test per database
for (const dbConfig of databases) {
  const id = getDatabaseId(dbConfig);
  const authFile = path.join(AUTH_DIR, `${id}.json`);

  setup(`authenticate ${dbConfig.type}`, async ({ page }) => {
    await mockAIProviders(page);

    const whodb = new WhoDB(page);
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

    // Save authenticated state
    if (!fs.existsSync(AUTH_DIR)) {
      fs.mkdirSync(AUTH_DIR, { recursive: true });
    }
    await page.context().storageState({ path: authFile });
  });
}
