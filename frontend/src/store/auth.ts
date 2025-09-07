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
import { v4 } from 'uuid';
import { LoginCredentials } from '@graphql';

export type LocalLoginProfile = (LoginCredentials & {Id: string, Saved?: boolean, IsEnvironmentDefined?: boolean});

export type IAuthState = {
  status: "logged-in" | "unauthorized";
  current?: LocalLoginProfile;
  profiles: LocalLoginProfile[];
}

const initialState: IAuthState = {
  status: "unauthorized",
  profiles: [],
};

export const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    login: (state, action: PayloadAction<LoginCredentials | LocalLoginProfile>) => {
      const profile = action.payload.Id != null ? action.payload as LocalLoginProfile : {
        Id: v4(),
        ...(action.payload as LoginCredentials),
      }
      state.current = profile as LocalLoginProfile;
      
      // Check if profile already exists to prevent duplicates
      const existingProfileIndex = state.profiles.findIndex(p => p.Id === profile.Id);
      if (existingProfileIndex >= 0) {
        // Update existing profile instead of adding duplicate
        state.profiles[existingProfileIndex] = profile as LocalLoginProfile;
      } else {
        // Add new profile
        state.profiles.push(profile as LocalLoginProfile);
      }
      
      state.status = "logged-in";
    },
    switch: (state, action: PayloadAction<{id: string}>) => {
      state.current = state.profiles.find(profile => profile.Id === action.payload.id);
    },
    remove: (state, action: PayloadAction<{id: string}>) => {
      state.profiles = state.profiles.filter(profile => profile.Id !== action.payload.id);
      if (state.current?.Id === action.payload.id) {
        state.current = undefined;
      }
    },
    logout: (state) => {
      state.profiles = [];
      state.current = undefined;
      state.status = "unauthorized";
    },
    setLoginProfileDatabase: (state, action: PayloadAction<{ id: string, database: string }>) => {
      const profile = state.profiles.find(profile => profile.Id === action.payload.id);
      if (profile == null) {
        return;
      }
      if (state.current?.Id === profile.Id) {
        state.current.Database = action.payload.database;
      }
      profile.Database = action.payload.database;
    }
  },
});

export const AuthActions = authSlice.actions;
export const authReducers = authSlice.reducer;