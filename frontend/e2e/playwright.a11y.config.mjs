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
 * Playwright config for accessibility scans.
 *
 * The default E2E config ignores accessibility specs to keep the main E2E
 * suite lean. For a11y workflows, we explicitly enable accessibility specs.
 */

import baseConfig from "./playwright.config.mjs";

function withoutAccessibilityIgnore(value) {
  if (!value) return value;
  return value.filter((p) => {
    if (typeof p === "string") return !p.includes("accessibility");
    if (p instanceof RegExp) return p.source !== "accessibility";
    return true;
  });
}

export default {
  ...baseConfig,
  testIgnore: withoutAccessibilityIgnore(baseConfig.testIgnore),
  projects: (baseConfig.projects || []).map((p) => ({
    ...p,
    testIgnore: withoutAccessibilityIgnore(p.testIgnore),
  })),
};

