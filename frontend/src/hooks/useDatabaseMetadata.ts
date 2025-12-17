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

import { useEffect, useCallback } from 'react';
import { useGetDatabaseMetadataLazyQuery } from '@graphql';
import { useAppDispatch, useAppSelector } from '../store/hooks';
import {
    DatabaseMetadataActions,
    shouldRefreshMetadata,
} from '../store/database-metadata';

/**
 * Hook that fetches and caches database metadata from the backend.
 *
 * Use this hook in components that need access to database metadata.
 * The metadata is automatically fetched when:
 * - User logs in (database type changes)
 * - Cache expires (5 minutes)
 * - Manual refresh is triggered
 *
 * @returns Object with metadata state and refresh function
 */
export const useDatabaseMetadata = () => {
    const dispatch = useAppDispatch();
    const auth = useAppSelector(state => state.auth);
    const metadata = useAppSelector(state => state.databaseMetadata);

    const [fetchMetadata, { loading }] = useGetDatabaseMetadataLazyQuery({
        fetchPolicy: 'network-only',
        onCompleted: (data) => {
            if (data.DatabaseMetadata) {
                dispatch(DatabaseMetadataActions.setMetadata(data.DatabaseMetadata));
            }
        },
        onError: (error) => {
            console.error('Failed to fetch database metadata:', error);
            dispatch(DatabaseMetadataActions.setLoading(false));
        },
    });

    const currentDbType = auth.current?.Type;

    // Fetch metadata when database type changes or cache expires
    useEffect(() => {
        if (auth.status === 'logged-in' && currentDbType) {
            if (shouldRefreshMetadata(metadata, currentDbType)) {
                dispatch(DatabaseMetadataActions.setLoading(true));
                fetchMetadata();
            }
        }
    }, [auth.status, currentDbType, dispatch, fetchMetadata, metadata]);

    // Clear metadata on logout
    useEffect(() => {
        if (auth.status === 'unauthorized') {
            dispatch(DatabaseMetadataActions.clearMetadata());
        }
    }, [auth.status, dispatch]);

    // Manual refresh function
    const refresh = useCallback(() => {
        if (auth.status === 'logged-in') {
            dispatch(DatabaseMetadataActions.setLoading(true));
            fetchMetadata();
        }
    }, [auth.status, dispatch, fetchMetadata]);

    return {
        /** Type definitions for the current database */
        typeDefinitions: metadata.typeDefinitions,
        /** Valid operators for the current database */
        operators: metadata.operators,
        /** Alias map for the current database */
        aliasMap: metadata.aliasMap,
        /** Current database type */
        databaseType: metadata.databaseType,
        /** Whether metadata is being fetched */
        loading: loading || metadata.loading,
        /** Whether metadata has been fetched */
        hasFetched: metadata.lastFetched !== null,
        /** Manually refresh metadata */
        refresh,
    };
};

/**
 * Get database metadata from Redux store (non-hook version for utilities)
 * This function can be used outside of React components.
 *
 * @param store The Redux store
 * @returns The database metadata state
 */
export const getDatabaseMetadataFromStore = (store: { getState: () => { databaseMetadata: ReturnType<typeof import('../store/database-metadata').databaseMetadataSlice.getInitialState> } }) => {
    return store.getState().databaseMetadata;
};
