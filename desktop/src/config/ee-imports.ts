/**
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
