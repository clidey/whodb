/*
 * Copyright 2026 Clidey, Inc.
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

import { FC, lazy, Suspense } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { useAppSelector } from "../store/hooks";
import { LogoutPage } from "../pages/auth/logout";
import { LoadingPage } from "../components/loading";
import { IInternalRoute, getEERoutes } from "./route-registry";

// Re-export IInternalRoute so existing imports from this file continue to work.
export type { IInternalRoute };

// Lazy load heavy CE components
const LoginPage = lazy(() => import("../pages/auth/login").then(m => ({ default: m.LoginPage })));
const GraphPage = lazy(() => import("../pages/graph/graph").then(m => ({ default: m.GraphPage })));
const ExploreStorageUnit = lazy(() => import("../pages/storage-unit/explore-storage-unit").then(m => ({ default: m.ExploreStorageUnit })));
const StorageUnitPage = lazy(() => import("../pages/storage-unit/storage-unit").then(m => ({ default: m.StorageUnitPage })));
const RawExecutePage = lazy(() => import("../pages/raw-execute/raw-execute").then(m => ({ default: m.RawExecutePage })));
const ChatPage = lazy(() => import("../pages/chat/chat").then(m => ({ default: m.ChatPage })));
const SettingsPage = lazy(() => import("../pages/settings/settings").then(m => ({ default: m.SettingsPage })));
const ContactUsPage = lazy(() => import("../pages/contact-us/contact-us").then(m => ({ default: m.ContactUsPage })));

// Wrapper component for lazy loaded routes
const LazyRoute: FC<{ component: React.ComponentType<any> }> = ({ component: Component }) => (
  <Suspense fallback={<LoadingPage />}>
    <Component />
  </Suspense>
);

export const PublicRoutes = {
    Login: {
        name: "Login",
        path: "/login",
        component: <Suspense fallback={<LoadingPage />}><LoginPage /></Suspense>,
    },
}

export const InternalRoutes = {
    Dashboard: {
        StorageUnit: {
            name: "Storage Unit",
            path: "/storage-unit",
            component: <LazyRoute component={StorageUnitPage} />,
        },
        ExploreStorageUnit: {
            name: "Explore",
            path: "/storage-unit/explore",
            component: <LazyRoute component={ExploreStorageUnit} />,
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
    Settings: {
        name: "Settings",
        path: "/settings",
        component: <LazyRoute component={SettingsPage} />,
    },
    ContactUs: {
        name: "Contact Us",
        path: "/contact-us",
        component: <LazyRoute component={ContactUsPage} />,
    },
    Logout: {
        name: "Logout",
        path: "/logout",
        component: <LogoutPage />,
    },
}

export const PrivateRoute: FC = () => {
    const loggedIn = useAppSelector(state => state.auth.status === "logged-in");
    if(loggedIn) {
        return <Outlet />;
    }
    return <Navigate to={PublicRoutes.Login.path} />
}

export const getRoutes = (): IInternalRoute[] => {
    // Collect CE routes
    const ceRoutes: IInternalRoute[] = [];
    const queue = Object.values(InternalRoutes) as any[];
    while (queue.length > 0) {
        const current = queue.shift();
        if (current == null) continue;
        if ("path" in current) {
            ceRoutes.push(current);
        } else {
            queue.push(...Object.values(current));
        }
    }

    // Merge: EE routes override CE routes with the same path
    const routeMap = new Map<string, IInternalRoute>();
    for (const r of ceRoutes) routeMap.set(r.path, r);
    for (const r of getEERoutes()) routeMap.set(r.path, r);

    return Array.from(routeMap.values()).filter(r => r.component != null);
}
