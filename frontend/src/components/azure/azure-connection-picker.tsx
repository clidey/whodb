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
import { FC, useCallback, useEffect, useMemo } from "react";
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
    CloudProviderType,
    GetAzureProvidersDocument,
    GetDiscoveredConnectionsDocument,
    RefreshAzureProviderDocument,
} from "@graphql";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ProvidersActions, LocalDiscoveredConnection } from "../../store/providers";
import { useTranslation } from "@/hooks/use-translation";
import { Icons } from "../icons";
import { AzureProviderModal } from "./azure-provider-modal";
import { Tip } from "../tip";
import {
    ArrowPathIcon,
    CloudIcon,
    PlusIcon,
    QuestionMarkCircleIcon,
} from "../heroicons";
import { ReactElement } from "react";
import type { SourceTypeItem } from "@/config/source-types";
import { buildConnectionPrefill, ConnectionPrefillData } from "@/utils/cloud-connection-prefill";
import { getAppName } from "@/config/features";

export type AzureConnectionPrefillData = ConnectionPrefillData;

/** Checks if a profile ID belongs to an Azure-discovered connection. */
export function isAzureConnection(profileId: string | undefined): boolean {
    return profileId?.startsWith("azure-") ?? false;
}

interface AzureConnectionPickerProps {
    /** Called when user clicks a discovered connection - prefills the main login form */
    onSelectConnection?: (data: AzureConnectionPrefillData) => void;
    /** Available source types used to apply backend-owned discovery-prefill metadata. */
    sourceTypes: SourceTypeItem[];
}

/**
 * A picker component for Azure-discovered database connections.
 * Shows on the login page when Azure providers are configured.
 * When a connection is clicked, it calls onSelectConnection to prefill the main login form.
 */
export const AzureConnectionPicker: FC<AzureConnectionPickerProps> = ({
    onSelectConnection,
    sourceTypes,
}) => {
    const { t } = useTranslation('components/azure-connection-picker');
    const appName = getAppName();
    const dispatch = useAppDispatch();

    // Redux state
    const isModalOpen = useAppSelector(state => state.providers.isProviderModalOpen);
    const providerModalType = useAppSelector(state => state.providers.providerModalType);

    // GraphQL - these operations are on the auth allowlist, so no skip needed
    const { data: providersData, loading: providersLoading, refetch: refetchProviders } = useQuery(GetAzureProvidersDocument);
    const { data: connectionsData, loading: connectionsLoading, refetch: refetchConnections } = useQuery(GetDiscoveredConnectionsDocument);
    const [refreshProvider, { loading: refreshLoading }] = useMutation(RefreshAzureProviderDocument);

    const azureProviders = providersData?.AzureProviders ?? [];

    // Filter discovered connections to Azure provider type only
    const azureConnections: LocalDiscoveredConnection[] = useMemo(() => {
        const allConnections = (connectionsData?.DiscoveredConnections ?? []) as LocalDiscoveredConnection[];
        return allConnections.filter(c => c.ProviderType === CloudProviderType.Azure);
    }, [connectionsData]);

    // Sync all discovered connections with Redux (needed for other components)
    useEffect(() => {
        if (connectionsData?.DiscoveredConnections) {
            dispatch(ProvidersActions.setDiscoveredConnections(connectionsData.DiscoveredConnections as LocalDiscoveredConnection[]));
        }
    }, [connectionsData, dispatch]);

    const handleAddProvider = useCallback(() => {
        dispatch(ProvidersActions.openAddProviderModal({ providerType: CloudProviderType.Azure }));
    }, [dispatch]);

    const handleRefresh = useCallback(async () => {
        const providers = azureProviders;
        let hasErrors = false;
        for (const provider of providers) {
            try {
                const result = await refreshProvider({ variables: { id: provider.Id } });
                if (result.data?.RefreshAzureProvider?.Error) {
                    hasErrors = true;
                }
            } catch (error) {
                hasErrors = true;
                console.error(`Failed to refresh Azure provider ${provider.Id}:`, error);
            }
        }
        await refetchProviders();
        await refetchConnections();

        if (hasErrors) {
            toast.warning(t('refreshPartialError'));
        }
    }, [azureProviders, refreshProvider, refetchProviders, refetchConnections, t]);

    /**
     * Build prefill data from a discovered connection and call the callback.
     * Uses shared prefill rules that handle SSL/TLS settings for each database type.
     */
    const handleSelectConnection = useCallback((conn: LocalDiscoveredConnection) => {
        if (!onSelectConnection) return;

        const sourceType = sourceTypes.find(item => item.id.toLowerCase() === conn.DatabaseType.toLowerCase());
        onSelectConnection(buildConnectionPrefill(conn, sourceType));
        toast.success(t('connectionSelected'));
    }, [onSelectConnection, sourceTypes, t]);

    const handleModalOpenChange = useCallback((open: boolean) => {
        if (!open) {
            dispatch(ProvidersActions.closeProviderModal());
        }
    }, [dispatch]);

    const loading = providersLoading || connectionsLoading || refreshLoading;
    const showModal = isModalOpen && providerModalType === CloudProviderType.Azure;

    // Get database type icon
    const getDbIcon = useCallback((dbType: string): ReactElement | null => {
        const iconMap: Record<string, ReactElement> = Icons.Logos as Record<string, ReactElement>;
        return iconMap[dbType] ?? null;
    }, []);

    // Don't render if no providers and no connections
    if (azureProviders.length === 0 && azureConnections.length === 0) {
        return (
            <div className="flex flex-col items-center gap-4 py-6">
                <div className="flex items-center gap-2 text-muted-foreground">
                    <CloudIcon className="w-5 h-5" />
                    <span className="text-sm">{t('connectAzureAccount')}</span>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    onClick={handleAddProvider}
                    data-testid="add-azure-provider-login"
                >
                    <PlusIcon className="w-4 h-4 mr-1" />
                    {t('addAzureAccount')}
                </Button>
                <AzureProviderModal
                    open={showModal}
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
                    <Label className="font-medium">{t('azureConnections')}</Label>
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
                                    <p>{t('helpIntro', { appName })}</p>
                                    <div className="space-y-1">
                                        <p className="font-medium text-foreground">{t('helpAuthMethods')}</p>
                                        <ul className="list-disc list-inside space-y-1 pl-1">
                                            <li><span className="font-medium">{t('helpAuthDefault')}</span> – {t('helpAuthDefaultDesc')}</li>
                                            <li><span className="font-medium">{t('servicePrincipal')}</span> – {t('helpAuthSPDesc')}</li>
                                        </ul>
                                    </div>
                                    <div className="space-y-1">
                                        <p className="font-medium text-foreground">{t('helpServices')}</p>
                                        <ul className="list-disc list-inside space-y-1 pl-1">
                                            <li><span className="font-medium">PostgreSQL</span> – {t('postgresqlDesc')}</li>
                                            <li><span className="font-medium">MySQL</span> – {t('mysqlDesc')}</li>
                                            <li><span className="font-medium">Redis</span> – {t('redisDesc')}</li>
                                            <li><span className="font-medium">Cosmos DB</span> – {t('cosmosdbDesc')}</li>
                                        </ul>
                                    </div>
                                    <p className="pt-1 border-t">{t('helpCredentialNote', { appName })}</p>
                                </div>
                            </div>
                        </PopoverContent>
                    </Popover>
                </div>
                <div className="flex items-center gap-2">
                    <Tip className="w-fit">
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={handleRefresh}
                            disabled={loading}
                            aria-label={t('refresh')}
                        >
                            <ArrowPathIcon className={cn("w-4 h-4", { "animate-spin": loading })} />
                        </Button>
                        <p>{t('refresh')}</p>
                    </Tip>
                    <Tip className="w-fit">
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={handleAddProvider}
                            aria-label={t('addProvider')}
                        >
                            <PlusIcon className="w-4 h-4" />
                        </Button>
                        <p>{t('addProvider')}</p>
                    </Tip>
                </div>
            </div>

            {azureConnections.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-4">
                    {t('noDiscoveredConnections')}
                </p>
            ) : (
                <>
                    <p className="text-xs text-muted-foreground">
                        {t('clickToFillForm')}
                    </p>
                    <div className="flex flex-col gap-2 max-h-[200px] overflow-y-auto">
                        {azureConnections.map((conn) => (
                            <button
                                key={conn.Id}
                                onClick={() => handleSelectConnection(conn)}
                                className="flex items-center gap-3 p-3 rounded-lg border text-left transition-colors border-border hover:border-brand/50 hover:bg-brand/5"
                                data-testid={`azure-connection-${conn.Id}`}
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
                                    <div className="flex items-center gap-1 text-xs text-muted-foreground truncate">
                                        <span>{conn.Region} • {conn.Status}</span>
                                        {conn.Metadata?.find(m => m.Key === "connectivity")?.Value === "unreachable" && (
                                            <span className="text-destructive" title={t('unreachableTooltip')}>• {t('unreachable')}</span>
                                        )}
                                    </div>
                                </div>
                                <CloudIcon className="w-4 h-4 text-muted-foreground shrink-0" />
                            </button>
                        ))}
                    </div>
                </>
            )}

            <AzureProviderModal
                open={showModal}
                onOpenChange={handleModalOpenChange}
            />
        </div>
    );
};
