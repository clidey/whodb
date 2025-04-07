import classNames from "classnames";
import { map } from "lodash";
import { Route, Routes } from "react-router-dom";
import { Notifications } from './components/notifications';
import { PrivateRoute, PublicRoutes, getRoutes } from './config/routes';
import { NavigateToDefault } from "./pages/chat/default-chat-route";
import { useAppSelector } from "./store/hooks";
import {initHighlight, startHighlight, stopHighlight} from "./config/highlight";
import {isDevelopment} from "./utils/functions";
import {useCallback, useEffect} from "react";
import {useUpdateSettingsMutation} from "./generated/graphql";

export const App = () => {
  const [updateSettings, ] = useUpdateSettingsMutation();
  const darkModeEnabled = useAppSelector(state => state.global.theme === "dark");
  const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled);

  useEffect(() => {
      if (metricsEnabled) {
        initHighlight(isDevelopment() ? "development" : "production")
        startHighlight()
      } else {
        stopHighlight()
      }
  }, [metricsEnabled]);

  const updateBackendWithSettings = useCallback(() => {
    updateSettings({
      variables: {
        newSettings: {
          MetricsEnabled: String(metricsEnabled)
        }
      }
    });
  }, [updateSettings, metricsEnabled])

  useEffect(() => {
    updateBackendWithSettings()
  }, [updateBackendWithSettings]);

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
