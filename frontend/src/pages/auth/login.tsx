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

import {Badge, Button, cn, Input, Label, ModeToggle, SearchSelect, Separator, toast} from '@clidey/ux';
import {
    DatabaseType,
    LoginCredentials,
    useGetDatabaseLazyQuery,
    useGetProfilesQuery,
    useLoginMutation,
    useLoginWithProfileMutation
} from '@graphql';
import classNames from "classnames";
import entries from "lodash/entries";
import {FC, ReactElement, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {useNavigate, useSearchParams} from "react-router-dom";
import {Icons} from "../../components/icons";
import {Loading} from "../../components/loading";
import {Container} from "../../components/page";
import {updateProfileLastAccessed} from "../../components/profile-info-tooltip";
import {baseDatabaseTypes, getDatabaseTypeDropdownItems, IDatabaseDropdownItem} from "../../config/database-types";
import {extensions, sources} from '../../config/features';
import {InternalRoutes} from "../../config/routes";
import {AuthActions} from "../../store/auth";
import {DatabaseActions} from "../../store/database";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {AdjustmentsHorizontalIcon, CheckCircleIcon, CircleStackIcon} from '../../components/heroicons';
import logoImage from "../../../public/images/logo.png";
import { v4 } from 'uuid';
import { isDesktopApp } from '../../utils/external-links';
import { useDesktopFile } from '../../hooks/useDesktop';

/**
 * Generate a consistent ID for desktop credentials based on connection details.
 * This ensures the same credentials always produce the same ID, preventing duplicate keyring entries.
 * For browser environments, returns undefined to rely on cookie-based auth.
 */
function generateCredentialId(type: string, hostname: string, username: string, database: string): string | undefined {
    // browser environment just uses a random ID
    if (!isDesktopApp()) {
        return v4();
    }

    // desktop environment uses a deterministic ID based on connection details
    const parts = [
        'whodb',
        type || 'unknown',
        hostname || 'localhost',
        username || 'default',
        database || 'default'
    ];

    const combined = parts.join('::');
    try {
        const encoded = btoa(combined).replace(/[+/=]/g, '');
        return encoded.substring(0, 16).toLowerCase();
    } catch {
        return v4();
    }
}


// Embeddable LoginForm component
export interface LoginFormProps {
    // Optionally override navigation after login (e.g. for sidebar)
    onLoginSuccess?: () => void;
    // Optionally hide logo/title (for sidebar)
    hideHeader?: boolean;
    // Optionally compact mode (for sidebar)
    compact?: boolean;
    // Optionally override container className
    className?: string;
    advancedDirection?: "horizontal" | "vertical";
}

export const LoginForm: FC<LoginFormProps> = ({
                                                  onLoginSuccess,
                                                  hideHeader = false,
                                                  className = "",
                                                  advancedDirection = "horizontal",
                                              }) => {
    const dispatch = useAppDispatch();
    const navigate = useNavigate();
    const currentProfile = useAppSelector(state => state.auth.current);
    const shouldUpdateLastAccessed = useRef(false);

    const [login, { loading: loginLoading }] = useLoginMutation();
    const [loginWithProfile, { loading: loginWithProfileLoading }] = useLoginWithProfileMutation();
    const [getDatabases, { loading: databasesLoading, data: foundDatabases }] = useGetDatabaseLazyQuery();
    const { loading: profilesLoading, data: profiles } = useGetProfilesQuery();
    const [searchParams, ] = useSearchParams();

    const [databaseTypeItems, setDatabaseTypeItems] = useState<IDatabaseDropdownItem[]>(baseDatabaseTypes);
    const [databaseType, setDatabaseType] = useState<IDatabaseDropdownItem>(baseDatabaseTypes[0]);
    const [hostName, setHostName] = useState("");
    const [database, setDatabase] = useState("");
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string>();
    const [advancedForm, setAdvancedForm] = useState<Record<string, string>>(
        databaseType.extra ?? {}
    );
    const [showAdvanced, setShowAdvanced] = useState(false);
    const [selectedAvailableProfile, setSelectedAvailableProfile] = useState<string>();

    const { isDesktop, selectSQLiteDatabase } = useDesktopFile();

    const loading = useMemo(() => {
        return loginLoading || loginWithProfileLoading;
    }, [loginLoading, loginWithProfileLoading]);

    const handleSubmit = useCallback(() => {
        if (([DatabaseType.MySql, DatabaseType.Postgres].includes(databaseType.id as DatabaseType) && (hostName.length === 0 || database.length === 0 || username.length === 0))
            || (databaseType.id === DatabaseType.Sqlite3 && database.length === 0)
            || ((databaseType.id === DatabaseType.MongoDb || databaseType.id === DatabaseType.Redis) && (hostName.length === 0))) {
            return setError("All fields are required");
        }
        setError(undefined);

        // Generate ID only for desktop apps, using consistent ID for same credentials
        const credentialId = generateCredentialId(databaseType.id, hostName, username, database);

        const credentials: LoginCredentials = {
            Id: credentialId,
            Type: databaseType.id,
            Hostname: hostName,
            Database: database,
            Username: username,
            Password: password,
            Advanced: entries(advancedForm).map(([Key, Value]) => ({ Key, Value })),
        };

        login({
            variables: {
                credentials,
            },
            onCompleted(data) {
                if (data.Login.Status) {
                    const profileData = { ...credentials };
                    shouldUpdateLastAccessed.current = true;
                    dispatch(AuthActions.login(profileData));
                    if (onLoginSuccess) {
                        onLoginSuccess();
                    } else {
                        navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    }
                    return toast.success("Login successful");
                }
                return toast.error("Login failed");
            },
            onError(error) {
                return toast.error(`Login failed: ${error.message}`);
            }
        });
    }, [databaseType.id, hostName, database, username, password, advancedForm, login, dispatch, navigate, onLoginSuccess]);

    const handleLoginWithProfileSubmit = useCallback((overrideProfileId?: string) => {
        const profileId = overrideProfileId ?? selectedAvailableProfile;
        if (profileId == null) {
            return setError("Select a profile");
        }
        setError(undefined);

        const profile = profiles?.Profiles.find(p => p.Id === profileId);

        loginWithProfile({
            variables: {
                profile: {
                    Id:  profileId,
                    Type: profile?.Type as DatabaseType,
                },
            },
            onCompleted(data) {
                if (data.LoginWithProfile.Status) {
                    updateProfileLastAccessed(profileId);
                    dispatch(AuthActions.login({
                        Type: profile?.Type as DatabaseType,
                        Id: profileId,
                        Database: profile?.Database ?? "",
                        Hostname: "",
                        Password: "",
                        Username: "",
                        Saved: true,
                        IsEnvironmentDefined: profile?.IsEnvironmentDefined ?? false,
                    }));
                    if (onLoginSuccess) {
                        onLoginSuccess();
                    } else {
                        navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    }
                    return toast.success("Login successfully");
                }
                return toast.error("Login failed");
            },
            onError(error) {
                return toast.error(`Login failed: ${error.message}`);
            }
        });
    }, [dispatch, loginWithProfile, navigate, profiles?.Profiles, selectedAvailableProfile, onLoginSuccess]);

    const handleDatabaseTypeChange = useCallback((item: IDatabaseDropdownItem) => {
        if (item.id === DatabaseType.Sqlite3) {
            getDatabases({
                variables: {
                    type: DatabaseType.Sqlite3,
                },
            });
        }
        setHostName("");
        setUsername("");
        setPassword("");
        setDatabase("");
        setDatabaseType(item);
        setAdvancedForm(item.extra ?? {});
    }, [getDatabases]);

    const handleAdvancedToggle = useCallback(() => {
        setShowAdvanced(a => !a);
    }, []);

    const handleAdvancedForm = useCallback((key: string, value: string) => {
        setAdvancedForm(form => {
            const newForm = {...form};
            newForm[key] = value;
            return newForm;
        });
    }, []);

    const handleAvailableProfileChange = useCallback((itemId: string) => {
        setSelectedAvailableProfile(itemId);
    }, []);

    const handleBrowseSQLiteFile = useCallback(async () => {
        try {
            const filePath = await selectSQLiteDatabase();
            if (filePath) {
                setDatabase(filePath);
            }
        } catch (error) {
            console.error('Failed to select SQLite database:', error);
            toast.error('Failed to select database file');
        }
    }, [selectSQLiteDatabase]);

    useEffect(() => {
        dispatch(DatabaseActions.setSchema(""));
    }, [dispatch]);

    // Load EE database types if available
    useEffect(() => {
        getDatabaseTypeDropdownItems().then(items => {
            setDatabaseTypeItems(items);
        });
    }, []);

    // Update last accessed time when a new profile is created during login
    useEffect(() => {
        if (shouldUpdateLastAccessed.current && currentProfile?.Id) {
            updateProfileLastAccessed(currentProfile.Id);
            shouldUpdateLastAccessed.current = false;
        }
    }, [currentProfile]);

    const availableProfiles = useMemo(() => {
        return profiles?.Profiles.map(profile => ({
            value: profile.Id,
            label: profile.Alias ?? profile.Id,
            icon: (Icons.Logos as Record<string, ReactElement>)[profile.Type],
            rightIcon: sources[profile.Source],
        })) ?? [];
    }, [profiles?.Profiles]);
    
    useEffect(() => {
        if (searchParams.size > 0) {
            if (searchParams.has("type")) {
                const databaseType = searchParams.get("type")!;
                setDatabaseType(databaseTypeItems.find(item => item.id === databaseType) ?? databaseTypeItems[0]);
            }

            if (searchParams.has("host")) setHostName(searchParams.get("host")!);
            if (searchParams.has("username")) setUsername(searchParams.get("username")!);
            if (searchParams.has("password")) setPassword(searchParams.get("password")!);
            if (searchParams.has("database")) setDatabase(searchParams.get("database")!);

            if (searchParams.has("resource")) {
                const selectedProfile = availableProfiles.find(profile => profile.value === searchParams.get("resource"));
                if (selectedProfile?.value) {
                    setSelectedAvailableProfile(selectedProfile?.value);
                    handleLoginWithProfileSubmit(selectedProfile.value);
                }
            } else if (searchParams.has("login")) {
                setTimeout(() => {
                    handleSubmit();
                    searchParams.delete("login");
                }, 10);
            } else {
                setSelectedAvailableProfile(undefined);
            }
        } else {
            setSelectedAvailableProfile(undefined);
        }
    }, [searchParams, databaseTypeItems, profiles?.Profiles, availableProfiles]);

    const handleHostNameChange = useCallback((newHostName: string) => {
        if (databaseType.id !== DatabaseType.MongoDb || !newHostName.startsWith("mongodb+srv://")) {
            // Checks the valid postgres URL
            if (databaseType.id === DatabaseType.Postgres && (newHostName.startsWith("postgres://") || newHostName.startsWith("postgresql://"))) {
                try {
                    const url = new URL(newHostName);
                    const hostname = url.hostname;
                    const username = url.username;
                    const password = url.password;
                    const database = url.pathname.substring(1);

                    // gives warning
                    if (!hostname || !username || !password || !database) {
                        toast.warning("We could not extract all required details (host, username, password, or database) from this URL. Please enter the information manually.");
                    }
                    setHostName(hostname);
                    setUsername(username);
                    setPassword(password);
                    setDatabase(database);

                    if (url.port) {
                        const advancedForm = {
                            "Port": url.port,
                            "SSL Mode": "disable"
                        };
                        setAdvancedForm(advancedForm);
                        setShowAdvanced(true);
                    }
                } catch (error) {
                    toast.warning("We could not extract all required details (host, username, password, or database) from this URL. Please enter the information manually.");
                }
            } else {
                return setHostName(newHostName);
            }
        } else {
            const url = new URL(newHostName);
            setHostName(url.hostname);
            setUsername(url.username);
            setPassword(url.password);
            setDatabase(url.pathname.substring(1));
            const advancedForm = {
                "Port": "27017",
                "URL Params": `?${url.searchParams.toString()}`,
                "DNS Enabled": "false"
            };
            if (url.port.length === 0) {
                advancedForm["Port"] = "";
                advancedForm["DNS Enabled"] = "true";
            }
            setAdvancedForm(advancedForm);
            setShowAdvanced(true);
        }
    }, [databaseType.id]);

    const fields = useMemo(() => {
        if (databaseType.id === DatabaseType.Sqlite3) {
            return <div className="flex flex-col gap-lg w-full">
                <div className="flex flex-col gap-xs w-full">
                    <Label>Database</Label>
                    {isDesktop ? (
                        <div className="flex flex-col gap-sm w-full">
                            <Input
                                value={database}
                                onChange={(e) => setDatabase(e.target.value)}
                                placeholder="Select or enter database file path"
                                data-testid="database"
                            />
                            <Button
                                onClick={handleBrowseSQLiteFile}
                                variant="outline"
                                className="w-full"
                            >
                                Browse for SQLite File
                            </Button>
                        </div>
                    ) : (
                        <SearchSelect
                            value={database}
                            onChange={setDatabase}
                            disabled={databasesLoading}
                            options={
                                databasesLoading
                                    ? []
                                    : foundDatabases?.Database?.map(db => ({
                                    value: db,
                                    label: db,
                                    icon: <CircleStackIcon className="w-4 h-4"/>,
                                })) ?? []
                            }
                            placeholder="Select Database"
                            buttonProps={{
                                "data-testid": "database",
                            }}
                        />
                    )}
                </div>
            </div>
        }
        return <div className="flex flex-col gap-lg w-full">
            { databaseType.fields?.hostname && (
                <div className="flex flex-col gap-sm w-full">
                    <Label>{databaseType.id === DatabaseType.MongoDb || databaseType.id === DatabaseType.Postgres ? "Host Name (or paste Connection URL)" : "Host Name"}</Label>
                    <Input value={hostName} onChange={(e) => handleHostNameChange(e.target.value)} data-testid="hostname" placeholder="Enter host name" />
                </div>
            )}
            { databaseType.fields?.username && (
                <div className="flex flex-col gap-sm w-full">
                    <Label>Username</Label>
                    <Input value={username} onChange={(e) => setUsername(e.target.value)} data-testid="username" placeholder="Enter username" />
                </div>
            )}
            { databaseType.fields?.password && (
                <div className="flex flex-col gap-sm w-full">
                    <Label>Password</Label>
                    <Input value={password} onChange={(e) => setPassword(e.target.value)} type="password" data-testid="password" placeholder="Enter password" />
                </div>
            )}
            { databaseType.fields?.database && (
                <div className="flex flex-col gap-sm w-full">
                    <Label>Database</Label>
                    <Input value={database} onChange={(e) => setDatabase(e.target.value)} data-testid="database" placeholder="Enter database" />
                </div>
            )}
        </div>
    }, [database, databaseType.id, databaseType.fields, databasesLoading, foundDatabases?.Database, handleHostNameChange, hostName, password, username, isDesktop, handleBrowseSQLiteFile]);

    const loginWithCredentialsEnabled = useMemo(() => {
        if (databaseType.id === DatabaseType.Sqlite3) {
            return database.length > 0;
        }
        if (databaseType.id === DatabaseType.MongoDb || databaseType.id === DatabaseType.Redis) {
            return hostName.length > 0;
        }
        if (databaseType.id === DatabaseType.MySql || databaseType.id === DatabaseType.Postgres) {
            return hostName.length > 0 && username.length > 0 && password.length > 0 && database.length > 0;
        }
        return false;
    }, [databaseType.id, hostName, username, password, database]);

    const loginWithProfileEnabled = useMemo(() => {
        return selectedAvailableProfile != null;
    }, [selectedAvailableProfile]);

    if (loading || profilesLoading)  {
        return (
            <div className={classNames("flex flex-col justify-center items-center gap-lg w-full", className)}>
                <div>
                    <Loading hideText={true} />
                </div>
                <h1 className="text-xl">
                    Logging in
                </h1>
            </div>
        );
    }

    return (
        <div className={classNames("w-fit h-fit", className, {
            "w-full h-full": advancedDirection === "vertical",
        })}>
            <div className="fixed top-4 right-4" data-testid="mode-toggle">
                <ModeToggle />
            </div>
            <div className={classNames("flex flex-col grow gap-4", {
                "justify-between": advancedDirection === "horizontal",
                "h-full": advancedDirection === "vertical" && availableProfiles.length === 0,
            })}>
                <div className={classNames("flex", {
                    "flex-row grow": advancedDirection === "horizontal",
                    "flex-col w-full gap-4": advancedDirection === "vertical",
                })}>
                    <div className={classNames("flex flex-col gap-lg grow", advancedDirection === "vertical" ? "w-full" : "w-[350px]")}>
                        {!hideHeader && (
                            <div className="flex justify-between">
                                <div className="flex items-center gap-sm text-xl">
                                    {extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-4"/>}
                                    <h1 className="text-brand-foreground">{extensions.AppName ?? "WhoDB"}</h1>
                                    <h1>Login</h1>
                                </div>
                                {
                                    error &&
                                    <Badge variant="destructive" className="self-end">
                                        {error}
                                    </Badge>
                                }
                            </div>
                        )}
                        <div className={cn("flex flex-col grow gap-4", {
                            "justify-center": advancedDirection === "horizontal",
                        })}>
                            <div className="flex flex-col gap-sm w-full">
                                <Label>Database Type</Label>
                                <SearchSelect
                                    value={databaseType?.id || ""}
                                    onChange={(value) => {
                                        const selected = databaseTypeItems.find(item => item.id === value);
                                        handleDatabaseTypeChange(selected ?? databaseTypeItems[0]);
                                    }}
                                    options={databaseTypeItems.map(item => ({
                                        value: item.id,
                                        label: item.label,
                                        icon: item.icon,
                                    }))}
                                    buttonProps={{
                                        "data-testid": "database-type-select",
                                    }}
                                    contentClassName="w-[var(--radix-popover-trigger-width)] login-select-popover"
                                />
                            </div>
                            {fields}
                        </div>
                    </div>
                    {
                        (showAdvanced && advancedForm != null) &&
                        <div className={classNames("transition-all h-full overflow-hidden flex flex-col gap-lg pt-[5px]", {
                            "w-[350px] ml-4 mt-[43px]": advancedDirection === "horizontal",
                            "w-full": advancedDirection === "vertical",
                        })}>
                            {entries(advancedForm).map(([key, value]) => (
                                <div className="flex flex-col gap-sm" key={key}>
                                    <Label htmlFor={`${key}-input`}>{key}</Label>
                                    <Input
                                        id={`${key}-input`}
                                        value={value}
                                        onChange={e => handleAdvancedForm(key, e.target.value)}
                                        data-testid={`${key}-input`}
                                    />
                                </div>
                            ))}
                        </div>
                    }
                </div>
                <div className={classNames("flex login-action-buttons", {
                    "justify-end": advancedForm == null,
                    "justify-between": advancedForm != null,
                })}>
                    <Button className={classNames({
                        "hidden": advancedForm == null,
                    })} onClick={handleAdvancedToggle} data-testid="advanced-button" variant="secondary">
                        <AdjustmentsHorizontalIcon className="w-4 h-4" /> {showAdvanced ? "Less Advanced" : "Advanced"}
                    </Button>
                    {advancedDirection === "horizontal" && (
                        <Button onClick={handleSubmit} data-testid="login-button" variant={loginWithCredentialsEnabled ? "default" : "secondary"}>
                            <CheckCircleIcon className="w-4 h-4" /> Login
                        </Button>
                    )}
                </div>
                {advancedDirection === "vertical" && (
                    <div className={cn("flex flex-col justify-end", {
                        "grow": availableProfiles.length === 0,
                    })}>
                        <Button onClick={handleSubmit} data-testid="login-button" variant={loginWithCredentialsEnabled ? "default" : "secondary"}>
                            <CheckCircleIcon className="w-4 h-4" /> Login
                        </Button>
                    </div>
                )}
            </div>
            {
                availableProfiles.length > 0 &&
                <>
                    <Separator className="my-8" />
                    <div className="flex flex-col gap-4">
                        <Label>Available profiles</Label>
                        <SearchSelect
                            value={selectedAvailableProfile}
                            onChange={handleAvailableProfileChange}
                            placeholder="Select a profile"
                            contentClassName="w-[var(--radix-popover-trigger-width)]"
                            options={availableProfiles}
                            buttonProps={{
                                "data-testid": "available-profiles-select",
                            }}
                        />
                        <Button onClick={() => handleLoginWithProfileSubmit()} data-testid="login-with-profile-button" variant={loginWithProfileEnabled ? "default" : "secondary"}>
                            <CheckCircleIcon className="w-4 h-4" /> Login
                        </Button>
                    </div>
                </>
            }
        </div>
    );
};

export const LoginPage: FC = () => {
    return (
        <Container className="justify-center items-center">
            <LoginForm />
        </Container>
    );
};
