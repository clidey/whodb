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

import { WhereCondition, WhereConditionType } from '@graphql';

/**
 * Escapes a SQL string value by replacing single quotes with two single quotes
 */
function escapeSqlString(value: string): string {
    return value.replace(/'/g, "''");
}

/**
 * Converts a WhereCondition object to a SQL WHERE clause string
 * @param condition The WhereCondition to convert
 * @returns SQL WHERE clause string (without the "WHERE" keyword)
 */
export function whereConditionToSql(condition: WhereCondition | undefined): string {
    if (!condition) {
        return '';
    }

    switch (condition.Type) {
        case WhereConditionType.Atomic:
            if (!condition.Atomic) return '';
            const { Key, Operator, Value } = condition.Atomic;

            // Handle NULL comparisons
            if (Value.toUpperCase() === 'NULL') {
                if (Operator === '=') return `${Key} IS NULL`;
                if (Operator === '!=') return `${Key} IS NOT NULL`;
            }

            // For numeric types, don't quote the value
            const isNumeric = !isNaN(Number(Value)) && Value.trim() !== '';
            const formattedValue = isNumeric ? Value : `'${escapeSqlString(Value)}'`;

            return `${Key} ${Operator} ${formattedValue}`;

        case WhereConditionType.And:
            if (!condition.And?.Children || condition.And.Children.length === 0) return '';
            const andClauses = condition.And.Children
                .map(child => whereConditionToSql(child))
                .filter(clause => clause !== '');
            if (andClauses.length === 0) return '';
            if (andClauses.length === 1) return andClauses[0];
            return `(${andClauses.join(' AND ')})`;

        case WhereConditionType.Or:
            if (!condition.Or?.Children || condition.Or.Children.length === 0) return '';
            const orClauses = condition.Or.Children
                .map(child => whereConditionToSql(child))
                .filter(clause => clause !== '');
            if (orClauses.length === 0) return '';
            if (orClauses.length === 1) return orClauses[0];
            return `(${orClauses.join(' OR ')})`;

        default:
            return '';
    }
}
