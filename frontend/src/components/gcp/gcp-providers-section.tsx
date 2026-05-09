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

import { useMutation } from "@apollo/client/react";
import { FC, useCallback, useMemo } from "react";
import { Badge, Button, cn, toast } from "@clidey/ux";
import {
    CloudProviderStatus,
    CloudProviderType,
    GetDiscoveredConnectionsDocument,
    RefreshCloudProviderDocument,
    RemoveCloudProviderDocument,
} from "@graphql";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ProvidersActions, LocalCloudProvider } from "../../store/providers";
import { useTranslation } from "@/hooks/use-translation";
import { GcpProviderModal } from "./gcp-provider-modal";
import { Tip } from "../tip";
import {
    ArrowPathIcon,
    CloudIcon,
    PencilIcon,
    PlusIcon,
    TrashIcon,
} from "../heroicons";
import { removeCloudProviderCache, upsertCloudProviderCache } from "../../utils/apollo-provider-cache";

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
 * GCP Providers section for the settings page.
 * Displays configured GCP providers and allows management.
 */
export const GcpProvidersSection: FC = () => {
    const { t } = useTranslation('components/gcp-providers-section');
    const dispatch = useAppDispatch();

    // Redux state - filter to GCP providers only
    const allProviders = useAppSelector(state => state.providers.cloudProviders);
    const gcpProviders = useMemo(() =>
        allProviders.filter(p => p.ProviderType === CloudProviderType.Gcp),
        [allProviders]
    );
    const isModalOpen = useAppSelector(state => state.providers.isProviderModalOpen);
    const providerModalType = useAppSelector(state => state.providers.providerModalType);

    // GraphQL mutations
    const [refreshProvider, { loading: refreshLoading }] = useMutation(RefreshCloudProviderDocument);
    const [removeProvider, { loading: removeLoading }] = useMutation(RemoveCloudProviderDocument);

    const handleAddProvider = useCallback(() => {
        dispatch(ProvidersActions.openAddProviderModal({ providerType: CloudProviderType.Gcp }));
    }, [dispatch]);

    const handleEditProvider = useCallback((id: string) => {
        dispatch(ProvidersActions.openEditProviderModal({ id, providerType: CloudProviderType.Gcp }));
    }, [dispatch]);

    const handleRemoveProvider = useCallback(async (id: string, name: string) => {
        try {
            const { data } = await removeProvider({
                variables: { id },
                refetchQueries: [GetDiscoveredConnectionsDocument],
                update(cache) {
                    removeCloudProviderCache(cache, id);
                },
            });
            if (data?.RemoveCloudProvider?.Status) {
                dispatch(ProvidersActions.removeCloudProvider({ id }));
                toast.success(t('providerRemoved', { name }));
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('removeFailed', { error: errorMessage }));
        }
    }, [removeProvider, dispatch, t]);

    const handleRefreshProvider = useCallback(async (id: string) => {
        dispatch(ProvidersActions.setProviderStatus({ id, status: CloudProviderStatus.Discovering }));
        try {
            const { data } = await refreshProvider({
                variables: { id },
                refetchQueries: [GetDiscoveredConnectionsDocument],
                update(cache, result) {
                    if (result.data?.RefreshCloudProvider) {
                        upsertCloudProviderCache(cache, result.data.RefreshCloudProvider);
                    }
                },
            });
            if (data?.RefreshCloudProvider) {
                dispatch(ProvidersActions.updateCloudProvider(data.RefreshCloudProvider as LocalCloudProvider));
                toast.success(t('refreshComplete', { count: data.RefreshCloudProvider.DiscoveredCount }));
            }
        } catch (error) {
            dispatch(ProvidersActions.setProviderStatus({ id, status: CloudProviderStatus.Error }));
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('refreshFailed', { error: errorMessage }));
        }
    }, [refreshProvider, dispatch, t]);

    const handleModalOpenChange = useCallback((open: boolean) => {
        if (!open) {
            dispatch(ProvidersActions.closeProviderModal());
        }
    }, [dispatch]);

    const isLoading = refreshLoading || removeLoading;

    const showModal = isModalOpen && providerModalType === CloudProviderType.Gcp;

    return (
        <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <CloudIcon className="w-5 h-5" />
                    <h3 className="text-lg font-bold">{t('title')}</h3>
                </div>
                <div className="flex items-center gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={handleAddProvider}
                        data-testid="add-gcp-provider"
                    >
                        <PlusIcon className="w-4 h-4 mr-1" />
                        {t('addProvider')}
                    </Button>
                </div>
            </div>

            <p className="text-sm text-muted-foreground">
                {t('description')}
            </p>

            {gcpProviders.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-8 px-4 border border-dashed rounded-lg">
                    <CloudIcon className="w-10 h-10 text-muted-foreground mb-3" />
                    <p className="text-sm text-muted-foreground text-center mb-4">
                        {t('noProviders')}
                    </p>
                    <Button
                        variant="outline"
                        onClick={handleAddProvider}
                        data-testid="add-first-gcp-provider"
                    >
                        <PlusIcon className="w-4 h-4 mr-1" />
                        {t('addFirstProvider')}
                    </Button>
                </div>
            ) : (
                <div className="flex flex-col gap-3">
                    {gcpProviders.map((provider) => {
                        const gcpProvider = provider as LocalCloudProvider & { ProjectID?: string };
                        return (
                            <div
                                key={provider.Id}
                                className="flex items-center justify-between p-4 border rounded-lg"
                                data-testid={`gcp-provider-${provider.Id}`}
                            >
                                <div className="flex flex-col gap-1">
                                    <div className="flex items-center gap-2">
                                        <span className="font-medium">{provider.Name}</span>
                                        <Badge variant={getStatusVariant(provider.Status)}>
                                            {t(`status.${provider.Status.toLowerCase()}`)}
                                        </Badge>
                                    </div>
                                    <div className="flex items-center gap-4 text-sm text-muted-foreground">
                                        <span>{provider.Region}</span>
                                        {gcpProvider.ProjectID && (
                                            <span>{gcpProvider.ProjectID}</span>
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
                                            disabled={isLoading}
                                            aria-label={t('remove')}
                                            data-testid={`remove-${provider.Id}`}
                                        >
                                            <TrashIcon className="w-4 h-4" />
                                        </Button>
                                        <p>{t('remove')}</p>
                                    </Tip>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}

            <GcpProviderModal
                open={showModal}
                onOpenChange={handleModalOpenChange}
            />
        </div>
    );
};
