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

import { values } from "lodash";
import { FC, ReactNode } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { GraphPage } from "../pages/graph/graph";
import { LoginPage } from "../pages/auth/login";
import { ExploreStorageUnit } from "../pages/storage-unit/explore-storage-unit";
import { StorageUnitPage } from "../pages/storage-unit/storage-unit";
import { useAppSelector } from "../store/hooks";
import { RawExecutePage } from "../pages/raw-execute/raw-execute";
import { LogoutPage } from "../pages/auth/logout";
import { ChatPage } from "../pages/chat/chat";
import {SettingsPage} from "../pages/settings/settings";
import {ContactUsPage} from "../pages/contact-us/contact-us";
import { isEEFeatureEnabled } from "../utils/ee-loader";

export type IInternalRoute = {
    name: string;
    path: string;
    component: ReactNode;
    public?: boolean;
}

export const PublicRoutes = {
    Login: {
        name: "Login",
        path: "/login",
        component: <LoginPage />,
    },
}

export const InternalRoutes = {
    Dashboard: {
        StorageUnit: {
            name: "Storage Unit", // should update on the page
            path: "/storage-unit",
            component: <StorageUnitPage />,
        },
        ExploreStorageUnit: {
            name: "Explore",
            path: "/storage-unit/explore",
            component: <ExploreStorageUnit />,
        },
    },
    Graph: {
        name: "Graph",
        path: "/graph",
        component: <GraphPage />,
    },
    RawExecute: {
        name: "Scratchpad",
        path: "/scratchpad",
        component: <RawExecutePage />,
    },
    Chat: {
        name: "Chat",
        path: "/chat",
        component: <ChatPage />,
    },
    Logout: {
        name: "Logout",
        path: "/logout",
        component: <LogoutPage />,
    },
    ...(isEEFeatureEnabled('settingsPage') ? {
        Settings: {
            name: "Settings",
            path: "/settings",
            component: <SettingsPage />
        }
    } : {}),
    ...(isEEFeatureEnabled('contactUsPage') ? {
        ContactUs: {
            name: "Contact Us",
            path: "/contact-us",
            component: <ContactUsPage />
        }
    } : {})
}

export const PrivateRoute: FC = () => {
    const loggedIn = useAppSelector(state => state.auth.status === "logged-in");
    if(loggedIn) {
        return <Outlet />;
    }
    return <Navigate to={PublicRoutes.Login.path} />
}


export const getRoutes = (): IInternalRoute[] => {
    const allRoutes: IInternalRoute[] = [];
    const currentRoutes = values(InternalRoutes);
    while (currentRoutes.length > 0) {
        const currentRoute = currentRoutes.shift();
        if (currentRoute == null) {
            continue;
        }
        if ("path" in currentRoute) {
            allRoutes.push(currentRoute);
            continue;
        }
        currentRoutes.push(...values((currentRoute)));
    }
    return allRoutes;
}