import type { Connection } from "@/stores/useConnectionStore";

export type NodeType =
  | "connection"
  | "database"
  | "schema"
  | "table_folder"
  | "view_folder"
  | "table"
  | "view"
  | "collection"
  | "redis_keys_folder"
  | "redis_key";

export interface TreeNodeData {
  id: string;
  name: string;
  type: NodeType;
  parentId?: string;
  connectionId: string;
  /** Storage unit provisioned by the engine, an extension, or platform tooling — rendered muted. */
  system?: boolean;
  metadata: {
    database?: string;
    schema?: string;
    table?: string;
    redisKeyType?: string;
  };
}

/** Types that can be expanded to show children */
export const EXPANDABLE_TYPES: ReadonlySet<NodeType> = new Set([
  "connection",
  "database",
  "schema",
  "table_folder",
  "view_folder",
  "redis_keys_folder",
]);

/** Icon color class per node type */
export const NODE_ICON_COLORS: Record<NodeType, string> = {
  connection: "text-primary",
  database: "text-chart-3",
  schema: "text-chart-4",
  table_folder: "text-chart-2",
  view_folder: "text-muted-foreground",
  table: "text-chart-2",
  view: "text-muted-foreground",
  collection: "text-chart-5",
  redis_keys_folder: "text-chart-2",
  redis_key: "text-muted-foreground",
};

/** Database brand icons (connection-level, keyed by Connection.type) */
export const DB_ICONS: Record<string, string> = {
  MYSQL: "/images/mysql.svg",
  POSTGRES: "/images/postgresql.svg",
  MONGODB: "/images/mongodb.svg",
  REDIS: "/images/redis.svg",
  // ClickHouse has no brand icon — falls through to default Database icon
};

/** Convert a Connection to a root-level TreeNodeData */
export function connectionToNode(conn: Connection): TreeNodeData {
  return {
    id: conn.id,
    name: conn.name,
    type: "connection",
    connectionId: conn.id,
    metadata: {},
  };
}
