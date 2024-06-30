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

export type IInternalRoute = {
    name: string;
    path: string;
    component: ReactNode;
    public?: boolean;
}

export const PublicRoutes = {
    Login: {
        name: "Home",
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
        name: "Raw Execute",
        path: "/raw-execute",
        component: <RawExecutePage />,
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