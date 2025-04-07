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
import { AiChatMessage } from '../generated/graphql';


type IChatMessage = AiChatMessage & {
    isUserInput?: boolean;
};

type IChatState = {
    chats: IChatMessage[];
}

const initialState: IChatState = {
    chats: [],
}

export const houdiniSlice = createSlice({
  name: 'chats',
  initialState,
  reducers: {
    addChatMessage: (state, action: PayloadAction<IChatMessage>) => {
        state.chats.push(action.payload);
    },
    clear: (state) => {
        state.chats = [];
    },
  },
});

export const HoudiniActions = houdiniSlice.actions;
export const houdiniReducers = houdiniSlice.reducer;