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

import { skipToken, useMutation, useQuery } from "@apollo/client/react";
import type { FC} from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
    Badge,
    Button,
    cn,
    Dialog,
    DialogContent,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    Input,
    Label,
    Separator,
    Switch,
    toast,
} from "@clidey/ux";
import { SearchSelect } from "../ux";
import type {
    AzureProviderInput} from "@graphql";
import {
    AddAzureProviderDocument,
    CloudProviderStatus,
    GetAzureProvidersDocument,
    GetAzureRegionsDocument,
    GetAzureSubscriptionsDocument,
    GetDiscoveredConnectionsDocument,
    TestAzureCredentialsDocument,
    UpdateAzureProviderDocument,
} from "@graphql";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ProvidersActions } from "../../store/providers";
import { useTranslation } from "@/hooks/use-translation";
import { ChevronDownIcon, CloudIcon } from "../heroicons";
import { upsertAzureProviderCache, upsertCloudProviderCache } from "../../utils/apollo-provider-cache";

export interface AzureProviderModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

/**
 * Modal for adding or editing an Azure provider.
 * Supports Default (managed identity / CLI) and Service Principal auth methods.
 */
export const AzureProviderModal: FC<AzureProviderModalProps> = ({
    open,
    onOpenChange,
}) => {
    const { t } = useTranslation('components/azure-provider-modal');
    const dispatch = useAppDispatch();

    // Get editing state from Redux
    const editingProviderId = useAppSelector(state => state.providers.editingProviderId);
    const isEditMode = editingProviderId !== null;

    // Fetch current Azure providers to find editing provider
    const { data: providersData } = useQuery(GetAzureProvidersDocument);
    const editingProvider = useMemo(() => {
        if (!editingProviderId || !providersData?.AzureProviders) return null;
        return providersData.AzureProviders.find(p => p.Id === editingProviderId) ?? null;
    }, [editingProviderId, providersData]);
    const subscriptionsQueryOptions = isEditMode ? skipToken : {};

    // Query Azure subscriptions for picker
    const { data: subscriptionsData } = useQuery(GetAzureSubscriptionsDocument, subscriptionsQueryOptions);
    const subscriptions = subscriptionsData?.AzureSubscriptions ?? [];

    // Query Azure regions from backend
    const { data: regionsData } = useQuery(GetAzureRegionsDocument);
    const azureRegions = regionsData?.AzureRegions ?? [];

    // Form state
    const [name, setName] = useState("");
    const [subscriptionId, setSubscriptionId] = useState("");
    const [authMethod, setAuthMethod] = useState("Default");
    const [tenantId, setTenantId] = useState("");
    const [clientId, setClientId] = useState("");
    const [clientSecret, setClientSecret] = useState("");
    const [resourceGroup, setResourceGroup] = useState("");
    const [region, setRegion] = useState("");
    const [discoverPostgreSQL, setDiscoverPostgreSQL] = useState(true);
    const [discoverMySQL, setDiscoverMySQL] = useState(true);
    const [discoverRedis, setDiscoverRedis] = useState(true);
    const [discoverCosmosDB, setDiscoverCosmosDB] = useState(true);

    // GraphQL mutations
    const [addProvider, { loading: addLoading }] = useMutation(AddAzureProviderDocument, {
        refetchQueries: [GetDiscoveredConnectionsDocument],
        update(cache, { data }) {
            if (data?.AddAzureProvider) {
                upsertAzureProviderCache(cache, data.AddAzureProvider);
                upsertCloudProviderCache(cache, data.AddAzureProvider);
            }
        },
    });
    const [updateProvider, { loading: updateLoading }] = useMutation(UpdateAzureProviderDocument, {
        refetchQueries: [GetDiscoveredConnectionsDocument],
        update(cache, { data }) {
            if (data?.UpdateAzureProvider) {
                upsertAzureProviderCache(cache, data.UpdateAzureProvider);
                upsertCloudProviderCache(cache, data.UpdateAzureProvider);
            }
        },
    });
    const [testCredentials, { loading: testCredentialsLoading }] = useMutation(TestAzureCredentialsDocument);

    const loading = addLoading || updateLoading || testCredentialsLoading;

    const isServicePrincipal = authMethod === "ServicePrincipal";

    // Reset form when modal opens/closes or editingProvider changes
    useEffect(() => {
        if (open) {
            if (editingProvider) {
                setName(editingProvider.Name);
                setSubscriptionId(editingProvider.SubscriptionID);
                setTenantId(editingProvider.TenantID ?? "");
                setClientId("");
                setClientSecret("");
                setResourceGroup(editingProvider.ResourceGroup ?? "");
                setRegion(editingProvider.Region ?? "");
                setDiscoverPostgreSQL(editingProvider.DiscoverPostgreSQL);
                setDiscoverMySQL(editingProvider.DiscoverMySQL);
                setDiscoverRedis(editingProvider.DiscoverRedis);
                setDiscoverCosmosDB(editingProvider.DiscoverCosmosDB);
                // Infer auth method from whether TenantID is set
                setAuthMethod(editingProvider.TenantID ? "ServicePrincipal" : "Default");
            } else {
                setName("");
                setSubscriptionId("");
                setAuthMethod("Default");
                setTenantId("");
                setClientId("");
                setClientSecret("");
                setResourceGroup("");
                setRegion("");
                setDiscoverPostgreSQL(true);
                setDiscoverMySQL(true);
                setDiscoverRedis(true);
                setDiscoverCosmosDB(true);
            }
        }
    }, [open, editingProvider]);

    const handleClose = useCallback(() => {
        onOpenChange(false);
        dispatch(ProvidersActions.closeProviderModal());
    }, [onOpenChange, dispatch]);

    /** Auto-fill form from a discovered subscription. */
    const handleSelectSubscription = useCallback((subId: string) => {
        const sub = subscriptions.find(s => s.Id === subId);
        if (!sub) return;

        setSubscriptionId(sub.Id);
        if (!name) {
            setName(sub.DisplayName);
        }
        if (sub.TenantID && isServicePrincipal) {
            setTenantId(sub.TenantID);
        }
    }, [subscriptions, name, isServicePrincipal]);

    const buildInput = useCallback((): AzureProviderInput => {
        const input: AzureProviderInput = {
            Name: name,
            SubscriptionID: subscriptionId,
            AuthMethod: authMethod,
            DiscoverPostgreSQL: discoverPostgreSQL,
            DiscoverMySQL: discoverMySQL,
            DiscoverRedis: discoverRedis,
            DiscoverCosmosDB: discoverCosmosDB,
        };

        if (isServicePrincipal) {
            if (tenantId) input.TenantID = tenantId;
            if (clientId) input.ClientID = clientId;
            if (clientSecret) input.ClientSecret = clientSecret;
        }

        if (resourceGroup) {
            input.ResourceGroup = resourceGroup;
        }

        return input;
    }, [name, subscriptionId, authMethod, tenantId, clientId, clientSecret, resourceGroup, discoverPostgreSQL, discoverMySQL, discoverRedis, discoverCosmosDB, isServicePrincipal]);

    const handleSubmit = useCallback(async () => {
        if (!name.trim()) {
            toast.error(t('nameRequired'));
            return;
        }

        if (!subscriptionId.trim()) {
            toast.error(t('subscriptionRequired'));
            return;
        }

        const input = buildInput();

        try {
            if (isEditMode && editingProviderId) {
                const { data } = await updateProvider({
                    variables: { id: editingProviderId, input },
                });
                if (data?.UpdateAzureProvider) {
                    toast.success(t('providerUpdated'));
                    handleClose();
                }
            } else {
                const { data } = await addProvider({
                    variables: { input },
                });
                if (data?.AddAzureProvider) {
                    toast.success(t('providerAdded'));
                    handleClose();
                }
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('operationFailed', { error: errorMessage }));
        }
    }, [name, subscriptionId, buildInput, isEditMode, editingProviderId, updateProvider, addProvider, handleClose, t]);

    const handleTest = useCallback(async () => {
        if (!name.trim() || !subscriptionId.trim()) {
            toast.error(t('nameRequired'));
            return;
        }

        try {
            const input = buildInput();
            const { data } = await testCredentials({
                variables: { input },
            });
            if (data?.TestAzureCredentials === CloudProviderStatus.Connected) {
                toast.success(t('connectionSuccessful'));
            } else {
                toast.error(t('connectionFailed'));
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('testFailed', { error: errorMessage }));
        }
    }, [name, subscriptionId, buildInput, testCredentials, t]);

    const authMethodOptions = useMemo(() => [
        { value: "Default", label: t('authDefault') },
        { value: "ServicePrincipal", label: t('servicePrincipal') },
    ], [t]);

    const regionOptions = useMemo(() => {
        const grouped: { value: string; label: string }[] = [];
        let lastGeography = "";
        for (const r of azureRegions) {
            const geography = r.Geography ?? "";
            if (geography !== lastGeography && geography) {
                grouped.push({ value: `__header_${geography}`, label: `── ${geography} ──` });
            }
            lastGeography = geography;
            grouped.push({ value: r.Id, label: `${r.Id} - ${r.DisplayName}` });
        }
        return grouped;
    }, [azureRegions]);

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto" data-testid="azure-provider-modal">
                <DialogHeader>
                    <DialogTitle>
                        {isEditMode ? t('editProvider') : t('addAzureAccount')}
                    </DialogTitle>
                </DialogHeader>

                <div className="flex flex-col gap-4 py-4">
                    {/* Discovered Subscriptions */}
                    {!isEditMode && subscriptions.length > 0 && (
                        <div className="flex flex-col gap-3">
                            <div className="flex items-center gap-2">
                                <CloudIcon className="w-4 h-4 text-muted-foreground" />
                                <Label className="text-sm font-medium">{t('detectedSubscriptions')}</Label>
                            </div>
                            <div className="flex flex-wrap gap-2">
                                {subscriptions.map((sub) => (
                                    <button
                                        key={sub.Id}
                                        type="button"
                                        onClick={() => handleSelectSubscription(sub.Id)}
                                        className={cn(
                                            "flex items-center gap-2 px-3 py-2 rounded-md border text-sm transition-colors",
                                            "hover:border-brand hover:bg-brand/5",
                                            "focus:outline-none focus:ring-2 focus:ring-brand focus:ring-offset-2"
                                        )}
                                    >
                                        <span className="font-medium">{sub.DisplayName}</span>
                                        <Badge variant="outline" className="text-xs">
                                            {sub.State}
                                        </Badge>
                                    </button>
                                ))}
                            </div>
                            <p className="text-xs text-muted-foreground">
                                {t('clickToAutofill')}
                            </p>
                            <Separator />
                        </div>
                    )}

                    {/* Name */}
                    <div className="flex flex-col gap-2">
                        <Label htmlFor="provider-name">{t('displayName')}</Label>
                        <Input
                            id="provider-name"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            placeholder={t('namePlaceholder')}
                            data-testid="provider-name"
                        />
                    </div>

                    {/* Subscription ID */}
                    <div className="flex flex-col gap-2">
                        <Label htmlFor="subscription-id">{t('subscriptionId')}</Label>
                        <Input
                            id="subscription-id"
                            value={subscriptionId}
                            onChange={(e) => setSubscriptionId(e.target.value)}
                            placeholder={t('uuidPlaceholder')}
                            data-testid="subscription-id"
                        />
                    </div>

                    {/* Auth Method */}
                    <div className="flex flex-col gap-2">
                        <Label>{t('authMethod')}</Label>
                        <SearchSelect
                            value={authMethod}
                            onChange={setAuthMethod}
                            options={authMethodOptions}
                            placeholder={t('selectAuthMethod')}
                            contentClassName="w-[var(--radix-popover-trigger-width)]"
                            rightIcon={<ChevronDownIcon className="w-4 h-4" />}
                        />
                        <p className="text-xs text-muted-foreground">
                            {t('authMethodDesc')}
                        </p>
                    </div>

                    {/* Service Principal fields */}
                    {isServicePrincipal && (
                        <>
                            <div className="flex flex-col gap-2">
                                <Label htmlFor="tenant-id">{t('tenantId')}</Label>
                                <Input
                                    id="tenant-id"
                                    value={tenantId}
                                    onChange={(e) => setTenantId(e.target.value)}
                                    placeholder={t('uuidPlaceholder')}
                                    data-testid="tenant-id"
                                />
                            </div>
                            <div className="flex flex-col gap-2">
                                <Label htmlFor="client-id">{t('clientId')}</Label>
                                <Input
                                    id="client-id"
                                    value={clientId}
                                    onChange={(e) => setClientId(e.target.value)}
                                    placeholder={t('uuidPlaceholder')}
                                    data-testid="client-id"
                                />
                            </div>
                            <div className="flex flex-col gap-2">
                                <Label htmlFor="client-secret">{t('clientSecret')}</Label>
                                <Input
                                    id="client-secret"
                                    type="password"
                                    value={clientSecret}
                                    onChange={(e) => setClientSecret(e.target.value)}
                                    placeholder={t('clientSecretPlaceholder')}
                                    data-testid="client-secret"
                                />
                            </div>
                        </>
                    )}

                    {/* Resource Group (optional) */}
                    <div className="flex flex-col gap-2">
                        <Label htmlFor="resource-group">{t('resourceGroup')}</Label>
                        <Input
                            id="resource-group"
                            value={resourceGroup}
                            onChange={(e) => setResourceGroup(e.target.value)}
                            placeholder={t('resourceGroupPlaceholder')}
                            data-testid="resource-group"
                        />
                        <p className="text-xs text-muted-foreground">
                            {t('resourceGroupDesc')}
                        </p>
                    </div>

                    {/* Region (optional) */}
                    {regionOptions.length > 0 && (
                        <div className="flex flex-col gap-2">
                            <Label>{t('azureRegion')}</Label>
                            <SearchSelect
                                value={region}
                                onChange={setRegion}
                                options={regionOptions}
                                placeholder={t('selectRegion')}
                                contentClassName="w-[var(--radix-popover-trigger-width)]"
                                rightIcon={<ChevronDownIcon className="w-4 h-4" />}
                            />
                        </div>
                    )}

                    {/* Discovery toggles */}
                    <div className="flex flex-col gap-3 pt-4 border-t">
                        <Label className="font-medium">{t('resourceDiscovery')}</Label>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('postgresqlLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('postgresqlDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverPostgreSQL}
                                onCheckedChange={setDiscoverPostgreSQL}
                                data-testid="discover-postgresql"
                            />
                        </div>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('mysqlLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('mysqlDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverMySQL}
                                onCheckedChange={setDiscoverMySQL}
                                data-testid="discover-mysql"
                            />
                        </div>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('redisLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('redisDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverRedis}
                                onCheckedChange={setDiscoverRedis}
                                data-testid="discover-redis"
                            />
                        </div>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('cosmosdbLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('cosmosdbDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverCosmosDB}
                                onCheckedChange={setDiscoverCosmosDB}
                                data-testid="discover-cosmosdb"
                            />
                        </div>
                    </div>
                </div>

                <DialogFooter className="flex gap-2">
                    <Button
                        variant="outline"
                        onClick={handleTest}
                        disabled={loading}
                        data-testid="test-connection"
                    >
                        {testCredentialsLoading ? t('testing') : t('testConnection')}
                    </Button>
                    <Button
                        variant="outline"
                        onClick={handleClose}
                        disabled={loading}
                    >
                        {t('cancel')}
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={loading}
                        data-testid="submit-provider"
                    >
                        {(addLoading || updateLoading) ? t('saving') : (isEditMode ? t('save') : t('add'))}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
};
