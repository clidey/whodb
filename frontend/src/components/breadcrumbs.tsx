/*
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
import {FC} from "react";
import {useNavigate} from "react-router-dom";
import {IInternalRoute, InternalRoutes} from "../config/routes";
import {
  Breadcrumb as UxBreadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@clidey/ux";
import {ChevronRightIcon, HomeIcon} from "./heroicons";

export type IBreadcrumbRoute = Omit<IInternalRoute, "component">;

type IBreadcrumbProps = {
  routes: IBreadcrumbRoute[];
  active?: IBreadcrumbRoute;
};

export const Breadcrumb: FC<IBreadcrumbProps> = ({ routes, active }) => {
  const navigate = useNavigate();

  return (
    <UxBreadcrumb className="py-2">
      <BreadcrumbList>
        {routes.map((route, i) => {
          const isActive = active === route;
          const isLast = i === routes.length - 1;
          return (
            <BreadcrumbItem key={route.name}>
              {i > 0 && <BreadcrumbSeparator><ChevronRightIcon className="w-4 h-4 mr-1"/></BreadcrumbSeparator>}
              {isLast || isActive ? (
                  <BreadcrumbPage className="flex items-center gap-xs">
                  {i === 0 && (
                      <HomeIcon className="w-4 h-4" onClick={() => navigate(InternalRoutes.Dashboard.StorageUnit.path)} />
                  )}
                  {route.name}
                </BreadcrumbPage>
              ) : (
                <BreadcrumbLink
                  asChild
                  className="cursor-pointer"
                  onClick={() => navigate(route.path)}
                >
                  <div className="flex items-center gap-xs" onClick={() => navigate(InternalRoutes.Dashboard.StorageUnit.path)}>
                    {i === 0 && (
                        <HomeIcon className="w-4 h-4"/>
                    )}
                    {route.name}
                  </div>
                </BreadcrumbLink>
              )}
            </BreadcrumbItem>
          );
        })}
      </BreadcrumbList>
    </UxBreadcrumb>
  );
};
