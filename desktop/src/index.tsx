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

// @ts-ignore
import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import {ApolloProvider} from "@apollo/client";
import {createGraphqlClient} from './config/graphql-client';
import {Provider} from "react-redux";
import {reduxStore, reduxStorePersistor} from '@/store';
import {App} from '@/app';
import {HashRouter} from "react-router-dom";
import {PersistGate} from 'redux-persist/integration/react';
import {PostHogProvider} from 'posthog-js/react';
import {initPosthog} from "@/config/posthog";
import {ThemeProvider} from '@clidey/ux'
import {isEEMode} from '@/config/ee-imports';
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
  // Show loading state while creating client
  root.render(
      <div style={{display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh'}}>
        <div>Loading WhoDB Desktop...</div>
      </div>
  );

  try {
    console.log('[DEBUG] Starting boot sequence');
    const graphqlClient = await createGraphqlClient();
    console.log('[DEBUG] GraphQL client created successfully');

    root.render(
        // Temporarily disabled StrictMode to debug infinite loop issue
        // <React.StrictMode>
        <HashRouter>
          <ApolloProvider client={graphqlClient as any}>
            <Provider store={reduxStore}>
              <PersistGate loading={null} persistor={reduxStorePersistor}>
                <AppWithProviders/>
              </PersistGate>
            </Provider>
          </ApolloProvider>
        </HashRouter>
        // </React.StrictMode>
    );
    console.log('[DEBUG] App rendered successfully');
  } catch (error) {
    console.error('[ERROR] Failed to boot application:', error);
    root.render(
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          alignItems: 'center',
          height: '100vh'
        }}>
          <div>Failed to start WhoDB Desktop</div>
          <div style={{marginTop: '10px', color: 'red'}}>{String(error)}</div>
        </div>
    );
  }
}

boot();
