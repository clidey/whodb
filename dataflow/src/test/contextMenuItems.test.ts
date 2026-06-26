import { describe, expect, it, vi } from 'vitest'

import {
  getConnectionMenuItems,
  getDatabaseMenuItems,
  getSchemaMenuItems,
  getTableMenuItems,
} from '@/components/sidebar/contextMenuItems'
import type { ContextMenuItem } from '@/components/ui/ContextMenu'

function labels(items: ContextMenuItem[]) {
  return items.flatMap((item) => ('label' in item ? [item.label] : []))
}

describe('MongoDB context menu items', () => {
  const callbacks = {
    onAction: vi.fn(),
    t: (key: string) => key,
  }

  it('does not expose unsupported MongoDB database creation from the connection menu', () => {
    const items = getConnectionMenuItems('MONGODB', callbacks)

    expect(labels(items)).toContain('sidebar.menu.newDatabase')
  })

  it('keeps collection actions and only exposes supported database-level MongoDB actions', () => {
    const items = getDatabaseMenuItems('MONGODB', callbacks)
    const itemLabels = labels(items)

    expect(itemLabels).toContain('sidebar.menu.newCollection')
    expect(itemLabels).toContain('sidebar.menu.deleteDatabase')
    expect(itemLabels).not.toContain('sidebar.menu.import')
    expect(itemLabels).not.toContain('sidebar.menu.renameDatabase')
  })
})

describe('SQL database context menu items', () => {
  const callbacks = {
    onAction: vi.fn(),
    t: (key: string) => key,
  }

  it('still exposes Postgres database rename support', () => {
    const items = getDatabaseMenuItems('POSTGRES', callbacks)

    expect(labels(items)).toContain('sidebar.menu.renameDatabase')
  })

  it('exposes database import from supported SQL database contexts', () => {
    expect(labels(getDatabaseMenuItems('MYSQL', callbacks))).toContain('sidebar.menu.import')
    expect(labels(getDatabaseMenuItems('POSTGRES', callbacks))).toContain('sidebar.menu.import')
    expect(labels(getDatabaseMenuItems('CLICKHOUSE', callbacks))).toContain('sidebar.menu.import')
  })

  it('exposes import from Postgres schema and SQL table contexts', () => {
    expect(labels(getSchemaMenuItems('POSTGRES', callbacks))).toContain('sidebar.menu.import')
    expect(labels(getTableMenuItems('POSTGRES', callbacks))).toContain('sidebar.menu.import')
  })
})
