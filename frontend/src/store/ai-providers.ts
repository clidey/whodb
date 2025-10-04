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

import { PayloadAction, createSlice } from '@reduxjs/toolkit';

export type IAIProvider = {
  id: string;
  name: string;
  type: string;
  baseURL?: string;
  apiKey: string;
  isEnvironmentDefined: boolean;
  isUserDefined: boolean;
  settings?: Record<string, any>;
}

type IAIProvidersState = {
  currentProviderId?: string;
  providers: IAIProvider[];
  currentModel?: string;
  models: string[];
}

const initialState: IAIProvidersState = {
  providers: [],
  models: [],
}

export const aiProvidersSlice = createSlice({
  name: 'aiProviders',
  initialState,
  reducers: {
    setProviders: (state, action: PayloadAction<IAIProvider[]>) => {
      state.providers = Array.isArray(action.payload) ? action.payload : [];
    },
    setCurrentProvider: (state, action: PayloadAction<{ id: string }>) => {
      state.currentProviderId = action.payload.id;
    },
    addProvider(state, action: PayloadAction<IAIProvider>) {
      state.providers.push(action.payload);
    },
    updateProvider(state, action: PayloadAction<IAIProvider>) {
      const index = state.providers.findIndex(p => p.id === action.payload.id);
      if (index !== -1) {
        state.providers[index] = action.payload;
      }
    },
    removeProvider(state, action: PayloadAction<{ id: string }>) {
      if (state.currentProviderId === action.payload.id) {
        state.currentProviderId = undefined;
        state.currentModel = undefined;
      }
      state.providers = state.providers.filter(provider => provider.id !== action.payload.id);
    },
    setCurrentModel(state, action: PayloadAction<IAIProvidersState["currentModel"]>) {
      state.currentModel = action.payload;
    },
    setModels: (state, action: PayloadAction<string[]>) => {
      state.models = Array.isArray(action.payload) ? action.payload : [];
    },
  },
});

export const AIProvidersActions = aiProvidersSlice.actions;
export const aiProvidersReducers = aiProvidersSlice.reducer;