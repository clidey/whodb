import { PayloadAction, createSlice } from '@reduxjs/toolkit';

export const availableInternalModelTypes = ["Ollama"];
export const availableExternalModelTypes = ["ChatGPT"];

type IAIModel = {
  id: string;
  modelType: string;
  token?: string;
}

type IDatabaseState = {
  schema: string;
  current?: IAIModel;
  models: IAIModel[];
}

const defaultModels = availableInternalModelTypes.map(modelType => ({
  id: modelType,
  modelType,
}));

const initialState: IDatabaseState = {
  schema: "",
  models: defaultModels,
  current: defaultModels[0],
}

export const databaseSlice = createSlice({
  name: 'database',
  initialState,
  reducers: {
    setSchema: (state, action: PayloadAction<string>) => {
      state.schema = action.payload;
    },
    setCurrentModel: (state, action: PayloadAction<{ id: string }>) => {
      state.current = state.models.find(model => model.id === action.payload.id)!;
    },
    addAIModel(state, action: PayloadAction<IAIModel>) {
      state.models.push(action.payload);
    },
    removeAIModel(state, action: PayloadAction<{ id: string }>) {
      if (availableInternalModelTypes.includes(action.payload.id)) {
        return;
      }
      if (state.current?.id === action.payload.id) {
        state.current = undefined;
      }
      state.models = state.models.filter(model => model.id !== action.payload.id);
    },
  },
});

export const DatabaseActions = databaseSlice.actions;
export const databaseReducers = databaseSlice.reducer;