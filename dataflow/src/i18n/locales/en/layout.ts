import { zhLayoutMessages } from '../zh/layout'

type MessageShape<T extends Record<string, unknown>> = {
  [K in keyof T]: string
}

export const enLayoutMessages = {
  'layout.activity.connections': 'Workbench',
  'layout.activity.analysis': 'Dashboard',
  'layout.empty.noTabsTitle': 'No tabs open',
  'layout.empty.noTabsDescription': 'Select a table from the sidebar or create a new query',
  'layout.invalid.tableConfig': 'Invalid table configuration',
  'layout.invalid.collectionConfig': 'Invalid collection configuration',
  'layout.invalid.databaseConfig': 'Invalid database configuration',
  'layout.invalid.unknownTabType': 'Unknown tab type',
  'layout.tab.close': 'Close tab',
  'layout.tab.closeOthers': 'Close Others',
  'layout.tab.closeAll': 'Close All',
  'layout.tab.newQuery': 'New Query',
  'layout.leaveGuard.title': 'Discard unsaved database edits?',
  'layout.leaveGuard.message': '{count} tab(s) have unsaved database edits. Discard them and continue?',
} satisfies MessageShape<typeof zhLayoutMessages>
