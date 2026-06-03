import { describe, expect, it } from 'vitest'
import { getSidebarRevealAncestors, getSidebarSelectionForTab } from '@/components/sidebar/sidebar-selection'
import type { Connection } from '@/stores/useConnectionStore'
import type { Tab } from '@/stores/useTabStore'

const connections: Connection[] = [
  {
    id: 'sealos',
    name: 'Postgres @ localhost',
    type: 'POSTGRES',
    host: 'localhost',
    port: '5432',
    user: '',
    password: '',
    database: 'app',
    createdAt: '2026-06-03T00:00:00.000Z',
  },
]

function tab(overrides: Partial<Tab>): Tab {
  return {
    id: 'tab-1',
    type: 'query',
    title: 'Query',
    connectionId: 'sealos',
    ...overrides,
  }
}

describe('getSidebarSelectionForTab', () => {
  it('focuses the active relational table tab', () => {
    const selection = getSidebarSelectionForTab(
      tab({
        type: 'table',
        databaseName: 'app',
        schemaName: 'public',
        tableName: 'orders',
        storageUnitType: 'table',
      }),
      connections,
    )

    expect(selection).toMatchObject({
      type: 'table',
      id: 'sealos-app-public-tables-orders',
      parentId: 'sealos-app-public-tables',
      metadata: { database: 'app', schema: 'public', table: 'orders' },
    })
  })

  it('focuses the active relational view tab', () => {
    const selection = getSidebarSelectionForTab(
      tab({
        type: 'table',
        databaseName: 'app',
        schemaName: 'public',
        tableName: 'daily_orders',
        storageUnitType: 'view',
      }),
      connections,
    )

    expect(selection).toMatchObject({
      type: 'view',
      id: 'sealos-app-public-views-daily_orders',
      parentId: 'sealos-app-public-views',
    })
  })

  it('focuses the active collection tab', () => {
    const selection = getSidebarSelectionForTab(
      tab({
        type: 'collection',
        databaseName: 'app',
        collectionName: 'events',
      }),
      connections,
    )

    expect(selection).toMatchObject({
      type: 'collection',
      id: 'sealos-app-events',
      parentId: 'sealos-app',
      metadata: { database: 'app', table: 'events' },
    })
  })

  it('focuses the nearest query context', () => {
    expect(
      getSidebarSelectionForTab(tab({ type: 'query', databaseName: 'app', schemaName: 'public' }), connections),
    ).toMatchObject({ type: 'schema', id: 'sealos-app-public' })

    expect(getSidebarSelectionForTab(tab({ type: 'query', databaseName: 'app' }), connections)).toMatchObject({
      type: 'database',
      id: 'sealos-app',
    })

    expect(getSidebarSelectionForTab(tab({ type: 'query' }), connections)).toMatchObject({
      type: 'connection',
      id: 'sealos',
    })
  })

  it('focuses the active Redis key tab', () => {
    const selection = getSidebarSelectionForTab(
      tab({
        type: 'redis_key_detail',
        databaseName: '0',
        tableName: 'session:1',
      }),
      connections,
    )

    expect(selection).toMatchObject({
      type: 'redis_key',
      id: 'sealos-0-keys-session:1',
      parentId: 'sealos-0-keys',
      metadata: { database: '0' },
    })
  })
})

describe('getSidebarRevealAncestors', () => {
  it('expands the path to a relational table inside a schema folder', () => {
    const selection = getSidebarSelectionForTab(
      tab({
        type: 'table',
        databaseName: 'app',
        schemaName: 'public',
        tableName: 'orders',
        storageUnitType: 'table',
      }),
      connections,
    )

    expect(getSidebarRevealAncestors(selection, connections, {
      redisKeysFolder: 'Keys',
      tablesFolder: 'Tables',
      viewsFolder: 'Views',
    })).toMatchObject([
      { type: 'connection', id: 'sealos' },
      { type: 'database', id: 'sealos-app' },
      { type: 'schema', id: 'sealos-app-public' },
      { type: 'table_folder', id: 'sealos-app-public-tables' },
    ])
  })

  it('expands the path to a Redis key folder', () => {
    const selection = getSidebarSelectionForTab(
      tab({
        type: 'redis_key_detail',
        databaseName: '0',
        tableName: 'session:1',
      }),
      connections,
    )

    expect(getSidebarRevealAncestors(selection, connections, {
      redisKeysFolder: 'Keys',
      tablesFolder: 'Tables',
      viewsFolder: 'Views',
    })).toMatchObject([
      { type: 'connection', id: 'sealos' },
      { type: 'database', id: 'sealos-0' },
      { type: 'redis_keys_folder', id: 'sealos-0-keys', name: 'Keys' },
    ])
  })

  it('only expands ancestors of the focused query context', () => {
    const selection = getSidebarSelectionForTab(
      tab({
        type: 'query',
        databaseName: 'app',
        schemaName: 'public',
      }),
      connections,
    )

    expect(getSidebarRevealAncestors(selection, connections, {
      redisKeysFolder: 'Keys',
      tablesFolder: 'Tables',
      viewsFolder: 'Views',
    })).toMatchObject([
      { type: 'connection', id: 'sealos' },
      { type: 'database', id: 'sealos-app' },
    ])
  })
})
