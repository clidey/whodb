import { describe, expect, it } from 'vitest'

import {
  buildImportSqlInput,
  hasSqlScriptSource,
  isAcceptedSqlFile,
} from '@/components/database/import/sql-import-utils'

describe('SQL import source utilities', () => {
  it('accepts only .sql file names', () => {
    expect(isAcceptedSqlFile(new File(['SELECT 1;'], 'seed.sql'))).toBe(true)
    expect(isAcceptedSqlFile(new File(['SELECT 1;'], 'seed.SQL'))).toBe(true)
    expect(isAcceptedSqlFile(new File(['id,name'], 'seed.csv'))).toBe(false)
  })

  it('builds file upload input without adding script text', () => {
    const file = new File(['SELECT 1;'], 'seed.sql')

    expect(buildImportSqlInput({ kind: 'file', file, filename: file.name, preview: 'SELECT 1;' })).toEqual({
      File: file,
      Filename: 'seed.sql',
    })
  })

  it('builds script input after converting to editable text', () => {
    expect(buildImportSqlInput({ kind: 'text', script: 'SELECT 1;' })).toEqual({
      Script: 'SELECT 1;',
    })
  })

  it('treats blank sources as not executable', () => {
    expect(hasSqlScriptSource(null)).toBe(false)
    expect(hasSqlScriptSource({ kind: 'file', file: new File([''], 'empty.sql'), filename: 'empty.sql', preview: ' ' })).toBe(false)
    expect(hasSqlScriptSource({ kind: 'text', script: ' ' })).toBe(false)
    expect(hasSqlScriptSource({ kind: 'text', script: 'SELECT 1;' })).toBe(true)
  })
})
