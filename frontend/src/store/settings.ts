import { PayloadAction, createSlice } from '@reduxjs/toolkit';

type ISettingsState = {
    metricsEnabled: true | false;
}

const initialState: ISettingsState = {
    metricsEnabled: false,
}

export const settingsSlice = createSlice({
    name: 'settings',
    initialState,
    reducers: {
        setMetricsEnabled: (state, action: PayloadAction<ISettingsState["metricsEnabled"]>) => {
            state.metricsEnabled = action.payload;
        },
    },
});

export const SettingsActions = settingsSlice.actions;
export const settingsReducers = settingsSlice.reducer;