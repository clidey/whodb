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