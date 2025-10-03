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

import values from "lodash/values";
import { FC, lazy, ReactNode, Suspense } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { LoginPage } from "../pages/auth/login";
import { useAppSelector } from "../store/hooks";
import { LogoutPage } from "../pages/auth/logout";
import { isEEFeatureEnabled } from "../utils/ee-loader";

// Lazy load heavy components
const GraphPage = lazy(() => import("../pages/graph/graph").then(m => ({ default: m.GraphPage })));
const ExploreStorageUnit = lazy(() => import("../pages/storage-unit/explore-storage-unit").then(m => ({ default: m.ExploreStorageUnit })));
const StorageUnitPage = lazy(() => import("../pages/storage-unit/storage-unit").then(m => ({ default: m.StorageUnitPage })));
const RawExecutePage = lazy(() => import("../pages/raw-execute/raw-execute").then(m => ({ default: m.RawExecutePage })));
const ChatPage = lazy(() => import("../pages/chat/chat").then(m => ({ default: m.ChatPage })));
const SettingsPage = lazy(() => import("../pages/settings/settings").then(m => ({ default: m.SettingsPage })));
const ContactUsPage = lazy(() => import("../pages/contact-us/contact-us").then(m => ({ default: m.ContactUsPage })));

// Loading component
const PageLoader = () => (
  <div className="flex h-full w-full items-center justify-center">
    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-gray-100"></div>
  </div>
);

// Wrapper component for lazy loaded routes
const LazyRoute: FC<{ component: React.ComponentType<any> }> = ({ component: Component }) => (
  <Suspense fallback={<PageLoader />}>
    <Component />
  </Suspense>
);

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
            component: <LazyRoute component={StorageUnitPage} />,
        },
        ExploreStorageUnit: {
            name: "Explore",
            path: "/storage-unit/explore",
            component: <LazyRoute component={ExploreStorageUnit} />,
        },
        ExploreStorageUnitWithScratchpad: {
            name: "Explore",
            path: "/storage-unit/explore/scratchpad",
            component: <LazyRoute component={() => <ExploreStorageUnit scratchpad={true} />} />,
        },
    },
    Graph: {
        name: "Graph",
        path: "/graph",
        component: <LazyRoute component={GraphPage} />,
    },
    RawExecute: {
        name: "Scratchpad",
        path: "/scratchpad",
        component: <LazyRoute component={RawExecutePage} />,
    },
    Chat: {
        name: "Chat",
        path: "/chat",
        component: <LazyRoute component={ChatPage} />,
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
            component: <LazyRoute component={SettingsPage} />
        }
    } : {}),
    ...(isEEFeatureEnabled('contactUsPage') ? {
        ContactUs: {
            name: "Contact Us",
            path: "/contact-us",
            component: <LazyRoute component={ContactUsPage} />
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