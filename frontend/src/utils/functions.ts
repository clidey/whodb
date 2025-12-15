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

import isNaN from "lodash/isNaN";
import sampleSize from "lodash/sampleSize";
import { DatabaseType } from '@graphql';

/**
 * Checks if a string can be parsed as a numeric value.
 * @param str - The string to check
 * @returns True if the string represents a valid number
 */
export function isNumeric(str: string) {
    return !isNaN(Number(str));
}

/**
 * Converts a name to a URL-friendly slug.
 * @param name - The name to convert
 * @returns Lowercase hyphenated string (e.g., "My Name" -> "my-name")
 */
export function createStub(name: string) {
    return name.split(" ").map(word => word.toLowerCase()).join("-");
}

/**
 * Detects if a string contains markdown formatting.
 * @param text - The text to check
 * @returns True if the text contains common markdown patterns
 */
export function isMarkdown(text: string): boolean {
    const markdownPatterns = [
        /^#{1,6}\s+/,
        /^\s*[-*+]\s+/,
        /^\d+\.\s+/,
        /\*\*[^*]+\*\*/,
        /_[^_]+_/,
        /!\[.*?\]\(.*?\)/,
        /\[.*?\]\(.*?\)/,
        /^>\s+/,
        /`{1,3}[^`]*`{1,3}/,
        /-{3,}/,
    ];

    return markdownPatterns.some(pattern => pattern.test(text));
}

/**
 * Checks if a string could be valid JSON by checking for opening brace.
 * Used for early intellisense triggering before full JSON validation.
 * @param str - The string to check
 * @returns True if the string starts with "{"
 */
export function isValidJSON(str: string): boolean {
    return str.startsWith("{");
}

// Initialize EE NoSQL check function
let isEENoSQLDatabase: ((databaseType: string) => boolean) | null = null;

// Load EE NoSQL check if available
if (import.meta.env.VITE_BUILD_EDITION === 'ee') {
    import('@ee/index').then((eeModule) => {
        if (eeModule?.isEENoSQLDatabase) {
            isEENoSQLDatabase = eeModule.isEENoSQLDatabase;
        }
    }).catch(() => {
        // EE module not available, continue with CE functionality
    });
}

/**
 * Determines if a database type is a NoSQL database.
 * @param databaseType - The database type string to check
 * @returns True for MongoDB, Redis, ElasticSearch, and EE NoSQL databases
 */
export function isNoSQL(databaseType: string) {
    if (isEENoSQLDatabase && isEENoSQLDatabase(databaseType)) {
        return true;
    }

    switch (databaseType) {
        case DatabaseType.MongoDb:
        case DatabaseType.Redis:
        case DatabaseType.ElasticSearch:
            return true;
    }
    return false;
}

// Initialize EE storage label function
let getEEDatabaseStorageUnitLabel: ((databaseType: string | undefined, singular: boolean) => string | null) | null = null;

// Load EE function if available
if (import.meta.env.VITE_BUILD_EDITION === 'ee') {
    import('@ee/index').then((eeModule) => {
        if (eeModule?.getEEDatabaseStorageUnitLabel) {
            getEEDatabaseStorageUnitLabel = eeModule.getEEDatabaseStorageUnitLabel;
        }
    }).catch(() => {
        // EE module not available, continue with CE functionality
    });
}

/**
 * Returns the appropriate label for storage units based on database type.
 * @param databaseType - The database type
 * @param singular - Whether to return singular form (default: false)
 * @returns The label (e.g., "Tables", "Collections", "Indices")
 */
export function getDatabaseStorageUnitLabel(databaseType: string | undefined, singular: boolean = false) {
    if (getEEDatabaseStorageUnitLabel) {
        const eeLabel = getEEDatabaseStorageUnitLabel(databaseType, singular);
        if (eeLabel !== null) {
            return eeLabel;
        }
    }

    switch(databaseType) {
        case DatabaseType.ElasticSearch:
            return singular ? "Index" : "Indices";
        case DatabaseType.MongoDb:
            return singular ? "Collection" : "Collections";
        case DatabaseType.Redis:
            return singular ? "Key" : "Keys";
        case DatabaseType.MySql:
        case DatabaseType.Postgres:
        case DatabaseType.Sqlite3:
        case DatabaseType.ClickHouse:
            return singular ? "Table" : "Tables";
    }
    return singular ? "Storage Unit" : "Storage Units";
}

/**
 * Returns n random items from an array.
 * @param array - The source array
 * @param n - Number of items to return (default: 3)
 * @returns Array of n randomly selected items
 * @throws Error if n is greater than the array length
 */
export function chooseRandomItems<T>(array: T[], n: number = 3): T[] {
    if (n > array.length) {
        throw new Error("n cannot be greater than the array length");
    }
    return sampleSize(array, n);
}
