/*
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

export interface ITourState {
    isActive: boolean;
    tourId: string | null;
    shouldStartOnLoad: boolean;
}

const initialState: ITourState = {
    isActive: false,
    tourId: null,
    shouldStartOnLoad: false,
};

export const tourSlice = createSlice({
    name: 'tour',
    initialState,
    reducers: {
        startTour: (state, action: PayloadAction<string>) => {
            state.isActive = true;
            state.tourId = action.payload;
            state.shouldStartOnLoad = false;
        },
        stopTour: (state) => {
            state.isActive = false;
            state.tourId = null;
            state.shouldStartOnLoad = false;
        },
        scheduleTourOnLoad: (state, action: PayloadAction<string>) => {
            state.shouldStartOnLoad = true;
            state.tourId = action.payload;
        },
    },
});

export const TourActions = tourSlice.actions;
export const tourReducers = tourSlice.reducer;
export default tourSlice.reducer;
