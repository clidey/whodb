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
 * Global setup for Playwright E2E tests.
 *
 * - Animation disabling: done via page.addInitScript() in test-fixture.mjs
 * - Browser state clearing: done via clearBrowserState() in animation.mjs
 * - Uncaught exceptions: Playwright doesn't fail on page errors by default
 *
 * This file is imported by the playwright config as a global setup module.
 */

import {getDatabaseConfigs} from "./database-config.mjs";
import {assertFixturesValid} from "./helpers/fixture-validator.mjs";

export default async function globalSetup() {
  // Validate fixtures on startup — aborts the run if any fixture is invalid
  const configs = getDatabaseConfigs();
  assertFixturesValid(configs);
}
