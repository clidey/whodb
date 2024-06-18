import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { v4 } from 'uuid';
import { LoginCredentials } from '../generated/graphql';

export type LoginProfile = (LoginCredentials & {id: string});

export type IAuthState = {
  status: "logged-in" | "unauthorized";
  current?: LoginProfile;
  profiles: LoginProfile[];
}

const initialState: IAuthState = {
  status: "unauthorized",
  profiles: [],
};

export const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    login: (state, action: PayloadAction<LoginCredentials>) => {
      const profile = {
        id: v4(),
        ...action.payload,
      };
      state.current = profile;
      state.profiles.push(profile);
      state.status = "logged-in";
    },
    switch: (state, action: PayloadAction<{id: string}>) => {
      state.current = state.profiles.find(profile => profile.id === action.payload.id);
    },
    remove: (state, action: PayloadAction<{id: string}>) => {
      state.profiles = state.profiles.filter(profile => profile.id !== action.payload.id);
      if (state.current?.id === action.payload.id) {
        state.current = undefined;
      }
    },
    logout: (state) => {
      state.profiles = [];
      state.current = undefined;
      state.status = "unauthorized";
    },
  },
});

export const AuthActions = authSlice.actions;
export const authReducers = authSlice.reducer;