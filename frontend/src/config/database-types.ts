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

import {ReactElement} from "react";
import {Icons} from "../components/icons";

/**
 * Type category for grouping database types in the UI.
 */
export type TypeCategory = 'numeric' | 'text' | 'binary' | 'datetime' | 'boolean' | 'json' | 'other';

/**
 * Defines a canonical database type for use in type selectors.
 * Types are from each database's official documentation.
 */
export interface TypeDefinition {
    /** Canonical type name (e.g., "VARCHAR", "INTEGER") - stored internally */
    id: string;
    /** Display label shown in UI (e.g., "varchar", "integer") - database's preferred case */
    label: string;
    /** Shows length input when selected (VARCHAR, CHAR) */
    hasLength?: boolean;
    /** Shows precision/scale inputs when selected (DECIMAL, NUMERIC) */
    hasPrecision?: boolean;
    /** Default length value for types with hasLength */
    defaultLength?: number;
    /** Default precision for types with hasPrecision */
    defaultPrecision?: number;
    /** Type category for grouping and icon selection */
    category: TypeCategory;
}

// Extended dropdown item type with UI field configuration
export interface IDatabaseDropdownItem {
    id: string;
    label: string;
    icon: ReactElement;
    extra: Record<string, string>;
    // UI field configuration
    fields?: {
        hostname?: boolean;
        username?: boolean;
        password?: boolean;
        database?: boolean;
    };
    // Valid operators for this database type
    operators?: string[];
    /** Canonical type definitions for type selectors */
    typeDefinitions?: TypeDefinition[];
    /** Maps type aliases to canonical names (e.g., INT4 -> INTEGER) */
    aliasMap?: Record<string, string>;
    // Whether this database supports field modifiers (primary, nullable)
    supportsModifiers?: boolean;
    // Whether this database supports scratchpad/raw query execution
    supportsScratchpad?: boolean;
    // Whether this database supports schemas
    supportsSchema?: boolean;
    // Whether this database supports switching between databases in the UI
    supportsDatabaseSwitching?: boolean;
    // Whether this database should use the schema field (true) or database field (false) for graph queries
    usesSchemaForGraph?: boolean;
}

export const baseDatabaseTypes: IDatabaseDropdownItem[] = [
    {
        id: "Postgres",
        label: "Postgres",
        icon: Icons.Logos.Postgres,
        extra: {"Port": "5432"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        typeDefinitions: [
            // Numeric types
            { id: 'SMALLINT', label: 'smallint', category: 'numeric' },
            { id: 'INTEGER', label: 'integer', category: 'numeric' },
            { id: 'BIGINT', label: 'bigint', category: 'numeric' },
            { id: 'SERIAL', label: 'serial', category: 'numeric' },
            { id: 'BIGSERIAL', label: 'bigserial', category: 'numeric' },
            { id: 'SMALLSERIAL', label: 'smallserial', category: 'numeric' },
            { id: 'DECIMAL', label: 'decimal', hasPrecision: true, defaultPrecision: 10, category: 'numeric' },
            { id: 'NUMERIC', label: 'numeric', hasPrecision: true, defaultPrecision: 10, category: 'numeric' },
            { id: 'REAL', label: 'real', category: 'numeric' },
            { id: 'DOUBLE PRECISION', label: 'double precision', category: 'numeric' },
            { id: 'MONEY', label: 'money', category: 'numeric' },
            // Text types
            { id: 'CHARACTER VARYING', label: 'varchar', hasLength: true, defaultLength: 255, category: 'text' },
            { id: 'CHARACTER', label: 'char', hasLength: true, defaultLength: 1, category: 'text' },
            { id: 'TEXT', label: 'text', category: 'text' },
            // Binary types
            { id: 'BYTEA', label: 'bytea', category: 'binary' },
            // Date/time types
            { id: 'TIMESTAMP', label: 'timestamp', category: 'datetime' },
            { id: 'TIMESTAMP WITH TIME ZONE', label: 'timestamptz', category: 'datetime' },
            { id: 'DATE', label: 'date', category: 'datetime' },
            { id: 'TIME', label: 'time', category: 'datetime' },
            { id: 'TIME WITH TIME ZONE', label: 'timetz', category: 'datetime' },
            { id: 'INTERVAL', label: 'interval', category: 'datetime' },
            // Boolean
            { id: 'BOOLEAN', label: 'boolean', category: 'boolean' },
            // JSON types
            { id: 'JSON', label: 'json', category: 'json' },
            { id: 'JSONB', label: 'jsonb', category: 'json' },
            // UUID
            { id: 'UUID', label: 'uuid', category: 'other' },
            // Network types
            { id: 'CIDR', label: 'cidr', category: 'other' },
            { id: 'INET', label: 'inet', category: 'other' },
            { id: 'MACADDR', label: 'macaddr', category: 'other' },
            // Geometric types
            { id: 'POINT', label: 'point', category: 'other' },
            { id: 'LINE', label: 'line', category: 'other' },
            { id: 'BOX', label: 'box', category: 'other' },
            { id: 'CIRCLE', label: 'circle', category: 'other' },
            { id: 'POLYGON', label: 'polygon', category: 'other' },
            // XML
            { id: 'XML', label: 'xml', category: 'other' },
        ],
        aliasMap: {
            'INT': 'INTEGER',
            'INT4': 'INTEGER',
            'INT8': 'BIGINT',
            'INT2': 'SMALLINT',
            'FLOAT4': 'REAL',
            'FLOAT8': 'DOUBLE PRECISION',
            'BOOL': 'BOOLEAN',
            'VARCHAR': 'CHARACTER VARYING',
            'CHAR': 'CHARACTER',
            'SERIAL4': 'SERIAL',
            'SERIAL8': 'BIGSERIAL',
            'SERIAL2': 'SMALLSERIAL',
            'TIMESTAMPTZ': 'TIMESTAMP WITH TIME ZONE',
            'TIMETZ': 'TIME WITH TIME ZONE',
        },
        supportsModifiers: true,
        supportsScratchpad: true,
        supportsSchema: true,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: true,
    },
    {
        id: "MySQL",
        label: "MySQL",
        icon: Icons.Logos.MySQL,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "UTC", "Allow clear text passwords": "0"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        typeDefinitions: [
            // Numeric types
            { id: 'TINYINT', label: 'TINYINT', category: 'numeric' },
            { id: 'SMALLINT', label: 'SMALLINT', category: 'numeric' },
            { id: 'MEDIUMINT', label: 'MEDIUMINT', category: 'numeric' },
            { id: 'INT', label: 'INT', category: 'numeric' },
            { id: 'BIGINT', label: 'BIGINT', category: 'numeric' },
            { id: 'DECIMAL', label: 'DECIMAL', hasPrecision: true, defaultPrecision: 10, category: 'numeric' },
            { id: 'FLOAT', label: 'FLOAT', category: 'numeric' },
            { id: 'DOUBLE', label: 'DOUBLE', category: 'numeric' },
            // Text types
            { id: 'VARCHAR', label: 'VARCHAR', hasLength: true, defaultLength: 255, category: 'text' },
            { id: 'CHAR', label: 'CHAR', hasLength: true, defaultLength: 1, category: 'text' },
            { id: 'TINYTEXT', label: 'TINYTEXT', category: 'text' },
            { id: 'TEXT', label: 'TEXT', category: 'text' },
            { id: 'MEDIUMTEXT', label: 'MEDIUMTEXT', category: 'text' },
            { id: 'LONGTEXT', label: 'LONGTEXT', category: 'text' },
            // Binary types
            { id: 'BINARY', label: 'BINARY', hasLength: true, defaultLength: 1, category: 'binary' },
            { id: 'VARBINARY', label: 'VARBINARY', hasLength: true, defaultLength: 255, category: 'binary' },
            { id: 'TINYBLOB', label: 'TINYBLOB', category: 'binary' },
            { id: 'BLOB', label: 'BLOB', category: 'binary' },
            { id: 'MEDIUMBLOB', label: 'MEDIUMBLOB', category: 'binary' },
            { id: 'LONGBLOB', label: 'LONGBLOB', category: 'binary' },
            // Date/time types
            { id: 'DATE', label: 'DATE', category: 'datetime' },
            { id: 'TIME', label: 'TIME', category: 'datetime' },
            { id: 'DATETIME', label: 'DATETIME', category: 'datetime' },
            { id: 'TIMESTAMP', label: 'TIMESTAMP', category: 'datetime' },
            { id: 'YEAR', label: 'YEAR', category: 'datetime' },
            // Boolean
            { id: 'BOOL', label: 'BOOL', category: 'boolean' },
            // JSON
            { id: 'JSON', label: 'JSON', category: 'json' },
            // Other
            { id: 'ENUM', label: 'ENUM', category: 'other' },
            { id: 'SET', label: 'SET', category: 'other' },
        ],
        aliasMap: {
            'INTEGER': 'INT',
            'BOOLEAN': 'BOOL',
            'NUMERIC': 'DECIMAL',
            'DEC': 'DECIMAL',
            'REAL': 'DOUBLE',
        },
        supportsModifiers: true,
        supportsScratchpad: true,
        supportsSchema: true,
        supportsDatabaseSwitching: false,
        usesSchemaForGraph: true,
    },
    {
        id: "MariaDB",
        label: "MariaDB",
        icon: Icons.Logos.MariaDB,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "UTC", "Allow clear text passwords": "0"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        typeDefinitions: [
            // Numeric types
            { id: 'TINYINT', label: 'TINYINT', category: 'numeric' },
            { id: 'SMALLINT', label: 'SMALLINT', category: 'numeric' },
            { id: 'MEDIUMINT', label: 'MEDIUMINT', category: 'numeric' },
            { id: 'INT', label: 'INT', category: 'numeric' },
            { id: 'BIGINT', label: 'BIGINT', category: 'numeric' },
            { id: 'DECIMAL', label: 'DECIMAL', hasPrecision: true, defaultPrecision: 10, category: 'numeric' },
            { id: 'FLOAT', label: 'FLOAT', category: 'numeric' },
            { id: 'DOUBLE', label: 'DOUBLE', category: 'numeric' },
            // Text types
            { id: 'VARCHAR', label: 'VARCHAR', hasLength: true, defaultLength: 255, category: 'text' },
            { id: 'CHAR', label: 'CHAR', hasLength: true, defaultLength: 1, category: 'text' },
            { id: 'TINYTEXT', label: 'TINYTEXT', category: 'text' },
            { id: 'TEXT', label: 'TEXT', category: 'text' },
            { id: 'MEDIUMTEXT', label: 'MEDIUMTEXT', category: 'text' },
            { id: 'LONGTEXT', label: 'LONGTEXT', category: 'text' },
            // Binary types
            { id: 'BINARY', label: 'BINARY', hasLength: true, defaultLength: 1, category: 'binary' },
            { id: 'VARBINARY', label: 'VARBINARY', hasLength: true, defaultLength: 255, category: 'binary' },
            { id: 'TINYBLOB', label: 'TINYBLOB', category: 'binary' },
            { id: 'BLOB', label: 'BLOB', category: 'binary' },
            { id: 'MEDIUMBLOB', label: 'MEDIUMBLOB', category: 'binary' },
            { id: 'LONGBLOB', label: 'LONGBLOB', category: 'binary' },
            // Date/time types
            { id: 'DATE', label: 'DATE', category: 'datetime' },
            { id: 'TIME', label: 'TIME', category: 'datetime' },
            { id: 'DATETIME', label: 'DATETIME', category: 'datetime' },
            { id: 'TIMESTAMP', label: 'TIMESTAMP', category: 'datetime' },
            { id: 'YEAR', label: 'YEAR', category: 'datetime' },
            // Boolean
            { id: 'BOOL', label: 'BOOL', category: 'boolean' },
            // JSON
            { id: 'JSON', label: 'JSON', category: 'json' },
            // Other
            { id: 'ENUM', label: 'ENUM', category: 'other' },
            { id: 'SET', label: 'SET', category: 'other' },
            { id: 'UUID', label: 'UUID', category: 'other' },
        ],
        aliasMap: {
            'INTEGER': 'INT',
            'BOOLEAN': 'BOOL',
            'NUMERIC': 'DECIMAL',
            'DEC': 'DECIMAL',
            'REAL': 'DOUBLE',
        },
        supportsModifiers: true,
        supportsScratchpad: true,
        supportsSchema: true,
        supportsDatabaseSwitching: false,
        usesSchemaForGraph: true,
    },
    {
        id: "Sqlite3",
        label: "Sqlite3",
        icon: Icons.Logos.Sqlite3,
        extra: {},
        fields: {
            database: true,  // SQLite only needs database field
        },
        typeDefinitions: [
            { id: 'INTEGER', label: 'INTEGER', category: 'numeric' },
            { id: 'REAL', label: 'REAL', category: 'numeric' },
            { id: 'TEXT', label: 'TEXT', category: 'text' },
            { id: 'BLOB', label: 'BLOB', category: 'binary' },
            { id: 'NUMERIC', label: 'NUMERIC', category: 'numeric' },
            { id: 'BOOLEAN', label: 'BOOLEAN', category: 'boolean' },
            { id: 'DATE', label: 'DATE', category: 'datetime' },
            { id: 'DATETIME', label: 'DATETIME', category: 'datetime' },
        ],
        aliasMap: {
            'INT': 'INTEGER',
            'TINYINT': 'INTEGER',
            'SMALLINT': 'INTEGER',
            'MEDIUMINT': 'INTEGER',
            'BIGINT': 'INTEGER',
            'INT2': 'INTEGER',
            'INT8': 'INTEGER',
            'DOUBLE': 'REAL',
            'DOUBLE PRECISION': 'REAL',
            'FLOAT': 'REAL',
            'CHARACTER': 'TEXT',
            'VARCHAR': 'TEXT',
            'VARYING CHARACTER': 'TEXT',
            'NCHAR': 'TEXT',
            'NATIVE CHARACTER': 'TEXT',
            'NVARCHAR': 'TEXT',
            'CLOB': 'TEXT',
        },
        supportsModifiers: true,
        supportsScratchpad: true,
        supportsSchema: false,
        supportsDatabaseSwitching: false,
        usesSchemaForGraph: true,
    },
    {
        id: "MongoDB",
        label: "MongoDB",
        icon: Icons.Logos.MongoDB,
        extra: {"Port": "27017", "URL Params": "?", "DNS Enabled": "false"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        supportsScratchpad: false,  // MongoDB doesn't support SQL scratchpad
        supportsSchema: false,  // MongoDB doesn't have traditional schemas
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,  // Uses database field for graph queries
    },
    {
        id: "Redis",
        label: "Redis",
        icon: Icons.Logos.Redis,
        extra: {"Port": "6379"},
        fields: {
            hostname: true,
            // username: false - Redis doesn't use username
            password: true,
            // database: false - Redis doesn't use database field
        },
        supportsScratchpad: false,  // Redis doesn't support SQL scratchpad
        supportsSchema: false,  // Redis doesn't have schemas
        supportsDatabaseSwitching: false,
        usesSchemaForGraph: true,  // Uses schema field for graph queries
    },
    {
        id: "ElasticSearch",
        label: "ElasticSearch",
        icon: Icons.Logos.ElasticSearch,
        extra: {"Port": "9200", "SSL Mode": "disable"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            // database: false - ElasticSearch doesn't use database field
        },
        supportsScratchpad: false,  // ElasticSearch doesn't support SQL scratchpad
        supportsSchema: false,  // ElasticSearch doesn't have schemas
        supportsDatabaseSwitching: false,
        usesSchemaForGraph: false,  // Uses database field (empty) for graph queries
    },
    {
        id: "ClickHouse",
        label: "ClickHouse",
        icon: Icons.Logos.ClickHouse,
        extra: {
            "Port": "9000",
            "SSL mode": "disable",
            "HTTP Protocol": "disable",
            "Readonly": "disable",
            "Debug": "disable"
        },
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        typeDefinitions: [
            // Integer types
            { id: 'Int8', label: 'Int8', category: 'numeric' },
            { id: 'Int16', label: 'Int16', category: 'numeric' },
            { id: 'Int32', label: 'Int32', category: 'numeric' },
            { id: 'Int64', label: 'Int64', category: 'numeric' },
            { id: 'Int128', label: 'Int128', category: 'numeric' },
            { id: 'Int256', label: 'Int256', category: 'numeric' },
            { id: 'UInt8', label: 'UInt8', category: 'numeric' },
            { id: 'UInt16', label: 'UInt16', category: 'numeric' },
            { id: 'UInt32', label: 'UInt32', category: 'numeric' },
            { id: 'UInt64', label: 'UInt64', category: 'numeric' },
            { id: 'UInt128', label: 'UInt128', category: 'numeric' },
            { id: 'UInt256', label: 'UInt256', category: 'numeric' },
            // Float types
            { id: 'Float32', label: 'Float32', category: 'numeric' },
            { id: 'Float64', label: 'Float64', category: 'numeric' },
            // Decimal
            { id: 'Decimal', label: 'Decimal', hasPrecision: true, defaultPrecision: 10, category: 'numeric' },
            { id: 'Decimal32', label: 'Decimal32', hasPrecision: true, defaultPrecision: 9, category: 'numeric' },
            { id: 'Decimal64', label: 'Decimal64', hasPrecision: true, defaultPrecision: 18, category: 'numeric' },
            { id: 'Decimal128', label: 'Decimal128', hasPrecision: true, defaultPrecision: 38, category: 'numeric' },
            // String types
            { id: 'String', label: 'String', category: 'text' },
            { id: 'FixedString', label: 'FixedString', hasLength: true, defaultLength: 16, category: 'text' },
            // Date/time types
            { id: 'Date', label: 'Date', category: 'datetime' },
            { id: 'Date32', label: 'Date32', category: 'datetime' },
            { id: 'DateTime', label: 'DateTime', category: 'datetime' },
            { id: 'DateTime64', label: 'DateTime64', category: 'datetime' },
            // Boolean (ClickHouse 21.12+)
            { id: 'Bool', label: 'Bool', category: 'boolean' },
            // UUID
            { id: 'UUID', label: 'UUID', category: 'other' },
            // JSON
            { id: 'JSON', label: 'JSON', category: 'json' },
            // Network types
            { id: 'IPv4', label: 'IPv4', category: 'other' },
            { id: 'IPv6', label: 'IPv6', category: 'other' },
            // Enum
            { id: 'Enum8', label: 'Enum8', category: 'other' },
            { id: 'Enum16', label: 'Enum16', category: 'other' },
        ],
        aliasMap: {
            'TINYINT': 'Int8',
            'SMALLINT': 'Int16',
            'INT': 'Int32',
            'INTEGER': 'Int32',
            'BIGINT': 'Int64',
            'FLOAT': 'Float32',
            'DOUBLE': 'Float64',
            'VARCHAR': 'String',
            'CHAR': 'FixedString',
            'TEXT': 'String',
            'BOOLEAN': 'Bool',
        },
        supportsModifiers: true,
        supportsScratchpad: true,
        supportsSchema: false,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,
    },
];

// This will be populated if EE is loaded
let eeDatabaseTypes: IDatabaseDropdownItem[] = [];
let eeLoadPromise: Promise<void> | null = null;

// Load EE database types if in EE mode
if (import.meta.env.VITE_BUILD_EDITION === 'ee') {
    // Store the promise so we can await it later
    eeLoadPromise = Promise.all([
        import('@ee/config.tsx'),
        import('@ee/icons')
    ]).then(([eeConfig, eeIcons]) => {
        if (eeConfig?.eeDatabaseTypes && eeIcons?.EEIcons?.Logos) {
            // First merge the icons
            Object.assign(Icons.Logos, eeIcons.EEIcons.Logos);
            
            // Then map EE database types to the correct format with resolved icons
            // @ts-ignore - TODO: fix this
            eeDatabaseTypes = eeConfig.eeDatabaseTypes.map(dbType => ({
                id: dbType.id,
                label: dbType.label,
                icon: Icons.Logos[dbType.iconName as keyof typeof Icons.Logos],
                extra: dbType.extra,
                fields: dbType.fields,
                operators: dbType.operators,
                typeDefinitions: dbType.typeDefinitions,
                aliasMap: dbType.aliasMap,
                supportsModifiers: dbType.supportsModifiers,
                supportsScratchpad: dbType.supportsScratchpad,
                supportsSchema: dbType.supportsSchema,
                supportsDatabaseSwitching: dbType.supportsDatabaseSwitching,
                usesSchemaForGraph: dbType.usesSchemaForGraph,
            }));
            
        } else {
            console.warn('EE modules loaded but missing expected exports');
        }
    }).catch((error) => {
        console.error('Could not load EE database types:', error);
    });
}

// Get all database types - now returns a promise if EE is loading
export const getDatabaseTypeDropdownItems = async (): Promise<IDatabaseDropdownItem[]> => {
    const isEE = import.meta.env.VITE_BUILD_EDITION === 'ee';
    
    if (isEE && eeLoadPromise) {
        // Wait for EE to load
        await eeLoadPromise;
        
        if (eeDatabaseTypes.length > 0) {
            return [...baseDatabaseTypes, ...eeDatabaseTypes];
        }
    }
    
    return baseDatabaseTypes;
};

// For backward compatibility, provide a synchronous version that only returns base types initially
export const getDatabaseTypeDropdownItemsSync = (): IDatabaseDropdownItem[] => {
    const isEE = import.meta.env.VITE_BUILD_EDITION === 'ee';
    
    if (isEE && eeDatabaseTypes.length > 0) {
        return [...baseDatabaseTypes, ...eeDatabaseTypes];
    }
    
    return baseDatabaseTypes;
};

// Export this for components that need immediate access (will be updated when EE loads)
export let databaseTypeDropdownItems = baseDatabaseTypes;

// Update the exported items when EE loads
if (import.meta.env.VITE_BUILD_EDITION === 'ee' && eeLoadPromise) {
    eeLoadPromise.then(() => {
        if (eeDatabaseTypes.length > 0) {
            databaseTypeDropdownItems = [...baseDatabaseTypes, ...eeDatabaseTypes];
        }
    });
}
