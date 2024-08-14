import classNames from "classnames";
import { map } from "lodash";
import { Route, Routes } from "react-router-dom";
import { Notifications } from './components/notifications';
import { PrivateRoute, PublicRoutes, getRoutes } from './config/routes';
import { NavigateToDefault } from "./pages/chat/default-chat-route";
import { useAppSelector } from "./store/hooks";

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
          <Route path="/" element={<NavigateToDefault />} />
        </Route>
        <Route path={PublicRoutes.Login.path} element={PublicRoutes.Login.component} />
      </Routes>
    </div>
  );
}