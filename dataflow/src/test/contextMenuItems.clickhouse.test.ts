import { describe, expect, it, vi } from 'vitest'

import type { ContextMenuItem } from '@/components/ui/ContextMenu'
import {
  getDatabaseMenuItems,
  getTableMenuItems,
} from '@/components/sidebar/contextMenuItems'

function labels(items: ContextMenuItem[]) {
  return items.flatMap((item) => ('label' in item ? [item.label] : []))
}

const callbacks = {
  onAction: vi.fn(),
  t: (key: string) => key,
}

describe('database menu exposure', () => {
  it('shows PostgreSQL database export once full-database export is supported', () => {
    const items = getDatabaseMenuItems('POSTGRES', callbacks)

    expect(labels(items)).toContain('sidebar.menu.exportDatabase')
  })
})

describe('ClickHouse table menu exposure', () => {
  it('keeps only stable table actions exposed', () => {
    const items = getTableMenuItems('CLICKHOUSE', callbacks)
    const itemLabels = labels(items)

    expect(itemLabels).toContain('sidebar.menu.import')
    expect(itemLabels).toContain('sidebar.menu.exportData')
    expect(itemLabels).toContain('sidebar.menu.clearData')
    expect(itemLabels).toContain('sidebar.menu.deleteTable')
    expect(itemLabels).not.toContain('sidebar.menu.duplicateTable')
    expect(itemLabels).not.toContain('sidebar.menu.designTable')
    expect(itemLabels).not.toContain('sidebar.menu.renameTable')
  })
})
