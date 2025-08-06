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
    cn,
    CommandItem,
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
    SearchSelect,
    Sheet,
    SheetContent,
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
import { DatabaseType, useGetDatabaseQuery, useGetSchemaQuery, useGetVersionQuery, useLoginMutation, useLoginWithProfileMutation } from '@graphql';
import { Bars3Icon, ChevronDownIcon, PlusIcon } from "@heroicons/react/24/outline";
import classNames from "classnames";
import { FC, ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";
import { useLocation, useNavigate } from "react-router-dom";
import { InternalRoutes } from "../../config/routes";
import { LoginForm } from "../../pages/auth/login";
import { AuthActions, LocalLoginProfile } from "../../store/auth";
import { DatabaseActions } from "../../store/database";
import { useAppSelector } from "../../store/hooks";
import { databaseSupportsSchema, databaseSupportsScratchpad } from "../../utils/database-features";
import { isEEFeatureEnabled } from "../../utils/ee-loader";
import { getDatabaseStorageUnitLabel, isNoSQL } from "../../utils/functions";
import { ClassNames } from "../classes";
import { Icons } from "../icons";
import { Loading } from "../loading";
import { updateProfileLastAccessed } from "../profile-info-tooltip";

const logoImage = "/images/logo.png";

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
        skip: current == null || (current.Type !== DatabaseType.Redis && isNoSQL(current?.Type as DatabaseType)),
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

    const handleProfileChange = useCallback(async (value: string) => {
        const selectedProfile = profiles.find(profile => profile.Id === value);
        if (!selectedProfile) return;
        if (selectedProfile.Saved) {
            await loginWithProfile({
                variables: {
                    profile: {
                        Id: selectedProfile.Id,
                        Type: selectedProfile.Type as DatabaseType,
                        Database: current?.Database,
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
                        Database: selectedProfile.Database,
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
        if (!current?.Id) return;
        if (pathname !== InternalRoutes.Graph.path && pathname !== InternalRoutes.Dashboard.StorageUnit.path) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
        dispatch(AuthActions.setLoginProfileDatabase({ id: current?.Id, database: value }));
        handleProfileChange(current.Id);
    }, [current, dispatch, handleProfileChange, navigate, pathname]);

    // Schema select logic
    const schemaOptions = useMemo(() => {
        return availableSchemas?.Schema?.map(s => ({
            value: s,
            label: s,
        })) ?? [];
    }, [availableSchemas?.Schema]);

    const handleSchemaChange = useCallback((value: string) => {
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
                icon: Icons.Tables,
                path: InternalRoutes.Dashboard.StorageUnit.path,
            },
            {
                title: "Graph",
                icon: Icons.GraphLayout,
                path: InternalRoutes.Graph.path,
            },
        ];
        if (!isNoSQL(current.Type)) {
            routes.unshift({
                title: "Chat",
                icon: Icons.Sparkles,
                path: InternalRoutes.Chat.path,
            });
        }
        if (databaseSupportsScratchpad(current.Type)) {
            routes.push({
                title: "Scratchpad",
                icon: Icons.Console,
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

    return (
        <div>
            <SidebarComponent variant="sidebar" collapsible="icon" className="dark:group-data-[side=left]:border-r-neutral-800 z-[50]">
                <SidebarHeader className={cn({
                    "ml-4": open,
                })}>
                    <div className="flex items-center gap-2 justify-between">
                        <div className={cn("flex items-center gap-2", {
                            "hidden": !open,
                        })}>
                            <img src={logoImage} alt="clidey logo" className="w-auto h-4" />
                            {open && <span className={classNames(ClassNames.BrandText, "text-lg")}>WhoDB</span>}
                        </div>
                        <SidebarTrigger onClick={toggleSidebar} className="px-0">
                            <Bars3Icon />
                        </SidebarTrigger>
                    </div>
                </SidebarHeader>
                <SidebarContent className={cn("mt-8 mb-16", {
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
                                <div className="flex flex-col gap-2 w-full">
                                    <h2 className={classNames(ClassNames.Text, "text-sm", !open &&  "hidden")}>Profile</h2>
                                    <SearchSelect
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
                                                <span className="flex items-center gap-2 text-green-500">
                                                    <PlusIcon className="w-4 h-4 stroke-green-500" />
                                                    Add another profile
                                                </span>
                                            </CommandItem>
                                        }
                                        side="left" align="start"
                                    />
                                </div>
                                {/* Database Select */}
                                <div className={cn("flex flex-col gap-2 w-full", {
                                    "opacity-0 pointer-events-none": !open,
                                })}>
                                    <h2 className={classNames(ClassNames.Text, "text-sm")}>Database</h2>
                                    <SearchSelect
                                        options={databaseOptions}
                                        value={current?.Database}
                                        onChange={handleDatabaseChange}
                                        placeholder="Select database"
                                        searchPlaceholder="Search database..."
                                        side="left" align="start"
                                    />
                                </div>
                                <div className={cn("flex flex-col gap-2 w-full", {
                                    "opacity-0 pointer-events-none": !open,
                                })}>
                                    <h2 className={cn(ClassNames.Text, "text-sm")}>Schema</h2>
                                    <SearchSelect
                                        options={schemaOptions}
                                        value={schema}
                                        onChange={handleSchemaChange}
                                        placeholder="Select schema"
                                        searchPlaceholder="Search schema..."
                                        side="left" align="start"
                                    />
                                </div>
                            </div>
                            {/* Main navigation */}
                            <SidebarMenu className="grow mt-16 gap-4">
                                {sidebarRoutes.map(route => (
                                    <SidebarMenuItem key={route.title}>
                                        <SidebarMenuButton asChild>
                                            <a
                                                href={route.path}
                                                className={classNames("flex items-center gap-2", {
                                                    "font-bold": pathname === route.path,
                                                })}
                                            >
                                                {route.icon}
                                                {open && <span>{route.title}</span>}
                                            </a>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                ))}

                                <SidebarSeparator />

                                {isEEFeatureEnabled('contactUsPage') && InternalRoutes.ContactUs && (
                                    <SidebarMenuItem>
                                        <SidebarMenuButton asChild>
                                            <a href={InternalRoutes.ContactUs.path} className="flex items-center gap-2">
                                                {Icons.QuestionMark}
                                                {open && <span>Contact Us</span>}
                                            </a>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                )}
                                {isEEFeatureEnabled('settingsPage') && InternalRoutes.Settings && (
                                    <SidebarMenuItem>
                                        <SidebarMenuButton asChild>
                                            <a href={InternalRoutes.Settings.path} className="flex items-center gap-2">
                                                {Icons.Settings}
                                                {open && <span>Settings</span>}
                                            </a>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                )}
                                <div className="grow" />
                                    <SidebarMenuItem>
                                    <SidebarMenuButton asChild>
                                        <div className="relative flex items-center gap-2">
                                            <div className="flex items-center gap-2" onClick={handleLogout}>
                                                {Icons.Logout}
                                                {open && <span>Logout Profile</span>}
                                            </div>
                                            {/* Dropdown for additional logout options */}
                                            <div className="ml-2">
                                                <DropdownMenu>
                                                    <DropdownMenuTrigger asChild>
                                                        <button
                                                            className="flex items-center justify-center p-1 rounded hover:bg-gray-100 dark:hover:bg-neutral-800"
                                                            aria-label="More logout options"
                                                            type="button"
                                                        >
                                                            <ChevronDownIcon className="w-4 h-4" />
                                                        </button>
                                                    </DropdownMenuTrigger>
                                                    <DropdownMenuContent side="right" align="start">
                                                        <DropdownMenuItem
                                                            onClick={handleLogout}
                                                        >
                                                            {Icons.Logout}
                                                            <span className="ml-2">Logout All Profiles</span>
                                                        </DropdownMenuItem>
                                                    </DropdownMenuContent>
                                                </DropdownMenu>
                                            </div>
                                        </div>
                                    </SidebarMenuButton>
                                </SidebarMenuItem>
                            </SidebarMenu>
                            <div className={classNames(ClassNames.Text, "absolute right-8 bottom-8 text-sm text-gray-300 hover:text-gray-600 self-end dark:hover:text-neutral-300 transition-all")}>
                                {version?.Version}
                            </div>
                        </SidebarGroup>
                    )}
                </SidebarContent>
            </SidebarComponent>
            <Sheet open={showLoginCard} onOpenChange={setShowLoginCard}>
                <SheetContent side="right" className="p-8">
                    <LoginForm advancedDirection="vertical" />
                </SheetContent>
            </Sheet>
        </div>
    );
};