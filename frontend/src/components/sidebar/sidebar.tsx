
import { useLazyQuery, useMutation } from "@apollo/client";
import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { debounce } from "lodash";
import { FC, MouseEvent, cloneElement, useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { InternalRoutes, PublicRoutes } from "../../config/routes";
import { DatabaseType, GetSchemaDocument, GetSchemaQuery, GetSchemaQueryVariables, LoginDocument, LoginMutation, LoginMutationVariables } from "../../generated/graphql";
import { AuthActions, LoginProfile } from "../../store/auth";
import { DatabaseActions } from "../../store/database";
import { notify } from "../../store/function";
import { useAppSelector } from "../../store/hooks";
import { createStub, isNoSQL } from "../../utils/functions";
import { AnimatedButton } from "../button";
import { BRAND_COLOR } from "../classes";
import { Dropdown, IDropdownItem } from "../dropdown";
import { Icons } from "../icons";
import { Loading } from "../loading";

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
            <div className={twMerge(classNames("cursor-default text-md inline-flex gap-2 transition-all hover:gap-2 relative w-full py-4 rounded-md hover:bg-gray-100", {
                "cursor-pointer": props.path != null,
                "pl-4": !props.collapse,
                "pl-2": props.collapse,
            }))} onClick={handleClick}>
                {cloneElement(props.icon, {
                    className: classNames("transition-all", {
                        "w-4 h-4": !props.collapse,
                        "w-6 h-6 hover:scale-110 ml-1": props.collapse,
                    })
                })}
                {!props.collapse  && props.title}
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

function getDropdownLoginProfileItem(profile: LoginProfile): IDropdownItem {
    if (profile.Type === DatabaseType.MongoDb) {
        return {
            id: profile.id,
            label: `${profile.Hostname} - ${profile.Username} [${profile.Type}]`,
        }
    }
    if (profile.Type === DatabaseType.Sqlite3) {
        return {
            id: profile.id,
            label: `${profile.Database} [${profile.Type}]`,
        }
    }
    return {
        id: profile.id,
        label: `${profile.Hostname} - ${profile.Database} [${profile.Type}]`,
    };
}

export const Sidebar: FC = () => {
    const [collapsed, setCollapsed] = useState(false);
    const schema = useAppSelector(state => state.database.schema);
    const dispatch = useDispatch();
    const pathname = useLocation().pathname;
    const current = useAppSelector(state => state.auth.current);
    const profiles = useAppSelector(state => state.auth.profiles);
    const [getSchema,{ data, loading }] = useLazyQuery<GetSchemaQuery, GetSchemaQueryVariables>(GetSchemaDocument, {
        onError() {
            notify("Unable to connect to database", "error");
        }
    });
    const [login, ] = useMutation<LoginMutation, LoginMutationVariables>(LoginDocument);
    const navigate = useNavigate();

    const handleSchemaChange = useCallback((item: IDropdownItem) => {
        if (pathname !== InternalRoutes.Graph.path && pathname !== InternalRoutes.Dashboard.StorageUnit.path) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
        dispatch(DatabaseActions.setSchema(item.id));
    }, [dispatch, navigate, pathname]);

    useEffect(() => {
        if (current == null || [DatabaseType.Sqlite3, DatabaseType.Redis].includes(current?.Type as DatabaseType)) {
            return;
        }
        if (schema === "") {
            getSchema({
                variables: {
                    type: current?.Type as DatabaseType,
                },
                onCompleted(data) {
                    if (data.Schema.length > 0) dispatch(DatabaseActions.setSchema(data.Schema[0]));
                },
            });
            return;
        }
        getSchema({
            variables: {
                type: current?.Type as DatabaseType,
            },
        });
    }, [current, dispatch, getSchema, schema]);

    const sidebarRoutes: IRouteProps[] = useMemo(() => {
        if (current == null) {
            return [];
        }
        const routes = [
            {
                title: isNoSQL(current.Type) ? "Collections" : "Tables",
                icon: Icons.Tables,
                path: InternalRoutes.Dashboard.StorageUnit.path,
            },
            {
                title: "Graph",
                icon: Icons.GraphLayout,
                path: InternalRoutes.Graph.path,
            },
        ];
        if (current.Type !== DatabaseType.MongoDb && current.Type !== DatabaseType.Redis) {
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

    const handleProfileChange = useCallback((item: IDropdownItem) => {
        const selectedProfile = profiles.find(profile => profile.id === item.id);
        if (selectedProfile == null) {
            return;
        }
        login({
            variables: {
                credentails: {
                    Type: selectedProfile.Type,
                    Database: selectedProfile.Database,
                    Hostname: selectedProfile.Hostname,
                    Password: selectedProfile.Password,
                    Username: selectedProfile.Username,
                },
            },
            onCompleted(status) {
                if (status.Login.Status) {
                    dispatch(DatabaseActions.setSchema(""));
                    dispatch(AuthActions.switch({ id: selectedProfile.id }));
                    navigate(InternalRoutes.Dashboard.StorageUnit.path);
                }
            },
            onError(error) {
                notify(`Error signing you in: ${error.message}`, "error")
            },
        })
    }, [dispatch, login, navigate, profiles]);

    const handleNavigateToLogin = useCallback(() => {
        navigate(PublicRoutes.Login.path);
    }, [navigate]);

    const routes = useMemo(() => {
        return sidebarRoutes.map(route => (
            <SideMenu key={`sidebar-routes-${createStub(route.title)}`} collapse={collapsed} title={route.title} icon={route.icon} routes={route.routes} path={route.path} />
        ));
    }, [collapsed, sidebarRoutes]);

    const loginItems: IDropdownItem[] = useMemo(() => {
        return profiles.map(profile => getDropdownLoginProfileItem(profile));
    }, [profiles]);

    const handleMenuLogout = useCallback((e: MouseEvent, item: IDropdownItem) => {
        e.stopPropagation();
        const selectedProfile = profiles.find(profile => profile.id === item.id);
        if (selectedProfile == null) {
            return;
        }
        if (selectedProfile.id === current?.id) {
            return navigate(InternalRoutes.Logout.path);
        }
        dispatch(AuthActions.remove({ id: selectedProfile.id }));
    }, [current?.id, dispatch, navigate, profiles]);

    const currentProfile = useMemo(() => {
        if (current == null) {
            return;
        }
        if (current.Type === DatabaseType.Redis) {
            return {
                id: current.id,
                label: current.Hostname,
            }
        }
        if (current.Type === DatabaseType.Sqlite3) {
            return {
                id: current.id,
                label: current.Database,
            }
        }
        return {
            id: current.id,
            label: `${current.Hostname} [${current.Username}]`,
        }
    }, [current]);

    const animate = collapsed ? "hide" : "show";

    return (
        <div className={
            classNames("h-[100vh] flex flex-col gap-4 shadow-md relative transition-all duration-500", {
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
                    <img src="https://clidey.com/logo.svg" alt="clidey logo" className="w-8 h-8" />
                    <span className={classNames(BRAND_COLOR, "text-2xl")}>WhoDB</span>
                </div>
            </motion.div>
            <motion.div className={classNames("absolute top-4 cursor-pointer transition-all", {
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
                :  <div className="flex flex-col justify-center mt-[10vh] grow">
                        <AnimatePresence mode="wait">
                            <div className="flex flex-col">
                                <div className="flex flex-col mb-[10vh] gap-4 ml-4">
                                    <div className={classNames("flex gap-2 items-center", {
                                        "hidden": collapsed,
                                    })}>
                                        <div className="text-sm text-gray-600 mr-2.5">Profile:</div>
                                        {
                                            current != null &&
                                            <Dropdown className="w-[140px]" items={loginItems} value={currentProfile} onChange={handleProfileChange}
                                                defaultItem={{
                                                    label: "Add another profile",
                                                    icon: cloneElement(Icons.Add, {
                                                        className: "w-6 h-6 stroke-green-800",
                                                    }),
                                                }} defaultItemClassName="text-green-800" onDefaultItemClick={handleNavigateToLogin} 
                                                action={<AnimatedButton icon={Icons.Logout} label="Logout" onClick={handleMenuLogout} /> }/>
                                        }
                                    </div>
                                    {
                                        data != null &&
                                        <div className={classNames("flex gap-2 items-center w-full", {
                                            "hidden": pathname === InternalRoutes.RawExecute.path || collapsed || [DatabaseType.Sqlite3, DatabaseType.Redis].includes(current?.Type as DatabaseType),
                                        })}>
                                            <div className="text-sm text-gray-600">Schema:</div>
                                            <Dropdown className="w-[140px]" value={{ id: schema, label: schema }} items={data.Schema.map(schema => ({ id: schema, label: schema }))} onChange={handleSchemaChange}
                                                noItemsLabel="No schema found"/>
                                        </div>
                                    }
                                </div>
                                {routes}
                            </div>
                            <div className="grow" />
                            <div className="flex flex-col">
                                <SideMenu collapse={collapsed} title="Logout" icon={Icons.Logout} path={InternalRoutes.Logout.path} />
                            </div>
                        </AnimatePresence>
                    </div>
            }
        </div>
    )
}