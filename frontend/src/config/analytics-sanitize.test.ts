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

import {
    sanitizeAnalyticsIdentityProperties,
    sanitizeAnalyticsProperties,
    type AnalyticsRuntimeContext,
} from './analytics-sanitize';

const context: AnalyticsRuntimeContext = {
    buildEdition: 'ee',
    buildEnvironment: 'production',
    appType: 'web',
    platform: 'browser',
    deployment: 'test-deployment',
};

const assert = (condition: boolean, message: string) => {
    if (!condition) {
        throw new Error(message);
    }
};

const assertEqual = (actual: unknown, expected: unknown, message: string) => {
    assert(Object.is(actual, expected), `${message}: expected ${String(expected)}, got ${String(actual)}`);
};

const assertMissing = (object: Record<string, unknown>, key: string) => {
    assert(!(key in object), `expected ${key} to be removed`);
};

const run = () => {
    const eventProperties = sanitizeAnalyticsProperties({
        database_type: 'postgres',
        input_length_bucket: '32_127',
        sql: 'select * from users',
        query: 'select * from users',
        prompt: 'summarize this table',
        token: 'secret-token',
        api_key: 'secret-api-key',
        email: 'person@example.com',
        username: 'admin',
        path: '/Users/person/private.db',
        url: 'https://example.com/private',
        password: 'password',
        secret: 'secret',
        unsafe_object: { nested: true },
    }, context);

    assertEqual(eventProperties.build_edition, 'ee', 'build edition is attached');
    assertEqual(eventProperties.build_environment, 'production', 'build environment is attached');
    assertEqual(eventProperties.app_type, 'web', 'app type is attached');
    assertEqual(eventProperties.platform, 'browser', 'platform is attached');
    assertEqual(eventProperties.deployment, 'test-deployment', 'deployment is attached');
    assertEqual(eventProperties.database_type, 'postgres', 'safe property is retained');
    assertEqual(eventProperties.input_length_bucket, '32_127', 'safe bucket property is retained');

    for (const key of ['sql', 'query', 'prompt', 'token', 'api_key', 'email', 'username', 'path', 'url', 'password', 'secret', 'unsafe_object']) {
        assertMissing(eventProperties, key);
    }

    const identityProperties = sanitizeAnalyticsIdentityProperties({
        is_super_admin: true,
        partner_cohort: 'alpha',
        org_id: 'org-123',
        project_id: 'project-123',
        email: 'person@example.com',
        name: 'Acme',
    });

    assertEqual(identityProperties.is_super_admin, true, 'safe identity property is retained');
    assertEqual(identityProperties.partner_cohort, 'alpha', 'safe cohort property is retained');
    for (const key of ['org_id', 'project_id', 'email', 'name']) {
        assertMissing(identityProperties, key);
    }
};

run();
