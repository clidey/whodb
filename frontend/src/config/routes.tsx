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

import type { FC, ReactNode} from "react";
import { lazy, Suspense } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { useAppSelector } from "../store/hooks";
import { LogoutPage } from "../pages/auth/logout";
import { getComponent } from "./component-registry";
import { featureFlags } from "./features";
import { LoadingPage } from "../components/loading";
import { getRegisteredUnscopedRoutes, getSurfaceFallbackPath } from "./route-registry";
import { useSourceContract } from "../hooks/useSourceContract";
export { registerRoute } from "./route-registry";

// Allow EE to override the login page via the component registry (e.g. for SSO).
// Falls back to the CE database-credential login page when no override is registered.
// Must be lazy — getComponent must run at render time (after EE register.ts populates the registry).
const CELoginPage = lazy(() => import("../pages/auth/login").then(m => ({ default: m.LoginPage })));
const LoginPage = () => {
    const Registered = getComponent('login-page');
    if (Registered) return <Registered />;
    return <CELoginPage />;
};

const GraphPage = lazy(() => import("../pages/graph/graph").then(m => ({ default: m.GraphPage })));
const ExploreStorageUnit = lazy(() => import("../pages/storage-unit/explore-storage-unit").then(m => ({ default: m.ExploreStorageUnit })));
const StorageUnitPage = lazy(() => import("../pages/storage-unit/storage-unit").then(m => ({ default: m.StorageUnitPage })));
const RawExecutePage = lazy(() => import("../pages/raw-execute/raw-execute").then(m => ({ default: m.RawExecutePage })));
const ChatPage = lazy(() => import("../pages/chat/chat").then(m => ({ default: m.ChatPage })));
const SettingsPage = lazy(() => import("../pages/settings/settings").then(m => ({ default: m.SettingsPage })));
const ContactUsPage = lazy(() => import("../pages/contact-us/contact-us").then(m => ({ default: m.ContactUsPage })));
const ChatRouteComponent: FC = () => {
    const Agent = getComponent('sql-agent');
    if (Agent) {
        return <Suspense fallback={<LoadingPage />}><Agent /></Suspense>;
    }
    return <SourceSurfaceRoute surface="chat" component={<LazyRoute component={ChatPage} />} />;
};

// Wrapper component for lazy loaded routes
const LazyRoute: FC<{ component: React.ComponentType<any> }> = ({ component: Component }) => (
  <Suspense fallback={<LoadingPage />}>
    <Component />
  </Suspense>
);

const SourceSurfaceRoute: FC<{
    surface: "chat" | "graph" | "scratchpad";
    component: ReactNode;
}> = ({ surface, component }) => {
    const currentType = useAppSelector(state => state.auth.current?.Type);
    const { loading, supportsChat, supportsGraph, supportsScratchpad } = useSourceContract(currentType);

    if (loading) {
        return <LoadingPage />;
    }

    const isAllowed = surface === "chat"
        ? supportsChat
        : surface === "graph"
            ? supportsGraph
            : supportsScratchpad;

    if (!isAllowed) {
        return <Navigate to={getSurfaceFallbackPath()} replace />;
    }

    return component;
};

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
        component: <Suspense fallback={<LoadingPage />}><LoginPage /></Suspense>,
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
    },
    Graph: {
        name: "Graph",
        path: "/graph",
        component: <SourceSurfaceRoute surface="graph" component={<LazyRoute component={GraphPage} />} />,
    },
    RawExecute: {
        name: "Scratchpad",
        path: "/scratchpad",
        component: <SourceSurfaceRoute surface="scratchpad" component={<LazyRoute component={RawExecutePage} />} />,
    },
    Chat: {
        name: "Chat",
        path: "/chat",
        component: <ChatRouteComponent />,
    },
    Logout: {
        name: "Logout",
        path: "/logout",
        component: <LogoutPage />,
    },
    ...(featureFlags.settingsPage ? {
        Settings: {
            name: "Settings",
            path: "/settings",
            component: <LazyRoute component={SettingsPage} />
        }
    } : {}),
    ...(featureFlags.contactUsPage ? {
        ContactUs: {
            name: "Contact Us",
            path: "/contact-us",
            component: <LazyRoute component={ContactUsPage} />
        }
    } : {})
}

export const PrivateRoute: FC = () => {
    const loggedIn = useAppSelector(state => state.auth.status === "logged-in");
    const SetupGuard = getComponent('setup-guard') as FC<{ children: ReactNode }> | undefined;
    const AuthSessionGuard = getComponent('auth-session-guard') as FC<{ loggedIn: boolean; children: ReactNode }> | undefined;
    const children = SetupGuard ? (
        <Suspense fallback={<LoadingPage />}>
            <SetupGuard><Outlet /></SetupGuard>
        </Suspense>
    ) : <Outlet />;

    if (AuthSessionGuard) {
        return (
            <Suspense fallback={<LoadingPage />}>
                <AuthSessionGuard loggedIn={loggedIn}>{children}</AuthSessionGuard>
            </Suspense>
        );
    }

    if(loggedIn) {
        return children;
    }
    return <Navigate to={PublicRoutes.Login.path} />
}

export const getRoutes = (): IInternalRoute[] => {
    const allRoutes: IInternalRoute[] = [];
    const currentRoutes = Object.values(InternalRoutes);
    while (currentRoutes.length > 0) {
        const currentRoute = currentRoutes.shift();
        if (currentRoute == null) {
            continue;
        }
        if ("path" in currentRoute) {
            allRoutes.push(currentRoute);
            continue;
        }
        currentRoutes.push(...Object.values((currentRoute)));
    }
    const extra = getRegisteredUnscopedRoutes().map(({ name, path, lazyComponent }) => ({
        name,
        path,
        component: <LazyRoute component={lazyComponent} />,
    }));
    return [...allRoutes, ...extra];
}
