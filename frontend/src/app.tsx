/*
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

import classNames from "classnames";
import { map } from "lodash";
import { Route, Routes } from "react-router-dom";
import { Notifications } from './components/notifications';
import { PrivateRoute, PublicRoutes, getRoutes } from './config/routes';
import { NavigateToDefault } from "./pages/chat/default-chat-route";
import { useAppSelector } from "./store/hooks";
import { useCallback, useEffect } from "react";
import { useUpdateSettingsMutation } from '@graphql';
import { optInUser, optOutUser } from "./config/posthog";
import { Toaster } from "@clidey/ux";

export const App = () => {
  const [updateSettings, ] = useUpdateSettingsMutation();
  const darkModeEnabled = useAppSelector(state => state.global.theme === "dark");
  const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled);

  useEffect(() => {
      if (metricsEnabled) {
        optInUser();
      } else {
        optOutUser();
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
      <Toaster />
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
