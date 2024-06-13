import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { LoginCredentials } from '../generated/graphql';

type IAuthStatus = "pending" | "success";

export type IAuthState = {
  status: IAuthStatus;
  profiles: LoginCredentials[];
}

const initialState: IAuthState = {
  status: "pending",
  profiles: [],
};

export const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    login: (state, action: PayloadAction<LoginCredentials>) => {
      state.profiles.push(action.payload);
    },
    logout: (state) => {
      state.profiles = [];
    },
  },
})

export const AuthActions = authSlice.actions;
export const authReducers = authSlice.reducer;