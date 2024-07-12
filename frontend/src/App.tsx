import { map } from "lodash";
import { Navigate, Route, Routes } from "react-router-dom";
import { Notifications } from './components/notifications';
import { InternalRoutes, PrivateRoute, PublicRoutes, getRoutes } from './config/routes';
import { useAppSelector } from "./store/hooks";
import classNames from "classnames";

export const App = () => {
  const darkModeEnabled = useAppSelector(state => state.global.theme === "dark");
  return (
    <div className={classNames("h-[100vh] w-[100vw]", {
      "dark": darkModeEnabled,
    })} id="whodb-app-container">
      <Notifications />
      <Routes>
        <Route path="/" element={<PrivateRoute />}>
          {map(getRoutes(), route => (
            <Route key={route.path} path={route.path} element={route.component} />
          ))}
          <Route path="/" element={<Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />} />
        </Route>
        <Route path={PublicRoutes.Login.path} element={PublicRoutes.Login.component} />
      </Routes>
    </div>
  );
}