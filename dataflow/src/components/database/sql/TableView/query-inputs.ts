import {
  SortDirection,
  WhereConditionType,
  type SortCondition,
  type WhereCondition,
} from '@graphql'
import type { FilterCondition } from './types'

/** Builds the GraphQL sort input used by SQL table row queries and exports. */
export function buildTableSortInput(
  sortColumn: string | null,
  sortDirection: 'asc' | 'desc' | null,
): SortCondition[] | undefined {
  if (!sortColumn || !sortDirection) return undefined
  return [{
    Column: sortColumn,
    Direction: sortDirection === 'asc' ? SortDirection.Asc : SortDirection.Desc,
  }]
}

/** Builds the GraphQL where input used by SQL table row queries and exports. */
export function buildTableWhereInput(
  filterConditions: FilterCondition[],
  columnTypes: Record<string, string> | undefined,
): WhereCondition | undefined {
  if (filterConditions.length === 0) return undefined

  const noValueOperators = ['IS NULL', 'IS NOT NULL']
  const atomicConditions: WhereCondition[] = filterConditions
    .filter((fc) => fc.column && fc.operator && (noValueOperators.includes(fc.operator) || fc.value !== ''))
    .map((fc) => ({
      Type: WhereConditionType.Atomic,
      Atomic: {
        Key: fc.column,
        Operator: fc.operator,
        Value: fc.value ?? '',
        ColumnType: columnTypes?.[fc.column] ?? 'string',
      },
    }))

  if (atomicConditions.length === 1) return atomicConditions[0]
  if (atomicConditions.length > 1) {
    return { Type: WhereConditionType.And, And: { Children: atomicConditions } }
  }
  return undefined
}
