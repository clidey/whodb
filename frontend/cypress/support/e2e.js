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

// Import shared helpers
import {clearBrowserState, disableAnimations} from './helpers/animation'

// Handle uncaught exceptions that shouldn't fail tests
Cypress.on('uncaught:exception', (err) => {
    // Ignore clipboard errors that occur when document loses focus during test cleanup
    if (err.message.includes('Document is not focused') ||
        err.message.includes('writeText') ||
        err.name === 'NotAllowedError') {
        return false;
    }
    // Let other errors fail the test
    return true;
});

// Clear browser state before each test to ensure isolation
beforeEach(() => {
    clearBrowserState();

    // Inject CSS to disable animations on every page visit
    cy.on('window:before:load', disableAnimations);
});
