import { createContext, use, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useConnectionStore } from '@/stores/useConnectionStore'
import {
  SortDirection,
  useGetStorageUnitRowsLazyQuery,
  useGetColumnsLazyQuery,
  WhereConditionType,
  type SortCondition,
  type WhereCondition,
} from '@graphql'
import { resolveSchemaParam } from '@/utils/database-features'
import type { CollectionViewContextValue, MongoCollectionViewMode, MongoSortDirection } from './types'
import type { Alert } from '@/components/database/shared/types'
import type { FlatMongoFilter } from '@/components/database/mongodb/filter-collection.types'
import { useDocumentChangesetManager } from './useDocumentChangesetManager'
import { buildMongoTableColumns } from './mongo-table-utils'
import { useI18n } from '@/i18n/useI18n'
import { useColumnResize } from '@/components/database/shared/useColumnResize'

const CollectionViewCtx = createContext<CollectionViewContextValue | null>(null)

/** Hook to access CollectionView context. Throws if used outside CollectionViewProvider. */
export function useCollectionView(): CollectionViewContextValue {
  const ctx = use(CollectionViewCtx)
  if (!ctx) throw new Error('useCollectionView must be used within CollectionViewProvider')
  return ctx
}

interface CollectionViewProviderProps {
  connectionId: string
  databaseName: string
  collectionName: string
  children: ReactNode
}

/** Provider that owns all CollectionDetailView state, GraphQL operations, and handlers. */
export function CollectionViewProvider({ connectionId, databaseName, collectionName, children }: CollectionViewProviderProps) {
  const { t } = useI18n()
  const { connections, collectionRefreshKey } = useConnectionStore()

  // ---- GraphQL hooks ----
  const [getRows] = useGetStorageUnitRowsLazyQuery({ fetchPolicy: 'no-cache' })
  const [getColumns] = useGetColumnsLazyQuery({ fetchPolicy: 'no-cache' })

  // ---- Core state ----
  const [loading, setLoading] = useState(true)
  const [documents, setDocuments] = useState<any[]>([])
  const [error, setError] = useState<string | null>(null)
  const [viewMode, setViewModeState] = useState<MongoCollectionViewMode>('table')
  const [currentPage, setCurrentPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [pageSize, setPageSize] = useState(50)
  const [refreshKey, setRefreshKey] = useState(0)
  const [sampledFields, setSampledFields] = useState<string[]>([])
  const [hasSampledFields, setHasSampledFields] = useState(false)

  // ---- Sorting state ----
  const [sortColumn, setSortColumn] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<MongoSortDirection | null>(null)
  const [activeColumnMenu, setActiveColumnMenu] = useState<string | null>(null)

  // ---- Export state ----
  const [showExportModal, setShowExportModal] = useState(false)

  // ---- Filter state ----
  const [isFilterModalOpen, setIsFilterModalOpen] = useState(false)
  const [activeFilter, setActiveFilter] = useState<FlatMongoFilter>({})
  const [preferredFilterField, setPreferredFilterField] = useState<string | null>(null)

  // ---- Alert state ----
  const [alert, setAlert] = useState<Alert | null>(null)

  // ---- Alert helpers ----
  const showAlert = useCallback((title: string, message: string, type: Alert['type'] = 'info') => {
    setAlert({ title, message, type })
  }, [])

  const closeAlert = useCallback(() => setAlert(null), [])

  // ---- Refresh ----
  const refresh = useCallback(() => {
    setRefreshKey(prev => prev + 1)
  }, [])

  const pageOffset = (currentPage - 1) * pageSize

  // ---- Document changeset (add / edit / delete) ----
  const { state: changesetState, actions: changesetActions } = useDocumentChangesetManager({
    connectionId,
    databaseName,
    collectionName,
    documents,
    pageOffset,
    refresh,
    showAlert,
  })

  // ---- Discard guard ----
  const pendingReloadActionRef = useRef<null | (() => void)>(null)

  const runWithDiscardGuard = useCallback((action: () => void) => {
    if (!changesetState.hasPendingChanges) {
      action()
      return
    }
    pendingReloadActionRef.current = action
    changesetActions.setShowDiscardModal(true)
  }, [changesetActions, changesetState.hasPendingChanges])

  const confirmDiscardAndContinue = useCallback(() => {
    changesetActions.discardChanges()
    changesetActions.setShowDiscardModal(false)
    pendingReloadActionRef.current?.()
    pendingReloadActionRef.current = null
  }, [changesetActions])

  const setViewMode = useCallback((mode: MongoCollectionViewMode) => {
    setViewModeState(mode)
  }, [])

  const sampleCollectionFields = useCallback(async () => {
    const conn = connections.find(c => c.id === connectionId)
    if (!conn) return

    const graphqlSchema = resolveSchemaParam(conn.type, databaseName)
    const { data: result, error: queryError } = await getColumns({
      variables: {
        schema: graphqlSchema,
        storageUnit: collectionName,
      },
      context: { database: databaseName },
    })

    if (queryError) {
      setError(queryError.message)
      setHasSampledFields(true)
      return
    }

    setSampledFields(result?.Columns.map((column) => column.Name) ?? [])
    setHasSampledFields(true)
  }, [collectionName, connectionId, connections, databaseName, getColumns])

  useEffect(() => {
    setViewModeState('table')
    setCurrentPage(1)
    setSortColumn(null)
    setSortDirection(null)
    setActiveColumnMenu(null)
    setActiveFilter({})
    setPreferredFilterField(null)
    setSampledFields([])
    setHasSampledFields(false)
    changesetActions.discardChanges()
  }, [changesetActions.discardChanges, collectionName, connectionId, databaseName])

  useEffect(() => {
    if (viewMode !== 'table' || hasSampledFields) return
    void sampleCollectionFields()
  }, [hasSampledFields, sampleCollectionFields, viewMode])

  // ---- beforeunload guard ----
  useEffect(() => {
    if (!changesetState.hasPendingChanges) return

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [changesetState.hasPendingChanges])

  const tableColumns = useMemo(() => buildMongoTableColumns({
    sampledFields,
    documents: documents as Record<string, unknown>[],
    changes: changesetState.changes,
  }), [changesetState.changes, documents, sampledFields])
  const { columnWidths, resizingColumn, resizedColumns, handleResizeStart } = useColumnResize(tableColumns, {
    initialWidth: 160,
    minimumWidth: 80,
  })

  const availableFields = tableColumns

  // ---- Main data fetch ----
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      setError(null)

      const conn = connections.find(c => c.id === connectionId)
      if (!conn) {
        setError(t('common.error.connectionNotFound'))
        setLoading(false)
        return
      }

      const graphqlSchema = resolveSchemaParam(conn.type, databaseName)

      // Build WhereCondition from activeFilter
      // FilterCollectionModal outputs MongoDB-native format:
      //   $eq:    { field: value }
      //   $regex: { field: { $regex: "...", $options: "i" } }
      //   others: { field: { $gt: value } }
      const filterConditions: WhereCondition[] = []
      for (const [fieldName, condition] of Object.entries(activeFilter)) {
        if (condition === undefined || condition === null) continue
        if (typeof condition !== 'object' || Array.isArray(condition)) {
          // Primitive value -> $eq
          filterConditions.push({
            Type: WhereConditionType.Atomic,
            Atomic: { Key: fieldName, Operator: 'eq', Value: String(condition), ColumnType: 'string' },
          })
        } else {
          // Object with MongoDB operators: { $regex: "...", $options: "..." } or { $gt: value }
          for (const [op, val] of Object.entries(condition as Record<string, unknown>)) {
            if (op === '$options') continue // Skip $options (handled with $regex)
            const operator = op.replace('$', '')
            const value = Array.isArray(val) ? val.join(', ') : String(val ?? '')
            filterConditions.push({
              Type: WhereConditionType.Atomic,
              Atomic: { Key: fieldName, Operator: operator, Value: value, ColumnType: 'string' },
            })
          }
        }
      }

      let where: WhereCondition | undefined
      if (filterConditions.length === 1) {
        where = filterConditions[0]
      } else if (filterConditions.length > 1) {
        where = { Type: WhereConditionType.And, And: { Children: filterConditions } }
      }

      const sort: SortCondition[] | undefined =
        sortColumn && sortDirection
          ? [{ Column: sortColumn, Direction: sortDirection === 'asc' ? SortDirection.Asc : SortDirection.Desc }]
          : undefined

      try {
        const { data: result, error: queryError } = await getRows({
          variables: {
            schema: graphqlSchema,
            storageUnit: collectionName,
            where,
            sort,
            pageSize,
            pageOffset: (currentPage - 1) * pageSize,
          },
          context: { database: databaseName },
        })

        if (queryError) {
          setError(queryError.message)
          return
        }

        if (result?.Row) {
          const parsedDocs = result.Row.Rows.map(row => {
            try {
              return JSON.parse(row[0])
            } catch {
              return { _raw: row[0] }
            }
          })
          setDocuments(parsedDocs)
          setTotal(result.Row.TotalCount)
        }
      } catch (err: any) {
        setError(err.message || t('mongodb.error.fetchCollectionData'))
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [connectionId, databaseName, collectionName, connections, collectionRefreshKey, currentPage, pageSize, activeFilter, refreshKey, sortColumn, sortDirection, getRows, t])

  // ---- Page change ----
  const handlePageChange = useCallback((page: number) => {
    runWithDiscardGuard(() => setCurrentPage(page))
  }, [runWithDiscardGuard])

  // ---- Page size change ----
  const handlePageSizeChange = useCallback((size: number) => {
    runWithDiscardGuard(() => { setPageSize(size); setCurrentPage(1) })
  }, [runWithDiscardGuard])

  // ---- Sorting ----
  const handleSort = useCallback((column: string, direction: MongoSortDirection) => {
    runWithDiscardGuard(() => {
      setSortColumn(column)
      setSortDirection(direction)
      setActiveColumnMenu(null)
      setCurrentPage(1)
    })
  }, [runWithDiscardGuard])

  const clearSort = useCallback(() => {
    runWithDiscardGuard(() => {
      setSortColumn(null)
      setSortDirection(null)
      setActiveColumnMenu(null)
      setCurrentPage(1)
    })
  }, [runWithDiscardGuard])

  const setFilterModalOpen = useCallback((open: boolean) => {
    setIsFilterModalOpen(open)
    if (!open) setPreferredFilterField(null)
  }, [])

  const openFilterForField = useCallback((field: string) => {
    setPreferredFilterField(field)
    setActiveColumnMenu(null)
    setIsFilterModalOpen(true)
  }, [])

  // ---- Filter apply ----
  const handleFilterApply = useCallback((filter: FlatMongoFilter) => {
    runWithDiscardGuard(() => {
      setActiveFilter(filter)
      setPreferredFilterField(null)
      setCurrentPage(1)
    })
  }, [runWithDiscardGuard])

  // ---- Derived values ----
  const totalPages = Math.ceil(total / pageSize)

  const state: CollectionViewContextValue['state'] = {
    loading,
    documents,
    error,
    viewMode,
    tableColumns,
    currentPage,
    pageSize,
    total,
    totalPages,
    sortColumn,
    sortDirection,
    activeColumnMenu,
    activeFilter,
    availableFields,
    preferredFilterField,
    showExportModal,
    isFilterModalOpen,
    alert,
    columnWidths,
    resizingColumn,
    resizedColumns,
    ...changesetState,
  }

  const actions: CollectionViewContextValue['actions'] = {
    refresh: () => runWithDiscardGuard(refresh),
    handlePageChange,
    handlePageSizeChange,
    setViewMode,
    handleSort,
    clearSort,
    setActiveColumnMenu,
    setIsFilterModalOpen: setFilterModalOpen,
    openFilterForField,
    handleFilterApply,
    setShowExportModal,
    handleResizeStart,
    showAlert,
    closeAlert,
    confirmDiscardAndContinue,
    ...changesetActions,
  }

  return <CollectionViewCtx value={{ state, actions }}>{children}</CollectionViewCtx>
}
