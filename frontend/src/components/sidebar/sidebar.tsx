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
    toast,
    useSidebar
} from "@clidey/ux";
import {SearchSelect} from "../ux";
import {
    DatabaseType,
    useGetDatabaseQuery,
    useGetSchemaQuery,
    useGetSslStatusQuery,
    useGetVersionQuery,
    useLoginMutation,
    useLoginWithProfileMutation
} from '@graphql';
import {useTranslation} from '@/hooks/use-translation';
import {VisuallyHidden} from "@radix-ui/react-visually-hidden";
import classNames from "classnames";
import {FC, ReactElement, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {useDispatch} from "react-redux";
import {Link, useLocation, useNavigate} from "react-router-dom";
import logoImage from "../../../public/images/logo.svg";
import {extensions} from "../../config/features";
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
import {isEEFeatureEnabled} from "../../utils/ee-loader";
import {getDatabaseStorageUnitLabel, isNoSQL} from "../../utils/functions";
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
import {Loading} from "../loading";
import {updateProfileLastAccessed} from "../profile-info-tooltip";
import {DatabaseIconWithBadge, isAwsConnection} from "../aws";

function getProfileLabel(profile: LocalLoginProfile) {
    if (profile.Saved) return profile.Id;
    if (profile.Type === DatabaseType.Redis) return profile.Hostname;
    if (profile.Type === DatabaseType.Sqlite3) return profile.Database;
    return `${profile.Hostname} [${profile.Username}]`;
}

function getProfileIcon(profile: LocalLoginProfile) {
    return (Icons.Logos as Record<string, ReactElement>)[profile.Type];
}

export const Sidebar: FC = () => {
    const { t } = useTranslation('components/sidebar');
    const schema = useAppSelector(state => state.database.schema);
    const databaseSchemaTerminology = useAppSelector(state => state.settings.databaseSchemaTerminology);
    const cloudProvidersEnabled = useAppSelector(state => state.settings.cloudProvidersEnabled);
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
        onCompleted(data) {
            if (current == null) return;
            if (schema === "") {
                dispatch(DatabaseActions.setSchema(data.Schema[0] ?? ""));
            }
        },
        skip: current == null || !databaseSupportsSchema(current?.Type),
    });
    const { data: version } = useGetVersionQuery();
    const { refetch: refetchSslStatus } = useGetSslStatusQuery({
        skip: current == null || sslStatus !== undefined,
        onCompleted(data) {
            if (data.SSLStatus) {
                dispatch(AuthActions.setSSLStatus(data.SSLStatus));
            }
        },
    });
    const [login] = useLoginMutation();
    const [loginWithProfile] = useLoginWithProfileMutation();
    const navigate = useNavigate();
    const [showLoginCard, setShowLoginCard] = useState(false);
    const { toggleSidebar, open } = useSidebar();
    const isInitialMount = useRef(true);

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
        dispatch(DatabaseActions.setSchema(""));
        if (selectedProfile.Saved) {
            await loginWithProfile({
                variables: {
                    profile: {
                        Id: selectedProfile.Id,
                        Type: selectedProfile.Type as DatabaseType,
                        Database: database ?? selectedProfile.Database,
                    },
                },
                onCompleted(status) {
                    if (status.LoginWithProfile.Status) {
                        updateProfileLastAccessed(selectedProfile.Id);
                        dispatch(DatabaseActions.setSchema(""));
                        dispatch(AuthActions.switch({ id: selectedProfile.Id }));
                        navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    }
                },
                onError(error) {
                    toast.error(`${t('errorSigningIn')} ${error.message}`);
                },
            });
        } else {
            await login({
                variables: {
                    credentials: {
                        Type: selectedProfile.Type,
                        Database: database ?? selectedProfile.Database,
                        Hostname: selectedProfile.Hostname,
                        Password: selectedProfile.Password,
                        Username: selectedProfile.Username,
                        Advanced: selectedProfile.Advanced,
                    },
                },
                onCompleted(status) {
                    if (status.Login.Status) {
                        updateProfileLastAccessed(selectedProfile.Id);
                        dispatch(DatabaseActions.setSchema(""));
                        dispatch(AuthActions.switch({ id: selectedProfile.Id }));
                        navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    }
                },
                onError(error) {
                    toast.error(`${t('errorSigningIn')} ${error.message}`);
                },
            });
        }
    }, [profiles, login, loginWithProfile, dispatch, navigate, t]);

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
        dispatch(AuthActions.setLoginProfileDatabase({ id: current?.Id, database: value }));
        handleProfileChange(current.Id, value);
    }, [current, dispatch, handleProfileChange, navigate, pathname]);

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

    // Sidebar routes
    const sidebarRoutes = useMemo(() => {
        if (!current) return [];
        const routes = [
            {
                title: getDatabaseStorageUnitLabel(current.Type),
                icon: <TableCellsIcon className="w-4 h-4" />,
                path: InternalRoutes.Dashboard.StorageUnit.path,
            },
            {
                title: t('graph'),
                icon: <RectangleGroupIcon className="w-4 h-4" />,
                path: InternalRoutes.Graph.path,
            },
        ];
        if (!isNoSQL(current.Type)) {
            routes.unshift({
                title: t('chat'),
                icon: <SparklesIcon className="w-4 h-4" />,
                path: InternalRoutes.Chat.path,
            });
        }
        if (databaseSupportsScratchpad(current.Type)) {
            routes.push({
                title: t('scratchpad'),
                icon: <CommandLineIcon className="w-4 h-4" />,
                path: InternalRoutes.RawExecute.path,
            });
        }
        return routes;
    }, [current, t]);

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

    const loading = availableDatabasesLoading || availableSchemasLoading;

    // Compute the label for the database dropdown based on the database type and user terminology preference
    const databaseDropdownLabel = useMemo(() => {
        // For databases where database=schema (MySQL, MariaDB, ClickHouse, MongoDB, Redis), allow user to choose terminology
        if (databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type)) {
            return databaseSchemaTerminology === 'schema' ? t('schema') : t('database');
        }
        // For all other databases, use "Database"
        return t('database');
    }, [current?.Type, databaseSchemaTerminology, t]);

    useEffect(() => {
        if (pathname.includes(InternalRoutes.Dashboard.ExploreStorageUnit.path) && open) {
            toggleSidebar();
        }
    }, []);

    // Refetch databases, schemas, and SSL status when the current profile changes
    // This ensures queries use the correct auth context after profile switch
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
    }, [current?.Id]);

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
            <SidebarComponent variant="sidebar" collapsible="icon" className="dark:group-data-[side=left]:border-r-neutral-800 z-[50]">
                <SidebarHeader className={cn({
                    "ml-4": open,
                })}>
                    <div className="flex items-center gap-sm justify-between">
                        <div className={cn("flex items-center gap-sm mt-2", {
                            "hidden": !open,
                        })}>
                            {extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8" />}
                            {open && <span className="text-3xl font-bold">{extensions.AppName ?? "WhoDB"}</span>}
                        </div>
                        <SidebarTrigger className="px-0" />
                    </div>
                </SidebarHeader>
                <SidebarContent className={cn("mt-8 mb-16 overflow-y-auto", {
                    "mx-4": open,
                })}>
                    {loading ? (
                        <div className="flex justify-center items-center h-full">
                            <Loading />
                        </div>
                    ) : (
                        <SidebarGroup className="grow">
                            <div className="flex flex-col gap-lg">
                                {/* Profile Select */}
                                <div className="flex flex-col gap-sm w-full">
                                    <h2 className={cn("text-sm", !open &&  "hidden")}>{t('profile')}</h2>
                                    <SearchSelect
                                        label={t('profile')}
                                        options={profileOptions}
                                        value={currentProfileOption?.value}
                                        onChange={handleProfileChange}
                                        placeholder={t('selectProfile')}
                                        searchPlaceholder={t('searchProfile')}
                                        onlyIcon={!open}
                                        extraOptions={
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
                                        }
                                        side="left" align="start"
                                        buttonProps={{
                                            "data-testid": "sidebar-profile",
                                            "data-collapsed": !open,
                                        }}
                                    />
                                </div>
                                {databaseSupportsDatabaseSwitching(current?.Type) && (
                                    <div className={cn("flex flex-col gap-sm w-full", {
                                        "opacity-0 pointer-events-none": !open,
                                    })}>
                                        <h2 className="text-sm">{databaseDropdownLabel}</h2>
                                        <SearchSelect
                                            label={databaseDropdownLabel}
                                            options={databaseOptions}
                                            value={current?.Database}
                                            onChange={handleDatabaseChange}
                                            placeholder={databaseSchemaTerminology === 'schema' && databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type) ? t('selectSchema') : t('selectDatabase')}
                                            searchPlaceholder={databaseSchemaTerminology === 'schema' && databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type) ? t('searchSchema') : t('searchDatabase')}
                                            side="left" align="start"
                                            buttonProps={{
                                                "data-testid": "sidebar-database",
                                            }}
                                        />
                                    </div>
                                )}
                                {databaseSupportsSchema(current?.Type) && (
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
                            
                            {/* Main navigation */}
                            <SidebarMenu className="grow mt-8 gap-4">
                                {sidebarRoutes.map(route => (
                                    <SidebarMenuItem key={route.title}>
                                        <SidebarMenuButton asChild>
                                            <Link
                                                to={route.path}
                                                className={classNames("flex items-center gap-2", {
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

                                {isEEFeatureEnabled('contactUsPage') && InternalRoutes.ContactUs && (
                                    <SidebarMenuItem>
                                        <SidebarMenuButton asChild>
                                            <Link
                                                to={InternalRoutes.ContactUs.path}
                                                className={classNames("flex items-center gap-2", {
                                                    "font-bold": pathname === InternalRoutes.ContactUs.path,
                                                })}
                                            >
                                                <QuestionMarkCircleIcon className="w-4 h-4" />
                                                {open && <span>{t('contactUs')}</span>}
                                            </Link>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                )}
                                {isEEFeatureEnabled('settingsPage') && InternalRoutes.Settings && (
                                    <SidebarMenuItem>
                                        <SidebarMenuButton asChild>
                                            <Link
                                                to={InternalRoutes.Settings.path}
                                                className={classNames("flex items-center gap-2", {
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
                                    <SidebarMenuItem className="flex justify-between items-center w-full">
                                    {/* Logout Profile button */}
                                    <SidebarMenuButton asChild>
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
                            </SidebarMenu>
                        </SidebarGroup>
                    )}
                </SidebarContent>
                <div className={cn("absolute right-4 bottom-4 text-xs text-muted-foreground", {
                    "hidden": !open,
                })}>
                    {t('version')} {version?.Version}
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
