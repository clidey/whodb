/*
 * Copyright 2025 Clidey, Inc.
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
    SearchSelect,
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
import {
    DatabaseType,
    useGetDatabaseQuery,
    useGetSchemaQuery,
    useGetVersionQuery,
    useLoginMutation,
    useLoginWithProfileMutation
} from '@graphql';
import { VisuallyHidden } from "@radix-ui/react-visually-hidden";
import classNames from "classnames";
import { FC, ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";
import { Link, useLocation, useNavigate } from "react-router-dom";
import logoImage from "../../../public/images/logo.png";
import { extensions } from "../../config/features";
import { InternalRoutes } from "../../config/routes";
import { LoginForm } from "../../pages/auth/login";
import { AuthActions, LocalLoginProfile } from "../../store/auth";
import { DatabaseActions } from "../../store/database";
import { useAppSelector } from "../../store/hooks";
import { databaseSupportsDatabaseSwitching, databaseSupportsSchema, databaseSupportsScratchpad, databaseTypesThatUseDatabaseInsteadOfSchema } from "../../utils/database-features";
import { isEEFeatureEnabled } from "../../utils/ee-loader";
import { getDatabaseStorageUnitLabel, isNoSQL } from "../../utils/functions";
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
import { Icons } from "../icons";
import { Loading } from "../loading";
import { updateProfileLastAccessed } from "../profile-info-tooltip";

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
    const schema = useAppSelector(state => state.database.schema);
    const dispatch = useDispatch();
    const pathname = useLocation().pathname;
    const current = useAppSelector(state => state.auth.current);
    const profiles = useAppSelector(state => state.auth.profiles);
    const { data: availableDatabases, loading: availableDatabasesLoading } = useGetDatabaseQuery({
        variables: {
            type: current?.Type as DatabaseType,
        },
        skip: current == null || databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type),
    });
    const { data: availableSchemas, loading: availableSchemasLoading, refetch: getSchemas } = useGetSchemaQuery({
        onCompleted(data) {
            if (current == null) return;
            if (schema === "") {
                if (([DatabaseType.MySql, DatabaseType.MariaDb].includes(current.Type as DatabaseType)) && data.Schema.includes(current.Database)) {
                    dispatch(DatabaseActions.setSchema(current.Database));
                    return;
                }
                dispatch(DatabaseActions.setSchema(data.Schema[0] ?? ""));
            }
        },
        skip: current == null || !databaseSupportsSchema(current?.Type),
    });
    const { data: version } = useGetVersionQuery();
    const [login] = useLoginMutation();
    const [loginWithProfile] = useLoginWithProfileMutation();
    const navigate = useNavigate();
    const [showLoginCard, setShowLoginCard] = useState(false);
    const { toggleSidebar, open } = useSidebar();

    // Profile select logic
    const profileOptions = useMemo(() => profiles.map(profile => ({
        value: profile.Id,
        label: getProfileLabel(profile),
        icon: getProfileIcon(profile),
        profile,
    })), [profiles]);

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
                        Database: database ?? current?.Database,
                    },
                },
                onCompleted(status) {
                    if (status.LoginWithProfile.Status) {
                        updateProfileLastAccessed(selectedProfile.Id);
                        dispatch(DatabaseActions.setSchema(""));
                        dispatch(AuthActions.switch({ id: selectedProfile.Id }));
                        navigate(InternalRoutes.Dashboard.StorageUnit.path);
                        if (databaseSupportsSchema(current?.Type)) getSchemas();
                    }
                },
                onError(error) {
                    toast.error(`Error signing you in: ${error.message}`);
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
                        getSchemas();
                    }
                },
                onError(error) {
                    toast.error(`Error signing you in: ${error.message}`);
                },
            });
        }
    }, [profiles, login, loginWithProfile, dispatch, navigate, getSchemas, current?.Database, current?.Type]);

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
                title: "Graph",
                icon: <RectangleGroupIcon className="w-4 h-4" />,
                path: InternalRoutes.Graph.path,
            },
        ];
        if (!isNoSQL(current.Type)) {
            routes.unshift({
                title: "Chat",
                icon: <SparklesIcon className="w-4 h-4" />,
                path: InternalRoutes.Chat.path,
            });
        }
        if (databaseSupportsScratchpad(current.Type)) {
            routes.push({
                title: "Scratchpad",
                icon: <CommandLineIcon className="w-4 h-4" />,
                path: InternalRoutes.RawExecute.path,
            });
        }
        return routes;
    }, [current]);

    // Logout logic
    const handleLogout = useCallback(() => {
        navigate(InternalRoutes.Logout.path);
    }, [navigate]);

    // Add profile logic
    const handleAddProfile = useCallback(() => {
        setShowLoginCard(true);
    }, [navigate]);

    const loading = availableDatabasesLoading || availableSchemasLoading;

    useEffect(() => {
        if (pathname.includes(InternalRoutes.Dashboard.ExploreStorageUnit.path) && open) {
            toggleSidebar();
        }
    }, []);

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
        <div className="dark">
            <SidebarComponent variant="sidebar" collapsible="icon" className="dark:group-data-[side=left]:border-r-neutral-800 z-[50]">
                <SidebarHeader className={cn({
                    "ml-4": open,
                })}>
                    <div className="flex items-center gap-sm justify-between">
                        <div className={cn("flex items-center gap-sm mt-2", {
                            "hidden": !open,
                        })}>
                            {extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-4" />}
                            {open && <span className="text-lg text-brand-foreground">{extensions.AppName ?? "WhoDB"}</span>}
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
                            <div className="flex flex-col gap-4">
                                {/* Profile Select */}
                                <div className="flex flex-col gap-sm w-full">
                                    <h2 className={cn("text-sm", !open &&  "hidden")}>Profile</h2>
                                    <SearchSelect
                                        label="Profile"
                                        options={profileOptions}
                                        value={currentProfileOption?.value}
                                        onChange={handleProfileChange}
                                        placeholder="Select profile"
                                        searchPlaceholder="Search profile..."
                                        onlyIcon={!open}
                                        extraOptions={
                                            <CommandItem
                                                key="__add__"
                                                value="__add__"
                                                onSelect={handleAddProfile}
                                            >
                                                <span className="flex items-center gap-sm text-green-500">
                                                    <PlusCircleIcon className="w-4 h-4 stroke-green-500" />
                                                    Add another profile
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
                                {/* Database Select */}
                                <div className={cn("flex flex-col gap-sm w-full", {
                                    "opacity-0 pointer-events-none": !open,
                                    "hidden": !databaseSupportsDatabaseSwitching(current?.Type),
                                })}>
                                    <h2 className="text-sm">Database</h2>
                                    <SearchSelect
                                        label="Database"
                                        options={databaseOptions}
                                        value={current?.Database}
                                        onChange={handleDatabaseChange}
                                        placeholder="Select database"
                                        searchPlaceholder="Search database..."
                                        side="left" align="start"
                                        buttonProps={{
                                            "data-testid": "sidebar-database",
                                        }}
                                    />
                                </div>
                                <div className={cn("flex flex-col gap-sm w-full", {
                                    "opacity-0 pointer-events-none": !open || pathname.includes(InternalRoutes.RawExecute.path),
                                    "hidden": !databaseSupportsSchema(current?.Type),
                                })}>
                                    <h2 className="text-sm">Schema</h2>
                                    <SearchSelect
                                        label="Schema"
                                        options={schemaOptions}
                                        value={schema}
                                        onChange={handleSchemaChange}
                                        placeholder="Select schema"
                                        searchPlaceholder="Search schema..."
                                        side="left" align="start"
                                        buttonProps={{
                                            "data-testid": "sidebar-schema",
                                        }}
                                    />
                                </div>
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
                                            <Link to={InternalRoutes.ContactUs.path} className="flex items-center gap-2">
                                                <QuestionMarkCircleIcon className="w-4 h-4" />
                                                {open && <span>Contact Us</span>}
                                            </Link>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                )}
                                {isEEFeatureEnabled('settingsPage') && InternalRoutes.Settings && (
                                    <SidebarMenuItem>
                                        <SidebarMenuButton asChild>
                                            <Link to={InternalRoutes.Settings.path} className="flex items-center gap-2">
                                                <CogIcon className="w-4 h-4" />
                                                {open && <span>Settings</span>}
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
                                            {open && <span>Logout Profile</span>}
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
                                                    aria-label="More logout options"
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
                                                    <span className="ml-2">Logout All Profiles</span>
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
                    {version?.Version}
                </div>
            </SidebarComponent>
            <Sheet open={showLoginCard} onOpenChange={setShowLoginCard}>
                <SheetContent side="right" className="p-8">
                    <VisuallyHidden>
                        <SheetTitle>Database Login</SheetTitle>
                    </VisuallyHidden>
                    <LoginForm advancedDirection="vertical" />
                </SheetContent>
            </Sheet>
        </div>
    );
};
