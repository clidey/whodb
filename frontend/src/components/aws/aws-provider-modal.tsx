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

import { FC, useCallback, useEffect, useMemo, useState } from "react";
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
import {
    AwsProviderInput,
    CloudProviderStatus,
    LocalAwsProfile,
    useAddAwsProviderMutation,
    useUpdateAwsProviderMutation,
    useTestCloudProviderMutation,
    useGetLocalAwsProfilesQuery,
    useGetAwsRegionsQuery,
} from "@graphql";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ProvidersActions, LocalCloudProvider } from "../../store/providers";
import { useTranslation } from "@/hooks/use-translation";
import { ChevronDownIcon, CloudIcon } from "../heroicons";

export interface AwsProviderModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export const AwsProviderModal: FC<AwsProviderModalProps> = ({
    open,
    onOpenChange,
}) => {
    const { t } = useTranslation('components/aws-provider-modal');
    const dispatch = useAppDispatch();

    // Get editing state from Redux
    const editingProviderId = useAppSelector(state => state.providers.editingProviderId);
    const cloudProviders = useAppSelector(state => state.providers.cloudProviders);
    const editingProvider = useMemo(() => {
        if (!editingProviderId) return null;
        return cloudProviders.find(p => p.Id === editingProviderId) ?? null;
    }, [editingProviderId, cloudProviders]);

    const isEditMode = editingProviderId !== null;

    // Query local AWS profiles
    const { data: localProfilesData, loading: profilesLoading } = useGetLocalAwsProfilesQuery({
        skip: isEditMode, // Only fetch for new providers
    });
    const localProfiles = localProfilesData?.LocalAWSProfiles ?? [];

    // Query AWS regions from backend
    const { data: regionsData } = useGetAwsRegionsQuery();
    const awsRegions = regionsData?.AWSRegions ?? [];

    // Form state
    const [name, setName] = useState("");
    const [region, setRegion] = useState("us-east-1");
    const [customRegion, setCustomRegion] = useState("");
    const [profileName, setProfileName] = useState("");
    const [discoverRDS, setDiscoverRDS] = useState(true);
    const [discoverElastiCache, setDiscoverElastiCache] = useState(true);
    const [discoverDocumentDB, setDiscoverDocumentDB] = useState(true);

    // GraphQL mutations - refetch providers and connections after add/update
    const [addProvider, { loading: addLoading }] = useAddAwsProviderMutation({
        refetchQueries: ['GetCloudProviders', 'GetDiscoveredConnections'],
    });
    const [updateProvider, { loading: updateLoading }] = useUpdateAwsProviderMutation({
        refetchQueries: ['GetCloudProviders', 'GetDiscoveredConnections'],
    });
    const [testProvider, { loading: testLoading }] = useTestCloudProviderMutation();

    const loading = addLoading || updateLoading || testLoading;

    // Reset form when modal opens/closes or editingProvider changes
    useEffect(() => {
        if (open) {
            if (editingProvider) {
                setName(editingProvider.Name);
                // Check if region is in the predefined list
                const isCustomRegion = !awsRegions.some(r => r.Id === editingProvider.Region);
                if (isCustomRegion) {
                    setRegion("custom");
                    setCustomRegion(editingProvider.Region);
                } else {
                    setRegion(editingProvider.Region);
                    setCustomRegion("");
                }
                setProfileName(editingProvider.ProfileName ?? "");
                setDiscoverRDS(editingProvider.DiscoverRDS);
                setDiscoverElastiCache(editingProvider.DiscoverElastiCache);
                setDiscoverDocumentDB(editingProvider.DiscoverDocumentDB);
            } else {
                // Reset to defaults for add mode
                setName("");
                setRegion("us-east-1");
                setCustomRegion("");
                setProfileName("");
                setDiscoverRDS(true);
                setDiscoverElastiCache(true);
                setDiscoverDocumentDB(true);
            }
        }
    }, [open, editingProvider]);

    const handleClose = useCallback(() => {
        onOpenChange(false);
        dispatch(ProvidersActions.closeProviderModal());
    }, [onOpenChange, dispatch]);

    /** Auto-fill form from a discovered local profile. */
    const handleSelectLocalProfile = useCallback((profile: LocalAwsProfile) => {
        // Set name based on profile
        const displayName = profile.IsDefault
            ? t('defaultAwsName')
            : t('awsProfileName', { name: profile.Name });
        setName(displayName);

        // Set region if available, otherwise use us-east-1
        if (profile.Region) {
            const isKnownRegion = awsRegions.some(r => r.Id === profile.Region);
            if (isKnownRegion) {
                setRegion(profile.Region);
                setCustomRegion("");
            } else {
                setRegion("custom");
                setCustomRegion(profile.Region);
            }
        }

        // Set profile name (env source uses default chain, file-based uses profile)
        if (profile.Source === "environment") {
            setProfileName("");
        } else {
            setProfileName(profile.Name);
        }
    }, [t, awsRegions]);

    const getEffectiveRegion = useCallback(() => {
        return region === "custom" ? customRegion : region;
    }, [region, customRegion]);

    const buildInput = useCallback((): AwsProviderInput => {
        const input: AwsProviderInput = {
            Name: name,
            Region: getEffectiveRegion(),
            DiscoverRDS: discoverRDS,
            DiscoverElastiCache: discoverElastiCache,
            DiscoverDocumentDB: discoverDocumentDB,
        };

        if (profileName) {
            input.ProfileName = profileName;
        }

        return input;
    }, [name, getEffectiveRegion, profileName, discoverRDS, discoverElastiCache, discoverDocumentDB]);

    const handleSubmit = useCallback(async () => {
        if (!name.trim()) {
            toast.error(t('nameRequired'));
            return;
        }

        const effectiveRegion = getEffectiveRegion();
        if (!effectiveRegion.trim()) {
            toast.error(t('regionRequired'));
            return;
        }

        const input = buildInput();

        try {
            if (isEditMode && editingProviderId) {
                const { data } = await updateProvider({
                    variables: { id: editingProviderId, input },
                });
                if (data?.UpdateAWSProvider) {
                    dispatch(ProvidersActions.updateCloudProvider(data.UpdateAWSProvider as LocalCloudProvider));
                    toast.success(t('providerUpdated'));
                    handleClose();
                }
            } else {
                const { data } = await addProvider({
                    variables: { input },
                });
                if (data?.AddAWSProvider) {
                    dispatch(ProvidersActions.addCloudProvider(data.AddAWSProvider as LocalCloudProvider));
                    toast.success(t('providerAdded'));
                    handleClose();
                }
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('operationFailed', { error: errorMessage }));
        }
    }, [name, getEffectiveRegion, buildInput, isEditMode, editingProviderId, updateProvider, addProvider, dispatch, handleClose, t]);

    const handleTest = useCallback(async () => {
        if (!editingProviderId) {
            toast.error(t('saveBeforeTest'));
            return;
        }

        try {
            const { data } = await testProvider({
                variables: { id: editingProviderId },
            });
            if (data?.TestCloudProvider === CloudProviderStatus.Connected) {
                dispatch(ProvidersActions.setProviderStatus({
                    id: editingProviderId,
                    status: CloudProviderStatus.Connected,
                }));
                toast.success(t('connectionSuccessful'));
            } else {
                dispatch(ProvidersActions.setProviderStatus({
                    id: editingProviderId,
                    status: CloudProviderStatus.Error,
                }));
                toast.error(t('connectionFailed'));
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('testFailed', { error: errorMessage }));
        }
    }, [editingProviderId, testProvider, dispatch, t]);

    const regionOptions = useMemo(() => [
        ...awsRegions.map(r => ({ value: r.Id, label: `${r.Id} - ${r.Description}` })),
        { value: "custom", label: t('customRegion') },
    ], [awsRegions, t]);

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto" data-testid="aws-provider-modal">
                <DialogHeader>
                    <DialogTitle>
                        {isEditMode ? t('editProvider') : t('addProvider')}
                    </DialogTitle>
                </DialogHeader>

                <div className="flex flex-col gap-4 py-4">
                    {/* Discovered Local Profiles */}
                    {!isEditMode && localProfiles.length > 0 && (
                        <div className="flex flex-col gap-3">
                            <div className="flex items-center gap-2">
                                <CloudIcon className="w-4 h-4 text-muted-foreground" />
                                <Label className="text-sm font-medium">{t('detectedProfiles')}</Label>
                            </div>
                            <div className="flex flex-wrap gap-2">
                                {localProfiles.map((profile) => {
                                    // Map source to localized display
                                    const sourceKey = {
                                        credentials: 'sourceCredentials',
                                        config: 'sourceConfig',
                                        environment: 'sourceEnvironment',
                                    }[profile.Source];
                                    const sourceDisplay = sourceKey ? t(sourceKey) : profile.Source;

                                    return (
                                        <button
                                            key={`${profile.Source}-${profile.Name}`}
                                            type="button"
                                            onClick={() => handleSelectLocalProfile(profile)}
                                            className={cn(
                                                "flex items-center gap-2 px-3 py-2 rounded-md border text-sm transition-colors",
                                                "hover:border-brand hover:bg-brand/5",
                                                "focus:outline-none focus:ring-2 focus:ring-brand focus:ring-offset-2"
                                            )}
                                            title={t('profileSourceTooltip', { source: sourceDisplay })}
                                        >
                                            <span className="font-medium">
                                                {profile.IsDefault ? t('defaultProfileName') : profile.Name}
                                            </span>
                                            {profile.Region && (
                                                <Badge variant="outline" className="text-xs">
                                                    {profile.Region}
                                                </Badge>
                                            )}
                                            <Badge variant="secondary" className="text-xs font-normal">
                                                {sourceDisplay}
                                            </Badge>
                                        </button>
                                    );
                                })}
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

                    {/* Region */}
                    <div className="flex flex-col gap-2">
                        <Label>{t('awsRegion')}</Label>
                        <SearchSelect
                            value={region}
                            onChange={setRegion}
                            options={regionOptions}
                            placeholder={t('selectRegion')}
                            contentClassName="w-[var(--radix-popover-trigger-width)]"
                            rightIcon={<ChevronDownIcon className="w-4 h-4" />}
                        />
                        {region === "custom" && (
                            <Input
                                value={customRegion}
                                onChange={(e) => setCustomRegion(e.target.value)}
                                placeholder={t('customRegionPlaceholder')}
                                data-testid="custom-region"
                                className="mt-2"
                            />
                        )}
                    </div>

                    {/* Profile name */}
                    <div className="flex flex-col gap-2">
                        <Label htmlFor="profile-name">{t('profileName')}</Label>
                        <Input
                            id="profile-name"
                            value={profileName}
                            onChange={(e) => setProfileName(e.target.value)}
                            placeholder={t('profilePlaceholder')}
                            data-testid="profile-name"
                        />
                        <p className="text-xs text-muted-foreground">
                            {t('profileDesc')}
                        </p>
                    </div>

                    {/* Discovery toggles */}
                    <div className="flex flex-col gap-3 pt-4 border-t">
                        <Label className="font-medium">{t('resourceDiscovery')}</Label>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('rdsLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('rdsDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverRDS}
                                onCheckedChange={setDiscoverRDS}
                                data-testid="discover-rds"
                            />
                        </div>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('elasticacheLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('elasticacheDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverElastiCache}
                                onCheckedChange={setDiscoverElastiCache}
                                data-testid="discover-elasticache"
                            />
                        </div>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('documentdbLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('documentdbDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverDocumentDB}
                                onCheckedChange={setDiscoverDocumentDB}
                                data-testid="discover-documentdb"
                            />
                        </div>
                    </div>
                </div>

                <DialogFooter className="flex gap-2">
                    {isEditMode && (
                        <Button
                            variant="outline"
                            onClick={handleTest}
                            disabled={loading}
                            data-testid="test-connection"
                        >
                            {testLoading ? t('testing') : t('testConnection')}
                        </Button>
                    )}
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
