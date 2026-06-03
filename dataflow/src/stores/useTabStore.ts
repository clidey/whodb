import { create } from 'zustand';

export type TabType = 'query' | 'table' | 'collection' | 'redis_key_detail';

export interface Tab {
  id: string;
  type: TabType;
  title: string;
  connectionId: string;
  databaseName?: string;
  schemaName?: string;
  sqlContent?: string;
  tableName?: string;
  storageUnitType?: 'table' | 'view';
  collectionName?: string;
  isDirty?: boolean;
}

interface TabState {
  tabs: Tab[];
  activeTabId: string | null;
  openTab: (tab: Omit<Tab, 'id'> & { id?: string }) => string;
  closeTab: (tabId: string) => void;
  setActiveTab: (tabId: string) => void;
  updateTab: (tabId: string, updates: Partial<Tab>) => void;
  findExistingTab: (type: TabType, connectionId: string, identifier: string, databaseName?: string) => Tab | undefined;
  closeOtherTabs: (tabId: string) => void;
  closeAllTabs: () => void;
}

function generateTabId(): string {
  return `tab_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
}

export const useTabStore = create<TabState>((set, get) => ({
  tabs: [],
  activeTabId: null,

  findExistingTab: (type, connectionId, identifier, databaseName) => {
    return get().tabs.find((tab) => {
      if (tab.type !== type || tab.connectionId !== connectionId) return false;
      if (databaseName && tab.databaseName !== databaseName) return false;
      if (type === 'table') return tab.tableName === identifier;
      if (type === 'collection') return tab.collectionName === identifier;
      if (type === 'redis_key_detail') return tab.tableName === identifier;
      return false;
    });
  },

  openTab: (tabData) => {
    const { findExistingTab } = get();
    const existingTab =
      tabData.type !== 'query'
        ? findExistingTab(
            tabData.type,
            tabData.connectionId,
            tabData.tableName || tabData.collectionName || '',
            tabData.databaseName,
          )
        : undefined;

    if (existingTab) {
      set({ activeTabId: existingTab.id });
      return existingTab.id;
    }

    const newTab: Tab = { ...tabData, id: tabData.id || generateTabId() };
    set((state) => ({
      tabs: [...state.tabs, newTab],
      activeTabId: newTab.id,
    }));
    return newTab.id;
  },

  closeTab: (tabId) => {
    set((state) => {
      const index = state.tabs.findIndex((t) => t.id === tabId);
      const newTabs = state.tabs.filter((t) => t.id !== tabId);
      let newActiveTabId = state.activeTabId;

      if (state.activeTabId === tabId) {
        if (newTabs.length > 0) {
          const newActiveIndex = Math.min(index, newTabs.length - 1);
          newActiveTabId = newTabs[newActiveIndex].id;
        } else {
          newActiveTabId = null;
        }
      }

      return { tabs: newTabs, activeTabId: newActiveTabId };
    });
  },

  setActiveTab: (tabId) => set({ activeTabId: tabId }),

  updateTab: (tabId, updates) => {
    set((state) => ({
      tabs: state.tabs.map((tab) => (tab.id === tabId ? { ...tab, ...updates } : tab)),
    }));
  },

  closeOtherTabs: (tabId) => {
    set((state) => {
      const tabToKeep = state.tabs.find((t) => t.id === tabId);
      if (!tabToKeep) return state;
      return { tabs: [tabToKeep], activeTabId: tabId };
    });
  },

  closeAllTabs: () => set({ tabs: [], activeTabId: null }),
}));
