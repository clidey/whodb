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

import { useLazyQuery, useMutation, useQuery } from "@apollo/client/react";
import {Badge, Button, Card, cn, Input, Label, ModeToggle, Separator, toast, useTheme} from '@clidey/ux';
import {SearchSelect} from '../../components/ux';
import {
    SettingsConfigDocument,
    SourceFieldOptionsDocument,
    SourceProfilesDocument,
    LoginSourceDocument,
    TestSourceConnectionDocument,
    LoginWithSourceProfileDocument,
    SourceHostInputMode,
    SourceHostInputUrlParser,
} from '@graphql';
import classNames from "classnames";
import {FC, ReactElement, Suspense, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {getComponent} from "../../config/component-registry";
import {useNavigate, useSearchParams} from "react-router-dom";
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
import {SourceTypeItem} from "../../config/source-types";
import {extensions, featureFlags, getAppName, sources} from '../../config/features';
import {InternalRoutes} from "../../config/routes";
import {useSourceTypeItems} from "../../hooks/useSourceCatalog";
import {useDesktopFile} from '../../hooks/useDesktop';
import {useTranslation} from '@/hooks/use-translation';
import {AuthActions} from "../../store/auth";
import {DatabaseActions} from "../../store/database";
import {TourActions} from "../../store/tour";
import {SettingsActions} from "../../store/settings";
import {isSupportedLanguage} from "@/utils/languages";
import {HealthActions} from "../../store/health";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {isDesktopApp} from '../../utils/external-links';
import {v4 as uuidv4} from 'uuid';
import {hasCompletedOnboarding, markOnboardingComplete} from '../../utils/onboarding';
import {
    AwsConnectionPicker,
    DatabaseIconWithBadge,
    isAwsConnection
} from '../../components/aws';
import {AzureConnectionPicker, isAzureConnection} from '../../components/azure';
import {
    GcpConnectionPicker,
    isGcpConnection,
} from '../../components/gcp';
import {ConnectionPrefillData, isAwsHostname, isAzureHostname, isGcpHostname} from '../../utils/cloud-connection-prefill';
import { SourceAdvancedFields } from '@/components/source-advanced-fields';
import { clearGraphqlStore } from '@/config/graphql-client';
import {
    buildRecordInputs,
    buildSourceValues,
    createProfilePayload,
    createProfilePayloadFromSourceProfile,
    getValue,
} from '../../utils/source-credentials';
import {
    buildSourceAdvancedSectionState,
    canSubmitCustomConnectionForm,
    canSubmitStandardConnectionForm,
    findConnectionFieldByKey,
    getPromotedConnectionFieldKeys,
    supportsDatabaseFieldOptions,
    usesFileTransport,
} from '@/utils/source-connection-form';
import { SSL_KEYS } from '@/utils/source-ssl';

/**
 * URL params that are reserved for the standard login form fields and control flags.
 * These are never treated as advanced database fields.
 */
const LOGIN_RESERVED_PARAMS = new Set([
    "type", "host", "username", "password", "database",
    "port", "region", "search_path",
    "login", "resource", "credentials",
]);

/**
 * URL params that control UI behavior and should be preserved after login.
 */
const LOGIN_UI_PARAMS = new Set(["locale", "mode", "theme", "os"]);

function getLoginUiSearchParams(searchParams: URLSearchParams): URLSearchParams {
    const uiParams = new URLSearchParams();
    LOGIN_UI_PARAMS.forEach(key => {
        if (searchParams.has(key)) {
            uiParams.set(key, searchParams.get(key)!);
        }
    });
    return uiParams;
}

function getStorageUnitPath(searchParams: URLSearchParams): string {
    const uiParams = getLoginUiSearchParams(searchParams);
    const uiSearch = uiParams.toString();
    return `${InternalRoutes.Dashboard.StorageUnit.path}${uiSearch ? `?${uiSearch}` : ""}`;
}

const EMPTY_DATABASE_TYPE: SourceTypeItem = {
    id: "",
    label: "",
    connector: "",
    icon: <span className="w-6 h-6" />,
    extra: {},
    fields: {},
    requiredFields: {},
};


/**
 * Generate a consistent ID for desktop credentials based on connection details.
 * This ensures the same credentials always produce the same ID, preventing duplicate keyring entries.
 * For browser environments, returns undefined to rely on cookie-based auth.
 */
function generateCredentialId(type: string, hostname: string, username: string, database: string): string | undefined {
    // browser environment just uses a random ID
    if (!isDesktopApp()) {
        return uuidv4();
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
        return uuidv4();
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
    const appName = getAppName();
    const dispatch = useAppDispatch();
    const navigate = useNavigate();
    const currentProfile = useAppSelector(state => state.auth.current);
    const shouldUpdateLastAccessed = useRef(false);
    const usernameInputRef = useRef<HTMLInputElement>(null);
    const handleSubmitRef = useRef<() => void>(() => {});
    const handleLoginWithSourceProfileSubmitRef = useRef<(overrideProfileId?: string) => void>(() => {});
    const [pendingAutoLogin, setPendingAutoLogin] = useState(false);
    const { setTheme } = useTheme();

    const FIRST_LOGIN_KEY = 'whodb_has_logged_in';
    const [isFirstLogin, setIsFirstLogin] = useState(() => {
        return !localStorage.getItem(FIRST_LOGIN_KEY);
    });

    const [login, { loading: loginLoading }] = useMutation(LoginSourceDocument);
    const [testConnection, { loading: testConnectionLoading }] = useMutation(TestSourceConnectionDocument);
    const [loginWithSourceProfile, { loading: loginWithSourceProfileLoading }] = useMutation(LoginWithSourceProfileDocument);
    const [getDatabases, { loading: databasesLoading, data: foundDatabases }] = useLazyQuery(SourceFieldOptionsDocument);
    const { loading: profilesLoading, data: profiles } = useQuery(SourceProfilesDocument);
    const { data: settingsData } = useQuery(SettingsConfigDocument);
    const cloudProvidersEnabled = settingsData?.SettingsConfig?.CloudProvidersEnabled ?? false;
    const awsProviderEnabled = settingsData?.SettingsConfig?.AWSProviderEnabled ?? false;
    const azureProviderEnabled = settingsData?.SettingsConfig?.AzureProviderEnabled ?? false;
    const gcpProviderEnabled = settingsData?.SettingsConfig?.GCPProviderEnabled ?? false;
    const disableCredentialForm = settingsData?.SettingsConfig?.DisableCredentialForm ?? false;
    const maxPageSize = settingsData?.SettingsConfig?.MaxPageSize ?? 10000;
    const {
        items: databaseTypeItems,
        loading: databaseTypesLoading,
        error: databaseTypesError,
    } = useSourceTypeItems({ cloudProvidersEnabled, awsProviderEnabled });
    const [searchParams, setSearchParams] = useSearchParams();

    const databaseTypesLoaded = !databaseTypesLoading;
    const [databaseType, setDatabaseType] = useState<SourceTypeItem>(EMPTY_DATABASE_TYPE);
    const [hostName, setHostName] = useState("");
    const [database, setDatabase] = useState("");
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string>();
    const [missingDriver, setMissingDriver] = useState<string | null>(null);
    const [advancedForm, setAdvancedForm] = useState<Record<string, string>>({});
    const [showAdvanced, setShowAdvanced] = useState(false);
    const [formResetKey, setFormResetKey] = useState(0);
    const [selectedAvailableProfile, setSelectedAvailableProfile] = useState<string>();
    const [isAutoLoggingIn, setIsAutoLoggingIn] = useState(() => {
        // Detect auto-login on initial render to prevent flash of login form
        return searchParams.has("resource") || searchParams.has("login");
    });
    const [isEmbedded] = useState(() => {
        return searchParams.has("credentials") || searchParams.has("resource") || searchParams.has("login");
    });
    const { isDesktop, selectDatabaseFile } = useDesktopFile();

    useEffect(() => {
        dispatch(SettingsActions.setCloudProvidersEnabled(cloudProvidersEnabled));
        dispatch(SettingsActions.setAWSProviderEnabled(awsProviderEnabled));
        dispatch(SettingsActions.setAzureProviderEnabled(azureProviderEnabled));
        dispatch(SettingsActions.setGCPProviderEnabled(gcpProviderEnabled));
        dispatch(SettingsActions.setMaxPageSize(maxPageSize));
    }, [cloudProvidersEnabled, awsProviderEnabled, azureProviderEnabled, gcpProviderEnabled, maxPageSize, dispatch]);

    useEffect(() => {
        if (databaseTypeItems.length === 0) {
            return;
        }

        const nextType = databaseTypeItems.find(item => item.id === databaseType.id) ?? databaseTypeItems[0];
        if (databaseType.id !== nextType.id) {
            setDatabaseType(nextType);
            setAdvancedForm(nextType.extra ?? {});
        }
    }, [databaseType.id, databaseTypeItems]);

    useEffect(() => {
        if (databaseTypesError) {
            console.error('Failed to load source catalog:', databaseTypesError);
        }
    }, [databaseTypesError]);

    const loading = useMemo(() => {
        return loginLoading || loginWithSourceProfileLoading || isAutoLoggingIn;
    }, [loginLoading, loginWithSourceProfileLoading, isAutoLoggingIn]);

    const markFirstLoginComplete = useCallback(() => {
        if (isFirstLogin) {
            localStorage.setItem(FIRST_LOGIN_KEY, 'true');
            setIsFirstLogin(false);
            markOnboardingComplete();
        }
    }, [isFirstLogin, FIRST_LOGIN_KEY]);

    const handleLoginError = useCallback((loginError: unknown, allowDriverInstallPrompt = false) => {
        setIsAutoLoggingIn(false);

        const errorMessage = loginError instanceof Error ? loginError.message : String(loginError);

        if (allowDriverInstallPrompt) {
            const driverMatch = errorMessage.match(/driver_not_installed:(\w+)/);
            if (driverMatch) {
                setMissingDriver(driverMatch[1]);
                return;
            }
        }

        const lowerCaseMessage = errorMessage.toLowerCase();
        const isNetworkError = lowerCaseMessage.includes('network') ||
            lowerCaseMessage.includes('fetch') ||
            lowerCaseMessage.includes('econnrefused') ||
            (loginError instanceof Error && 'statusCode' in loginError);

        if (isNetworkError) {
            dispatch(HealthActions.setHealthStatus({
                server: 'error',
                database: 'unavailable',
            }));
        }

        toast.error(t('loginFailedWithError', { error: errorMessage }));
    }, [dispatch, t]);

    const handleSubmit = useCallback(() => {
        const credentialsAreComplete = databaseType.customFormRenderer != null
            ? canSubmitCustomConnectionForm(databaseType, hostName, username, password, advancedForm)
            : canSubmitStandardConnectionForm(databaseType, hostName, username, password, database, advancedForm);

        if (databaseType.id === "" || !credentialsAreComplete) {
            setIsAutoLoggingIn(false);
            return setError(t('allFieldsRequired'));
        }
        setError(undefined);

        // Generate ID only for desktop apps, using consistent ID for same credentials
        const credentialId = generateCredentialId(databaseType.id, hostName, username, database);

        const values = buildSourceValues(hostName, database, username, password, advancedForm);
        const credentials = {
            Id: credentialId,
            SourceType: databaseType.id,
            Values: buildRecordInputs(hostName, database, username, password, advancedForm),
        };

        void (async () => {
            try {
                const { data } = await login({
                    variables: {
                        credentials,
                    },
                });

                if (!data?.LoginSource.Status) {
                    setIsAutoLoggingIn(false);
                    toast.error(t('loginFailed'));
                    return;
                }

                const sslMode = advancedForm[SSL_KEYS.MODE];
                const profileData = createProfilePayload(credentialId, databaseType.id, values, {
                    SSLConfigured: sslMode != null && sslMode !== 'disabled' && sslMode !== '',
                });

                await clearGraphqlStore();
                shouldUpdateLastAccessed.current = true;
                dispatch(AuthActions.login(profileData));
                markFirstLoginComplete();

                const storageUnitPath = getStorageUnitPath(searchParams);

                // Clear all login-related URL params before navigation, preserving UI-only params.
                const hasLoginParams = [...searchParams.keys()].some(k => !LOGIN_UI_PARAMS.has(k));
                if (hasLoginParams) {
                    setSearchParams(getLoginUiSearchParams(searchParams), { replace: true });
                }

                if (onLoginSuccess) {
                    onLoginSuccess();
                } else {
                    navigate(storageUnitPath);
                }
                toast.success(t('loginSuccessful'));
            } catch (error) {
                handleLoginError(error, true);
            }
        })();
    }, [advancedForm, database, databaseType, dispatch, handleLoginError, hostName, login, markFirstLoginComplete, navigate, onLoginSuccess, password, searchParams, setSearchParams, t, username]);

    const handleTestConnection = useCallback(() => {
        const values = buildRecordInputs(hostName, database, username, password, advancedForm);
        void (async () => {
            try {
                const { data } = await testConnection({
                    variables: {
                        credentials: {
                            SourceType: databaseType.id,
                            Values: values,
                        },
                    },
                });
                if (data?.TestSourceConnection.Status) {
                    toast.success(t('testConnectionSuccess'));
                }
            } catch (e: any) {
                toast.error(t('testConnectionFailed', { error: e?.message ?? '' }));
            }
        })();
    }, [advancedForm, database, databaseType.id, hostName, password, testConnection, t, username]);

    const handleLoginWithSourceProfileSubmit = useCallback((overrideProfileId?: string) => {
        const profileId = overrideProfileId ?? selectedAvailableProfile;
        if (profileId == null) {
            return setError(t('selectProfile'));
        }
        setError(undefined);

        const profile = profiles?.SourceProfiles.find(p => p.Id === profileId);

        void (async () => {
            try {
                const { data } = await loginWithSourceProfile({
                    variables: {
                        profile: {
                            Id:  profileId,
                        },
                    },
                });

                if (!data?.LoginWithSourceProfile.Status) {
                    setIsAutoLoggingIn(false);
                    toast.error(t('loginFailed'));
                    return;
                }

                updateProfileLastAccessed(profileId);
                await clearGraphqlStore();
                if (profile != null) {
                    dispatch(AuthActions.login(createProfilePayloadFromSourceProfile(profile)));
                }
                markFirstLoginComplete();

                const storageUnitPath = getStorageUnitPath(searchParams);

                // Clear login-related URL params before navigation
                if (searchParams.has("resource")) {
                    setSearchParams(getLoginUiSearchParams(searchParams), { replace: true });
                }

                if (onLoginSuccess) {
                    onLoginSuccess();
                } else {
                    navigate(storageUnitPath);
                }
                toast.success(t('loginSuccessful'));
            } catch (error) {
                handleLoginError(error);
            }
        })();
    }, [dispatch, handleLoginError, loginWithSourceProfile, markFirstLoginComplete, navigate, onLoginSuccess, profiles?.SourceProfiles, searchParams, selectedAvailableProfile, setSearchParams, t]);

    // Keep refs in sync with latest callback versions each render to avoid stale closures
    handleSubmitRef.current = handleSubmit;
    handleLoginWithSourceProfileSubmitRef.current = handleLoginWithSourceProfileSubmit;

    const handleSampleDatabaseLogin = useCallback(() => {
        const sampleProfile = profiles?.SourceProfiles.find(p => p.Source === "builtin");
        if (!sampleProfile) {
            return toast.error(t('sampleDatabaseNotFound'));
        }

        setError(undefined);

        void (async () => {
            try {
                const { data } = await loginWithSourceProfile({
                    variables: {
                        profile: {
                            Id: sampleProfile.Id,
                        },
                    },
                });

                if (!data?.LoginWithSourceProfile.Status) {
                    setIsAutoLoggingIn(false);
                    toast.error(t('loginFailed'));
                    return;
                }

                updateProfileLastAccessed(sampleProfile.Id);
                await clearGraphqlStore();
                dispatch(AuthActions.login(createProfilePayloadFromSourceProfile(sampleProfile)));
                markFirstLoginComplete();
                if (featureFlags.autoStartTourOnLogin) {
                    dispatch(TourActions.scheduleTourOnLoad('sample-database-tour'));
                }
                if (onLoginSuccess) {
                    onLoginSuccess();
                } else {
                    navigate(InternalRoutes.Dashboard.StorageUnit.path);
                }
                toast.success(t('welcomeToWhodb', { appName }));
            } catch (error) {
                handleLoginError(error);
            }
        })();
    }, [appName, dispatch, handleLoginError, loginWithSourceProfile, markFirstLoginComplete, navigate, onLoginSuccess, profiles?.SourceProfiles, t]);

    const handleDatabaseTypeChange = useCallback((item: SourceTypeItem) => {
        setHostName("");
        setUsername("");
        setPassword("");
        setDatabase("");
        setDatabaseType(item);
        setAdvancedForm(item.extra ?? {});
        setFormResetKey(k => k + 1);
    }, []);

    const handleAdvancedToggle = useCallback(() => {
        setShowAdvanced(a => !a);
    }, []);

    // Fetch available databases for file-based types after the form re-mounts.
    // This must be in useEffect (not in handleDatabaseTypeChange) because
    // setFormResetKey causes a re-mount that resets the useLazyQuery hook state.
    useEffect(() => {
        if (supportsDatabaseFieldOptions(databaseType)) {
            getDatabases({ variables: { sourceType: databaseType.id } });
        }
    }, [databaseType, getDatabases, formResetKey]);

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
     * Handle prefill from a cloud connection picker (AWS, Azure, GCP).
     * Updates the main login form with discovered connection details,
     * then focuses the username field for easy credential entry.
     */
    const handleCloudConnectionPrefill = useCallback((data: ConnectionPrefillData) => {
        // Find the database type in our dropdown items
        const dbType = databaseTypeItems.find(item =>
            item.id.toLowerCase() === data.databaseType.toLowerCase()
        );

        if (dbType) {
            // Use the proper handler to set database type and reset fields
            handleDatabaseTypeChange(dbType);

            // Set hostname and advanced settings after type change completes
            setTimeout(() => {
                if (data.hostname) {
                    setHostName(data.hostname);
                }
                if (data.database) {
                    setDatabase(data.database);
                }

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

    const handleBrowseDatabaseFile = useCallback(async () => {
        try {
            const filePath = await selectDatabaseFile(databaseType.id);
            if (filePath) {
                setDatabase(filePath);
            }
        } catch (error) {
            console.error('Failed to select database file:', error);
            toast.error(t('failedToSelectDatabaseFile'));
        }
    }, [selectDatabaseFile, databaseType.id, t]);

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
            const locale = searchParams.get("locale");
            if (locale && isSupportedLanguage(locale)) {
                dispatch(SettingsActions.setLanguage(locale));
            }
        }
    }, [searchParams, dispatch]);

    // Handle mode URL parameter (light/dark/system)
    useEffect(() => {
        if (searchParams.has("mode")) {
            const mode = searchParams.get("mode")?.toLowerCase();
            if (mode === 'light' || mode === 'dark' || mode === 'system') {
                setTheme(mode);
            }
        }
    }, [searchParams, setTheme]);

    // Handle theme URL parameter (visual theme name)
    useEffect(() => {
        if (searchParams.has("theme")) {
            const theme = searchParams.get("theme")?.toLowerCase();
            if (theme === 'default') {
                dispatch(SettingsActions.setAppTheme('default'));
            }
        }
    }, [searchParams, dispatch]);

    // Handle os URL parameter (keyboard shortcut OS override)
    useEffect(() => {
        if (searchParams.has("os")) {
            const os = searchParams.get("os")?.toLowerCase();
            if (os === 'linux' || os === 'macos' || os === 'windows') {
                dispatch(SettingsActions.setOS(os));
            }
        }
    }, [searchParams, dispatch]);

    // Update last accessed time when a new profile is created during login
    useEffect(() => {
        if (shouldUpdateLastAccessed.current && currentProfile?.Id) {
            updateProfileLastAccessed(currentProfile.Id);
            shouldUpdateLastAccessed.current = false;
        }
    }, [currentProfile]);

    const availableProfiles = useMemo(() => {
        return profiles?.SourceProfiles
            .filter(profile => profile.Source !== "builtin")
            .filter(profile => {
                const hostname = getValue(profile.Values, "Hostname");
                if (isAwsHostname(hostname)) return awsProviderEnabled;
                if (isAzureHostname(hostname)) return azureProviderEnabled;
                if (isGcpHostname(hostname)) return gcpProviderEnabled;
                return true;
            })
            .map(profile => ({
                value: profile.Id,
                label: profile.Alias ?? profile.Id,
                icon: (
                    <DatabaseIconWithBadge
                        icon={(Icons.Logos as Record<string, ReactElement>)[profile.Type]}
                        showCloudBadge={isAwsConnection(profile.Id) || isAzureConnection(profile.Id) || isGcpConnection(profile.Id)}
                        sslStatus={profile.SSLConfigured ? { IsEnabled: true, Mode: 'configured' } : undefined}
                        size="sm"
                    />
                ),
                rightIcon: sources[profile.Source],
            })) ?? [];
    }, [profiles?.SourceProfiles, awsProviderEnabled, azureProviderEnabled, gcpProviderEnabled]);

    const hasAvailableProfiles = availableProfiles.length > 0;

    const sampleProfile = useMemo(() => {
        return profiles?.SourceProfiles.find(p => p.Source === "builtin");
    }, [profiles?.SourceProfiles]);

    // Handle URL parameters for pre-filling credentials or auto-login
    // Note: This effect intentionally does NOT clear selectedAvailableProfile because:
    // 1. Initial state is already undefined via useState
    // 2. Clearing on re-runs would reset user's manual profile selection
    // 3. Multiple dependencies (handleSubmit, profiles, etc.) can trigger re-runs
    useEffect(() => {
        if (searchParams.size === 0) {
            return;
        }

        // Wait until database types have finished loading before processing auto-login.
        // This ensures all registered types are available for type lookup.
        if (!databaseTypesLoaded) {
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
                    setAdvancedForm(prev => ({...prev, 'Port': String(credentials.port)}));
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
                        advancedFormData['Port'] = String(credentials.port);
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
            // Handle individual URL parameters
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

            // Merge known URL params into advancedForm with their canonical key names
            const hasPort = searchParams.has("port");
            const hasRegion = searchParams.has("region");
            const hasSearchPath = searchParams.has("search_path");
            if (hasPort || hasRegion || hasSearchPath) {
                setAdvancedForm(prev => ({
                    ...prev,
                    ...(hasPort ? {'Port': searchParams.get("port")!} : {}),
                    ...(hasRegion ? {'Region': searchParams.get("region")!} : {}),
                    ...(hasSearchPath ? {'Search Path': searchParams.get("search_path")!} : {}),
                }));
                setShowAdvanced(true);
            }

            // All other non-reserved params go into advanced form generically,
            // supporting any registered database type.
            const advancedEntries: Record<string, string> = {};
            searchParams.forEach((value, key) => {
                if (!LOGIN_RESERVED_PARAMS.has(key) && !LOGIN_UI_PARAMS.has(key)) {
                    advancedEntries[key] = value;
                }
            });
            if (Object.keys(advancedEntries).length > 0) {
                setAdvancedForm(prev => ({...prev, ...advancedEntries}));
                setShowAdvanced(true);
            }
        }

        // Handle auto-login with profile from URL
        if (searchParams.has("resource")) {
            const selectedProfile = availableProfiles.find(profile => profile.value === searchParams.get("resource"));
            if (selectedProfile?.value) {
                setSelectedAvailableProfile(selectedProfile.value);
                handleLoginWithSourceProfileSubmitRef.current(selectedProfile.value);
            }
        } else if (searchParams.has("login")) {
            setPendingAutoLogin(true);
            const newParams = new URLSearchParams(searchParams);
            newParams.delete("login");
            setSearchParams(newParams, { replace: true });
        }
    }, [searchParams, databaseTypeItems, databaseTypesLoaded, profiles?.SourceProfiles, availableProfiles, handleDatabaseTypeChange]);

    // Fire credential-based login after React has committed all form-field state updates.
    // Using a state flag (not a ref) ensures this effect runs on the render AFTER the parsing
    // effect, so handleSubmitRef.current has fresh field values instead of the stale initial ones.
    useEffect(() => {
        if (!pendingAutoLogin) return;
        setPendingAutoLogin(false);
        handleSubmitRef.current();
    }, [pendingAutoLogin]);

    const handleHostNameChange = useCallback((newHostName: string) => {
        const urlParser = databaseType.traits?.connection.hostInputUrlParser ?? SourceHostInputUrlParser.None;

        if (urlParser === SourceHostInputUrlParser.MongoSrv && newHostName.startsWith("mongodb+srv://")) {
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
            return;
        }

        if (urlParser === SourceHostInputUrlParser.Postgres && (newHostName.startsWith("postgres://") || newHostName.startsWith("postgresql://"))) {
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
            return;
        }

        setHostName(newHostName);
    }, [databaseType.traits?.connection.hostInputUrlParser, t]);

    const fields = useMemo(() => {
        if (databaseType.customFormRenderer) {
            const CustomForm = databaseType.customFormRenderer;
            return <CustomForm
                key={formResetKey}
                hostName={hostName}
                setHostName={setHostName}
                username={username}
                setUsername={setUsername}
                password={password}
                setPassword={setPassword}
                advancedForm={advancedForm}
                setAdvancedForm={setAdvancedForm}
            />;
        }

        const hostnameField = findConnectionFieldByKey(databaseType, "Hostname");
        const portField = findConnectionFieldByKey(databaseType, "Port");
        const usernameField = findConnectionFieldByKey(databaseType, "Username");
        const passwordField = findConnectionFieldByKey(databaseType, "Password");
        const databaseField = findConnectionFieldByKey(databaseType, "Database");
        const searchPathField = findConnectionFieldByKey(databaseType, "Search Path");

        if (usesFileTransport(databaseType)) {
            return <div className="flex flex-col gap-lg w-full">
                <div className="flex flex-col gap-xs w-full">
                    <Label htmlFor="sqlite-database">{t(databaseField?.LabelKey ?? 'database')}</Label>
                    {isDesktop ? (
                        <div className="flex flex-col gap-sm w-full">
                            <Input
                                id="sqlite-database"
                                value={database}
                                onChange={(e) => setDatabase(e.target.value)}
                                placeholder={databaseField?.PlaceholderKey ? t(databaseField.PlaceholderKey) : t('selectOrEnterDatabasePath')}
                                data-testid="database"
                                aria-required={databaseField?.Required ? "true" : undefined}
                                aria-invalid={error ? "true" : undefined}
                                aria-describedby={error ? "login-error" : undefined}
                            />
                            <Button
                                onClick={handleBrowseDatabaseFile}
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
                                    : foundDatabases?.SourceFieldOptions?.map(db => ({
                                    value: db,
                                    label: db,
                                    icon: <CircleStackIcon className="w-4 h-4"/>,
                                })) ?? []
                            }
                            placeholder={t('selectDatabase')}
                            buttonProps={{
                                "data-testid": "database",
                                "aria-required": databaseField?.Required ? "true" : undefined,
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
                <div className="flex gap-sm w-full items-end">
                    <div className="flex flex-col gap-sm flex-1">
                        <Label htmlFor="login-hostname">{databaseType.traits?.connection.hostInputMode === SourceHostInputMode.HostnameOrUrl ? t('hostNameOrUrl') : t('hostName')}</Label>
                        <Input id="login-hostname" value={hostName} onChange={(e) => handleHostNameChange(e.target.value)} data-testid="hostname" placeholder={hostnameField?.PlaceholderKey ? t(hostnameField.PlaceholderKey) : t('enterHostName')} aria-required={hostnameField?.Required ? "true" : undefined} aria-invalid={error ? "true" : undefined} aria-describedby={error ? "login-error" : undefined} />
                    </div>
                    { portField && (
                        <div className="flex flex-col gap-sm w-24">
                            <Label htmlFor="login-port">{t(portField.LabelKey)}</Label>
                            <Input id="login-port" value={advancedForm['Port'] ?? portField.DefaultValue ?? ''} onChange={(e) => handleAdvancedForm('Port', e.target.value)} data-testid="port" placeholder={portField.DefaultValue ?? ''} />
                        </div>
                    )}
                </div>
            )}
            { databaseType.fields?.username && (
                <div className="flex flex-col gap-sm w-full">
                    <Label htmlFor="login-username">{t('username')}</Label>
                    <Input ref={usernameInputRef} id="login-username" value={username} onChange={(e) => setUsername(e.target.value)} data-testid="username" placeholder={usernameField?.PlaceholderKey ? t(usernameField.PlaceholderKey) : t('enterUsername')} aria-required={usernameField?.Required ? "true" : undefined} aria-invalid={error ? "true" : undefined} aria-describedby={error ? "login-error" : undefined} />
                </div>
            )}
            { databaseType.fields?.password && (
                <div className="flex flex-col gap-sm w-full">
                    <Label htmlFor="login-password">{t('password')}</Label>
                    <Input id="login-password" value={password} onChange={(e) => setPassword(e.target.value)} type="password" data-testid="password" placeholder={passwordField?.PlaceholderKey ? t(passwordField.PlaceholderKey) : t('enterPassword')} aria-required={passwordField?.Required ? "true" : undefined} aria-invalid={error ? "true" : undefined} aria-describedby={error ? "login-error" : undefined} showPasswordToggle={!isEmbedded} />
                </div>
            )}
            { databaseType.fields?.database && (
                <div className="flex flex-col gap-sm w-full">
                    <Label htmlFor="login-database">{t(databaseField?.LabelKey ?? 'database')}</Label>
                    <Input id="login-database" value={database} onChange={(e) => setDatabase(e.target.value)} data-testid="database" placeholder={databaseField?.PlaceholderKey ? t(databaseField.PlaceholderKey) : t('enterDatabase')} aria-required={databaseField?.Required ? "true" : undefined} aria-invalid={error ? "true" : undefined} aria-describedby={error ? "login-error" : undefined} />
                </div>
            )}
            { databaseType.fields?.searchPath && (
                <div className="flex flex-col gap-sm w-full">
                    <Label htmlFor="login-search-path">{t(searchPathField?.LabelKey ?? 'advancedFields.searchPath')}</Label>
                    <Input id="login-search-path" value={advancedForm['Search Path'] ?? ''} onChange={(e) => handleAdvancedForm('Search Path', e.target.value)} data-testid="search-path" placeholder={searchPathField?.PlaceholderKey ? t(searchPathField.PlaceholderKey) : t('enterSearchPath')} aria-required={searchPathField?.Required ? "true" : undefined} />
                </div>
            )}
        </div>
    }, [database, databaseType, databasesLoading, foundDatabases?.SourceFieldOptions, handleHostNameChange, hostName, password, username, isDesktop, handleBrowseDatabaseFile, advancedForm, formResetKey, t, error, isEmbedded]);

    const loginWithCredentialsEnabled = useMemo(() => {
        if (databaseType.customFormRenderer) {
            return canSubmitCustomConnectionForm(databaseType, hostName, username, password, advancedForm);
        }
        return canSubmitStandardConnectionForm(databaseType, hostName, username, password, database, advancedForm);
    }, [databaseType, hostName, username, password, database, advancedForm]);

    const loginWithSourceProfileEnabled = useMemo(() => {
        return selectedAvailableProfile != null;
    }, [selectedAvailableProfile]);

    const promotedConnectionFieldKeys = useMemo(() => {
        return getPromotedConnectionFieldKeys(databaseType);
    }, [databaseType]);

    const advancedSection = useMemo(() => {
        return buildSourceAdvancedSectionState(databaseType, advancedForm, promotedConnectionFieldKeys);
    }, [advancedForm, databaseType, promotedConnectionFieldKeys]);

    // Always show loading during auto-login, regardless of mutation or profile loading state
    // Only show form if auto-login fails (isAutoLoggingIn set to false in error handlers)
    if (!databaseTypesLoaded || isAutoLoggingIn || loading || profilesLoading)  {
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
                            {extensions.Logo ?? <img src={logoImage} alt="WhoDB" className="w-auto h-8 mr-1"/>}
                            <span className="text-brand-foreground" data-testid="app-name">{getAppName()}</span>
                        </h1>
                        <span className="text-xl">{t('title')}</span>
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
                                                    if (selected) {
                                                        handleDatabaseTypeChange(selected);
                                                    }
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
                        (showAdvanced && advancedSection.hasAdvancedSection && !databaseType.customFormRenderer) &&
                        <div className={classNames("transition-all h-full overflow-hidden flex flex-col gap-lg", {
                            "w-[350px] ml-4": advancedDirection === "horizontal",
                            "w-full": advancedDirection === "vertical",
                        })}>
                            <SourceAdvancedFields
                                databaseType={databaseType}
                                advancedState={advancedSection}
                                advancedForm={advancedForm}
                                onAdvancedFormChange={handleAdvancedForm}
                                translate={t}
                                showPasswordToggle={!isEmbedded}
                                fieldClassName="flex flex-col gap-sm"
                                checkboxClassName="flex items-center justify-between gap-sm"
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
                        "hidden": !advancedSection.hasAdvancedSection || usesFileTransport(databaseType) || databaseType.customFormRenderer != null,
                    })} onClick={handleAdvancedToggle} data-testid="advanced-button" variant="secondary">
                        <AdjustmentsHorizontalIcon className="w-4 h-4" /> {showAdvanced ? t('lessAdvancedButton') : t('advancedButton')}
                    </Button>
                    {advancedDirection === "horizontal" && (<>
                        <Button onClick={handleTestConnection} variant="secondary" disabled={!loginWithCredentialsEnabled || testConnectionLoading}>
                            {t('testConnection')}
                        </Button>
                        <Button onClick={handleSubmit} data-testid="login-button" variant={loginWithCredentialsEnabled ? "default" : "secondary"} disabled={!loginWithCredentialsEnabled}>
                            <CheckCircleIcon className="w-4 h-4" /> {t('title')}
                        </Button>
                    </>)}
                    </>}
                </div>
                {advancedDirection === "vertical" && (
                    <div className={cn("flex flex-col justify-end gap-2", {
                        "grow": availableProfiles.length === 0,
                    })}>
                        {!disableCredentialForm && <>
                        <div className="flex gap-2">
                            <Button onClick={handleTestConnection} variant="secondary" disabled={!loginWithCredentialsEnabled || testConnectionLoading} className="flex-1">
                                {t('testConnection')}
                            </Button>
                            <Button onClick={handleSubmit} data-testid="login-button" variant={loginWithCredentialsEnabled ? "default" : "secondary"} disabled={!loginWithCredentialsEnabled} className="flex-1">
                                <CheckCircleIcon className="w-4 h-4" /> {t('title')}
                            </Button>
                        </div>
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
                            <Button onClick={() => handleLoginWithSourceProfileSubmit()} data-testid="login-with-profile-button" variant={loginWithSourceProfileEnabled ? "default" : "secondary"} disabled={!loginWithSourceProfileEnabled}>
                                <CheckCircleIcon className="w-4 h-4" /> {t('title')}
                            </Button>
                        </div>
                    </>
                }
                {cloudProvidersEnabled && (
                    <>
                        {awsProviderEnabled && (
                            <>
                                <Separator className="my-8" />
                                <AwsConnectionPicker
                                    onSelectConnection={handleCloudConnectionPrefill}
                                    sourceTypes={databaseTypeItems}
                                />
                            </>
                        )}
                        {azureProviderEnabled && (
                            <>
                                <Separator className="my-8" />
                                <AzureConnectionPicker
                                    onSelectConnection={handleCloudConnectionPrefill}
                                    sourceTypes={databaseTypeItems}
                                />
                            </>
                        )}
                        {gcpProviderEnabled && (
                            <>
                                <Separator className="my-8" />
                                <GcpConnectionPicker
                                    onSelectConnection={handleCloudConnectionPrefill}
                                    sourceTypes={databaseTypeItems}
                                />
                            </>
                        )}
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
                                        {t('tryWhodb', { appName })}
                                    </h2>
                                    <Badge variant="secondary" className="w-fit">
                                        {t('noSetupRequired')}
                                    </Badge>
                                </div>
                            </div>

                            <p className="text-base text-muted-foreground leading-relaxed">
                                {t('experienceDescription', { appName })}
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
        {(() => {
            const DriverInstallDialog = getComponent('driver-install-dialog') as React.LazyExoticComponent<FC<{driverName: string; onInstalled: () => void; onCancel: () => void}>> | undefined;
            if (!DriverInstallDialog || !missingDriver) return null;
            return (
                <Suspense fallback={null}>
                    <DriverInstallDialog
                        driverName={missingDriver}
                        onInstalled={() => {
                            setMissingDriver(null);
                            handleSubmit();
                        }}
                        onCancel={() => setMissingDriver(null)}
                    />
                </Suspense>
            );
        })()}
        </div>
    );
};

export const LoginPage: FC = () => {
    const { t } = useTranslation('pages/login');

    return (
        <Container className="justify-center items-center">
            <LoginForm />
            <div className="fixed bottom-4 left-1/2 -translate-x-1/2 text-xs text-foreground/60" data-testid="login-page-version">
                {t('version')}: {__APP_VERSION__}
            </div>
        </Container>
    );
};
