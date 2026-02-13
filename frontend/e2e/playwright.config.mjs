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
 * Playwright config for WhoDB E2E tests.
 *
 * Projects:
 *   - "setup": Authenticates once per database, saves storageState.
 *     Runs BEFORE test projects via `dependencies`.
 *   - "standalone": Read-only tests (launches its own Chromium).
 *   - "standalone-mutating": Destructive tests (run-e2e.sh invokes AFTER standalone).
 *   - "gateway": Read-only tests (connects via CDP).
 *   - "gateway-mutating": Destructive tests (run-e2e.sh invokes AFTER gateway).
 *
 * Environment variables:
 *   BASE_URL        - WhoDB URL (default: http://localhost:3000 for local dev)
 *   CDP_ENDPOINT    - CDP URL for connecting to an existing browser
 *   DATABASE        - Target single database (e.g., "postgres")
 *   CATEGORY        - Target category (e.g., "sql", "document", "keyvalue")
 */

import { defineConfig } from "@playwright/test";

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const DATABASE = process.env.DATABASE || "default";

/** Test files that mutate data (INSERT, UPDATE, DELETE, DROP, etc.). */
const MUTATING_TESTS = [
  /crud\.spec/,
  /mock-data\.spec/,
  /import\.spec/,
  /data-types\.spec/,
  /key-types\.spec/,
  /schema-management\.spec/,
  /chat\.spec/,
  /keyboard-shortcuts\.spec/,
  /type-casting\.spec/,
];

/** Shared browser config for standalone projects (launches own Chromium). */
const standaloneBrowser = {
  browserName: "chromium",
  viewport: { width: 1920, height: 1080 },
  launchOptions: {
    args: [
      "--window-size=1920,1080",
      "--force-device-scale-factor=1",
    ],
  },
};

/** Shared browser config for gateway projects (connects via CDP). */
const gatewayBrowser = {
  browserName: "chromium",
  viewport: { width: 1920, height: 1080 },
};

export default defineConfig({
  globalSetup: "./support/global-setup.mjs",
  testDir: "./tests",
  testIgnore: ["**/postgres-screenshots*"],
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 2 : 1,
  reporter: [
    ["blob", { outputDir: "reports/blobs" }],
    ["list"],
  ],
  outputDir: `reports/test-results/${DATABASE}`,
  use: {
    baseURL: BASE_URL,
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    trace: "retain-on-failure",
  },
  projects: [
    // Setup project: logs in once per database, saves auth state.
    // Runs before test projects via `dependencies`.
    {
      name: "setup",
      testMatch: /auth\.setup\.mjs/,
      use: standaloneBrowser,
    },

    // Read-only tests run first — excludes mutating test files.
    {
      name: "standalone",
      dependencies: ["setup"],
      testIgnore: [/auth\.setup\.mjs/, /postgres-screenshots/, ...MUTATING_TESTS],
      use: standaloneBrowser,
    },
    // Destructive tests — run via a separate Playwright invocation in run-e2e.sh
    // so they always execute regardless of read-only test results.
    {
      name: "standalone-mutating",
      dependencies: ["setup"],
      testMatch: MUTATING_TESTS,
      use: standaloneBrowser,
    },

    // Gateway: same read-only → mutating split.
    {
      name: "gateway",
      dependencies: ["setup"],
      testIgnore: [/auth\.setup\.mjs/, /postgres-screenshots/, ...MUTATING_TESTS],
      use: gatewayBrowser,
    },
    {
      name: "gateway-mutating",
      dependencies: ["setup"],
      testMatch: MUTATING_TESTS,
      use: gatewayBrowser,
    },
  ],
});
