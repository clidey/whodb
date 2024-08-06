import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { v4 } from 'uuid';
import { LoginCredentials } from '../generated/graphql';

export type LocalLoginProfile = (LoginCredentials & {Id: string, Saved?: boolean});

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
      state.profiles.push(profile as LocalLoginProfile);
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