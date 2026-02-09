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
 * Shared animation disabling helper for Playwright E2E tests.
 * Ported from cypress/support/helpers/animation.js
 */

export const ANIMATION_DISABLE_CSS = `
*, *::before, *::after {
  -moz-animation: none !important;
  -moz-transition: none !important;
  -webkit-animation: none !important;
  -webkit-transition: none !important;
  animation: none !important;
  transition: none !important;
  animation-duration: 0ms !important;
  animation-delay: 0ms !important;
  transition-duration: 0ms !important;
  transition-delay: 0ms !important;
}
`;

/**
 * Injects CSS to disable all animations on the page.
 * @param {import('@playwright/test').Page} page
 */
export async function disableAnimations(page) {
  await page.addStyleTag({ content: ANIMATION_DISABLE_CSS });
}

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

/**
 * Standard test setup that should run before each test.
 * @param {import('@playwright/test').Page} page
 */
export async function standardTestSetup(page) {
  await clearBrowserState(page);
  // Playwright's addInitScript runs before every navigation, equivalent to window:before:load
  await page.addInitScript(() => {
    const style = document.createElement("style");
    style.setAttribute("data-pw-animation-disable", "true");
    style.textContent = `
      *, *::before, *::after {
        animation: none !important;
        transition: none !important;
        animation-duration: 0ms !important;
        animation-delay: 0ms !important;
        transition-duration: 0ms !important;
        transition-delay: 0ms !important;
      }
    `;
    document.head.appendChild(style);
  });
}
