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

import {createSlice, PayloadAction} from '@reduxjs/toolkit';

type HealthStatus = 'healthy' | 'error' | 'unavailable' | 'unknown';

type IHealthState = {
    serverStatus: HealthStatus;
    databaseStatus: HealthStatus;
    lastChecked: number | null;
}

const initialState: IHealthState = {
    serverStatus: 'unknown',
    databaseStatus: 'unknown',
    lastChecked: null,
};

export const healthSlice = createSlice({
    name: 'health',
    initialState,
    reducers: {
        setServerStatus: (state, action: PayloadAction<HealthStatus>) => {
            state.serverStatus = action.payload;
            state.lastChecked = Date.now();
        },
        setDatabaseStatus: (state, action: PayloadAction<HealthStatus>) => {
            state.databaseStatus = action.payload;
            state.lastChecked = Date.now();
        },
        setHealthStatus: (state, action: PayloadAction<{server: HealthStatus, database: HealthStatus}>) => {
            state.serverStatus = action.payload.server;
            state.databaseStatus = action.payload.database;
            state.lastChecked = Date.now();
        },
        resetHealth: (state) => {
            state.serverStatus = 'unknown';
            state.databaseStatus = 'unknown';
            state.lastChecked = null;
        },
    },
});

export const HealthActions = healthSlice.actions;
export const healthReducers = healthSlice.reducer;
