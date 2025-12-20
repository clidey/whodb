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

import {VALID_FEATURES, validateAllFixtures} from './helpers/fixture-validator';

// CE Database configurations - loaded from fixtures
const ceDatabaseConfigs = {
    postgres: require('../fixtures/databases/postgres.json'),
    mysql: require('../fixtures/databases/mysql.json'),
    mysql8: require('../fixtures/databases/mysql8.json'),
    mariadb: require('../fixtures/databases/mariadb.json'),
    sqlite: require('../fixtures/databases/sqlite.json'),
    mongodb: require('../fixtures/databases/mongodb.json'),
    redis: require('../fixtures/databases/redis.json'),
    elasticsearch: require('../fixtures/databases/elasticsearch.json'),
    clickhouse: require('../fixtures/databases/clickhouse.json'),
};

// Validate CE fixtures on module load
const ceValidation = validateAllFixtures(ceDatabaseConfigs);
if (!ceValidation.allValid) {
    console.error('CE fixture validation failed - some tests may be skipped');
}

// Additional database configurations - dynamically loaded at build time using webpack's require.context
// This scans for any JSON files without knowing specific database names
let additionalConfigs = {};
try {
    const context = require.context('../../../ee/frontend/cypress/fixtures/databases', false, /\.json$/);
    context.keys().forEach(key => {
        const name = key.replace('./', '').replace('.json', '');
        additionalConfigs[name] = context(key);
    });
} catch (e) {
    // Extension fixtures directory not available
}

// Active database configs
let databaseConfigs = {...ceDatabaseConfigs, ...additionalConfigs};

// Track whether we've loaded configs from Cypress.env (to avoid duplicate loading)
let envConfigsLoaded = false;

/**
 * Load additional database configs from Cypress.env (set by cypress.config.js)
 * This is needed because esbuild doesn't support require.context
 */
function loadEnvConfigs() {
    if (envConfigsLoaded) {
        return;
    }
    if (typeof Cypress !== 'undefined') {
        const envConfigs = Cypress.env('additionalDatabaseConfigs');
        if (envConfigs && typeof envConfigs === 'object') {
            databaseConfigs = {...databaseConfigs, ...envConfigs};
            envConfigsLoaded = true;
        }
    }
}

/**
 * Register additional database configurations (used by EE to add EE databases)
 * @param {Object} additionalConfigs - Map of database configs to add
 */
export function registerDatabases(additionalConfigs) {
    databaseConfigs = {...databaseConfigs, ...additionalConfigs};
}

/**
 * Get the current database configurations
 * @returns {Object} Map of all registered database configs
 */
export function getDatabaseConfigs() {
    return databaseConfigs;
}

/**
 * Get database configuration by name
 * @param {string} dbName - Database identifier (e.g., 'postgres', 'mysql8')
 * @returns {Object} Database configuration
 */
export function getDatabaseConfig(dbName) {
    const config = databaseConfigs[dbName.toLowerCase()];
    if (!config) {
        throw new Error(`Unknown database: ${dbName}. Available: ${Object.keys(databaseConfigs).join(', ')}`);
    }
    return config;
}

/**
 * Get all database configurations
 * @returns {Object} Map of all database configs
 */
export function getAllDatabaseConfigs() {
    return databaseConfigs;
}

/**
 * Get databases filtered by category
 * @param {string} category - 'sql', 'document', 'keyvalue', or 'all'
 * @returns {Array<Object>} Array of database configurations
 */
export function getDatabasesByCategory(category) {
    const allDatabases = Object.values(databaseConfigs);

    if (category === 'all') {
        return allDatabases;
    }

    return allDatabases.filter(db => db.category === category);
}

/**
 * Get database identifier from config
 * @param {Object} dbConfig - Database configuration
 * @returns {string} Database identifier
 */
export function getDatabaseId(dbConfig) {
    return dbConfig.id || dbConfig.type.toLowerCase();
}

/**
 * Login to database using configuration with session caching
 * Uses cy.session() to cache login state and avoid re-authenticating on every test
 * @param {Object} dbConfig - Database configuration
 * @param {Object} options - Additional options
 * @param {boolean} options.visitStorageUnit - Whether to navigate to storage-unit after login (default: true)
 */
export function loginToDatabase(dbConfig, options = {}) {
    const {visitStorageUnit = true} = options;
    // Convert null to undefined for login command (it has special handling for undefined)
    const host = dbConfig.connection.host ?? undefined;
    const user = dbConfig.connection.user ?? undefined;
    const password = dbConfig.connection.password ?? undefined;
    const database = dbConfig.connection.database ?? undefined;
    const advanced = dbConfig.connection.advanced || {};

    // Use uiType for dropdown selection if available, otherwise use type
    const databaseType = dbConfig.uiType || dbConfig.type;

    // Create unique session key based on connection details
    const sessionKey = [
        databaseType,
        host || 'default',
        database || 'default',
        dbConfig.schema || 'default'
    ];

    cy.session(sessionKey, () => {
        // Perform actual login (only runs if session not cached)
        cy.login(
            databaseType,
            host,
            user,
            password,
            database,
            advanced
        );

        // Select schema if applicable
        if (dbConfig.schema) {
            cy.selectSchema(dbConfig.schema);
        }
    }, {
        validate() {
            // Quick validation that session is still valid
            cy.visit('/storage-unit', { failOnStatusCode: false });
            cy.get('[data-testid="sidebar-database"], [data-testid="sidebar-schema"]', { timeout: 5000 })
                .should('exist');
        },
        cacheAcrossSpecs: true // Share session across spec files for same database
    });

    if (visitStorageUnit) {
        cy.visit('/storage-unit');
        cy.get('[data-testid="storage-unit-card"]', { timeout: 15000 })
            .should('have.length.at.least', 1);
    }
}

/**
 * Main test runner - iterates over databases based on filter
 *
 * Usage:
 *   forEachDatabase('sql', (db) => {
 *     it('does something', () => { ... });
 *   });
 *
 *   // Only run for databases that support specific features (skips others entirely)
 *   forEachDatabase('sql', (db) => {
 *     it('tests graph', () => { ... });
 *   }, { features: ['graph'] });
 *
 * @param {string} categoryFilter - 'sql', 'document', 'keyvalue', or 'all'
 * @param {Function} testFn - Function that receives database config and defines tests
 * @param {Object} options - Additional options
 * @param {boolean} options.login - Whether to auto-login before each test (default: true)
 * @param {boolean} options.logout - Whether to auto-logout after each test (default: true)
 * @param {boolean} options.navigateToStorageUnit - Whether to navigate to storage-unit after login (default: true)
 * @param {string[]} options.features - Required features; databases without ALL features are skipped entirely
 */
export function forEachDatabase(categoryFilter, testFn, options = {}) {
    const {login = true, logout = true, navigateToStorageUnit = true, features = []} = options;

    // Load any additional configs from Cypress.env (needed for EE databases with esbuild)
    loadEnvConfigs();

    // Validate requested features to catch typos early
    for (const feature of features) {
        if (!VALID_FEATURES.includes(feature)) {
            throw new Error(`Unknown feature '${feature}' in forEachDatabase options. Valid features: ${VALID_FEATURES.join(', ')}`);
        }
    }

    // Get target database from env (if running single database)
    const targetDb = Cypress.env('database');
    const targetCategory = Cypress.env('category');

    // Get databases matching category filter
    let databases = getDatabasesByCategory(categoryFilter);

    // Filter by required features BEFORE creating any test blocks
    // This avoids login/setup overhead for databases that don't support the feature
    if (features.length > 0) {
        databases = databases.filter(db =>
            features.every(feature => hasFeature(db, feature))
        );
    }

    // If running specific database, filter to just that one
    if (targetDb) {
        databases = databases.filter(db => {
            const dbId = getDatabaseId(db);
            return dbId === targetDb.toLowerCase() ||
                db.type.toLowerCase() === targetDb.toLowerCase();
        });
    }

    // If running specific category via env, filter
    if (targetCategory && categoryFilter !== 'all') {
        if (categoryFilter !== targetCategory) {
            // Skip this entire block - category doesn't match
            return;
        }
    }

    // If no databases match, skip
    if (databases.length === 0) {
        return;
    }

    // Create describe block for each database
    databases.forEach(dbConfig => {
        const dbId = getDatabaseId(dbConfig);

        describe(`[${dbConfig.type}]`, () => {
            // Store current database config for access in tests
            beforeEach(function () {
                // Make config available via this.db
                this.db = dbConfig;
                Cypress.env('currentDatabase', dbConfig);

                if (login) {
                    loginToDatabase(dbConfig, {visitStorageUnit: navigateToStorageUnit});
                }
            });

            afterEach(function () {
                if (logout) {
                    cy.logout();
                }
            });

            // Run the test function with database config
            testFn(dbConfig);
        });
    });
}

/**
 * Check if a feature is supported for the given database
 * @param {Object} dbConfig - Database configuration
 * @param {string} feature - Feature name (e.g., 'graph', 'export', 'chat')
 * @returns {boolean} Whether feature is supported
 */
export function hasFeature(dbConfig, feature) {
    return dbConfig.features && dbConfig.features[feature] === true;
}

/**
 * Skip test if feature is not supported
 * @param {Object} dbConfig - Database configuration
 * @param {string} feature - Feature name
 */
export function skipIfNoFeature(dbConfig, feature) {
    if (!hasFeature(dbConfig, feature)) {
        return true;
    }
    return false;
}

/**
 * Get SQL query for database (handles schema prefixes)
 * @param {Object} dbConfig - Database configuration
 * @param {string} queryKey - Query key from config (e.g., 'selectAllUsers')
 * @returns {string} SQL query
 */
export function getSqlQuery(dbConfig, queryKey) {
    if (!dbConfig.sql || !dbConfig.sql[queryKey]) {
        throw new Error(`SQL query '${queryKey}' not found for ${dbConfig.type}`);
    }
    return dbConfig.sql[queryKey];
}

/**
 * Get expected error pattern for database
 * @param {Object} dbConfig - Database configuration
 * @param {string} errorKey - Error pattern key (e.g., 'tableNotFound')
 * @returns {string} Error pattern
 */
export function getErrorPattern(dbConfig, errorKey) {
    if (!dbConfig.sql || !dbConfig.sql.errorPatterns || !dbConfig.sql.errorPatterns[errorKey]) {
        return null;
    }
    return dbConfig.sql.errorPatterns[errorKey];
}

/**
 * Get table configuration from database config
 * @param {Object} dbConfig - Database configuration
 * @param {string} tableName - Table name
 * @returns {Object} Table configuration
 */
export function getTableConfig(dbConfig, tableName) {
    if (!dbConfig.tables || !dbConfig.tables[tableName]) {
        return null;
    }
    return dbConfig.tables[tableName];
}

/**
 * Conditional test - only runs if condition is met
 * Use: conditionalIt(hasFeature(db, 'chat'), 'tests chat', () => { ... })
 * @param {boolean} condition - Whether to run the test
 * @param {string} name - Test name
 * @param {Function} fn - Test function
 */
export function conditionalIt(condition, name, fn) {
    if (condition) {
        it(name, fn);
    } else {
        it.skip(name, fn);
    }
}

/**
 * Conditional describe - only runs if condition is met
 * @param {boolean} condition - Whether to run the describe block
 * @param {string} name - Describe block name
 * @param {Function} fn - Describe function
 */
export function conditionalDescribe(condition, name, fn) {
    if (condition) {
        describe(name, fn);
    } else {
        describe.skip(name, fn);
    }
}
