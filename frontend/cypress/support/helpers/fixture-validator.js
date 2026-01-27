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

/**
 * Fixture Schema Validator for Cypress E2E Tests.
 * Validates that database fixtures conform to the expected schema.
 */

/**
 * Required fields for all database fixtures
 */
const REQUIRED_FIELDS = [
    'type',
    'category',
    'connection',
    'features'
];

/**
 * Required fields in testTable config
 */
const TEST_TABLE_REQUIRED_FIELDS = {
    sql: ['name', 'identifierField', 'identifierColIndex', 'testValues'],
    document: ['name', 'identifierField', 'testValues'],
    keyvalue: ['name', 'identifierField', 'testValues']
};

/**
 * Required fields in testValues
 */
const TEST_VALUES_REQUIRED_FIELDS = ['original', 'modified', 'rowIndex'];

/**
 * Required feature flags - must be present in every fixture
 */
const REQUIRED_FEATURES = [
    'graph',
    'export',
    'scratchpad',
    'mockData',
    'chat',
    'whereConditions',
    'queryHistory'
];

/**
 * All valid feature names - used to catch typos in fixtures and test files
 * Optional features don't need to be present in fixtures, but if present must be boolean
 */
export const VALID_FEATURES = [
    ...REQUIRED_FEATURES,
    'crud',                // Optional: databases with async mutations may disable (e.g., ClickHouse)
    'scratchpadUpdate',    // Optional: databases that don't support UPDATE via scratchpad
    'multiConditionFilter', // Optional: databases that don't support multiple WHERE conditions
    'typeCasting',         // Optional: databases that support numeric type casting
    'sslConnection'        // Optional: databases configured with SSL/TLS
];

/**
 * Validates a single database fixture against the schema.
 * @param {Object} fixture - The fixture object to validate
 * @param {string} name - The name of the fixture file (for error messages)
 * @returns {Object} - { valid: boolean, errors: string[] }
 */
export function validateFixture(fixture, name) {
    const errors = [];

    // Check required top-level fields
    for (const field of REQUIRED_FIELDS) {
        if (fixture[field] === undefined) {
            errors.push(`Missing required field: ${field}`);
        }
    }

    // Validate category
    const validCategories = ['sql', 'document', 'keyvalue'];
    if (fixture.category && !validCategories.includes(fixture.category)) {
        errors.push(`Invalid category: ${fixture.category}. Must be one of: ${validCategories.join(', ')}`);
    }

    // Validate features object
    if (fixture.features) {
        // Check required features are present
        for (const feature of REQUIRED_FEATURES) {
            if (typeof fixture.features[feature] !== 'boolean') {
                errors.push(`Missing or invalid feature flag: ${feature} (must be boolean)`);
            }
        }

        // Check for unknown feature names (typos)
        for (const feature of Object.keys(fixture.features)) {
            if (!VALID_FEATURES.includes(feature)) {
                errors.push(`Unknown feature: '${feature}'. Valid features: ${VALID_FEATURES.join(', ')}`);
            }
        }
    }

    // Validate testTable if present
    if (fixture.testTable) {
        const category = fixture.category;
        const requiredTestTableFields = TEST_TABLE_REQUIRED_FIELDS[category] || [];

        for (const field of requiredTestTableFields) {
            if (fixture.testTable[field] === undefined) {
                errors.push(`testTable missing required field: ${field}`);
            }
        }

        // Validate testValues
        if (fixture.testTable.testValues) {
            for (const field of TEST_VALUES_REQUIRED_FIELDS) {
                if (fixture.testTable.testValues[field] === undefined) {
                    errors.push(`testTable.testValues missing required field: ${field}`);
                }
            }
        }
    } else {
        errors.push('Missing testTable config - required for feature-focused tests');
    }

    // Validate connection object
    if (fixture.connection) {
        // At minimum, connection should exist (fields vary by database)
        if (typeof fixture.connection !== 'object') {
            errors.push('connection must be an object');
        }
    }

    // Validate featureNotes for disabled features
    if (fixture.features && fixture.featureNotes) {
        for (const [feature, enabled] of Object.entries(fixture.features)) {
            if (!enabled && !fixture.featureNotes[feature]) {
                // This is a warning, not an error - disabled features should have explanations
                errors.push(`Warning: Feature '${feature}' is disabled but has no explanation in featureNotes`);
            }
        }
    }

    return {
        valid: errors.filter(e => !e.startsWith('Warning:')).length === 0,
        errors,
        warnings: errors.filter(e => e.startsWith('Warning:'))
    };
}

/**
 * Validates all fixtures and logs results.
 * @param {Object} fixtures - Map of fixture name to fixture object
 * @returns {Object} - { allValid: boolean, results: Object }
 */
export function validateAllFixtures(fixtures) {
    const results = {};
    let allValid = true;

    for (const [name, fixture] of Object.entries(fixtures)) {
        const result = validateFixture(fixture, name);
        results[name] = result;

        if (!result.valid) {
            allValid = false;
            console.error(`❌ Fixture validation failed for ${name}:`);
            result.errors.forEach(err => console.error(`   - ${err}`));
        } else if (result.warnings.length > 0) {
            console.warn(`⚠️  Fixture ${name} has warnings:`);
            result.warnings.forEach(warn => console.warn(`   - ${warn}`));
        }
    }

    if (allValid) {
        console.log('✅ All fixtures validated successfully');
    }

    return { allValid, results };
}

/**
 * Runs fixture validation and throws if any fixtures are invalid.
 * Call this in your test setup to fail fast on schema violations.
 * @param {Object} fixtures - Map of fixture name to fixture object
 */
export function assertFixturesValid(fixtures) {
    const { allValid, results } = validateAllFixtures(fixtures);

    if (!allValid) {
        const failedFixtures = Object.entries(results)
            .filter(([_, r]) => !r.valid)
            .map(([name, r]) => `${name}: ${r.errors.filter(e => !e.startsWith('Warning:')).join(', ')}`)
            .join('\n');

        throw new Error(`Fixture validation failed:\n${failedFixtures}`);
    }
}

export default {
    validateFixture,
    validateAllFixtures,
    assertFixturesValid,
    REQUIRED_FIELDS,
    REQUIRED_FEATURES,
    VALID_FEATURES
};
