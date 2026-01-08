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
import {
    Badge,
    Button,
    cn,
    Label,
    Popover,
    PopoverContent,
    PopoverTrigger,
    toast,
} from "@clidey/ux";
import {
    useGetCloudProvidersQuery,
    useGetDiscoveredConnectionsQuery,
    useRefreshCloudProviderMutation,
} from "@graphql";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ProvidersActions, LocalCloudProvider, LocalDiscoveredConnection } from "../../store/providers";
import { useTranslation } from "@/hooks/use-translation";
import { Icons } from "../icons";
import { AwsProviderModal } from "./aws-provider-modal";
import {
    ArrowPathIcon,
    CloudIcon,
    PlusIcon,
    QuestionMarkCircleIcon,
} from "../heroicons";
import { ReactElement } from "react";
import { buildConnectionPrefill, ConnectionPrefillData } from "@/utils/cloud-connection-prefill";

export type AwsConnectionPrefillData = ConnectionPrefillData;

interface AwsConnectionPickerProps {
    /** Called when user clicks a discovered connection - prefills the main login form */
    onSelectConnection?: (data: AwsConnectionPrefillData) => void;
}

/**
 * A picker component for AWS-discovered database connections.
 * Shows on the login page when AWS providers are configured.
 * When a connection is clicked, it calls onSelectConnection to prefill the main login form.
 */
export const AwsConnectionPicker: FC<AwsConnectionPickerProps> = ({
    onSelectConnection,
}) => {
    const { t } = useTranslation('components/aws-connection-picker');
    const dispatch = useAppDispatch();

    // Redux state
    const cloudProviders = useAppSelector(state => state.providers.cloudProviders);
    const discoveredConnections = useAppSelector(state => state.providers.discoveredConnections);
    const isModalOpen = useAppSelector(state => state.providers.isProviderModalOpen);

    // GraphQL - these operations are on the auth allowlist, so no skip needed
    const { data: providersData, loading: providersLoading, refetch: refetchProviders } = useGetCloudProvidersQuery();
    const { data: connectionsData, loading: connectionsLoading, refetch: refetchConnections } = useGetDiscoveredConnectionsQuery();
    const [refreshProvider, { loading: refreshLoading }] = useRefreshCloudProviderMutation();

    // Sync GraphQL data with Redux
    useEffect(() => {
        if (providersData?.CloudProviders) {
            dispatch(ProvidersActions.setCloudProviders(providersData.CloudProviders as LocalCloudProvider[]));
        }
    }, [providersData, dispatch]);

    useEffect(() => {
        if (connectionsData?.DiscoveredConnections) {
            dispatch(ProvidersActions.setDiscoveredConnections(connectionsData.DiscoveredConnections as LocalDiscoveredConnection[]));
        }
    }, [connectionsData, dispatch]);

    const handleAddProvider = useCallback(() => {
        dispatch(ProvidersActions.openAddProviderModal());
    }, [dispatch]);

    const handleRefresh = useCallback(async () => {
        // Refresh all providers to trigger fresh discovery
        const providers = providersData?.CloudProviders ?? [];
        let hasErrors = false;
        for (const provider of providers) {
            try {
                const result = await refreshProvider({ variables: { id: provider.Id } });
                // Check if there was a partial error (some services failed)
                if (result.data?.RefreshCloudProvider?.Error) {
                    hasErrors = true;
                }
            } catch (error) {
                hasErrors = true;
                console.error(`Failed to refresh provider ${provider.Id}:`, error);
            }
        }
        // Then refetch the updated data
        await refetchProviders();
        await refetchConnections();

        if (hasErrors) {
            toast.warning(t('refreshPartialError'));
        }
    }, [providersData, refreshProvider, refetchProviders, refetchConnections, t]);

    /**
     * Build prefill data from a discovered connection and call the callback.
     * Uses shared prefill rules that handle SSL/TLS settings for each database type.
     */
    const handleSelectConnection = useCallback((conn: LocalDiscoveredConnection) => {
        if (!onSelectConnection) return;

        onSelectConnection(buildConnectionPrefill(conn));
        toast.success(t('connectionSelected'));
    }, [onSelectConnection, t]);

    const handleModalOpenChange = useCallback((open: boolean) => {
        if (!open) {
            dispatch(ProvidersActions.closeProviderModal());
        }
    }, [dispatch]);

    const loading = providersLoading || connectionsLoading || refreshLoading;

    // Get database type icon
    const getDbIcon = useCallback((dbType: string): ReactElement | null => {
        const iconMap: Record<string, ReactElement> = Icons.Logos as Record<string, ReactElement>;
        // Normalize type name (e.g., "Postgres" -> "Postgres", "MySQL" -> "MySql")
        return iconMap[dbType] ?? null;
    }, []);

    // Don't render if no providers and no connections
    if (cloudProviders.length === 0 && discoveredConnections.length === 0) {
        return (
            <div className="flex flex-col items-center gap-4 py-6">
                <div className="flex items-center gap-2 text-muted-foreground">
                    <CloudIcon className="w-5 h-5" />
                    <span className="text-sm">{t('connectAwsAccount')}</span>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    onClick={handleAddProvider}
                    data-testid="add-aws-provider-login"
                >
                    <PlusIcon className="w-4 h-4 mr-1" />
                    {t('addAwsAccount')}
                </Button>
                <AwsProviderModal
                    open={isModalOpen}
                    onOpenChange={handleModalOpenChange}
                />
            </div>
        );
    }

    return (
        <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <CloudIcon className="w-5 h-5" />
                    <Label className="font-medium">{t('awsConnections')}</Label>
                    <Popover>
                        <PopoverTrigger asChild>
                            <button
                                type="button"
                                className="text-muted-foreground hover:text-foreground transition-colors"
                                aria-label={t('helpLabel')}
                            >
                                <QuestionMarkCircleIcon className="w-4 h-4" />
                            </button>
                        </PopoverTrigger>
                        <PopoverContent className="w-80 p-4" side="bottom" align="start">
                            <div className="flex flex-col gap-3">
                                <h4 className="font-medium text-sm">{t('helpTitle')}</h4>
                                <div className="text-xs text-muted-foreground space-y-2">
                                    <p>{t('helpIntro')}</p>
                                    <div className="space-y-1">
                                        <p className="font-medium text-foreground">{t('helpAuthMethods')}</p>
                                        <ul className="list-disc list-inside space-y-1 pl-1">
                                            <li><span className="font-medium">{t('helpAuthDefault')}</span> – {t('helpAuthDefaultDesc')}</li>
                                            <li><span className="font-medium">{t('helpAuthProfile')}</span> – {t('helpAuthProfileDesc')}</li>
                                            <li><span className="font-medium">{t('helpAuthStatic')}</span> – {t('helpAuthStaticDesc')}</li>
                                            <li><span className="font-medium">{t('helpAuthIam')}</span> – {t('helpAuthIamDesc')}</li>
                                        </ul>
                                    </div>
                                    <div className="space-y-1">
                                        <p className="font-medium text-foreground">{t('helpServices')}</p>
                                        <ul className="list-disc list-inside space-y-1 pl-1">
                                            <li><span className="font-medium">RDS</span> – {t('helpRdsDesc')}</li>
                                            <li><span className="font-medium">ElastiCache</span> – {t('helpElasticacheDesc')}</li>
                                            <li><span className="font-medium">DocumentDB</span> – {t('helpDocumentdbDesc')}</li>
                                        </ul>
                                    </div>
                                    <p className="pt-1 border-t">{t('helpCredentialNote')}</p>
                                </div>
                            </div>
                        </PopoverContent>
                    </Popover>
                </div>
                <div className="flex items-center gap-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={handleRefresh}
                        disabled={loading}
                        title={t('refresh')}
                    >
                        <ArrowPathIcon className={cn("w-4 h-4", { "animate-spin": loading })} />
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={handleAddProvider}
                        title={t('addProvider')}
                    >
                        <PlusIcon className="w-4 h-4" />
                    </Button>
                </div>
            </div>

            {discoveredConnections.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-4">
                    {t('noDiscoveredConnections')}
                </p>
            ) : (
                <>
                    <p className="text-xs text-muted-foreground">
                        {t('clickToFillForm')}
                    </p>
                    <div className="flex flex-col gap-2 max-h-[200px] overflow-y-auto">
                        {discoveredConnections.map((conn) => (
                            <button
                                key={conn.Id}
                                onClick={() => handleSelectConnection(conn)}
                                className="flex items-center gap-3 p-3 rounded-lg border text-left transition-colors border-border hover:border-brand/50 hover:bg-brand/5"
                                data-testid={`aws-connection-${conn.Id}`}
                            >
                                <div className="w-8 h-8 flex items-center justify-center">
                                    {getDbIcon(conn.DatabaseType)}
                                </div>
                                <div className="flex flex-col gap-0.5 flex-1 min-w-0">
                                    <div className="flex items-center gap-2">
                                        <span className="font-medium truncate">{conn.Name}</span>
                                        <Badge variant="outline" className="text-xs shrink-0">
                                            {conn.DatabaseType}
                                        </Badge>
                                    </div>
                                    <span className="text-xs text-muted-foreground truncate">
                                        {conn.Region} • {conn.Status}
                                    </span>
                                </div>
                                <CloudIcon className="w-4 h-4 text-muted-foreground shrink-0" />
                            </button>
                        ))}
                    </div>
                </>
            )}

            <AwsProviderModal
                open={isModalOpen}
                onOpenChange={handleModalOpenChange}
            />
        </div>
    );
};
