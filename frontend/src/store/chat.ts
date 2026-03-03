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

import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { AiChatMessage } from '@graphql';


export type IChatMessage = AiChatMessage & {
    isUserInput?: boolean;
    isStreaming?: boolean;
    id?: number;
};

export type ChatSession = {
    id: string;
    name: string;
    messages: IChatMessage[];
    createdAt: Date;
};

export type IChatState = {
    // Legacy single chat support (for backward compatibility during migration)
    chats: IChatMessage[];
    // New multi-session support
    sessions: ChatSession[];
    activeSessionId: string | null;
}

const initialState: IChatState = {
    chats: [],
    sessions: [],
    activeSessionId: null,
}

export const houdiniSlice = createSlice({
  name: 'chats',
  initialState,
  reducers: {
    // Legacy actions (for backward compatibility)
    addChatMessage: (state, action: PayloadAction<IChatMessage>) => {
        // If using sessions, add to active session
        if (state.sessions.length > 0 && state.activeSessionId) {
            const session = state.sessions.find(s => s.id === state.activeSessionId);
            if (session) {
                session.messages.push(action.payload);
            }
        } else {
            // Fallback to legacy behavior
            state.chats.push(action.payload);
        }
    },
    updateChatMessage: (state, action: PayloadAction<{ id: number; Text: string }>) => {
        // Update in active session if using sessions
        if (state.sessions.length > 0 && state.activeSessionId) {
            const session = state.sessions.find(s => s.id === state.activeSessionId);
            if (session) {
                const message = session.messages.find(chat => chat.id === action.payload.id);
                if (message) {
                    message.Text = action.payload.Text;
                }
            }
        } else {
            // Fallback to legacy behavior
            const message = state.chats.find(chat => chat.id === action.payload.id);
            if (message) {
                message.Text = action.payload.Text;
            }
        }
    },
    completeStreamingMessage: (state, action: PayloadAction<{ id: number; message: Partial<IChatMessage> }>) => {
        // Update in active session if using sessions
        if (state.sessions.length > 0 && state.activeSessionId) {
            const session = state.sessions.find(s => s.id === state.activeSessionId);
            if (session) {
                const message = session.messages.find(chat => chat.id === action.payload.id);
                if (message) {
                    Object.assign(message, action.payload.message);
                    message.isStreaming = false;
                }
            }
        } else {
            // Fallback to legacy behavior
            const message = state.chats.find(chat => chat.id === action.payload.id);
            if (message) {
                Object.assign(message, action.payload.message);
                message.isStreaming = false;
            }
        }
    },
    removeChatMessage: (state, action: PayloadAction<number>) => {
        // Remove from active session if using sessions
        if (state.sessions.length > 0 && state.activeSessionId) {
            const session = state.sessions.find(s => s.id === state.activeSessionId);
            if (session) {
                session.messages = session.messages.filter(chat => chat.id !== action.payload);
            }
        } else {
            // Fallback to legacy behavior
            state.chats = state.chats.filter(chat => chat.id !== action.payload);
        }
    },
    clear: (state) => {
        // Clear active session if using sessions
        if (state.sessions.length > 0 && state.activeSessionId) {
            const session = state.sessions.find(s => s.id === state.activeSessionId);
            if (session) {
                session.messages = [];
            }
        } else {
            // Fallback to legacy behavior
            state.chats = [];
        }
    },

    // New session management actions
    initializeChatSessions: (state) => {
        if (state.sessions.length === 0) {
            const newId = crypto.randomUUID();
            const newSession: ChatSession = {
                id: newId,
                name: "Chat 1",
                messages: [],
                createdAt: new Date()
            };
            state.sessions = [newSession];
            state.activeSessionId = newId;
        }

        // Ensure we have an active session
        if (!state.activeSessionId && state.sessions.length > 0) {
            state.activeSessionId = state.sessions[0].id;
        }
    },
    addChatSession: (state, action: PayloadAction<{ name?: string }>) => {
        const newId = crypto.randomUUID();
        const newSession: ChatSession = {
            id: newId,
            name: action.payload.name || `Chat ${state.sessions.length + 1}`,
            messages: [],
            createdAt: new Date()
        };
        // Add new session at the top (beginning) of the list
        state.sessions.unshift(newSession);
        state.activeSessionId = newId;
    },
    deleteChatSession: (state, action: PayloadAction<{ sessionId: string }>) => {
        if (state.sessions.length <= 1) return;

        const sessionIndex = state.sessions.findIndex(session => session.id === action.payload.sessionId);
        if (sessionIndex === -1) return;

        // Remove the session
        state.sessions.splice(sessionIndex, 1);

        // If we deleted the active session, switch to the first remaining session
        if (state.activeSessionId === action.payload.sessionId) {
            state.activeSessionId = state.sessions[0]?.id || null;
        }
    },
    setActiveSession: (state, action: PayloadAction<{ sessionId: string }>) => {
        const session = state.sessions.find(s => s.id === action.payload.sessionId);
        if (session) {
            state.activeSessionId = action.payload.sessionId;
        }
    },
    updateSessionName: (state, action: PayloadAction<{ sessionId: string; name: string }>) => {
        const session = state.sessions.find(s => s.id === action.payload.sessionId);
        if (session) {
            session.name = action.payload.name;
        }
    },
    clearAllChatSessions: (state) => {
        state.sessions = [];
        state.activeSessionId = null;
    },
  },
});

export const HoudiniActions = houdiniSlice.actions;
export const houdiniReducers = houdiniSlice.reducer;