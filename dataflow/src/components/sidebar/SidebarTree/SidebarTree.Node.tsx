import React, { createContext, use, useEffect, useRef } from "react";
import {
  ChevronRight, ChevronDown, Loader2,
  Database, ListTree, Table, Files, Eye, Folder,
  Type, Hash, ListOrdered, CircleDot, ArrowUpDown,
} from "lucide-react";
import { cn } from "@/lib/utils";
import type { TreeNodeData, NodeType } from "./types";
import { EXPANDABLE_TYPES, DB_ICONS, NODE_ICON_COLORS } from "./types";
import { useSidebarTree } from "./SidebarTreeProvider";

/** Per-connection context passed by Sidebar to each connection's tree. */
export interface TreeNodeContextValue {
  selectedItemId: string | null;
  connectionDbType: string;
  onItemClick: (node: TreeNodeData) => void;
  onToggle: (node: TreeNodeData) => void;
  onContextMenu: (e: React.MouseEvent, node: TreeNodeData) => void;
}

const TreeNodeCtx = createContext<TreeNodeContextValue | null>(null);

export const TreeNodeProvider = TreeNodeCtx.Provider;

function useTreeNodeContext(): TreeNodeContextValue {
  const ctx = use(TreeNodeCtx);
  if (!ctx) throw new Error("TreeNode must be used within TreeNodeProvider");
  return ctx;
}

const NODE_ICONS: Record<NodeType, React.ComponentType<{ className?: string }>> = {
  connection: Database,
  database: Database,
  schema: ListTree,
  table_folder: Folder,
  view_folder: Folder,
  table: Table,
  view: Eye,
  collection: Files,
  redis_keys_folder: Folder,
  redis_key: Type,
};

const REDIS_TYPE_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  string: Type,
  hash: Hash,
  list: ListOrdered,
  set: CircleDot,
  zset: ArrowUpDown,
};

interface TreeNodeProps {
  node: TreeNodeData;
  depth: number;
}

export function TreeNode({ node, depth }: TreeNodeProps) {
  const nodeRef = useRef<HTMLDivElement>(null);
  const { expandedItems, isLoading: loadingItems, treeData } = useSidebarTree();
  const {
    selectedItemId,
    connectionDbType,
    onItemClick, onToggle, onContextMenu,
  } = useTreeNodeContext();

  const isExpandable = EXPANDABLE_TYPES.has(node.type);
  const isRoot = depth === 0;
  const isExpanded = expandedItems.has(node.id);
  const isSelected = selectedItemId === node.id;
  const nodeIsLoading = !!loadingItems[node.id];
  const children = treeData[node.id];

  const iconColor = NODE_ICON_COLORS[node.type];

  const Icon = node.type === "redis_key" && node.metadata.redisKeyType
    ? (REDIS_TYPE_ICONS[node.metadata.redisKeyType] ?? NODE_ICONS[node.type])
    : NODE_ICONS[node.type];
  const brandIcon = isRoot ? DB_ICONS[connectionDbType] : null;

  useEffect(() => {
    if (!isSelected) return;
    nodeRef.current?.scrollIntoView({ block: "nearest", inline: "nearest" });
  }, [isSelected]);

  return (
    <div>
      <div
        ref={nodeRef}
        data-testid="database.sidebar.tree-node"
        data-qa-module="database"
        data-qa-object="sidebar-node"
        data-qa-state={[
          isSelected ? "selected" : "idle",
          isExpandable ? (isExpanded ? "expanded" : "collapsed") : "leaf",
          nodeIsLoading ? "loading" : null,
        ].filter(Boolean).join(" ")}
        data-qa-resource-type={node.type}
        data-qa-resource-id={node.id}
        data-qa-connection-id={node.connectionId || node.id}
        data-qa-database={node.metadata.database}
        data-qa-schema={node.metadata.schema}
        className={cn(
          "group flex items-center gap-2 rounded-md text-sm transition-colors cursor-pointer select-none px-2 py-2",
          isSelected
            ? "bg-input text-accent-foreground font-medium"
            : "text-muted-foreground hover:bg-input hover:text-foreground"
        )}
        onClick={() => onItemClick(node)}
        onContextMenu={(e) => onContextMenu(e, node)}
      >
        {isExpandable ? (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggle(node);
            }}
            data-testid="database.sidebar.tree-node-toggle"
            data-qa-module="database"
            data-qa-object="sidebar-node"
            data-qa-action={isExpanded ? "collapse" : "expand"}
            data-qa-state={nodeIsLoading ? "loading" : isExpanded ? "expanded" : "collapsed"}
            data-qa-resource-type={node.type}
            data-qa-resource-id={node.id}
            className={cn(
              "rounded p-0.5 transition-colors",
              isSelected ? "hover:bg-primary/20" : "hover:bg-muted"
            )}
          >
            {isExpanded ? (
              <ChevronDown className="h-4 w-4 opacity-70" />
            ) : (
              <ChevronRight className="h-4 w-4 opacity-70" />
            )}
          </button>
        ) : (
          <span className="p-0.5"><ChevronRight className="h-4 w-4 opacity-0" /></span>
        )}

        {brandIcon ? (
          <img src={brandIcon} alt={connectionDbType} className="h-4 w-4 shrink-0" />
        ) : (
          <Icon className={cn("h-4 w-4", iconColor)} />
        )}

        <span className="truncate flex-1">{node.name}</span>

        {nodeIsLoading && (
          <Loader2
            className="h-3 w-3 animate-spin text-muted-foreground"
            data-testid="database.sidebar.tree-node-loading"
            data-qa-module="database"
            data-qa-object="sidebar-node"
            data-qa-state="loading"
            data-qa-resource-type={node.type}
            data-qa-resource-id={node.id}
          />
        )}
      </div>

      {isExpanded && children && children.length > 0 && (
        <div
          className="ml-3 pl-3 border-l border-border/50 mt-1 space-y-0.5"
          data-testid="database.sidebar.tree-node-children"
          data-qa-module="database"
          data-qa-object="sidebar-node-children"
          data-qa-state="expanded"
          data-qa-resource-type={node.type}
          data-qa-resource-id={node.id}
        >
          {children.map((child) => (
            <TreeNode key={child.id} node={child} depth={depth + 1} />
          ))}
        </div>
      )}
    </div>
  );
}
