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

import {Badge, Button, Card, cn, Input, Label, ModeToggle, Separator, toast, useTheme} from '@clidey/ux';
import {SearchSelect} from '../../components/ux';
import {
    DatabaseType,
    LoginCredentials,
    useGetDatabaseLazyQuery,
    useGetProfilesQuery,
    useGetVersionQuery,
    useLoginMutation,
    useLoginWithProfileMutation,
    useSettingsConfigQuery
} from '@graphql';
import classNames from "classnames";
import entries from "lodash/entries";
import {FC, ReactElement, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {useNavigate, useSearchParams} from "react-router-dom";
import {v4} from 'uuid';
import logoImage from "../../../public/images/logo.svg";
import {
    AdjustmentsHorizontalIcon,
    ChatBubbleLeftRightIcon,
    CheckCircleIcon,
    ChevronDownIcon,
    CircleStackIcon,
    CodeBracketIcon,
    ShareIcon,
    SparklesIcon,
    TableCellsIcon
} from '../../components/heroicons';
import {Icons} from "../../components/icons";
import {Loading} from "../../components/loading";
import {Container} from "../../components/page";
import {updateProfileLastAccessed} from "../../components/profile-info-tooltip";
import {baseDatabaseTypes, getDatabaseTypeDropdownItems, IDatabaseDropdownItem} from "../../config/database-types";
import {extensions, featureFlags, sources} from '../../config/features';
import {InternalRoutes} from "../../config/routes";
import {useDesktopFile} from '../../hooks/useDesktop';
import {useTranslation} from '@/hooks/use-translation';
import {AuthActions} from "../../store/auth";
import {DatabaseActions} from "../../store/database";
import {TourActions} from "../../store/tour";
import {SettingsActions} from "../../store/settings";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {isDesktopApp} from '../../utils/external-links';
import {hasCompletedOnboarding, markOnboardingComplete} from '../../utils/onboarding';
import {AwsConnectionPicker, AwsConnectionPrefillData, DatabaseIconWithBadge, isAwsConnection} from '../../components/aws';
import {isAwsHostname} from '../../utils/cloud-connection-prefill';
import {SSLConfig, SSL_KEYS} from '../../components/ssl-config';

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
    const { t } = useTranslation('pages/login');
    const dispatch = useAppDispatch();
    const navigate = useNavigate();
    const currentProfile = useAppSelector(state => state.auth.current);
    const shouldUpdateLastAccessed = useRef(false);
    const usernameInputRef = useRef<HTMLInputElement>(null);
    const { setTheme } = useTheme();

    const FIRST_LOGIN_KEY = 'whodb_has_logged_in';
    const [isFirstLogin, setIsFirstLogin] = useState(() => {
        return !localStorage.getItem(FIRST_LOGIN_KEY);
    });

    const [login, { loading: loginLoading }] = useLoginMutation();
    const [loginWithProfile, { loading: loginWithProfileLoading }] = useLoginWithProfileMutation();
    const [getDatabases, { loading: databasesLoading, data: foundDatabases }] = useGetDatabaseLazyQuery();
    const { loading: profilesLoading, data: profiles } = useGetProfilesQuery();
    const { data: settingsData } = useSettingsConfigQuery();
    const cloudProvidersEnabled = settingsData?.SettingsConfig?.CloudProvidersEnabled ?? false;
    const disableCredentialForm = settingsData?.SettingsConfig?.DisableCredentialForm ?? false;

    useEffect(() => {
        dispatch(SettingsActions.setCloudProvidersEnabled(cloudProvidersEnabled));
    }, [cloudProvidersEnabled, dispatch]);

    const [searchParams, setSearchParams] = useSearchParams();

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

    const markFirstLoginComplete = useCallback(() => {
        if (isFirstLogin) {
            localStorage.setItem(FIRST_LOGIN_KEY, 'true');
            setIsFirstLogin(false);
            markOnboardingComplete();
        }
    }, [isFirstLogin, FIRST_LOGIN_KEY]);

    const handleSubmit = useCallback(() => {
        if (([DatabaseType.MySql, DatabaseType.Postgres].includes(databaseType.id as DatabaseType) && (hostName.length === 0 || database.length === 0 || username.length === 0))
            || (databaseType.id === DatabaseType.Sqlite3 && database.length === 0)
            || ((databaseType.id === DatabaseType.MongoDb || databaseType.id === DatabaseType.Redis) && (hostName.length === 0))) {
            return setError(t('allFieldsRequired'));
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
                    const sslMode = advancedForm[SSL_KEYS.MODE];
                    const profileData = {
                        ...credentials,
                        SSLConfigured: sslMode != null && sslMode !== 'disabled' && sslMode !== '',
                    };
                    shouldUpdateLastAccessed.current = true;
                    dispatch(AuthActions.login(profileData));
                    markFirstLoginComplete();

                    // Clear login-related URL params before navigation
                    if (searchParams.has("credentials") || searchParams.has("login")) {
                        const newParams = new URLSearchParams(searchParams);
                        newParams.delete("credentials");
                        newParams.delete("login");
                        newParams.delete("type");
                        newParams.delete("host");
                        newParams.delete("username");
                        newParams.delete("password");
                        newParams.delete("database");
                        setSearchParams(newParams, { replace: true });
                    }

                    if (onLoginSuccess) {
                        onLoginSuccess();
                    } else {
                        navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    }
                    return toast.success(t('loginSuccessful'));
                }
                return toast.error(t('loginFailed'));
            },
            onError(error) {
                return toast.error(t('loginFailedWithError', { error: error.message }));
            }
        });
    }, [databaseType.id, hostName, database, username, password, advancedForm, login, dispatch, navigate, onLoginSuccess, markFirstLoginComplete, t]);

    const handleLoginWithProfileSubmit = useCallback((overrideProfileId?: string) => {
        const profileId = overrideProfileId ?? selectedAvailableProfile;
        if (profileId == null) {
            return setError(t('selectProfileRequired'));
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
                        SSLConfigured: profile?.SSLConfigured ?? false,
                    }));
                    markFirstLoginComplete();

                    // Clear login-related URL params before navigation
                    if (searchParams.has("resource")) {
                        const newParams = new URLSearchParams(searchParams);
                        newParams.delete("resource");
                        setSearchParams(newParams, { replace: true });
                    }

                    if (onLoginSuccess) {
                        onLoginSuccess();
                    } else {
                        navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    }
                    return toast.success(t('loginSuccessful'));
                }
                return toast.error(t('loginFailed'));
            },
            onError(error) {
                return toast.error(t('loginFailedWithError', { error: error.message }));
            }
        });
    }, [dispatch, loginWithProfile, navigate, profiles?.Profiles, selectedAvailableProfile, onLoginSuccess, markFirstLoginComplete, t]);

    const handleSampleDatabaseLogin = useCallback(() => {
        const sampleProfile = profiles?.Profiles.find(p => p.Source === "builtin");
        if (!sampleProfile) {
            return toast.error(t('sampleDatabaseNotFound'));
        }

        setError(undefined);

        loginWithProfile({
            variables: {
                profile: {
                    Id: sampleProfile.Id,
                    Type: sampleProfile.Type as DatabaseType,
                },
            },
            onCompleted(data) {
                if (data.LoginWithProfile.Status) {
                    updateProfileLastAccessed(sampleProfile.Id);
                    dispatch(AuthActions.login({
                        Type: sampleProfile.Type as DatabaseType,
                        Id: sampleProfile.Id,
                        Database: sampleProfile.Database ?? "",
                        Hostname: "",
                        Password: "",
                        Username: "",
                        Saved: true,
                        IsEnvironmentDefined: sampleProfile.IsEnvironmentDefined ?? false,
                        SSLConfigured: sampleProfile.SSLConfigured ?? false,
                    }));
                    markFirstLoginComplete();
                    if (featureFlags.autoStartTourOnLogin) {
                        dispatch(TourActions.scheduleTourOnLoad('sample-database-tour'));
                    }
                    if (onLoginSuccess) {
                        onLoginSuccess();
                    } else {
                        navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    }
                    return toast.success(t('welcomeToWhodb'));
                }
                return toast.error(t('loginFailed'));
            },
            onError(error) {
                return toast.error(t('loginFailedWithError', { error: error.message }));
            }
        });
    }, [dispatch, loginWithProfile, navigate, profiles?.Profiles, onLoginSuccess, markFirstLoginComplete, t]);

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

    /**
     * Handle prefill from AWS connection picker.
     * Updates the main login form with discovered connection details,
     * then focuses the username field for easy credential entry.
     */
    const handleAwsConnectionPrefill = useCallback((data: AwsConnectionPrefillData) => {
        // Find the database type in our dropdown items
        const dbType = databaseTypeItems.find(item =>
            item.id.toLowerCase() === data.databaseType.toLowerCase()
        );

        if (dbType) {
            // Use the proper handler to set database type and reset fields
            handleDatabaseTypeChange(dbType);

            // Set hostname and advanced settings after type change completes
            setTimeout(() => {
                setHostName(data.hostname);

                // Merge advanced settings
                if (data.advanced && Object.keys(data.advanced).length > 0) {
                    setAdvancedForm(prev => ({
                        ...prev,
                        ...data.advanced,
                    }));
                    setShowAdvanced(true);
                }

                // Focus username field after form updates
                setTimeout(() => {
                    usernameInputRef.current?.focus();
                }, 50);
            }, 0);
        }
    }, [databaseTypeItems, handleDatabaseTypeChange]);

    const handleBrowseSQLiteFile = useCallback(async () => {
        try {
            const filePath = await selectSQLiteDatabase();
            if (filePath) {
                setDatabase(filePath);
            }
        } catch (error) {
            console.error('Failed to select SQLite database:', error);
            toast.error(t('failedToSelectDatabaseFile'));
        }
    }, [selectSQLiteDatabase, t]);

    useEffect(() => {
        dispatch(DatabaseActions.setSchema(""));
    }, [dispatch]);

    // Detect embedded mode from URL parameters
    useEffect(() => {
        const hasAutoLoginParams = searchParams.has("credentials") ||
                                   searchParams.has("resource") ||
                                   searchParams.has("login");
        if (hasAutoLoginParams) {
            dispatch(AuthActions.setEmbedded(true));
        }
    }, [searchParams, dispatch]);

    // Handle locale URL parameter
    useEffect(() => {
        if (searchParams.has("locale")) {
            const locale = searchParams.get("locale")?.toLowerCase();
            if (locale === 'en' || locale === 'es') {
                dispatch(SettingsActions.setLanguage(locale));
            }
        }
    }, [searchParams, dispatch]);

    // Handle theme URL parameter
    useEffect(() => {
        if (searchParams.has("theme")) {
            const theme = searchParams.get("theme")?.toLowerCase();
            if (theme === 'light' || theme === 'dark' || theme === 'system') {
                setTheme(theme);
            }
        }
    }, [searchParams, setTheme]);

    // Load database types, filtering out AWS types when cloud providers are disabled
    useEffect(() => {
        getDatabaseTypeDropdownItems({ cloudProvidersEnabled }).then(items => {
            setDatabaseTypeItems(items);
        });
    }, [cloudProvidersEnabled]);

    // Update last accessed time when a new profile is created during login
    useEffect(() => {
        if (shouldUpdateLastAccessed.current && currentProfile?.Id) {
            updateProfileLastAccessed(currentProfile.Id);
            shouldUpdateLastAccessed.current = false;
        }
    }, [currentProfile]);

    const availableProfiles = useMemo(() => {
        return profiles?.Profiles
            .filter(profile => profile.Source !== "builtin")
            // Filter out AWS-hosted profiles when cloud providers are disabled
            .filter(profile => cloudProvidersEnabled || !isAwsHostname(profile.Hostname))
            .map(profile => ({
                value: profile.Id,
                label: profile.Alias ?? profile.Id,
                icon: (
                    <DatabaseIconWithBadge
                        icon={(Icons.Logos as Record<string, ReactElement>)[profile.Type]}
                        showCloudBadge={isAwsConnection(profile.Id)}
                        sslStatus={profile.SSLConfigured ? { IsEnabled: true, Mode: 'configured' } : undefined}
                        size="sm"
                    />
                ),
                rightIcon: sources[profile.Source],
            })) ?? [];
    }, [profiles?.Profiles, cloudProvidersEnabled]);

    const hasAvailableProfiles = availableProfiles.length > 0;

    const sampleProfile = useMemo(() => {
        return profiles?.Profiles.find(p => p.Source === "builtin");
    }, [profiles?.Profiles]);

    // Handle URL parameters for pre-filling credentials or auto-login
    // Note: This effect intentionally does NOT clear selectedAvailableProfile because:
    // 1. Initial state is already undefined via useState
    // 2. Clearing on re-runs would reset user's manual profile selection
    // 3. Multiple dependencies (handleSubmit, profiles, etc.) can trigger re-runs
    useEffect(() => {
        if (searchParams.size === 0) {
            return;
        }

        // Handle credentials parameter (base64 encoded JSON)
        if (searchParams.has("credentials")) {
            try {
                const credentialsBase64 = searchParams.get("credentials")!;
                const credentialsJson = atob(credentialsBase64);
                const credentials = JSON.parse(credentialsJson);

                // Map Go backend field names to frontend state
                if (credentials.type) {
                    const dbType = databaseTypeItems.find(item =>
                        item.id.toLowerCase() === credentials.type.toLowerCase()
                    );
                    if (dbType) {
                        handleDatabaseTypeChange(dbType);
                    }
                }
                if (credentials.host) setHostName(credentials.host);
                if (credentials.username) setUsername(credentials.username);
                if (credentials.password) setPassword(credentials.password);
                if (credentials.database) setDatabase(credentials.database);

                if (credentials.port) {
                    setAdvancedForm(prev => ({...prev, 'Port': credentials.port}));
                    setShowAdvanced(true);
                }

                // Handle advanced/config fields
                if (credentials.config && typeof credentials.config === 'object') {
                    const advancedFormData: Record<string, string> = {};
                    for (const [key, value] of Object.entries(credentials.config)) {
                        advancedFormData[key] = String(value);
                    }
                    // Add port if provided
                    if (credentials.port) {
                        advancedFormData['Port'] = credentials.port;
                    }
                    setAdvancedForm(advancedFormData);
                    if (Object.keys(advancedFormData).length > 0) {
                        setShowAdvanced(true);
                    }
                }
            } catch (error) {
                console.error('Failed to parse credentials:', error);
                toast.error(t('failedToParseCredentials'));
            }
        } else {
            // Handle individual URL parameters (existing logic)
            if (searchParams.has("type")) {
                const typeParam = searchParams.get("type")!;
                const dbType = databaseTypeItems.find(item =>
                    item.id.toLowerCase() === typeParam.toLowerCase()
                );
                if (dbType) {
                    handleDatabaseTypeChange(dbType);
                }
            }

            if (searchParams.has("host")) setHostName(searchParams.get("host")!);
            if (searchParams.has("username")) setUsername(searchParams.get("username")!);
            if (searchParams.has("password")) setPassword(searchParams.get("password")!);
            if (searchParams.has("database")) setDatabase(searchParams.get("database")!);
        }

        // Handle auto-login with profile from URL
        if (searchParams.has("resource")) {
            const selectedProfile = availableProfiles.find(profile => profile.value === searchParams.get("resource"));
            if (selectedProfile?.value) {
                setSelectedAvailableProfile(selectedProfile?.value);
                handleLoginWithProfileSubmit(selectedProfile.value);
            }
        } else if (searchParams.has("login")) {
            setTimeout(() => {
                handleSubmit();
                const newParams = new URLSearchParams(searchParams);
                newParams.delete("login");
                setSearchParams(newParams, { replace: true });
            }, 10);
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
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

                    if (!hostname || !username || !password || !database) {
                        toast.warning(t('urlParseWarning'));
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
                    toast.warning(t('urlParseWarning'));
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
    }, [databaseType.id, t]);

    const fields = useMemo(() => {
        if (databaseType.id === DatabaseType.Sqlite3) {
            return <div className="flex flex-col gap-lg w-full">
                <div className="flex flex-col gap-xs w-full">
                    <Label htmlFor="sqlite-database">{t('database')}</Label>
                    {isDesktop ? (
                        <div className="flex flex-col gap-sm w-full">
                            <Input
                                id="sqlite-database"
                                value={database}
                                onChange={(e) => setDatabase(e.target.value)}
                                placeholder={t('selectOrEnterDatabasePath')}
                                data-testid="database"
                                aria-required="true"
                                aria-invalid={error ? "true" : undefined}
                                aria-describedby={error ? "login-error" : undefined}
                            />
                            <Button
                                onClick={handleBrowseSQLiteFile}
                                variant="outline"
                                className="w-full"
                            >
                                {t('browseForSqliteFile')}
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
                            placeholder={t('selectDatabase')}
                            buttonProps={{
                                "data-testid": "database",
                                "aria-required": "true",
                                "aria-invalid": error ? "true" : undefined,
                                "aria-describedby": error ? "login-error" : undefined,
                            }}
                            contentClassName="w-[var(--radix-popover-trigger-width)] login-select-popover"
                            rightIcon={<ChevronDownIcon className="w-4 h-4"/>}
                        />
                    )}
                </div>
            </div>
        }
        return <div className="flex flex-col gap-lg w-full">
            { databaseType.fields?.hostname && (
                <div className="flex flex-col gap-sm w-full">
                    <Label htmlFor="login-hostname">{databaseType.id === DatabaseType.MongoDb || databaseType.id === DatabaseType.Postgres ? t('hostNameOrUrl') : t('hostName')}</Label>
                    <Input id="login-hostname" value={hostName} onChange={(e) => handleHostNameChange(e.target.value)} data-testid="hostname" placeholder={t('enterHostName')} aria-required="true" aria-invalid={error ? "true" : undefined} aria-describedby={error ? "login-error" : undefined} />
                </div>
            )}
            { databaseType.fields?.username && (
                <div className="flex flex-col gap-sm w-full">
                    <Label htmlFor="login-username">{t('username')}</Label>
                    <Input ref={usernameInputRef} id="login-username" value={username} onChange={(e) => setUsername(e.target.value)} data-testid="username" placeholder={t('enterUsername')} aria-required="true" aria-invalid={error ? "true" : undefined} aria-describedby={error ? "login-error" : undefined} />
                </div>
            )}
            { databaseType.fields?.password && (
                <div className="flex flex-col gap-sm w-full">
                    <Label htmlFor="login-password">{t('password')}</Label>
                    <Input id="login-password" value={password} onChange={(e) => setPassword(e.target.value)} type="password" data-testid="password" placeholder={t('enterPassword')} aria-required="true" aria-invalid={error ? "true" : undefined} aria-describedby={error ? "login-error" : undefined} showPasswordToggle={true} />
                </div>
            )}
            { databaseType.fields?.database && (
                <div className="flex flex-col gap-sm w-full">
                    <Label htmlFor="login-database">{t('database')}</Label>
                    <Input id="login-database" value={database} onChange={(e) => setDatabase(e.target.value)} data-testid="database" placeholder={t('enterDatabase')} aria-required="true" aria-invalid={error ? "true" : undefined} aria-describedby={error ? "login-error" : undefined} />
                </div>
            )}
        </div>
    }, [database, databaseType.id, databaseType.fields, databasesLoading, foundDatabases?.Database, handleHostNameChange, hostName, password, username, isDesktop, handleBrowseSQLiteFile, t, error]);

    const loginWithCredentialsEnabled = useMemo(() => {
        if (databaseType.id === DatabaseType.Sqlite3) {
            return database.length > 0;
        }
        const redisCompatible = [DatabaseType.Redis, "ElastiCache"];
        const mongoCompatible = [DatabaseType.MongoDb, "DocumentDB"];

        if (redisCompatible.includes(databaseType.id) || mongoCompatible.includes(databaseType.id) || (databaseType.id === DatabaseType.ElasticSearch)) {
            return hostName.length > 0;
        }

        return hostName.length > 0 && username.length > 0 && password.length > 0 && database.length > 0;
    }, [databaseType.id, hostName, username, password, database]);

    const loginWithProfileEnabled = useMemo(() => {
        return selectedAvailableProfile != null;
    }, [selectedAvailableProfile]);

    if (loading || profilesLoading)  {
        return (
            <div className={classNames("flex flex-col justify-center items-center gap-lg w-full", className)}>
                <div>
                    <Loading size="lg" />
                </div>
                <h1 className="text-xl">
                    {t('loggingIn')}
                </h1>
            </div>
        );
    }

    const showSidePanel = sampleProfile && !hideHeader && featureFlags.sampleDatabaseTour && isFirstLogin && !hasCompletedOnboarding();

    return (
        <div className={classNames("w-fit h-fit", className, {
            "w-full h-full": advancedDirection === "vertical",
            "flex gap-8": showSidePanel && advancedDirection === "horizontal",
        })} data-testid="login-form-container">
            <div className={cn("fixed top-4 right-4 z-20", {
                "hidden": !showSidePanel,
            })} data-testid="mode-toggle-login">
                <ModeToggle />
            </div>
            <div className={classNames("flex flex-col grow gap-lg", {
                "justify-between": advancedDirection === "horizontal",
                "h-full": advancedDirection === "vertical" && availableProfiles.length === 0,
            })}>
                {!hideHeader && (
                    <header className="flex justify-between" data-testid="login-header">
                        <h1 className="flex items-center gap-xs text-xl">
                            {extensions.Logo ?? <img src={logoImage} alt="" className="w-auto h-8 mr-1"/>}
                            <span className="text-brand-foreground">{extensions.AppName ?? "WhoDB"}</span>
                            <span>{t('title')}</span>
                        </h1>
                        {
                            error &&
                            <Badge id="login-error" variant="destructive" className="self-end" role="alert">
                                {error}
                            </Badge>
                        }
                    </header>
                )}
                <div className={classNames("flex", {
                    "flex-row grow": advancedDirection === "horizontal",
                    "flex-col w-full gap-lg": advancedDirection === "vertical",
                })} data-testid="login-form">
                    <div className={classNames("flex flex-col gap-lg grow", advancedDirection === "vertical" ? "w-full" : "w-[350px]")}>
                        <div className={cn("flex flex-col grow gap-lg", {
                            "justify-center": advancedDirection === "horizontal" && !showSidePanel,
                        })}>
                            {disableCredentialForm && !hasAvailableProfiles ? (
                                <Card className="p-6 max-w-md">
                                    <h1 className="text-xl font-semibold">
                                        {t('noConnectionsTitle')}
                                    </h1>
                                    <p className="mt-2 text-sm text-muted-foreground">
                                        {t('noConnectionsDescription')}
                                    </p>
                                </Card>
                            ) : (
                                <>
                                    {!disableCredentialForm && (
                                        <div className="flex flex-col gap-sm w-full">
                                            <Label>{t('databaseType')}</Label>
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
                                                rightIcon={<ChevronDownIcon className="w-4 h-4"/>}
                                            />
                                        </div>
                                    )}
                                    {!disableCredentialForm && fields}
                                </>
                            )}
                        </div>
                    </div>
                    {
                        (showAdvanced && advancedForm != null) &&
                        <div className={classNames("transition-all h-full overflow-hidden flex flex-col gap-lg", {
                            "w-[350px] ml-4": advancedDirection === "horizontal",
                            "w-full": advancedDirection === "vertical",
                        })}>
                            {entries(advancedForm)
                                .filter(([key]) => !Object.values(SSL_KEYS).includes(key as any))
                                .map(([key, value]) => (
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
                            <SSLConfig
                                databaseType={databaseType.id}
                                sslModes={databaseType.sslModes}
                                advancedForm={advancedForm}
                                onAdvancedFormChange={handleAdvancedForm}
                            />
                        </div>
                    }
                </div>
                <div className={classNames("flex login-action-buttons", {
                    "justify-end": advancedForm == null,
                    "justify-between": advancedForm != null,
                })}>
                    {!disableCredentialForm && <>
                    <Button className={classNames({
                        "hidden": advancedForm == null || databaseType.id === DatabaseType.Sqlite3,
                    })} onClick={handleAdvancedToggle} data-testid="advanced-button" variant="secondary">
                        <AdjustmentsHorizontalIcon className="w-4 h-4" /> {showAdvanced ? t('lessAdvancedButton') : t('advancedButton')}
                    </Button>
                    {advancedDirection === "horizontal" && (
                        <Button onClick={handleSubmit} data-testid="login-button" variant={loginWithCredentialsEnabled ? "default" : "secondary"} disabled={!loginWithCredentialsEnabled}>
                            <CheckCircleIcon className="w-4 h-4" /> {t('loginButton')}
                        </Button>
                    )}
                    </>}
                </div>
                {advancedDirection === "vertical" && (
                    <div className={cn("flex flex-col justify-end", {
                        "grow": availableProfiles.length === 0,
                    })}>
                        {!disableCredentialForm && <>
                        <Button onClick={handleSubmit} data-testid="login-button" variant={loginWithCredentialsEnabled ? "default" : "secondary"} disabled={!loginWithCredentialsEnabled}>
                            <CheckCircleIcon className="w-4 h-4" /> {t('loginButton')}
                        </Button>
                        </>}
                    </div>
                )}
                {
                    availableProfiles.length > 0 &&
                    <>
                        {!disableCredentialForm && <Separator className="my-8" />}
                        <div className="flex flex-col gap-lg">
                            <Label>{t('availableProfiles')}</Label>
                            <SearchSelect
                                value={selectedAvailableProfile}
                                onChange={handleAvailableProfileChange}
                                placeholder={t('selectProfile')}
                                contentClassName="w-[var(--radix-popover-trigger-width)]"
                                options={availableProfiles}
                                buttonProps={{
                                    "data-testid": "available-profiles-select",
                                }}
                                rightIcon={<ChevronDownIcon className="w-4 h-4"/>}
                            />
                            <Button onClick={() => handleLoginWithProfileSubmit()} data-testid="login-with-profile-button" variant={loginWithProfileEnabled ? "default" : "secondary"} disabled={!loginWithProfileEnabled}>
                                <CheckCircleIcon className="w-4 h-4" /> {t('loginButton')}
                            </Button>
                        </div>
                    </>
                }
                {cloudProvidersEnabled && (
                    <>
                        <Separator className="my-8" />
                        <AwsConnectionPicker onSelectConnection={handleAwsConnectionPrefill} />
                    </>
                )}
            </div>
            {
                showSidePanel && advancedDirection === "horizontal" && (
                    <Card className="flex flex-col gap-6 p-8 w-[380px] shadow-xl" data-testid="sample-database-panel" aria-labelledby="sample-db-heading">
                        <div className="flex flex-col gap-4">
                            <div className="flex items-center gap-3">
                                <div className="h-14 w-14 rounded-2xl flex justify-center items-center bg-gradient-to-br from-brand to-brand/80 shadow-lg" aria-hidden="true">
                                    <SparklesIcon className="w-7 h-7 text-brand-foreground" />
                                </div>
                                <div className="flex flex-col gap-1">
                                    <h2 id="sample-db-heading" className="text-2xl font-bold text-foreground">
                                        {t('tryWhodb')}
                                    </h2>
                                    <Badge variant="secondary" className="w-fit">
                                        {t('noSetupRequired')}
                                    </Badge>
                                </div>
                            </div>

                            <p className="text-base text-muted-foreground leading-relaxed">
                                {t('experienceDescription')}
                            </p>
                        </div>

                        <Separator />

                        <div className="flex flex-col gap-3" role="list" aria-label={t('whatsIncluded')}>
                            <h3 className="text-sm font-semibold text-foreground uppercase tracking-wide">
                                {t('whatsIncluded')}
                            </h3>
                            <div className="flex flex-col gap-3">
                                <div className="flex items-start gap-3" role="listitem">
                                    <div className="h-6 w-6 rounded-lg flex justify-center items-center bg-brand/10 mt-0.5" aria-hidden="true">
                                        <ChatBubbleLeftRightIcon className="w-3.5 h-3.5 stroke-brand" />
                                    </div>
                                    <div className="flex flex-col gap-1">
                                        <p className="text-sm font-medium text-foreground">{t('aiChatAssistant')}</p>
                                        <p className="text-xs text-muted-foreground">{t('aiChatDescription')}</p>
                                    </div>
                                </div>
                                <div className="flex items-start gap-3" role="listitem">
                                    <div className="h-6 w-6 rounded-lg flex justify-center items-center bg-brand/10 mt-0.5" aria-hidden="true">
                                        <ShareIcon className="w-3.5 h-3.5 stroke-brand" />
                                    </div>
                                    <div className="flex flex-col gap-1">
                                        <p className="text-sm font-medium text-foreground">{t('visualSchema')}</p>
                                        <p className="text-xs text-muted-foreground">{t('visualSchemaDescription')}</p>
                                    </div>
                                </div>
                                <div className="flex items-start gap-3" role="listitem">
                                    <div className="h-6 w-6 rounded-lg flex justify-center items-center bg-brand/10 mt-0.5" aria-hidden="true">
                                        <TableCellsIcon className="w-3.5 h-3.5 stroke-brand" />
                                    </div>
                                    <div className="flex flex-col gap-1">
                                        <p className="text-sm font-medium text-foreground">{t('dataGrid')}</p>
                                        <p className="text-xs text-muted-foreground">{t('dataGridDescription')}</p>
                                    </div>
                                </div>
                                <div className="flex items-start gap-3" role="listitem">
                                    <div className="h-6 w-6 rounded-lg flex justify-center items-center bg-brand/10 mt-0.5" aria-hidden="true">
                                        <CodeBracketIcon className="w-3.5 h-3.5 stroke-brand" />
                                    </div>
                                    <div className="flex flex-col gap-1">
                                        <p className="text-sm font-medium text-foreground">{t('sqlEditor')}</p>
                                        <p className="text-xs text-muted-foreground">{t('sqlEditorDescription')}</p>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <Button
                            onClick={handleSampleDatabaseLogin}
                            data-testid="get-started-sample-db"
                            size="lg"
                            className="w-full mt-2"
                        >
                            <SparklesIcon className="w-4 h-4" aria-hidden="true" />
                            {t('getStarted')}
                        </Button>

                        <p className="text-xs text-center text-muted-foreground">
                            {t('quickStartFooter')}
                        </p>
                    </Card>
                )
            }
        </div>
    );
};

export const LoginPage: FC = () => {
    const { t } = useTranslation('pages/login');
    const {data: version} = useGetVersionQuery();

    return (
        <Container className="justify-center items-center">
            <LoginForm />
            <div className="fixed bottom-4 left-1/2 -translate-x-1/2 text-xs text-foreground/60">
                {t('version')}: {version?.Version}
            </div>
        </Container>
    );
};
