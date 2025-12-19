/*
 * Copyright 2025 Clidey, Inc.
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

import {defineConfig} from "cypress";
import codeCoverageTask from "@cypress/code-coverage/task.js";
import createBundler from "@bahmutov/cypress-esbuild-preprocessor";
import {exec as execCallback} from "node:child_process";
import {existsSync, readdirSync, readFileSync} from "node:fs";
import path from "node:path";
import {fileURLToPath} from "node:url";
import {promisify} from "node:util";

const exec = promisify(execCallback);
const __dirname = path.dirname(fileURLToPath(import.meta.url));

function loadAdditionalDatabaseConfigs() {
    const eeFixturesDir = path.resolve(__dirname, "../ee/frontend/cypress/fixtures/databases");
    if (!existsSync(eeFixturesDir)) {
        return {};
    }

    const configs = {};
    for (const file of readdirSync(eeFixturesDir)) {
        if (!file.endsWith(".json")) {
            continue;
        }

        try {
            const name = path.basename(file, ".json");
            const raw = readFileSync(path.join(eeFixturesDir, file), "utf8");
            configs[name] = JSON.parse(raw);
        } catch (error) {
            console.warn(`⚠️ Failed to load EE fixture ${file}: ${error.message}`);
        }
    }

    return configs;
}

export default defineConfig({
    numTestsKeptInMemory: 0,
    viewportWidth: 1280,
    viewportHeight: 768,
    // Screenshot and video settings
    screenshotOnRunFailure: true,
    screenshotsFolder: 'cypress/screenshots',
    video: false,  // Disabled for performance - screenshots capture failures
    videosFolder: 'cypress/videos',
    trashAssetsBeforeRuns: false,  // Disabled - run-cypress.sh handles cleanup once at suite start
  e2e: {
      baseUrl: 'http://localhost:3000', // Default for local development
      testIsolation: true, // Ensure clean state between tests
      experimentalMemoryManagement: true, // Reduce memory pressure during long test runs
    async setupNodeEvents(on, config) {
        codeCoverageTask(on, config);
        on('file:preprocessor', createBundler({
          sourcemap: 'inline', // Enable source maps for better debugging
        }));

        config.env = config.env || {};
        const additionalConfigs = loadAdditionalDatabaseConfigs();
        if (Object.keys(additionalConfigs).length > 0) {
          config.env.additionalDatabaseConfigs = additionalConfigs;
        }

      on("task", {
        async execCommand(cmd) {
          try {
            const {stdout = "", stderr = ""} = await exec(cmd, {shell: true});
            const output = [stdout, stderr].filter(Boolean).join("\n");
            return {success: true, output};
          } catch (error) {
            const stderr = error?.stderr || error?.stdout || error?.message || String(error);
            return {success: false, error: stderr};
          }
        },
        // JSON parsing tasks - faster in Node than browser
        parseJSON(text) {
          try {
            return JSON.parse(text);
          } catch (e) {
            return null;
          }
        },
        parseDocuments(rows) {
          // Parse document column (index 1) from multiple rows
          return rows.map(row => {
            try {
              return JSON.parse(row[1] || '{}');
            } catch (e) {
              return {};
            }
          });
        },
      });

      // list of browsers in order of preference
      const preferred = ["chromium", "chrome", "edge", "firefox", "electron"];

      // Cypress gives you detected browsers in config
      const installed = (config.browsers || []).map((b) => b.name);
      const found = preferred.find((name) => installed.includes(name));

      if (found) {
        console.log(`✅ Found preferred browser: ${found}`);
        // Instead of setting config.browser here,
        // tell Cypress to use it when you launch
        config.env.PREFERRED_BROWSER = found;
      } else {
        console.warn("⚠️ No preferred browser found, Cypress will use default.");
      }

      return config;
    },
  },
});
