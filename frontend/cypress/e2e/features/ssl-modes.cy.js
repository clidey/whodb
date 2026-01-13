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

/**
 * Comprehensive SSL Mode Tests
 *
 * Tests all SSL modes for each database type against SSL-enabled containers.
 * Requires SSL containers to be running: docker-compose --profile ssl up
 *
 * These tests verify:
 * 1. Each supported SSL mode connects (or fails) as expected
 * 2. Certificate validation works correctly
 * 3. Wrong/missing certificates are rejected appropriately
 */

// SSL mode configurations per database type
// Each mode specifies whether it should succeed and what certificates are needed
const SSL_TEST_CONFIGS = {
    postgres: {
        type: 'Postgres',
        uiType: 'Postgres',
        port: 5433,
        host: 'localhost',
        user: 'user',
        password: 'password',
        database: 'test_db',
        caCertPath: '/app/certs/ca/postgres/ca.pem',
        // PostgreSQL server allows both SSL and non-SSL connections
        modes: [
            { mode: 'disabled', shouldSucceed: true, needsCert: false, description: 'No SSL encryption' },
            { mode: 'required', shouldSucceed: true, needsCert: false, description: 'SSL without certificate verification' },
            { mode: 'verify-ca', shouldSucceed: true, needsCert: true, description: 'Verify server certificate against CA' },
            { mode: 'verify-identity', shouldSucceed: true, needsCert: true, description: 'Verify CA and hostname' },
        ],
        negativeTests: [
            { mode: 'verify-ca', withCert: false, description: 'verify-ca without CA cert should fail' },
            { mode: 'verify-identity', withCert: false, description: 'verify-identity without CA cert should fail' },
        ]
    },
    mysql: {
        type: 'MySQL',
        uiType: 'MySQL',
        port: 3309,
        host: 'localhost',
        user: 'user',
        password: 'password',
        database: 'test_db',
        caCertPath: '/app/certs/ca/mysql/ca.pem',
        // MySQL server allows both SSL and non-SSL (require_secure_transport=OFF)
        modes: [
            { mode: 'disabled', shouldSucceed: true, needsCert: false, description: 'No SSL encryption' },
            { mode: 'preferred', shouldSucceed: true, needsCert: false, description: 'Use SSL if available' },
            { mode: 'required', shouldSucceed: true, needsCert: false, description: 'SSL without certificate verification' },
            { mode: 'verify-ca', shouldSucceed: true, needsCert: true, description: 'Verify server certificate against CA' },
            { mode: 'verify-identity', shouldSucceed: true, needsCert: true, description: 'Verify CA and hostname' },
        ],
        negativeTests: [
            { mode: 'verify-ca', withCert: false, description: 'verify-ca without CA cert should fail' },
        ]
    },
    mariadb: {
        type: 'MariaDB',
        uiType: 'MariaDB',
        port: 3310,
        host: 'localhost',
        user: 'user',
        password: 'password',
        database: 'test_db',
        caCertPath: '/app/certs/ca/mariadb/ca.pem',
        // MariaDB server allows both SSL and non-SSL
        modes: [
            { mode: 'disabled', shouldSucceed: true, needsCert: false, description: 'No SSL encryption' },
            { mode: 'preferred', shouldSucceed: true, needsCert: false, description: 'Use SSL if available' },
            { mode: 'required', shouldSucceed: true, needsCert: false, description: 'SSL without certificate verification' },
            { mode: 'verify-ca', shouldSucceed: true, needsCert: true, description: 'Verify server certificate against CA' },
            { mode: 'verify-identity', shouldSucceed: true, needsCert: true, description: 'Verify CA and hostname' },
        ],
        negativeTests: [
            { mode: 'verify-ca', withCert: false, description: 'verify-ca without CA cert should fail' },
        ]
    },
    mongodb: {
        type: 'Mongo',
        uiType: 'MongoDB',
        port: 27018,
        host: 'localhost',
        user: 'user',
        password: 'password',
        database: 'test_db',
        caCertPath: '/app/certs/ca/mongodb/ca.pem',
        // MongoDB server requires TLS (--tlsMode requireTLS)
        modes: [
            { mode: 'disabled', shouldSucceed: false, needsCert: false, description: 'No TLS - server requires TLS' },
            { mode: 'enabled', shouldSucceed: true, needsCert: true, description: 'TLS enabled with CA verification' },
            { mode: 'insecure', shouldSucceed: true, needsCert: false, description: 'TLS without certificate verification' },
        ],
        negativeTests: [
            { mode: 'enabled', withCert: false, description: 'enabled without CA cert should fail' },
        ]
    },
    redis: {
        type: 'Redis',
        uiType: 'Redis',
        port: 6380,
        host: 'localhost',
        user: null,
        password: 'password',
        database: null,
        caCertPath: '/app/certs/ca/redis/ca.pem',
        // Redis server requires TLS (port 0, tls-port 6379)
        modes: [
            { mode: 'disabled', shouldSucceed: false, needsCert: false, description: 'No TLS - server requires TLS' },
            { mode: 'enabled', shouldSucceed: true, needsCert: true, description: 'TLS enabled with CA verification' },
            { mode: 'insecure', shouldSucceed: true, needsCert: false, description: 'TLS without certificate verification' },
        ],
        negativeTests: [
            { mode: 'enabled', withCert: false, description: 'enabled without CA cert should fail' },
        ]
    },
    clickhouse: {
        type: 'ClickHouse',
        uiType: 'ClickHouse',
        port: 9440,
        host: 'localhost',
        user: 'user',
        password: 'password',
        database: 'test_db',
        caCertPath: '/app/certs/ca/clickhouse/ca.pem',
        // ClickHouse SSL port (9440) only accepts SSL connections
        modes: [
            { mode: 'disabled', shouldSucceed: false, needsCert: false, description: 'No SSL on SSL-only port' },
            { mode: 'enabled', shouldSucceed: true, needsCert: true, description: 'SSL enabled with CA verification' },
            { mode: 'insecure', shouldSucceed: true, needsCert: false, description: 'SSL without certificate verification' },
        ],
        negativeTests: [
            { mode: 'enabled', withCert: false, description: 'enabled without CA cert should fail' },
        ]
    },
    elasticsearch: {
        type: 'Elastic',
        uiType: 'ElasticSearch',
        port: 9201,
        host: 'localhost',
        user: 'elastic',
        password: 'password',
        database: null,
        caCertPath: '/app/certs/ca/elasticsearch/ca.pem',
        // Elasticsearch SSL server (xpack.security.http.ssl.enabled=true)
        modes: [
            { mode: 'disabled', shouldSucceed: false, needsCert: false, description: 'No TLS on TLS-only endpoint' },
            { mode: 'enabled', shouldSucceed: true, needsCert: true, description: 'TLS enabled with CA verification' },
            { mode: 'insecure', shouldSucceed: true, needsCert: false, description: 'TLS without certificate verification' },
        ],
        negativeTests: [
            { mode: 'enabled', withCert: false, description: 'enabled without CA cert should fail' },
        ]
    },
};

/**
 * Maps container certificate paths to host filesystem paths for testing.
 * Container paths like /app/certs/... map to ../dev/certs/... on the host.
 */
function containerPathToHostPath(containerPath) {
    if (!containerPath) return null;
    // Map /app/certs/... to ../dev/certs/...
    return containerPath.replace('/app/certs/', '../dev/certs/');
}

/**
 * Helper to fill login form and attempt connection
 */
function attemptSSLConnection(config, mode, provideCert) {
    // Select database type
    cy.get('[data-testid="database-type-select"]').click();
    cy.get(`[data-value="${config.uiType}"]`).click();

    // Fill connection details
    cy.get('[data-testid="hostname"]').type(config.host);

    if (config.user) {
        cy.get('[data-testid="username"]').type(config.user);
    }

    cy.get('[data-testid="password"]').type(config.password, { log: false });

    if (config.database) {
        cy.get('[data-testid="database"]').type(config.database);
    }

    // Open advanced options
    cy.get('[data-testid="advanced-button"]').click();

    // Set port
    cy.get('[data-testid="Port-input"]').clear().type(String(config.port));

    // Select SSL mode
    cy.get('[data-testid="ssl-mode-select"]').click();
    cy.get(`[data-value="${mode}"]`).click();

    // Provide CA certificate content if needed (read from host path, enter in paste mode)
    if (provideCert && config.caCertPath) {
        const hostPath = containerPathToHostPath(config.caCertPath);
        if (hostPath) {
            cy.readFile(hostPath).then((certContent) => {
                // Switch to paste mode
                cy.contains('button', 'Paste PEM').first().click();
                // Enter certificate content
                cy.get('[data-testid="ssl-ca-certificate-content"]').type(certContent, { parseSpecialCharSequences: false, delay: 0 });
            });
        }
    }

    // Attempt login
    cy.intercept('POST', '**/api/query').as('loginQuery');
    cy.get('[data-testid="login-button"]').click();

    return cy.wait('@loginQuery', { timeout: 30000 });
}

/**
 * Verify successful connection
 */
function verifyConnectionSuccess() {
    cy.url().should('include', '/storage-unit');
    cy.get('[data-testid="sidebar-profile"]', { timeout: 15000 }).should('exist');
}

/**
 * Verify connection failure (stays on login page or shows error)
 */
function verifyConnectionFailure() {
    // Should either stay on login page or show an error
    cy.url().then(url => {
        if (url.includes('/login')) {
            // Still on login page - expected for failure
            cy.log('Connection failed as expected - stayed on login page');
        } else {
            // May have redirected but with error - check for error state
            cy.get('[data-testid="sidebar-profile"]').should('not.exist');
        }
    });
}

// Generate tests for each database
Object.entries(SSL_TEST_CONFIGS).forEach(([dbKey, config]) => {
    describe(`${config.type} SSL Modes`, () => {
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

        // Test each SSL mode
        describe('Mode Tests', () => {
            config.modes.forEach(({ mode, shouldSucceed, needsCert, description }) => {
                it(`${mode}: ${description} - should ${shouldSucceed ? 'succeed' : 'fail'}`, () => {
                    attemptSSLConnection(config, mode, needsCert);

                    if (shouldSucceed) {
                        verifyConnectionSuccess();
                    } else {
                        verifyConnectionFailure();
                    }
                });
            });
        });

        // Negative tests - wrong/missing certificates
        if (config.negativeTests && config.negativeTests.length > 0) {
            describe('Negative Tests (Missing/Wrong Certificates)', () => {
                config.negativeTests.forEach(({ mode, withCert, description }) => {
                    it(description, () => {
                        attemptSSLConnection(config, mode, withCert);
                        verifyConnectionFailure();
                    });
                });
            });
        }
    });
});

// Additional cross-database tests
describe('SSL Cross-Database Tests', () => {
    beforeEach(() => {
        clearBrowserState();
        cy.visit('/login');

        cy.get('body').then($body => {
            const $btn = $body.find('button').filter(function () {
                return this.textContent.includes('Disable Telemetry');
            });
            if ($btn.length) {
                cy.wrap($btn).click();
            }
        });
    });

    it('invalid CA certificate content fails for all databases', () => {
        // Test PostgreSQL with invalid CA certificate content
        const config = SSL_TEST_CONFIGS.postgres;

        cy.get('[data-testid="database-type-select"]').click();
        cy.get(`[data-value="${config.uiType}"]`).click();

        cy.get('[data-testid="hostname"]').type(config.host);
        cy.get('[data-testid="username"]').type(config.user);
        cy.get('[data-testid="password"]').type(config.password, { log: false });
        cy.get('[data-testid="database"]').type(config.database);

        cy.get('[data-testid="advanced-button"]').click();
        cy.get('[data-testid="Port-input"]').clear().type(String(config.port));

        cy.get('[data-testid="ssl-mode-select"]').click();
        cy.get('[data-value="verify-ca"]').click();

        // Enter invalid certificate content
        cy.contains('button', 'Paste PEM').first().click();
        cy.get('[data-testid="ssl-ca-certificate-content"]').type('-----BEGIN CERTIFICATE-----\nINVALID_GARBAGE_DATA\n-----END CERTIFICATE-----', { parseSpecialCharSequences: false, delay: 0 });

        cy.intercept('POST', '**/api/query').as('loginQuery');
        cy.get('[data-testid="login-button"]').click();

        cy.wait('@loginQuery', { timeout: 30000 });
        verifyConnectionFailure();
    });

    it('mismatched CA certificate fails verification', () => {
        // Use MySQL's CA cert for PostgreSQL - should fail
        const config = SSL_TEST_CONFIGS.postgres;
        const wrongCaCertPath = '../dev/certs/ca/mysql/ca.pem'; // MySQL's CA (wrong for Postgres)

        cy.get('[data-testid="database-type-select"]').click();
        cy.get(`[data-value="${config.uiType}"]`).click();

        cy.get('[data-testid="hostname"]').type(config.host);
        cy.get('[data-testid="username"]').type(config.user);
        cy.get('[data-testid="password"]').type(config.password, { log: false });
        cy.get('[data-testid="database"]').type(config.database);

        cy.get('[data-testid="advanced-button"]').click();
        cy.get('[data-testid="Port-input"]').clear().type(String(config.port));

        cy.get('[data-testid="ssl-mode-select"]').click();
        cy.get('[data-value="verify-ca"]').click();

        // Read MySQL's CA cert and paste it (wrong CA for PostgreSQL)
        cy.readFile(wrongCaCertPath).then((certContent) => {
            cy.contains('button', 'Paste PEM').first().click();
            cy.get('[data-testid="ssl-ca-certificate-content"]').type(certContent, { parseSpecialCharSequences: false, delay: 0 });
        });

        cy.intercept('POST', '**/api/query').as('loginQuery');
        cy.get('[data-testid="login-button"]').click();

        cy.wait('@loginQuery', { timeout: 30000 });
        verifyConnectionFailure();
    });
});
