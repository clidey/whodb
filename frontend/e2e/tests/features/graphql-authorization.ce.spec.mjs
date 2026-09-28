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

import { test, expect } from '../../support/test-fixture.mjs';

async function graphQL(request, operationName, query, variables = {}) {
    const response = await request.post('/api/query', {
        data: { operationName, query, variables },
    });
    expect(response.ok()).toBe(true);
    return response.json();
}

test('authorizes CE by selected fields rather than operation name', async ({ browser, baseURL }) => {
    const anonymous = await browser.newContext({ baseURL });
    try {
        const before = await graphQL(
            anonymous.request,
            'SettingsConfig',
            'query SettingsConfig { SettingsConfig { MetricsEnabled } }',
        );
        expect(before.errors).toBeUndefined();
        const originalValue = before.data.SettingsConfig.MetricsEnabled;
        const attemptedValue = originalValue === true ? 'false' : 'true';

        const spoofed = await graphQL(
            anonymous.request,
            'SettingsConfig',
            `mutation SettingsConfig($value: String) {
                UpdateSettings(newSettings: { MetricsEnabled: $value }) { Status }
            }`,
            { value: attemptedValue },
        );
        expect(spoofed.errors?.[0]?.extensions?.code).toBe('UNAUTHENTICATED');

        const after = await graphQL(
            anonymous.request,
            'SettingsConfig',
            'query SettingsConfig { SettingsConfig { MetricsEnabled } }',
        );
        expect(after.errors).toBeUndefined();
        expect(after.data.SettingsConfig.MetricsEnabled).toBe(originalValue);
    } finally {
        await anonymous.close();
    }
});
