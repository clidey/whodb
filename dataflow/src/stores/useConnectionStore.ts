import { create } from 'zustand';
import { graphqlClient } from '@/config/graphql-client';
import { useAuthStore } from '@/stores/useAuthStore';
import type { AuthSessionSummary } from '@/config/auth-store';
import { getAuthSession } from '@/config/auth-store';
import {
  GetDatabaseDocument,
  type GetDatabaseQuery,
  type GetDatabaseQueryVariables,
  GetDatabaseMetadataDocument,
  type GetDatabaseMetadataQuery,
  GetSchemaDocument,
  type GetSchemaQuery,
  GetStorageUnitsDocument,
  type GetStorageUnitsQuery,
  type GetStorageUnitsQueryVariables,
  ExecuteConfirmedSqlDocument,
  type ExecuteConfirmedSqlMutation,
  type ExecuteConfirmedSqlMutationVariables,
  AddStorageUnitDocument,
  type AddStorageUnitMutation,
  type AddStorageUnitMutationVariables,
  type RecordInput,
  RawExecuteDocument,
  type RawExecuteQuery,
  type RawExecuteQueryVariables,
} from '@graphql';
import type { SqlDialect } from '@/utils/ddl-sql';
import {
  createDatabaseSQL, dropDatabaseSQL, renameDatabaseSQL,
  dropTableSQL, truncateTableSQL, deleteAllRowsSQL,
  renameTableSQL, copyTableStructureSQL, copyTableWithDataSQL,
} from '@/utils/ddl-sql';
import {
  buildMongoCollectionCommand,
  buildMongoDropDatabaseCommand,
} from '@/utils/mongodb-shell';

export interface Connection {
  id: string;
  name: string;
  type: 'MYSQL' | 'POSTGRES' | 'MONGODB' | 'REDIS' | 'CLICKHOUSE';
  host: string;
  port: string;
  user: string;
  password: string;
  database: string;
  createdAt: string;
}

export type SelectedItemType = 'connection' | 'database' | 'schema' | 'table_folder' | 'view_folder' | 'table' | 'view' | 'collection' | 'key' | 'redis_keys_folder' | 'redis_key' | null;

export interface SelectedItem {
  type: SelectedItemType;
  id: string;
  name: string;
  parentId?: string;
  connectionId?: string;
  metadata?: any;
}

interface ConnectionState {
  connections: Connection[];
  selectedItem: SelectedItem | null;
  tableRefreshKey: number;
  triggerTableRefresh: () => void;
  collectionRefreshKey: number;
  triggerCollectionRefresh: () => void;
  sidebarRefreshKey: number;
  triggerSidebarRefresh: () => void;
  createDatabase: (databaseName: string) => Promise<DDLResult>;
  renameDatabase: (oldName: string, newName: string) => Promise<DDLResult>;
  deleteDatabase: (databaseName: string) => Promise<DDLResult>;
  createTable: (databaseName: string, schema: string, tableName: string, fields: RecordInput[]) => Promise<DDLResult>;
  renameTable: (databaseName: string, schema: string | undefined, oldName: string, newName: string) => Promise<DDLResult>;
  deleteTable: (databaseName: string, schema: string | undefined, tableName: string) => Promise<DDLResult>;
  clearTableData: (databaseName: string, schema: string | undefined, tableName: string, mode: 'truncate' | 'delete') => Promise<DDLResult>;
  copyTable: (databaseName: string, schema: string | undefined, sourceTable: string, targetTable: string, copyData: boolean) => Promise<DDLResult>;
  dropCollection: (databaseName: string, collectionName: string) => Promise<DDLResult>;
  selectItem: (item: SelectedItem | null) => void;
  fetchDatabases: (connectionId: string) => Promise<string[]>;
  fetchSchemas: (connectionId: string, database: string) => Promise<string[]>;
  fetchTables: (connectionId: string, database: string, schema?: string) => Promise<StorageUnitSummary[]>;
  systemSchemas: string[];
  /** Node IDs where system objects are visible */
  showSystemObjectsFor: Set<string>;
  toggleSystemObjects: (nodeId: string) => void;
  fetchSystemSchemas: () => Promise<void>;
}

export interface DDLResult {
  success: boolean;
  message?: string;
}

/** Sidebar-facing shape of a storage unit (table, view, collection, or key). */
export interface StorageUnitSummary {
  name: string;
  type: string;
  /** Provisioned by the engine, an extension, or platform tooling — not user-authored. */
  system: boolean;
}

/** Transport-only marker the backend appends to System Object storage units. */
const SYSTEM_OBJECT_ATTRIBUTE_KEY = 'whodb:system-object';

/**
 * Map a GraphQL StorageUnit to the sidebar's table shape, promoting the
 * transport-only system-object attribute to a typed flag.
 */
export function storageUnitToSummary(unit: { Name: string; Attributes: { Key: string; Value: string }[] }): StorageUnitSummary {
  return {
    name: unit.Name,
    type: unit.Attributes.find(a => a.Key === 'Type')?.Value ?? 'table',
    system: unit.Attributes.some(a => a.Key === SYSTEM_OBJECT_ATTRIBUTE_KEY && a.Value === 'true'),
  };
}

/** Map auth store Type (e.g. "Postgres") to SqlDialect. */
function getDialect(): SqlDialect {
  const dbType = getAuthSession()?.type;
  const map: Record<string, SqlDialect> = {
    Postgres: 'POSTGRES', MySQL: 'MYSQL',
    SQLite3: 'SQLITE3', ClickHouse: 'CLICKHOUSE',
  };
  return map[dbType ?? ''] ?? 'POSTGRES';
}

/** Execute a DDL statement via ExecuteConfirmedSQL and return a result. */
async function executeDDL(sql: string, database?: string): Promise<DDLResult> {
  try {
    const { data, errors } = await graphqlClient.mutate<
      ExecuteConfirmedSqlMutation,
      ExecuteConfirmedSqlMutationVariables
    >({
      mutation: ExecuteConfirmedSqlDocument,
      variables: { query: sql, operationType: 'DDL' },
      context: database ? { database } : undefined,
    });
    if (errors?.length) {
      return { success: false, message: errors[0].message };
    }
    const msg = data?.ExecuteConfirmedSQL;
    if (msg?.Type === 'error') {
      return { success: false, message: msg.Text };
    }
    return { success: true, message: msg?.Text };
  } catch (err: any) {
    return { success: false, message: err.message ?? 'Unknown error' };
  }
}

const connectionTypeMap: Record<string, Connection['type']> = {
  Postgres: 'POSTGRES',
  MySQL: 'MYSQL',
  MongoDB: 'MONGODB',
  Redis: 'REDIS',
  ClickHouse: 'CLICKHOUSE',
};

function deriveConnection(session: AuthSessionSummary, createdAt: string): Connection {
  return {
    id: 'sealos',
    name: session.displayName || `${session.type} @ ${session.hostname}`,
    type: connectionTypeMap[session.type] ?? 'POSTGRES',
    host: session.hostname,
    port: session.port,
    user: '',
    password: '',
    database: session.database,
    createdAt,
  };
}

const createdAt = new Date().toISOString();

export const useConnectionStore = create<ConnectionState>((set) => ({
  connections: [],
  selectedItem: null,
  tableRefreshKey: 0,
  /** Increment table refresh key to trigger re-fetch in TableDetailView. */
  triggerTableRefresh: () => set((s) => ({ tableRefreshKey: s.tableRefreshKey + 1 })),
  collectionRefreshKey: 0,
  /** Increment collection refresh key to trigger re-fetch in CollectionViewProvider. */
  triggerCollectionRefresh: () => set((s) => ({ collectionRefreshKey: s.collectionRefreshKey + 1 })),
  sidebarRefreshKey: 0,
  /** Increment sidebar refresh key to trigger re-fetch of sidebar tree nodes (e.g. after DDL in editor). */
  triggerSidebarRefresh: () => set((s) => ({ sidebarRefreshKey: s.sidebarRefreshKey + 1 })),
  systemSchemas: [],
  showSystemObjectsFor: new Set<string>(),
  toggleSystemObjects: (nodeId) => set((state) => {
    const next = new Set(state.showSystemObjectsFor);
    if (next.has(nodeId)) next.delete(nodeId); else next.add(nodeId);
    return { showSystemObjectsFor: next };
  }),

  fetchSystemSchemas: async () => {
    const { data } = await graphqlClient.query<GetDatabaseMetadataQuery>({
      query: GetDatabaseMetadataDocument,
    });
    set({ systemSchemas: data?.DatabaseMetadata?.systemSchemas ?? [] });
  },

  selectItem: (item) => set({ selectedItem: item }),

  fetchDatabases: async (_connectionId) => {
    const session = useAuthStore.getState().session;
    if (!session) return [];
    const { data, error } = await graphqlClient.query<GetDatabaseQuery, GetDatabaseQueryVariables>({
      query: GetDatabaseDocument,
      variables: { type: session.type },
    });
    if (error) {
      console.error('[useConnectionStore] fetchDatabases failed:', error);
      throw error;
    }
    return data?.Database ?? [];
  },

  fetchSchemas: async (_connectionId, database) => {
    const session = useAuthStore.getState().session;
    if (!session) return [];
    const { data, error } = await graphqlClient.query<GetSchemaQuery>({
      query: GetSchemaDocument,
      context: { database },
    });
    if (error) {
      console.error('[useConnectionStore] fetchSchemas failed:', error);
      throw error;
    }
    return data?.Schema ?? [];
  },

  fetchTables: async (_connectionId, database, schema?) => {
    const session = useAuthStore.getState().session;
    if (!session) return [];
    const schemaParam = schema ?? database;
    const { data, error } = await graphqlClient.query<GetStorageUnitsQuery, GetStorageUnitsQueryVariables>({
      query: GetStorageUnitsDocument,
      variables: { schema: schemaParam },
      context: { database },
    });
    if (error) {
      console.error('[useConnectionStore] fetchTables failed:', error);
      throw error;
    }
    return data?.StorageUnit?.map(storageUnitToSummary) ?? [];
  },

  createDatabase: async (databaseName) => {
    if (getAuthSession()?.type === 'MongoDB') {
      return { success: false, message: 'MongoDB database creation requires creating the first collection' };
    }
    const sql = createDatabaseSQL(getDialect(), databaseName);
    return executeDDL(sql);
  },

  renameDatabase: async (oldName, newName) => {
    if (getAuthSession()?.type === 'MongoDB') {
      return { success: false, message: 'Rename database is not supported for MongoDB' };
    }
    const sql = renameDatabaseSQL(getDialect(), oldName, newName);
    if (!sql) {
      return { success: false, message: 'Rename database is not supported for this database type' };
    }
    return executeDDL(sql);
  },

  deleteDatabase: async (databaseName) => {
    if (getAuthSession()?.type === 'MongoDB') {
      try {
        const { data, errors } = await graphqlClient.query<
          RawExecuteQuery,
          RawExecuteQueryVariables
        >({
          query: RawExecuteDocument,
          variables: { query: buildMongoDropDatabaseCommand() },
          fetchPolicy: 'no-cache',
          context: { database: databaseName },
        });
        if (errors?.length) {
          return { success: false, message: errors[0].message };
        }
        const acknowledged = data?.RawExecute?.Rows?.[0]?.[0];
        return { success: acknowledged === 'true' };
      } catch (err: any) {
        return { success: false, message: err.message ?? 'Unknown error' };
      }
    }
    const sql = dropDatabaseSQL(getDialect(), databaseName);
    return executeDDL(sql);
  },

  createTable: async (databaseName, schema, tableName, fields) => {
    try {
      const { data, errors } = await graphqlClient.mutate<
        AddStorageUnitMutation,
        AddStorageUnitMutationVariables
      >({
        mutation: AddStorageUnitDocument,
        variables: { schema, storageUnit: tableName, fields },
        context: { database: databaseName },
      });
      if (errors?.length) {
        return { success: false, message: errors[0].message };
      }
      return { success: data?.AddStorageUnit.Status ?? false };
    } catch (err: any) {
      return { success: false, message: err.message ?? 'Unknown error' };
    }
  },

  renameTable: async (databaseName, schema, oldName, newName) => {
    const sql = renameTableSQL(getDialect(), oldName, newName, schema);
    return executeDDL(sql, databaseName);
  },

  deleteTable: async (databaseName, schema, tableName) => {
    const sql = dropTableSQL(getDialect(), tableName, schema);
    return executeDDL(sql, databaseName);
  },

  clearTableData: async (databaseName, schema, tableName, mode) => {
    const dialect = getDialect();
    const sql = mode === 'truncate'
      ? truncateTableSQL(dialect, tableName, schema)
      : deleteAllRowsSQL(dialect, tableName, schema);
    return executeDDL(sql, databaseName);
  },

  copyTable: async (databaseName, schema, sourceTable, targetTable, copyData) => {
    const dialect = getDialect();
    const sql = copyData
      ? copyTableWithDataSQL(dialect, sourceTable, targetTable, schema)
      : copyTableStructureSQL(dialect, sourceTable, targetTable, schema);
    const statements = sql.split('\n').filter(s => s.trim());
    for (const stmt of statements) {
      const result = await executeDDL(stmt, databaseName);
      if (!result.success) return result;
    }
    return { success: true };
  },

  /** Drop a MongoDB collection via RawExecute in the selected database context. */
  dropCollection: async (databaseName, collectionName) => {
    try {
      const { data, errors } = await graphqlClient.query<
        RawExecuteQuery,
        RawExecuteQueryVariables
      >({
        query: RawExecuteDocument,
        variables: { query: buildMongoCollectionCommand(collectionName, 'drop') },
        fetchPolicy: 'no-cache',
        context: { database: databaseName },
      });
      if (errors?.length) {
        return { success: false, message: errors[0].message };
      }
      const acknowledged = data?.RawExecute?.Rows?.[0]?.[0];
      return { success: acknowledged === 'true' };
    } catch (err: any) {
      return { success: false, message: err.message ?? 'Unknown error' };
    }
  },
}));

// Keep `connections` in sync with auth credentials.
// The original ConnectionContext used useMemo(credentials) — this subscription is the Zustand equivalent.
useAuthStore.subscribe((s) => {
  const session = s.session;
  useConnectionStore.setState({
    connections: session ? [deriveConnection(session, createdAt)] : [],
  });
});
