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

import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { v4 } from 'uuid';
import { WhereCondition } from '@graphql';

export type ScratchpadPage = {
  id: string;
  name: string;
  cellIds: string[];
  conditions?: WhereCondition;
}

export type ScratchpadCell = {
  id: string;
  code: string;
  mode: string;
  history: Array<{
    id: string;
    item: string;
    status: boolean;
    date: Date;
  }>;
}

export type IScratchpadState = {
  pages: ScratchpadPage[];
  cells: Record<string, ScratchpadCell>;
  activePageId: string | null;
}

const initialState: IScratchpadState = {
  pages: [],
  cells: {},
  activePageId: null,
}

export const scratchpadSlice = createSlice({
  name: 'scratchpad',
  initialState,
  reducers: {
    initializeScratchpad: (state) => {
      if (state.pages.length === 0) {
        const newId = v4();
        const cellId = v4();
        const firstPage: ScratchpadPage = { id: newId, name: "Page 1", cellIds: [cellId] };
        const firstCell: ScratchpadCell = {
          id: cellId,
          code: "",
          mode: "Query",
          history: []
        };
        state.pages = [firstPage];
        state.cells[cellId] = firstCell;
        state.activePageId = newId;
      }
    },
    addPage: (state, action: PayloadAction<{ name?: string; initialQuery?: string }>) => {
      const newId = v4();
      const cellId = v4();
      const newPage: ScratchpadPage = {
        id: newId,
        name: action.payload.name || `Page ${state.pages.length + 1}`,
        cellIds: [cellId]
      };
      const newCell: ScratchpadCell = {
        id: cellId,
        code: action.payload.initialQuery || "",
        mode: "Query",
        history: []
      };
      state.pages.push(newPage);
      state.cells[cellId] = newCell;
      state.activePageId = newId;
    },
    deletePage: (state, action: PayloadAction<{ pageId: string }>) => {
      if (state.pages.length <= 1) return;
      
      const pageIndex = state.pages.findIndex(page => page.id === action.payload.pageId);
      if (pageIndex === -1) return;
      
      // Delete all cells associated with this page
      const page = state.pages[pageIndex];
      page.cellIds.forEach(cellId => {
        delete state.cells[cellId];
      });
      
      // Remove the page
      state.pages.splice(pageIndex, 1);
      
      // If we deleted the active page, switch to the first remaining page
      if (state.activePageId === action.payload.pageId) {
        state.activePageId = state.pages[0]?.id || null;
      }
    },
    setActivePage: (state, action: PayloadAction<{ pageId: string }>) => {
      const page = state.pages.find(p => p.id === action.payload.pageId);
      if (page) {
        state.activePageId = action.payload.pageId;
      }
    },
    updatePageName: (state, action: PayloadAction<{ pageId: string; name: string }>) => {
      const page = state.pages.find(p => p.id === action.payload.pageId);
      if (page) {
        page.name = action.payload.name;
      }
    },
    addCell: (state, action: PayloadAction<{ pageId: string; afterCellId?: string; initialQuery?: string }>) => {
      const newCellId = v4();
      const newCell: ScratchpadCell = {
        id: newCellId,
        code: action.payload.initialQuery || "",
        mode: "Query",
        history: []
      };
      
      state.cells[newCellId] = newCell;
      
      const page = state.pages.find(p => p.id === action.payload.pageId);
      if (page) {
        if (action.payload.afterCellId) {
          const index = page.cellIds.indexOf(action.payload.afterCellId);
          if (index !== -1) {
            page.cellIds.splice(index + 1, 0, newCellId);
          } else {
            page.cellIds.push(newCellId);
          }
        } else {
          page.cellIds.push(newCellId);
        }
      }
    },
    deleteCell: (state, action: PayloadAction<{ pageId: string; cellId: string }>) => {
      const page = state.pages.find(p => p.id === action.payload.pageId);
      if (page && page.cellIds.length > 1) {
        page.cellIds = page.cellIds.filter(id => id !== action.payload.cellId);
        delete state.cells[action.payload.cellId];
      }
    },
    updateCellCode: (state, action: PayloadAction<{ cellId: string; code: string }>) => {
      const cell = state.cells[action.payload.cellId];
      if (cell) {
        cell.code = action.payload.code;
      }
    },
    updateCellMode: (state, action: PayloadAction<{ cellId: string; mode: string }>) => {
      const cell = state.cells[action.payload.cellId];
      if (cell) {
        cell.mode = action.payload.mode;
      }
    },
    addCellHistory: (state, action: PayloadAction<{ cellId: string; item: string; status: boolean }>) => {
      const cell = state.cells[action.payload.cellId];
      if (cell) {
        cell.history.unshift({
          id: v4(),
          item: action.payload.item,
          status: action.payload.status,
          date: new Date()
        });
      }
    },
    clearCellHistory: (state, action: PayloadAction<{ cellId: string }>) => {
      const cell = state.cells[action.payload.cellId];
      if (cell) {
        cell.history = [];
      }
    },
    clearAllScratchpad: (state) => {
      state.pages = [];
      state.cells = {};
      state.activePageId = null;
    },
    ensurePagesHaveCells: (state) => {
      // Migration: Ensure all pages have at least one cell
      state.pages.forEach(page => {
        // Handle case where cellIds might be undefined or null
        if (!page.cellIds || page.cellIds.length === 0) {
          const cellId = v4();
          const newCell: ScratchpadCell = {
            id: cellId,
            code: "",
            mode: "Query",
            history: []
          };
          // Initialize cellIds array if it doesn't exist
          page.cellIds = [cellId];
          state.cells[cellId] = newCell;
        }
      });
      
      // If no pages exist, initialize with a default page
      if (state.pages.length === 0) {
        const newId = v4();
        const cellId = v4();
        const firstPage: ScratchpadPage = { id: newId, name: "Page 1", cellIds: [cellId] };
        const firstCell: ScratchpadCell = {
          id: cellId,
          code: "",
          mode: "Query",
          history: []
        };
        state.pages = [firstPage];
        state.cells[cellId] = firstCell;
        state.activePageId = newId;
      }
      
      // Ensure we have an active page
      if (!state.activePageId && state.pages.length > 0) {
        state.activePageId = state.pages[0].id;
      }
    },
    addConditionToPage: (state, action: PayloadAction<{ pageId: string; condition: WhereCondition }>) => {
      const page = state.pages.find(p => p.id === action.payload.pageId);
      if (page) {
        page.conditions = action.payload.condition;
      }
    },
    removeConditionFromPage: (state, action: PayloadAction<{ pageId: string }>) => {
      const page = state.pages.find(p => p.id === action.payload.pageId);
      if (page) {
        delete page.conditions;
      }
    },
    addCellToPageAndActivate: (state, action: PayloadAction<{ pageId: string; initialQuery?: string }>) => {
      const newCellId = v4();
      const newCell: ScratchpadCell = {
        id: newCellId,
        code: action.payload.initialQuery || "",
        mode: "Query",
        history: []
      };
      
      state.cells[newCellId] = newCell;
      
      const page = state.pages.find(p => p.id === action.payload.pageId);
      if (page) {
        page.cellIds.push(newCellId);
        // Set this page as active
        state.activePageId = action.payload.pageId;
      }
    }
  },
});

export const ScratchpadActions = scratchpadSlice.actions;
export const scratchpadReducers = scratchpadSlice.reducer;
