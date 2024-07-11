import classNames from "classnames";
import { entries } from "lodash";
import { FC, cloneElement, useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { AnimatedButton } from "../../components/button";
import { BASE_CARD_CLASS, BRAND_COLOR } from "../../components/classes";
import { DropdownWithLabel, IDropdownItem } from "../../components/dropdown";
import { Icons } from "../../components/icons";
import { InputWithlabel } from "../../components/input";
import { Loading } from "../../components/loading";
import { Page } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, LoginCredentials, useGetDatabaseLazyQuery, useLoginMutation } from '../../generated/graphql';
import { AuthActions } from "../../store/auth";
import { DatabaseActions } from "../../store/database";
import { notify } from "../../store/function";
import { useAppDispatch } from "../../store/hooks";

const databaseTypeDropdownItems: IDropdownItem<Record<string, string>>[] = [
    {
        id: "Postgres",
        label: "Postgres",
        icon: Icons.Logos.Postgres,
        extra: {"Port": "5432", "SSL Mode": "disable"},
    },
    {
        id: "MySQL",
        label: "MySQL",
        icon: Icons.Logos.MySQL,
        extra: {"Port": "3306", "Charset": "utf8mb4", "Parse Time": "True", "Loc": "Local"},
    },
    {
        id: "MariaDB",
        label: "MariaDB",
        icon: Icons.Logos.MariaDB,
        extra: {"Port": "3306", "Charset": "utf8mb4", "Parse Time": "True", "Loc": "Local"},
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
        extra: {"Port": "27017", "URL Params [starts with ?]": "?"},
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
]

export const LoginPage: FC = () => {
    const dispatch = useAppDispatch();
    const navigate = useNavigate();
    
    const [login, { loading }] = useLoginMutation();
    const [getDatabases, { loading: databasesLoading, data: foundDatabases }] = useGetDatabaseLazyQuery();
    const [searchParams, ] = useSearchParams();
    
    const [databaseType, setDatabaseType] = useState<IDropdownItem>(databaseTypeDropdownItems[0]);
    const [hostName, setHostName] = useState("");
    const [database, setDatabase] = useState("");
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string>();
    const [advancedForm, setAdvancedForm] = useState<Record<string, string>>(databaseType.extra);
    const [showAdvanced, setShowAdvanced] = useState(false);

    const handleSubmit = useCallback(() => {
        if (([DatabaseType.MySql, DatabaseType.Postgres].includes(databaseType.id as DatabaseType) && (hostName.length === 0 || database.length === 0 || username.length === 0 || password.length === 0))
            || (databaseType.id === DatabaseType.Sqlite3 && database.length === 0)
            || (databaseType.id === DatabaseType.MongoDb && (hostName.length === 0 || username.length === 0))
            || (databaseType.id === DatabaseType.Redis && (hostName.length === 0))) {
            return setError(`All fields are required`);
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
        })
    }, [databaseType.id, hostName, database, username, password, advancedForm, login, dispatch, navigate]);

    const handleDatabaseTypeChange = useCallback((item: IDropdownItem) => {
        if (item.id === DatabaseType.Sqlite3) {
            getDatabases({
                variables: {
                    type: item.id,
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
            <InputWithlabel label="Host Name" value={hostName} setValue={setHostName} />
            { databaseType.id !== DatabaseType.Redis && <InputWithlabel label="Username" value={username} setValue={setUsername} /> }
            <InputWithlabel label="Password" value={password} setValue={setPassword} type="password" />
            { (databaseType.id !== DatabaseType.MongoDb && databaseType.id !== DatabaseType.Redis && databaseType.id !== DatabaseType.ElasticSearch)  && <InputWithlabel label="Database" value={database} setValue={setDatabase} /> }
        </>
    }, [database, databaseType.id, databasesLoading, foundDatabases?.Database, hostName, password, username]);

    if (loading)  {
        return (
            <Page className="justify-center items-center">
                <div className={twMerge(BASE_CARD_CLASS, "w-[350px] h-fit flex-col gap-4 justify-center py-16")}>
                    <Loading hideText={true} />
                    <div className="text-gray-600 text-center">
                        Logging in
                    </div>
                </div>
            </Page>)
    }

    return (
        <Page className="justify-center items-center">
            <div className={twMerge(BASE_CARD_CLASS, "w-fit h-fit")}>
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
                                    <span className={BRAND_COLOR}>WhoDB</span> Login
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
                            <div className="transition-all h-full overflow-hidden mt-[56px] w-[350px] ml-4">
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
            </div>
        </Page>
    )
}