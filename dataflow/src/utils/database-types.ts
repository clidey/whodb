import { TypeCategory, type GetDatabaseMetadataQuery } from '@graphql'

type DatabaseMetadataValue = NonNullable<GetDatabaseMetadataQuery['DatabaseMetadata']>

export interface ColumnTypeOption {
  id: string
  label: string
}

const TEXT_TYPE_PREFERENCES = ['TEXT', 'String'] as const
const PRIMARY_TYPE_PREFERENCES = ['INT', 'INTEGER', 'BIGINT', 'Int64', 'Int32', 'UInt64', 'UInt32'] as const

/** Returns type selector options in the order provided by the active database plugin. */
export function getColumnTypeOptions(metadata: DatabaseMetadataValue | null | undefined): ColumnTypeOption[] {
  return metadata?.typeDefinitions.map((definition) => ({
    id: definition.id,
    label: definition.label,
  })) ?? []
}

/** Returns the default text-like type from the active database metadata. */
export function getDefaultTextColumnType(metadata: DatabaseMetadataValue | null | undefined): string {
  return getPreferredType(metadata, TypeCategory.Text, TEXT_TYPE_PREFERENCES)
}

/** Returns a primary-key-friendly type from the active database metadata. */
export function getDefaultPrimaryColumnType(metadata: DatabaseMetadataValue | null | undefined): string {
  return getPreferredType(metadata, TypeCategory.Numeric, PRIMARY_TYPE_PREFERENCES) || getDefaultTextColumnType(metadata)
}

function getPreferredType(
  metadata: DatabaseMetadataValue | null | undefined,
  category: TypeCategory,
  preferredIds: readonly string[],
): string {
  if (!metadata) return ''

  for (const preferredId of preferredIds) {
    const match = metadata.typeDefinitions.find((definition) => definition.id === preferredId)
    if (match) return match.id
  }

  return metadata.typeDefinitions.find((definition) => definition.category === category)?.id ?? ''
}
