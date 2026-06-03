import { beforeEach, describe, expect, it } from 'vitest'
import { useTabStore } from '@/stores/useTabStore'

const initialState = useTabStore.getState()

describe('useTabStore', () => {
  beforeEach(() => {
    useTabStore.setState(initialState)
  })

  it('keeps table and view workspace tabs distinct when they share a name', () => {
    const tableTabId = useTabStore.getState().openTab({
      type: 'table',
      title: 'orders',
      connectionId: 'sealos',
      databaseName: 'app',
      schemaName: 'public',
      tableName: 'orders',
      storageUnitType: 'table',
    })

    const viewTabId = useTabStore.getState().openTab({
      type: 'table',
      title: 'orders',
      connectionId: 'sealos',
      databaseName: 'app',
      schemaName: 'public',
      tableName: 'orders',
      storageUnitType: 'view',
    })

    expect(viewTabId).not.toBe(tableTabId)
    expect(useTabStore.getState().tabs).toHaveLength(2)
  })

  it('reuses matching view workspace tabs', () => {
    const firstTabId = useTabStore.getState().openTab({
      type: 'table',
      title: 'daily_orders',
      connectionId: 'sealos',
      databaseName: 'app',
      schemaName: 'public',
      tableName: 'daily_orders',
      storageUnitType: 'view',
    })

    const secondTabId = useTabStore.getState().openTab({
      type: 'table',
      title: 'daily_orders',
      connectionId: 'sealos',
      databaseName: 'app',
      schemaName: 'public',
      tableName: 'daily_orders',
      storageUnitType: 'view',
    })

    expect(secondTabId).toBe(firstTabId)
    expect(useTabStore.getState().tabs).toHaveLength(1)
  })
})
