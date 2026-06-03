import { beforeEach, describe, expect, it } from 'vitest'
import { useTabStore } from '@/stores/useTabStore'

const initialState = useTabStore.getState()

describe('useTabStore', () => {
  beforeEach(() => {
    useTabStore.setState(initialState, true)
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

  it('tracks unsaved database edits separately from query dirtiness', () => {
    const tabId = useTabStore.getState().openTab({
      type: 'table',
      title: 'orders',
      connectionId: 'sealos',
      databaseName: 'app',
      tableName: 'orders',
      isDirty: true,
    })

    useTabStore.getState().setTabUnsavedDatabaseEdits(tabId, 2)

    expect(useTabStore.getState().tabs[0]).toMatchObject({
      isDirty: true,
      hasUnsavedDatabaseEdits: true,
      unsavedDatabaseEditCount: 2,
    })
  })

  it('runs registered database edit discarders and clears edit state', () => {
    const tabId = useTabStore.getState().openTab({
      type: 'collection',
      title: 'users',
      connectionId: 'sealos',
      databaseName: 'app',
      collectionName: 'users',
    })
    let discardCount = 0

    useTabStore.getState().registerDatabaseEditDiscarder(tabId, () => { discardCount += 1 })
    useTabStore.getState().setTabUnsavedDatabaseEdits(tabId, 3)
    useTabStore.getState().discardUnsavedDatabaseEdits([tabId])

    expect(discardCount).toBe(1)
    expect(useTabStore.getState().tabs[0]).toMatchObject({
      hasUnsavedDatabaseEdits: false,
      unsavedDatabaseEditCount: undefined,
    })
  })

  it('removes database edit discarders when tabs close', () => {
    const tabId = useTabStore.getState().openTab({
      type: 'table',
      title: 'orders',
      connectionId: 'sealos',
      databaseName: 'app',
      tableName: 'orders',
    })

    useTabStore.getState().registerDatabaseEditDiscarder(tabId, () => {})
    useTabStore.getState().closeTab(tabId)

    expect(useTabStore.getState().databaseEditDiscarders[tabId]).toBeUndefined()
  })
})
