import { createContext, use, useCallback, useState, type ReactNode } from 'react'
import { FileJson, FileSpreadsheet, FileCode, FileText, Table2 } from 'lucide-react'
import {
  SqlDataExportMode,
  useCreateSqlDataExportMutation,
  useGetStorageUnitRowsLazyQuery,
  type SortCondition,
  type WhereCondition,
} from '@/generated/graphql'
import { toCSV, toJSON, toExcel, downloadBlob } from '@/utils/export-utils'
import { fetchExportDownloadBlob } from '@/utils/export-download'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Input } from '@/components/ui/Input'
import { ModalForm, useModalForm } from '@/components/ui/ModalForm'
import { FormatSelector, type FormatOption } from '@/components/database/shared/FormatSelector'
import { ExportProgress, ExportFooter } from '@/components/database/shared/ExportProgress'
import { useI18n } from '@/i18n/useI18n'
import { useConnectionStore } from '@/stores/useConnectionStore'
import { resolveSchemaParam } from '@/utils/database-features'
import { cn } from '@/lib/utils'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

type ExportFormat = 'csv' | 'json' | 'sql' | 'excel'
type StorageUnitType = 'table' | 'view'

const EXPORT_PAGE_SIZE = 500

const FORMAT_OPTIONS: FormatOption<ExportFormat>[] = [
  { id: 'csv', label: 'CSV', icon: FileText },
  { id: 'json', label: 'JSON', icon: FileJson },
  { id: 'sql', label: 'SQL', icon: FileCode },
  { id: 'excel', label: 'Excel', icon: FileSpreadsheet },
]

const FORMAT_EXTENSIONS: Record<ExportFormat, string> = {
  csv: 'csv',
  json: 'json',
  sql: 'sql',
  excel: 'xlsx',
}

type GetRowsForExport = ReturnType<typeof useGetStorageUnitRowsLazyQuery>[0]

interface FetchRowsForExportParams {
  getRows: GetRowsForExport
  databaseName: string
  schema: string
  tableName: string
  where?: WhereCondition
  sort?: SortCondition[]
  limit?: number
  noDataMessage: string
}

async function fetchRowsForExport({
  getRows,
  databaseName,
  schema,
  tableName,
  where,
  sort,
  limit,
  noDataMessage,
}: FetchRowsForExportParams): Promise<{ Columns: Array<{ Name: string }>, Rows: string[][] }> {
  const rows: string[][] = []
  let columns: Array<{ Name: string }> | null = null
  let pageOffset = 0
  let totalCount: number | null = null

  do {
    const remaining = limit === undefined ? EXPORT_PAGE_SIZE : Math.max(limit - rows.length, 0)
    const pageSize = limit === 0 ? 1 : Math.min(EXPORT_PAGE_SIZE, remaining)
    const { data, error } = await getRows({
      variables: {
        schema,
        storageUnit: tableName,
        where,
        sort,
        pageSize,
        pageOffset,
      },
      context: { database: databaseName },
    })

    if (error) throw new Error(error.message)
    if (!data?.Row) throw new Error(noDataMessage)

    columns ??= data.Row.Columns
    totalCount = data.Row.TotalCount
    if (limit === 0) break

    const pageRows = data.Row.Rows
    rows.push(...(limit === undefined ? pageRows : pageRows.slice(0, remaining)))

    if (pageRows.length === 0 || rows.length >= totalCount) break
    pageOffset += pageRows.length
  } while (limit === undefined || rows.length < limit)

  if (!columns) throw new Error(noDataMessage)
  return { Columns: columns, Rows: rows }
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

interface ExportDataCtxValue {
  format: ExportFormat
  setFormat: (v: ExportFormat) => void
  formatOptions: FormatOption<ExportFormat>[]
  rowCount: number | ''
  setRowCount: (v: number | '') => void
  sqlMode: SqlDataExportMode
  setSqlMode: (v: SqlDataExportMode) => void
  sqlExportUnavailableReason: string | null
  sqlUpdateUnavailableReason: string | null
  isSuccess: boolean
  handleExport: () => void
}

const ExportDataCtx = createContext<ExportDataCtxValue | null>(null)

/** Hook to access ExportDataModal domain state. Throws if used outside the provider. */
function useExportDataCtx(): ExportDataCtxValue {
  const ctx = use(ExportDataCtx)
  if (!ctx) throw new Error('useExportDataCtx must be used within ExportDataProvider')
  return ctx
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

/** Wraps ModalForm.Provider (complex mode, no onSubmit) and domain context for SQL table export. */
function ExportDataProvider({
  connectionId,
  databaseName,
  schema,
  tableName,
  storageUnitType,
  primaryKeyColumns,
  where,
  sort,
  children,
}: {
  connectionId: string
  databaseName: string
  schema?: string | null
  tableName: string
  storageUnitType?: StorageUnitType
  primaryKeyColumns: string[]
  where?: WhereCondition
  sort?: SortCondition[]
  children: ReactNode
}) {
  const { t } = useI18n()

  return (
    <ModalForm.Provider
      meta={{
        title: t('sql.export.title'),
        description: schema ? `${databaseName}.${schema}.${tableName}` : `${databaseName}.${tableName}`,
        icon: Table2,
      }}
    >
      <ExportDataBridge
        connectionId={connectionId}
        databaseName={databaseName}
        schema={schema}
        tableName={tableName}
        storageUnitType={storageUnitType}
        primaryKeyColumns={primaryKeyColumns}
        where={where}
        sort={sort}
      >
        {children}
      </ExportDataBridge>
    </ModalForm.Provider>
  )
}

/** Inner bridge that owns domain state and export logic, accessing ModalForm actions via useModalForm(). */
function ExportDataBridge({
  connectionId,
  databaseName,
  schema,
  tableName,
  storageUnitType,
  primaryKeyColumns,
  where,
  sort,
  children,
}: {
  connectionId: string
  databaseName: string
  schema?: string | null
  tableName: string
  storageUnitType?: StorageUnitType
  primaryKeyColumns: string[]
  where?: WhereCondition
  sort?: SortCondition[]
  children: ReactNode
}) {
  const { t } = useI18n()
  const [format, setFormat] = useState<ExportFormat>('csv')
  const [rowCount, setRowCount] = useState<number | ''>(1000)
  const [sqlMode, setSqlMode] = useState<SqlDataExportMode>(SqlDataExportMode.Insert)
  const [isSuccess, setIsSuccess] = useState(false)
  const { actions } = useModalForm()
  const [getRows] = useGetStorageUnitRowsLazyQuery({ fetchPolicy: 'no-cache' })
  const [createSQLDataExport] = useCreateSqlDataExportMutation()
  const connections = useConnectionStore((s) => s.connections)
  const connectionType = connections.find((connection) => connection.id === connectionId)?.type
  const sqlExportUnavailableReason = storageUnitType === 'view'
    ? t('sql.export.sqlUnavailableForView')
    : null
  const sqlUpdateUnavailableReason = storageUnitType === 'view'
    ? t('sql.export.updateUnavailableView')
    : connectionType === 'CLICKHOUSE'
      ? t('sql.export.updateUnavailableClickHouse')
      : primaryKeyColumns.length === 0
        ? t('sql.export.updateUnavailableNoPrimaryKey')
        : null
  const formatOptions = sqlExportUnavailableReason
    ? FORMAT_OPTIONS.filter((option) => option.id !== 'sql')
    : FORMAT_OPTIONS

  const handleExport = useCallback(async () => {
    actions.setSubmitting(true)
    actions.closeAlert()
    setIsSuccess(false)

    try {
      const graphqlSchema = resolveSchemaParam(connectionType, databaseName, schema ?? undefined)
      const limit = rowCount === '' || !Number.isFinite(rowCount) ? undefined : Math.max(rowCount, 0)

      if (format === 'sql') {
        const { data } = await createSQLDataExport({
          variables: {
            input: {
              Schema: graphqlSchema,
              StorageUnit: tableName,
              Mode: sqlUpdateUnavailableReason ? SqlDataExportMode.Insert : sqlMode,
              Where: where,
              Sort: sort,
              Limit: limit,
            },
          },
          context: { database: databaseName },
        })
        if (!data?.CreateSQLDataExport) throw new Error(t('sql.export.noDataReturned'))

        const blob = await fetchExportDownloadBlob(data.CreateSQLDataExport, databaseName)
        downloadBlob(blob, data.CreateSQLDataExport.Filename)
        setIsSuccess(true)
        return
      }

      const { Columns, Rows } = await fetchRowsForExport({
        getRows,
        databaseName,
        schema: graphqlSchema,
        tableName,
        where,
        sort,
        limit,
        noDataMessage: t('sql.export.noDataReturned'),
      })
      const blob = format === 'csv'
        ? toCSV(Columns, Rows)
        : format === 'json'
          ? toJSON(Columns, Rows)
          : toExcel(tableName, Columns, Rows)

      downloadBlob(blob, `${tableName}.${FORMAT_EXTENSIONS[format]}`)
      setIsSuccess(true)
    } catch (err: any) {
      actions.setAlert({
        type: 'error',
        title: t('sql.export.failed'),
        message: err.message || t('common.unknownError'),
      })
    } finally {
      actions.setSubmitting(false)
    }
  }, [
    actions,
    connectionType,
    createSQLDataExport,
    databaseName,
    format,
    getRows,
    rowCount,
    schema,
    sort,
    sqlMode,
    sqlUpdateUnavailableReason,
    t,
    tableName,
    where,
  ])

  return (
    <ExportDataCtx value={{
      format,
      setFormat,
      formatOptions,
      rowCount,
      setRowCount,
      sqlMode,
      setSqlMode,
      sqlExportUnavailableReason,
      sqlUpdateUnavailableReason,
      isSuccess,
      handleExport,
    }}>
      {children}
    </ExportDataCtx>
  )
}

// ---------------------------------------------------------------------------
// Subcomponents
// ---------------------------------------------------------------------------

/** Format selector, SQL statement mode, row limit, and progress display. */
function ExportDataFields() {
  const { t } = useI18n()
  const {
    format,
    setFormat,
    formatOptions,
    rowCount,
    setRowCount,
    sqlMode,
    setSqlMode,
    sqlExportUnavailableReason,
    sqlUpdateUnavailableReason,
    isSuccess,
  } = useExportDataCtx()
  const { state } = useModalForm()
  const disabled = state.isSubmitting || isSuccess

  return (
    <div className="flex flex-col gap-4">
      <FormatSelector options={formatOptions} value={format} onChange={setFormat} disabled={disabled} />
      {sqlExportUnavailableReason && (
        <p className="text-xs text-muted-foreground">{sqlExportUnavailableReason}</p>
      )}

      {format === 'sql' && (
        <div className="flex flex-col gap-2">
          <label className="text-sm font-medium text-foreground">{t('sql.export.statementType')}</label>
          <div className="inline-flex w-fit rounded-md border border-input bg-background p-1">
            <button
              type="button"
              onClick={() => setSqlMode(SqlDataExportMode.Insert)}
              disabled={disabled}
              className={cn(
                'rounded-sm px-3 py-1.5 text-sm transition-colors',
                sqlMode === SqlDataExportMode.Insert
                  ? 'bg-highlight-background text-foreground'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {t('sql.export.insertStatements')}
            </button>
            <button
              type="button"
              onClick={() => setSqlMode(SqlDataExportMode.Update)}
              disabled={disabled || Boolean(sqlUpdateUnavailableReason)}
              className={cn(
                'rounded-sm px-3 py-1.5 text-sm transition-colors',
                sqlMode === SqlDataExportMode.Update && !sqlUpdateUnavailableReason
                  ? 'bg-highlight-background text-foreground'
                  : 'text-muted-foreground hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50',
              )}
            >
              {t('sql.export.updateStatements')}
            </button>
          </div>
          {sqlUpdateUnavailableReason && (
            <p className="text-xs text-muted-foreground">{sqlUpdateUnavailableReason}</p>
          )}
        </div>
      )}

      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium text-foreground">{t('sql.export.rowLimit')}</label>
        <Input
          type="number"
          min={0}
          value={rowCount}
          onChange={(e) => setRowCount(e.target.value === '' ? '' : parseInt(e.target.value))}
          placeholder={t('sql.export.rowLimitPlaceholder')}
          disabled={disabled}
        />
        <p className="text-xs text-muted-foreground">
          {t('sql.export.rowLimitHint')}
        </p>
      </div>

      <ExportProgress isExporting={state.isSubmitting} isSuccess={isSuccess} />
    </div>
  )
}

/** Footer bridge: reads isSuccess and handleExport from domain context, delegates to shared ExportFooter. */
function ExportDataFooterBridge() {
  const { isSuccess, handleExport } = useExportDataCtx()
  return <ExportFooter isSuccess={isSuccess} onClick={handleExport} />
}

// ---------------------------------------------------------------------------
// Modal
// ---------------------------------------------------------------------------

interface ExportDataModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  connectionId: string
  databaseName: string
  schema?: string | null
  tableName: string
  storageUnitType?: StorageUnitType
  primaryKeyColumns?: string[]
  where?: WhereCondition
  sort?: SortCondition[]
}

/** Modal for exporting a single SQL table with optional row limit and table query context. */
export function ExportDataModal({
  open,
  onOpenChange,
  connectionId,
  databaseName,
  schema,
  tableName,
  storageUnitType,
  primaryKeyColumns = [],
  where,
  sort,
}: ExportDataModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <ExportDataProvider
          connectionId={connectionId}
          databaseName={databaseName}
          schema={schema}
          tableName={tableName}
          storageUnitType={storageUnitType}
          primaryKeyColumns={primaryKeyColumns}
          where={where}
          sort={sort}
        >
          <ModalForm.Header />
          <ExportDataFields />
          <ModalForm.Alert />
          <ExportDataFooterBridge />
        </ExportDataProvider>
      </DialogContent>
    </Dialog>
  )
}
