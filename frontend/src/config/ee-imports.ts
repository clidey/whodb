/*
 * Copyright 2026 Clidey, Inc.
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

/**
 * EE Import Configuration
 * This file handles conditional imports for Enterprise Edition features
 *
 * IMPORTANT: This module uses top-level await to ensure EE settings are loaded
 * synchronously before Redux store initialization. Do not import this module
 * from any code that runs before the store is created.
 */

import { EEComponentTypes, SettingsDefaults } from './ee-types';

// Check if we're in EE mode
export const isEEMode = import.meta.env.VITE_BUILD_EDITION === 'ee';

// Export properly typed components
export const EEComponents: EEComponentTypes = {
    AnalyzeGraph: null,
    LineChart: null,
    PieChart: null,
};

// Load EE settings FIRST using top-level await before exporting
// This ensures eeSettingsDefaults is populated before any consumer can access it
let settingsDefaults: SettingsDefaults = {};

if (isEEMode) {
    try {
        // Top-level await - blocks module initialization until config is loaded
        const eeConfig = await import('@ee/config');
        if (eeConfig?.eeSettingsDefaults) {
            settingsDefaults = eeConfig.eeSettingsDefaults;
        }
    } catch (error) {
        console.warn('EE config could not be loaded:', error);
    }
}

// Export AFTER loading
export const eeSettingsDefaults: SettingsDefaults = settingsDefaults;

// Load EE components asynchronously (they're not needed at module initialization)
if (isEEMode) {
    const loadEEComponents = async () => {
        try {
            // Load all EE exports at once
            const eeModule: Record<string, unknown> = await import('@ee/index').catch(() => null) as Record<string, unknown>;
            if (eeModule) {
                if (eeModule.AnalyzeGraph) EEComponents.AnalyzeGraph = eeModule.AnalyzeGraph as EEComponentTypes['AnalyzeGraph'];
                if (eeModule.LineChart) EEComponents.LineChart = eeModule.LineChart as EEComponentTypes['LineChart'];
                if (eeModule.PieChart) EEComponents.PieChart = eeModule.PieChart as EEComponentTypes['PieChart'];
            }
        } catch (error) {
            console.warn('EE components could not be loaded:', error);
        }
    };

    loadEEComponents();
}