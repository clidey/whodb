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