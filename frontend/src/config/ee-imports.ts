/**
 * EE Import Configuration
 * This file handles conditional imports for Enterprise Edition features
 */

import { EEComponentTypes } from './ee-types';

// Check if we're in EE mode
export const isEEMode = import.meta.env.VITE_BUILD_EDITION === 'ee';

// Export properly typed components
export const EEComponents: EEComponentTypes = {
    AnalyzeGraph: null,
    LineChart: null,
    PieChart: null,
};

// Load EE components if in EE mode
if (isEEMode) {
    // Dynamic imports for EE components
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