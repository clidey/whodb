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

import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { TypeCategory } from '../config/database-types';
import { AuthActions } from './auth';

/**
 * TypeDefinition from backend - matches the GraphQL schema
 */
export interface BackendTypeDefinition {
    id: string;
    label: string;
    hasLength: boolean;
    hasPrecision: boolean;
    defaultLength?: number | null;
    defaultPrecision?: number | null;
    category: TypeCategory;
}

/**
 * Capabilities from backend - declares which optional features a plugin supports.
 */
export interface BackendCapabilities {
    supportsScratchpad: boolean;
    supportsChat: boolean;
    supportsGraph: boolean;
    supportsSchema: boolean;
    supportsDatabaseSwitch: boolean;
    supportsModifiers: boolean;
}

/**
 * DatabaseMetadata from backend - matches the GraphQL schema
 */
export interface BackendDatabaseMetadata {
    databaseType: string;
    typeDefinitions: BackendTypeDefinition[];
    operators: string[];
    aliasMap: { Key: string; Value: string }[];
    capabilities: BackendCapabilities;
}

/**
 * State for database metadata fetched from backend
 */
export interface IDatabaseMetadataState {
    /** The database type this metadata belongs to */
    databaseType: string | null;
    /** Type definitions for the current database */
    typeDefinitions: BackendTypeDefinition[];
    /** Valid operators for the current database */
    operators: string[];
    /** Alias map (key: alias, value: canonical name) */
    aliasMap: Record<string, string>;
    /** Capabilities declared by the backend plugin */
    capabilities: BackendCapabilities | null;
    /** Timestamp of last fetch */
    lastFetched: number | null;
    /** Whether metadata is currently being fetched */
    loading: boolean;
}

const initialState: IDatabaseMetadataState = {
    databaseType: null,
    typeDefinitions: [],
    operators: [],
    aliasMap: {},
    capabilities: null,
    lastFetched: null,
    loading: false,
};

export const databaseMetadataSlice = createSlice({
    name: 'databaseMetadata',
    initialState,
    reducers: {
        setLoading: (state, action: PayloadAction<boolean>) => {
            state.loading = action.payload;
        },
        setMetadata: (state, action: PayloadAction<BackendDatabaseMetadata>) => {
            state.databaseType = action.payload.databaseType;
            state.typeDefinitions = action.payload.typeDefinitions;
            state.operators = action.payload.operators;
            // Convert array of {Key, Value} to Record<string, string>
            state.aliasMap = action.payload.aliasMap.reduce((acc, item) => {
                acc[item.Key] = item.Value;
                return acc;
            }, {} as Record<string, string>);
            state.capabilities = action.payload.capabilities;
            state.lastFetched = Date.now();
            state.loading = false;
        },
        clearMetadata: (state) => {
            state.databaseType = null;
            state.typeDefinitions = [];
            state.operators = [];
            state.aliasMap = {};
            state.capabilities = null;
            state.lastFetched = null;
            state.loading = false;
        },
    },
    extraReducers: (builder) => {
        // Clear metadata when user logs out - this ensures cleanup happens
        // synchronously in the same dispatch cycle, not relying on React effects
        builder.addCase(AuthActions.logout, (state) => {
            state.databaseType = null;
            state.typeDefinitions = [];
            state.operators = [];
            state.aliasMap = {};
            state.capabilities = null;
            state.lastFetched = null;
            state.loading = false;
        });
    },
});

export const DatabaseMetadataActions = databaseMetadataSlice.actions;
export const databaseMetadataReducers = databaseMetadataSlice.reducer;

/**
 * Cache duration for metadata (5 minutes)
 */
export const METADATA_CACHE_DURATION = 5 * 60 * 1000;

/**
 * Check if metadata needs to be refreshed
 */
export function shouldRefreshMetadata(
    state: IDatabaseMetadataState,
    currentDbType: string
): boolean {
    // Refresh if no metadata, different db type, or cache expired
    if (state.databaseType !== currentDbType) {
        return true;
    }
    if (!state.lastFetched) {
        return true;
    }
    return Date.now() - state.lastFetched > METADATA_CACHE_DURATION;
}
