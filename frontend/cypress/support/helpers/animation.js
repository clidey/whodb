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

/**
 * Shared animation disabling helper for Cypress E2E tests.
 * Used by both CE and EE test suites to ensure consistent test behavior.
 */

const ANIMATION_DISABLE_CSS = `
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
 * Should be called in window:before:load event.
 * @param {Window} win - The window object from Cypress
 */
export function disableAnimations(win) {
    const style = win.document.createElement('style');
    style.setAttribute('data-cy-animation-disable', 'true');
    style.textContent = ANIMATION_DISABLE_CSS;
    win.document.head.appendChild(style);
}

/**
 * Sets up the animation disabling hook for beforeEach.
 * Call this function once in your support/e2e.js file.
 */
export function setupAnimationDisabling() {
    cy.on('window:before:load', disableAnimations);
}

/**
 * Clears browser state for test isolation.
 * Clears cookies, localStorage, sessionStorage, and IndexedDB.
 * Sets telemetry consent to 'denied' to prevent the consent banner from appearing.
 */
export function clearBrowserState() {
    cy.clearCookies();

    cy.window().then((win) => {
        win.localStorage.clear();
        win.localStorage.setItem('whodb.analytics.consent', 'denied');
    });

    cy.window().then((win) => {
        win.sessionStorage.clear();
    });

    cy.window().then((win) => {
        if (win.indexedDB && win.indexedDB.databases) {
            win.indexedDB.databases().then(databases => {
                databases.forEach(db => {
                    win.indexedDB.deleteDatabase(db.name);
                });
            });
        }
    });
}

/**
 * Standard test setup that should run in beforeEach.
 * Clears browser state and sets up animation disabling.
 */
export function standardTestSetup() {
    clearBrowserState();
    setupAnimationDisabling();
}

export default {
    disableAnimations,
    setupAnimationDisabling,
    clearBrowserState,
    standardTestSetup,
    ANIMATION_DISABLE_CSS
};
