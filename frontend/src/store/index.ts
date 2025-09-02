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
import { persistReducer, persistStore } from 'redux-persist';
import storage from 'redux-persist/lib/storage';
import { authReducers } from './auth';
import { databaseReducers } from './database';
import { settingsReducers } from "./settings";
import { houdiniReducers } from './chat';
import { aiModelsReducers } from './ai-models';
import { runMigrations } from './migrations';

// Run migrations before initializing the store
runMigrations();

const persistedReducer = combineReducers({
  auth: persistReducer({ key: "auth", storage, }, authReducers),
  database: persistReducer({ key: "database", storage, }, databaseReducers),
  settings: persistReducer({ key: "settings", storage }, settingsReducers),
  houdini: persistReducer({ key: "houdini", storage }, houdiniReducers),
  aiModels: persistReducer({ key: "aiModels", storage }, aiModelsReducers),
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