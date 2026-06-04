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

import { useEffect } from 'react';
import { useAppSelector } from '../store/hooks';
import { applyUICustomization } from '../utils/theme-customization';

/**
 * Hook that applies UI customization settings to CSS variables
 * This should be used at the root level of the application
 */
export const useThemeCustomization = () => {
  const fontSize = useAppSelector(state => state.settings.fontSize);
  const borderRadius = useAppSelector(state => state.settings.borderRadius);
  const spacing = useAppSelector(state => state.settings.spacing);
  const disableAnimations = useAppSelector(state => state.settings.disableAnimations);

  useEffect(() => {
    applyUICustomization({ fontSize, borderRadius, spacing });
  }, [fontSize, borderRadius, spacing]);

  useEffect(() => {
    if (disableAnimations) {
      document.body.classList.add('disable-animations');
    } else {
      document.body.classList.remove('disable-animations');
    }
  }, [disableAnimations]);

};
