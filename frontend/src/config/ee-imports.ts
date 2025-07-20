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
    ThemeConfig: null,
};

// Load EE components if in EE mode
if (isEEMode) {
    // Dynamic imports for EE components
    const loadEEComponents = async () => {
        try {
            // Load analyze view
            const analyzeModule = await import('@ee/pages/raw-execute/analyze-view').catch(() => null);
            if (analyzeModule?.AnalyzeGraph) {
                EEComponents.AnalyzeGraph = analyzeModule.AnalyzeGraph;
            }

            // Load theme config
            const themeModule = await import('@ee/components/theme/theme').catch(() => null);
            if (themeModule?.ThemeConfig) {
                EEComponents.ThemeConfig = themeModule.ThemeConfig;
            }

            // Load charts
            const chartsModule = await import('@ee/components/charts').catch(() => null);
            if (chartsModule) {
                if (chartsModule.LineChart) EEComponents.LineChart = chartsModule.LineChart;
                if (chartsModule.PieChart) EEComponents.PieChart = chartsModule.PieChart;
            }
        } catch (error) {
            console.warn('Some EE components could not be loaded:', error);
        }
    };

    loadEEComponents();
}