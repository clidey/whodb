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
 *   - "standalone-mutating": Destructive tests (depends on "standalone", runs after).
 *   - "gateway": Read-only tests (connects via CDP).
 *   - "gateway-mutating": Destructive tests (depends on "gateway", runs after).
 *
 * EE override:
 *   When EE_E2E_DIR is set (by ee/dev/run-e2e.sh), EE test files are included
 *   via an "ee-standalone" project. If an EE test file has the same name as a CE
 *   test file (e.g. ssl-modes.spec.mjs), the CE version is excluded so only the
 *   EE version runs.
 *
 * Environment variables:
 *   BASE_URL        - WhoDB URL (default: http://localhost:3000 for local dev)
 *   CDP_ENDPOINT    - CDP URL for connecting to an existing browser
 *   DATABASE        - Target single database (e.g., "postgres")
 *   CATEGORY        - Target category (e.g., "sql", "document", "keyvalue")
 *   EE_E2E_DIR      - Path to EE test directory (set by ee/dev/run-e2e.sh)
 */

import { readdirSync } from "fs";
import { defineConfig } from "@playwright/test";

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const DATABASE = process.env.DATABASE || "default";
const EE_E2E_DIR = process.env.EE_E2E_DIR || "";

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

/**
 * Build ignore patterns for CE test files that are overridden by EE.
 * When an EE test file has the same name as a CE test file, the CE version
 * is excluded so only the EE version runs.
 */
function getEEOverrideIgnores() {
  if (!EE_E2E_DIR) return [];
  try {
    const eeFeatureDir = `${EE_E2E_DIR}/features`;
    const eeFiles = readdirSync(eeFeatureDir).filter(f => f.endsWith(".spec.mjs"));
    return eeFiles.map(f => new RegExp(f.replace(/\./g, "\\.")));
  } catch {
    return [];
  }
}

const eeOverrides = getEEOverrideIgnores();

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
  testIgnore: ["**/postgres-screenshots*", "**/accessibility*"],
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
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
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
    // When EE is active, also excludes CE test files overridden by EE.
    {
      name: "standalone",
      dependencies: ["setup"],
      testIgnore: [/auth\.setup\.mjs/, /postgres-screenshots/, /accessibility/, ...MUTATING_TESTS, ...eeOverrides],
      use: standaloneBrowser,
    },
    // Destructive tests — run after read-only tests complete via dependencies.
    // retries: 0 prevents re-running partially-completed mutations.
    {
      name: "standalone-mutating",
      dependencies: ["standalone"],
      retries: 0,
      testMatch: MUTATING_TESTS,
      testIgnore: [...eeOverrides],
      use: standaloneBrowser,
    },

    // Gateway: same read-only → mutating split.
    {
      name: "gateway",
      dependencies: ["setup"],
      testIgnore: [/auth\.setup\.mjs/, /postgres-screenshots/, /accessibility/, ...MUTATING_TESTS, ...eeOverrides],
      use: gatewayBrowser,
    },
    {
      name: "gateway-mutating",
      dependencies: ["gateway"],
      retries: 0,
      testMatch: MUTATING_TESTS,
      testIgnore: [...eeOverrides],
      use: gatewayBrowser,
    },

    // EE-specific tests (only when EE_E2E_DIR is set by ee/dev/run-e2e.sh).
    // Includes EE-only tests AND EE overrides of CE tests.
    ...(EE_E2E_DIR
      ? [
          {
            name: "ee-standalone",
            testDir: EE_E2E_DIR,
            dependencies: ["setup"],
            testIgnore: [/auth\.setup\.mjs/, ...MUTATING_TESTS],
            use: standaloneBrowser,
          },
        ]
      : []),
  ],
});
