import { describe, expect, it } from 'vitest'

import {
  getColumnTypeOptions,
  getDefaultPrimaryColumnType,
  getDefaultTextColumnType,
} from '@/utils/database-types'
import { TypeCategory, type GetDatabaseMetadataQuery } from '@graphql'

type Metadata = NonNullable<GetDatabaseMetadataQuery['DatabaseMetadata']>

const metadata: Metadata = {
  __typename: 'DatabaseMetadata',
  databaseType: 'postgres',
  operators: [],
  systemSchemas: [],
  aliasMap: [],
  capabilities: {
    __typename: 'Capabilities',
    supportsScratchpad: true,
    supportsChat: true,
    supportsGraph: true,
    supportsSchema: true,
    supportsDatabaseSwitch: true,
    supportsModifiers: true,
  },
  typeDefinitions: [
    {
      __typename: 'TypeDefinition',
      id: 'VARCHAR',
      label: 'VARCHAR',
      hasLength: true,
      hasPrecision: false,
      category: TypeCategory.Text,
    },
    {
      __typename: 'TypeDefinition',
      id: 'TEXT',
      label: 'TEXT',
      hasLength: false,
      hasPrecision: false,
      category: TypeCategory.Text,
    },
    {
      __typename: 'TypeDefinition',
      id: 'INT',
      label: 'INT',
      hasLength: false,
      hasPrecision: false,
      category: TypeCategory.Numeric,
    },
  ],
}

describe('database type metadata helpers', () => {
  it('keeps type selector options metadata-driven', () => {
    expect(getColumnTypeOptions(metadata)).toEqual([
      { id: 'VARCHAR', label: 'VARCHAR' },
      { id: 'TEXT', label: 'TEXT' },
      { id: 'INT', label: 'INT' },
    ])
  })

  it('chooses preferred text and primary-key types from metadata', () => {
    expect(getDefaultTextColumnType(metadata)).toBe('TEXT')
    expect(getDefaultPrimaryColumnType(metadata)).toBe('INT')
  })

  it('returns empty defaults when metadata is unavailable', () => {
    expect(getColumnTypeOptions(null)).toEqual([])
    expect(getDefaultTextColumnType(null)).toBe('')
    expect(getDefaultPrimaryColumnType(null)).toBe('')
  })
})
