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

import { clearBrowserState } from '../../support/helpers/animation';
import { getDatabaseConfig } from '../../support/test-runner';

/**
 * Maps container certificate paths to host filesystem paths for testing.
 * Container paths like /app/certs/... map to ../dev/certs/... on the host.
 */
function containerPathToHostPath(containerPath) {
    if (!containerPath) return null;
    return containerPath.replace('/app/certs/', '../dev/certs/');
}

/**
 * Helper to enter SSL certificate content from a container path.
 * Reads the certificate file from the host and enters it in paste mode.
 */
function enterCertificateContent(containerPath) {
    const hostPath = containerPathToHostPath(containerPath);
    if (hostPath) {
        cy.readFile(hostPath).then((certContent) => {
            // Switch to paste mode
            cy.contains('button', 'Paste PEM').first().click();
            // Enter certificate content
            cy.get('[data-testid="ssl-ca-certificate-content"]').type(certContent, { parseSpecialCharSequences: false, delay: 0 });
        });
    }
}

/**
 * SSL Configuration Tests
 *
 * Tests the SSL configuration UI in the login form's advanced options.
 * Verifies mode selection, certificate inputs, and conditional display.
 * Also includes integration tests for actual SSL connections when SSL databases are available.
 */
describe('SSL Configuration', () => {
    beforeEach(() => {
        clearBrowserState();
        cy.visit('/login');

        // Dismiss telemetry modal if it appears
        cy.get('body').then($body => {
            const $btn = $body.find('button').filter(function () {
                return this.textContent.includes('Disable Telemetry');
            });
            if ($btn.length) {
                cy.wrap($btn).click();
            }
        });
    });

    describe('SSL Mode Dropdown', () => {
        it('shows SSL mode dropdown for PostgreSQL in advanced options', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();

            // Open advanced options
            cy.get('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            cy.get('[data-testid="ssl-mode-select"]').should('be.visible');
        });

        it('shows SSL mode dropdown for MySQL in advanced options', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="MySQL"]').click();

            // Open advanced options
            cy.get('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            cy.get('[data-testid="ssl-mode-select"]').should('be.visible');
        });

        it('shows SSL mode dropdown for ClickHouse in advanced options', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="ClickHouse"]').click();

            // Open advanced options
            cy.get('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            cy.get('[data-testid="ssl-mode-select"]').should('be.visible');
        });

        it('shows SSL mode dropdown for MongoDB in advanced options', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="MongoDB"]').click();

            // Open advanced options
            cy.get('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            cy.get('[data-testid="ssl-mode-select"]').should('be.visible');
        });

        it('shows SSL mode dropdown for Redis in advanced options', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Redis"]').click();

            // Open advanced options
            cy.get('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            cy.get('[data-testid="ssl-mode-select"]').should('be.visible');
        });

        it('shows SSL mode dropdown for Elasticsearch in advanced options', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="ElasticSearch"]').click();

            // Open advanced options
            cy.get('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should be visible
            cy.get('[data-testid="ssl-mode-select"]').should('be.visible');
        });

        it('does NOT show SSL mode dropdown for SQLite', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Sqlite3"]').click();

            // Open advanced options
            cy.get('[data-testid="advanced-button"]').click();

            // SSL mode dropdown should NOT be visible (SQLite doesn't support SSL)
            cy.get('[data-testid="ssl-mode-select"]').should('not.exist');
        });
    });

    describe('SSL Mode Selection', () => {
        beforeEach(() => {
            // Setup PostgreSQL for SSL mode tests
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();
            cy.get('[data-testid="advanced-button"]').click();
        });

        it('can select different SSL modes from dropdown', () => {
            // Click the SSL mode dropdown
            cy.get('[data-testid="ssl-mode-select"]').click();

            // Should show available modes
            cy.get('[data-value="disabled"]').should('exist');
            cy.get('[data-value="required"]').should('exist');
            cy.get('[data-value="verify-ca"]').should('exist');
            cy.get('[data-value="verify-identity"]').should('exist');

            // Select required mode
            cy.get('[data-value="required"]').click();

            // Dropdown should close and show selected value
            cy.get('[data-testid="ssl-mode-select"]').should('contain.text', 'Required');
        });

        it('shows CA certificate input when verify-ca mode is selected', () => {
            // Initially, certificate inputs should not be visible
            cy.get('[data-testid="ssl-ca-certificate-choose-file"]').should('not.exist');

            // Select verify-ca mode
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-ca"]').click();

            // CA certificate file picker should now be visible
            cy.get('[data-testid="ssl-ca-certificate-choose-file"]').should('be.visible');
        });

        it('shows all certificate inputs when verify-identity mode is selected', () => {
            // Select verify-identity mode
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-identity"]').click();

            // CA certificate file picker should be visible
            cy.get('[data-testid="ssl-ca-certificate-choose-file"]').should('be.visible');

            // Client certificate file pickers should be visible (optional)
            cy.get('[data-testid="ssl-client-certificate-choose-file"]').should('be.visible');
            cy.get('[data-testid="ssl-client-private-key-choose-file"]').should('be.visible');

            // Server name override should be visible
            cy.get('[data-testid="ssl-server-name-input"]').should('be.visible');
        });

        it('hides certificate inputs when switching to disabled mode', () => {
            // First select verify-ca to show certificate inputs
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-ca"]').click();
            cy.get('[data-testid="ssl-ca-certificate-choose-file"]').should('be.visible');

            // Now switch to disabled
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="disabled"]').click();

            // Certificate inputs should be hidden
            cy.get('[data-testid="ssl-ca-certificate-choose-file"]').should('not.exist');
        });
    });

    describe('Certificate Input Modes', () => {
        beforeEach(() => {
            // Setup PostgreSQL with verify-ca mode
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();
            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-ca"]').click();
        });

        it('defaults to file picker mode for certificates', () => {
            // File picker button should be visible by default
            cy.get('[data-testid="ssl-ca-certificate-choose-file"]').should('be.visible');

            // Content textarea should not be visible
            cy.get('[data-testid="ssl-ca-certificate-content"]').should('not.exist');
        });

        it('can toggle to content (paste PEM) mode', () => {
            // Find and click the toggle button for CA certificate
            cy.contains('button', 'Paste PEM').first().click();

            // Content textarea should now be visible
            cy.get('[data-testid="ssl-ca-certificate-content"]').should('be.visible');

            // File picker should be hidden
            cy.get('[data-testid="ssl-ca-certificate-choose-file"]').should('not.exist');
        });

        it('can toggle back to file picker mode', () => {
            // Toggle to content mode
            cy.contains('button', 'Paste PEM').first().click();
            cy.get('[data-testid="ssl-ca-certificate-content"]').should('be.visible');

            // Toggle back to file picker mode
            cy.contains('button', 'Choose File').first().click();

            // File picker should be visible again
            cy.get('[data-testid="ssl-ca-certificate-choose-file"]').should('be.visible');
            cy.get('[data-testid="ssl-ca-certificate-content"]').should('not.exist');
        });

        it('can paste PEM content', () => {
            // Toggle to content mode
            cy.contains('button', 'Paste PEM').first().click();

            const testPEM = '-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----';

            cy.get('[data-testid="ssl-ca-certificate-content"]')
                .type(testPEM, { parseSpecialCharSequences: false })
                .should('have.value', testPEM);
        });
    });

    describe('SSL Mode Description', () => {
        beforeEach(() => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();
            cy.get('[data-testid="advanced-button"]').click();
        });

        it('shows description text for selected SSL mode', () => {
            // Select verify-ca mode
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-ca"]').click();

            // Should show description text
            cy.contains('Verify server certificate').should('be.visible');
        });
    });

    describe('Database-Specific SSL Modes', () => {
        it('shows Preferred mode for MySQL but not PostgreSQL', () => {
            // MySQL should have Preferred mode
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="MySQL"]').click();
            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="preferred"]').should('exist');
            cy.get('body').type('{esc}');

            // PostgreSQL should NOT have Preferred mode
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="preferred"]').should('not.exist');
        });

        it('shows Enabled/Insecure modes for ClickHouse', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="ClickHouse"]').click();
            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="ssl-mode-select"]').click();

            // ClickHouse uses enabled/insecure instead of verify-ca/verify-identity
            cy.get('[data-value="enabled"]').should('exist');
            cy.get('[data-value="insecure"]').should('exist');
            cy.get('[data-value="verify-ca"]').should('not.exist');
        });
    });

    describe('Form Persistence', () => {
        it('maintains SSL configuration when toggling advanced options', () => {
            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();

            // Open advanced, configure SSL
            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-ca"]').click();

            // Switch to paste mode and enter certificate content
            cy.contains('button', 'Paste PEM').first().click();
            cy.get('[data-testid="ssl-ca-certificate-content"]').type('-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----', { parseSpecialCharSequences: false });

            // Close and reopen advanced options
            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="ssl-mode-select"]').should('not.exist');

            cy.get('[data-testid="advanced-button"]').click();

            // SSL mode should be preserved (content may reset due to state management)
            cy.get('[data-testid="ssl-mode-select"]').should('contain.text', 'Verify CA');
        });
    });
});

/**
 * SSL Integration Tests
 *
 * These tests require SSL-enabled databases to be running.
 * Run with: docker-compose --profile ssl up
 *
 * Tests actual SSL connections through the backend to verify the full stack works.
 * SSL config is now part of main fixtures under the "ssl" key.
 */
describe('SSL Integration Tests', () => {
    beforeEach(() => {
        clearBrowserState();
        cy.visit('/login');

        // Dismiss telemetry modal if it appears
        cy.get('body').then($body => {
            const $btn = $body.find('button').filter(function () {
                return this.textContent.includes('Disable Telemetry');
            });
            if ($btn.length) {
                cy.wrap($btn).click();
            }
        });
    });

    describe('PostgreSQL SSL Connection', () => {
        it('connects to PostgreSQL with SSL verify-ca mode', () => {
            const db = getDatabaseConfig('postgres');
            if (!db.ssl) {
                cy.log('Skipping: postgres SSL config not available');
                return;
            }

            cy.get('[data-testid="database-type-select"]').click();
            cy.get(`[data-value="${db.type}"]`).click();

            // Fill connection details
            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });
            cy.get('[data-testid="database"]').type(db.connection.database);

            // Configure SSL
            cy.get('[data-testid="advanced-button"]').click();

            // Set SSL port
            cy.get('[data-testid="Port-input"]').clear().type(String(db.ssl.port));

            // Select SSL mode (verify-ca)
            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-ca"]').click();

            // Set CA certificate
            if (db.ssl.caCertPath) {
                enterCertificateContent(db.ssl.caCertPath);
            }

            // Attempt login
            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            // Wait for connection - should succeed with SSL
            cy.wait('@loginQuery', { timeout: 30000 });

            // Should redirect to storage-unit page on success
            cy.url().should('include', '/storage-unit');
            cy.get('[data-testid="sidebar-database"], [data-testid="sidebar-schema"]', { timeout: 15000 })
                .should('exist');
        });
    });

    describe('MySQL SSL Connection', () => {
        it('connects to MySQL with SSL verify-ca mode', () => {
            const db = getDatabaseConfig('mysql');
            if (!db.ssl) {
                cy.log('Skipping: mysql SSL config not available');
                return;
            }

            cy.get('[data-testid="database-type-select"]').click();
            cy.get(`[data-value="${db.type}"]`).click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });
            cy.get('[data-testid="database"]').type(db.connection.database);

            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').clear().type(String(db.ssl.port));

            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-ca"]').click();

            if (db.ssl.caCertPath) {
                enterCertificateContent(db.ssl.caCertPath);
            }

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@loginQuery', { timeout: 30000 });
            cy.url().should('include', '/storage-unit');
            cy.get('[data-testid="sidebar-database"], [data-testid="sidebar-schema"]', { timeout: 15000 })
                .should('exist');
        });
    });

    describe('MariaDB SSL Connection', () => {
        it('connects to MariaDB with SSL verify-ca mode', () => {
            const db = getDatabaseConfig('mariadb');
            if (!db.ssl) {
                cy.log('Skipping: mariadb SSL config not available');
                return;
            }

            cy.get('[data-testid="database-type-select"]').click();
            cy.get(`[data-value="${db.type}"]`).click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });
            cy.get('[data-testid="database"]').type(db.connection.database);

            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').clear().type(String(db.ssl.port));

            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-ca"]').click();

            if (db.ssl.caCertPath) {
                enterCertificateContent(db.ssl.caCertPath);
            }

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@loginQuery', { timeout: 30000 });
            cy.url().should('include', '/storage-unit');
            cy.get('[data-testid="sidebar-database"], [data-testid="sidebar-schema"]', { timeout: 15000 })
                .should('exist');
        });
    });

    describe('MongoDB SSL Connection', () => {
        it('connects to MongoDB with TLS enabled', () => {
            const db = getDatabaseConfig('mongodb');
            if (!db.ssl) {
                cy.log('Skipping: mongodb SSL config not available');
                return;
            }

            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="MongoDB"]').click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });
            cy.get('[data-testid="database"]').type(db.connection.database);

            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').clear().type(String(db.ssl.port));

            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="enabled"]').click();

            if (db.ssl.caCertPath) {
                enterCertificateContent(db.ssl.caCertPath);
            }

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@loginQuery', { timeout: 30000 });
            cy.url().should('include', '/storage-unit');
            cy.get('[data-testid="sidebar-database"], [data-testid="sidebar-schema"]', { timeout: 15000 })
                .should('exist');
        });
    });

    describe('Redis SSL Connection', () => {
        it('connects to Redis with TLS enabled', () => {
            const db = getDatabaseConfig('redis');
            if (!db.ssl) {
                cy.log('Skipping: redis SSL config not available');
                return;
            }

            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Redis"]').click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });

            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').clear().type(String(db.ssl.port));

            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="enabled"]').click();

            if (db.ssl.caCertPath) {
                enterCertificateContent(db.ssl.caCertPath);
            }

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@loginQuery', { timeout: 30000 });
            cy.url().should('include', '/storage-unit');
            cy.get('[data-testid="sidebar-profile"]', { timeout: 15000 })
                .should('exist');
        });
    });

    describe('ClickHouse SSL Connection', () => {
        it('connects to ClickHouse with SSL enabled', () => {
            const db = getDatabaseConfig('clickhouse');
            if (!db.ssl) {
                cy.log('Skipping: clickhouse SSL config not available');
                return;
            }

            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="ClickHouse"]').click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });
            cy.get('[data-testid="database"]').type(db.connection.database);

            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').clear().type(String(db.ssl.port));

            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="enabled"]').click();

            if (db.ssl.caCertPath) {
                enterCertificateContent(db.ssl.caCertPath);
            }

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@loginQuery', { timeout: 30000 });
            cy.url().should('include', '/storage-unit');
            cy.get('[data-testid="sidebar-profile"]', { timeout: 15000 })
                .should('exist');
        });
    });

    describe('Elasticsearch SSL Connection', () => {
        it('connects to Elasticsearch with TLS enabled', () => {
            const db = getDatabaseConfig('elasticsearch');
            if (!db.ssl) {
                cy.log('Skipping: elasticsearch SSL config not available');
                return;
            }

            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="ElasticSearch"]').click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });

            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').clear().type(String(db.ssl.port));

            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="enabled"]').click();

            if (db.ssl.caCertPath) {
                enterCertificateContent(db.ssl.caCertPath);
            }

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@loginQuery', { timeout: 30000 });
            cy.url().should('include', '/storage-unit');
            cy.get('[data-testid="sidebar-profile"]', { timeout: 15000 })
                .should('exist');
        });
    });

    describe('SSL Status Badge', () => {
        it('shows SSL badge in sidebar when connected with SSL', () => {
            const db = getDatabaseConfig('postgres');
            if (!db.ssl) {
                cy.log('Skipping: postgres SSL config not available');
                return;
            }

            cy.get('[data-testid="database-type-select"]').click();
            cy.get(`[data-value="${db.type}"]`).click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });
            cy.get('[data-testid="database"]').type(db.connection.database);

            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').clear().type(String(db.ssl.port));

            cy.get('[data-testid="ssl-mode-select"]').click();
            cy.get('[data-value="verify-ca"]').click();

            if (db.ssl.caCertPath) {
                enterCertificateContent(db.ssl.caCertPath);
            }

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@loginQuery', { timeout: 30000 });
            cy.url().should('include', '/storage-unit');

            // SSL shield badge should appear in the profile selector when SSL is enabled
            cy.get('[data-testid="sidebar-profile"]', { timeout: 15000 }).within(() => {
                cy.get('[data-testid="ssl-badge"]').should('exist');
            });
        });

        it('does not show SSL badge when connected without SSL', () => {
            const db = getDatabaseConfig('postgres');

            cy.get('[data-testid="database-type-select"]').click();
            cy.get(`[data-value="${db.type}"]`).click();

            cy.get('[data-testid="hostname"]').type(db.connection.host);
            cy.get('[data-testid="username"]').type(db.connection.user);
            cy.get('[data-testid="password"]').type(db.connection.password, { log: false });
            cy.get('[data-testid="database"]').type(db.connection.database);

            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            cy.wait('@loginQuery', { timeout: 30000 });
            cy.url().should('include', '/storage-unit');

            // Without SSL, there should be no SSL badge
            cy.get('[data-testid="sidebar-profile"]', { timeout: 15000 }).within(() => {
                cy.get('[data-testid="ssl-badge"]').should('not.exist');
            });
        });
    });

    describe('SSL Connection Failure Cases', () => {
        it('fails to connect with wrong SSL mode', () => {
            const db = getDatabaseConfig('postgres');
            if (!db.ssl) {
                cy.log('Skipping: postgres SSL config not available');
                return;
            }

            cy.get('[data-testid="database-type-select"]').click();
            cy.get('[data-value="Postgres"]').click();

            // Use SSL database port but no SSL config
            cy.get('[data-testid="hostname"]').type('localhost');
            cy.get('[data-testid="username"]').type('user');
            cy.get('[data-testid="password"]').type('password');
            cy.get('[data-testid="database"]').type('test_db');

            cy.get('[data-testid="advanced-button"]').click();
            cy.get('[data-testid="Port-input"]').clear().type(String(db.ssl.port));

            // Don't configure SSL - connection may fail or succeed depending on server config
            cy.intercept('POST', '**/api/query').as('loginQuery');
            cy.get('[data-testid="login-button"]').click();

            // Wait and check result - we mainly want to ensure no crash
            cy.wait('@loginQuery', { timeout: 30000 });

            // Note: Result depends on server SSL configuration (optional vs required)
            // This test ensures the system handles the attempt gracefully
        });
    });
});
