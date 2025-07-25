/**
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

import { DatabaseType } from '@graphql';
import { getDatabaseTypeDropdownItemsSync } from '../config/database-types';

/**
 * Get valid data types for a database
 * @param databaseType The database type (can be CE or EE type)
 * @returns Array of valid data types for the database
 */
export function getDatabaseDataTypes(databaseType: DatabaseType | string): string[] {
    // Try to get data types from the database configuration first
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);
    
    if (dbConfig?.dataTypes) {
        return dbConfig.dataTypes;
    }
    
    // Fall back to built-in data types for known database types
    switch (databaseType) {
        case DatabaseType.MariaDb:
            return [
                "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "FLOAT", "DOUBLE", "DECIMAL",
                "DATE", "DATETIME", "TIMESTAMP", "TIME", "YEAR",
                "CHAR", "VARCHAR", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", 
                "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT",
                "ENUM", "SET", "JSON", "BOOLEAN"
            ];
        case DatabaseType.MySql:
        case DatabaseType.ClickHouse:
            return [
                "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "FLOAT", "DOUBLE", "DECIMAL",
                "DATE", "DATETIME", "TIMESTAMP", "TIME", "YEAR",
                "CHAR", "VARCHAR(255)", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", 
                "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT",
                "ENUM", "SET", "JSON", "BOOLEAN", "VARCHAR(100)", "VARCHAR(1000)"
            ];
        case DatabaseType.Postgres:
            return [
                "SMALLINT", "INTEGER", "BIGINT", "DECIMAL", "NUMERIC", "REAL", "DOUBLE PRECISION", "SMALLSERIAL", 
                "SERIAL", "BIGSERIAL", "MONEY",
                "CHAR", "VARCHAR", "TEXT", "BYTEA",
                "TIMESTAMP", "TIMESTAMPTZ", "DATE", "TIME", "TIMETZ",
                "BOOLEAN", "POINT", "LINE", "LSEG", "BOX", "PATH", "POLYGON", "CIRCLE",
                "CIDR", "INET", "MACADDR", "UUID", "XML", "JSON", "JSONB", "ARRAY", "HSTORE"
            ];
        case DatabaseType.Sqlite3:
            return [
                "NULL", "INTEGER", "REAL", "TEXT", "BLOB",
                "NUMERIC", "BOOLEAN", "DATE", "DATETIME"
            ];
        default:
            return [];
    }
}

/**
 * Check if a database supports field modifiers (primary, nullable)
 * @param databaseType The database type (can be CE or EE type)
 * @returns boolean indicating if the database supports modifiers
 */
export function databaseSupportsModifiers(databaseType: DatabaseType | string): boolean {
    // Try to get modifier support from the database configuration first
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);
    
    if (dbConfig?.supportsModifiers !== undefined) {
        return dbConfig.supportsModifiers;
    }
    
    // Fall back to checking known databases that support modifiers
    return [
        DatabaseType.MySql, 
        DatabaseType.MariaDb, 
        DatabaseType.Postgres, 
        DatabaseType.Sqlite3, 
        DatabaseType.ClickHouse
    ].includes(databaseType as DatabaseType);
}