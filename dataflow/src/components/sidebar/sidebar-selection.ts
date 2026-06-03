import type { Connection } from '@/stores/useConnectionStore'
import type { Tab } from '@/stores/useTabStore'
import type { TreeNodeData } from './SidebarTree/types'

interface SidebarTreeLabels {
  redisKeysFolder: string
  tablesFolder: string
  viewsFolder: string
}

function connectionNode(connectionId: string, connections: Connection[]): TreeNodeData | null {
  const connection = connections.find((item) => item.id === connectionId)
  if (!connection) return null

  return {
    type: 'connection',
    id: connection.id,
    name: connection.name,
    connectionId: connection.id,
    metadata: {},
  }
}

function databaseNode(connectionId: string, databaseName: string | undefined): TreeNodeData | null {
  if (!databaseName) return null

  return {
    type: 'database',
    id: `${connectionId}-${databaseName}`,
    name: databaseName,
    parentId: connectionId,
    connectionId,
    metadata: { database: databaseName },
  }
}

function schemaNode(connectionId: string, databaseName: string | undefined, schemaName: string | undefined): TreeNodeData | null {
  if (!databaseName || !schemaName) return null

  const databaseId = `${connectionId}-${databaseName}`
  return {
    type: 'schema',
    id: `${databaseId}-${schemaName}`,
    name: schemaName,
    parentId: databaseId,
    connectionId,
    metadata: { database: databaseName, schema: schemaName },
  }
}

function storageUnitNode(tab: Tab): TreeNodeData | null {
  if (!tab.databaseName || !tab.tableName) return null

  const databaseId = `${tab.connectionId}-${tab.databaseName}`
  const storageUnitType = tab.storageUnitType ?? 'table'
  const parentId = tab.schemaName
    ? `${databaseId}-${tab.schemaName}-${storageUnitType === 'view' ? 'views' : 'tables'}`
    : databaseId

  return {
    type: storageUnitType,
    id: `${parentId}-${tab.tableName}`,
    name: tab.tableName,
    parentId,
    connectionId: tab.connectionId,
    metadata: {
      database: tab.databaseName,
      schema: tab.schemaName,
      table: tab.tableName,
    },
  }
}

function collectionNode(tab: Tab): TreeNodeData | null {
  if (!tab.databaseName || !tab.collectionName) return null

  const databaseId = `${tab.connectionId}-${tab.databaseName}`
  return {
    type: 'collection',
    id: `${databaseId}-${tab.collectionName}`,
    name: tab.collectionName,
    parentId: databaseId,
    connectionId: tab.connectionId,
    metadata: { database: tab.databaseName, table: tab.collectionName },
  }
}

function redisKeyNode(tab: Tab): TreeNodeData | null {
  if (!tab.databaseName || !tab.tableName) return null

  const databaseId = `${tab.connectionId}-${tab.databaseName}`
  const keysFolderId = `${databaseId}-keys`
  return {
    type: 'redis_key',
    id: `${keysFolderId}-${tab.tableName}`,
    name: tab.tableName,
    parentId: keysFolderId,
    connectionId: tab.connectionId,
    metadata: { database: tab.databaseName },
  }
}

/** Derives the sidebar focus that should represent the active workspace tab. */
export function getSidebarSelectionForTab(tab: Tab | null, connections: Connection[]): TreeNodeData | null {
  if (!tab) return null

  if (tab.type === 'query') {
    return (
      schemaNode(tab.connectionId, tab.databaseName, tab.schemaName) ??
      databaseNode(tab.connectionId, tab.databaseName) ??
      connectionNode(tab.connectionId, connections)
    )
  }

  if (tab.type === 'table') {
    return storageUnitNode(tab)
  }

  if (tab.type === 'collection') {
    return collectionNode(tab)
  }

  if (tab.type === 'redis_key_detail') {
    return redisKeyNode(tab)
  }

  return null
}

/** Returns the collapsed ancestors that must be expanded to show a sidebar focus. */
export function getSidebarRevealAncestors(
  selection: TreeNodeData | null,
  connections: Connection[],
  labels: SidebarTreeLabels,
): TreeNodeData[] {
  if (!selection) return []

  const connection = connectionNode(selection.connectionId, connections)
  if (!connection) return []

  const database = databaseNode(selection.connectionId, selection.metadata.database)
  const schema = schemaNode(selection.connectionId, selection.metadata.database, selection.metadata.schema)

  if (selection.type === 'connection') return []
  if (selection.type === 'database') return [connection]
  if (selection.type === 'schema') return [connection, database].filter((node): node is TreeNodeData => Boolean(node))

  if (selection.type === 'redis_key') {
    if (!database) return [connection]
    return [
      connection,
      database,
      {
        type: 'redis_keys_folder',
        id: `${database.id}-keys`,
        name: labels.redisKeysFolder,
        parentId: database.id,
        connectionId: selection.connectionId,
        metadata: { database: selection.metadata.database },
      },
    ]
  }

  if ((selection.type === 'table' || selection.type === 'view') && schema) {
    const folderType = selection.type === 'view' ? 'view_folder' : 'table_folder'
    return [
      connection,
      database,
      schema,
      {
        type: folderType,
        id: `${schema.id}-${selection.type === 'view' ? 'views' : 'tables'}`,
        name: selection.type === 'view' ? labels.viewsFolder : labels.tablesFolder,
        parentId: schema.id,
        connectionId: selection.connectionId,
        metadata: {
          database: selection.metadata.database,
          schema: selection.metadata.schema,
        },
      },
    ].filter((node): node is TreeNodeData => Boolean(node))
  }

  if (selection.type === 'table' || selection.type === 'view' || selection.type === 'collection') {
    return [connection, database].filter((node): node is TreeNodeData => Boolean(node))
  }

  return []
}
