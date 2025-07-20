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

import { ThemeConfig } from '../theme/theme-provider';

export const BRAND_COLOR_BG = "bg-[#ca6f1e] dark:bg-[#ca6f1e]";

// Default class names
const defaultClassNames = {
    Text: "text-[#333333] dark:text-[#E0E0E0]",
    BrandText: "text-[#ca6f1e] dark:text-[#ca6f1e]",
    Button: "rounded-lg border flex justify-center items-center text-xs px-2 py-1 cursor-pointer gap-1 bg-gradient-to-r from-white to-gray-100/50 dark:from-[#2C2F33] dark:to-[#23272A] border-neutral-600/20 dark:border-white/5",
    IconBackground: "bg-teal-500",
    Hover: "bg-gradient-to-r hover:from-[#f0f0f0] hover:to-[#e0e0e0] dark:hover:from-[#3A3D42] dark:hover:to-[#2C2F33]"
};

// Create a proxy to dynamically get theme values
let currentTheme: ThemeConfig | null = null;

// Function to update the current theme (called by ThemeProvider)
export const updateThemeClasses = (theme: ThemeConfig) => {
    currentTheme = theme;
};

// Export ClassNames as a getter that checks for EE theme overrides
export const ClassNames = new Proxy(defaultClassNames, {
    get(target, prop: keyof typeof defaultClassNames) {
        if (currentTheme?.components) {
            // Map ClassNames properties to theme properties
            const themeMapping: Record<string, keyof NonNullable<ThemeConfig['components']>> = {
                Text: 'text',
                BrandText: 'brandText',
                Button: 'button',
                IconBackground: 'icon',
            };
            
            const themeKey = themeMapping[prop as string];
            if (themeKey && currentTheme.components[themeKey]) {
                return currentTheme.components[themeKey] as string;
            }
        }
        
        return target[prop];
    }
})
