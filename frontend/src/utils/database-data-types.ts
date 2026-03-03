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

import { DatabaseType } from '@graphql';
import { getDatabaseTypeDropdownItemsSync, TypeDefinition } from '../config/database-types';
import { reduxStore } from '../store';

/**
 * Get type definitions for a database from the backend-driven Redux store.
 *
 * @param databaseType The database type (can be CE or EE type)
 * @returns Array of TypeDefinition objects for the database
 */
export function getDatabaseTypeDefinitions(databaseType: DatabaseType | string): TypeDefinition[] {
    const metadataState = reduxStore.getState().databaseMetadata;

    if (
        metadataState.databaseType === databaseType &&
        metadataState.typeDefinitions.length > 0
    ) {
        // Convert backend format to frontend TypeDefinition format
        return metadataState.typeDefinitions.map(td => ({
            id: td.id,
            label: td.label,
            hasLength: td.hasLength || undefined,
            hasPrecision: td.hasPrecision || undefined,
            defaultLength: td.defaultLength ?? undefined,
            defaultPrecision: td.defaultPrecision ?? undefined,
            category: td.category,
        }));
    }

    // No fallback - backend is the source of truth.
    console.warn(
        `[database-data-types] No type definitions found for ${databaseType}. ` +
            `Ensure DatabaseMetadata query has completed.`
    );
    return [];
}

/**
 * Get the alias map for a database from the backend-driven Redux store.
 *
 * @param databaseType The database type (can be CE or EE type)
 * @returns Record mapping aliases to canonical type names
 */
export function getDatabaseAliasMap(databaseType: DatabaseType | string): Record<string, string> {
    const metadataState = reduxStore.getState().databaseMetadata;

    if (
        metadataState.databaseType === databaseType &&
        Object.keys(metadataState.aliasMap).length > 0
    ) {
        return metadataState.aliasMap;
    }

    // Note: Empty alias map is valid for some databases, so only warn if databaseType mismatch
    if (metadataState.databaseType !== databaseType) {
        console.warn(
            `[database-data-types] No alias map found for ${databaseType}. ` +
                `Ensure DatabaseMetadata query has completed.`
        );
    }
    return {};
}

/**
 * Normalize a type name to its canonical form for a specific database
 * @param typeName The type name to normalize (may include length, e.g., "VARCHAR(255)")
 * @param databaseType The database type
 * @returns The canonical type name (without length specifier)
 */
export function normalizeTypeName(typeName: string, databaseType: DatabaseType | string): string {
    // Strip length/precision specifier: "VARCHAR(255)" -> "VARCHAR"
    const baseType = typeName.replace(/\(.*\)$/, '').trim().toUpperCase();

    // Get alias map for this database
    const aliasMap = getDatabaseAliasMap(databaseType);

    // Return canonical form if alias exists, otherwise return the base type
    return aliasMap[baseType] ?? baseType;
}

/**
 * Find a type definition by its id or alias
 * @param typeId The type id or alias to find
 * @param databaseType The database type
 * @returns The TypeDefinition or undefined if not found
 */
export function findTypeDefinition(typeId: string, databaseType: DatabaseType | string): TypeDefinition | undefined {
    const typeDefs = getDatabaseTypeDefinitions(databaseType);
    const normalizedType = normalizeTypeName(typeId, databaseType);

    // First try exact match
    let typeDef = typeDefs.find(t => t.id.toUpperCase() === normalizedType);

    // If not found, try case-insensitive match
    if (!typeDef) {
        typeDef = typeDefs.find(t => t.id.toUpperCase() === typeId.toUpperCase());
    }

    return typeDef;
}

/**
 * Check if a database supports field modifiers (primary, nullable)
 * @param databaseType The database type (can be CE or EE type)
 * @returns boolean indicating if the database supports modifiers
 */
export function databaseSupportsModifiers(databaseType: DatabaseType | string): boolean {
    const dbTypeItems = getDatabaseTypeDropdownItemsSync();
    const dbConfig = dbTypeItems.find(item => item.id === databaseType);

    // Return from config if defined, otherwise false
    return dbConfig?.supportsModifiers ?? false;
}

/**
 * Parse a type specification into its components
 * @param fullType The full type string (e.g., "VARCHAR(255)", "DECIMAL(10,2)")
 * @returns Object with baseType, length, precision, scale
 */
export function parseTypeSpec(fullType: string): {
    baseType: string;
    length?: number;
    precision?: number;
    scale?: number;
} {
    const match = fullType.match(/^([A-Za-z0-9_ ]+)(?:\((\d+)(?:,\s*(\d+))?\))?$/);

    if (!match) {
        return { baseType: fullType.trim() };
    }

    const baseType = match[1].trim();
    const firstNum = match[2] ? parseInt(match[2], 10) : undefined;
    const secondNum = match[3] ? parseInt(match[3], 10) : undefined;

    // If there's a second number, it's precision/scale (DECIMAL(10,2))
    // Otherwise it's just length (VARCHAR(255))
    if (secondNum !== undefined) {
        return { baseType, precision: firstNum, scale: secondNum };
    }

    return { baseType, length: firstNum };
}

/**
 * Format a type specification into a full type string
 * @param baseType The base type name
 * @param length Optional length for character types
 * @param precision Optional precision for decimal types
 * @param scale Optional scale for decimal types
 * @returns The formatted type string (e.g., "VARCHAR(255)", "DECIMAL(10,2)")
 */
export function formatTypeSpec(
    baseType: string,
    length?: number,
    precision?: number,
    scale?: number
): string {
    if (precision !== undefined) {
        if (scale !== undefined) {
            return `${baseType}(${precision},${scale})`;
        }
        return `${baseType}(${precision})`;
    }
    if (length !== undefined) {
        return `${baseType}(${length})`;
    }
    return baseType;
}
