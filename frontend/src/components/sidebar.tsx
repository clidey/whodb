
import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { debounce } from "lodash";
import { FC, cloneElement, useCallback, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { InternalRoutes } from "../config/routes";
import { createStub } from "../utils/functions";
import { BRAND_COLOR } from "./classes";
import { Icons } from "./icons";
import { twMerge } from "tailwind-merge";
import { useNavigate } from "react-router-dom";

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
        <AnimatePresence mode="wait">
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
                                <Link className="flex items-center gap-1 transition-all hover:gap-2 hover:bg-gray-100 w-full rounded-md pl-2 py-2" key={route.path} to={route.path}>
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

export const Sidebar: FC = () => {
    const [collapsed, setCollapsed] = useState(false);

    const sidebarRoutes: IRouteProps[] = useMemo(() => {
        return [
            {
                title: "Tables",
                icon: Icons.Tables,
                path: InternalRoutes.Dashboard.StorageUnit.path,
            },
            {
                title: "Graph",
                icon: Icons.GraphLayout,
                path: InternalRoutes.Graph.path,
            },
        ];
    }, []);

    const handleCollapseToggle = useCallback(() => {
        setCollapsed(c => !c);
    }, []);

    const routes = useMemo(() => {
        return sidebarRoutes.map(route => (
            <SideMenu key={`sidebar-routes-${createStub(route.title)}`} collapse={collapsed} title={route.title} icon={route.icon} routes={route.routes} path={route.path} />
        ));
    }, [collapsed, sidebarRoutes]);

    const animate = collapsed ? "hide" : "show";

    return (
        <div className={
            classNames("h-[100vh] flex flex-col gap-4 shadow-md relative transition-all duration-500", {
                "w-[50px] py-20": collapsed,
                "w-[350px] p-20": !collapsed,
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
            <div className="flex flex-col justify-center mt-[20vh] grow max-w-[150px]">
                <AnimatePresence mode="wait">
                    <div className="flex flex-col">
                        {routes}
                    </div>
                    <div className="grow" />
                    <SideMenu collapse={collapsed} title="Logout" icon={Icons.Logout} path={InternalRoutes.Logout.path} />
                </AnimatePresence>
            </div>
        </div>
    )
}