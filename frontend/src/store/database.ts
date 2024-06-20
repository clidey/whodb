import { PayloadAction, createSlice } from '@reduxjs/toolkit';

type IDatabaseState = {
  schema: string;
}

const initialState: IDatabaseState = {
  schema: "",
}

export const databaseSlice = createSlice({
  name: 'database',
  initialState,
  reducers: {
    setSchema: (state, action: PayloadAction<string>) => {
      state.schema = action.payload;
    },
  },
});

export const DatabaseActions = databaseSlice.actions;
export const databaseReducers = databaseSlice.reducer;