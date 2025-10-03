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

import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import { ApolloProvider } from "@apollo/client";
import { graphqlClient } from './config/graphql-client';
import { Provider } from "react-redux";
import { reduxStore, reduxStorePersistor } from './store';
import { App } from './app';
import { BrowserRouter, HashRouter } from "react-router-dom";
import { PersistGate } from 'redux-persist/integration/react';
import { PostHogProvider } from 'posthog-js/react';
import {initPosthog} from "./config/posthog";
import { ThemeProvider } from '@clidey/ux'
import { isEEMode } from './config/ee-imports';
import { isDesktopApp } from './utils/external-links';

if (isEEMode) {
  import("@ee/index.css");
}

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

// Component to handle async PostHog initialization
const AppWithProviders = () => {
  const [posthogClient, setPosthogClient] = useState<any>(null);
  const [isLoading, setIsLoading] = useState(!isEEMode); // Only load for CE

  useEffect(() => {
    if (!isEEMode) {
      initPosthog().then(client => {
        setPosthogClient(client);
        setIsLoading(false);
      });
    }
  }, []);

  const app = (
    <ThemeProvider>
      <App />
    </ThemeProvider>
  );

  // For EE mode, no PostHog provider needed
  if (isEEMode) {
    return app;
  }

  // For CE mode, wait for PostHog to initialize
  if (isLoading) {
    return app; // Show app without PostHog while loading
  }

  // Wrap with PostHogProvider once loaded (CE builds)
  if (posthogClient) {
    // @ts-ignore
    return <PostHogProvider client={posthogClient}>{app}</PostHogProvider>;
  }

  return app;
};

// Use HashRouter for desktop app (avoids full page reloads)
// Use BrowserRouter for web version
const Router = isDesktopApp() ? HashRouter : BrowserRouter;

root.render(
  <React.StrictMode>
    <Router>
      <ApolloProvider client={graphqlClient}>
        <Provider store={reduxStore}>
          <PersistGate loading={null} persistor={reduxStorePersistor}>
            <AppWithProviders />
          </PersistGate>
        </Provider>
      </ApolloProvider>
    </Router>
  </React.StrictMode>
);
