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

import {skipToken, useQuery} from "@apollo/client/react";
import {
    Button,
    cn,
    CommandItem,
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
    Sheet,
    SheetContent,
    SheetTitle,
    Sidebar as SidebarComponent,
    SidebarContent,
    SidebarGroup,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarSeparator,
    SidebarTrigger,
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
    useSidebar
} from "@clidey/ux";
import {SearchSelect} from "../ux";
import {
    GetSchemaDocument,
    GetSslStatusDocument,
    GetUpdateInfoDocument,
    SourceProfileLabelStrategy,
    SourceFieldOptionsDocument,
} from '@graphql';
import {useTranslation} from '@/hooks/use-translation';
import {VisuallyHidden} from "@radix-ui/react-visually-hidden";
import {FC, LazyExoticComponent, ReactElement, ReactNode, Suspense, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {useDispatch} from "react-redux";
import {Link, useLocation, useNavigate} from "react-router-dom";
import logoImage from "../../../public/images/logo.svg";
import {extensions, getAppName} from "../../config/features";
import {InternalRoutes} from "../../config/routes";
import {LoginForm} from "../../pages/auth/login";
import {AuthActions, LocalLoginProfile} from "../../store/auth";
import {DatabaseActions} from "../../store/database";
import {useAppSelector} from "../../store/hooks";
import {featureFlags} from "../../config/features";
import {getComponent} from "../../config/component-registry";
import { findSourceTypeItem, type SourceTypeItem } from "../../config/source-types";
import {isAwsHostname, isAzureHostname, isGcpHostname} from "../../utils/cloud-connection-prefill";
import {useSourceContract} from "../../hooks/useSourceContract";
import {useSourceTypeItems} from "../../hooks/useSourceCatalog";
import {
    ArrowLeftStartOnRectangleIcon,
    ChevronDownIcon,
    CogIcon,
    CommandLineIcon,
    InformationCircleIcon,
    PlusCircleIcon,
    QuestionMarkCircleIcon,
    RectangleGroupIcon,
    SparklesIcon,
    TableCellsIcon
} from "../heroicons";
import {Icons} from "../icons";
import {Loading} from "../loading";
import {DatabaseIconWithBadge, isAwsConnection} from "../aws";
import {isAzureConnection} from "../azure";
import {isGcpConnection} from "../gcp";
import {useProfileSwitch} from "@/hooks/use-profile-switch";
import {buildSourceSchemaQuery} from "@/utils/source-refs";

function getProfileLabel(profile: LocalLoginProfile, item: SourceTypeItem | undefined) {
    if (profile.Saved) return profile.Id;
    if (item?.traits?.presentation.profileLabelStrategy === SourceProfileLabelStrategy.Hostname && profile.Hostname) {
        return profile.Hostname;
    }
    if (item?.traits?.presentation.profileLabelStrategy === SourceProfileLabelStrategy.Database && profile.Database) {
        return profile.Database;
    }
    if (profile.Hostname && profile.Username) {
        return `${profile.Hostname} [${profile.Username}]`;
    }
    if (profile.Database) {
        return profile.Database;
    }
    if (profile.Hostname) {
        return profile.Hostname;
    }
    return profile.Type;
}

function getProfileIcon(profile: LocalLoginProfile) {
    return (Icons.Logos as Record<string, ReactElement>)[profile.Type];
}

/** Section header shown above each navigation group in the sidebar. */
export const NavSectionHeader: FC<{ label: string; open: boolean; disabled?: boolean }> = ({ label, open, disabled }) => {
    if (!open) return <div className="h-4" />;
    return (
        <div className="flex items-center gap-2 px-1 pt-5 pb-1">
            <span className="text-[10px] font-semibold uppercase tracking-[0.2em] text-neutral-600 select-none">
                {label}
            </span>
            {disabled && (
                <span className="text-[9px] uppercase tracking-wider text-neutral-700 border border-neutral-800 rounded px-1 py-0.5 leading-none">
                    Soon
                </span>
            )}
        </div>
    );
};

/** A single navigation item with active-state styling. */
export const NavItem: FC<{
    icon: React.ReactNode;
    label: string;
    path: string;
    pathname: string;
    open: boolean;
    tooltip: string;
    disabled?: boolean;
}> = ({ icon, label, path, pathname, open, tooltip, disabled }) => {
    const isActive = pathname === path;

    if (disabled) {
        return (
            <SidebarMenuItem>
                <SidebarMenuButton tooltip={tooltip} className="opacity-35 cursor-not-allowed pointer-events-none">
                    <span className="text-neutral-600">{icon}</span>
                    {open && <span className="text-neutral-600 text-sm">{label}</span>}
                </SidebarMenuButton>
            </SidebarMenuItem>
        );
    }

    return (
        <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip={tooltip}>
                <Link
                    to={path}
                    className={cn(
                        "flex items-center gap-2 transition-all duration-150",
                        isActive
                            ? "text-blue-300 font-medium"
                            : "text-neutral-500 hover:text-neutral-200"
                    )}
                >
                    <span className={cn(
                        "flex-shrink-0",
                        isActive ? "text-blue-400" : "text-neutral-600"
                    )}>{icon}</span>
                    {open && <span className="text-sm">{label}</span>}
                    {open && isActive && (
                        <span className="ml-auto w-1 h-1 rounded-full bg-blue-400" />
                    )}
                </Link>
            </SidebarMenuButton>
        </SidebarMenuItem>
    );
};

/** Props passed to the EE sidebar nav component when registered. */
export type EESidebarNavProps = {
    open: boolean;
    pathname: string;
    dropdowns: ReactNode;
};

export const Sidebar: FC = () => {
    const { t } = useTranslation('components/sidebar');
    const schema = useAppSelector(state => state.database.schema);
    const databaseSchemaTerminology = useAppSelector(state => state.settings.databaseSchemaTerminology);
    const awsProviderEnabled = useAppSelector(state => state.settings.awsProviderEnabled);
    const azureProviderEnabled = useAppSelector(state => state.settings.azureProviderEnabled);
    const gcpProviderEnabled = useAppSelector(state => state.settings.gcpProviderEnabled);
    const newUIEnabled = useAppSelector(state => state.settings.newUIEnabled);
    const isEmbedded = useAppSelector(state => state.auth.isEmbedded);
    const dispatch = useDispatch();
    const pathname = useLocation().pathname;
    const current = useAppSelector(state => state.auth.current);
    const profiles = useAppSelector(state => state.auth.profiles);
    const sslStatus = useAppSelector(state => state.auth.sslStatus);
    const {
        item,
        storageUnitLabel,
        supportsChat,
        supportsGraph,
        supportsScratchpad,
        supportsSchema,
        supportsDatabaseSwitching,
        usesDatabaseInsteadOfSchema,
    } = useSourceContract(current?.Type);
    const databaseQueryOptions = current != null && supportsDatabaseSwitching && current.Type
        ? {
            variables: {
                sourceType: current.Type,
            },
        }
        : skipToken;
    const schemaQueryVariables = useMemo(() => buildSourceSchemaQuery(item, current), [current, item]);
    const schemaQueryOptions = current != null && supportsSchema
        ? {
            variables: schemaQueryVariables,
        }
        : skipToken;
    const sslStatusQueryOptions = current != null && sslStatus === undefined
        ? {}
        : skipToken;
    const {data: availableDatabases, loading: availableDatabasesLoading, refetch: getDatabases} = useQuery(SourceFieldOptionsDocument, databaseQueryOptions);
    const { data: availableSchemas, loading: availableSchemasLoading, refetch: getSchemas } = useQuery(GetSchemaDocument, schemaQueryOptions);
    const loading = availableDatabasesLoading || availableSchemasLoading;
    const availableSchemaNames = useMemo(() => {
        return availableSchemas?.Schema?.map(schemaObject => schemaObject.Name) ?? [];
    }, [availableSchemas?.Schema]);

    // Default schema selection: prefer Search Path, then profile database for schema-scoped
    // sources, then fall back to the first available schema.
    useEffect(() => {
        if (current == null || schema !== "" || availableSchemaNames.length === 0) return;
        const searchPath = current.Advanced?.find(a => a.Key === "Search Path")?.Value;
        const profileDatabase = (!supportsDatabaseSwitching && current.Database && availableSchemaNames.includes(current.Database))
            ? current.Database
            : undefined;
        const defaultSchema = (searchPath && availableSchemaNames.includes(searchPath))
            ? searchPath
            : (profileDatabase ?? availableSchemaNames[0] ?? "");
        dispatch(DatabaseActions.setSchema(defaultSchema));
    }, [availableSchemaNames, current, dispatch, schema, supportsDatabaseSwitching]);
    const { data: updateInfo } = useQuery(GetUpdateInfoDocument);
    const { data: sslStatusData, refetch: refetchSslStatus } = useQuery(GetSslStatusDocument, sslStatusQueryOptions);
    useEffect(() => {
        if (sslStatusData?.SSLStatus) {
            dispatch(AuthActions.setSSLStatus(sslStatusData.SSLStatus));
        }
    }, [dispatch, sslStatusData?.SSLStatus]);
    const navigate = useNavigate();
    const [showLoginCard, setShowLoginCard] = useState(false);
    const [showProfileSwitchDialog, setShowProfileSwitchDialog] = useState(false);
    const [logoutProfileId, setLogoutProfileId] = useState<string | null>(null);
    const { toggleSidebar, open } = useSidebar();
    const isInitialMount = useRef(true);
    const { switchProfile } = useProfileSwitch({
        errorMessage: t('errorSigningIn'),
    });
    const { items: sourceTypeItems } = useSourceTypeItems();

    const profileOptions = useMemo(() => profiles
        .filter(profile => {
            if (isAwsHostname(profile.Hostname)) return awsProviderEnabled;
            if (isAzureHostname(profile.Hostname)) return azureProviderEnabled;
            if (isGcpHostname(profile.Hostname)) return gcpProviderEnabled;
            return true;
        })
        .map(profile => ({
            value: profile.Id,
            label: getProfileLabel(profile, findSourceTypeItem(sourceTypeItems, profile.Type)),
            icon: (
                <DatabaseIconWithBadge
                    icon={getProfileIcon(profile)}
                    showCloudBadge={isAwsConnection(profile.Id) || isAzureConnection(profile.Id) || isGcpConnection(profile.Id)}
                    sslStatus={profile.Id === current?.Id
                        ? sslStatus
                        : (profile.SSLConfigured ? { IsEnabled: true, Mode: 'configured' } : undefined)}
                    size="sm"
                />
            ),
            profile,
        })), [profiles, current?.Id, sslStatus, awsProviderEnabled, azureProviderEnabled, gcpProviderEnabled, sourceTypeItems]);

    const currentProfileOption = useMemo(() => {
        if (!current) return undefined;
        return profileOptions.find(opt => opt.value === current.Id);
    }, [current, profileOptions]);

    const handleProfileChange = useCallback(async (value: string, database?: string) => {
        const selectedProfile = profiles.find(profile => profile.Id === value);
        if (!selectedProfile) return;

        await switchProfile(selectedProfile, database);
    }, [profiles, switchProfile]);

    // Database select logic
    const databaseOptions = useMemo(() => {
        if (!availableDatabases?.SourceFieldOptions) return [];
        return availableDatabases.SourceFieldOptions.map(db => ({
            value: db,
            label: db,
        }));
    }, [availableDatabases?.SourceFieldOptions]);
    
    const handleDatabaseChange = useCallback((value: string) => {
        if (value === "") {
            return;
        }
        if (!current?.Id) return;
        if (pathname !== InternalRoutes.Graph.path && pathname !== InternalRoutes.Dashboard.StorageUnit.path) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
        handleProfileChange(current.Id, value);
    }, [current, handleProfileChange, navigate, pathname]);

    // Schema select logic
    const schemaOptions = useMemo(() => {
        return availableSchemaNames.map(schemaName => ({
            value: schemaName,
            label: schemaName,
        })) ?? [];
    }, [availableSchemaNames]);

    const handleSchemaChange = useCallback((value: string) => {
        if (value === "") {
            return;
        }
        if (pathname === InternalRoutes.Dashboard.StorageUnit.path) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path, {
                replace: true,
                state: {},
            });
        } else if (pathname !== InternalRoutes.Graph.path) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
        dispatch(DatabaseActions.setSchema(value));
    }, [dispatch, navigate, pathname]);

    // Sidebar routes
    const sidebarRoutes = useMemo(() => {
        if (!current) return [];
        const routes = [
            {
                title: storageUnitLabel,
                icon: <TableCellsIcon className="w-4 h-4" />,
                path: InternalRoutes.Dashboard.StorageUnit.path,
            },
        ];
        if (supportsChat) {
            routes.unshift({
                title: t('chat'),
                icon: <SparklesIcon className="w-4 h-4" />,
                path: InternalRoutes.Chat.path,
            });
        }
        if (supportsGraph) {
            routes.push({
                title: t('graph'),
                icon: <RectangleGroupIcon className="w-4 h-4" />,
                path: InternalRoutes.Graph.path,
            });
        }
        if (supportsScratchpad) {
            routes.push({
                title: t('scratchpad'),
                icon: <CommandLineIcon className="w-4 h-4" />,
                path: InternalRoutes.RawExecute.path,
            });
        }
        return routes;
    }, [current, storageUnitLabel, supportsChat, supportsGraph, supportsScratchpad, t]);

    // Logout single profile — show dialog first, remove after switch
    const handleLogoutProfile = useCallback(() => {
        if (!current) return;
        const remainingProfiles = profiles.filter(p => p.Id !== current.Id);
        if (remainingProfiles.length === 0) {
            navigate(InternalRoutes.Logout.path);
            return;
        }
        setLogoutProfileId(current.Id);
        setShowProfileSwitchDialog(true);
    }, [current, profiles, navigate]);

    // Logout all profiles
    const handleLogoutAll = useCallback(() => {
        navigate(InternalRoutes.Logout.path);
    }, [navigate]);

    // Profile switch dialog handlers
    const handleDialogProfileSwitch = useCallback(async (profile: LocalLoginProfile) => {
        setShowProfileSwitchDialog(false);
        await switchProfile(profile);
        if (logoutProfileId) {
            dispatch(AuthActions.remove({ id: logoutProfileId }));
            setLogoutProfileId(null);
        }
    }, [switchProfile, logoutProfileId, dispatch]);

    const handleDialogAddProfile = useCallback(() => {
        setShowProfileSwitchDialog(false);
        // logoutProfileId stays set so onLoginSuccess can clean it up
        setShowLoginCard(true);
    }, []);

    const handleLoginSuccess = useCallback(() => {
        setShowLoginCard(false);
        if (logoutProfileId) {
            dispatch(AuthActions.remove({ id: logoutProfileId }));
            setLogoutProfileId(null);
        }
    }, [logoutProfileId, dispatch]);

    const handleProfileSwitchDialogChange = useCallback((open: boolean) => {
        if (!open) {
            setLogoutProfileId(null);
        }
        setShowProfileSwitchDialog(open);
    }, []);

    // Add profile logic
    const handleAddProfile = useCallback(() => {
        // Small delay to allow dropdown to close before opening sheet
        setTimeout(() => {
            setShowLoginCard(true);
        }, 100);
    }, []);

    // Load EE sidebar nav component if registered
    const EESidebarNav = getComponent('ee-sidebar-nav') as LazyExoticComponent<FC<EESidebarNavProps>> | undefined;

    // Compute the label for the database dropdown based on the database type and user terminology preference
    const databaseDropdownLabel = useMemo(() => {
        // For databases where database=schema (MySQL, MariaDB, ClickHouse, MongoDB, Redis), allow user to choose terminology
        if (usesDatabaseInsteadOfSchema) {
            return databaseSchemaTerminology === 'schema' ? t('schema') : t('database');
        }
        // For all other databases, use "Database"
        return t('database');
    }, [databaseSchemaTerminology, t, usesDatabaseInsteadOfSchema]);

    useEffect(() => {
        if (pathname.includes(InternalRoutes.Dashboard.ExploreStorageUnit.path) && open) {
            toggleSidebar();
        }
    }, []);

    // Refetch databases, schemas, and SSL status when the connection context changes
    // (profile switch or database switch within the same profile)
    useEffect(() => {
        if (isInitialMount.current) {
            isInitialMount.current = false;
            return;
        }
        if (!current) return;
        if (supportsDatabaseSwitching && current.Type) {
            getDatabases({ sourceType: current.Type });
        }
        if (supportsSchema) {
            getSchemas();
        }
        refetchSslStatus().then(({ data }) => {
            if (data?.SSLStatus) {
                dispatch(AuthActions.setSSLStatus(data.SSLStatus));
            }
        });
    }, [current, dispatch, getDatabases, getSchemas, refetchSslStatus, supportsDatabaseSwitching, supportsSchema]);

    // Listen for menu event to open add profile form
    useEffect(() => {
        const handleOpenAddProfile = () => {
            // Open the sidebar if it's closed
            if (!open) {
                toggleSidebar();
            }
            // Open the add profile sheet
            setShowLoginCard(true);
        };

        const handleToggleSidebar = () => {
            toggleSidebar();
        };

        window.addEventListener('menu:open-add-profile', handleOpenAddProfile);
        window.addEventListener('menu:toggle-sidebar', handleToggleSidebar);
        return () => {
            window.removeEventListener('menu:open-add-profile', handleOpenAddProfile);
            window.removeEventListener('menu:toggle-sidebar', handleToggleSidebar);
        };
    }, [open, toggleSidebar]);

    return (
        <nav className="dark" aria-label={t('mainNavigation')}>
            <SidebarComponent
                variant="sidebar"
                collapsible="icon"
                className="dark:group-data-[side=left]:border-r-neutral-800 z-[50]"
            >
                <SidebarHeader className={cn({ "ml-4": open })}>
                    <div className="flex items-center gap-sm justify-between">
                        <div className={cn("flex items-center gap-sm mt-2", { "hidden": !open })}>
                            {extensions.Logo ?? <img src={logoImage} alt="clidey logo" className={cn("w-auto", newUIEnabled ? "h-6" : "h-8")} />}
                            {open && <span className={cn("font-bold", newUIEnabled ? "text-lg" : "text-3xl")} data-testid="app-name">{getAppName()}</span>}
                        </div>
                        <SidebarTrigger className="px-0" />
                    </div>
                </SidebarHeader>
                <SidebarContent className={cn("mt-8 mb-16 overflow-y-auto", { "mx-4": open })}>
                    {newUIEnabled ? (
                    <SidebarGroup className="grow">
                        <SidebarMenu className="gap-0 grow">
                            {EESidebarNav ? (
                                <Suspense fallback={null}>
                                    <EESidebarNav open={open} pathname={pathname} dropdowns={
                                        <>
                                            <SearchSelect
                                                label={t('profile')}
                                                options={profileOptions}
                                                value={currentProfileOption?.value}
                                                onChange={handleProfileChange}
                                                placeholder={t('selectProfile')}
                                                searchPlaceholder={t('searchProfile')}
                                                onlyIcon={false}
                                                extraOptions={!isEmbedded ? (
                                                    <CommandItem key="__add__" value="__add__" onSelect={handleAddProfile}>
                                                        <span className="flex items-center gap-sm text-green-500">
                                                            <PlusCircleIcon className="w-4 h-4 stroke-green-500" />
                                                            {t('addAnotherProfile')}
                                                        </span>
                                                    </CommandItem>
                                                ) : undefined}
                                                side="left" align="start"
                                                buttonProps={{ "data-testid": "sidebar-profile", "data-collapsed": !open }}
                                            />
                                            {supportsDatabaseSwitching && (
                                                <SearchSelect
                                                    label={databaseDropdownLabel}
                                                    options={databaseOptions}
                                                    value={current?.Database}
                                                    onChange={handleDatabaseChange}
                                                    placeholder={databaseSchemaTerminology === 'schema' && usesDatabaseInsteadOfSchema ? t('selectSchema') : t('selectDatabase')}
                                                    searchPlaceholder={databaseSchemaTerminology === 'schema' && usesDatabaseInsteadOfSchema ? t('searchSchema') : t('searchDatabase')}
                                                    side="left" align="start"
                                                    buttonProps={{ "data-testid": "sidebar-database", "aria-label": databaseDropdownLabel }}
                                                />
                                            )}
                                            {supportsSchema && !pathname.includes(InternalRoutes.RawExecute.path) && (
                                                <SearchSelect
                                                    label={t('schema')}
                                                    options={schemaOptions}
                                                    value={schema}
                                                    onChange={handleSchemaChange}
                                                    placeholder={t('selectSchema')}
                                                    searchPlaceholder={t('searchSchema')}
                                                    side="left" align="start"
                                                    buttonProps={{ "data-testid": "sidebar-schema" }}
                                                />
                                            )}
                                        </>
                                    } />
                                </Suspense>
                            ) : (
                                <>
                                    {open && (
                                        <div className="flex flex-col gap-sm mb-4">
                                            <SearchSelect
                                                label={t('profile')}
                                                options={profileOptions}
                                                value={currentProfileOption?.value}
                                                onChange={handleProfileChange}
                                                placeholder={t('selectProfile')}
                                                searchPlaceholder={t('searchProfile')}
                                                onlyIcon={!open}
                                                extraOptions={!isEmbedded ? (
                                                    <CommandItem key="__add__" value="__add__" onSelect={handleAddProfile}>
                                                        <span className="flex items-center gap-sm text-green-500">
                                                            <PlusCircleIcon className="w-4 h-4 stroke-green-500" />
                                                            {t('addAnotherProfile')}
                                                        </span>
                                                    </CommandItem>
                                                ) : undefined}
                                                side="left" align="start"
                                                buttonProps={{ "data-testid": "sidebar-profile", "data-collapsed": !open }}
                                            />
                                            {supportsDatabaseSwitching && (
                                                <SearchSelect
                                                    label={databaseDropdownLabel}
                                                    options={databaseOptions}
                                                    value={current?.Database}
                                                    onChange={handleDatabaseChange}
                                                    placeholder={databaseSchemaTerminology === 'schema' && usesDatabaseInsteadOfSchema ? t('selectSchema') : t('selectDatabase')}
                                                    searchPlaceholder={databaseSchemaTerminology === 'schema' && usesDatabaseInsteadOfSchema ? t('searchSchema') : t('searchDatabase')}
                                                    side="left" align="start"
                                                    buttonProps={{ "data-testid": "sidebar-database", "aria-label": databaseDropdownLabel }}
                                                />
                                            )}
                                            {supportsSchema && !pathname.includes(InternalRoutes.RawExecute.path) && (
                                                <SearchSelect
                                                    label={t('schema')}
                                                    options={schemaOptions}
                                                    value={schema}
                                                    onChange={handleSchemaChange}
                                                    placeholder={t('selectSchema')}
                                                    searchPlaceholder={t('searchSchema')}
                                                    side="left" align="start"
                                                    buttonProps={{ "data-testid": "sidebar-schema" }}
                                                />
                                            )}
                                        </div>
                                    )}
                                    {sidebarRoutes.map(route => (
                                        <NavItem
                                            key={route.path}
                                            icon={route.icon}
                                            label={route.title}
                                            path={route.path}
                                            pathname={pathname}
                                            open={open}
                                            tooltip={route.title}
                                        />
                                    ))}

                                    <SidebarSeparator className={cn("my-4", { "mx-0": !open })} />

                                    {featureFlags.contactUsPage && InternalRoutes.ContactUs && (
                                        <NavItem icon={<QuestionMarkCircleIcon className="w-4 h-4" />} label={t('contactUs')} path={InternalRoutes.ContactUs.path} pathname={pathname} open={open} tooltip={t('contactUs')} />
                                    )}
                                    {featureFlags.settingsPage && InternalRoutes.Settings && (
                                        <NavItem icon={<CogIcon className="w-4 h-4" />} label={t('settings')} path={InternalRoutes.Settings.path} pathname={pathname} open={open} tooltip={t('settings')} />
                                    )}
                                </>
                            )}

                            {!EESidebarNav && <div className="grow" />}

                            {!EESidebarNav && !isEmbedded && (
                                <SidebarMenuItem className="flex justify-between items-center w-full">
                                    <SidebarMenuButton asChild tooltip={t('logOutProfile')}>
                                        <div className="flex items-center gap-sm text-nowrap w-fit cursor-pointer" onClick={handleLogoutProfile}>
                                            <ArrowLeftStartOnRectangleIcon className="w-4 h-4" />
                                            {open && <span>{t('logOutProfile')}</span>}
                                        </div>
                                    </SidebarMenuButton>
                                    <SidebarMenuButton asChild>
                                        <DropdownMenu>
                                            <DropdownMenuTrigger className={cn({ "hidden": !open })}>
                                                <Button
                                                    className="flex items-center justify-center p-1 rounded hover:bg-gray-100 dark:hover:bg-neutral-800 ml-2"
                                                    aria-label={t('moreLogoutOptions')}
                                                    variant="ghost"
                                                >
                                                    <ChevronDownIcon className="w-4 h-4" />
                                                </Button>
                                            </DropdownMenuTrigger>
                                            <DropdownMenuContent side="right" align="start">
                                                <DropdownMenuItem onClick={handleLogoutAll}>
                                                    <ArrowLeftStartOnRectangleIcon className="w-4 h-4" />
                                                    <span className="ml-2">{t('logoutAllProfiles')}</span>
                                                </DropdownMenuItem>
                                            </DropdownMenuContent>
                                        </DropdownMenu>
                                    </SidebarMenuButton>
                                </SidebarMenuItem>
                            )}
                        </SidebarMenu>
                    </SidebarGroup>
                    ) : loading ? (
                        <div className="flex justify-center items-center h-full">
                            <Loading size="lg" />
                        </div>
                    ) : (
                        <SidebarGroup className="grow">
                            <div className="flex flex-col gap-lg">
                                <div className="flex flex-col gap-sm w-full">
                                    <h2 className={cn("text-sm", { "hidden": !open })}>{t('profile')}</h2>
                                    <SearchSelect
                                        label={t('profile')}
                                        options={profileOptions}
                                        value={currentProfileOption?.value}
                                        onChange={handleProfileChange}
                                        placeholder={t('selectProfile')}
                                        searchPlaceholder={t('searchProfile')}
                                        onlyIcon={!open}
                                        extraOptions={!isEmbedded ? (
                                            <CommandItem
                                                key="__add__"
                                                value="__add__"
                                                onSelect={handleAddProfile}
                                            >
                                                <span className="flex items-center gap-sm text-green-500">
                                                    <PlusCircleIcon className="w-4 h-4 stroke-green-500" />
                                                    {t('addAnotherProfile')}
                                                </span>
                                            </CommandItem>
                                        ) : undefined}
                                        side="left" align="start"
                                        buttonProps={{
                                            "data-testid": "sidebar-profile",
                                            "data-collapsed": !open,
                                        }}
                                    />
                                </div>
                                {supportsDatabaseSwitching && (
                                    <div className={cn("flex flex-col gap-sm w-full", {
                                        "opacity-0 pointer-events-none": !open,
                                    })}>
                                        <h2 className="text-sm" data-testid="sidebar-database-label">{databaseDropdownLabel}</h2>
                                        <SearchSelect
                                            label={databaseDropdownLabel}
                                            options={databaseOptions}
                                            value={current?.Database}
                                            onChange={handleDatabaseChange}
                                            placeholder={databaseSchemaTerminology === 'schema' && usesDatabaseInsteadOfSchema ? t('selectSchema') : t('selectDatabase')}
                                            searchPlaceholder={databaseSchemaTerminology === 'schema' && usesDatabaseInsteadOfSchema ? t('searchSchema') : t('searchDatabase')}
                                            side="left" align="start"
                                            buttonProps={{
                                                "data-testid": "sidebar-database",
                                                "aria-label": databaseDropdownLabel,
                                            }}
                                        />
                                    </div>
                                )}
                                {supportsSchema && (
                                    <div className={cn("flex flex-col gap-sm w-full", {
                                        "opacity-0 pointer-events-none": !open || pathname.includes(InternalRoutes.RawExecute.path),
                                    })}>
                                        <h2 className="text-sm">{t('schema')}</h2>
                                        <SearchSelect
                                            label={t('schema')}
                                            options={schemaOptions}
                                            value={schema}
                                            onChange={handleSchemaChange}
                                            placeholder={t('selectSchema')}
                                            searchPlaceholder={t('searchSchema')}
                                            side="left" align="start"
                                            buttonProps={{
                                                "data-testid": "sidebar-schema",
                                            }}
                                        />
                                    </div>
                                )}
                            </div>

                            <SidebarMenu className="grow mt-8 gap-4">
                                {sidebarRoutes.map(route => (
                                    <SidebarMenuItem key={route.title}>
                                        <SidebarMenuButton asChild tooltip={route.title}>
                                            <Link
                                                to={route.path}
                                                className={cn("flex items-center gap-2", {
                                                    "font-bold": pathname === route.path,
                                                })}
                                            >
                                                {route.icon}
                                                {open && <span>{route.title}</span>}
                                            </Link>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                ))}

                                <SidebarSeparator className={cn("my-2", {
                                    "mx-0": !open,
                                })} />

                                {featureFlags.contactUsPage && InternalRoutes.ContactUs && (
                                    <SidebarMenuItem>
                                        <SidebarMenuButton asChild tooltip={t('contactUs')}>
                                            <Link
                                                to={InternalRoutes.ContactUs.path}
                                                className={cn("flex items-center gap-2", {
                                                    "font-bold": pathname === InternalRoutes.ContactUs.path,
                                                })}
                                            >
                                                <QuestionMarkCircleIcon className="w-4 h-4" />
                                                {open && <span>{t('contactUs')}</span>}
                                            </Link>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                )}
                                {featureFlags.settingsPage && InternalRoutes.Settings && (
                                    <SidebarMenuItem>
                                        <SidebarMenuButton asChild tooltip={t('settings')}>
                                            <Link
                                                to={InternalRoutes.Settings.path}
                                                className={cn("flex items-center gap-2", {
                                                    "font-bold": pathname === InternalRoutes.Settings.path,
                                                })}
                                            >
                                                <CogIcon className="w-4 h-4" />
                                                {open && <span>{t('settings')}</span>}
                                            </Link>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                )}
                                <div className="grow" />
                                {!isEmbedded && (
                                    <SidebarMenuItem className="flex justify-between items-center w-full">
                                        <SidebarMenuButton asChild tooltip={t('logOutProfile')}>
                                            <div className="flex items-center gap-sm text-nowrap w-fit cursor-pointer" onClick={handleLogoutProfile}>
                                                <ArrowLeftStartOnRectangleIcon className="w-4 h-4" />
                                                {open && <span>{t('logOutProfile')}</span>}
                                            </div>
                                        </SidebarMenuButton>
                                        <SidebarMenuButton asChild>
                                            <DropdownMenu>
                                                <DropdownMenuTrigger className={cn({
                                                    "hidden": !open,
                                                })}>
                                                    <Button
                                                        className="flex items-center justify-center p-1 rounded hover:bg-gray-100 dark:hover:bg-neutral-800 ml-2"
                                                        aria-label={t('moreLogoutOptions')}
                                                        variant="ghost"
                                                    >
                                                        <ChevronDownIcon className="w-4 h-4" />
                                                    </Button>
                                                </DropdownMenuTrigger>
                                                <DropdownMenuContent side="right" align="start">
                                                    <DropdownMenuItem onClick={handleLogoutAll}>
                                                        <ArrowLeftStartOnRectangleIcon className="w-4 h-4" />
                                                        <span className="ml-2">{t('logoutAllProfiles')}</span>
                                                    </DropdownMenuItem>
                                                </DropdownMenuContent>
                                            </DropdownMenu>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                )}
                            </SidebarMenu>
                        </SidebarGroup>
                    )}
                </SidebarContent>
                {newUIEnabled ? (
                    <div className="absolute right-3 bottom-3">
                        <TooltipProvider delayDuration={200}>
                            <Tooltip>
                                <TooltipTrigger asChild>
                                    <button className={cn(
                                        "rounded-full p-1 text-muted-foreground/60 hover:text-muted-foreground transition-colors",
                                        updateInfo?.UpdateInfo?.updateAvailable && "text-blue-500/70 hover:text-blue-400"
                                    )} data-testid="sidebar-version-info">
                                        <InformationCircleIcon className="w-4 h-4" />
                                    </button>
                                </TooltipTrigger>
                                <TooltipContent side="top" align="end">
                                    <p className="text-xs">{t('version')} {__APP_VERSION__}</p>
                                    {updateInfo?.UpdateInfo?.updateAvailable && (
                                        <a
                                            href={updateInfo.UpdateInfo.releaseURL}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-xs text-blue-500 hover:text-blue-400 transition-colors"
                                        >
                                            {t('updateAvailable', { version: updateInfo.UpdateInfo.latestVersion })}
                                        </a>
                                    )}
                                </TooltipContent>
                            </Tooltip>
                        </TooltipProvider>
                    </div>
                ) : (
                    <div className={cn("absolute right-4 bottom-4 text-xs text-muted-foreground", {
                        "hidden": !open,
                    })}>
                        {t('version')} {__APP_VERSION__}
                        {updateInfo?.UpdateInfo?.updateAvailable && (
                            <a
                                href={updateInfo.UpdateInfo.releaseURL}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="ml-2 text-blue-500 hover:text-blue-400 transition-colors"
                                title={t('updateAvailable', { version: updateInfo.UpdateInfo.latestVersion })}
                            >
                                &uarr; {updateInfo.UpdateInfo.latestVersion}
                            </a>
                        )}
                    </div>
                )}
            </SidebarComponent>
            <Sheet open={showLoginCard} onOpenChange={setShowLoginCard}>
                <SheetContent side="right" className="p-8">
                    <VisuallyHidden>
                        <SheetTitle>{t('databaseLogin')}</SheetTitle>
                    </VisuallyHidden>
                    <LoginForm advancedDirection="vertical" onLoginSuccess={handleLoginSuccess}/>
                </SheetContent>
            </Sheet>
            <Dialog open={showProfileSwitchDialog} onOpenChange={handleProfileSwitchDialogChange}>
                <DialogContent className="max-w-sm" onInteractOutside={(e) => e.preventDefault()} onEscapeKeyDown={(e) => e.preventDefault()}>
                    <DialogHeader>
                        <DialogTitle>{t('switchProfile')}</DialogTitle>
                        <DialogDescription>{t('switchProfileDescription')}</DialogDescription>
                    </DialogHeader>
                    <div className="flex flex-col gap-1 mt-2">
                        {profiles.filter(p => p.Id !== logoutProfileId).map(profile => {
                            const sourceTypeItem = findSourceTypeItem(sourceTypeItems, profile.Type);
                            return (
                                <button
                                    key={profile.Id}
                                    className="flex items-center gap-3 p-3 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors text-left w-full"
                                    onClick={() => handleDialogProfileSwitch(profile)}
                                >
                                    <DatabaseIconWithBadge
                                        icon={getProfileIcon(profile)}
                                        showCloudBadge={isAwsConnection(profile.Id)}
                                        size="sm"
                                    />
                                    <span className="text-sm font-medium truncate">{getProfileLabel(profile, sourceTypeItem)}</span>
                                </button>
                            );
                        })}
                        <button
                            className="flex items-center gap-3 p-3 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors text-left w-full text-green-500"
                            onClick={handleDialogAddProfile}
                        >
                            <PlusCircleIcon className="w-4 h-4 stroke-green-500" />
                            <span className="text-sm font-medium">{t('addAnotherProfile')}</span>
                        </button>
                    </div>
                </DialogContent>
            </Dialog>
        </nav>
    );
};
