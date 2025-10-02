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
import { DatabaseType } from '@graphql';

export function isNumeric(str: string) {
    return !isNaN(Number(str));
}

export function createStub(name: string) {
    return name.split(" ").map(word => word.toLowerCase()).join("-");
}

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

export function isValidJSON(str: string): boolean {
    // this allows it to start showing intellisense when it starts with {
    // even if it is not valid - better UX?
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

export function isNoSQL(databaseType: string) {
    // Check EE databases first if EE is enabled
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

export function getDatabaseStorageUnitLabel(databaseType: string | undefined, singular: boolean = false) {
    // Check EE databases first if EE is enabled
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

export function chooseRandomItems<T>(array: T[], n: number = 3): T[] {
    if (n > array.length) {
        throw new Error("n cannot be greater than the array length");
    }

    const shuffledArray = [...array];
    for (let i = shuffledArray.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [shuffledArray[i], shuffledArray[j]] = [shuffledArray[j], shuffledArray[i]];
    }

    return shuffledArray.slice(0, n);
}
