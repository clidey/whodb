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

// This module must stay free of side effects and DOM access so that
// Node-based tooling (e.g. Playwright specs) can import the flag defaults.

// Feature flags control which UI features are active.
export interface FeatureFlags {
    analyzeView: boolean;
    explainView: boolean;
    generateView: boolean;
    customTheme: boolean;
    dataVisualization: boolean;
    aiChat: boolean;
    multiProfile: boolean;
    advancedDatabases: boolean;
    contactUsPage: boolean;
    settingsPage: boolean;
    sampleDatabaseTour: boolean;
    autoStartTourOnLogin: boolean;
    sqlAgent: boolean;
    keyboardShortcutsHelp: boolean;
    platformFunnel: boolean;
}

/** Default feature flag values for the CE build. */
export const defaultFeatures: FeatureFlags = {
    analyzeView: false,
    explainView: false,
    generateView: false,
    customTheme: false,
    dataVisualization: false,
    aiChat: false,
    multiProfile: false,
    advancedDatabases: false,
    contactUsPage: true, // Enabled in CE
    settingsPage: true, // Enabled in CE
    sampleDatabaseTour: true, // Enabled in CE
    autoStartTourOnLogin: true, // Enabled in CE
    sqlAgent: false,
    keyboardShortcutsHelp: true, // Enabled in CE
    platformFunnel: false,
};
