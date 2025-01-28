import classNames from "classnames";
import { entries } from "lodash";
import { cloneElement, FC, ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { AnimatedButton } from "../../components/button";
import { BRAND_COLOR } from "../../components/classes";
import { createDropdownItem, DropdownWithLabel, IDropdownItem } from "../../components/dropdown";
import { Icons } from "../../components/icons";
import { InputWithlabel } from "../../components/input";
import { Loading } from "../../components/loading";
import { Container } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, LoginCredentials, useGetDatabaseLazyQuery, useGetProfilesQuery, useLoginMutation, useLoginWithProfileMutation } from '../../generated/graphql';
import { AuthActions } from "../../store/auth";
import { DatabaseActions } from "../../store/database";
import { notify } from "../../store/function";
import { useAppDispatch } from "../../store/hooks";

const databaseTypeDropdownItems: IDropdownItem<Record<string, string>>[] = [
    {
        id: "Postgres",
        label: "Postgres",
        icon: Icons.Logos.Postgres,
        extra: {"Port": "5432"},
    },
    {
        id: "MySQL",
        label: "MySQL",
        icon: Icons.Logos.MySQL,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "Local", "Allow clear text passwords": "0"},
    },
    {
        id: "MariaDB",
        label: "MariaDB",
        icon: Icons.Logos.MariaDB,
        extra: {"Port": "3306", "Parse Time": "True", "Loc": "Local", "Allow clear text passwords": "0"},
    },
    {
        id: "Sqlite3",
        label: "Sqlite3",
        icon: Icons.Logos.Sqlite3,
    },
    {
        id: "MongoDB",
        label: "MongoDB",
        icon: Icons.Logos.MongoDB,
        extra: {"Port": "27017", "URL Params": "?", "DNS Enabled": "false"},
    },
    {
        id: "Redis",
        label: "Redis",
        icon: Icons.Logos.Redis,
        extra: {"Port": "6379"},
    },
    {
        id: "ElasticSearch",
        label: "ElasticSearch",
        icon: Icons.Logos.ElasticSearch,
        extra: {"Port": "9200", "SSL Mode": "disable"},
    },
    {
        id: "ClickHouse",
        label: "ClickHouse",
        icon: Icons.Logos.ClickHouse,
        extra: {
            "Port": "9000",
            "SSL mode": "disable",
            "HTTP Protocol": "disable",
            "Readonly": "disable",
            "Debug": "disable"
        }
    }
]

export const LoginPage: FC = () => {
    const dispatch = useAppDispatch();
    const navigate = useNavigate();
    
    const [login, { loading: loginLoading }] = useLoginMutation();
    const [loginWithProfile, { loading: loginWithProfileLoading }] = useLoginWithProfileMutation();
    const [getDatabases, { loading: databasesLoading, data: foundDatabases }] = useGetDatabaseLazyQuery();
    const { loading: profilesLoading, data: profiles } = useGetProfilesQuery();
    const [searchParams, ] = useSearchParams();
    
    const [databaseType, setDatabaseType] = useState<IDropdownItem>(databaseTypeDropdownItems[0]);
    const [hostName, setHostName] = useState("");
    const [database, setDatabase] = useState("");
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string>();
    const [advancedForm, setAdvancedForm] = useState<Record<string, string>>(databaseType.extra);
    const [showAdvanced, setShowAdvanced] = useState(false);
    const [selectedAvailableProfile, setSelectedAvailableProfile] = useState<IDropdownItem>();

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

        const credentials: LoginCredentials = {
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
                    dispatch(AuthActions.login(credentials));
                    navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    return notify("Login successfully", "success");
                }
                return notify("Login failed", "error");
            },
            onError(error) {
                return notify(`Login failed: ${error.message}`, "error");
            }
        });
    }, [databaseType.id, hostName, database, username, password, advancedForm, login, dispatch, navigate]);

    const handleLoginWithProfileSubmit = useCallback(() => {
        if (selectedAvailableProfile == null) {
            return setError("Select a profile");
        }
        setError(undefined);

        const profile = profiles?.Profiles.find(p => p.Id === selectedAvailableProfile.id);

        loginWithProfile({
            variables: {
                profile: {
                    Id:  selectedAvailableProfile.id,
                    Type: profile?.Type as DatabaseType,
                },
            },
            onCompleted(data) {
                if (data.LoginWithProfile.Status) {
                    dispatch(AuthActions.login({
                        Type: profile?.Type as DatabaseType,
                        Id: selectedAvailableProfile.id,
                        Database: profile?.Database ?? "",
                        Hostname: "",
                        Password: "",
                        Username: "",
                        Saved: true,
                    }));
                    navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    return notify("Login successfully", "success");
                }
                return notify("Login failed", "error");
            },
            onError(error) {
                return notify(`Login failed: ${error.message}`, "error");
            }
        });
    }, [dispatch, loginWithProfile, navigate, profiles?.Profiles, selectedAvailableProfile]);

    const handleDatabaseTypeChange = useCallback((item: IDropdownItem) => {
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
        setAdvancedForm(item.extra);
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

    const handleAvailableProfileChange = useCallback((item: IDropdownItem) => {
        setSelectedAvailableProfile(item);
    }, []);

    useEffect(() => {
        dispatch(DatabaseActions.setSchema(""));
    }, [dispatch]);

    useEffect(() => {
        if (searchParams.size > 0) {
            if (searchParams.has("type")) {
                const databaseType = searchParams.get("type")!;
                setDatabaseType(databaseTypeDropdownItems.find(item => item.id === databaseType) ?? databaseTypeDropdownItems[0]);
            }
            if (searchParams.has("host")) setHostName(searchParams.get("host")!);
            if (searchParams.has("username")) setUsername(searchParams.get("username")!);
            if (searchParams.has("password")) setPassword(searchParams.get("password")!);
            if (searchParams.has("database")) setDatabase(searchParams.get("database")!);
        }
    }, [searchParams]);

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
                        notify("We could not extract all required details (host, username, password, or database) from this URL. Please enter the information manually.", "warning");
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
                    notify("We could not extract all required details (host, username, password, or database) from this URL. Please enter the information manually.", "warning");
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
            return <>
                <DropdownWithLabel label="Database" items={foundDatabases?.Database?.map(database => ({
                    id: database,
                    label: database,
                    icon: Icons.Database,
                })) ?? []} loading={databasesLoading} noItemsLabel="Not available. Mount SQLite file in /db/" fullWidth={true} value={{
                    id: database,
                    label: database,
                    icon: Icons.Database,
                }} onChange={(item) => setDatabase(item.id)} />
            </>
        }
        return <>
            <InputWithlabel label={databaseType.id === DatabaseType.MongoDb || databaseType.id === DatabaseType.Postgres ? "Host Name (or paste Connection URL)" : "Host Name"} value={hostName} setValue={handleHostNameChange} />
            { databaseType.id !== DatabaseType.Redis && <InputWithlabel label="Username" value={username} setValue={setUsername} /> }
            <InputWithlabel label="Password" value={password} setValue={setPassword} type="password" />
            { (databaseType.id !== DatabaseType.MongoDb && databaseType.id !== DatabaseType.Redis && databaseType.id !== DatabaseType.ElasticSearch)  && <InputWithlabel label="Database" value={database} setValue={setDatabase} /> }
        </>
    }, [database, databaseType.id, databasesLoading, foundDatabases?.Database, handleHostNameChange, hostName, password, username]);

    const availableProfiles = useMemo(() => {
        return profiles?.Profiles.map(profile => createDropdownItem(profile.Alias ?? profile.Id, (Icons.Logos as Record<string, ReactElement>)[profile.Type])) ?? [];
    }, [profiles?.Profiles])

    if (loading || profilesLoading)  {
        return <Container>
                <div className="flex flex-col justify-center items-center gap-4 w-full">
                    <div>
                        <Loading hideText={true} />
                    </div>
                    <div className="text-neutral-800 dark:text-neutral-300">
                        Logging in
                    </div>
                </div>
        </Container>
    }

    return (
        <Container className="justify-center items-center">
            <div className="w-fit h-fit">
                <div className="flex flex-col justify-between grow gap-4">
                    <div className="flex grow">
                        <div className="flex flex-col gap-4 grow w-[350px]">
                            <div className="flex justify-between">
                                <div className="text-lg text-gray-600 flex gap-2 items-center">
                                    <div className="h-[40px] w-[40px] rounded-xl flex justify-center items-center bg-teal-500">
                                        {cloneElement(Icons.Lock, {
                                            className: "w-6 h-6 stroke-white",
                                        })}
                                    </div>
                                    <span className={BRAND_COLOR}>WhoDB</span> <span className="dark:text-neutral-300">Login</span>
                                </div>
                                <div className="text-red-500 text-xs flex items-center">
                                    {error}
                                </div>
                            </div>
                            <div className="flex flex-col grow justify-center gap-1">
                                <DropdownWithLabel fullWidth label="Database Type" value={databaseType} onChange={handleDatabaseTypeChange} items={databaseTypeDropdownItems} />
                                {fields}
                            </div>
                        </div>
                        {
                            (showAdvanced && advancedForm != null) &&
                            <div className="transition-all h-full overflow-hidden mt-[56px] w-[350px] ml-4 flex flex-col gap-1">
                                {entries(advancedForm).map(([key, value]) => (
                                    <InputWithlabel label={key} value={value} setValue={(newValue) => handleAdvancedForm(key, newValue)} />
                                ))}
                            </div>
                        }
                    </div>
                    <div className={classNames("flex", {
                        "justify-end": advancedForm == null,
                        "justify-between": advancedForm != null,
                    })}>
                        <AnimatedButton className={classNames({
                            "hidden": advancedForm == null,
                        })} icon={Icons.Adjustments} label={showAdvanced ? "Less Advanced" : "Advanced"} onClick={handleAdvancedToggle} />
                        <AnimatedButton icon={Icons.CheckCircle} label="Submit" onClick={handleSubmit} />
                    </div>
                </div>
                {
                    availableProfiles.length > 0 &&
                    <div className="mt-4 pt-2 border-t border-t-neutral-100/10 flex flex-col gap-2">
                        <DropdownWithLabel fullWidth label="Available profiles" value={selectedAvailableProfile} onChange={handleAvailableProfileChange}
                            items={availableProfiles} noItemsLabel="No available profiles" />
                        <AnimatedButton className="self-end" icon={Icons.CheckCircle} label="Login" onClick={handleLoginWithProfileSubmit} />
                    </div>
                }
            </div>
        </Container>
    )
}