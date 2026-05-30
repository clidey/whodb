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
import { v4 as uuidv4 } from 'uuid';


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
    updatedAt?: Date;
    projectId?: string;
    sourceId?: string;
    status?: string;
    activeRunId?: string;
    lastEventSequence?: number;
    modelType?: string;
    providerId?: string;
    model?: string;
    autoScrollEnabled?: boolean;
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
    addChatMessage: (state, action: PayloadAction<IChatMessage & { sessionId?: string; insertAfterId?: number }>) => {
        const targetId = action.payload.sessionId ?? state.activeSessionId;
        const insertMessage = (messages: IChatMessage[]) => {
            const { sessionId: _, insertAfterId, ...msg } = action.payload;
            if (insertAfterId == null) {
                messages.push(msg);
                return;
            }
            const index = messages.findIndex(chat => chat.id === insertAfterId);
            if (index === -1) {
                messages.push(msg);
                return;
            }
            messages.splice(index + 1, 0, msg);
        };

        if (state.sessions.length > 0 && targetId) {
            const session = state.sessions.find(s => s.id === targetId);
            if (session) {
                insertMessage(session.messages);
            }
        } else {
            insertMessage(state.chats);
        }
    },
    updateChatMessage: (state, action: PayloadAction<{ id: number; Text: string; sessionId?: string }>) => {
        const targetId = action.payload.sessionId ?? state.activeSessionId;
        if (state.sessions.length > 0 && targetId) {
            const session = state.sessions.find(s => s.id === targetId);
            if (session) {
                const message = session.messages.find(chat => chat.id === action.payload.id);
                if (message) {
                    message.Text = action.payload.Text;
                }
            }
        } else {
            const message = state.chats.find(chat => chat.id === action.payload.id);
            if (message) {
                message.Text = action.payload.Text;
            }
        }
    },
    completeStreamingMessage: (state, action: PayloadAction<{ id: number; message: Partial<IChatMessage>; sessionId?: string }>) => {
        const targetId = action.payload.sessionId ?? state.activeSessionId;
        if (state.sessions.length > 0 && targetId) {
            const session = state.sessions.find(s => s.id === targetId);
            if (session) {
                const message = session.messages.find(chat => chat.id === action.payload.id);
                if (message) {
                    Object.assign(message, action.payload.message);
                    message.isStreaming = false;
                }
            }
        } else {
            const message = state.chats.find(chat => chat.id === action.payload.id);
            if (message) {
                Object.assign(message, action.payload.message);
                message.isStreaming = false;
            }
        }
    },
    removeChatMessage: (state, action: PayloadAction<number | { id: number; sessionId?: string }>) => {
        const { id: msgId, sessionId } = typeof action.payload === 'number'
            ? { id: action.payload, sessionId: undefined }
            : action.payload;
        const targetId = sessionId ?? state.activeSessionId;
        if (state.sessions.length > 0 && targetId) {
            const session = state.sessions.find(s => s.id === targetId);
            if (session) {
                session.messages = session.messages.filter(chat => chat.id !== msgId);
            }
        } else {
            state.chats = state.chats.filter(chat => chat.id !== msgId);
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
            const newId = uuidv4();
            const newSession: ChatSession = {
                id: newId,
                name: "Chat 1",
                messages: [],
                createdAt: new Date(),
                autoScrollEnabled: true,
            };
            state.sessions = [newSession];
            state.activeSessionId = newId;
        }

        // Ensure we have an active session
        if (!state.activeSessionId && state.sessions.length > 0) {
            state.activeSessionId = state.sessions[0].id;
        }
    },
    hydrateChatSessions: (state, action: PayloadAction<{ sessions: ChatSession[]; activeSessionId?: string | null }>) => {
        state.sessions = action.payload.sessions;
        if (action.payload.activeSessionId && state.sessions.some(session => session.id === action.payload.activeSessionId)) {
            state.activeSessionId = action.payload.activeSessionId;
            return;
        }
        state.activeSessionId = state.sessions[0]?.id ?? null;
    },
    addChatSession: (state, action: PayloadAction<{ name?: string; projectId?: string; sourceId?: string; modelType?: string; providerId?: string; model?: string }>) => {
        const newId = uuidv4();
        const newSession: ChatSession = {
            id: newId,
            name: action.payload.name || `Chat ${state.sessions.length + 1}`,
            messages: [],
            createdAt: new Date(),
            updatedAt: new Date(),
            projectId: action.payload.projectId,
            sourceId: action.payload.sourceId,
            status: 'idle',
            lastEventSequence: 0,
            modelType: action.payload.modelType,
            providerId: action.payload.providerId,
            model: action.payload.model,
            autoScrollEnabled: true,
        };
        // Add new session at the top (beginning) of the list
        state.sessions.unshift(newSession);
        state.activeSessionId = newId;
    },
    deleteChatSession: (state, action: PayloadAction<{ sessionId: string }>) => {
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
    updateSessionModel: (state, action: PayloadAction<{ sessionId: string; modelType: string; providerId: string; model: string }>) => {
        const session = state.sessions.find(s => s.id === action.payload.sessionId);
        if (session) {
            session.modelType = action.payload.modelType;
            session.providerId = action.payload.providerId;
            session.model = action.payload.model;
        }
    },
    updateSessionStatus: (state, action: PayloadAction<{ sessionId: string; status: string; activeRunId?: string }>) => {
        const session = state.sessions.find(s => s.id === action.payload.sessionId);
        if (session) {
            session.status = action.payload.status;
            if (action.payload.activeRunId !== undefined) {
                session.activeRunId = action.payload.activeRunId;
            }
        }
    },
    updateSessionEventSequence: (state, action: PayloadAction<{ sessionId: string; sequence: number }>) => {
        const session = state.sessions.find(s => s.id === action.payload.sessionId);
        if (session) {
            session.lastEventSequence = Math.max(session.lastEventSequence ?? 0, action.payload.sequence);
        }
    },
    updateSessionAutoScroll: (state, action: PayloadAction<{ sessionId: string; autoScrollEnabled: boolean }>) => {
        const session = state.sessions.find(s => s.id === action.payload.sessionId);
        if (session) {
            session.autoScrollEnabled = action.payload.autoScrollEnabled;
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
