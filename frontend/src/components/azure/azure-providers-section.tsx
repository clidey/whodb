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

import { useMutation, useQuery } from "@apollo/client/react";
import { FC, useCallback } from "react";
import { Badge, Button, cn, toast } from "@clidey/ux";
import {
    AzureProvider,
    CloudProviderStatus,
    CloudProviderType,
    GetAzureProvidersDocument,
    GetDiscoveredConnectionsDocument,
    RefreshAzureProviderDocument,
    RemoveCloudProviderDocument,
} from "@graphql";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ProvidersActions } from "../../store/providers";
import { useTranslation } from "@/hooks/use-translation";
import { AzureProviderModal } from "./azure-provider-modal";
import { Tip } from "../tip";
import {
    ArrowPathIcon,
    CloudIcon,
    PencilIcon,
    PlusIcon,
    TrashIcon,
} from "../heroicons";
import { removeAzureProviderCache, removeCloudProviderCache, upsertAzureProviderCache, upsertCloudProviderCache } from "../../utils/apollo-provider-cache";

/**
 * Local Azure provider with optional environment-defined flag.
 */
type LocalAzureProvider = AzureProvider & {
    IsEnvironmentDefined?: boolean;
};

/**
 * Returns the appropriate badge variant for a provider status.
 */
function getStatusVariant(status: CloudProviderStatus): "default" | "secondary" | "destructive" | "outline" {
    switch (status) {
        case CloudProviderStatus.Connected:
            return "default";
        case CloudProviderStatus.Discovering:
            return "secondary";
        case CloudProviderStatus.Error:
            return "destructive";
        case CloudProviderStatus.Disconnected:
        default:
            return "outline";
    }
}

/**
 * Azure Providers section for the settings page.
 * Displays configured Azure providers and allows management.
 */
export const AzureProvidersSection: FC = () => {
    const { t } = useTranslation('components/azure-providers-section');
    const dispatch = useAppDispatch();

    // Local state for Azure providers (separate from AWS cloud providers in Redux)
    const azureModalOpen = useAppSelector(state => state.providers.isProviderModalOpen);
    const providerModalType = useAppSelector(state => state.providers.providerModalType);

    // GraphQL queries and mutations
    const { data, loading, refetch } = useQuery(GetAzureProvidersDocument);
    const [refreshProvider, { loading: refreshLoading }] = useMutation(RefreshAzureProviderDocument);
    const [removeProvider, { loading: removeLoading }] = useMutation(RemoveCloudProviderDocument);

    const azureProviders: LocalAzureProvider[] = (data?.AzureProviders as LocalAzureProvider[] | undefined) ?? [];

    const handleAddProvider = useCallback(() => {
        dispatch(ProvidersActions.openAddProviderModal({ providerType: CloudProviderType.Azure }));
    }, [dispatch]);

    const handleEditProvider = useCallback((id: string) => {
        dispatch(ProvidersActions.openEditProviderModal({ id, providerType: CloudProviderType.Azure }));
    }, [dispatch]);

    const handleRemoveProvider = useCallback(async (id: string, name: string) => {
        try {
            const { data } = await removeProvider({
                variables: { id },
                refetchQueries: [GetDiscoveredConnectionsDocument],
                update(cache) {
                    removeAzureProviderCache(cache, id);
                    removeCloudProviderCache(cache, id);
                },
            });
            if (data?.RemoveCloudProvider?.Status) {
                toast.success(t('providerRemoved', { name }));
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('removeFailed', { error: errorMessage }));
        }
    }, [removeProvider, t]);

    const handleRefreshProvider = useCallback(async (id: string) => {
        try {
            const { data } = await refreshProvider({
                variables: { id },
                refetchQueries: [GetDiscoveredConnectionsDocument],
                update(cache, result) {
                    if (result.data?.RefreshAzureProvider) {
                        upsertAzureProviderCache(cache, result.data.RefreshAzureProvider);
                        upsertCloudProviderCache(cache, result.data.RefreshAzureProvider);
                    }
                },
            });
            if (data?.RefreshAzureProvider) {
                toast.success(t('refreshComplete', { count: data.RefreshAzureProvider.DiscoveredCount }));
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('refreshFailed', { error: errorMessage }));
        }
    }, [refreshProvider, t]);

    const handleModalOpenChange = useCallback((open: boolean) => {
        if (!open) {
            dispatch(ProvidersActions.closeProviderModal());
        }
    }, [dispatch]);

    const handleRefetchProviders = useCallback(() => {
        refetch();
    }, [refetch]);

    const isLoading = loading || refreshLoading || removeLoading;

    return (
        <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <CloudIcon className="w-5 h-5" />
                    <h3 className="text-lg font-bold">{t('title')}</h3>
                </div>
                <div className="flex items-center gap-2">
                    <Tip className="w-fit">
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={handleRefetchProviders}
                            disabled={isLoading}
                            aria-label={t('refresh')}
                        >
                            <ArrowPathIcon className={cn("w-4 h-4", { "animate-spin": loading })} />
                        </Button>
                        <p>{t('refresh')}</p>
                    </Tip>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={handleAddProvider}
                        data-testid="add-azure-provider"
                    >
                        <PlusIcon className="w-4 h-4 mr-1" />
                        {t('addProvider')}
                    </Button>
                </div>
            </div>

            <p className="text-sm text-muted-foreground">
                {t('description')}
            </p>

            {azureProviders.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-8 px-4 border border-dashed rounded-lg">
                    <CloudIcon className="w-10 h-10 text-muted-foreground mb-3" />
                    <p className="text-sm text-muted-foreground text-center mb-4">
                        {t('noProviders')}
                    </p>
                    <Button
                        variant="outline"
                        onClick={handleAddProvider}
                        data-testid="add-first-azure-provider"
                    >
                        <PlusIcon className="w-4 h-4 mr-1" />
                        {t('addFirstProvider')}
                    </Button>
                </div>
            ) : (
                <div className="flex flex-col gap-3">
                    {azureProviders.map((provider) => (
                        <div
                            key={provider.Id}
                            className="flex items-center justify-between p-4 border rounded-lg"
                            data-testid={`azure-provider-${provider.Id}`}
                        >
                            <div className="flex flex-col gap-1">
                                <div className="flex items-center gap-2">
                                    <span className="font-medium">{provider.Name}</span>
                                    <Badge variant={getStatusVariant(provider.Status)}>
                                        {t(`status.${provider.Status.toLowerCase()}`)}
                                    </Badge>
                                    {provider.IsEnvironmentDefined && (
                                        <Badge variant="outline" className="text-xs">
                                            {t('envDefined')}
                                        </Badge>
                                    )}
                                </div>
                                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                                    <span>{provider.SubscriptionID}</span>
                                    {provider.ResourceGroup && (
                                        <span>{provider.ResourceGroup}</span>
                                    )}
                                    {provider.DiscoveredCount > 0 && (
                                        <span>{t('resourcesDiscovered', { count: provider.DiscoveredCount })}</span>
                                    )}
                                </div>
                                {provider.LastDiscoveryAt && (
                                    <span className="text-xs text-muted-foreground">
                                        {t('lastDiscovery', { time: new Date(provider.LastDiscoveryAt).toLocaleString() })}
                                    </span>
                                )}
                            </div>
                            <div className="flex items-center gap-1">
                                <Tip className="w-fit">
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => handleRefreshProvider(provider.Id)}
                                        disabled={isLoading || provider.Status === CloudProviderStatus.Discovering}
                                        aria-label={t('refreshResources')}
                                        data-testid={`refresh-${provider.Id}`}
                                    >
                                        <ArrowPathIcon className={cn("w-4 h-4", {
                                            "animate-spin": provider.Status === CloudProviderStatus.Discovering
                                        })} />
                                    </Button>
                                    <p>{t('refreshResources')}</p>
                                </Tip>
                                <Tip className="w-fit">
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => handleEditProvider(provider.Id)}
                                        disabled={isLoading}
                                        aria-label={t('edit')}
                                        data-testid={`edit-${provider.Id}`}
                                    >
                                        <PencilIcon className="w-4 h-4" />
                                    </Button>
                                    <p>{t('edit')}</p>
                                </Tip>
                                <Tip className="w-fit">
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => handleRemoveProvider(provider.Id, provider.Name)}
                                        disabled={isLoading || provider.IsEnvironmentDefined}
                                        aria-label={provider.IsEnvironmentDefined ? t('cannotRemoveEnv') : t('remove')}
                                        data-testid={`remove-${provider.Id}`}
                                    >
                                        <TrashIcon className="w-4 h-4" />
                                    </Button>
                                    <p>{provider.IsEnvironmentDefined ? t('cannotRemoveEnv') : t('remove')}</p>
                                </Tip>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            <AzureProviderModal
                open={azureModalOpen && providerModalType === CloudProviderType.Azure}
                onOpenChange={handleModalOpenChange}
            />
        </div>
    );
};
