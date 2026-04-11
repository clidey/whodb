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

import {DatabaseType} from '@graphql';
import {getDatabaseTypeDropdownItemsSync} from '../config/database-types';
import {reduxStore} from '../store';

/**
 * Get backend capabilities from the Redux store.
 * Returns null if capabilities haven't been fetched yet.
 */
function getBackendCapabilities() {
    return reduxStore.getState().databaseMetadata.capabilities;
}

/**
 * Check if a database supports scratchpad/raw query execution.
 * Reads from backend capabilities first, falls back to config/hardcoded lists.
 */
export function databaseSupportsScratchpad(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }

    const capabilities = getBackendCapabilities();
    if (capabilities != null) {
        return capabilities.supportsScratchpad;
    }

    // Fallback: check database configuration
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);
    if (dbConfig?.supportsScratchpad != null) {
        return dbConfig.supportsScratchpad;
    }

    const databasesThatDontSupportScratchpad = [
        DatabaseType.MongoDb,
        DatabaseType.Redis,
        DatabaseType.ElasticSearch,
        DatabaseType.Memcached,
    ];
    return !databasesThatDontSupportScratchpad.includes(databaseType as DatabaseType);
}

/**
 * Check if a database supports schemas.
 * Reads from backend capabilities first, falls back to config/hardcoded lists.
 */
export function databaseSupportsSchema(databaseType: DatabaseType | string | undefined): boolean {
    if (databaseType == null) {
        return false;
    }

    const capabilities = getBackendCapabilities();
    if (capabilities != null) {
        return capabilities.supportsSchema;
    }

    // Fallback: check database configuration
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);
    if (dbConfig?.supportsSchema != null) {
        return dbConfig.supportsSchema;
    }

    const databasesThatDontSupportSchema = [
        DatabaseType.Sqlite3,
        DatabaseType.Redis,
        DatabaseType.ElasticSearch,
        DatabaseType.MongoDb,
        DatabaseType.ClickHouse,
        DatabaseType.MySql,
        DatabaseType.MariaDb,
        DatabaseType.Memcached,
        DatabaseType.TiDb,
    ];
    return !databasesThatDontSupportSchema.includes(databaseType as DatabaseType);
}

/**
 * Check if a database supports switching between databases in the UI.
 * Reads from backend capabilities first, falls back to config/hardcoded lists.
 */
export function databaseSupportsDatabaseSwitching(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }

    const capabilities = getBackendCapabilities();
    if (capabilities != null) {
        return capabilities.supportsDatabaseSwitch;
    }

    // Fallback: check database configuration
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);
    if (dbConfig?.supportsDatabaseSwitching !== undefined) {
        return dbConfig.supportsDatabaseSwitching;
    }

    const databasesThatSupportDatabaseSwitching = [
        DatabaseType.MongoDb,
        DatabaseType.ClickHouse,
        DatabaseType.Postgres,
        DatabaseType.MySql,
        DatabaseType.MariaDb,
        DatabaseType.TiDb,
        DatabaseType.Redis,
    ];
    return databasesThatSupportDatabaseSwitching.includes(databaseType as DatabaseType);
}

/**
 * Check if a database should use the schema field for graph queries.
 */
export function databaseUsesSchemaForGraph(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return true;
    }

    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);
    if (dbConfig?.usesSchemaForGraph !== undefined) {
        return dbConfig.usesSchemaForGraph;
    }

    return !databaseSupportsDatabaseSwitching(databaseType);
}

/**
 * Check if a database type uses the "database" concept instead of "schema".
 */
export function databaseTypesThatUseDatabaseInsteadOfSchema(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }

    const databasesThatUseDatabaseInsteadOfSchema = [
        DatabaseType.MongoDb,
        DatabaseType.ClickHouse,
        DatabaseType.MySql,
        DatabaseType.MariaDb,
        DatabaseType.TiDb,
        DatabaseType.Redis,
    ];
    return databasesThatUseDatabaseInsteadOfSchema.includes(databaseType as DatabaseType);
}

/**
 * Check if a database supports mock data generation.
 */
export function databaseSupportsMockData(databaseType: DatabaseType | string | undefined): boolean {
    if (!databaseType) {
        return false;
    }

    const databasesThatDontSupportMockData = [
        DatabaseType.Redis,
        DatabaseType.ElasticSearch,
        DatabaseType.Memcached,
    ];
    return !databasesThatDontSupportMockData.includes(databaseType as DatabaseType);
}
