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

import {ComponentType, lazy, LazyExoticComponent} from 'react';
import {featureFlags} from '../config/features';

// Type for EE components that may or may not be available
export type OptionalComponent<T = any> = ComponentType<T> | null;

// Safely load an EE component with fallback
export const loadEEComponent = <T extends ComponentType<any>>(
    importPath: () => Promise<{ default: T } | any>,
    fallback: T | null = null
): LazyExoticComponent<T> | T | null => {
    // Check if EE features are enabled
    const eeEnabled = import.meta.env.VITE_BUILD_EDITION === 'ee';
    
    if (!eeEnabled) {
        return fallback;
    }
    
    try {
        return lazy(async () => {
            try {
                const module = await importPath();
                if (!module || !module.default) {
                    console.warn('EE component module loaded but no default export found');
                    if (fallback) {
                        return { default: fallback };
                    }
                    throw new Error('EE component not properly exported');
                }
                return module;
            } catch (error) {
                // More detailed error logging
                if (error instanceof Error) {
                    if (error.message.includes('Cannot find module')) {
                        console.warn('EE component not found - this is expected in CE builds');
                    } else {
                        console.error('Error loading EE component:', error.message);
                    }
                } else {
                    console.warn('EE component not available');
                }

                return {default: fallback};
            }
        });
    } catch (error) {
        console.warn('Failed to create lazy component:', error);
        return fallback;
    }
};

// Type-safe conditional import for non-component modules
export const loadEEModule = async <T>(
    importPath: () => Promise<T>,
    fallback: T | null = null
): Promise<T | null> => {
    const eeEnabled = import.meta.env.VITE_BUILD_EDITION === 'ee';
    
    if (!eeEnabled) {
        return fallback;
    }
    
    try {
        const module = await importPath();
        if (!module) {
            console.warn('EE module loaded but returned null/undefined');
            return fallback;
        }
        return module;
    } catch (error) {
        if (error instanceof Error) {
            if (error.message.includes('Cannot find module')) {
                console.warn('EE module not found - this is expected in CE builds');
            } else {
                console.error('Error loading EE module:', error.message);
            }
        } else {
            console.warn('EE module not available');
        }
        return fallback;
    }
};

// Check if a specific EE feature is enabled
export const isEEFeatureEnabled = (feature: keyof typeof featureFlags): boolean => {
    return featureFlags[feature] === true;
};