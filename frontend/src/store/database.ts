import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { v4 } from 'uuid';

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

const initialState: IDatabaseState = {
  schema: "",
  models: [],
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
    addAIModel(state, action: PayloadAction<Omit<IAIModel, "id">>) {
      state.models.push({
        id: v4(),
        ...action.payload,
      });
    },
    removeAIModel(state, action: PayloadAction<{ id: string }>) {
      if (availableInternalModelTypes.includes(action.payload.id)) {
        return;
      }
      state.models = state.models.filter(model => model.id !== action.payload.id);
    },
  },
});

export const DatabaseActions = databaseSlice.actions;
export const databaseReducers = databaseSlice.reducer;