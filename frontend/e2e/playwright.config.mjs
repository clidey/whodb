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
 * Two projects:
 *   - "standalone": Launches its own Chromium, hits WhoDB directly.
 *     Use for local dev or Docker-network testing.
 *   - "gateway": Connects to an existing browser via CDP.
 *     Use for testing through an external browser (e.g., embedded in a container).
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
    {
      name: "standalone",
      use: {
        browserName: "chromium",
        viewport: { width: 1920, height: 1080 },
        launchOptions: {
          args: [
            "--window-size=1920,1080",
            "--force-device-scale-factor=1",
          ],
        },
      },
    },
    {
      name: "gateway",
      use: {
        browserName: "chromium",
        viewport: { width: 1920, height: 1080 },
      },
    },
  ],
});
