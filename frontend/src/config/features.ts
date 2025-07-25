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

import { FeatureFlags } from './ee-types';

// Default feature flags (all disabled for open source version)
const defaultFeatures: FeatureFlags = {
    analyzeView: false,
    customTheme: false,
    dataVisualization: false,
    aiChat: false,
    multiProfile: false,
    advancedDatabases: false,
    contactUsPage: true, // Enabled in CE
    settingsPage: true, // Enabled in CE
};

// Check if EE modules are available
const checkEEAvailability = (): boolean => {
    try {
        // This will be replaced by the build system when EE is enabled
        return import.meta.env.VITE_BUILD_EDITION === 'ee';
    } catch {
        return false;
    }
};

// Store the loaded EE feature flags
let loadedEEFeatures: FeatureFlags | null = null;

// Get feature flags based on environment and EE availability
export const getFeatureFlags = (): FeatureFlags => {
    const isEEAvailable = checkEEAvailability();
    
    if (!isEEAvailable) {
        return defaultFeatures;
    }
    
    // If we've already loaded EE features, return them
    if (loadedEEFeatures) {
        return loadedEEFeatures;
    }
    
    // Try to load EE features asynchronously
    if (isEEAvailable) {
        import('@ee/config').then(eeConfig => {
            if (eeConfig?.eeFeatures) {
                loadedEEFeatures = eeConfig.eeFeatures;
            }
        }).catch(() => {
            console.warn('Could not load EE feature flags');
        });
    }
    
    // Return default enabled features for EE mode until async load completes
    return {
        analyzeView: true,
        customTheme: true,
        dataVisualization: true,
        aiChat: true,
        multiProfile: true,
        advancedDatabases: true,
        contactUsPage: false, // Disabled in EE
        settingsPage: false, // Disabled in EE
    };
};

// Singleton instance of feature flags
export const featureFlags = getFeatureFlags();