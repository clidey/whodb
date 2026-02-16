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
import {eeSettingsDefaults} from '../config/ee-imports';

const ANALYTICS_CONSENT_KEY = 'whodb.analytics.consent';

type ISettingsState = {
    metricsEnabled: boolean;
    cloudProvidersEnabled: boolean;
    storageUnitView: 'list' | 'card';
    fontSize: 'small' | 'medium' | 'large';
    borderRadius: 'none' | 'small' | 'medium' | 'large';
    spacing: 'compact' | 'comfortable' | 'spacious';
    whereConditionMode: 'popover' | 'sheet';
    defaultPageSize: number;
    language: 'en' | 'es' | 'de' | 'fr';
    databaseSchemaTerminology: 'database' | 'schema';
    disableAnimations: boolean;
}

const getInitialMetricsEnabled = (): boolean => {
    if (typeof window === 'undefined') {
        return true;
    }

    const consent = window.localStorage.getItem(ANALYTICS_CONSENT_KEY);
    if (consent === 'denied') {
        return false;
    }
    return true;
};

const getInitialState = (): ISettingsState => {
    return {
        metricsEnabled: getInitialMetricsEnabled(),
        cloudProvidersEnabled: false,
        storageUnitView: 'card',
        fontSize: 'medium',
        borderRadius: 'medium',
        spacing: 'comfortable',
        // Use EE default if available, otherwise default to 'popover'
        whereConditionMode: eeSettingsDefaults.whereConditionMode ?? 'popover',
        defaultPageSize: 100,
        language: 'en',
        databaseSchemaTerminology: 'database',  // Default to "Database" label for databases where database=schema
        // Use EE default if available, otherwise default to false (animations enabled)
        disableAnimations: eeSettingsDefaults.disableAnimations ?? false,
    };
};

const initialState = getInitialState();

export const settingsSlice = createSlice({
    name: 'settings',
    initialState,
    reducers: {
        setMetricsEnabled: (state, action: PayloadAction<ISettingsState["metricsEnabled"]>) => {
            state.metricsEnabled = action.payload;
        },
        setCloudProvidersEnabled: (state, action: PayloadAction<ISettingsState["cloudProvidersEnabled"]>) => {
            state.cloudProvidersEnabled = action.payload;
        },
        setStorageUnitView: (state, action: PayloadAction<ISettingsState["storageUnitView"]>) => {
            state.storageUnitView = action.payload;
        },
        // UI Customization actions
        setFontSize: (state, action: PayloadAction<ISettingsState["fontSize"]>) => {
            state.fontSize = action.payload;
        },
        setBorderRadius: (state, action: PayloadAction<ISettingsState["borderRadius"]>) => {
            state.borderRadius = action.payload;
        },
        setSpacing: (state, action: PayloadAction<ISettingsState["spacing"]>) => {
            state.spacing = action.payload;
        },
        setWhereConditionMode: (state, action: PayloadAction<ISettingsState["whereConditionMode"]>) => {
            state.whereConditionMode = action.payload;
        },
        setDefaultPageSize: (state, action: PayloadAction<ISettingsState["defaultPageSize"]>) => {
            state.defaultPageSize = action.payload;
        },
        setLanguage: (state, action: PayloadAction<ISettingsState["language"]>) => {
            state.language = action.payload;
        },
        setDatabaseSchemaTerminology: (state, action: PayloadAction<ISettingsState["databaseSchemaTerminology"]>) => {
            state.databaseSchemaTerminology = action.payload;
        },
        setDisableAnimations: (state, action: PayloadAction<ISettingsState["disableAnimations"]>) => {
            state.disableAnimations = action.payload;
        },
    },
});

export const SettingsActions = settingsSlice.actions;
export const settingsReducers = settingsSlice.reducer;
