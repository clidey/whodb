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

import { WebStorage } from 'redux-persist';

/**
 * Custom storage implementation that handles localStorage not being available
 * or throwing errors during initialization (e.g., on first startup)
 */
export const createSafeStorage = (): WebStorage => {
  const noopStorage: WebStorage = {
    getItem: (_key: string) => Promise.resolve(null),
    setItem: (_key: string, _value: string) => Promise.resolve(),
    removeItem: (_key: string) => Promise.resolve(),
  };

  // Check if localStorage is available
  const hasLocalStorage = (() => {
    try {
      const testKey = '__whodb_storage_test__';
      if (typeof window === 'undefined' || !window.localStorage) {
        return false;
      }
      window.localStorage.setItem(testKey, 'test');
      window.localStorage.removeItem(testKey);
      return true;
    } catch {
      return false;
    }
  })();

  if (!hasLocalStorage) {
    console.warn('localStorage is not available, using noop storage');
    return noopStorage;
  }

  // Return a wrapped localStorage that handles errors gracefully
  return {
    getItem: async (key: string) => {
      try {
        const item = window.localStorage.getItem(key);
        return item;
      } catch (error) {
        console.warn(`Error getting item from localStorage for key "${key}"`, error);
        return null;
      }
    },
    setItem: async (key: string, value: string) => {
      try {
        window.localStorage.setItem(key, value);
      } catch (error) {
        console.warn(`Error setting item in localStorage for key "${key}"`, error);
      }
    },
    removeItem: async (key: string) => {
      try {
        window.localStorage.removeItem(key);
      } catch (error) {
        console.warn(`Error removing item from localStorage for key "${key}"`, error);
      }
    },
  };
};

export default createSafeStorage();