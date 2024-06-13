import { FC, ReactNode } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { DashboardPage } from "../pages/dashboard/dashboard";
import { LoginPage } from "../pages/login/login";
import { useAppSelector } from "../store/hooks";
import { values } from "lodash";
import { StorageUnitPage } from "../pages/storage-unit/storage-unit";

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
        name: "Home",
        path: "/",
        component: <DashboardPage />,
    },
    StorageUnit: {
        StorageUnit: {
            name: "Storage Unit",
            path: "/storage-unit",
            component: <StorageUnitPage />,
        },
    },
    Logout: {
        name: "Logout",
        path: "/logout",
        component: <LoginPage />,
    },
}

export const PrivateRoute: FC = () => {
    const loggedIn = useAppSelector(state => state.auth.profiles.length > 0);
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
        currentRoutes.push(...values(currentRoute));
    }
    return allRoutes;
}