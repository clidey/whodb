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
    // Valid data types for creating tables/collections
    dataTypes?: string[];
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
        supportsScratchpad: true,
        supportsSchema: true,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: true,  // Uses database field for graph queries
    },
    {
        id: "MySQL",
        label: "MySQL",
        icon: Icons.Logos.MySQL,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "Local", "Allow clear text passwords": "0"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        supportsScratchpad: true,
        supportsSchema: true,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: true,  // Uses database field for graph queries
    },
    {
        id: "MariaDB",
        label: "MariaDB",
        icon: Icons.Logos.MariaDB,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "Local", "Allow clear text passwords": "0"},
        fields: {
            hostname: true,
            username: true,
            password: true,
            database: true,
        },
        supportsScratchpad: true,
        supportsSchema: true,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: true,  // Uses database field for graph queries
    },
    {
        id: "Sqlite3",
        label: "Sqlite3",
        icon: Icons.Logos.Sqlite3,
        extra: {},
        fields: {
            database: true,  // SQLite only needs database field
        },
        supportsScratchpad: true,
        supportsSchema: false,  // SQLite doesn't support schemas
        supportsDatabaseSwitching: false,
        usesSchemaForGraph: true,  // Uses schema field for graph queries
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
        usesSchemaForGraph: true,  // Uses schema field for graph queries
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
        supportsScratchpad: true,
        supportsSchema: false,
        supportsDatabaseSwitching: true,
        usesSchemaForGraph: false,  // Uses database field for graph queries
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
                dataTypes: dbType.dataTypes,
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