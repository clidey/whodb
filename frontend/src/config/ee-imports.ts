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
 * This file handles conditional imports for Enterprise Edition features.
 *
 * NO top-level await — this module is synchronous so it does not block
 * the ES module graph.  All async EE work is deferred to initEE(), which
 * is called once from index.tsx before root.render().
 */

import { EEComponentTypes, SettingsDefaults } from './ee-types';

// Check if we're in EE mode
export const isEEMode = import.meta.env.VITE_BUILD_EDITION === 'ee';

// Export properly typed components (populated lazily by initEE)
export const EEComponents: EEComponentTypes = {
    AnalyzeGraph: null,
    LineChart: null,
    PieChart: null,
};

// Mutable defaults — populated by initEE() before root.render() in EE mode.
// In CE mode this stays empty ({}) which is correct.
export let eeSettingsDefaults: SettingsDefaults = {};

/**
 * Async EE initialisation. Must be awaited in index.tsx before root.render()
 * so that EE routes and settings are ready for the first render.
 *
 * In CE mode this is a no-op and resolves immediately.
 */
export async function initEE(): Promise<void> {
    if (import.meta.env.VITE_BUILD_EDITION !== 'ee') {
        return;
    }

    // Load EE config (settings defaults)
    try {
        const eeConfig = await import('@ee/config');
        if (eeConfig?.eeSettingsDefaults) {
            eeSettingsDefaults = eeConfig.eeSettingsDefaults;
        }
    } catch (error) {
        console.warn('EE config could not be loaded:', error);
    }

    // Register EE routes before the first render
    try {
        await import('@ee/routes');
    } catch (error) {
        console.warn('EE routes could not be loaded:', error);
    }

    // Load EE components (fire-and-forget — not needed for first render)
    import('@ee/index').then((eeModule: Record<string, unknown>) => {
        if (eeModule) {
            if (eeModule.AnalyzeGraph) EEComponents.AnalyzeGraph = eeModule.AnalyzeGraph as EEComponentTypes['AnalyzeGraph'];
            if (eeModule.LineChart) EEComponents.LineChart = eeModule.LineChart as EEComponentTypes['LineChart'];
            if (eeModule.PieChart) EEComponents.PieChart = eeModule.PieChart as EEComponentTypes['PieChart'];
        }
    }).catch(() => null);
}
