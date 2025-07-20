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

// Feature flags for Enterprise Edition features
export interface FeatureFlags {
    analyzeView: boolean;
    customTheme: boolean;
    // Add more EE features as needed
}

// Default feature flags (all disabled for open source version)
const defaultFeatures: FeatureFlags = {
    analyzeView: false,
    customTheme: false,
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

// Get feature flags based on environment and EE availability
export const getFeatureFlags = (): FeatureFlags => {
    const isEEAvailable = checkEEAvailability();
    
    if (!isEEAvailable) {
        return defaultFeatures;
    }
    
    // When EE is available, enable EE features
    return {
        analyzeView: true,
        customTheme: true,
    };
};

// Singleton instance of feature flags
export const featureFlags = getFeatureFlags();