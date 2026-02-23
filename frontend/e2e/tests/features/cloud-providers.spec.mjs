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

import { test, expect, forEachDatabase } from '../../support/test-fixture.mjs';
import { TIMEOUT } from '../../support/helpers/test-utils.mjs';

test.describe('Cloud Providers', () => {

    // Cloud providers section is global UI — only test with Postgres
    forEachDatabase('sql', (db) => {
        if (db.type !== 'Postgres') {
            return;
        }

        /**
         * Helper to mock SettingsConfig with cloud providers enabled/disabled.
         * Also mocks GetCloudProviders to return the given list.
         */
        async function mockCloudProviders(page, { enabled, providers = [] }) {
            await page.route('**/api/query', async (route) => {
                let postData;
                try {
                    postData = route.request().postDataJSON();
                } catch {
                    return route.fallback();
                }

                const op = postData?.operationName;

                if (op === 'SettingsConfig') {
                    return route.fulfill({
                        contentType: 'application/json',
                        body: JSON.stringify({
                            data: {
                                SettingsConfig: {
                                    MetricsEnabled: true,
                                    CloudProvidersEnabled: enabled,
                                    DisableCredentialForm: false,
                                    MaxPageSize: 10000,
                                    __typename: 'SettingsConfig',
                                },
                            },
                        }),
                    });
                }

                if (op === 'GetCloudProviders') {
                    return route.fulfill({
                        contentType: 'application/json',
                        body: JSON.stringify({
                            data: {
                                CloudProviders: providers,
                            },
                        }),
                    });
                }

                if (op === 'RemoveCloudProvider') {
                    return route.fulfill({
                        contentType: 'application/json',
                        body: JSON.stringify({
                            data: {
                                RemoveCloudProvider: {
                                    Status: true,
                                    __typename: 'StatusResponse',
                                },
                            },
                        }),
                    });
                }

                await route.fallback();
            });
        }

        test.describe('Visibility', () => {
            test('shows AWS providers section when enabled', async ({ whodb, page }) => {
                await mockCloudProviders(page, { enabled: true, providers: [] });
                await whodb.goto('settings');

                await expect(page.locator('[data-testid="add-aws-provider"]')).toBeVisible({ timeout: TIMEOUT.ACTION });
            });

            test('hides AWS providers section when disabled', async ({ whodb, page }) => {
                await mockCloudProviders(page, { enabled: false });
                await whodb.goto('settings');

                // Wait for the page to load fully
                await expect(page.locator('#font-size')).toBeVisible({ timeout: TIMEOUT.ACTION });

                // The add-aws-provider button should not be present
                await expect(page.locator('[data-testid="add-aws-provider"]')).not.toBeAttached();
            });

            test('displays existing provider cards', async ({ whodb, page }) => {
                const testProvider = {
                    Id: 'test-provider-1',
                    Name: 'My AWS Dev',
                    Region: 'us-east-1',
                    AuthMethod: 'access_key',
                    Status: 'Connected',
                    DiscoveredCount: 5,
                    IsEnvironmentDefined: false,
                    LastDiscoveryAt: null,
                    __typename: 'CloudProvider',
                };

                await mockCloudProviders(page, { enabled: true, providers: [testProvider] });
                await whodb.goto('settings');

                // Provider card should be visible with the provider name
                const providerCard = page.locator('[data-testid="aws-provider-test-provider-1"]');
                await expect(providerCard).toBeVisible({ timeout: TIMEOUT.ACTION });
                await expect(providerCard).toContainText('My AWS Dev');
                await expect(providerCard).toContainText('us-east-1');
            });
        });

        test.describe('Management', () => {
            test('removes AWS provider', async ({ whodb, page }) => {
                const testProvider = {
                    Id: 'remove-me',
                    Name: 'Remove Test',
                    Region: 'eu-west-1',
                    AuthMethod: 'access_key',
                    Status: 'Connected',
                    DiscoveredCount: 0,
                    IsEnvironmentDefined: false,
                    LastDiscoveryAt: null,
                    __typename: 'CloudProvider',
                };

                await mockCloudProviders(page, { enabled: true, providers: [testProvider] });
                await whodb.goto('settings');

                // Verify provider is visible
                const providerCard = page.locator('[data-testid="aws-provider-remove-me"]');
                await expect(providerCard).toBeVisible({ timeout: TIMEOUT.ACTION });

                // Click the remove button
                await page.locator('[data-testid="remove-remove-me"]').click();

                // Provider card should disappear (Redux dispatch removes it immediately)
                await expect(providerCard).not.toBeAttached({ timeout: TIMEOUT.ACTION });
            });
        });
    });

});
