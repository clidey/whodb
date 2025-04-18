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

import { PayloadAction, createSlice } from '@reduxjs/toolkit';

export const availableInternalModelTypes = ["Ollama"];
export const availableExternalModelTypes = ["ChatGPT", "Anthropic"];

type IAIModelType = {
  id: string;
  modelType: string;
  token?: string;
}

type IDatabaseState = {
  schema: string;
  current?: IAIModelType;
  modelTypes: IAIModelType[];
}

const defaultModelTypes = availableInternalModelTypes.map(modelType => ({
  id: modelType,
  modelType,
}));

const initialState: IDatabaseState = {
  schema: "",
  modelTypes: defaultModelTypes,
}

export const databaseSlice = createSlice({
  name: 'database',
  initialState,
  reducers: {
    setSchema: (state, action: PayloadAction<string>) => {
      state.schema = action.payload;
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
  },
});

export const DatabaseActions = databaseSlice.actions;
export const databaseReducers = databaseSlice.reducer;