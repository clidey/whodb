import { useCallback, useEffect, useRef, useState } from 'react'
import { useConnectionStore } from '@/stores/useConnectionStore'
import {
  useGetStorageUnitRowsLazyQuery,
  WhereConditionType,
  SortDirection,
  type WhereCondition,
  type SortCondition,
} from '@graphql'
import { transformRowsResult, type TableData } from '@/utils/graphql-transforms'
import { resolveSchemaParam } from '@/utils/database-features'
import { useI18n } from '@/i18n/useI18n'
import type { FilterCondition } from './types'

interface UseDataQueryParams {
  connectionId: string
  databaseName: string
  schema?: string
  tableName: string
  currentPage: number
  pageSize: number
  sortColumn: string | null
  sortDirection: 'asc' | 'desc' | null
  filterConditions: FilterCondition[]
  visibleColumnsCount: number
  /** Called once when query returns columns and no visible columns are set yet. */
  onInitVisibleColumns: (columns: string[]) => void
}

/** State returned by useDataQuery. */
export interface DataQueryState {
  loading: boolean
  data: TableData | null
  error: string | null
  primaryKey: string | null
  foreignKeyColumns: string[]
  total: number
  totalPages: number
  canEdit: boolean
}

/** Actions returned by useDataQuery. */
export interface DataQueryActions {
  refresh: () => void
  handleSubmitRequest: (overridePageOffset?: number) => Promise<void>
}

/** Hook that owns data fetching, loading/error state, and race condition prevention for TableView. */
export function useDataQuery(params: UseDataQueryParams): { state: DataQueryState; actions: DataQueryActions } {
  const { t } = useI18n()
  const {
    connectionId,
    databaseName,
    schema,
    tableName,
    currentPage,
    pageSize,
    sortColumn,
    sortDirection,
    filterConditions,
    visibleColumnsCount,
    onInitVisibleColumns,
  } = params

  const { connections, tableRefreshKey } = useConnectionStore()

  const [getRows] = useGetStorageUnitRowsLazyQuery({ fetchPolicy: 'no-cache' })

  const [loading, setLoading] = useState(true)
  const [data, setData] = useState<TableData | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [primaryKey, setPrimaryKey] = useState<string | null>(null)
  const [foreignKeyColumns, setForeignKeyColumns] = useState<string[]>([])
  const [refreshKey, setRefreshKey] = useState(0)

  const latestRequestIdRef = useRef(0)
  const filterConditionsRef = useRef(filterConditions)

  // Keep refs in sync
  useEffect(() => { filterConditionsRef.current = filterConditions }, [filterConditions])

  const handleSubmitRequest = useCallback(async (overridePageOffset?: number) => {
    const conn = connections.find((c) => c.id === connectionId)
    if (!conn) return

    setLoading(true)
    setError(null)

    latestRequestIdRef.current += 1
    const thisRequestId = latestRequestIdRef.current

    const graphqlSchema = resolveSchemaParam(conn.type, databaseName, schema)

    // Build sort condition
    const sort: SortCondition[] | undefined =
      sortColumn && sortDirection
        ? [{ Column: sortColumn, Direction: sortDirection === 'asc' ? SortDirection.Asc : SortDirection.Desc }]
        : undefined

    // Build filter where condition
    const currentFilters = filterConditionsRef.current
    let filterWhere: WhereCondition | undefined
    if (currentFilters.length > 0) {
      const noValueOperators = ['IS NULL', 'IS NOT NULL']
      const atomicConditions: WhereCondition[] = currentFilters
        .filter((fc) => fc.column && fc.operator && (noValueOperators.includes(fc.operator) || fc.value !== ''))
        .map((fc) => ({
          Type: WhereConditionType.Atomic,
          Atomic: {
            Key: fc.column,
            Operator: fc.operator,
            Value: fc.value ?? '',
            ColumnType: data?.columnTypes[fc.column] ?? 'string',
          },
        }))

      if (atomicConditions.length === 1) {
        filterWhere = atomicConditions[0]
      } else if (atomicConditions.length > 1) {
        filterWhere = { Type: WhereConditionType.And, And: { Children: atomicConditions } }
      }
    }

    const where = filterWhere

    try {
      const { data: result, error: queryError } = await getRows({
        variables: {
          schema: graphqlSchema,
          storageUnit: tableName,
          where,
          sort,
          pageSize,
          pageOffset: overridePageOffset ?? (currentPage - 1) * pageSize,
        },
        context: { database: databaseName },
      })

      if (thisRequestId !== latestRequestIdRef.current) return

      if (queryError) {
        setError(queryError.message)
        return
      }

      if (result?.Row) {
        const tableData = transformRowsResult(result.Row)
        setData(tableData)
        setPrimaryKey(tableData.primaryKey)
        setForeignKeyColumns(tableData.foreignKeyColumns)
        if (visibleColumnsCount === 0 && tableData.columns.length > 0) {
          onInitVisibleColumns(tableData.columns)
        }
      }
    } catch (err: any) {
      if (thisRequestId !== latestRequestIdRef.current) return
      setError(err.message || t('sql.table.errorFetchData'))
    } finally {
      if (thisRequestId === latestRequestIdRef.current) {
        setLoading(false)
      }
    }
  }, [connections, connectionId, databaseName, schema, tableName, sortColumn, sortDirection, pageSize, currentPage, getRows, visibleColumnsCount, onInitVisibleColumns, t])

  // Fetch on mount and when data-changing params change
  useEffect(() => {
    handleSubmitRequest()
  }, [handleSubmitRequest, refreshKey, tableRefreshKey])

  const refresh = useCallback(() => {
    setRefreshKey(prev => prev + 1)
  }, [])

  const canEdit = data ? !data.disableUpdate : false
  const total = data?.total || 0
  const totalPages = Math.ceil(total / pageSize)

  return {
    state: { loading, data, error, primaryKey, foreignKeyColumns, total, totalPages, canEdit },
    actions: { refresh, handleSubmitRequest },
  }
}
