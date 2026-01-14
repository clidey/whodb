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

import { forEachDatabase } from '../../support/test-runner';

/**
 * SSL Mode Tests
 *
 * Tests SSL modes for each database type against SSL-enabled containers.
 * Requires SSL containers: docker-compose --profile ssl up
 *
 * SSL configuration is loaded from database fixtures under the "ssl" key.
 */

/**
 * Maps container certificate paths to host filesystem paths.
 */
function containerPathToHostPath(containerPath) {
    if (!containerPath) return null;
    return containerPath.replace('/app/certs/', '../dev/certs/');
}

/**
 * Verify SSL badge status in sidebar
 */
function verifySSLStatus(mode) {
    cy.get('[data-testid="sidebar-profile"]', { timeout: 15000 }).within(() => {
        if (mode === 'disabled') {
            cy.get('[data-testid="ssl-badge"]').should('not.exist');
        } else {
            cy.get('[data-testid="ssl-badge"]').should('exist');
        }
    });
}

describe('SSL Modes', () => {
    forEachDatabase('all', (db) => {
        // Skip databases without SSL config
        if (!db.ssl || !db.ssl.modes) {
            return;
        }

        const ssl = db.ssl;
        const conn = db.connection;

        db.ssl.modes
            .filter(({ shouldSucceed }) => shouldSucceed)
            .forEach(({ mode, needsCert, description }) => {
                it(`${mode}: ${description}`, () => {
                    if (needsCert && ssl.caCertPath) {
                        const hostPath = containerPathToHostPath(ssl.caCertPath);
                        cy.readFile(hostPath).then((certContent) => {
                            cy.login(
                                db.type,
                                conn.host,
                                conn.user,
                                conn.password,
                                conn.database,
                                {
                                    Port: String(ssl.port),
                                    ssl: { mode, caCertContent: certContent }
                                }
                            );
                            verifySSLStatus(mode);
                        });
                    } else {
                        cy.login(
                            db.type,
                            conn.host,
                            conn.user,
                            conn.password,
                            conn.database,
                            {
                                Port: String(ssl.port),
                                ssl: { mode }
                            }
                        );
                        verifySSLStatus(mode);
                    }
                });
            });
    }, { login: false });
});
