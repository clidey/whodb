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
 * Browser state helper for Playwright E2E tests.
 */

/**
 * Clears browser state for test isolation.
 * @param {import('@playwright/test').Page} page
 */
export async function clearBrowserState(page) {
  await page.context().clearCookies();
  // localStorage/sessionStorage are only accessible on a real page, not about:blank
  const url = page.url();
  if (url && url !== "about:blank") {
    await page.evaluate(() => {
      localStorage.clear();
      localStorage.setItem("whodb.analytics.consent", "denied");
      sessionStorage.clear();
      if (indexedDB && indexedDB.databases) {
        indexedDB.databases().then((databases) => {
          databases.forEach((db) => {
            indexedDB.deleteDatabase(db.name);
          });
        });
      }
    });
  }
}
