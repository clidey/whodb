import { createContext, use, useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { useChangesetManager } from './useChangesetManager'
import { useDataQuery } from './useDataQuery'
import { useColumnResize } from './useColumnResize'
import type { TableViewContextValue, TableViewState, TableViewActions, FilterCondition } from './types'
import type { Alert } from '@/components/database/shared/types'

const TableViewCtx = createContext<TableViewContextValue | null>(null)

/** Hook to access TableView context. Throws if used outside TableViewProvider. */
export function useTableView(): TableViewContextValue {
  const ctx = use(TableViewCtx)
  if (!ctx) throw new Error('useTableView must be used within TableViewProvider')
  return ctx
}

/** Simplify verbose PostgreSQL column type names for display. */
export function simplifyColumnType(typeStr: string): string {
  if (!typeStr) return ''
  return typeStr
    .replace(/ varying/gi, '')
    .replace(/ without time zone/gi, '')
    .replace(/ with time zone/gi, ' tz')
    .replace(/character/gi, 'char')
    .replace(/double precision/gi, 'double')
    .trim()
}

interface TableViewProviderProps {
  connectionId: string
  databaseName: string
  tableName: string
  schema?: string
  children: ReactNode
}

/** Provider that owns all TableDetailView state, GraphQL operations, and handlers. */
export function TableViewProvider({ connectionId, databaseName, tableName, schema, children }: TableViewProviderProps) {
  // ---- UI state ----
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize] = useState(50)

  // ---- Sorting state ----
  const [sortColumn, setSortColumn] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc' | null>(null)
  const [activeColumnMenu, setActiveColumnMenu] = useState<string | null>(null)

  // ---- Filter state ----
  const [isFilterModalOpen, setIsFilterModalOpen] = useState(false)
  const [visibleColumns, setVisibleColumns] = useState<string[]>([])
  const [filterConditions, setFilterConditions] = useState<FilterCondition[]>([])

  // ---- Modal state ----
  const [showExportModal, setShowExportModal] = useState(false)

  // ---- Alert state ----
  const [alert, setAlert] = useState<Alert | null>(null)

  // ---- Refs ----
  const lastTableRef = useRef<string>('')

  // ---- Callback for initial visible columns population ----
  const onInitVisibleColumns = useCallback((columns: string[]) => {
    setVisibleColumns(columns)
  }, [])

  // ---- Data query (GraphQL fetch, loading/error, race condition prevention) ----
  const { state: queryState, actions: queryActions } = useDataQuery({
    connectionId,
    databaseName,
    schema,
    tableName,
    currentPage,
    pageSize,
    sortColumn,
    sortDirection,
    filterConditions,
    visibleColumnsCount: visibleColumns.length,
    onInitVisibleColumns,
  })

  // ---- Column resizing ----
  const { columnWidths, resizingColumn, resizedColumns, handleResizeStart } = useColumnResize(queryState.data?.columns)

  // ---- Alert helpers ----
  const showAlert = useCallback((title: string, message: string, type: Alert['type'] = 'info') => {
    setAlert({ title, message, type })
  }, [])

  const closeAlert = useCallback(() => setAlert(null), [])

  const pageOffset = (currentPage - 1) * pageSize

  // ---- Changeset editing ----
  const { state: changesetState, actions: changesetActions } = useChangesetManager({
    connectionId,
    databaseName,
    schema,
    tableName,
    data: queryState.data,
    pageOffset,
    visibleColumns,
    primaryKey: queryState.primaryKey,
    refresh: queryActions.refresh,
    showAlert,
  })

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

  // ---- Table switch: reset state ----
  useEffect(() => {
    const currentTableKey = `${connectionId}:${databaseName}:${schema || ''}:${tableName}`
    if (lastTableRef.current !== currentTableKey) {
      lastTableRef.current = currentTableKey
      setVisibleColumns([])
      setFilterConditions([])
      setSortColumn(null)
      setSortDirection(null)
      setCurrentPage(1)
      changesetActions.discardChanges()
    }
  }, [changesetActions, connectionId, databaseName, schema, tableName])

  useEffect(() => {
    if (!changesetState.hasPendingChanges) return

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [changesetState.hasPendingChanges])

  // ---- Sorting ----
  const handleSort = useCallback((column: string, direction: 'asc' | 'desc') => {
    runWithDiscardGuard(() => {
      setSortColumn(column)
      setSortDirection(direction)
      setActiveColumnMenu(null)
    })
  }, [runWithDiscardGuard])

  const clearSort = useCallback(() => {
    runWithDiscardGuard(() => {
      setSortColumn(null)
      setSortDirection(null)
      setActiveColumnMenu(null)
    })
  }, [runWithDiscardGuard])

  // ---- Page change ----
  const handlePageChange = useCallback((page: number) => {
    runWithDiscardGuard(() => {
      setCurrentPage(page)
    })
  }, [runWithDiscardGuard])

  // ---- Page size change ----
  const handlePageSizeChange = useCallback((size: number) => {
    runWithDiscardGuard(() => {
      setPageSize(size)
      setCurrentPage(1)
    })
  }, [runWithDiscardGuard])

  // ---- Filter apply ----
  const handleFilterApply = useCallback((cols: string[], conditions: FilterCondition[]) => {
    runWithDiscardGuard(() => {
      setVisibleColumns(cols)
      setFilterConditions(conditions)
      setCurrentPage(1)
      queryActions.refresh()
    })
  }, [queryActions.refresh, runWithDiscardGuard])

  const state: TableViewState = {
    ...queryState,
    currentPage,
    pageSize,
    visibleColumns,
    filterConditions,
    sortColumn,
    sortDirection,
    activeColumnMenu,
    ...changesetState,
    columnWidths,
    resizingColumn,
    resizedColumns,
    showExportModal,
    isFilterModalOpen,
    alert,
  }

  const actions: TableViewActions = {
    refresh: () => runWithDiscardGuard(queryActions.refresh),
    handleSubmitRequest: queryActions.handleSubmitRequest,
    handlePageChange,
    handlePageSizeChange,
    handleSort,
    clearSort,
    setActiveColumnMenu,
    ...changesetActions,
    handleResizeStart,
    setIsFilterModalOpen,
    handleFilterApply,
    setShowExportModal,
    confirmDiscardAndContinue,
    showAlert,
    closeAlert,
  }

  return <TableViewCtx value={{ state, actions }}>{children}</TableViewCtx>
}
