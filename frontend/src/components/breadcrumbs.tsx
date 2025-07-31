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
import { FC } from "react";
import { useNavigate } from "react-router-dom";
import { IInternalRoute } from "../config/routes";
import { Icons } from "./icons";
import {
  Breadcrumb as UxBreadcrumb,
  BreadcrumbList,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@clidey/ux";

export type IBreadcrumbRoute = Omit<IInternalRoute, "component">;

type IBreadcrumbProps = {
  routes: IBreadcrumbRoute[];
  active?: IBreadcrumbRoute;
};

export const Breadcrumb: FC<IBreadcrumbProps> = ({ routes, active }) => {
  const navigate = useNavigate();

  return (
    <UxBreadcrumb className="py-2 px-2">
      <BreadcrumbList>
        {routes.map((route, i) => {
          const isActive = active === route;
          const isLast = i === routes.length - 1;
          return (
            <BreadcrumbItem key={route.name}>
              {i > 0 && <BreadcrumbSeparator>{Icons.RightChevron}</BreadcrumbSeparator>}
              {isLast || isActive ? (
                <BreadcrumbPage>
                  {i === 0 && (
                    <span className="inline-flex items-center mr-1">
                      {Icons.Home}
                    </span>
                  )}
                  {route.name}
                </BreadcrumbPage>
              ) : (
                <BreadcrumbLink
                  asChild
                  className="cursor-pointer"
                  onClick={() => navigate(route.path)}
                >
                  <span className="flex items-center gap-2">
                    {i === 0 && (
                      <span className="inline-flex items-center mr-1">
                        {Icons.Home}
                      </span>
                    )}
                    {route.name}
                  </span>
                </BreadcrumbLink>
              )}
            </BreadcrumbItem>
          );
        })}
      </BreadcrumbList>
    </UxBreadcrumb>
  );
};
