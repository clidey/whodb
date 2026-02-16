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

import { WhereCondition, WhereConditionType } from "@graphql";

/**
 * Parsed search query result
 */
export type ParsedSearch = {
    type: 'column-specific' | 'full-text' | 'compound';
    column?: string;
    operator?: string;
    value?: string;
    rawValue?: string;
    compoundOperator?: 'AND' | 'OR';
    conditions?: ParsedSearch[];
};

/**
 * Parse a search string into structured query components
 * Supports:
 * - Column-specific: "id=1", "name = Alice", "age > 18"
 * - Wildcards: "name=%Alice%", "email LIKE %@gmail.com"
 * - Compound: "id=1 AND name=Alice", "age>18 OR status=active"
 * - Full-text: "Alice" (searches all columns)
 */
export function parseSearchString(search: string): ParsedSearch {
    const trimmed = search.trim();

    if (!trimmed) {
        return { type: 'full-text', rawValue: '' };
    }

    // Check for compound conditions (AND/OR)
    // Split by AND/OR while preserving which operator was used
    const compoundMatch = splitByLogicalOperators(trimmed);

    if (compoundMatch && compoundMatch.parts.length > 1) {
        // Parse each part as an atomic condition, ensuring each part is trimmed
        const conditions = compoundMatch.parts
            .map(part => part.trim())
            .filter(part => part.length > 0)
            .map(part => parseAtomicCondition(part));

        // If all parts are valid column-specific conditions, create compound
        if (conditions.every(c => c.type === 'column-specific')) {
            return {
                type: 'compound',
                compoundOperator: compoundMatch.operator,
                conditions,
                rawValue: trimmed
            };
        }
        // Otherwise fall back to full-text search
        return { type: 'full-text', rawValue: trimmed };
    }

    // No compound operators - parse as atomic condition
    return parseAtomicCondition(trimmed);
}

/**
 * Parse a single atomic condition (no AND/OR)
 */
function parseAtomicCondition(search: string): ParsedSearch {
    const trimmed = search.trim();

    if (!trimmed) {
        return { type: 'full-text', rawValue: '' };
    }

    // Match column-specific patterns: column operator value
    // Operators: =, !=, <>, >, <, >=, <=, LIKE, NOT LIKE, IN, NOT IN, IS, IS NOT
    const columnPattern = /^([a-zA-Z_][a-zA-Z0-9_]*)\s*(=|!=|<>|>=|<=|>|<|LIKE|NOT\s+LIKE|IN|NOT\s+IN|IS|IS\s+NOT)\s*(.+)$/i;
    const match = trimmed.match(columnPattern);

    if (match) {
        const [, column, operator, value] = match;
        // Trim all components to handle cases like "likes_count=42 " with trailing space
        const trimmedColumn = column.trim();
        const trimmedOperator = operator.trim().toUpperCase();
        const trimmedValue = value.trim();

        // Validate that we have non-empty values after trimming
        if (!trimmedColumn || !trimmedOperator || !trimmedValue) {
            return { type: 'full-text', rawValue: trimmed };
        }

        return {
            type: 'column-specific',
            column: trimmedColumn,
            operator: trimmedOperator,
            value: trimmedValue,
            rawValue: trimmed
        };
    }

    // No column pattern found - treat as full-text search
    return {
        type: 'full-text',
        rawValue: trimmed
    };
}

/**
 * Split search string by AND/OR operators
 * Returns null if no operators found, otherwise returns parts and the operator used
 */
function splitByLogicalOperators(search: string): { operator: 'AND' | 'OR'; parts: string[] } | null {
    // Look for AND operator (must be surrounded by spaces or at boundaries)
    const andPattern = /\s+AND\s+/i;
    const orPattern = /\s+OR\s+/i;

    const hasAnd = andPattern.test(search);
    const hasOr = orPattern.test(search);

    if (hasAnd && hasOr) {
        // Both AND and OR - not supported yet, treat as full-text
        return null;
    }

    if (hasAnd) {
        const parts = search.split(andPattern);
        if (parts.length > 1) {
            return { operator: 'AND', parts };
        }
    }

    if (hasOr) {
        const parts = search.split(orPattern);
        if (parts.length > 1) {
            return { operator: 'OR', parts };
        }
    }

    return null;
}

/**
 * Convert parsed search into a WhereCondition
 * @param parsed - Parsed search result
 * @param columns - Available column names for full-text search
 * @param columnTypes - Column types (for type inference)
 * @param validOperators - Valid operators for the database
 */
export function parseSearchToWhereCondition(
    search: string,
    columns: string[],
    columnTypes?: (string | undefined)[],
    validOperators?: string[]
): WhereCondition | undefined {
    const parsed = parseSearchString(search);

    if (!parsed.rawValue) {
        return undefined;
    }

    // Handle compound conditions (AND/OR)
    if (parsed.type === 'compound' && parsed.conditions && parsed.compoundOperator) {
        const atomicConditions: WhereCondition[] = [];

        // Convert each parsed condition to a WhereCondition
        for (const condition of parsed.conditions) {
            if (condition.type === 'column-specific' && condition.column && condition.operator && condition.value) {
                const atomicCondition = createAtomicCondition(
                    condition.column,
                    condition.operator,
                    condition.value,
                    columns,
                    columnTypes,
                    validOperators
                );

                if (atomicCondition) {
                    atomicConditions.push(atomicCondition);
                } else {
                    // If any condition is invalid, fall back to full-text search
                    return createFullTextCondition(parsed.rawValue!, columns, columnTypes);
                }
            }
        }

        // Create compound condition
        if (atomicConditions.length === 0) {
            return undefined;
        }

        if (atomicConditions.length === 1) {
            return atomicConditions[0];
        }

        if (parsed.compoundOperator === 'AND') {
            return {
                Type: WhereConditionType.And,
                And: {
                    Children: atomicConditions
                }
            };
        } else {
            return {
                Type: WhereConditionType.Or,
                Or: {
                    Children: atomicConditions
                }
            };
        }
    }

    if (parsed.type === 'column-specific' && parsed.column && parsed.operator && parsed.value) {
        return createAtomicCondition(
            parsed.column,
            parsed.operator,
            parsed.value,
            columns,
            columnTypes,
            validOperators
        ) || createFullTextCondition(parsed.rawValue!, columns, columnTypes);
    }

    // Full-text search across all text columns
    return createFullTextCondition(parsed.rawValue!, columns, columnTypes);
}

/**
 * Create an atomic WHERE condition from column, operator, and value
 */
function createAtomicCondition(
    column: string,
    operator: string,
    value: string,
    columns: string[],
    columnTypes?: (string | undefined)[],
    validOperators?: string[]
): WhereCondition | undefined {
    // Ensure all inputs are trimmed
    const trimmedColumn = column.trim();
    const trimmedOperator = operator.trim();
    const trimmedValue = value.trim();

    // Validate non-empty after trimming
    if (!trimmedColumn || !trimmedOperator || !trimmedValue) {
        return undefined;
    }

    // Validate column exists
    const columnIndex = columns.findIndex(col =>
        col.toLowerCase() === trimmedColumn.toLowerCase()
    );

    if (columnIndex === -1) {
        return undefined;
    }

    // Validate operator if validOperators provided
    if (validOperators && !validOperators.includes(trimmedOperator)) {
        return undefined;
    }

    const actualColumn = columns[columnIndex];
    const columnType = columnTypes?.[columnIndex] || 'string';

    // Clean up value - remove quotes if present
    let cleanValue = trimmedValue;
    if ((cleanValue.startsWith("'") && cleanValue.endsWith("'")) ||
        (cleanValue.startsWith('"') && cleanValue.endsWith('"'))) {
        cleanValue = cleanValue.slice(1, -1);
    }

    // Handle wildcards in LIKE operator
    if (trimmedOperator === 'LIKE' || trimmedOperator === 'NOT LIKE') {
        // Keep wildcards as-is (%, _)
        // User can specify: name=%Alice% or name=Alice (we'll add % % for them)
        if (!cleanValue.includes('%') && !cleanValue.includes('_')) {
            cleanValue = `%${cleanValue}%`;
        }
    }

    return {
        Type: WhereConditionType.Atomic,
        Atomic: {
            Key: actualColumn,
            Operator: trimmedOperator,
            Value: cleanValue,
            ColumnType: inferColumnType(columnType)
        }
    };
}

/**
 * Create a full-text search condition that searches across all suitable columns
 */
function createFullTextCondition(
    searchText: string,
    columns: string[],
    columnTypes?: (string | undefined)[]
): WhereCondition | undefined {
    // Filter to searchable columns (text-based types)
    const searchableColumns = columns.filter((_, index) => {
        const type = columnTypes?.[index]?.toLowerCase() || '';
        return isTextType(type);
    });

    if (searchableColumns.length === 0) {
        // No searchable columns - search all columns
        return createOrCondition(columns, searchText, columnTypes);
    }

    return createOrCondition(searchableColumns, searchText, columnTypes);
}

/**
 * Create an OR condition across multiple columns
 */
function createOrCondition(
    columns: string[],
    searchText: string,
    columnTypes?: (string | undefined)[]
): WhereCondition | undefined {
    if (columns.length === 0) {
        return undefined;
    }

    const conditions: WhereCondition[] = columns.map((column, index) => {
        const columnType = columnTypes?.[columns.indexOf(column)] || 'string';

        return {
            Type: WhereConditionType.Atomic,
            Atomic: {
                Key: column,
                Operator: 'LIKE',
                Value: `%${searchText}%`,
                ColumnType: inferColumnType(columnType)
            }
        };
    });

    if (conditions.length === 1) {
        return conditions[0];
    }

    return {
        Type: WhereConditionType.Or,
        Or: {
            Children: conditions
        }
    };
}

/**
 * Check if a column type is text-based and searchable
 */
function isTextType(type: string): boolean {
    const textTypes = [
        'text', 'varchar', 'char', 'string', 'nvarchar', 'nchar',
        'clob', 'longtext', 'mediumtext', 'tinytext',
        'character', 'varying'
    ];

    return textTypes.some(t => type.includes(t));
}

/**
 * Infer a simple column type for the GraphQL WhereCondition
 */
function inferColumnType(dbType: string): string {
    const lower = dbType.toLowerCase();

    if (lower.includes('int') || lower.includes('number') || lower.includes('decimal') ||
        lower.includes('float') || lower.includes('double') || lower.includes('numeric')) {
        return 'number';
    }

    if (lower.includes('bool')) {
        return 'boolean';
    }

    if (lower.includes('date') || lower.includes('time')) {
        return 'date';
    }

    return 'string';
}

/**
 * Merge search condition with existing where condition
 */
export function mergeSearchWithWhere(
    searchCondition: WhereCondition | undefined,
    whereCondition: WhereCondition | undefined
): WhereCondition | undefined {
    if (!searchCondition && !whereCondition) {
        return undefined;
    }

    if (!searchCondition) {
        return whereCondition;
    }

    if (!whereCondition) {
        return searchCondition;
    }

    // Both exist - create AND condition
    return {
        Type: WhereConditionType.And,
        And: {
            Children: [whereCondition, searchCondition]
        }
    };
}
