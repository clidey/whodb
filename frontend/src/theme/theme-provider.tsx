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

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { loadEEModule, isEEFeatureEnabled } from '../utils/ee-loader';
import { updateThemeClasses } from '../components/classes';
import { ThemeConfigType } from '../config/ee-types';

// Use a partial version of ThemeConfigType for the default theme
export type ThemeConfig = Partial<ThemeConfigType>;

// Default theme values
const defaultTheme: ThemeConfig = {
    //@ts-ignore
    components: {
        text: "text-[#333333] dark:text-[#E0E0E0]",
        brandText: "text-[#ca6f1e] dark:text-[#ca6f1e]",
        button: "rounded-lg border flex justify-center items-center text-xs px-2 py-1 cursor-pointer gap-1 bg-gradient-to-r from-white to-gray-100/50 dark:from-[#2C2F33] dark:to-[#23272A] border-neutral-600/20 dark:border-white/5",
        icon: "bg-teal-500",
    }
};

// Theme context
interface ThemeContextValue {
    theme: ThemeConfig;
    setTheme: (theme: ThemeConfig) => void;
    isEEThemeAvailable: boolean;
}

const ThemeContext = createContext<ThemeContextValue>({
    theme: defaultTheme,
    setTheme: () => {},
    isEEThemeAvailable: false,
});

export const useTheme = () => useContext(ThemeContext);

interface ThemeProviderProps {
    children: ReactNode;
}

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ children }) => {
    const [theme, setTheme] = useState<ThemeConfig>(defaultTheme);
    const [isEEThemeAvailable, setIsEEThemeAvailable] = useState(false);

    useEffect(() => {
        const loadEETheme = async () => {
            if (isEEFeatureEnabled('customTheme')) {
                try {
                    const eeTheme = await loadEEModule(
                        () => import('@ee/components/theme/theme')
                    );
                    
                    if (eeTheme?.ThemeConfig) {
                        // Merge EE theme with default theme
                        const mergedTheme = {
                            ...defaultTheme,
                            ...eeTheme.ThemeConfig,
                            components: {
                                ...defaultTheme.components,
                                ...eeTheme.ThemeConfig.components,
                            },
                            layout: {
                                ...defaultTheme.layout,
                                ...eeTheme.ThemeConfig.layout,
                            },
                        };
                        setTheme(mergedTheme);
                        updateThemeClasses(mergedTheme);
                        setIsEEThemeAvailable(true);
                    }
                } catch (error) {
                    console.warn('Failed to load EE theme:', error);
                }
            }
        };

        loadEETheme();
    }, []);

    return (
        <ThemeContext.Provider value={{ theme, setTheme, isEEThemeAvailable }}>
            {children}
        </ThemeContext.Provider>
    );
};