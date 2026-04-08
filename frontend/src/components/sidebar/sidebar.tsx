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

import {
    Button,
    cn,
    CommandItem,
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
    useSidebar
} from "@clidey/ux";
import {SearchSelect} from "../ux";
import {
    DatabaseType,
    useGetDatabaseQuery,
    useGetSchemaQuery,
    useGetSslStatusQuery,
    useGetUpdateInfoQuery,
} from '@graphql';
import {useTranslation} from '@/hooks/use-translation';
import {VisuallyHidden} from "@radix-ui/react-visually-hidden";
import {FC, ReactElement, ReactNode, Suspense, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {useDispatch} from "react-redux";
import {Link, useLocation, useNavigate} from "react-router-dom";
import logoImage from "../../../public/images/logo.svg";
import {extensions, getAppName, isEEMode} from "../../config/features";
import {InternalRoutes} from "../../config/routes";
import {LoginForm} from "../../pages/auth/login";
import {AuthActions, LocalLoginProfile} from "../../store/auth";
import {DatabaseActions} from "../../store/database";
import {useAppSelector} from "../../store/hooks";
import {
    databaseSupportsDatabaseSwitching,
    databaseSupportsSchema,
    databaseSupportsScratchpad,
    databaseTypesThatUseDatabaseInsteadOfSchema
} from "../../utils/database-features";
import {isEEFeatureEnabled, loadEEComponent} from "../../utils/ee-loader";
import {isNoSQL} from "../../utils/functions";
import {isAwsHostname} from "../../utils/cloud-connection-prefill";
import {
    ArrowLeftStartOnRectangleIcon,
    ChevronDownIcon,
    CogIcon,
    CommandLineIcon,
    PlusCircleIcon,
    QuestionMarkCircleIcon,
    RectangleGroupIcon,
    SparklesIcon,
    TableCellsIcon
} from "../heroicons";
import {Icons} from "../icons";
import {DatabaseIconWithBadge, isAwsConnection} from "../aws";
import {useProfileSwitch} from "@/hooks/use-profile-switch";

function getProfileLabel(profile: LocalLoginProfile) {
    if (profile.Saved) return profile.Id;
    if (profile.Type === DatabaseType.Redis) return profile.Hostname;
    if (profile.Type === DatabaseType.Sqlite3) return profile.Database;
    return `${profile.Hostname} [${profile.Username}]`;
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

// Load EE sidebar nav at module level — handles install + platform mode layout
const EESidebarNav = isEEFeatureEnabled('platformMode')
    ? loadEEComponent(() => import('@ee/components/sidebar-nav').then(m => ({ default: m.EESidebarNav })), null)
    : null;

export const Sidebar: FC = () => {
    const { t } = useTranslation('components/sidebar');
    const schema = useAppSelector(state => state.database.schema);
    const databaseSchemaTerminology = useAppSelector(state => state.settings.databaseSchemaTerminology);
    const cloudProvidersEnabled = useAppSelector(state => state.settings.cloudProvidersEnabled);
    const isEmbedded = useAppSelector(state => state.auth.isEmbedded);
    const dispatch = useDispatch();
    const pathname = useLocation().pathname;
    const current = useAppSelector(state => state.auth.current);
    const profiles = useAppSelector(state => state.auth.profiles);
    const sslStatus = useAppSelector(state => state.auth.sslStatus);
    const {data: availableDatabases, loading: availableDatabasesLoading, refetch: getDatabases} = useGetDatabaseQuery({
        variables: {
            type: current?.Type as DatabaseType,
        },
        skip: current == null || !databaseSupportsDatabaseSwitching(current?.Type),
    });
    const { data: availableSchemas, loading: availableSchemasLoading, refetch: getSchemas } = useGetSchemaQuery({
        skip: current == null || !databaseSupportsSchema(current?.Type),
    });

    // Default schema selection: prefer Search Path from login config, fall back to first schema
    useEffect(() => {
        if (current == null || schema !== "" || !availableSchemas?.Schema?.length) return;
        const searchPath = current.Advanced?.find(a => a.Key === "Search Path")?.Value;
        const defaultSchema = (searchPath && availableSchemas.Schema.includes(searchPath))
            ? searchPath
            : availableSchemas.Schema[0] ?? "";
        dispatch(DatabaseActions.setSchema(defaultSchema));
    }, [current, schema, availableSchemas, dispatch]);
    const { data: updateInfo } = useGetUpdateInfoQuery();
    const { refetch: refetchSslStatus } = useGetSslStatusQuery({
        skip: current == null || sslStatus !== undefined,
        onCompleted(data) {
            if (data.SSLStatus) {
                dispatch(AuthActions.setSSLStatus(data.SSLStatus));
            }
        },
    });
    const navigate = useNavigate();
    const [showLoginCard, setShowLoginCard] = useState(false);
    const { toggleSidebar, open } = useSidebar();
    const isInitialMount = useRef(true);
    const { switchProfile } = useProfileSwitch({
        errorMessage: t('errorSigningIn'),
    });

    // Profile select logic - filter out AWS profiles when cloud providers disabled
    const profileOptions = useMemo(() => profiles
        .filter(profile => cloudProvidersEnabled || !isAwsHostname(profile.Hostname))
        .map(profile => ({
            value: profile.Id,
            label: getProfileLabel(profile),
            icon: (
                <DatabaseIconWithBadge
                    icon={getProfileIcon(profile)}
                    showCloudBadge={isAwsConnection(profile.Id)}
                    sslStatus={profile.Id === current?.Id
                        ? sslStatus
                        : (profile.SSLConfigured ? { IsEnabled: true, Mode: 'configured' } : undefined)}
                    size="sm"
                />
            ),
            profile,
        })), [profiles, current?.Id, sslStatus, cloudProvidersEnabled]);

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
        if (!availableDatabases?.Database) return [];
        return availableDatabases.Database.map(db => ({
            value: db,
            label: db,
        }));
    }, [availableDatabases?.Database]);

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
        return availableSchemas?.Schema?.map(s => ({
            value: s,
            label: s,
        })) ?? [];
    }, [availableSchemas?.Schema]);

    const handleSchemaChange = useCallback((value: string) => {
        if (value === "") {
            return;
        }
        if (pathname !== InternalRoutes.Graph.path && pathname !== InternalRoutes.Dashboard.StorageUnit.path) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
        dispatch(DatabaseActions.setSchema(value));
    }, [dispatch, navigate, pathname]);

    // Logout logic
    const handleLogout = useCallback(() => {
        navigate(InternalRoutes.Logout.path);
    }, [navigate]);

    // Add profile logic
    const handleAddProfile = useCallback(() => {
        // Small delay to allow dropdown to close before opening sheet
        setTimeout(() => {
            setShowLoginCard(true);
        }, 100);
    }, []);

    // Compute the label for the database dropdown based on the database type and user terminology preference
    const databaseDropdownLabel = useMemo(() => {
        // For databases where database=schema (MySQL, MariaDB, ClickHouse, MongoDB, Redis), allow user to choose terminology
        if (databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type)) {
            return databaseSchemaTerminology === 'schema' ? t('schema') : t('database');
        }
        // For all other databases, use "Database"
        return t('database');
    }, [current?.Type, databaseSchemaTerminology, t]);

    // Pre-render the dropdowns so EE component can place them without duplicating logic
    const dropdownsJSX: ReactNode = (
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
            {databaseSupportsDatabaseSwitching(current?.Type) && (
                <SearchSelect
                    label={databaseDropdownLabel}
                    options={databaseOptions}
                    value={current?.Database}
                    onChange={handleDatabaseChange}
                    placeholder={databaseSchemaTerminology === 'schema' && databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type) ? t('selectSchema') : t('selectDatabase')}
                    searchPlaceholder={databaseSchemaTerminology === 'schema' && databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type) ? t('searchSchema') : t('searchDatabase')}
                    side="left" align="start"
                    buttonProps={{ "data-testid": "sidebar-database" }}
                />
            )}
            {databaseSupportsSchema(current?.Type) && !pathname.includes(InternalRoutes.RawExecute.path) && (
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
    );

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
        if (databaseSupportsDatabaseSwitching(current.Type)) {
            getDatabases();
        }
        if (databaseSupportsSchema(current.Type)) {
            getSchemas();
        }
        refetchSslStatus().then(({ data }) => {
            if (data?.SSLStatus) {
                dispatch(AuthActions.setSSLStatus(data.SSLStatus));
            }
        });
    }, [current?.Id, current?.Database]);

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
                className="dark:group-data-[side=left]:border-r-neutral-800/40 z-[50] dark:[&>[data-sidebar=sidebar]]:bg-gradient-to-b dark:[&>[data-sidebar=sidebar]]:from-[#080e1e] dark:[&>[data-sidebar=sidebar]]:via-[#060b17] dark:[&>[data-sidebar=sidebar]]:to-[#050911]"
            >
                <SidebarHeader className={cn({
                    "ml-4": open,
                })}>
                    <div className="flex items-center gap-sm justify-between">
                        <div className={cn("flex items-center gap-sm mt-2", {
                            "hidden": !open,
                        })}>
                            {extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-6" />}
                            {open && <span className="text-lg font-bold" data-testid="app-name">{getAppName()}</span>}
                        </div>
                        <SidebarTrigger className="px-0" />
                    </div>
                </SidebarHeader>
                <SidebarContent className={cn("mt-8 mb-16 overflow-y-auto", {
                    "mx-4": open,
                })}>
                    <SidebarGroup className="grow">
                        <SidebarMenu className="gap-0">

                            {EESidebarNav ? (
                                <Suspense fallback={null}>
                                    <EESidebarNav open={open} pathname={pathname} dropdowns={dropdownsJSX} />
                                </Suspense>
                            ) : (
                                <>
                                    {/* CE CLASSIC LAYOUT */}
                                    {open && (
                                        <div className="flex flex-col gap-sm mb-4">
                                            {dropdownsJSX}
                                        </div>
                                    )}
                                    {current != null && !isNoSQL(current.Type) && (
                                        <NavItem icon={<SparklesIcon className="w-4 h-4" />} label={t('navAIChat')} path={InternalRoutes.Chat.path} pathname={pathname} open={open} tooltip={t('navAIChat')} />
                                    )}
                                    <NavItem icon={<TableCellsIcon className="w-4 h-4" />} label={t('navExplorer')} path={InternalRoutes.Dashboard.StorageUnit.path} pathname={pathname} open={open} tooltip={t('navExplorer')} />
                                    <NavItem icon={<RectangleGroupIcon className="w-4 h-4" />} label={t('navGraph')} path={InternalRoutes.Graph.path} pathname={pathname} open={open} tooltip={t('navGraph')} />
                                    {databaseSupportsScratchpad(current?.Type) && (
                                        <NavItem icon={<CommandLineIcon className="w-4 h-4" />} label={t('navScratchpad')} path={InternalRoutes.RawExecute.path} pathname={pathname} open={open} tooltip={t('navScratchpad')} />
                                    )}
                                </>
                            )}

                            <SidebarSeparator className={cn("my-4", { "mx-0": !open })} />

                            {/* Settings / Contact Us */}
                            {isEEFeatureEnabled('contactUsPage') && InternalRoutes.ContactUs && (
                                <NavItem icon={<QuestionMarkCircleIcon className="w-4 h-4" />} label={t('contactUs')} path={InternalRoutes.ContactUs.path} pathname={pathname} open={open} tooltip={t('contactUs')} />
                            )}
                            {isEEFeatureEnabled('settingsPage') && InternalRoutes.Settings && (
                                <NavItem icon={<CogIcon className="w-4 h-4" />} label={t('settings')} path={InternalRoutes.Settings.path} pathname={pathname} open={open} tooltip={t('settings')} />
                            )}

                            <div className="grow" />

                                {/* Logout */}
                                {!isEmbedded && (
                                    <SidebarMenuItem className="flex justify-between items-center w-full">
                                        {/* Logout Profile button */}
                                        <SidebarMenuButton asChild tooltip={t('logOutProfile')}>
                                            <div className="flex items-center gap-sm text-nowrap w-fit cursor-pointer" onClick={handleLogout}>
                                                <ArrowLeftStartOnRectangleIcon className="w-4 h-4" />
                                                {open && <span>{t('logOutProfile')}</span>}
                                            </div>
                                        </SidebarMenuButton>
                                        {/* Dropdown for additional logout options */}
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
                                                    <DropdownMenuItem
                                                        onClick={handleLogout}
                                                    >
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
                </SidebarContent>
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
                            ↑ {updateInfo.UpdateInfo.latestVersion}
                        </a>
                    )}
                </div>
            </SidebarComponent>
            <Sheet open={showLoginCard} onOpenChange={setShowLoginCard}>
                <SheetContent side="right" className="p-8">
                    <VisuallyHidden>
                        <SheetTitle>{t('databaseLogin')}</SheetTitle>
                    </VisuallyHidden>
                    <LoginForm advancedDirection="vertical" onLoginSuccess={() => setShowLoginCard(false)}/>
                </SheetContent>
            </Sheet>
        </nav>
    );
};
