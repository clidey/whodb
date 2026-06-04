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

import { useLazyQuery } from '@apollo/client/react';
import { useEffect, useCallback } from 'react';
import { SourceSessionMetadataDocument } from '@graphql';
import { useAppSelector } from '../store/hooks';
import {
    clearSourceSessionMetadata,
    setSourceSessionMetadata,
    setSourceSessionMetadataLoading,
    shouldRefreshSourceSessionMetadata,
    useSourceSessionMetadataState,
} from '../utils/source-session-metadata-cache';

/**
 * Hook that fetches and caches source session metadata from the backend.
 *
 * Use this hook in components that need access to source session metadata.
 * The metadata is automatically fetched when:
 * - User logs in (source type changes)
 * - Cache expires (5 minutes)
 * - Manual refresh is triggered
 *
 * @returns Object with metadata state and refresh function
 */
export const useSourceSessionMetadata = () => {
    const authStatus = useAppSelector(state => state.auth.status);
    const currentSourceType = useAppSelector(state => state.auth.current?.Type);
    const metadata = useSourceSessionMetadataState();

    const [fetchMetadata, { data, error, loading }] = useLazyQuery(SourceSessionMetadataDocument, {
        fetchPolicy: 'network-only',
    });

    useEffect(() => {
        if (data?.SourceSessionMetadata) {
            setSourceSessionMetadata(data.SourceSessionMetadata);
        }
    }, [data?.SourceSessionMetadata]);

    useEffect(() => {
        if (error) {
            console.error('Failed to fetch source session metadata:', error);
            setSourceSessionMetadataLoading(false);
        }
    }, [error]);

    // Fetch metadata when source type changes or the session cache expires.
    useEffect(() => {
        if (authStatus === 'logged-in' && currentSourceType) {
            if (shouldRefreshSourceSessionMetadata(currentSourceType)) {
                setSourceSessionMetadataLoading(true);
                void fetchMetadata();
            }
        }
    }, [authStatus, currentSourceType, fetchMetadata, metadata.lastFetched, metadata.sourceType]);

    // Clear metadata on logout.
    useEffect(() => {
        if (authStatus === 'unauthorized') {
            clearSourceSessionMetadata();
        }
    }, [authStatus]);

    // Manual refresh function.
    const refresh = useCallback(() => {
        if (authStatus === 'logged-in') {
            setSourceSessionMetadataLoading(true);
            void fetchMetadata();
        }
    }, [authStatus, fetchMetadata]);

    return {
        queryLanguages: metadata.queryLanguages,
        typeDefinitions: metadata.typeDefinitions,
        operators: metadata.operators,
        aliasMap: metadata.aliasMap,
        sourceType: metadata.sourceType,
        loading: loading || metadata.loading,
        hasFetched: metadata.lastFetched !== null,
        refresh,
    };
};
