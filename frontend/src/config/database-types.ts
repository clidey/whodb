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

import {ReactElement} from "react";
import {Icons} from "../components/icons";

/**
 * Type category for grouping database types in the UI.
 */
export type TypeCategory = 'numeric' | 'text' | 'binary' | 'datetime' | 'boolean' | 'json' | 'other';

/**
 * SSL mode option for database connections.
 * Matches backend ssl.SSLModeInfo structure.
 */
export interface SSLModeOption {
    /** Mode value used in configuration (e.g., "required", "verify-ca") */
    value: string;
    /** Accepted aliases for this mode (e.g., PostgreSQL's "require" for "required") */
    aliases?: string[];
}

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
    // Whether this database type is an AWS managed service (hidden when cloud providers disabled)
    isAwsManaged?: boolean;
    // SSL modes supported by this database (undefined = no SSL support, e.g., SQLite)
    sslModes?: SSLModeOption[];
}

// Common SSL mode sets matching backend ssl.go definitions
const SSL_MODES_STANDARD: SSLModeOption[] = [
    { value: 'disabled' },
    { value: 'required', aliases: ['require'] },  // PostgreSQL uses 'require'
    { value: 'verify-ca' },
    { value: 'verify-identity', aliases: ['verify-full'] },  // PostgreSQL uses 'verify-full'
];

const SSL_MODES_WITH_PREFERRED: SSLModeOption[] = [
    { value: 'disabled', aliases: ['DISABLED'] },
    { value: 'preferred', aliases: ['PREFERRED'] },
    { value: 'required', aliases: ['REQUIRED'] },
    { value: 'verify-ca', aliases: ['VERIFY_CA'] },
    { value: 'verify-identity', aliases: ['VERIFY_IDENTITY'] },
];

const SSL_MODES_SIMPLE: SSLModeOption[] = [
    { value: 'disabled' },
    { value: 'enabled' },
    { value: 'insecure' },
];

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
        supportsModifiers: true,
        supportsScratchpad: true,
        supportsSchema: true,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: true,
        sslModes: SSL_MODES_STANDARD,
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
        supportsModifiers: true,
        supportsScratchpad: true,
        supportsSchema: false,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,
        sslModes: SSL_MODES_WITH_PREFERRED,
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
        supportsModifiers: true,
        supportsScratchpad: true,
        supportsSchema: false,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,
        sslModes: SSL_MODES_WITH_PREFERRED,
    },
    {
        id: "Sqlite3",
        label: "Sqlite3",
        icon: Icons.Logos.Sqlite3,
        extra: {},
        fields: {
            database: true,
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
        supportsScratchpad: false,
        supportsSchema: false,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,
        sslModes: SSL_MODES_SIMPLE,
    },
    {
        id: "Redis",
        label: "Redis",
        icon: Icons.Logos.Redis,
        extra: {"Port": "6379"},
        fields: {
            hostname: true,
            username: true,  // Redis 6+ supports ACL with username
            password: true,
        },
        supportsScratchpad: false,
        supportsSchema: false,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,
        sslModes: SSL_MODES_SIMPLE,
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
        },
        supportsScratchpad: false,
        supportsSchema: false,
        supportsDatabaseSwitching: false,
        usesSchemaForGraph: false,
        sslModes: SSL_MODES_SIMPLE,
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
        supportsModifiers: true,
        supportsScratchpad: true,
        supportsSchema: false,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,
        sslModes: SSL_MODES_SIMPLE,
    },
    // AWS managed database types (discovered via AWS providers, use underlying plugins)
    {
        id: "ElastiCache",
        label: "ElastiCache",
        icon: Icons.Logos.ElastiCache,
        extra: {"Port": "6379", "TLS": "true"},
        fields: {
            hostname: true,
            username: true,
            password: true,
        },
        supportsScratchpad: false,
        supportsSchema: false,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,
        isAwsManaged: true,
        sslModes: SSL_MODES_SIMPLE,  // Uses Redis SSL modes
    },
    {
        id: "DocumentDB",
        label: "DocumentDB",
        icon: Icons.Logos.DocumentDB,
        extra: {"Port": "27017"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        supportsScratchpad: false,
        supportsSchema: false,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,
        isAwsManaged: true,
        sslModes: SSL_MODES_SIMPLE,  // Uses MongoDB SSL modes
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

/**
 * Filter options for database type retrieval.
 */
export interface DatabaseTypeFilterOptions {
    /** When false, AWS managed database types (ElastiCache, DocumentDB) are excluded */
    cloudProvidersEnabled?: boolean;
}

/**
 * Get all database types, optionally filtered by cloud provider availability.
 * @param options Filter options for database types
 * @returns Promise resolving to filtered list of database types
 */
export const getDatabaseTypeDropdownItems = async (
    options: DatabaseTypeFilterOptions = {}
): Promise<IDatabaseDropdownItem[]> => {
    const { cloudProvidersEnabled = true } = options;
    const isEE = import.meta.env.VITE_BUILD_EDITION === 'ee';

    let allTypes: IDatabaseDropdownItem[];

    if (isEE && eeLoadPromise) {
        // Wait for EE to load
        await eeLoadPromise;

        if (eeDatabaseTypes.length > 0) {
            allTypes = [...baseDatabaseTypes, ...eeDatabaseTypes];
        } else {
            allTypes = baseDatabaseTypes;
        }
    } else {
        allTypes = baseDatabaseTypes;
    }

    // Filter out AWS managed types when cloud providers are disabled
    if (!cloudProvidersEnabled) {
        return allTypes.filter(item => !item.isAwsManaged);
    }

    return allTypes;
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
