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
            const eeModule = await import('@ee/index').catch(() => null);
            if (eeModule) {
                if (eeModule.AnalyzeGraph) EEComponents.AnalyzeGraph = eeModule.AnalyzeGraph;
                // @ts-ignore - TODO: fix this
                if (eeModule.LineChart) EEComponents.LineChart = eeModule.LineChart;
                // @ts-ignore - TODO: fix this
                if (eeModule.PieChart) EEComponents.PieChart = eeModule.PieChart;
            }
        } catch (error) {
            console.warn('EE components could not be loaded:', error);
        }
    };

    loadEEComponents();
}