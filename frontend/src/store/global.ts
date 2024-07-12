import { PayloadAction, createSlice } from '@reduxjs/toolkit';

type IGlobalState = {
  theme: "light" | "dark";
}

const initialState: IGlobalState = {
    theme: "dark",
}

export const globalSlice = createSlice({
  name: 'global',
  initialState,
  reducers: {
    setTheme: (state, action: PayloadAction<IGlobalState["theme"]>) => {
        state.theme = action.payload;
    },
  },
});

export const GlobalActions = globalSlice.actions;
export const globalReducers = globalSlice.reducer;