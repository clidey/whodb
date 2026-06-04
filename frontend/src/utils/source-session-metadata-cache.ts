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

import { makeVar } from '@apollo/client';
import { useReactiveVar } from '@apollo/client/react';
import type { SourceSessionMetadataQuery } from '@graphql';
import type { TypeDefinition } from '../config/source-types';

type SourceSessionMetadataPayload = NonNullable<SourceSessionMetadataQuery['SourceSessionMetadata']>;

/**
 * Session-scoped source metadata stored in Apollo state.
 */
export interface SourceSessionMetadataState {
    /** The source type this metadata belongs to. */
    sourceType: string | null;
    /** Canonical type definitions for the active source. */
    typeDefinitions: TypeDefinition[];
    /** Valid operators for the active source. */
    operators: string[];
    /** Alias map for normalizing type names. */
    aliasMap: Record<string, string>;
    /** Query languages supported by the active source session. */
    queryLanguages: string[];
    /** Timestamp of the last successful fetch. */
    lastFetched: number | null;
    /** Whether metadata is currently being fetched. */
    loading: boolean;
}

/**
 * Cache duration for source session metadata (5 minutes).
 */
export const METADATA_CACHE_DURATION = 5 * 60 * 1000;

function createInitialSourceSessionMetadataState(): SourceSessionMetadataState {
    return {
        sourceType: null,
        typeDefinitions: [],
        operators: [],
        aliasMap: {},
        queryLanguages: [],
        lastFetched: null,
        loading: false,
    };
}

function mapTypeDefinitions(typeDefinitions: SourceSessionMetadataPayload['typeDefinitions']): TypeDefinition[] {
    return typeDefinitions.map(typeDefinition => ({
        id: typeDefinition.id,
        label: typeDefinition.label,
        hasLength: typeDefinition.hasLength || undefined,
        hasPrecision: typeDefinition.hasPrecision || undefined,
        defaultLength: typeDefinition.defaultLength ?? undefined,
        defaultPrecision: typeDefinition.defaultPrecision ?? undefined,
        category: typeDefinition.category,
    }));
}

function mapAliasMap(aliasMap: SourceSessionMetadataPayload['aliasMap']): Record<string, string> {
    return aliasMap.reduce((acc, item) => {
        acc[item.Key] = item.Value;
        return acc;
    }, {} as Record<string, string>);
}

const sourceSessionMetadataStateVar = makeVar<SourceSessionMetadataState>(createInitialSourceSessionMetadataState());

/**
 * Reads the current session-scoped source metadata snapshot.
 *
 * @returns Current Apollo-backed source metadata state.
 */
export function getSourceSessionMetadataState(): SourceSessionMetadataState {
    return sourceSessionMetadataStateVar();
}

/**
 * Subscribes a component to Apollo-backed source metadata updates.
 *
 * @returns Current Apollo-backed source metadata state.
 */
export function useSourceSessionMetadataState(): SourceSessionMetadataState {
    return useReactiveVar(sourceSessionMetadataStateVar);
}

/**
 * Updates the in-memory loading flag for source metadata.
 *
 * @param loading Whether a metadata request is in flight.
 */
export function setSourceSessionMetadataLoading(loading: boolean): void {
    const currentState = sourceSessionMetadataStateVar();

    if (currentState.loading === loading) {
        return;
    }

    sourceSessionMetadataStateVar({
        ...currentState,
        loading,
    });
}

/**
 * Writes a fresh source session metadata payload into Apollo state.
 *
 * @param metadata GraphQL metadata payload returned by the backend.
 */
export function setSourceSessionMetadata(metadata: SourceSessionMetadataPayload): void {
    sourceSessionMetadataStateVar({
        sourceType: metadata.sourceType,
        queryLanguages: metadata.queryLanguages,
        typeDefinitions: mapTypeDefinitions(metadata.typeDefinitions),
        operators: metadata.operators,
        aliasMap: mapAliasMap(metadata.aliasMap),
        lastFetched: Date.now(),
        loading: false,
    });
}

/**
 * Clears the Apollo-backed source metadata snapshot.
 */
export function clearSourceSessionMetadata(): void {
    sourceSessionMetadataStateVar(createInitialSourceSessionMetadataState());
}

/**
 * Determines whether the current session metadata should be refetched.
 *
 * @param currentSourceType Active source type for the current session.
 * @returns True when metadata is missing, stale, or for a different source.
 */
export function shouldRefreshSourceSessionMetadata(currentSourceType: string): boolean {
    const state = sourceSessionMetadataStateVar();

    if (state.loading) {
        return false;
    }

    if (state.sourceType !== currentSourceType) {
        return true;
    }

    if (!state.lastFetched) {
        return true;
    }

    return Date.now() - state.lastFetched > METADATA_CACHE_DURATION;
}
