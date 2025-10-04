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

import { combineReducers, configureStore } from '@reduxjs/toolkit';
import { persistReducer, persistStore, createTransform } from 'redux-persist';
import storage from 'redux-persist/lib/storage';
import { authReducers } from './auth';
import { databaseReducers } from './database';
import { settingsReducers } from "./settings";
import { houdiniReducers } from './chat';
import { aiProvidersReducers } from './ai-providers';
import { scratchpadReducers, IScratchpadState } from './scratchpad';
import { runMigrations } from './migrations';

// Run migrations before initializing the store
runMigrations();

// Clear any corrupted scratchpad data on startup to prevent date serialization issues
if (typeof window !== 'undefined') {
  try {
    const scratchpadData = localStorage.getItem('persist:scratchpad');
    if (scratchpadData) {
      const parsed = JSON.parse(scratchpadData);
      if (parsed && parsed.cells) {
        // Check if any cells have invalid date strings in history
        const hasInvalidDates = Object.values(parsed.cells).some((cell: any) => {
          if (cell && cell.history && Array.isArray(cell.history)) {
            return cell.history.some((item: any) => 
              item.date && typeof item.date === 'string' && isNaN(new Date(item.date).getTime())
            );
          }
          return false;
        });
        
        if (hasInvalidDates) {
          console.warn('Clearing corrupted scratchpad data due to invalid dates');
          localStorage.removeItem('persist:scratchpad');
        }
      }
    }
  } catch (error) {
    console.warn('Error checking scratchpad data, clearing it:', error);
    localStorage.removeItem('persist:scratchpad');
  }
}

// Transform function to handle date serialization/deserialization for scratchpad
const scratchpadTransform = createTransform(
  // Transform state on its way to being serialized and persisted
  (inboundState: IScratchpadState) => {
    return inboundState;
  },
  // Transform state being rehydrated
  (outboundState: any) => {
    if (!outboundState || !outboundState.cells) {
      return outboundState;
    }

    // Convert date strings back to Date objects in cell history
    const transformedCells: Record<string, any> = {};
    Object.keys(outboundState.cells).forEach(cellId => {
      const cell = outboundState.cells[cellId];
      if (cell && cell.history && Array.isArray(cell.history)) {
        transformedCells[cellId] = {
          ...cell,
          history: cell.history.map((historyItem: any) => {
            let date = historyItem.date;
            // Handle various date formats
            if (typeof date === 'string') {
              date = new Date(date);
            } else if (!(date instanceof Date)) {
              date = new Date();
            }
            // Ensure the date is valid
            if (isNaN(date.getTime())) {
              date = new Date();
            }
            return {
              ...historyItem,
              date
            };
          })
        };
      } else {
        transformedCells[cellId] = cell;
      }
    });

    return {
      ...outboundState,
      cells: transformedCells
    };
  },
  { whitelist: ['scratchpad'] }
);

const persistedReducer = combineReducers({
  auth: persistReducer({ key: "auth", storage, }, authReducers),
  database: persistReducer({ key: "database", storage, }, databaseReducers),
  settings: persistReducer({ key: "settings", storage }, settingsReducers),
  houdini: persistReducer({ key: "houdini", storage }, houdiniReducers),
  aiProviders: persistReducer({ key: "aiProviders", storage }, aiProvidersReducers),
  scratchpad: persistReducer({
    key: "scratchpad",
    storage,
    transforms: [scratchpadTransform]
  }, scratchpadReducers),
});

export const reduxStore = configureStore({
  reducer: persistedReducer,
  middleware: (getDefaultMiddleware) => {
    return getDefaultMiddleware({
      serializableCheck: false,
    });
  },
});

export const reduxStorePersistor = persistStore(reduxStore);

export type RootState = ReturnType<typeof reduxStore.getState>;
export type AppDispatch = typeof reduxStore.dispatch;