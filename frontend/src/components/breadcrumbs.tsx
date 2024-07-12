import classNames from "classnames";
import { FC, cloneElement } from "react";
import { useNavigate } from "react-router-dom";
import { IInternalRoute } from "../config/routes";
import { Icons } from "./icons";

export type IBreadcrumbRoute = Omit<IInternalRoute, "component">;

type IBreadcrumbProps = {
    routes: IBreadcrumbRoute[];
    active?: IBreadcrumbRoute;
}

export const Breadcrumb: FC<IBreadcrumbProps> = ({ routes, active }) => {
    const handleNavigate = useNavigate();
    return (
        <nav className="flex" aria-label="Breadcrumb">
            <ol className="inline-flex items-center space-x-1 md:space-x-2 rtl:space-x-reverse">
                {
                    routes.map((route, i) => (
                        <li>
                            <div className="flex items-center transition-all gap-2 hover:gap-3 group/breadcrumb dark:text-neutral-300">
                                {i > 0 && Icons.RightChevron}
                                <div onClick={() => handleNavigate(route.path)} className={classNames("cursor-pointer text-sm font-medium text-gray-700 hover:text-teal-500 flex items-center gap-2 hover:gap-3 transition-all dark:text-neutral-300", {
                                    "text-teal-800 dark:text-teal-500": active === route,
                                })}>
                                    {
                                        i === 0 &&
                                        <div className="inline-flex items-center text-sm font-medium text-gray-700 dark:text-neutral-300">
                                            {cloneElement(Icons.Home, {
                                                className: classNames("w-3 h-3 group-hover/breadcrumb:fill-teal-500", {
                                                        "fill-teal-800 dark:fill-teal-500": active === route,
                                                    })
                                                })
                                            }
                                        </div>
                                    }
                                    {route.name}
                                </div>
                            </div>
                        </li>
                    ))
                }
            </ol>
        </nav>
    )
}