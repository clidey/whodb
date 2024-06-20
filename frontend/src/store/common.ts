import { PayloadAction, createSlice } from '@reduxjs/toolkit';

export type IIntent = "default" | "success" | "error" | "warning";

export type INotification = {
  id: string;
  message: string;
  intent?: IIntent;
}

type ICommonState = {
  notifications: INotification[];
}

const initialState: ICommonState = {
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
  },
});

export const CommonActions = commonSlice.actions;
export const commonReducers = commonSlice.reducer;