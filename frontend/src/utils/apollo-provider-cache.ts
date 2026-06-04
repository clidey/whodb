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

import type { ApolloCache } from "@apollo/client";
import type {
    GetAzureProvidersQuery,
    GetCloudProvidersQuery} from "@graphql";
import {
    GetAzureProvidersDocument,
    GetCloudProvidersDocument
} from "@graphql";

type WithId = {
    Id: string;
};

function upsertItemById<T extends WithId>(items: readonly T[] | undefined, item: T): T[] {
    const next = items ? [...items] : [];
    const existingIndex = next.findIndex(existing => existing.Id === item.Id);

    if (existingIndex === -1) {
        next.push(item);
        return next;
    }

    next[existingIndex] = item;
    return next;
}

function removeItemById<T extends WithId>(items: readonly T[] | undefined, id: string): T[] {
    return (items ?? []).filter(item => item.Id !== id);
}

/**
 * Upserts a cloud provider entry in the cached `GetCloudProviders` result.
 *
 * @param cache Apollo cache instance.
 * @param provider Provider payload returned by a cloud provider mutation.
 */
export function upsertCloudProviderCache(
    cache: ApolloCache,
    provider: GetCloudProvidersQuery["CloudProviders"][number]
): void {
    const existing = cache.readQuery<GetCloudProvidersQuery>({
        query: GetCloudProvidersDocument,
    });

    if (!existing) {
        return;
    }

    cache.writeQuery<GetCloudProvidersQuery>({
        query: GetCloudProvidersDocument,
        data: {
            CloudProviders: upsertItemById(existing.CloudProviders, provider),
        },
    });
}

/**
 * Removes a cloud provider entry from the cached `GetCloudProviders` result.
 *
 * @param cache Apollo cache instance.
 * @param id Provider identifier to remove.
 */
export function removeCloudProviderCache(
    cache: ApolloCache,
    id: string
): void {
    const existing = cache.readQuery<GetCloudProvidersQuery>({
        query: GetCloudProvidersDocument,
    });

    if (!existing) {
        return;
    }

    cache.writeQuery<GetCloudProvidersQuery>({
        query: GetCloudProvidersDocument,
        data: {
            CloudProviders: removeItemById(existing.CloudProviders, id),
        },
    });
}

/**
 * Upserts an Azure provider entry in the cached `GetAzureProviders` result.
 *
 * @param cache Apollo cache instance.
 * @param provider Provider payload returned by an Azure provider mutation.
 */
export function upsertAzureProviderCache(
    cache: ApolloCache,
    provider: GetAzureProvidersQuery["AzureProviders"][number]
): void {
    const existing = cache.readQuery<GetAzureProvidersQuery>({
        query: GetAzureProvidersDocument,
    });

    if (!existing) {
        return;
    }

    cache.writeQuery<GetAzureProvidersQuery>({
        query: GetAzureProvidersDocument,
        data: {
            AzureProviders: upsertItemById(existing.AzureProviders, provider),
        },
    });
}

/**
 * Removes an Azure provider entry from the cached `GetAzureProviders` result.
 *
 * @param cache Apollo cache instance.
 * @param id Provider identifier to remove.
 */
export function removeAzureProviderCache(
    cache: ApolloCache,
    id: string
): void {
    const existing = cache.readQuery<GetAzureProvidersQuery>({
        query: GetAzureProvidersDocument,
    });

    if (!existing) {
        return;
    }

    cache.writeQuery<GetAzureProvidersQuery>({
        query: GetAzureProvidersDocument,
        data: {
            AzureProviders: removeItemById(existing.AzureProviders, id),
        },
    });
}
