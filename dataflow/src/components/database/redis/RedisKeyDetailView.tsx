import { useCallback, useEffect, useRef, useState, type KeyboardEvent as ReactKeyboardEvent, type ReactNode } from 'react'
import { useConnectionStore } from '@/stores/useConnectionStore'
import {
  useGetStorageUnitRowsLazyQuery,
  useAddRowMutation,
  useDeleteRowMutation,
  useUpdateStorageUnitMutation,
  SortDirection,
  type SortCondition,
  type RecordInput,
} from '@graphql'
import { transformRowsResult, type TableData } from '@/utils/graphql-transforms'
import { resolveSchemaParam } from '@/utils/database-features'
import { useI18n } from '@/i18n/useI18n'
import { DataView } from '@/components/database/shared/DataView'
import { FindBar, useFindBar } from '@/components/database/shared/FindBar'
import { Button } from '@/components/ui/Button'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { Separator } from '@/components/ui/separator'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ConfirmationModal } from '@/components/ui/ConfirmationModal'
import { cn } from '@/lib/utils'
import {
  Loader2, RefreshCw, Plus, Minus, MoreHorizontal,
  ArrowUpAZ, ArrowDownAZ, X, Download, TerminalSquare, BarChart3,
} from 'lucide-react'
import { ExportRedisKeyModal } from './ExportRedisKeyModal'
import { useTabStore } from '@/stores/useTabStore'
import { ChartCreateModal } from '@/components/analysis/chart-create'

// ---------------------------------------------------------------------------
// Types & helpers
// ---------------------------------------------------------------------------

type RedisKeyType = 'string' | 'hash' | 'list' | 'set' | 'zset'

function detectRedisKeyType(columns: string[], disableUpdate: boolean): RedisKeyType {
  if (columns.includes('field')) return 'hash'
  if (columns.includes('member')) return 'zset'
  if (columns.includes('index')) return disableUpdate ? 'set' : 'list'
  return 'string'
}

/** Whether a column in an existing row is editable inline. */
function isEditableColumn(column: string, keyType: RedisKeyType): boolean {
  if (column === 'index') return false
  if (column === 'field' && keyType === 'hash') return false
  if (column === 'member' && keyType === 'zset') return false
  return true
}

/** Whether a column should accept input when adding a new row. */
function isNewRowInputColumn(column: string): boolean {
  return column !== 'index'
}

function buildAddRowValues(row: Record<string, string>, keyType: RedisKeyType): RecordInput[] {
  switch (keyType) {
    case 'hash':
      return [{ Key: 'field', Value: row['field'] ?? '' }, { Key: 'value', Value: row['value'] ?? '' }]
    case 'list':
      return [{ Key: 'value', Value: row['value'] ?? '' }]
    case 'set':
      return [{ Key: 'value', Value: row['value'] ?? '' }]
    case 'zset':
      return [{ Key: 'member', Value: row['member'] ?? '' }, { Key: 'score', Value: row['score'] ?? '0' }]
    case 'string':
      return [{ Key: 'value', Value: row['value'] ?? '' }]
  }
}

function buildDeleteRowValues(row: Record<string, string>, keyType: RedisKeyType, keyName: string): RecordInput[] {
  switch (keyType) {
    case 'hash':
      return [{ Key: 'field', Value: row['field'] }]
    case 'list':
      return [{ Key: 'index', Value: row['index'] }]
    case 'set':
      return [{ Key: 'member', Value: row['value'] }]
    case 'zset':
      return [{ Key: 'member', Value: row['member'] }]
    case 'string':
      return [{ Key: 'key', Value: keyName }]
  }
}

/** Render-prop consumer for FindBar context — allows inline access to find state. */
function FindBarConsumer({ children }: { children: (state: ReturnType<typeof useFindBar>['state']) => ReactNode }) {
  const { state } = useFindBar()
  return <>{children(state)}</>
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface RedisKeyDetailViewProps {
  connectionId: string
  databaseName: string
  keyName: string
}

/** Displays and allows inline editing of a single Redis key's contents. */
export function RedisKeyDetailView({ connectionId, databaseName, keyName }: RedisKeyDetailViewProps) {
  const { connections, tableRefreshKey } = useConnectionStore()
  const { t } = useI18n()
  const openTab = useTabStore((s) => s.openTab)

  // ---- GraphQL hooks ----
  const [getRows] = useGetStorageUnitRowsLazyQuery({ fetchPolicy: 'no-cache' })
  const [addRowMutation] = useAddRowMutation()
  const [deleteRowMutation] = useDeleteRowMutation()
  const [updateMutation] = useUpdateStorageUnitMutation()

  // ---- Data state ----
  const [data, setData] = useState<TableData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)

  // ---- Derived ----
  const columns = data?.columns ?? []
  const rows = data?.rows ?? []
  const total = data?.total ?? 0
  const disableUpdate = data?.disableUpdate ?? false
  const keyType = columns.length > 0 ? detectRedisKeyType(columns, disableUpdate) : 'string'

  // ---- Pagination ----
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize] = useState(50)
  const totalPages = Math.ceil(total / pageSize)

  // ---- Sort ----
  const [sortColumn, setSortColumn] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc' | null>(null)
  const [activeColumnMenu, setActiveColumnMenu] = useState<string | null>(null)

  // ---- Column resize ----
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({})
  const [resizingColumn, setResizingColumn] = useState<string | null>(null)
  const [resizedColumns, setResizedColumns] = useState<Set<string>>(new Set())
  const resizingRef = useRef<{ column: string; startX: number; startWidth: number } | null>(null)

  // ---- Inline editing ----
  const [activeCell, setActiveCell] = useState<{ rowIdx: number; column: string } | null>(null)
  const [activeDraftValue, setActiveDraftValue] = useState('')

  // ---- Row selection ----
  const [selectedRows, setSelectedRows] = useState<Set<number>>(new Set())

  // ---- New row ----
  const [newRow, setNewRow] = useState<Record<string, string> | null>(null)

  // ---- Mutation loading ----
  const [mutating, setMutating] = useState(false)

  // ---- Delete confirmation ----
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [showExport, setShowExport] = useState(false)
  const [isChartModalOpen, setIsChartModalOpen] = useState(false)

  // ---- Ref for race-condition prevention ----
  const latestRequestIdRef = useRef(0)

  // =========================================================================
  // Data fetching
  // =========================================================================

  const fetchData = useCallback(async () => {
    const conn = connections.find((c) => c.id === connectionId)
    if (!conn) return

    setLoading(true)
    setError(null)

    latestRequestIdRef.current += 1
    const thisRequestId = latestRequestIdRef.current

    const schema = resolveSchemaParam(conn.type, databaseName)

    const sort: SortCondition[] | undefined =
      sortColumn && sortDirection
        ? [{ Column: sortColumn, Direction: sortDirection === 'asc' ? SortDirection.Asc : SortDirection.Desc }]
        : undefined

    try {
      const { data: result, error: gqlError } = await getRows({
        variables: {
          schema,
          storageUnit: keyName,
          sort,
          pageSize,
          pageOffset: (currentPage - 1) * pageSize,
        },
        context: { database: databaseName },
      })

      if (thisRequestId !== latestRequestIdRef.current) return

      if (gqlError) { setError(gqlError.message); return }

      if (result?.Row) {
        const tableData = transformRowsResult(result.Row)
        setData(tableData)

        // Initialize column widths on first load
        if (Object.keys(columnWidths).length === 0) {
          const widths: Record<string, number> = {}
          tableData.columns.forEach((col) => { widths[col] = Math.max(120, col.length * 10 + 60) })
          setColumnWidths(widths)
        }
      }
    } catch (err) {
      if (thisRequestId !== latestRequestIdRef.current) return
      const message = err instanceof Error ? err.message.trim() : ''
      setError(message || t('redis.detail.fetchFailed'))
    } finally {
      if (thisRequestId === latestRequestIdRef.current) setLoading(false)
    }
  }, [connections, connectionId, databaseName, keyName, sortColumn, sortDirection, pageSize, currentPage, getRows, t])

  useEffect(() => { fetchData() }, [fetchData, refreshKey, tableRefreshKey])

  const refresh = useCallback(() => { setSelectedRows(new Set()); setRefreshKey((k) => k + 1) }, [])

  // =========================================================================
  // Column resize (copied pattern from SQL TableView)
  // =========================================================================

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!resizingRef.current) return
      const { column, startX, startWidth } = resizingRef.current
      const newWidth = Math.max(60, startWidth + (e.clientX - startX))
      document.querySelectorAll<HTMLElement>(`[data-col="${column}"]`).forEach(el => {
        el.style.minWidth = `${newWidth}px`
        el.style.maxWidth = `${newWidth}px`
      })
    }
    const handleMouseUp = (e: MouseEvent) => {
      if (!resizingRef.current) return
      const { column, startX, startWidth } = resizingRef.current
      const finalWidth = Math.max(60, startWidth + (e.clientX - startX))
      setColumnWidths(prev => ({ ...prev, [column]: finalWidth }))
      setResizedColumns(prev => {
        if (prev.has(column)) return prev
        const next = new Set(prev)
        next.add(column)
        return next
      })
      resizingRef.current = null
      setResizingColumn(null)
      document.body.style.cursor = 'default'
      document.querySelectorAll<HTMLElement>('[data-resize-active]').forEach((el) => { delete el.dataset.resizeActive })
    }
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    return () => { document.removeEventListener('mousemove', handleMouseMove); document.removeEventListener('mouseup', handleMouseUp) }
  }, [])

  const handleResizeStart = useCallback((e: React.MouseEvent, column: string) => {
    e.preventDefault()
    e.stopPropagation()
    resizingRef.current = { column, startX: e.clientX, startWidth: columnWidths[column] || 120 }
    setResizingColumn(column)
    document.body.style.cursor = 'col-resize'
  }, [columnWidths])

  // =========================================================================
  // Sort handlers
  // =========================================================================

  const handleSort = useCallback((col: string, dir: 'asc' | 'desc') => {
    setSortColumn(col)
    setSortDirection(dir)
    setCurrentPage(1)
    setActiveColumnMenu(null)
  }, [])

  const clearSort = useCallback(() => {
    setSortColumn(null)
    setSortDirection(null)
    setCurrentPage(1)
    setActiveColumnMenu(null)
  }, [])

  // =========================================================================
  // Pagination handlers
  // =========================================================================

  const handlePageChange = useCallback((page: number) => { setCurrentPage(page) }, [])
  const handlePageSizeChange = useCallback((size: number) => { setPageSize(size); setCurrentPage(1) }, [])

  // =========================================================================
  // Inline editing
  // =========================================================================

  const activateCell = useCallback((rowIdx: number, column: string) => {
    setActiveCell({ rowIdx, column })
    setActiveDraftValue(rows[rowIdx]?.[column] ?? '')
  }, [rows])

  const deactivateCell = useCallback(() => { setActiveCell(null) }, [])

  const commitCellEdit = useCallback(async () => {
    if (!activeCell) return
    const { rowIdx, column } = activeCell
    const row = rows[rowIdx]
    if (!row || activeDraftValue === row[column]) { deactivateCell(); return }

    const conn = connections.find((c) => c.id === connectionId)
    if (!conn) return

    const schema = resolveSchemaParam(conn.type, databaseName)
    setMutating(true)

    try {
      if (keyType === 'zset' && column === 'score') {
        // ZADD updates score for existing member
        await addRowMutation({
          variables: {
            schema, storageUnit: keyName,
            values: [{ Key: 'member', Value: row['member'] }, { Key: 'score', Value: activeDraftValue }],
          },
          context: { database: databaseName },
        })
      } else {
        const values: RecordInput[] = columns.map((col) => ({
          Key: col, Value: col === column ? activeDraftValue : row[col],
        }))
        await updateMutation({
          variables: { schema, storageUnit: keyName, values, updatedColumns: [column] },
          context: { database: databaseName },
        })
      }
      deactivateCell()
      refresh()
    } catch {
      setError(t('redis.detail.editFailed'))
    } finally {
      setMutating(false)
    }
  }, [activeCell, activeDraftValue, rows, columns, keyType, connections, connectionId, databaseName, keyName, addRowMutation, updateMutation, deactivateCell, refresh, t])

  const handleCellKeyDown = useCallback((e: ReactKeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Escape') { e.preventDefault(); deactivateCell(); return }
    if (e.key === 'Enter' && !e.nativeEvent.isComposing && e.keyCode !== 229) { e.preventDefault(); commitCellEdit(); return }
    if (e.key === 'Tab') {
      e.preventDefault()
      // Commit current, then move to next editable column
      commitCellEdit()
    }
  }, [deactivateCell, commitCellEdit])

  // =========================================================================
  // Add row
  // =========================================================================

  const startNewRow = useCallback(() => {
    const empty: Record<string, string> = {}
    columns.forEach((col) => { empty[col] = '' })
    setNewRow(empty)
  }, [columns])

  const cancelNewRow = useCallback(() => setNewRow(null), [])

  const confirmNewRow = useCallback(async () => {
    if (!newRow) return
    const conn = connections.find((c) => c.id === connectionId)
    if (!conn) return

    const schema = resolveSchemaParam(conn.type, databaseName)
    setMutating(true)

    try {
      await addRowMutation({
        variables: { schema, storageUnit: keyName, values: buildAddRowValues(newRow, keyType) },
        context: { database: databaseName },
      })
      setNewRow(null)
      refresh()
    } catch {
      setError(t('redis.detail.addFailed'))
    } finally {
      setMutating(false)
    }
  }, [newRow, keyType, connections, connectionId, databaseName, keyName, addRowMutation, refresh, t])

  const handleNewRowKeyDown = useCallback((e: ReactKeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Escape') { e.preventDefault(); cancelNewRow(); return }
    if (e.key === 'Enter' && !e.nativeEvent.isComposing && e.keyCode !== 229) { e.preventDefault(); confirmNewRow() }
  }, [cancelNewRow, confirmNewRow])

  // =========================================================================
  // Row selection & batch delete
  // =========================================================================

  const toggleRowSelection = useCallback((rowIdx: number) => {
    setSelectedRows((prev) => {
      const next = new Set(prev)
      if (next.has(rowIdx)) next.delete(rowIdx)
      else next.add(rowIdx)
      return next
    })
  }, [])

  const handleDeleteSelected = useCallback(async () => {
    if (selectedRows.size === 0) return
    const conn = connections.find((c) => c.id === connectionId)
    if (!conn) return

    const schema = resolveSchemaParam(conn.type, databaseName)
    setMutating(true)

    // For lists, delete from highest index to lowest to avoid index shifting
    const indices = [...selectedRows].sort((a, b) => b - a)

    try {
      for (const rowIdx of indices) {
        const row = rows[rowIdx]
        if (!row) continue
        await deleteRowMutation({
          variables: { schema, storageUnit: keyName, values: buildDeleteRowValues(row, keyType, keyName) },
          context: { database: databaseName },
        })
      }
      refresh()
    } catch {
      setError(t('redis.detail.deleteFailed'))
    } finally {
      setMutating(false)
    }
  }, [selectedRows, rows, keyType, connections, connectionId, databaseName, keyName, deleteRowMutation, refresh, t])

  // =========================================================================
  // Scroll shadow tracking
  // =========================================================================

  const scrollRef = useRef<HTMLDivElement>(null)
  const [isScrolledX, setIsScrolledX] = useState(false)
  const [isScrolledY, setIsScrolledY] = useState(false)
  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (el) { setIsScrolledX(el.scrollLeft > 0); setIsScrolledY(el.scrollTop > 0) }
  }, [])

  // =========================================================================
  // Render
  // =========================================================================

  if (loading && !data) {
    return (
      <div
        className="flex-1 flex items-center justify-center"
        data-testid="redis.key.detail-loading"
        data-qa-module="redis"
        data-qa-object="key-detail"
        data-qa-state="loading"
        data-qa-loading="true"
        data-qa-connection-id={connectionId}
        data-qa-database={databaseName}
        data-qa-resource-type="redis_key"
        data-qa-resource-id={keyName}
      >
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const canEdit = !disableUpdate
  const canAdd = keyType !== 'string'

  return (
    <FindBar.Provider rows={rows} columns={columns}>
    <div
      className="flex flex-col h-full bg-background"
      data-testid="redis.key.detail"
      data-qa-module="redis"
      data-qa-object="key-detail"
      data-qa-state={error ? 'error' : loading ? 'loading' : mutating ? 'mutating' : 'ready'}
      data-qa-loading={loading ? 'true' : 'false'}
      data-qa-connection-id={connectionId}
      data-qa-database={databaseName}
      data-qa-resource-type="redis_key"
      data-qa-resource-id={keyName}
      data-qa-key-type={keyType}
    >
      {/* ---- Toolbar ---- */}
      <div
        className="flex items-center justify-between h-12 px-2"
        data-testid="redis.key.toolbar"
        data-qa-module="redis"
        data-qa-object="key-toolbar"
        data-qa-state={loading ? 'loading' : mutating ? 'mutating' : 'ready'}
      >
        <div className="flex items-center">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                onClick={refresh}
                disabled={loading}
                data-testid="redis.key.refresh-button"
                data-qa-module="redis"
                data-qa-object="key-data"
                data-qa-action="refresh"
                data-qa-state={loading ? 'loading' : 'ready'}
                data-qa-disabled-reason={loading ? 'loading' : undefined}
              >
                <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t('common.actions.refresh')}</TooltipContent>
          </Tooltip>

          <Separator orientation="vertical" className="mx-1 data-[orientation=vertical]:h-4" />

          {canAdd && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={startNewRow}
                  disabled={newRow !== null || mutating}
                  data-testid="redis.key.add-row-button"
                  data-qa-module="redis"
                  data-qa-object="key-row"
                  data-qa-action="create"
                  data-qa-risk="resource_mutation"
                  data-qa-disabled-reason={newRow !== null ? 'pending_row_active' : mutating ? 'mutating' : undefined}
                >
                  <Plus className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t('redis.detail.addRow')}</TooltipContent>
            </Tooltip>
          )}

          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => setShowDeleteConfirm(true)}
                  disabled={selectedRows.size === 0 || mutating}
                  data-testid="redis.key.delete-selected-button"
                  data-qa-module="redis"
                  data-qa-object="key-row"
                  data-qa-action="delete"
                  data-qa-risk="resource_mutation"
                  data-qa-disabled-reason={selectedRows.size === 0 ? 'no_selection' : mutating ? 'mutating' : undefined}
                >
                  <Minus className="h-4 w-4" />
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>{t('redis.detail.deleteSelected')}</TooltipContent>
          </Tooltip>

          <Separator orientation="vertical" className="mx-1 data-[orientation=vertical]:h-4" />

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setIsChartModalOpen(true)}
                data-testid="redis.key.create-chart-button"
                data-qa-module="redis"
                data-qa-object="key-data"
                data-qa-action="create-chart"
              >
                <BarChart3 className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t('analysis.chart.create')}</TooltipContent>
          </Tooltip>
        </div>

        <div className="flex items-center gap-2">
          <Button
            className="rounded-lg gap-2.5 min-w-[86px]"
            onClick={() => setShowExport(true)}
            data-testid="redis.key.export-button"
            data-qa-module="redis"
            data-qa-object="key-data"
            data-qa-action="export"
          >
            <Download className="h-4 w-4" />
            {t('common.actions.export')}
          </Button>
          <Button className="rounded-lg gap-2.5 min-w-[86px]" onClick={() => openTab({
            type: 'query',
            title: t('sidebar.tab.queryWithDatabase', { database: databaseName }),
            connectionId,
            databaseName,
          })}
            data-testid="redis.key.open-query-button"
            data-qa-module="redis"
            data-qa-object="key-data"
            data-qa-action="open-query"
          >
            <TerminalSquare className="h-4 w-4" />
            {t('common.actions.query')}
          </Button>
        </div>
      </div>

      <FindBar.Bar />

      {/* ---- Error banner ---- */}
      {error && (
        <div
          className="px-4 py-2 text-sm text-destructive bg-destructive/10 border-b border-destructive/20 flex items-center justify-between"
          data-testid="redis.key.error"
          data-qa-module="redis"
          data-qa-object="key-data"
          data-qa-state="error"
          data-qa-error-code="redis_key_operation_failed"
        >
          <span>{error}</span>
          <Button variant="ghost" size="icon-xs" onClick={() => setError(null)}><X className="h-3 w-3" /></Button>
        </div>
      )}

      {/* ---- Data grid ---- */}
      <div
        ref={scrollRef}
        onScroll={handleScroll}
        data-scrolled-x={isScrolledX || undefined}
        data-scrolled-y={isScrolledY || undefined}
        className="flex-1 overflow-auto"
        data-testid="redis.key.grid-scroll"
        data-qa-module="redis"
        data-qa-object="key-grid"
        data-qa-state={rows.length > 0 ? 'ready' : 'empty'}
        data-qa-row-count={rows.length}
      >
        <table
          className="min-w-full border-collapse text-sm"
          data-testid="redis.key.grid"
          data-qa-module="redis"
          data-qa-object="key-grid"
          data-qa-state={rows.length > 0 ? 'ready' : 'empty'}
        >
          <thead className="border-b border-border bg-background">
            <tr>
              {/* Row number column */}
              <th
                className="sticky top-0 left-0 z-50 border-b border-r border-border/50 bg-background px-2 py-2 text-center text-xs font-semibold text-muted-foreground"
                style={{ width: 64, minWidth: 64, maxWidth: 64 }}
              > </th>

              {/* Data columns */}
              {columns.map((col, colIdx) => {
                const width = columnWidths[col] || 120
                return (
                  <th
                    key={col}
                    data-col={col}
                    style={{ minWidth: `${width}px`, ...(resizedColumns.has(col) && { maxWidth: `${width}px` }) }}
                    className="px-6 py-2 text-left font-medium text-sm text-muted-foreground whitespace-nowrap group/header relative overflow-hidden border-r border-border/50 select-none sticky top-0 bg-background z-40"
                  >
                    <div className="flex items-center justify-between h-full">
                      <div className="flex items-center gap-1 overflow-hidden mr-6">
                        <span className="truncate" title={col}>{col}</span>
                        {sortColumn === col && (
                          <span className="text-primary shrink-0">
                            {sortDirection === 'asc' ? <ArrowUpAZ className="h-3 w-3" /> : <ArrowDownAZ className="h-3 w-3" />}
                          </span>
                        )}
                      </div>

                      {/* Sort menu */}
                      <DropdownMenu
                        open={activeColumnMenu === col}
                        onOpenChange={(open) => setActiveColumnMenu(open ? col : null)}
                      >
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon-xs"
                            className={cn(
                              'absolute top-2 right-2 text-muted-foreground',
                              activeColumnMenu === col && 'bg-muted text-foreground',
                            )}
                            onClick={(e) => e.stopPropagation()}
                          >
                            <MoreHorizontal className="h-3.5 w-3.5" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align={colIdx === 0 ? 'start' : 'end'} className="w-40">
                          <DropdownMenuLabel className="text-[10px] text-muted-foreground">
                            {t('redis.detail.sortActions')}
                          </DropdownMenuLabel>
                          <DropdownMenuItem
                            onSelect={() => handleSort(col, 'asc')}
                            className={cn(sortColumn === col && sortDirection === 'asc' && 'bg-primary/5 font-medium text-primary')}
                          >
                            <ArrowUpAZ className="h-3.5 w-3.5" />
                            {t('redis.detail.sortAsc')}
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onSelect={() => handleSort(col, 'desc')}
                            className={cn(sortColumn === col && sortDirection === 'desc' && 'bg-primary/5 font-medium text-primary')}
                          >
                            <ArrowDownAZ className="h-3.5 w-3.5" />
                            {t('redis.detail.sortDesc')}
                          </DropdownMenuItem>
                          {sortColumn === col && (
                            <>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem onSelect={clearSort}>
                                <X className="h-3.5 w-3.5" />
                                {t('redis.detail.clearSort')}
                              </DropdownMenuItem>
                            </>
                          )}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>

                    {/* Resize handle */}
                    <div
                      data-resize-col={col}
                      className={cn(
                        'absolute right-0 top-0 -bottom-px w-1 cursor-col-resize z-20 data-[resize-active]:bg-primary/50',
                        resizingColumn === col && 'bg-primary/50',
                      )}
                      onMouseEnter={() => { if (!resizingColumn) document.querySelectorAll<HTMLElement>(`[data-resize-col="${col}"]`).forEach((el) => { el.dataset.resizeActive = '' }) }}
                      onMouseLeave={() => { if (!resizingColumn) document.querySelectorAll<HTMLElement>(`[data-resize-col="${col}"]`).forEach((el) => { delete el.dataset.resizeActive }) }}
                      onMouseDown={(e) => handleResizeStart(e, col)}
                    />
                  </th>
                )
              })}

              <th className="sticky top-0 z-40 border-b border-border/50 bg-background w-full" />
            </tr>
          </thead>
          <FindBarConsumer>{(findState) => (
          <tbody className="bg-background">
            {rows.map((row, rowIdx) => {
              const isSelected = selectedRows.has(rowIdx)
              return (
                <tr
                  key={rowIdx}
                  className="group transition-colors hover:bg-muted/50"
                  data-testid="redis.key.row"
                  data-qa-module="redis"
                  data-qa-object="key-row"
                  data-qa-state={isSelected ? 'selected' : 'ready'}
                  data-qa-resource-type="redis_key_row"
                  data-qa-resource-id={`${keyName}:${rowIdx}`}
                >
                  {/* Row number — click to toggle selection */}
                  <td
                    className={cn(
                      'sticky left-0 z-30 border-b border-r border-border/50 bg-background px-2 py-2 text-center text-xs font-medium',
                      isSelected && 'bg-primary/10',
                    )}
                    style={{ width: 64, minWidth: 64, maxWidth: 64 }}
                    onClick={() => toggleRowSelection(rowIdx)}
                    data-testid="redis.key.row-selector"
                    data-qa-module="redis"
                    data-qa-object="key-row"
                    data-qa-action="select"
                    data-qa-state={isSelected ? 'selected' : 'ready'}
                    data-qa-resource-type="redis_key_row"
                    data-qa-resource-id={`${keyName}:${rowIdx}`}
                  >
                    {(currentPage - 1) * pageSize + rowIdx + 1}
                  </td>

                  {/* Data cells */}
                  {columns.map((col) => {
                    const width = columnWidths[col] || 120
                    const isActive = activeCell?.rowIdx === rowIdx && activeCell.column === col
                    const editable = canEdit && isEditableColumn(col, keyType)
                    const highlight = findState.total
                      ? findState.matches.findIndex((m) => m.rowIndex === rowIdx && m.columnKey === col) === findState.currentMatchIndex
                        ? 'current'
                        : findState.matches.some((m) => m.rowIndex === rowIdx && m.columnKey === col)
                          ? 'match'
                          : null
                      : null

                    return (
                      <td
                        key={col}
                        data-col={col}
                        data-testid="redis.key.cell"
                        data-qa-module="redis"
                        data-qa-object="key-cell"
                        data-qa-field={col}
                        data-qa-state={isActive ? 'editing' : editable ? 'editable' : 'read_only'}
                        data-qa-disabled-reason={!editable ? 'read_only' : undefined}
                        data-qa-resource-type="redis_key_row"
                        data-qa-resource-id={`${keyName}:${rowIdx}`}
                        data-find-current={highlight === 'current' ? 'true' : undefined}
                        className={cn(
                          'relative overflow-hidden border-b border-r border-border/50 text-sm text-foreground/80 scroll-mt-14',
                          isActive ? 'p-0' : 'px-6 py-2',
                          isSelected && 'bg-primary/10',
                          highlight === 'current' && 'bg-blue-200',
                          highlight === 'match' && 'bg-blue-100/60',
                          editable && !isActive && 'cursor-default',
                        )}
                        style={{ minWidth: `${width}px`, ...(resizedColumns.has(col) && { maxWidth: `${width}px` }) }}
                        onDoubleClick={() => { if (editable && !mutating) activateCell(rowIdx, col) }}
                      >
                        {isActive ? (
                          <input
                            autoFocus
                            type="text"
                            data-redis-editor="true"
                            data-testid="redis.key.cell-editor"
                            data-qa-module="redis"
                            data-qa-object="key-cell"
                            data-qa-action="edit"
                            data-qa-field={col}
                            data-qa-state="editing"
                            data-qa-resource-type="redis_key_row"
                            data-qa-resource-id={`${keyName}:${rowIdx}`}
                            value={activeDraftValue}
                            onChange={(e) => setActiveDraftValue(e.target.value)}
                            onBlur={() => {
                              queueMicrotask(() => {
                                const el = document.activeElement
                                if (el instanceof HTMLInputElement && el.dataset.redisEditor === 'true') return
                                commitCellEdit()
                              })
                            }}
                            onKeyDown={handleCellKeyDown}
                            className="w-full min-h-[36px] bg-transparent px-6 py-2 text-sm focus:bg-background focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary"
                          />
                        ) : (
                          <span className="block truncate" title={row[col] ?? ''}>
                            {row[col] ?? ''}
                          </span>
                        )}

                        {/* Resize guide on cells */}
                        <div
                          data-resize-col={col}
                          className={cn(
                            'absolute right-0 top-0 -bottom-px w-1 cursor-col-resize z-20 data-[resize-active]:bg-primary/50',
                            resizingColumn === col && 'bg-primary/50',
                          )}
                          onMouseEnter={() => { if (!resizingColumn) document.querySelectorAll<HTMLElement>(`[data-resize-col="${col}"]`).forEach((el) => { el.dataset.resizeActive = '' }) }}
                          onMouseLeave={() => { if (!resizingColumn) document.querySelectorAll<HTMLElement>(`[data-resize-col="${col}"]`).forEach((el) => { delete el.dataset.resizeActive }) }}
                          onMouseDown={(e) => handleResizeStart(e, col)}
                        />
                      </td>
                    )
                  })}

                  <td className="border-b border-border/50 bg-background w-full" />
                </tr>
              )
            })}

            {/* ---- New row (inline add) ---- */}
            {newRow && (
              <tr
                className="bg-blue-50/30 dark:bg-blue-950/20"
                data-testid="redis.key.new-row"
                data-qa-module="redis"
                data-qa-object="key-row"
                data-qa-state="creating"
                data-qa-risk="resource_mutation"
              >
                <td
                  className="sticky left-0 z-30 border-b border-r border-border/50 bg-blue-50/60 dark:bg-blue-950/40 px-2 py-2 text-center text-xs font-medium text-primary"
                  style={{ width: 64, minWidth: 64, maxWidth: 64 }}
                >
                  +
                </td>
                {columns.map((col) => {
                  const width = columnWidths[col] || 120
                  const isInput = isNewRowInputColumn(col)

                  return (
                    <td
                      key={col}
                      data-col={col}
                      className="border-b border-r border-border/50 p-0"
                      style={{ minWidth: `${width}px`, ...(resizedColumns.has(col) && { maxWidth: `${width}px` }) }}
                    >
                      {isInput ? (
                        <input
                          autoFocus={col === columns.find(isNewRowInputColumn)}
                          type={col === 'score' ? 'number' : 'text'}
                          placeholder={t('redis.detail.newRowPlaceholder')}
                          data-testid="redis.key.new-row-input"
                          data-qa-module="redis"
                          data-qa-object="key-row"
                          data-qa-field={col}
                          data-qa-state={mutating ? 'disabled' : 'ready'}
                          data-qa-disabled-reason={mutating ? 'mutating' : undefined}
                          value={newRow[col] ?? ''}
                          onChange={(e) => setNewRow((prev) => prev ? { ...prev, [col]: e.target.value } : null)}
                          onKeyDown={handleNewRowKeyDown}
                          disabled={mutating}
                          className="w-full min-h-[36px] bg-transparent px-6 py-2 text-sm font-mono focus:bg-background focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary disabled:opacity-50"
                        />
                      ) : (
                        <span className="block px-6 py-2 text-sm italic text-muted-foreground">
                          {t('redis.detail.autoIndex')}
                        </span>
                      )}
                    </td>
                  )
                })}
                <td className="border-b border-border/50 w-full" />
              </tr>
            )}
          </tbody>
          )}</FindBarConsumer>
        </table>

        {rows.length === 0 && !newRow && (
          <div
            className="flex items-center justify-center py-12 text-muted-foreground text-sm"
            data-testid="redis.key.empty"
            data-qa-module="redis"
            data-qa-object="key-grid"
            data-qa-state="empty"
          >
            {t('redis.detail.empty')}
          </div>
        )}
      </div>

      {/* ---- Pagination ---- */}
      {total > 0 && (
        <DataView.Pagination
          currentPage={currentPage}
          totalPages={totalPages}
          pageSize={pageSize}
          total={total}
          loading={loading}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
        />
      )}

      {/* ---- Delete confirmation modal ---- */}
      <ConfirmationModal
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={handleDeleteSelected}
        title={t('redis.detail.confirmDeleteTitle')}
        message={t('redis.detail.confirmDeleteMessage', { count: String(selectedRows.size) })}
        isDestructive
      />

      <ExportRedisKeyModal
        open={showExport}
        onOpenChange={setShowExport}
        connectionId={connectionId}
        databaseName={databaseName}
        keyName={keyName}
      />

      <ChartCreateModal
        open={isChartModalOpen}
        onOpenChange={setIsChartModalOpen}
        initialData={data ? {
          connectionId,
          databaseName,
          query: keyType === 'hash' ? `HGETALL ${keyName}`
            : keyType === 'list' ? `LRANGE ${keyName} 0 -1`
            : keyType === 'set' ? `SMEMBERS ${keyName}`
            : keyType === 'zset' ? `ZRANGE ${keyName} 0 -1 WITHSCORES`
            : `GET ${keyName}`,
          columns,
          rows,
        } : undefined}
      />
    </div>
    </FindBar.Provider>
  )
}
