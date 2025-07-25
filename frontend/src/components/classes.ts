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
    Background: "bg-[#fbfaf8] dark:bg-[#121212]",
    Text: "text-[#333333] dark:text-[#E0E0E0]",
    BrandText: "text-[#ca6f1e] dark:text-[#ca6f1e]",
    Card: "bg-[#fbfaf8] h-[200px] w-[200px] rounded-3xl shadow-xs border border-neutral-600/5 p-4 flex flex-col justify-between relative transition-all duration-300 overflow-y-auto dark:bg-[#252525] dark:border-white/5",
    Button: "rounded-lg border flex justify-center items-center text-xs px-2 py-1 cursor-pointer gap-1 bg-gradient-to-r from-white to-gray-100/50 dark:from-[#2C2F33] dark:to-[#23272A] border-neutral-600/20 dark:border-white/5",
    Dropdown: "group/dropdown flex gap-1 justify-between items-center border border-neutral-600/20 rounded-lg w-full p-1 h-[34px] px-2 dark:bg-[#2C2F33] dark:border-white/5",
    DropdownPanel: "absolute z-10 divide-y rounded-lg shadow-sm bg-white py-1 border border-gray-200 overflow-y-auto max-h-40 dark:bg-[#2C2F33] dark:border-white/20",
    IconBackground: "bg-teal-500",
    SidebarItem: "cursor-default text-md inline-flex gap-2 transition-all hover:gap-2 relative w-full py-4 rounded-md dark:border-white/5",
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
                Card: 'card',
                Button: 'button',
                Dropdown: 'dropdown',
                DropdownPanel: 'dropdownPanel',
                IconBackground: 'icon',
            };
            
            const themeKey = themeMapping[prop as string];
            if (themeKey && currentTheme.components[themeKey]) {
                return currentTheme.components[themeKey] as string;
            }
        }
        
        // Check layout properties
        if (currentTheme?.layout) {
            const layoutMapping: Record<string, keyof NonNullable<ThemeConfig['layout']>> = {
                Background: 'background',
                SidebarItem: 'sidebarItem',
            };
            
            const layoutKey = layoutMapping[prop as string];
            if (layoutKey && currentTheme.layout[layoutKey]) {
                return currentTheme.layout[layoutKey] as string;
            }
        }
        
        return target[prop];
    }
})
