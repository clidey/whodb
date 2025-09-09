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

export default defineConfig({
    numTestsKeptInMemory: 0,
    experimentalStudio: true,
  e2e: {
      baseUrl: 'http://localhost:3000', // Default for local development
    async setupNodeEvents(on, config) {
        codeCoverageTask(on, config);

        // Pass Docker flag to tests
        config.env = config.env || {};
        config.env.isDocker = process.env.CYPRESS_IN_DOCKER === 'true';

      on("task", {
        async execCommand(command) {
          try {
            const result = await command(command, { shell: true });
            return { success: true, output: result.toString() };
          } catch (error) {
            return { success: false, error: error.toString() };
          }
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
