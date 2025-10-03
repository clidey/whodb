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

// ***********************************************************
// This example support/e2e.js is processed and
// loaded automatically before your test files.
//
// This is a great place to put global configuration and
// behavior that modifies Cypress.
//
// You can change the location of this file or turn off
// automatically serving support files with the
// 'supportFile' configuration option.
//
// You can read more here:
// https://on.cypress.io/configuration
// ***********************************************************

// Import commands.js using ES2015 syntax:
import './commands'

import '@cypress/code-coverage/support'

// Clear browser state before each test to ensure isolation
beforeEach(() => {
    // Clear cookies for the current domain
    cy.clearCookies();

    // Clear all localStorage
    cy.window().then((win) => {
        win.localStorage.clear();
    });

    // Clear all sessionStorage
    cy.window().then((win) => {
        win.sessionStorage.clear();
    });

    // Clear IndexedDB if used (common for state persistence)
    cy.window().then((win) => {
        if (win.indexedDB && win.indexedDB.databases) {
            win.indexedDB.databases().then(databases => {
                databases.forEach(db => {
                    win.indexedDB.deleteDatabase(db.name);
                });
            });
        }
    });

    // Inject CSS to disable animations on every page visit
    cy.on('window:before:load', (win) => {
        const style = win.document.createElement('style');
        style.innerHTML = `
            *,
            *::before,
            *::after {
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
        win.document.head.appendChild(style);
    });
});