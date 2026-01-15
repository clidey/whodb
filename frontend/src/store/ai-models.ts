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

export const availableInternalModelTypes = ["Ollama"];
export const availableExternalModelTypes = ["OpenAI", "Anthropic"];

export type IAIModelType = {
  id: string;
  modelType: string;
  name?: string;
  token?: string;
  isEnvironmentDefined?: boolean;
  isGeneric?: boolean;
}

type IAIModelsState = {
  current?: IAIModelType;
  modelTypes: IAIModelType[];
  currentModel?: string;
  models: string[];
}

let defaultModelTypes = availableInternalModelTypes.map(modelType => ({
  id: modelType,
  modelType,
}));

const initialState: IAIModelsState = {
  modelTypes: defaultModelTypes,
  models: [],
}

export const aiModelsSlice = createSlice({
  name: 'aiModels',
  initialState,
  reducers: {
    setModelTypes: (state, action: PayloadAction<IAIModelType[]>) => {
      state.modelTypes = Array.isArray(action.payload) ? action.payload : defaultModelTypes;
    },
    setCurrentModelType: (state, action: PayloadAction<{ id: string }>) => {
      state.current = state.modelTypes.find(model => model.id === action.payload.id)!;
    },
    addAIModelType(state, action: PayloadAction<IAIModelType>) {
      state.modelTypes.push(action.payload);
    },
    removeAIModelType(state, action: PayloadAction<{ id: string }>) {
      if (availableInternalModelTypes.includes(action.payload.id)) {
        return;
      }
      if (state.current?.id === action.payload.id) {
        state.current = undefined;
      }
      state.modelTypes = state.modelTypes.filter(model => model.id !== action.payload.id);
    },
    setCurrentModel(state, action: PayloadAction<IAIModelsState["currentModel"]>) {
      state.currentModel = action.payload;
    },
    setModels: (state, action: PayloadAction<string[]>) => {
      state.models = Array.isArray(action.payload) ? action.payload : [];
    },
  },
});

export const AIModelsActions = aiModelsSlice.actions;
export const aiModelsReducers = aiModelsSlice.reducer;