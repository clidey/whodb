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