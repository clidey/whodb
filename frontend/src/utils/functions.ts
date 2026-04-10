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

import sampleSize from "lodash/sampleSize";
import {DatabaseType} from '@graphql';

/**
 * Formats a number using locale-aware grouping separators (e.g. 1,000,000 in en-US, 10,00,000 in hi-IN).
 * @param value - The number to format
 * @param language - App locale string in underscore format (e.g. "en_US", "hi_IN")
 */
export function formatNumber(value: number, language: string): string {
    return new Intl.NumberFormat(language.replace('_', '-')).format(value);
}

/**
 * Checks if a string can be parsed as a numeric value.
 * @param str - The string to check
 * @returns True if the string represents a valid number
 */
export function isNumeric(str: string) {
    return !isNaN(Number(str));
}

// Extension NoSQL check function — set via registerDatabaseFunctions()
let isExtNoSQLDatabase: ((databaseType: string) => boolean) | null = null;

/**
 * Determines if a database type is a NoSQL database.
 * @param databaseType - The database type string to check
 * @returns True for MongoDB, Redis, ElasticSearch, and extension NoSQL databases
 */
export function isNoSQL(databaseType: string) {
    if (isExtNoSQLDatabase && isExtNoSQLDatabase(databaseType)) {
        return true;
    }

    switch (databaseType) {
        case DatabaseType.MongoDb:
        case DatabaseType.Redis:
        case DatabaseType.ElasticSearch:
        case DatabaseType.Memcached:
            return true;
    }
    return false;
}

// Extension storage label function — set via registerDatabaseFunctions()
let getExtDatabaseStorageUnitLabel: ((databaseType: string | undefined, singular: boolean) => string | null) | null = null;

/** Register extension utility functions. */
export function registerDatabaseFunctions(fns: {
    isExtNoSQLDatabase?: (databaseType: string) => boolean;
    getExtDatabaseStorageUnitLabel?: (databaseType: string | undefined, singular: boolean) => string | null;
}) {
    if (fns.isExtNoSQLDatabase) {
        isExtNoSQLDatabase = fns.isExtNoSQLDatabase;
    }
    if (fns.getExtDatabaseStorageUnitLabel) {
        getExtDatabaseStorageUnitLabel = fns.getExtDatabaseStorageUnitLabel;
    }
}

/**
 * Returns the appropriate label for storage units based on database type.
 * @param databaseType - The database type
 * @param singular - Whether to return singular form (default: false)
 * @returns The label (e.g., "Tables", "Collections", "Indices")
 */
export function getDatabaseStorageUnitLabel(databaseType: string | undefined, singular: boolean = false) {
    if (getExtDatabaseStorageUnitLabel) {
        const extLabel = getExtDatabaseStorageUnitLabel(databaseType, singular);
        if (extLabel !== null) {
            return extLabel;
        }
    }

    switch(databaseType) {
        case DatabaseType.ElasticSearch:
            return singular ? "Index" : "Indices";
        case DatabaseType.MongoDb:
            return singular ? "Collection" : "Collections";
        case DatabaseType.Redis:
            return singular ? "Key" : "Keys";
        case DatabaseType.Memcached:
            return singular ? "Item" : "Items";
        case DatabaseType.MySql:
        case DatabaseType.Postgres:
        case DatabaseType.MariaDb:
        case DatabaseType.TiDb:
        case DatabaseType.Sqlite3:
        case DatabaseType.DuckDb:
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
