import { createContext, use, useCallback, useState, type ReactNode } from 'react'
import { Database, FileJson, FileSpreadsheet, FileCode, FileText } from 'lucide-react'
import { SqlDataExportMode, useCreateSqlDataExportMutation, useRawExecuteLazyQuery } from '@/generated/graphql'
import { toCSV, toJSON, toExcel, downloadBlob } from '@/utils/export-utils'
import { fetchExportDownloadBlob } from '@/utils/export-download'
import JSZip from 'jszip'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { ModalForm, useModalForm } from '@/components/ui/ModalForm'
import { FormatSelector, type FormatOption } from '@/components/database/shared/FormatSelector'
import { ExportProgress, ExportFooter } from '@/components/database/shared/ExportProgress'
import { useI18n } from '@/i18n/useI18n'
import { useConnectionStore } from '@/stores/useConnectionStore'
import { buildStorageUnitReference } from '@/utils/ddl-sql'
import { resolveSchemaParam } from '@/utils/database-features'
import {
  buildDatabaseExportPlan,
  formatDatabaseExportEntryName,
  formatDatabaseExportTargetName,
} from '@/utils/database-export'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

type ExportFormat = 'csv' | 'json' | 'sql' | 'excel'

const FORMAT_OPTIONS: FormatOption<ExportFormat>[] = [
  { id: 'sql', label: 'SQL', icon: FileCode },
  { id: 'json', label: 'JSON', icon: FileJson },
  { id: 'csv', label: 'CSV', icon: FileText },
  { id: 'excel', label: 'Excel', icon: FileSpreadsheet },
]

function isViewStorageUnit(type: string): boolean {
  return type.toLowerCase().includes('view')
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

interface ExportDatabaseCtxValue {
  format: ExportFormat
  setFormat: (v: ExportFormat) => void
  isSuccess: boolean
  statusText: string
  handleExport: () => void
}

const ExportDatabaseCtx = createContext<ExportDatabaseCtxValue | null>(null)

/** Hook to access ExportDatabaseModal domain state. Throws if used outside the provider. */
function useExportDatabaseCtx(): ExportDatabaseCtxValue {
  const ctx = use(ExportDatabaseCtx)
  if (!ctx) throw new Error('useExportDatabaseCtx must be used within ExportDatabaseProvider')
  return ctx
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

/** Wraps ModalForm.Provider (complex mode, no onSubmit) and domain context for database ZIP export. */
function ExportDatabaseProvider({
  connectionId,
  databaseName,
  schema,
  children,
}: {
  connectionId: string
  databaseName: string
  schema: string
  children: ReactNode
}) {
  const { t } = useI18n()
  return (
    <ModalForm.Provider
      meta={{ title: t('database.export.title'), description: databaseName, icon: Database }}
    >
      <ExportDatabaseBridge connectionId={connectionId} databaseName={databaseName} schema={schema}>
        {children}
      </ExportDatabaseBridge>
    </ModalForm.Provider>
  )
}

/**
 * Inner bridge that owns domain state and multi-table export logic.
 * Fetches table list via GraphQL, iterates each table, converts to selected format,
 * bundles into ZIP via JSZip, triggers download. Partial failures surface as an info alert.
 */
function ExportDatabaseBridge({
  connectionId,
  databaseName,
  schema,
  children,
}: {
  connectionId: string
  databaseName: string
  schema: string
  children: ReactNode
}) {
  const { t } = useI18n()
  const [format, setFormat] = useState<ExportFormat>('sql')
  const [isSuccess, setIsSuccess] = useState(false)
  const [statusText, setStatusText] = useState('')
  const { actions } = useModalForm()
  const [executeQuery] = useRawExecuteLazyQuery({ fetchPolicy: 'no-cache' })
  const [createSQLDataExport] = useCreateSqlDataExportMutation()
  const connections = useConnectionStore((s) => s.connections)
  const fetchSchemas = useConnectionStore((s) => s.fetchSchemas)
  const fetchTables = useConnectionStore((s) => s.fetchTables)
  const systemSchemas = useConnectionStore((s) => s.systemSchemas)
  const showSystemObjectsFor = useConnectionStore((s) => s.showSystemObjectsFor)

  const handleExport = useCallback(async () => {
    actions.setSubmitting(true)
    actions.closeAlert()
    setIsSuccess(false)
    setStatusText(t('database.export.fetchingTableList'))

    try {
      const connectionType = connections.find((connection) => connection.id === connectionId)?.type
      const databaseNodeId = `${connectionId}-${databaseName}`
      const allSchemas = connectionType === 'POSTGRES'
        ? await fetchSchemas(connectionId, databaseName)
        : []
      const schemasToExport = buildDatabaseExportPlan({
        connectionType,
        fallbackSchema: schema,
        allSchemas,
        systemSchemas,
        includeSystemSchemas: showSystemObjectsFor.has(databaseNodeId),
      })
      const exportTargets: Array<{ schema: string; tableName: string }> = []

      for (const schemaName of schemasToExport) {
        const tables = await fetchTables(connectionId, databaseName, schemaName)
        for (const table of tables) {
          if (format === 'sql' && isViewStorageUnit(table.type)) continue
          exportTargets.push({ schema: schemaName, tableName: table.name })
        }
      }

      if (exportTargets.length === 0) throw new Error(t('database.export.noTablesFound'))

      const zip = new JSZip()
      const failedTables: string[] = []

      for (let i = 0; i < exportTargets.length; i++) {
        const target = exportTargets[i]
        const targetLabel = formatDatabaseExportTargetName(connectionType, target.schema, target.tableName)
        setStatusText(t('database.export.exportingTable', {
          current: i + 1,
          total: exportTargets.length,
          tableName: targetLabel,
        }))

        try {
          if (format === 'sql') {
            const graphqlSchema = resolveSchemaParam(connectionType, databaseName, target.schema)
            const { data } = await createSQLDataExport({
              variables: {
                input: {
                  Schema: graphqlSchema,
                  StorageUnit: target.tableName,
                  Mode: SqlDataExportMode.Insert,
                },
              },
              context: { database: databaseName },
            })

            if (!data?.CreateSQLDataExport) {
              failedTables.push(targetLabel)
              continue
            }

            const blob = await fetchExportDownloadBlob(data.CreateSQLDataExport, databaseName)
            zip.file(formatDatabaseExportEntryName(connectionType, target.schema, target.tableName, format), blob)
            continue
          }

          const qualifiedName = buildStorageUnitReference(connectionType, target.tableName, target.schema)
          const { data, error } = await executeQuery({
            variables: { query: `SELECT * FROM ${qualifiedName}` },
            context: { database: databaseName },
          })

          if (error || !data?.RawExecute) {
            failedTables.push(targetLabel)
            continue
          }

          const { Columns, Rows } = data.RawExecute
          let blob: Blob

          switch (format) {
            case 'csv': blob = toCSV(Columns, Rows); break
            case 'json': blob = toJSON(Columns, Rows); break
            case 'excel': blob = toExcel(target.tableName, Columns, Rows); break
          }

          zip.file(formatDatabaseExportEntryName(connectionType, target.schema, target.tableName, format), blob)
        } catch {
          failedTables.push(targetLabel)
        }
      }

      setStatusText(t('database.export.generatingZip'))

      const zipBlob = await zip.generateAsync({ type: 'blob' })
      downloadBlob(zipBlob, `export_${databaseName}.zip`)

      setIsSuccess(true)

      if (failedTables.length > 0) {
        actions.setAlert({
          type: 'info',
          title: t('database.export.partialExportTitle'),
          message: t('database.export.partialExportMessage', {
            successful: exportTargets.length - failedTables.length,
            total: exportTargets.length,
            failedTables: failedTables.join(', '),
          }),
        })
      }
    } catch (err: any) {
      actions.setAlert({
        type: 'error',
        title: t('database.export.failed'),
        message: err.message || t('common.unknownError'),
      })
    } finally {
      actions.setSubmitting(false)
    }
  }, [
    actions,
    connectionId,
    connections,
    createSQLDataExport,
    databaseName,
    executeQuery,
    fetchSchemas,
    fetchTables,
    format,
    schema,
    showSystemObjectsFor,
    systemSchemas,
    t,
  ])

  return (
    <ExportDatabaseCtx value={{ format, setFormat, isSuccess, statusText, handleExport }}>
      {children}
    </ExportDatabaseCtx>
  )
}

// ---------------------------------------------------------------------------
// Subcomponents
// ---------------------------------------------------------------------------

/** Format selector and progress display for database export. */
function ExportDatabaseFields() {
  const { format, setFormat, isSuccess, statusText } = useExportDatabaseCtx()
  const { state } = useModalForm()
  const disabled = state.isSubmitting || isSuccess

  return (
    <div className="flex flex-col gap-4">
      <FormatSelector options={FORMAT_OPTIONS} value={format} onChange={setFormat} disabled={disabled} />
      <ExportProgress isExporting={state.isSubmitting} isSuccess={isSuccess} statusText={statusText} />
    </div>
  )
}

/** Footer bridge: reads isSuccess and handleExport from domain context, delegates to shared ExportFooter. */
function ExportDatabaseFooterBridge() {
  const { isSuccess, handleExport } = useExportDatabaseCtx()
  return <ExportFooter isSuccess={isSuccess} onClick={handleExport} />
}

// ---------------------------------------------------------------------------
// Modal
// ---------------------------------------------------------------------------

interface ExportDatabaseModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  connectionId: string
  databaseName: string
  schema: string
}

/** Modal for exporting all tables in a database as a ZIP archive. */
export function ExportDatabaseModal({
  open,
  onOpenChange,
  connectionId,
  databaseName,
  schema,
}: ExportDatabaseModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <ExportDatabaseProvider connectionId={connectionId} databaseName={databaseName} schema={schema}>
          <ModalForm.Header />
          <ExportDatabaseFields />
          <ModalForm.Alert />
          <ExportDatabaseFooterBridge />
        </ExportDatabaseProvider>
      </DialogContent>
    </Dialog>
  )
}
