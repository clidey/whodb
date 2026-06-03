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
  /** Whether this tab has pending SQL row edits or MongoDB document edits. */
  hasUnsavedDatabaseEdits?: boolean;
  /** Number of pending SQL row edits or MongoDB document edits in this tab. */
  unsavedDatabaseEditCount?: number;
}

type DatabaseEditDiscarder = () => void;

interface TabState {
  tabs: Tab[];
  activeTabId: string | null;
  databaseEditDiscarders: Record<string, DatabaseEditDiscarder>;
  openTab: (tab: Omit<Tab, 'id'> & { id?: string }) => string;
  closeTab: (tabId: string) => void;
  setActiveTab: (tabId: string) => void;
  updateTab: (tabId: string, updates: Partial<Tab>) => void;
  setTabUnsavedDatabaseEdits: (tabId: string, count: number) => void;
  registerDatabaseEditDiscarder: (tabId: string, discarder: DatabaseEditDiscarder | null) => void;
  discardUnsavedDatabaseEdits: (tabIds: string[]) => void;
  findExistingTab: (
    type: TabType,
    connectionId: string,
    identifier: string,
    databaseName?: string,
    storageUnitType?: Tab['storageUnitType'],
  ) => Tab | undefined;
  closeOtherTabs: (tabId: string) => void;
  closeAllTabs: () => void;
}

function generateTabId(): string {
  return `tab_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
}

export const useTabStore = create<TabState>((set, get) => ({
  tabs: [],
  activeTabId: null,
  databaseEditDiscarders: {},

  findExistingTab: (type, connectionId, identifier, databaseName, storageUnitType) => {
    return get().tabs.find((tab) => {
      if (tab.type !== type || tab.connectionId !== connectionId) return false;
      if (databaseName && tab.databaseName !== databaseName) return false;
      if (type === 'table') {
        return tab.tableName === identifier && (tab.storageUnitType ?? 'table') === (storageUnitType ?? 'table');
      }
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
            tabData.storageUnitType,
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
      const { [tabId]: _discarder, ...databaseEditDiscarders } = state.databaseEditDiscarders;

      if (state.activeTabId === tabId) {
        if (newTabs.length > 0) {
          const newActiveIndex = Math.min(index, newTabs.length - 1);
          newActiveTabId = newTabs[newActiveIndex].id;
        } else {
          newActiveTabId = null;
        }
      }

      return { tabs: newTabs, activeTabId: newActiveTabId, databaseEditDiscarders };
    });
  },

  setActiveTab: (tabId) => set({ activeTabId: tabId }),

  updateTab: (tabId, updates) => {
    set((state) => ({
      tabs: state.tabs.map((tab) => (tab.id === tabId ? { ...tab, ...updates } : tab)),
    }));
  },

  setTabUnsavedDatabaseEdits: (tabId, count) => {
    set((state) => ({
      tabs: state.tabs.map((tab) => {
        if (tab.id !== tabId) return tab;
        return {
          ...tab,
          hasUnsavedDatabaseEdits: count > 0,
          unsavedDatabaseEditCount: count > 0 ? count : undefined,
        };
      }),
    }));
  },

  registerDatabaseEditDiscarder: (tabId, discarder) => {
    set((state) => {
      if (!discarder) {
        const { [tabId]: _discarder, ...databaseEditDiscarders } = state.databaseEditDiscarders;
        return { databaseEditDiscarders };
      }

      return {
        databaseEditDiscarders: {
          ...state.databaseEditDiscarders,
          [tabId]: discarder,
        },
      };
    });
  },

  discardUnsavedDatabaseEdits: (tabIds) => {
    const { databaseEditDiscarders } = get();
    tabIds.forEach((tabId) => databaseEditDiscarders[tabId]?.());

    set((state) => ({
      tabs: state.tabs.map((tab) => (
        tabIds.includes(tab.id)
          ? { ...tab, hasUnsavedDatabaseEdits: false, unsavedDatabaseEditCount: undefined }
          : tab
      )),
    }));
  },

  closeOtherTabs: (tabId) => {
    set((state) => {
      const tabToKeep = state.tabs.find((t) => t.id === tabId);
      if (!tabToKeep) return state;
      const discarder = state.databaseEditDiscarders[tabId];
      return {
        tabs: [tabToKeep],
        activeTabId: tabId,
        databaseEditDiscarders: discarder ? { [tabId]: discarder } : {},
      };
    });
  },

  closeAllTabs: () => set({ tabs: [], activeTabId: null, databaseEditDiscarders: {} }),
}));
