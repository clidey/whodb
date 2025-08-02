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

type ISettingsState = {
    metricsEnabled: true | false;
    performanceMonitoringEnabled: true | false;
    performanceMetricsConfig: {
        query_latency: boolean;
        query_count: boolean;
        connection_count: boolean;
        error_count: boolean;
        cpu_usage: boolean;
        memory_usage: boolean;
        disk_io: boolean;
        cache_hit_ratio: boolean;
        transaction_count: boolean;
        lock_wait_time: boolean;
    };
}

const initialState: ISettingsState = {
    metricsEnabled: true,
    performanceMonitoringEnabled: false,
    performanceMetricsConfig: {
        query_latency: true,
        query_count: true,
        connection_count: true,
        error_count: true,
        cpu_usage: false,
        memory_usage: false,
        disk_io: false,
        cache_hit_ratio: false,
        transaction_count: false,
        lock_wait_time: false,
    },
}

export const settingsSlice = createSlice({
    name: 'settings',
    initialState,
    reducers: {
        setMetricsEnabled: (state, action: PayloadAction<ISettingsState["metricsEnabled"]>) => {
            state.metricsEnabled = action.payload;
        },
        setPerformanceMonitoringEnabled: (state, action: PayloadAction<ISettingsState["performanceMonitoringEnabled"]>) => {
            state.performanceMonitoringEnabled = action.payload;
        },
        setPerformanceMetricsConfig: (state, action: PayloadAction<Partial<ISettingsState["performanceMetricsConfig"]>>) => {
            state.performanceMetricsConfig = {
                ...state.performanceMetricsConfig,
                ...action.payload,
            };
        },
    },
});

export const SettingsActions = settingsSlice.actions;
export const settingsReducers = settingsSlice.reducer;
