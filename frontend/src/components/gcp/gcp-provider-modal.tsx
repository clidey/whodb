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
    GcpProviderInput,
    GcpProvider,
    LocalGcpProject} from "@graphql";
import {
    AddGcpProviderDocument,
    CloudProviderStatus,
    GetDiscoveredConnectionsDocument,
    GetGcpRegionsDocument,
    GetLocalGcpProjectsDocument,
    TestCloudProviderDocument,
    TestGcpCredentialsDocument,
    UpdateGcpProviderDocument,
} from "@graphql";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import type { LocalCloudProvider } from "../../store/providers";
import { ProvidersActions } from "../../store/providers";
import { useTranslation } from "@/hooks/use-translation";
import { ChevronDownIcon, CloudIcon } from "../heroicons";
import { upsertCloudProviderCache } from "../../utils/apollo-provider-cache";

export interface GcpProviderModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export const GcpProviderModal: FC<GcpProviderModalProps> = ({
    open,
    onOpenChange,
}) => {
    const { t } = useTranslation('components/gcp-provider-modal');
    const dispatch = useAppDispatch();

    // Get editing state from Redux
    const editingProviderId = useAppSelector(state => state.providers.editingProviderId);
    const cloudProviders = useAppSelector(state => state.providers.cloudProviders);
    const editingProvider = useMemo(() => {
        if (!editingProviderId) return null;
        const found = cloudProviders.find(p => p.Id === editingProviderId);
        if (!found || found.__typename !== 'GCPProvider') return null;
        return found as GcpProvider;
    }, [editingProviderId, cloudProviders]);

    const isEditMode = editingProviderId !== null;
    const localProjectsQueryOptions = isEditMode ? skipToken : {};

    // Query local GCP projects
    const { data: localProjectsData } = useQuery(GetLocalGcpProjectsDocument, localProjectsQueryOptions);
    const localProjects = localProjectsData?.LocalGCPProjects ?? [];

    // Query GCP regions from backend
    const { data: regionsData } = useQuery(GetGcpRegionsDocument);
    const gcpRegions = regionsData?.GCPRegions ?? [];

    // Form state
    const [name, setName] = useState("");
    const [projectId, setProjectId] = useState("");
    const [region, setRegion] = useState("us-central1");
    const [serviceAccountKeyPath, setServiceAccountKeyPath] = useState("");
    const [discoverCloudSQL, setDiscoverCloudSQL] = useState(true);
    const [discoverAlloyDB, setDiscoverAlloyDB] = useState(true);
    const [discoverMemorystore, setDiscoverMemorystore] = useState(true);

    // GraphQL mutations
    const [addProvider, { loading: addLoading }] = useMutation(AddGcpProviderDocument, {
        refetchQueries: [GetDiscoveredConnectionsDocument],
        update(cache, { data }) {
            if (data?.AddGCPProvider) {
                upsertCloudProviderCache(cache, data.AddGCPProvider);
            }
        },
    });
    const [updateProvider, { loading: updateLoading }] = useMutation(UpdateGcpProviderDocument, {
        refetchQueries: [GetDiscoveredConnectionsDocument],
        update(cache, { data }) {
            if (data?.UpdateGCPProvider) {
                upsertCloudProviderCache(cache, data.UpdateGCPProvider);
            }
        },
    });
    const [testProvider, { loading: testLoading }] = useMutation(TestCloudProviderDocument);
    const [testCredentials, { loading: testCredentialsLoading }] = useMutation(TestGcpCredentialsDocument);

    const loading = addLoading || updateLoading || testLoading || testCredentialsLoading;

    // Reset form when modal opens/closes or editingProvider changes
    useEffect(() => {
        if (open) {
            if (editingProvider) {
                setName(editingProvider.Name);
                setProjectId(editingProvider.ProjectID);
                setRegion(editingProvider.Region);
                setServiceAccountKeyPath(editingProvider.ServiceAccountKeyPath ?? "");
                setDiscoverCloudSQL(editingProvider.DiscoverCloudSQL);
                setDiscoverAlloyDB(editingProvider.DiscoverAlloyDB);
                setDiscoverMemorystore(editingProvider.DiscoverMemorystore);
            } else {
                setName("");
                setProjectId("");
                setRegion("us-central1");
                setServiceAccountKeyPath("");
                setDiscoverCloudSQL(true);
                setDiscoverAlloyDB(true);
                setDiscoverMemorystore(true);
            }
        }
    }, [open, editingProvider]);

    const handleClose = useCallback(() => {
        onOpenChange(false);
        dispatch(ProvidersActions.closeProviderModal());
    }, [onOpenChange, dispatch]);

    /** Auto-fill form from a discovered local project. */
    const handleSelectLocalProject = useCallback((project: LocalGcpProject) => {
        const displayName = project.IsDefault
            ? t('defaultGcpName')
            : t('gcpProjectName', { name: project.Name });
        setName(displayName);
        setProjectId(project.ProjectID);
    }, [t]);

    const buildInput = useCallback((): GcpProviderInput => {
        const input: GcpProviderInput = {
            Name: name,
            ProjectID: projectId,
            Region: region,
            DiscoverCloudSQL: discoverCloudSQL,
            DiscoverAlloyDB: discoverAlloyDB,
            DiscoverMemorystore: discoverMemorystore,
        };

        if (serviceAccountKeyPath) {
            input.ServiceAccountKeyPath = serviceAccountKeyPath;
        }

        return input;
    }, [name, projectId, region, serviceAccountKeyPath, discoverCloudSQL, discoverAlloyDB, discoverMemorystore]);

    const handleSubmit = useCallback(async () => {
        if (!name.trim()) {
            toast.error(t('nameRequired'));
            return;
        }

        if (!projectId.trim()) {
            toast.error(t('projectIdRequired'));
            return;
        }

        if (!region.trim()) {
            toast.error(t('regionRequired'));
            return;
        }

        const input = buildInput();

        try {
            if (isEditMode && editingProviderId) {
                const { data } = await updateProvider({
                    variables: { id: editingProviderId, input },
                });
                if (data?.UpdateGCPProvider) {
                    dispatch(ProvidersActions.updateCloudProvider(data.UpdateGCPProvider as LocalCloudProvider));
                    toast.success(t('providerUpdated'));
                    handleClose();
                }
            } else {
                const { data } = await addProvider({
                    variables: { input },
                });
                if (data?.AddGCPProvider) {
                    dispatch(ProvidersActions.addCloudProvider(data.AddGCPProvider as LocalCloudProvider));
                    toast.success(t('providerAdded'));
                    handleClose();
                }
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('operationFailed', { error: errorMessage }));
        }
    }, [name, projectId, region, buildInput, isEditMode, editingProviderId, updateProvider, addProvider, dispatch, handleClose, t]);

    const handleTest = useCallback(async () => {
        if (!name.trim() || !projectId.trim() || !region.trim()) {
            toast.error(t('nameRequired'));
            return;
        }

        try {
            if (isEditMode && editingProviderId) {
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
            } else {
                const input = buildInput();
                const { data } = await testCredentials({
                    variables: { input },
                });
                if (data?.TestGCPCredentials === CloudProviderStatus.Connected) {
                    toast.success(t('connectionSuccessful'));
                } else {
                    toast.error(t('connectionFailed'));
                }
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : t('unknownError');
            toast.error(t('testFailed', { error: errorMessage }));
        }
    }, [name, projectId, region, isEditMode, editingProviderId, testProvider, testCredentials, buildInput, dispatch, t]);

    const regionOptions = useMemo(() => {
        return gcpRegions.map(r => ({
            value: r.Id,
            label: `${r.Id} - ${r.Description}`,
        }));
    }, [gcpRegions]);

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto" data-testid="gcp-provider-modal">
                <DialogHeader>
                    <DialogTitle>
                        {isEditMode ? t('editProvider') : t('addGcpAccount')}
                    </DialogTitle>
                </DialogHeader>

                <div className="flex flex-col gap-4 py-4">
                    {/* Discovered Local Projects */}
                    {!isEditMode && localProjects.length > 0 && (
                        <div className="flex flex-col gap-3">
                            <div className="flex items-center gap-2">
                                <CloudIcon className="w-4 h-4 text-muted-foreground" />
                                <Label className="text-sm font-medium">{t('detectedProjects')}</Label>
                            </div>
                            <div className="flex flex-wrap gap-2">
                                {localProjects.map((project) => {
                                    const sourceKey = {
                                        'gcloud-config': 'sourceGcloud',
                                        'environment': 'sourceEnvironment',
                                        'service-account': 'sourceServiceAccount',
                                    }[project.Source] as string | undefined;
                                    const sourceDisplay = sourceKey ? t(sourceKey) : project.Source;

                                    return (
                                        <button
                                            key={`${project.Source}-${project.ProjectID}`}
                                            type="button"
                                            onClick={() => handleSelectLocalProject(project)}
                                            className={cn(
                                                "flex items-center gap-2 px-3 py-2 rounded-md border text-sm transition-colors",
                                                "hover:border-brand hover:bg-brand/5",
                                                "focus:outline-none focus:ring-2 focus:ring-brand focus:ring-offset-2"
                                            )}
                                            title={t('foundInSource', { source: sourceDisplay })}
                                        >
                                            <span className="font-medium">
                                                {project.IsDefault ? t('defaultProjectName') : project.Name}
                                            </span>
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

                    {/* Project ID */}
                    <div className="flex flex-col gap-2">
                        <Label htmlFor="project-id">{t('projectId')}</Label>
                        <Input
                            id="project-id"
                            value={projectId}
                            onChange={(e) => setProjectId(e.target.value)}
                            placeholder={t('projectIdPlaceholder')}
                            data-testid="project-id"
                        />
                        <p className="text-xs text-muted-foreground">
                            {t('projectIdDesc')}
                        </p>
                    </div>

                    {/* Region */}
                    <div className="flex flex-col gap-2">
                        <Label>{t('gcpRegion')}</Label>
                        <SearchSelect
                            value={region}
                            onChange={setRegion}
                            options={regionOptions}
                            placeholder={t('selectRegion')}
                            contentClassName="w-[var(--radix-popover-trigger-width)]"
                            rightIcon={<ChevronDownIcon className="w-4 h-4" />}
                        />
                    </div>

                    {/* Service Account Key Path */}
                    <div className="flex flex-col gap-2">
                        <Label htmlFor="service-account-key">{t('serviceAccountKeyPath')}</Label>
                        <Input
                            id="service-account-key"
                            value={serviceAccountKeyPath}
                            onChange={(e) => setServiceAccountKeyPath(e.target.value)}
                            placeholder={t('serviceAccountKeyPlaceholder')}
                            data-testid="service-account-key"
                        />
                        <p className="text-xs text-muted-foreground">
                            {t('serviceAccountKeyDesc')}
                        </p>
                    </div>

                    {/* Discovery toggles */}
                    <div className="flex flex-col gap-3 pt-4 border-t">
                        <Label className="font-medium">{t('resourceDiscovery')}</Label>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('cloudSqlLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('cloudSqlDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverCloudSQL}
                                onCheckedChange={setDiscoverCloudSQL}
                                data-testid="discover-cloudsql"
                            />
                        </div>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('alloyDbLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('alloyDbDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverAlloyDB}
                                onCheckedChange={setDiscoverAlloyDB}
                                data-testid="discover-alloydb"
                            />
                        </div>
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm">{t('memorystoreLabel')}</span>
                                <span className="text-xs text-muted-foreground">{t('memorystoreDesc')}</span>
                            </div>
                            <Switch
                                checked={discoverMemorystore}
                                onCheckedChange={setDiscoverMemorystore}
                                data-testid="discover-memorystore"
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
                        {(testLoading || testCredentialsLoading) ? t('testing') : t('testConnection')}
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
