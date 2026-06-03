import React, { useState, useCallback, useEffect, useMemo, useReducer } from "react";

import { useConnectionStore } from "@/stores/useConnectionStore";
import { useTabStore, type Tab } from "@/stores/useTabStore";
import { ContextMenu } from "../ui/ContextMenu";
import type { Alert } from "@/components/ui/types";

import type { TreeNodeData } from "./SidebarTree/types";
import { connectionToNode, EXPANDABLE_TYPES } from "./SidebarTree/types";
import { SidebarTreeProvider, useSidebarTree, TreeNode, TreeNodeProvider } from "./SidebarTree";
import {
  getConnectionMenuItems,
  getDatabaseMenuItems,
  getSchemaMenuItems,
  getTableFolderMenuItems,
  getViewFolderMenuItems,
  getTableMenuItems,
  getCollectionMenuItems,
  getViewMenuItems,
  getRedisKeysFolderMenuItems,
  getRedisKeyMenuItems,
} from "./contextMenuItems";
import { SidebarModals } from "./SidebarModals";
import { useI18n } from "@/i18n/useI18n";
import { getSidebarSelectionForTab } from "./sidebar-selection";

function getSidebarFocusKey(tab: Tab | null): string {
  if (!tab) return "no-active-tab";

  return JSON.stringify([
    tab.id,
    tab.type,
    tab.connectionId,
    tab.databaseName ?? null,
    tab.schemaName ?? null,
    tab.tableName ?? null,
    tab.storageUnitType ?? null,
    tab.collectionName ?? null,
  ]);
}

// ── Modal reducer (inlined from former useSidebarModals) ────────────

/** All possible modal types and their parameter shapes */
export type ModalState =
  | { type: "create_database"; params: { connectionId: string } }
  | { type: "create_table"; params: { connectionId: string; databaseName: string; schema?: string } }
  | { type: "create_collection"; params: { connectionId: string; databaseName: string } }
  | { type: "edit_database"; params: { connectionId: string; databaseName: string } }
  | { type: "delete_database"; params: { connectionId: string; databaseName: string } }
  | { type: "edit_table"; params: { connectionId: string; databaseName: string; schema?: string; tableName: string } }
  | { type: "delete_table"; params: { connectionId: string; databaseName: string; schema?: string; tableName: string } }
  | { type: "export_data"; params: { connectionId: string; databaseName: string; schema: string | null; tableName: string } }
  | { type: "export_database"; params: { connectionId: string; databaseName: string; schema: string } }
  | { type: "clear_table_data"; params: { connectionId: string; databaseName: string; schema?: string; tableName: string } }
  | { type: "copy_table"; params: { connectionId: string; databaseName: string; schema?: string; tableName: string } }
  | { type: "rename_table"; params: { connectionId: string; databaseName: string; schema?: string; tableName: string } }
  | { type: "export_collection"; params: { connectionId: string; databaseName: string; collectionName: string } }
  | { type: "drop_collection"; params: { connectionId: string; databaseName: string; collectionName: string } }
  | { type: "create_redis_key"; params: { connectionId: string; databaseName: string } }
  | { type: "delete_redis_key"; params: { connectionId: string; databaseName: string; keyName: string } }
  | { type: "export_redis_key"; params: { connectionId: string; databaseName: string; keyName: string } };

type Action =
  | { action: "open"; modal: ModalState }
  | { action: "close" };

function modalReducer(_state: ModalState | null, action: Action): ModalState | null {
  if (action.action === "close") return null;
  return action.modal;
}

// ── Sidebar inner (consumes SidebarTreeProvider context) ────────────

function SidebarInner() {
  const { connections, selectedItem, selectItem, systemSchemas, showSystemObjectsFor, toggleSystemObjects, triggerCollectionRefresh } = useConnectionStore();
  const { tabs, activeTabId, openTab } = useTabStore();
  const { t } = useI18n();

  const {
    expandedItems, treeData, isLoading,
    toggleItem, fetchNodeChildren, refreshNode, revealNode,
  } = useSidebarTree();

  // Modal state (inlined from former useSidebarModals)
  const [activeModal, dispatch] = useReducer(modalReducer, null);
  const [alert, setAlert] = useState<Alert | null>(null);

  const openModal = useCallback(
    (modal: ModalState) => dispatch({ action: "open", modal }),
    []
  );
  const closeModal = useCallback(
    () => dispatch({ action: "close" }),
    []
  );

  const showAlert = useCallback(
    (title: string, message: string, type: Alert["type"]) =>
      setAlert({ title, message, type }),
    []
  );
  const closeAlert = useCallback(() => setAlert(null), []);

  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    node: TreeNodeData;
  } | null>(null);

  const activeTab = tabs.find((tab) => tab.id === activeTabId) ?? null;
  const sidebarFocusKey = getSidebarFocusKey(activeTab);
  const sidebarSelection = useMemo(
    () => getSidebarSelectionForTab(activeTab, connections),
    // Ignore title/sqlContent/isDirty changes; they do not affect sidebar focus.
    [connections, sidebarFocusKey],
  );

  useEffect(() => {
    selectItem(sidebarSelection);
    if (sidebarSelection) {
      void revealNode(sidebarSelection).catch((error) => {
        console.error("Failed to reveal sidebar selection:", sidebarSelection.id, error);
      });
    }
  }, [revealNode, selectItem, sidebarSelection]);

  const handleItemClick = useCallback(
    async (node: TreeNodeData) => {
      selectItem(node);

      if (EXPANDABLE_TYPES.has(node.type)) {
        try {
          await toggleItem(node);
        } catch (error: any) {
          if (node.type === "connection") {
            showAlert(
              t("sidebar.alert.connectionFailedTitle"),
              error.message || t("sidebar.alert.connectionFailedMessage"),
              "error",
            );
          }
        }
      }

      if (node.type === "table" || node.type === "view") {
        const tableTitle = node.metadata.database
          ? t("sidebar.tab.tableWithDatabase", { table: node.name, database: node.metadata.database })
          : node.name;
        openTab({
          type: "table",
          title: tableTitle,
          connectionId: node.connectionId,
          databaseName: node.metadata.database,
          schemaName: node.metadata.schema,
          tableName: node.name,
          storageUnitType: node.type,
        });
      } else if (node.type === "collection") {
        const collectionTitle = node.metadata.database
          ? t("sidebar.tab.tableWithDatabase", { table: node.name, database: node.metadata.database })
          : node.name;
        openTab({
          type: "collection",
          title: collectionTitle,
          connectionId: node.connectionId,
          databaseName: node.metadata.database,
          collectionName: node.name,
        });
        triggerCollectionRefresh();
      } else if (node.type === "redis_key") {
        const redisDatabase = node.metadata.database ?? "";
        openTab({
          type: "redis_key_detail",
          title: t("sidebar.tab.redisKeyDetail", { key: node.name, database: redisDatabase }),
          connectionId: node.connectionId,
          databaseName: redisDatabase,
          tableName: node.name,
        });
      }
    },
    [selectItem, toggleItem, showAlert, openTab, triggerCollectionRefresh, t],
  );

  const handleContextMenu = useCallback(
    (e: React.MouseEvent, node: TreeNodeData) => {
      e.preventDefault();
      e.stopPropagation();
      setContextMenu({ x: e.clientX, y: e.clientY, node });
    },
    [],
  );

  const handleContextMenuAction = useCallback(
    (action: string) => {
      if (!contextMenu) return;
      const { node } = contextMenu;

      switch (action) {
        case "new_query": {
          const queryConnectionId = node.connectionId || node.id;
          const queryDatabaseName =
            node.metadata?.database || (node.type === "database" ? node.name : undefined);
          const querySchemaName = node.metadata?.schema;
          const queryTitle = queryDatabaseName
            ? t("sidebar.tab.queryWithDatabase", { database: queryDatabaseName })
            : t("sidebar.tab.queryWithConnection", {
                connection:
                  connections.find((c) => c.id === queryConnectionId)?.name || t("common.untitled"),
              });
          openTab({
            type: "query",
            title: queryTitle,
            connectionId: queryConnectionId,
            databaseName: queryDatabaseName,
            schemaName: querySchemaName,
          });
          break;
        }
        case "new_database":
          openModal({ type: "create_database", params: { connectionId: node.id } });
          break;
        case "new_table":
          openModal({
            type: "create_table",
            params: {
              connectionId: node.connectionId,
              databaseName: node.type === "database" ? node.name : node.metadata.database!,
              schema: node.type === "schema" ? node.name : node.metadata.schema,
            },
          });
          break;
        case "new_collection":
          openModal({
            type: "create_collection",
            params: {
              connectionId: node.connectionId,
              databaseName: node.name,
            },
          });
          break;
        case "edit_database":
          openModal({
            type: "edit_database",
            params: { connectionId: node.connectionId, databaseName: node.name },
          });
          break;
        case "delete_database":
          openModal({
            type: "delete_database",
            params: { connectionId: node.connectionId, databaseName: node.name },
          });
          break;
        case "edit_table":
          openModal({
            type: "edit_table",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              schema: node.metadata?.schema,
              tableName: node.name,
            },
          });
          break;
        case "delete_table":
          openModal({
            type: "delete_table",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              schema: node.metadata?.schema,
              tableName: node.name,
            },
          });
          break;
        case "export_data":
          openModal({
            type: "export_data",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              schema: node.metadata.schema || null,
              tableName: node.name,
            },
          });
          break;
        case "export_database": {
          const conn = connections.find(c => c.id === node.connectionId);
          const dbType = conn?.type ?? '';
          const schema = dbType === 'POSTGRES' ? 'public' : dbType === 'MYSQL' || dbType === 'CLICKHOUSE' ? node.name : '';
          openModal({
            type: "export_database",
            params: { connectionId: node.connectionId, databaseName: node.name, schema },
          });
          break;
        }
        case "clear_table_data":
          openModal({
            type: "clear_table_data",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              schema: node.metadata?.schema,
              tableName: node.name,
            },
          });
          break;
        case "copy_table":
          openModal({
            type: "copy_table",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              schema: node.metadata?.schema,
              tableName: node.name,
            },
          });
          break;
        case "rename_table":
          openModal({
            type: "rename_table",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              schema: node.metadata?.schema,
              tableName: node.name,
            },
          });
          break;
        case "export_collection":
          openModal({
            type: "export_collection",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              collectionName: node.name,
            },
          });
          break;
        case "drop_collection":
          openModal({
            type: "drop_collection",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              collectionName: node.name,
            },
          });
          break;
        case "new_redis_key":
          openModal({
            type: "create_redis_key",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
            },
          });
          break;
        case "export_redis_key":
          openModal({
            type: "export_redis_key",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              keyName: node.name,
            },
          });
          break;
        case "delete_redis_key":
          openModal({
            type: "delete_redis_key",
            params: {
              connectionId: node.connectionId,
              databaseName: node.metadata.database!,
              keyName: node.name,
            },
          });
          break;
        case "refresh":
          if (expandedItems.has(node.id)) {
            fetchNodeChildren(node);
          } else {
            toggleItem(node);
          }
          break;
        case "toggle_system_objects":
          toggleSystemObjects(node.id);
          // Re-fetch this node's children with the new filter
          if (expandedItems.has(node.id)) {
            fetchNodeChildren(node);
          }
          break;
      }

      setContextMenu(null);
    },
    [
      contextMenu, connections, openTab, openModal,
      expandedItems, fetchNodeChildren, toggleItem,
      toggleSystemObjects, t,
    ],
  );

  // Determine context menu items based on the right-clicked node type
  const contextMenuItems = (() => {
    if (!contextMenu) return [];
    const { node } = contextMenu;
    const callbacks = {
      onAction: handleContextMenuAction,
      t,
    };

    const nodeId = node.type === "connection" ? node.id : node.id;
    const sysState = { systemSchemas, showSystemObjects: showSystemObjectsFor.has(nodeId) };

    switch (node.type) {
      case "connection":
        return getConnectionMenuItems(
          connections.find((c) => c.id === node.id)!.type,
          callbacks,
          sysState,
        );
      case "database":
        return getDatabaseMenuItems(
          connections.find((c) => c.id === node.connectionId)!.type,
          callbacks,
          sysState,
        );
      case "schema":
        return getSchemaMenuItems(callbacks);
      case "table_folder":
        return getTableFolderMenuItems(callbacks);
      case "view_folder":
        return getViewFolderMenuItems(callbacks);
      case "table":
        return getTableMenuItems(
          connections.find((c) => c.id === node.connectionId)!.type,
          callbacks,
        );
      case "view":
        return getViewMenuItems(callbacks);
      case "collection":
        return getCollectionMenuItems(callbacks);
      case "redis_keys_folder":
        return getRedisKeysFolderMenuItems(callbacks);
      case "redis_key":
        return getRedisKeyMenuItems(callbacks);
      default:
        return [];
    }
  })();

  return (
    <div
      className="flex h-full w-full flex-col border-r border-sidebar-border bg-sidebar"
      data-testid="database.sidebar"
      data-qa-module="database"
      data-qa-object="sidebar"
      data-qa-state={isLoading && Object.keys(isLoading).length > 0 ? "loading" : "ready"}
    >
      {/* Header */}
      <div
        className="flex items-center px-4 pt-5 pb-2 shrink-0"
        data-testid="database.sidebar.header"
        data-qa-module="database"
        data-qa-object="sidebar-header"
      >
        <span className="text-xl font-medium text-sidebar-foreground">{t("sidebar.title")}</span>
      </div>

      {/* Tree */}
      <div
        className="flex-1 overflow-y-auto p-2"
        data-testid="database.sidebar.tree"
        data-qa-module="database"
        data-qa-object="connection-tree"
        data-qa-state={connections.length > 0 ? "ready" : "empty"}
      >
        {connections.map((conn) => (
          <TreeNodeProvider
            key={conn.id}
            value={{
              selectedItemId: selectedItem?.id ?? null,
              connectionDbType: conn.type,
              onItemClick: handleItemClick,
              onToggle: toggleItem,
              onContextMenu: handleContextMenu,
            }}
          >
            <TreeNode node={connectionToNode(conn)} depth={0} />
          </TreeNodeProvider>
        ))}
      </div>

      {/* Context Menu */}
      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          onClose={() => setContextMenu(null)}
          items={contextMenuItems}
        />
      )}

      <SidebarModals
        activeModal={activeModal}
        closeModal={closeModal}
        alert={alert}
        closeAlert={closeAlert}
        refreshNode={refreshNode}
      />
    </div>
  );
}

// ── Public Sidebar (wraps with SidebarTreeProvider) ─────────────────

export function Sidebar() {
  return (
    <SidebarTreeProvider>
      <SidebarInner />
    </SidebarTreeProvider>
  );
}
