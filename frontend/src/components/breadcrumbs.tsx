/**
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

import classNames from "classnames";
import { FC, cloneElement } from "react";
import { useNavigate } from "react-router-dom";
import { IInternalRoute } from "../config/routes";
import { Icons } from "./icons";
import { BRAND_COLOR, ClassNames } from "./classes";
import { twMerge } from "tailwind-merge";

export type IBreadcrumbRoute = Omit<IInternalRoute, "component">;

type IBreadcrumbProps = {
    routes: IBreadcrumbRoute[];
    active?: IBreadcrumbRoute;
}

export const Breadcrumb: FC<IBreadcrumbProps> = ({ routes, active }) => {
    const handleNavigate = useNavigate();
    return (
        <nav className="flex py-2 px-2" aria-label="Breadcrumb">
            <ol className="flex items-center">
                {
                    routes.map((route, i) => (
                        <li key={route.name}>
                            <div className={classNames("flex items-center transition-all gap-2 group/breadcrumb dark:text-neutral-300 py-1 rounded-xl", ClassNames.Hover, {
                                "px-2": i === 0,
                                "pl-1 pr-2": i > 0,
                            })}>
                                {i > 0 && Icons.RightChevron}
                                <div onClick={() => handleNavigate(route.path)} className={twMerge(classNames("cursor-pointer text-sm font-medium text-neutral-800 hover:text-[#ca6f1e] flex items-center gap-2 dark:text-neutral-300", {
                                    [BRAND_COLOR]: active === route,
                                }))}>
                                    {
                                        i === 0 &&
                                        <div className="inline-flex items-center text-sm font-medium text-neutral-700 dark:text-neutral-300">
                                            {cloneElement(Icons.Home, {
                                                className: classNames("w-3 h-3 group-hover/breadcrumb:fill-[#ca6f1e]", {
                                                        "fill-[#ca6f1e] dark:fill-[#ca6f1e]": active === route,
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
