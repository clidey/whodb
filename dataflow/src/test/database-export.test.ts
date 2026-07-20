import { describe, expect, it } from 'vitest'

import {
  buildDatabaseExportPlan,
  filterDatabaseExportUnits,
  formatDatabaseExportEntryName,
} from '@/utils/database-export'

describe('buildDatabaseExportPlan', () => {
  it('exports every non-system PostgreSQL schema in a database', () => {
    expect(buildDatabaseExportPlan({
      connectionType: 'POSTGRES',
      fallbackSchema: 'public',
      allSchemas: ['public', 'analytics', 'pg_catalog'],
      systemSchemas: ['pg_catalog'],
      includeSystemSchemas: false,
    })).toEqual(['public', 'analytics'])
  })

  it('keeps single-schema export behavior for MySQL-style databases', () => {
    expect(buildDatabaseExportPlan({
      connectionType: 'MYSQL',
      fallbackSchema: 'app_db',
      allSchemas: ['ignored'],
      systemSchemas: [],
      includeSystemSchemas: false,
    })).toEqual(['app_db'])
  })
})

describe('filterDatabaseExportUnits', () => {
  const units = [
    { name: 'orders', system: false },
    { name: 'postgres_log', system: true },
    { name: 'pg_stat_statements', system: true },
  ]

  it('excludes system objects when they are not revealed', () => {
    expect(filterDatabaseExportUnits(units, false)).toEqual([
      { name: 'orders', system: false },
    ])
  })

  it('keeps system objects when the user has revealed them', () => {
    expect(filterDatabaseExportUnits(units, true)).toEqual(units)
  })
})

describe('formatDatabaseExportEntryName', () => {
  it('groups PostgreSQL tables under schema directories', () => {
    expect(formatDatabaseExportEntryName('POSTGRES', 'analytics', 'events', 'sql')).toBe(
      'analytics/events.sql',
    )
  })

  it('keeps flat filenames for MySQL exports', () => {
    expect(formatDatabaseExportEntryName('MYSQL', 'app_db', 'events', 'csv')).toBe(
      'events.csv',
    )
  })
})
