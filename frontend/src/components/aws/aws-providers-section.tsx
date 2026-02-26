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

import { FC, useCallback, useEffect } from "react";
import { Badge, Button, cn, toast } from "@clidey/ux";
import {
    CloudProviderStatus,
    useGetCloudProvidersQuery,
    useRefreshCloudProviderMutation,
    useRemoveCloudProviderMutation,
} from "@graphql";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ProvidersActions, LocalCloudProvider } from "../../store/providers";
import { useTranslation } from "@/hooks/use-translation";
import { AwsProviderModal } from "./aws-provider-modal";
import { Tip } from "../tip";
import {
    ArrowPathIcon,
    CloudIcon,
    PencilIcon,
    PlusIcon,
    TrashIcon,
} from "../heroicons";

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
 * Cloud Providers section for the settings page.
 * Displays configured cloud providers and allows management.
 * Currently supports AWS, with GCP/Azure planned.
 */
export const AwsProvidersSection: FC = () => {
    const { t } = useTranslation('components/aws-providers-section');
    const dispatch = useAppDispatch();

    // Redux state
    const cloudProviders = useAppSelector(state => state.providers.cloudProviders);
    const isModalOpen = useAppSelector(state => state.providers.isProviderModalOpen);

    // GraphQL queries and mutations
    const { data, loading, refetch } = useGetCloudProvidersQuery();
    const [refreshProvider, { loading: refreshLoading }] = useRefreshCloudProviderMutation();
    const [removeProvider, { loading: removeLoading }] = useRemoveCloudProviderMutation();

    // Sync GraphQL data with Redux store on fetch
    useEffect(() => {
        if (data?.CloudProviders) {
            dispatch(ProvidersActions.setCloudProviders(data.CloudProviders as LocalCloudProvider[]));
        }
    }, [data, dispatch]);

    const handleAddProvider = useCallback(() => {
        dispatch(ProvidersActions.openAddProviderModal());
    }, [dispatch]);

    const handleEditProvider = useCallback((id: string) => {
        dispatch(ProvidersActions.openEditProviderModal({ id }));
    }, [dispatch]);

    const handleRemoveProvider = useCallback(async (id: string, name: string) => {
        try {
            const { data } = await removeProvider({ variables: { id } });
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
            const { data } = await refreshProvider({ variables: { id } });
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
                        data-testid="add-aws-provider"
                    >
                        <PlusIcon className="w-4 h-4 mr-1" />
                        {t('addProvider')}
                    </Button>
                </div>
            </div>

            <p className="text-sm text-muted-foreground">
                {t('description')}
            </p>

            {cloudProviders.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-8 px-4 border border-dashed rounded-lg">
                    <CloudIcon className="w-10 h-10 text-muted-foreground mb-3" />
                    <p className="text-sm text-muted-foreground text-center mb-4">
                        {t('noProviders')}
                    </p>
                    <Button
                        variant="outline"
                        onClick={handleAddProvider}
                        data-testid="add-first-aws-provider"
                    >
                        <PlusIcon className="w-4 h-4 mr-1" />
                        {t('addFirstProvider')}
                    </Button>
                </div>
            ) : (
                <div className="flex flex-col gap-3">
                    {cloudProviders.map((provider) => (
                        <div
                            key={provider.Id}
                            className="flex items-center justify-between p-4 border rounded-lg"
                            data-testid={`aws-provider-${provider.Id}`}
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
                                    <span>{provider.Region}</span>
                                    {provider.ProfileName && (
                                        <span>{provider.ProfileName}</span>
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

            <AwsProviderModal
                open={isModalOpen}
                onOpenChange={handleModalOpenChange}
            />
        </div>
    );
};
