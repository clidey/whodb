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

import { Toaster } from "@clidey/ux";
import { useUpdateSettingsMutation } from '@graphql';
import map from "lodash/map";
import { useCallback, useEffect } from "react";
import { Route, Routes } from "react-router-dom";
import { optInUser, optOutUser } from "./config/posthog";
import { PrivateRoute, PublicRoutes, getRoutes } from './config/routes';
import { NavigateToDefault } from "./pages/chat/default-chat-route";
import { useAppSelector } from "./store/hooks";
import { useThemeCustomization } from "./hooks/use-theme-customization";
import { useDesktopMenu } from "./hooks/useDesktop";

export const App = () => {
  const [updateSettings, ] = useUpdateSettingsMutation();
  const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled);

  // Apply UI customization settings
  useThemeCustomization();

  // Setup desktop menu and keyboard shortcuts
  useDesktopMenu();

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
    <div className="h-[100vh] w-[100vw]" id="whodb-app-container">
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
