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
import { databaseTypeDropdownItems, getDatabaseTypeDropdownItemsSync } from '../config/database-types';

/**
 * Get valid operators for a database type
 * @param databaseType The database type (can be CE or EE type)
 * @returns Array of valid operators for the database
 */
export function getDatabaseOperators(databaseType: DatabaseType | string): string[] {
    // Try to get operators from the database configuration first
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);
    
    if (dbConfig?.operators) {
        return dbConfig.operators;
    }
    
    // Fall back to built-in operators for known database types
    switch (databaseType) {
        case DatabaseType.ElasticSearch:
            return [
                "match", "match_phrase", "match_phrase_prefix", "multi_match", "bool", 
                "term", "terms", "range", "exists", "prefix", "wildcard", "regexp", 
                "fuzzy", "ids", "constant_score", "function_score", "dis_max", "nested", 
                "has_child", "has_parent"
            ];
        case DatabaseType.MongoDb:
            return ["eq", "ne", "gt", "gte", "lt", "lte", "in", "nin", "and", "or", 
                    "not", "nor", "exists", "type", "regex", "expr", "mod", "all", 
                    "elemMatch", "size", "bitsAllClear", "bitsAllSet", "bitsAnyClear", 
                    "bitsAnySet", "geoIntersects", "geoWithin", "near", "nearSphere"];
        case DatabaseType.ClickHouse:
            return [
                "=", ">=", ">", "<=", "<", "!=", "<>", "==",
                "LIKE", "NOT LIKE", "ILIKE",  // ILIKE is handled specially for ClickHouse
                "IN", "NOT IN", "GLOBAL IN", "GLOBAL NOT IN",
                "BETWEEN", "NOT BETWEEN",
                "IS NULL", "IS NOT NULL",
                "AND", "OR", "NOT"
            ];
        default:
            // Default SQL operators
            return [
                "=", ">=", ">", "<=", "<", "<>", "!=", "!>", "!<", "BETWEEN", "NOT BETWEEN", 
                "LIKE", "NOT LIKE", "IN", "NOT IN", "IS NULL", "IS NOT NULL", "AND", "OR", 
                "NOT"
            ];
    }
}