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
import {SortCondition, WhereCondition} from '@graphql';

type ExploreConditionEntry = {
    whereCondition?: WhereCondition;
    sortConditions: SortCondition[];
};

type IExploreConditionsState = {
    conditions: Record<string, ExploreConditionEntry>;
};

const initialState: IExploreConditionsState = {
    conditions: {},
};

export const exploreConditionsSlice = createSlice({
    name: 'exploreConditions',
    initialState,
    reducers: {
        setExploreConditions: (state, action: PayloadAction<{ key: string } & ExploreConditionEntry>) => {
            const { key, whereCondition, sortConditions } = action.payload;
            state.conditions[key] = { whereCondition, sortConditions };
        },
    },
});

export const ExploreConditionsActions = exploreConditionsSlice.actions;
export const exploreConditionsReducers = exploreConditionsSlice.reducer;
