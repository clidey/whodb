/*
 * Copyright 2026 Clidey, Inc.
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

import { RootState } from '../store';

// Font size mappings
const fontSizeMap = {
  small: {
    '--font-size-xs': '0.75rem',
    '--font-size-sm': '0.875rem',
    '--font-size-base': '0.875rem',
    '--font-size-lg': '1rem',
    '--font-size-xl': '1.125rem',
    '--font-size-2xl': '1.25rem',
    '--font-size-3xl': '1.5rem',
  },
  medium: {
    '--font-size-xs': '0.75rem',
    '--font-size-sm': '0.875rem',
    '--font-size-base': '1rem',
    '--font-size-lg': '1.125rem',
    '--font-size-xl': '1.25rem',
    '--font-size-2xl': '1.5rem',
    '--font-size-3xl': '1.875rem',
  },
  large: {
    '--font-size-xs': '0.875rem',
    '--font-size-sm': '1rem',
    '--font-size-base': '1.125rem',
    '--font-size-lg': '1.25rem',
    '--font-size-xl': '1.375rem',
    '--font-size-2xl': '1.625rem',
    '--font-size-3xl': '2rem',
  },
};

// Border radius mappings
const borderRadiusMap = {
  none: {
    '--radius': '0px',
    '--radius-sm': '0px',
    '--radius-md': '0px',
    '--radius-lg': '0px',
    '--radius-xl': '0px',
  },
  small: {
    '--radius': '0.25rem',
    '--radius-sm': '0.125rem',
    '--radius-md': '0.25rem',
    '--radius-lg': '0.375rem',
    '--radius-xl': '0.5rem',
  },
  medium: {
    '--radius': '0.625rem',
    '--radius-sm': '0.375rem',
    '--radius-md': '0.5rem',
    '--radius-lg': '0.625rem',
    '--radius-xl': '0.75rem',
  },
  large: {
    '--radius': '1rem',
    '--radius-sm': '0.5rem',
    '--radius-md': '0.75rem',
    '--radius-lg': '1rem',
    '--radius-xl': '1.25rem',
  },
};

// Spacing mappings
const spacingMap = {
  compact: {
    '--spacing-xs': '0.25rem',
    '--spacing-sm': '0.5rem',
    '--spacing-md': '0.75rem',
    '--spacing-lg': '1rem',
    '--spacing-xl': '1.25rem',
    '--spacing-2xl': '1.5rem',
    '--spacing-3xl': '2rem',
  },
  comfortable: {
    '--spacing-xs': '0.5rem',
    '--spacing-sm': '0.75rem',
    '--spacing-md': '1rem',
    '--spacing-lg': '1.5rem',
    '--spacing-xl': '2rem',
    '--spacing-2xl': '2.5rem',
    '--spacing-3xl': '3rem',
  },
  spacious: {
    '--spacing-xs': '0.75rem',
    '--spacing-sm': '1rem',
    '--spacing-md': '1.5rem',
    '--spacing-lg': '2rem',
    '--spacing-xl': '2.5rem',
    '--spacing-2xl': '3rem',
    '--spacing-3xl': '4rem',
  },
};


/**
 * Applies UI customization settings to CSS variables
 */
export const applyUICustomization = (settings: RootState['settings']) => {
  const root = document.documentElement;
  
  // Apply font size settings
  const fontSizeVars = fontSizeMap[settings.fontSize];
  Object.entries(fontSizeVars).forEach(([property, value]) => {
    root.style.setProperty(property, value);
  });
  
  // Apply border radius settings
  const borderRadiusVars = borderRadiusMap[settings.borderRadius];
  Object.entries(borderRadiusVars).forEach(([property, value]) => {
    root.style.setProperty(property, value);
  });
  
  // Apply spacing settings
  const spacingVars = spacingMap[settings.spacing];
  Object.entries(spacingVars).forEach(([property, value]) => {
    root.style.setProperty(property, value);
  });
  
};

