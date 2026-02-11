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

import { readFileSync } from 'fs';
import { test, expect, forEachDatabase } from '../../support/test-fixture.mjs';

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
async function verifySSLStatus(page, mode) {
    const sidebarProfile = page.locator('[data-testid="sidebar-profile"]');
    await sidebarProfile.waitFor({ timeout: 15000 });
    if (mode === 'disabled') {
        await expect(sidebarProfile.locator('[data-testid="ssl-badge"]')).not.toBeAttached();
    } else {
        await expect(sidebarProfile.locator('[data-testid="ssl-badge"]')).toBeAttached();
    }
}

test.describe('SSL Modes', () => {
    forEachDatabase('all', (db) => {
        // Skip databases without SSL config
        if (!db.ssl || !db.ssl.modes) {
            return;
        }

        const ssl = db.ssl;
        const conn = db.connection;

        // Use SSL-specific credentials if provided, otherwise fall back to connection credentials
        // Pass undefined instead of null to skip fields that don't exist (Redis has no user/database)
        const sslUser = (ssl.user ?? conn.user) ?? undefined;
        const sslPassword = ssl.password ?? conn.password;
        const database = conn.database ?? undefined;

        db.ssl.modes
            .filter(({ shouldSucceed }) => shouldSucceed)
            .forEach(({ mode, needsCert, description }) => {
                test(`${mode}: ${description}`, async ({ whodb, page }) => {
                    if (needsCert && ssl.caCertPath) {
                        const hostPath = containerPathToHostPath(ssl.caCertPath);
                        let certContent = undefined;
                        if (hostPath) {
                            try {
                                certContent = readFileSync(hostPath, 'utf-8');
                            } catch {
                                // cert file not available
                            }
                        }
                        await whodb.login(
                            db.type,
                            conn.host,
                            sslUser,
                            sslPassword,
                            database,
                            {
                                Port: String(ssl.port),
                                ssl: { mode, ...(certContent ? { caCertContent: certContent } : {}) }
                            }
                        );
                        await verifySSLStatus(page, mode);
                    } else {
                        await whodb.login(
                            db.type,
                            conn.host,
                            sslUser,
                            sslPassword,
                            database,
                            {
                                Port: String(ssl.port),
                                ssl: { mode }
                            }
                        );
                        await verifySSLStatus(page, mode);
                    }
                });
            });
    }, { login: false });
});
