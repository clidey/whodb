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

import {Toaster} from "@clidey/ux";
import {useUpdateSettingsMutation} from '@graphql';
import map from "lodash/map";
import {useCallback, useEffect} from "react";
import {Route, Routes} from "react-router-dom";
import {getStoredConsentState, optInUser, optOutUser, resetAnalyticsIdentity} from "./config/posthog";
import {getRoutes, PrivateRoute, PublicRoutes} from './config/routes';
import {NavigateToDefault} from "./pages/chat/default-chat-route";
import {useAppDispatch, useAppSelector} from "./store/hooks";
import {SettingsActions} from "./store/settings";
import {useThemeCustomization} from "./hooks/use-theme-customization";
import {useDesktopMenu} from "./hooks/useDesktop";
import {useSidebarShortcuts} from "./hooks/useSidebarShortcuts";
import {TourProvider} from "./components/tour/tour-provider";
import {useKeyboardShortcutsHelp} from "./components/keyboard-shortcuts-help";
import {useCommandPalette} from "./components/command-palette";
import {healthCheckService} from "./services/health-check";
import {ServerDownOverlay, DatabaseDownOverlay} from "./components/health/health-overlays";
import {HealthActions} from "./store/health";

export const App = () => {
    const [updateSettings] = useUpdateSettingsMutation();
    const dispatch = useAppDispatch();
  const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled);

  // Apply UI customization settings
  useThemeCustomization();

  // Setup desktop menu and keyboard shortcuts
  useDesktopMenu();

  // Setup keyboard shortcuts help modal (? key)
  const { KeyboardShortcutsHelpModal } = useKeyboardShortcutsHelp();

  // Setup command palette (Cmd+K)
  const { CommandPaletteModal } = useCommandPalette();

  // Setup sidebar navigation shortcuts (Ctrl+1-4 on Mac, Alt+1-4 on Windows/Linux, Cmd/Ctrl+B)
  useSidebarShortcuts();

  useEffect(() => {
      const consent = getStoredConsentState();

      if (consent === 'denied' && metricsEnabled) {
          dispatch(SettingsActions.setMetricsEnabled(false));
          return;
      }

      if (consent === 'granted' && !metricsEnabled) {
          dispatch(SettingsActions.setMetricsEnabled(true));
          return;
      }

      if (consent === 'unknown') {
          return;
      }

      if (!metricsEnabled) {
          optOutUser();
          return;
      }

      optInUser();
  }, [metricsEnabled, dispatch]);

    useEffect(() => {
        const consent = getStoredConsentState();
        if (consent !== 'granted') {
            if (!metricsEnabled || consent === 'denied') {
                resetAnalyticsIdentity().catch(() => undefined);
            }
            return;
        }

        if (!metricsEnabled) {
            resetAnalyticsIdentity().catch(() => undefined);
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
    updateBackendWithSettings();
  }, [updateBackendWithSettings]);

  // Start health check service when user logs in, stop when they log out
  const authStatus = useAppSelector(state => state.auth.status);
  useEffect(() => {
    if (authStatus === 'logged-in') {
      healthCheckService.start();
    } else {
      healthCheckService.stop();
      // Reset health state when logged out
      dispatch(HealthActions.resetHealth());
    }

    return () => {
      healthCheckService.stop();
    };
  }, [authStatus, dispatch]);

  return (
    <TourProvider>
      <div className="h-[100vh] w-[100vw]" id="whodb-app-container">
        <Toaster />
        {KeyboardShortcutsHelpModal}
        {CommandPaletteModal}
        <ServerDownOverlay />
        <DatabaseDownOverlay />
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
    </TourProvider>
  );
}
