import { createContext, use, useState, useEffect, useCallback, useRef } from "react";
import { useConnectionStore } from "@/stores/useConnectionStore";
import { useI18n } from "@/i18n/useI18n";
import type { TreeNodeData, NodeType } from "./types";
import { connectionToNode } from "./types";
import { getSidebarRevealAncestors } from "../sidebar-selection";

const STORAGE_KEY = "sidebar_expanded_items";

interface SidebarTreeContextValue {
  expandedItems: Set<string>;
  treeData: Record<string, TreeNodeData[]>;
  isLoading: Record<string, boolean>;
  toggleItem: (node: TreeNodeData) => Promise<void>;
  fetchNodeChildren: (node: TreeNodeData) => Promise<TreeNodeData[]>;
  refreshNode: (node: TreeNodeData) => Promise<void>;
  collapseNode: (nodeId: string) => void;
  revealNode: (node: TreeNodeData) => Promise<void>;
}

const SidebarTreeContext = createContext<SidebarTreeContextValue | null>(null);

/** Consume the sidebar tree context. Must be called within a SidebarTreeProvider. */
export function useSidebarTree(): SidebarTreeContextValue {
  const ctx = use(SidebarTreeContext);
  if (!ctx) throw new Error("useSidebarTree must be used within SidebarTreeProvider");
  return ctx;
}

/** Provides tree state (expanded items, tree data, loading) and tree operations. */
export function SidebarTreeProvider({ children }: { children: React.ReactNode }) {
  const { connections, fetchDatabases, fetchSchemas, fetchTables, fetchSystemSchemas } =
    useConnectionStore();
  const { t } = useI18n();

  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());
  const [treeData, setTreeData] = useState<Record<string, TreeNodeData[]>>({});
  const [isLoading, setIsLoading] = useState<Record<string, boolean>>({});
  const [isRestoring, setIsRestoring] = useState(true);
  const hasRestored = useRef(false);
  const treeDataRef = useRef(treeData);

  useEffect(() => {
    treeDataRef.current = treeData;
  }, [treeData]);

  // Persist expanded items to localStorage
  useEffect(() => {
    if (!isRestoring) {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify(Array.from(expandedItems))
      );
    }
  }, [expandedItems, isRestoring]);

  /** Build children array for a given node by fetching from the store */
  const buildChildren = useCallback(
    async (node: TreeNodeData): Promise<TreeNodeData[]> => {
      // Read directly from store to avoid stale closure values
      const { systemSchemas, showSystemObjectsFor } = useConnectionStore.getState();

      if (node.type === "connection") {
        const conn = connections.find((c) => c.id === node.connectionId);
        let dbs = await fetchDatabases(node.connectionId);
        if (showSystemObjectsFor.has(node.id) && systemSchemas.length > 0 && conn?.type !== 'POSTGRES') {
          // Merge system databases that the backend may not return due to
          // permission restrictions (e.g. MongoDB admin/local/config)
          const dbSet = new Set(dbs);
          for (const sys of systemSchemas) {
            if (!dbSet.has(sys)) dbs.push(sys);
          }
        } else if (!showSystemObjectsFor.has(node.id) && systemSchemas.length > 0) {
          dbs = dbs.filter(db => !systemSchemas.includes(db));
        }
        return dbs.map((db) => ({
          id: `${node.id}-${db}`,
          name: db,
          type: "database" as const,
          parentId: node.id,
          connectionId: node.connectionId,
          metadata: { database: db },
        }));
      }

      if (node.type === "database") {
        const conn = connections.find((c) => c.id === node.connectionId);

        if (conn?.type === "POSTGRES") {
          let schemas = await fetchSchemas(node.connectionId, node.name);
          if (!showSystemObjectsFor.has(node.id) && systemSchemas.length > 0) {
            schemas = schemas.filter(s => !systemSchemas.includes(s));
          }
          return schemas.map((schema) => ({
            id: `${node.id}-${schema}`,
            name: schema,
            type: "schema" as const,
            parentId: node.id,
            connectionId: node.connectionId,
            metadata: { database: node.name, schema },
          }));
        }

        if (conn?.type === "REDIS") {
          return [
            {
              id: `${node.id}-keys`,
              name: t("sidebar.redis.keysFolder"),
              type: "redis_keys_folder" as const,
              parentId: node.id,
              connectionId: node.connectionId,
              metadata: { database: node.name },
            },
          ];
        }

        // MySQL / MongoDB / ClickHouse — fetch tables directly
        const tables = await fetchTables(node.connectionId, node.name);
        return tables.map((t) => ({
          id: `${node.id}-${t.name}`,
          name: t.name,
          type: (conn?.type === "MONGODB"
            ? "collection"
            : t.type.toLowerCase().includes("view") ? "view" : "table") as NodeType,
          parentId: node.id,
          connectionId: node.connectionId,
          metadata: { database: node.name, table: t.name },
        }));
      }

      if (node.type === "redis_keys_folder") {
        const keys = await fetchTables(node.connectionId, node.metadata.database!);
        return keys.map((k) => ({
          id: `${node.id}-${k.name}`,
          name: k.name,
          type: "redis_key" as const,
          parentId: node.id,
          connectionId: node.connectionId,
          metadata: { database: node.metadata.database, redisKeyType: k.type },
        }));
      }

      if (node.type === "schema") {
        return [
          {
            id: `${node.id}-tables`,
            name: t("sidebar.tree.tables"),
            type: "table_folder" as const,
            parentId: node.id,
            connectionId: node.connectionId,
            metadata: { database: node.metadata.database, schema: node.name },
          },
          {
            id: `${node.id}-views`,
            name: t("sidebar.tree.views"),
            type: "view_folder" as const,
            parentId: node.id,
            connectionId: node.connectionId,
            metadata: { database: node.metadata.database, schema: node.name },
          },
        ];
      }

      if (node.type === "table_folder" || node.type === "view_folder") {
        const isViewFolder = node.type === "view_folder";
        const tables = await fetchTables(
          node.connectionId,
          node.metadata.database!,
          node.metadata.schema!
        );
        return tables
          .filter((t) =>
            isViewFolder
              ? t.type.toLowerCase().includes("view")
              : !t.type.toLowerCase().includes("view")
          )
          .map((t) => ({
            id: `${node.id}-${t.name}`,
            name: t.name,
            type: (isViewFolder ? "view" : "table") as NodeType,
            parentId: node.id,
            connectionId: node.connectionId,
            metadata: {
              database: node.metadata.database,
              schema: node.metadata.schema,
              table: t.name,
            },
          }));
      }

      return [];
    },
    [connections, fetchDatabases, fetchSchemas, fetchTables, t]
  );

  /** Fetch and store children for a node */
  const fetchNodeChildren = useCallback(
    async (node: TreeNodeData) => {
      setIsLoading((prev) => ({ ...prev, [node.id]: true }));
      try {
        const children = await buildChildren(node);
        setTreeData((prev) => ({ ...prev, [node.id]: children }));
        return children;
      } catch (error) {
        console.error("Failed to fetch children:", error);
        throw error;
      } finally {
        setIsLoading((prev) => ({ ...prev, [node.id]: false }));
      }
    },
    [buildChildren]
  );

  /** Toggle expand/collapse for a node */
  const toggleItem = useCallback(
    async (node: TreeNodeData) => {
      const newExpanded = new Set(expandedItems);

      if (newExpanded.has(node.id)) {
        newExpanded.delete(node.id);
      } else {
        newExpanded.add(node.id);
        // Always re-fetch databases when expanding (original behavior)
        const shouldFetch = !treeData[node.id] || node.type === "database" || node.type === "redis_keys_folder";
        if (shouldFetch) {
          await fetchNodeChildren(node);
        }
      }

      setExpandedItems(newExpanded);
    },
    [expandedItems, treeData, fetchNodeChildren]
  );

  /** Re-fetch children for a node (used after mutations) */
  const refreshNode = useCallback(
    async (node: TreeNodeData) => {
      setTreeData((prev) => {
        const next = { ...prev };
        // Cascade: also clear direct children's data (e.g., folder contents when refreshing schema)
        const children = prev[node.id];
        if (children) {
          for (const child of children) {
            delete next[child.id];
          }
        }
        delete next[node.id];
        return next;
      });
      if (expandedItems.has(node.id)) {
        const children = await fetchNodeChildren(node);
        // Re-fetch expanded children whose data was cleared by the cascade
        for (const child of children) {
          if (expandedItems.has(child.id)) {
            await fetchNodeChildren(child);
          }
        }
      }
    },
    [expandedItems, fetchNodeChildren]
  );

  /** Collapse a node without fetching */
  const collapseNode = useCallback((nodeId: string) => {
    setExpandedItems((prev) => {
      const next = new Set(prev);
      next.delete(nodeId);
      return next;
    });
  }, []);

  /** Expand the collapsed ancestors needed to render a focused node. */
  const revealNode = useCallback(
    async (node: TreeNodeData) => {
      const ancestors = getSidebarRevealAncestors(node, connections, {
        redisKeysFolder: t("sidebar.redis.keysFolder"),
        tablesFolder: t("sidebar.tree.tables"),
        viewsFolder: t("sidebar.tree.views"),
      });
      const fetchedTreeData: Record<string, TreeNodeData[]> = {};
      const knownTreeData = { ...treeDataRef.current };

      for (const ancestor of ancestors) {
        const shouldFetch =
          !knownTreeData[ancestor.id] ||
          ancestor.type === "database" ||
          ancestor.type === "redis_keys_folder";
        if (!shouldFetch) continue;

        setIsLoading((prev) => ({ ...prev, [ancestor.id]: true }));
        try {
          const children = await buildChildren(ancestor);
          fetchedTreeData[ancestor.id] = children;
          knownTreeData[ancestor.id] = children;
        } catch (error) {
          console.error("Failed to reveal node:", node.id, error);
          throw error;
        } finally {
          setIsLoading((prev) => ({ ...prev, [ancestor.id]: false }));
        }
      }

      setTreeData((prev) => ({ ...prev, ...fetchedTreeData }));
      setExpandedItems((prev) => {
        const next = new Set(prev);
        for (const ancestor of ancestors) {
          next.add(ancestor.id);
        }
        return next;
      });
    },
    [buildChildren, connections, t],
  );

  // Restore expanded state from localStorage on mount (runs once)
  // Waits for systemSchemas to load first so filtering is applied correctly
  useEffect(() => {
    if (hasRestored.current || connections.length === 0) return;

    const restoreState = async () => {
      // Ensure system schemas are loaded before restoring tree
      await fetchSystemSchemas();

      const stored = localStorage.getItem(STORAGE_KEY);
      if (!stored) {
        setIsRestoring(false);
        return;
      }

      try {
        const expandedIds = new Set<string>(JSON.parse(stored));
        setExpandedItems(expandedIds);

        const fetchRecursively = async (nodes: TreeNodeData[]) => {
          for (const node of nodes) {
            if (!expandedIds.has(node.id)) continue;
            setIsLoading((prev) => ({ ...prev, [node.id]: true }));
            try {
              const children = await buildChildren(node);
              setTreeData((prev) => ({ ...prev, [node.id]: children }));
              if (children.length > 0) {
                await fetchRecursively(children);
              }
            } catch (error) {
              console.error("Failed to restore node:", node.id, error);
            } finally {
              setIsLoading((prev) => ({ ...prev, [node.id]: false }));
            }
          }
        };

        const connectionNodes = connections.map(connectionToNode);
        await fetchRecursively(connectionNodes);
      } catch (e) {
        console.error("Failed to restore expanded items", e);
      }

      setIsRestoring(false);
    };

    hasRestored.current = true;
    restoreState();
  }, [connections.length, buildChildren, fetchSystemSchemas]);

  // Refresh all expanded connection nodes when sidebarRefreshKey changes (e.g. after DDL in editor)
  const sidebarRefreshKey = useConnectionStore((s) => s.sidebarRefreshKey);
  const prevRefreshKey = useRef(sidebarRefreshKey);
  useEffect(() => {
    if (sidebarRefreshKey === prevRefreshKey.current) return;
    prevRefreshKey.current = sidebarRefreshKey;
    const connectionNodes = connections.map(connectionToNode);
    for (const node of connectionNodes) {
      if (expandedItems.has(node.id)) {
        refreshNode(node);
      }
    }
  }, [sidebarRefreshKey, connections, expandedItems, refreshNode]);

  const value: SidebarTreeContextValue = {
    expandedItems,
    treeData,
    isLoading,
    toggleItem,
    fetchNodeChildren,
    refreshNode,
    collapseNode,
    revealNode,
  };

  return (
    <SidebarTreeContext value={value}>
      {children}
    </SidebarTreeContext>
  );
}
