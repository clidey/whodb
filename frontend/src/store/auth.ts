/*
 * Copyright 2026 Clidey, Inc.
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

import type { PayloadAction} from '@reduxjs/toolkit';
import {createSlice} from '@reduxjs/toolkit';
import { v4 as uuidv4 } from 'uuid';

export interface SourceCredentialValue {
  Key: string;
  Value: string;
  Extra?: SourceCredentialValue[];
}

export interface SourceLoginPayload {
  Id?: string;
  SourceType: string;
  Values: SourceCredentialValue[];
  AccessToken?: string;
  Saved?: boolean;
  IsEnvironmentDefined?: boolean;
  SSLConfigured?: boolean;
  DisplayName?: string;
  Source?: string;
}

export interface LocalLoginProfile extends SourceLoginPayload {
  Id: string;
  Type: string;
  Hostname: string;
  Database: string;
  Username: string;
  Password: string;
  Advanced: SourceCredentialValue[];
}

export type SSLStatus = {
  IsEnabled: boolean;
  Mode: string;
};

export type IAuthState = {
  status: "logged-in" | "unauthorized";
  current?: LocalLoginProfile;
  profiles: LocalLoginProfile[];
  sslStatus?: SSLStatus;
  isEmbedded: boolean;
}

const initialState: IAuthState = {
  status: "unauthorized",
  profiles: [],
  sslStatus: undefined,
  isEmbedded: false,
};

const coreValueKeys = new Set(["hostname", "database", "username", "password"]);

function mapValues(values: SourceCredentialValue[]): Record<string, string> {
  return values.reduce<Record<string, string>>((acc, value) => {
    acc[value.Key] = value.Value;
    return acc;
  }, {});
}

function buildValues(valuesMap: Record<string, string>, advanced: SourceCredentialValue[]): SourceCredentialValue[] {
  const values: SourceCredentialValue[] = [];

  if (valuesMap.Hostname) {
    values.push({ Key: "Hostname", Value: valuesMap.Hostname });
  }
  if (valuesMap.Database) {
    values.push({ Key: "Database", Value: valuesMap.Database });
  }
  if (valuesMap.Username) {
    values.push({ Key: "Username", Value: valuesMap.Username });
  }
  if (valuesMap.Password) {
    values.push({ Key: "Password", Value: valuesMap.Password });
  }

  return values.concat(advanced);
}

function normalizeProfile(payload: SourceLoginPayload | LocalLoginProfile): LocalLoginProfile {
  if ("Type" in payload && "Advanced" in payload) {
    return payload;
  }

  const valuesMap = mapValues(payload.Values);
  const advanced = payload.Values.filter(value => !coreValueKeys.has(value.Key.toLowerCase()));

  return {
    ...payload,
    Id: payload.Id ?? uuidv4(),
    Type: payload.SourceType,
    Hostname: valuesMap.Hostname ?? "",
    Database: valuesMap.Database ?? "",
    Username: valuesMap.Username ?? "",
    Password: valuesMap.Password ?? "",
    Advanced: advanced,
  };
}

export const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    login: (state, action: PayloadAction<SourceLoginPayload | LocalLoginProfile>) => {
      const profile = normalizeProfile(action.payload);
      state.current = profile;

      const existingProfileIndex = state.profiles.findIndex(p => p.Id === profile.Id);
      if (existingProfileIndex >= 0) {
        state.profiles[existingProfileIndex] = profile;
      } else {
        state.profiles.push(profile);
      }

      state.status = "logged-in";
    },
    switch: (state, action: PayloadAction<{id: string}>) => {
      state.current = state.profiles.find(profile => profile.Id === action.payload.id);
      state.sslStatus = undefined;
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
      state.sslStatus = undefined;
    },
    setSSLStatus: (state, action: PayloadAction<SSLStatus | undefined>) => {
      state.sslStatus = action.payload;
    },
    setLoginProfileDatabase: (state, action: PayloadAction<{ id: string, database: string }>) => {
      const profile = state.profiles.find(candidate => candidate.Id === action.payload.id);
      if (profile == null) {
        return;
      }

      const valuesMap = mapValues(profile.Values);
      valuesMap.Database = action.payload.database;
      const values = buildValues(valuesMap, profile.Advanced);

      profile.Database = action.payload.database;
      profile.Values = values;

      if (state.current?.Id === profile.Id) {
        state.current.Database = action.payload.database;
        state.current.Values = values;
      }
    },
    setEmbedded: (state, action: PayloadAction<boolean>) => {
      state.isEmbedded = action.payload;
    }
  },
});

export const AuthActions = authSlice.actions;
export const authReducers = authSlice.reducer;
