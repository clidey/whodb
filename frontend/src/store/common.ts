import { PayloadAction, createSlice } from '@reduxjs/toolkit';

export type IIntent = "default" | "success" | "error" | "warning";

export type INotification = {
  id: string;
  message: string;
  intent?: IIntent;
}

type ICommonState = {
  schema: string;
  notifications: INotification[];
}

const initialState: ICommonState = {
  schema: "",
  notifications: [],
}

export const commonSlice = createSlice({
  name: 'common',
  initialState,
  reducers: {
    addNotifications: (state, action: PayloadAction<INotification>) => {
      state.notifications.push(action.payload);
    },
    removeNotifications: (state, action: PayloadAction<INotification>) => {
      state.notifications = state.notifications.filter(notification => notification.id !== action.payload.id);
    },
    setSchema: (state, action: PayloadAction<string>) => {
      state.schema = action.payload;
    },
  },
});

export const CommonActions = commonSlice.actions;
export const commonReducers = commonSlice.reducer;