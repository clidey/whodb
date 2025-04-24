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
import { useAppDispatch, useAppSelector } from "./store/hooks";
import {useCallback, useEffect} from "react";
import {useUpdateSettingsMutation, useGetAiProvidersLazyQuery} from "./generated/graphql";
import {optInUser, optOutUser} from "./config/posthog";
import { DatabaseActions } from "./store/database";
import { reduxStore } from "./store";

export const App = () => {
  const [updateSettings, ] = useUpdateSettingsMutation();
  const darkModeEnabled = useAppSelector(state => state.global.theme === "dark");
  const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled);
  const [getAiProviders, ] = useGetAiProvidersLazyQuery();
  const dispatch = useAppDispatch();

  useEffect(() => {
    getAiProviders({
      fetchPolicy: "network-only",
      onCompleted(data) {
        const aiProviders = data.AIProviders || [];
        const initialModelTypes = reduxStore.getState().database.modelTypes.filter(model => {
          const existingModel = aiProviders.find(provider => provider.ProviderId === model.id);
          return existingModel != null || (model.token != null && model.token !== "");
        });

        // Filter out providers that already exist in modelTypes
        const newProviders = aiProviders.filter(provider =>
            !initialModelTypes.some(model => model.id === provider.ProviderId)
        );

        const finalModelTypes = [
          ...newProviders.map(provider => ({
            id: provider.ProviderId,
            modelType: provider.Type,
          })),
          ...initialModelTypes
        ];

        // Check if current model type exists in final model types
        const currentModelType = reduxStore.getState().database.current;
        if (currentModelType && !finalModelTypes.some(model => model.id === currentModelType.id)) {
          dispatch(DatabaseActions.setCurrentModelType({ id: "" }));
          dispatch(DatabaseActions.setModels([]));
          dispatch(DatabaseActions.setCurrentModel(undefined));
        }

        dispatch(DatabaseActions.setModelTypes(finalModelTypes));
      },
    });
  }, []);

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
