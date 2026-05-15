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

import type { ErrorLike } from "@apollo/client";
import { useQuery } from "@apollo/client/react";
import { useEffect, useMemo, useState } from "react";
import { SourceTypesDocument } from "@graphql";
import {
    BackendSourceType,
    findSourceTypeItem,
    readCachedSourceCatalog,
    resolveSourceConnector,
    resolveSourceTypeItems,
    SourceTypeFilterOptions,
    SourceTypeItem,
    writeCachedSourceCatalog,
} from "../config/source-types";

/**
 * Result of resolving the source catalog for UI consumers.
 */
export interface UseSourceTypeItemsResult {
    /** Decorated, filtered source picker items. */
    items: SourceTypeItem[];
    /** Whether the live catalog query is still loading without any cached data. */
    loading: boolean;
    /** Query error, if the live fetch failed. */
    error?: ErrorLike;
}

/**
 * Result of resolving a single source catalog item.
 */
export interface UseSourceTypeItemResult extends UseSourceTypeItemsResult {
    /** Matching catalog item for the requested source type. */
    item?: SourceTypeItem;
}

/**
 * Loads the source catalog with a React/Apollo-owned lifecycle.
 *
 * The backend remains the source of truth. The frontend only reuses the
 * version-scoped local cache as initial data between app launches.
 *
 * @param options Optional UI filters for the returned source list.
 * @returns Decorated source type items plus loading/error state.
 */
export function useSourceTypeItems(
    options: SourceTypeFilterOptions = {}
): UseSourceTypeItemsResult {
    const [cachedCatalog] = useState<BackendSourceType[]>(() => readCachedSourceCatalog());
    const { data, loading, error } = useQuery(SourceTypesDocument, {
        fetchPolicy: import.meta.env.DEV ? "network-only" : "cache-and-network",
        nextFetchPolicy: "cache-first",
    });
    const cloudProvidersEnabled = options.cloudProvidersEnabled;
    const awsProviderEnabled = options.awsProviderEnabled;

    useEffect(() => {
        if (data?.SourceTypes) {
            writeCachedSourceCatalog(data.SourceTypes);
        }
    }, [data?.SourceTypes]);

    const items = useMemo(() => {
        const catalog = data?.SourceTypes ?? cachedCatalog;
        return resolveSourceTypeItems(catalog, { cloudProvidersEnabled, awsProviderEnabled });
    }, [awsProviderEnabled, cachedCatalog, cloudProvidersEnabled, data?.SourceTypes]);

    return {
        items,
        loading: loading && data?.SourceTypes == null && cachedCatalog.length === 0,
        error: error ?? undefined,
    };
}

/**
 * Resolves a single source catalog item by id.
 *
 * @param sourceType Source type identifier.
 * @param options Optional UI filters for catalog resolution.
 * @returns Matching source item plus loading/error state.
 */
export function useSourceTypeItem(
    sourceType: string | undefined,
    options: SourceTypeFilterOptions = {}
): UseSourceTypeItemResult {
    const result = useSourceTypeItems(options);

    const item = useMemo(() => {
        return findSourceTypeItem(result.items, sourceType);
    }, [result.items, sourceType]);

    return {
        ...result,
        item,
    };
}

/**
 * Resolves a displayed source type to its underlying connector id.
 *
 * @param sourceType Source type identifier.
 * @param options Optional UI filters for catalog resolution.
 * @returns The resolved connector id, or the original type if no catalog entry is available.
 */
export function useResolvedSourceConnector(
    sourceType: string | undefined,
    options: SourceTypeFilterOptions = {}
): string | undefined {
    const { item } = useSourceTypeItem(sourceType, options);
    return resolveSourceConnector(sourceType, item);
}
