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

import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import { ApolloProvider } from "@apollo/client";
import { createGraphqlClient } from './config/graphql-client';
import { Provider } from "react-redux";
import { reduxStore, reduxStorePersistor } from '@/store';
import { App } from '@/app';
import { HashRouter } from "react-router-dom";
import { PersistGate } from 'redux-persist/integration/react';
import { PostHogProvider } from 'posthog-js/react';
import {initPosthog} from "@/config/posthog";
import { ThemeProvider } from '@clidey/ux'
import { isEEMode } from '@/config/ee-imports';
import 'virtual:frontend-build-css';

if (isEEMode) {
  import("@ee/index.css");
}


const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

const posthogClient = initPosthog()

// Conditionally wrap with PostHogProvider only for CE builds
const AppWithProviders = () => {
  const app = (
    <ThemeProvider>
      <App />
    </ThemeProvider>
  );

  // Only wrap with PostHogProvider if we have a client (CE builds)
  if (posthogClient) {
    // @ts-ignore
    return <PostHogProvider client={posthogClient}>{app}</PostHogProvider>;
  }
  
  return app;
};

async function boot() {
  const graphqlClient = await createGraphqlClient();
  root.render(
    <React.StrictMode>
      <HashRouter>
        <ApolloProvider client={graphqlClient as any}>
          <Provider store={reduxStore}>
            <PersistGate loading={null} persistor={reduxStorePersistor}>
              <AppWithProviders />
            </PersistGate>
          </Provider>
        </ApolloProvider>
      </HashRouter>
    </React.StrictMode>
  );
}

boot();
