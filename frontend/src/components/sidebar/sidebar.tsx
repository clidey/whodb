// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.


import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { debounce } from "lodash";
import { cloneElement, FC, MouseEvent, ReactElement, useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { InternalRoutes, PublicRoutes } from "../../config/routes";
import { DatabaseType, useGetDatabaseQuery, useGetSchemaQuery, useGetVersionQuery, useLoginMutation, useLoginWithProfileMutation } from "../../generated/graphql";
import { AuthActions, LocalLoginProfile } from "../../store/auth";
import { DatabaseActions } from "../../store/database";
import { notify } from "../../store/function";
import { useAppSelector } from "../../store/hooks";
import { createStub, getDatabaseStorageUnitLabel, isNoSQL } from "../../utils/functions";
import { AnimatedButton } from "../button";
import { BRAND_COLOR, BRAND_COLOR_BG } from "../classes";
import { createDropdownItem, Dropdown, IDropdownItem } from "../dropdown";
import { Icons } from "../icons";
import { Loading } from "../loading";
import logoImage from "../../../public/images/logo.png";


type IRoute = {
    icon?: React.ReactElement;
    name: string;
    path: string;
}

type IRouteProps = {
    title: string;
    icon: React.ReactElement;
    path?: string;
    routes?: IRoute[];
    collapse?: boolean;
};

export const SideMenu: FC<IRouteProps> = (props) => {
    const navigate = useNavigate();
    const [hover, setHover] = useState(false);
    const status = hover ? "show" : "hide";
    const pathname = useLocation().pathname;

    const handleMouseEnter = useMemo(() => {
        return debounce(() => setHover(true));
    }, []);

    const handleMouseLeave = useMemo(() => {
        return debounce(() => setHover(false));
    }, []);

    const handleClick = useCallback(() => {
        if (props.path != null) {
            navigate(props.path);
        }
    }, [navigate, props.path]);

    return <div className={classNames("flex items-center", {
        "justify-center": props.collapse,
    })}  onMouseEnter={handleMouseEnter} onMouseOver={handleMouseEnter} onMouseLeave={handleMouseLeave}>
        <AnimatePresence mode="sync">
            <div className={twMerge(classNames("cursor-default text-md inline-flex gap-2 transition-all hover:gap-2 relative w-full py-4 rounded-md hover:bg-gray-100 dark:border-white/5 dark:hover:bg-white/15", {
                "cursor-pointer": props.path != null,
                "pl-4": !props.collapse,
                "pl-2": props.collapse,
            }))} onClick={handleClick}>
                {pathname === props.path && <motion.div layoutId="indicator" className={classNames("w-[5px] h-full absolute top-0 right-0 rounded-3xl", BRAND_COLOR_BG)} />}
                {cloneElement(props.icon, {
                    className: classNames("transition-all dark:stroke-white", {
                        "w-4 h-4": !props.collapse,
                        "w-6 h-6 hover:scale-110 ml-1": props.collapse,
                    })
                })}
                <span className="dark:text-neutral-300">
                    {!props.collapse  && props.title}
                </span>
                {
                    props.routes != null &&
                    <motion.div className="absolute z-40 divide-y rounded-lg shadow-lg min-w-[250px] bg-white left-[100%] -top-[20px] border border-gray-200" variants={{
                        hide: {
                            scale: 0.9,
                            opacity: 0,
                            x: 10,
                            transition: {
                                duration: 0.1,
                            },
                            transitionEnd: {
                                display: "none",
                            }
                        },
                        show: {
                            scale: 1,
                            opacity: 100,
                            x: 0,
                            display: "flex",
                        }
                    }} initial={status} animate={status}>
                        <ul className="py-2 px-2 text-sm flex flex-col justify-center w-full">
                            {props.routes.map(route => (
                                <Link key={route.name} className="flex items-center gap-1 transition-all hover:gap-2 hover:bg-gray-100 w-full rounded-md pl-2 py-2" to={route.path}>
                                    {route.icon && cloneElement(route.icon, {
                                        className: "w-4 h-4"
                                    })}
                                    {route.name}
                                </Link>
                            ))}
                        </ul>
                    </motion.div>
                }
            </div>
        </AnimatePresence>
    </div>
}

function getDropdownLoginProfileItem(profile: LocalLoginProfile): IDropdownItem {
    const icon = (Icons.Logos as Record<string, ReactElement>)[profile.Type];
    if (profile.Saved) {
        return {
            id: profile.Id,
            label: profile.Id,
            icon,
        }
    }
    if (profile.Type === DatabaseType.MongoDb) {
        return {
            id: profile.Id,
            label: `${profile.Hostname} - ${profile.Username} [${profile.Type}]`,
            icon,
        }
    }
    if (profile.Type === DatabaseType.Sqlite3) {
        return {
            id: profile.Id,
            label: `${profile.Database} [${profile.Type}]`,
            icon,
        }
    }
    return {
        id: profile.Id,
        label: `${profile.Hostname} - ${profile.Database} [${profile.Type}]`,
        icon,
    };
}

export const DATABASES_THAT_DONT_SUPPORT_SCRATCH_PAD = [DatabaseType.MongoDb, DatabaseType.Redis, DatabaseType.ElasticSearch];
const DATABASES_THAT_DONT_SUPPORT_SCHEMA = [DatabaseType.Sqlite3, DatabaseType.Redis, DatabaseType.ElasticSearch];

export const Sidebar: FC = () => {
    const [collapsed, setCollapsed] = useState(false);
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
            if (current == null) {
                return;
            }
            if (schema === "") {
                if (([DatabaseType.MySql, DatabaseType.MariaDb].includes(current.Type as DatabaseType)) && data.Schema.includes(current.Database)) {
                    dispatch(DatabaseActions.setSchema(current.Database));
                    return;   
                }
                dispatch(DatabaseActions.setSchema(data.Schema[0] ?? ""));
            }
        },
        skip: current == null || DATABASES_THAT_DONT_SUPPORT_SCHEMA.includes(current?.Type as DatabaseType),
    });
    const { data: version } = useGetVersionQuery();
    const [login, ] = useLoginMutation();
    const [loginWithProfile, ] = useLoginWithProfileMutation();
    const navigate = useNavigate();

    const handleProfileChange = useCallback((item: IDropdownItem, database?: string) => {
        const selectedProfile = profiles.find(profile => profile.Id === item.id);
        if (selectedProfile == null) {
            return;
        }
        if (selectedProfile.Saved) {
            return loginWithProfile({
                variables: {
                    profile: {
                        Id: item.id,
                        Type: selectedProfile.Type as DatabaseType,
                        Database: database ?? current?.Database,
                    },
                },
                onCompleted(status) {
                    if (status.LoginWithProfile.Status) {
                        dispatch(DatabaseActions.setSchema(""));
                        dispatch(AuthActions.switch({ id: item.id }));
                        navigate(InternalRoutes.Dashboard.StorageUnit.path);
                        if (!DATABASES_THAT_DONT_SUPPORT_SCHEMA.includes(current?.Type as DatabaseType)) {
                            getSchemas();
                        }
                    }
                },
                onError(error) {
                    notify(`Error signing you in: ${error.message}`, "error")
                },
            })
        }
        login({
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
                    dispatch(DatabaseActions.setSchema(""));
                    dispatch(AuthActions.switch({ id: selectedProfile.Id }));
                    navigate(InternalRoutes.Dashboard.StorageUnit.path);
                    getSchemas();
                }
            },
            onError(error) {
                notify(`Error signing you in: ${error.message}`, "error")
            },
        });
    }, [current?.Database, current?.Type, dispatch, getSchemas, login, loginWithProfile, navigate, profiles]);

    const handleDatabaseChange = useCallback((item: IDropdownItem) => {
        if (current?.Id == null) {
            return;
        }
        if (pathname !== InternalRoutes.Graph.path && pathname !== InternalRoutes.Dashboard.StorageUnit.path) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
        dispatch(AuthActions.setLoginProfileDatabase({ id: current?.Id, database: item.id }));
        handleProfileChange(createDropdownItem(current.Id), item.id);
    }, [current, dispatch, handleProfileChange, navigate, pathname]);

    const handleSchemaChange = useCallback((item: IDropdownItem) => {
        if (pathname !== InternalRoutes.Graph.path && pathname !== InternalRoutes.Dashboard.StorageUnit.path) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
        dispatch(DatabaseActions.setSchema(item.id));
    }, [dispatch, navigate, pathname]);

    const loading = useMemo(() => {
        return availableDatabasesLoading || availableSchemasLoading;
    }, [availableDatabasesLoading, availableSchemasLoading]);

    const sidebarRoutes: IRouteProps[] = useMemo(() => {
        if (current == null) {
            return [];
        }
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
                title: "Houdini",
                icon: Icons.Chat,
                path: InternalRoutes.Chat.path,
            });
        }
        if (!DATABASES_THAT_DONT_SUPPORT_SCRATCH_PAD.includes(current.Type as DatabaseType)) {
            routes.push({
                title: "Scratchpad",
                icon: Icons.Console,
                path: InternalRoutes.RawExecute.path,
            });
        }
        return routes;
    }, [current]);

    const handleCollapseToggle = useCallback(() => {
        setCollapsed(c => !c);
    }, []);

    const handleNavigateToLogin = useCallback(() => {
        navigate(PublicRoutes.Login.path);
    }, [navigate]);

    const routes = useMemo(() => {
        return sidebarRoutes.map(route => (
            <SideMenu key={`sidebar-routes-${createStub(route.title)}`} collapse={collapsed} title={route.title} icon={route.icon}
                routes={route.routes} path={route.path} />
        ));
    }, [collapsed, sidebarRoutes]);

    const loginItems: IDropdownItem[] = useMemo(() => {
        return profiles.map(profile => getDropdownLoginProfileItem(profile));
    }, [profiles]);

    const handleMenuLogout = useCallback((e: MouseEvent, item: IDropdownItem) => {
        e.stopPropagation();
        const selectedProfile = profiles.find(profile => profile.Id === item.id);
        if (selectedProfile == null) {
            return;
        }
        if (selectedProfile.Id === current?.Id) {
            return navigate(InternalRoutes.Logout.path);
        }
        dispatch(AuthActions.remove({ id: selectedProfile.Id }));
    }, [current?.Id, dispatch, navigate, profiles]);

    const currentProfile = useMemo(() => {
        if (current == null) {
            return;
        }
        const icon = (Icons.Logos as Record<string, ReactElement>)[current.Type];
        if (current.Saved) {
            return {
                id: current.Id,
                label: current.Id,
                icon,
            }
        }
        if (current.Type === DatabaseType.Redis) {
            return {
                id: current.Id,
                label: current.Hostname,
                icon,
            }
        }
        if (current.Type === DatabaseType.Sqlite3) {
            return {
                id: current.Id,
                label: current.Database,
                icon,
            }
        }
        return {
            id: current.Id,
            label: `${current.Hostname} [${current.Username}]`,
            icon,
        }
    }, [current]);

    const schemasDropdownItems = useMemo(() => {
        return availableSchemas?.Schema.map(schema => createDropdownItem(schema)) ?? [];
    }, [availableSchemas?.Schema]);

    const animate = collapsed ? "hide" : "show";

    return (
        <div className={
            classNames("h-[100vh] flex flex-col gap-4 shadow-md relative transition-all duration-500 dark:bg-white/10 dark:shadow-neutral-100/5", {
                "w-[50px] py-20": collapsed,
                "w-[300px] px-10 py-20": !collapsed,
            })}>
                <motion.div className="flex flex-col gap-4" variants={{
                    show: {
                        opacity: 1,
                        transition: {
                            delay: 0.3,
                        }
                    },
                    hide: {
                        opacity: 0,
                        transition: {
                            duration: 0.1,
                        }
                    }
                }} animate={animate}>
                <div className="flex gap-2">
                    <img src={logoImage} alt="clidey logo" className="w-auto h-8" />
                    <span className={classNames(BRAND_COLOR, "text-2xl")}>WhoDB</span>
                </div>
            </motion.div>
            <motion.div className={classNames("absolute top-4 cursor-pointer transition-all dark:text-neutral-300", {
                "right-2 hover:right-3": !collapsed,
                "right-3 hover:right-2": collapsed,
            })} initial="show" variants={{
                show: {
                    rotate: "180deg",
                },
                hide: {
                    rotate: "0deg",
                }
            }} animate={animate} onClick={handleCollapseToggle} transition={{
                duration: 0.1,
            }}>
                {Icons.DoubleRightArrow}
            </motion.div>
            {
                loading
                ? <Loading />
                : <div className="flex flex-col justify-center mt-[10vh] grow">
                        <div className="flex flex-col">
                            <div className="flex flex-col mb-[10vh] gap-4 ml-4">
                                <div className={classNames("flex gap-2 items-center", {
                                    "hidden": collapsed,
                                })}>
                                    <div className="text-sm text-gray-600 dark:text-neutral-300 mr-2.5">Profile:</div>
                                    {
                                        currentProfile != null &&
                                        <Dropdown className="w-[140px]" items={loginItems} value={currentProfile}
                                                  onChange={handleProfileChange}
                                                  defaultItem={{
                                                      label: "Add another profile",
                                                      icon: cloneElement(Icons.Add, {
                                                          className: "w-6 h-6 stroke-green-800 dark:stroke-green-400",
                                                      }),
                                                  }} defaultItemClassName="text-green-800"
                                                  onDefaultItemClick={handleNavigateToLogin}
                                                  action={<AnimatedButton icon={Icons.Logout} label="Logout"
                                                                          onClick={handleMenuLogout}/>}/>
                                    }
                                </div>
                                {
                                    availableDatabases != null && current != null &&
                                    <div className={classNames("flex gap-2 items-center w-full", {
                                        "opacity-0 pointer-events-none": collapsed || (current.Type !== DatabaseType.Redis && isNoSQL(current?.Type as DatabaseType)),
                                    })}>
                                        <div className="text-sm text-gray-600 dark:text-neutral-300">Database:</div>
                                        <Dropdown className="w-[140px]" value={createDropdownItem(current!.Database)}
                                                  items={availableDatabases.Database.map(database => createDropdownItem(database))}
                                                  onChange={handleDatabaseChange}
                                                  noItemsLabel="No available database found"/>
                                    </div>
                                }
                                {
                                    schemasDropdownItems.length > 0 &&
                                    <div className={classNames("flex gap-2 items-center w-full", {
                                        "opacity-0 pointer-events-none": pathname === InternalRoutes.RawExecute.path || collapsed || DATABASES_THAT_DONT_SUPPORT_SCHEMA.includes(current?.Type as DatabaseType),
                                    })}>
                                        <div className="text-sm text-gray-600 dark:text-neutral-300">Schema:</div>
                                        <Dropdown className="w-[140px]" value={createDropdownItem(schema)}
                                                  items={schemasDropdownItems} onChange={handleSchemaChange}
                                                  noItemsLabel="No schema found"/>
                                    </div>
                                }
                            </div>
                            {routes}
                        </div>
                        <div className="grow"/>
                        <div className="flex flex-col">
                            <SideMenu collapse={collapsed} title="Contact Us" icon={Icons.QuestionMark}
                                      path={InternalRoutes.ContactUs.path}/>
                        </div>
                        <div className="flex flex-col gap-8">
                            <SideMenu collapse={collapsed} title="Settings" icon={Icons.Settings}
                                      path={InternalRoutes.Settings.path}/>
                        </div>
                        <div className="flex flex-col gap-8">
                            <SideMenu collapse={collapsed} title="Logout" icon={Icons.Logout}
                                      path={InternalRoutes.Logout.path}/>
                        </div>
                    </div>
            }
            <div className="absolute right-8 bottom-8 text-sm text-gray-300 hover:text-gray-600 dark:text-neutral-600 self-end dark:hover:text-neutral-300 transition-all">{version?.Version}</div>
        </div>
    )
}