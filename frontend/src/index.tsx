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

import React, {useEffect, useState} from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import {ApolloProvider} from "@apollo/client";
import {graphqlClient} from './config/graphql-client';
import {Provider} from "react-redux";
import {reduxStore, reduxStorePersistor} from './store';
import {App} from './app';
import {BrowserRouter, HashRouter} from "react-router-dom";
import {PersistGate} from 'redux-persist/integration/react';
import {PostHogProvider} from 'posthog-js/react';
import type {PostHog} from 'posthog-js';
import {initPosthog} from "./config/posthog";
import {ThemeProvider} from '@clidey/ux'
import {isEEMode} from './config/ee-imports';
import {isDesktopApp} from './utils/external-links';
import {PosthogConsentBanner} from './components/analytics/posthog-consent-banner';
import {getBasePath} from './config/base-path';

// Detect desktop Linux and add a class for CSS-based overrides (e.g., fonts)
try {
    if (typeof navigator !== 'undefined' && typeof document !== 'undefined') {
        if (isDesktopApp() && /Linux/i.test(navigator.userAgent || '')) {
            document.documentElement.classList.add('linux');
        }
    }
} catch (e) {
    // best-effort; do not block startup on UA detection issues
}

if (isEEMode) {
  import("@ee/index.css");
}

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

// Initialize PostHog once and keep provider stable to prevent remounting
const AppWithProviders = () => {
    const [posthogClient, setPosthogClient] = useState<PostHog | null>(null);
    const [initialized, setInitialized] = useState(false);

    useEffect(() => {
        if (isEEMode || initialized) {
            return;
        }
        setInitialized(true);
        initPosthog()
            .then(client => setPosthogClient(client))
            .catch(() => setPosthogClient(null));
    }, [initialized]);

    const app = (
        <ThemeProvider>
            <App />
            <PosthogConsentBanner/>
        </ThemeProvider>
    );

    // For EE mode, no PostHog provider needed
    if (isEEMode) {
        return app;
    }

    if (posthogClient) {
        return <PostHogProvider client={posthogClient}>{app}</PostHogProvider>;
    }

    return app;
};

// Use HashRouter for desktop app (avoids full page reloads)
// Use BrowserRouter for web version
const Router = isDesktopApp() ? HashRouter : BrowserRouter;
const routerBasePath = getBasePath() || '/';

root.render(
  <React.StrictMode>
    <Router basename={routerBasePath}>
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
