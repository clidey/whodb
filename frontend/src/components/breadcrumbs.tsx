import classNames from "classnames";
import { FC } from "react";
import { useNavigate } from "react-router-dom";
import { IInternalRoute, InternalRoutes } from "../config/routes";
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
            <ol className="inline-flex items-center space-x-1 md:space-x-2 rtl:space-x-reverse mb-4">
                <li className="inline-flex items-center">
                    <div onClick={() => handleNavigate(InternalRoutes.Dashboard.StorageUnit.path)} className={classNames("cursor-pointer inline-flex items-center text-sm font-medium text-gray-700 hover:text-teal-500 transition-all gap-0 hover:gap-1", {
                        "text-teal-800": routes.length === 0,
                    })}>
                        {Icons.Home}
                        Home
                    </div>
                </li>
                {
                    routes.map(route => (
                        <li>
                            <div className="flex items-center transition-all gap-2 hover:gap-3">
                                {Icons.RightChevron}
                                <div onClick={() => handleNavigate(route.path)} className={classNames("cursor-pointer text-sm font-medium text-gray-700 hover:text-teal-500", {
                                    "text-teal-800": active === route,
                                })}>
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