import { createContext, use, useCallback, useState, type ReactNode } from 'react'
import { Download, FileJson, FileSpreadsheet, FileText } from 'lucide-react'
import { useConnectionStore } from '@/stores/useConnectionStore'
import { addAuthHeader } from '@/config/auth-headers'
import { resolveSchemaParam } from '@/utils/database-features'
import { downloadBlob } from '@/utils/export-utils'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Input } from '@/components/ui/Input'
import { ModalForm, useModalForm } from '@/components/ui/ModalForm'
import { FormatSelector, type FormatOption } from '@/components/database/shared/FormatSelector'
import { ExportProgress, ExportFooter } from '@/components/database/shared/ExportProgress'
import { useI18n } from '@/i18n/useI18n'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

type CollectionExportFormat = 'json' | 'csv' | 'excel'

const FORMAT_OPTIONS: FormatOption<CollectionExportFormat>[] = [
  { id: 'json', label: 'JSON', icon: FileJson },
  { id: 'csv', label: 'CSV', icon: FileText },
  { id: 'excel', label: 'Excel', icon: FileSpreadsheet },
]

const BACKEND_FORMATS: Record<CollectionExportFormat, 'ndjson' | 'csv' | 'excel'> = {
  json: 'ndjson',
  csv: 'csv',
  excel: 'excel',
}

const FORMAT_EXTENSIONS: Record<CollectionExportFormat, string> = {
  json: 'ndjson',
  csv: 'csv',
  excel: 'xlsx',
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

interface ExportCollectionCtxValue {
  format: CollectionExportFormat
  setFormat: (v: CollectionExportFormat) => void
  filter: string
  setFilter: (v: string) => void
  limit: number | ''
  setLimit: (v: number | '') => void
  isSuccess: boolean
  handleExport: () => void
}

const ExportCollectionCtx = createContext<ExportCollectionCtxValue | null>(null)

/** Hook to access ExportCollectionModal domain state. Throws if used outside the provider. */
function useExportCollectionCtx(): ExportCollectionCtxValue {
  const ctx = use(ExportCollectionCtx)
  if (!ctx) throw new Error('useExportCollectionCtx must be used within ExportCollectionProvider')
  return ctx
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

/** Wraps ModalForm.Provider (complex mode, no onSubmit) and domain context for MongoDB collection export. */
function ExportCollectionProvider({
  connectionId,
  databaseName,
  collectionName,
  children,
}: {
  connectionId: string
  databaseName: string
  collectionName: string
  children: ReactNode
}) {
  const { t } = useI18n()

  return (
    <ModalForm.Provider
      meta={{ title: t('mongodb.export.title'), description: collectionName, icon: Download }}
    >
      <ExportCollectionBridge
        connectionId={connectionId}
        databaseName={databaseName}
        collectionName={collectionName}
      >
        {children}
      </ExportCollectionBridge>
    </ModalForm.Provider>
  )
}

/**
 * Inner bridge that owns domain state and export logic.
 * POSTs to REST `/api/export` endpoint (the only non-GraphQL export modal),
 * maps JSON to NDJSON for backend, triggers download via downloadBlob utility.
 */
function ExportCollectionBridge({
  connectionId,
  databaseName,
  collectionName,
  children,
}: {
  connectionId: string
  databaseName: string
  collectionName: string
  children: ReactNode
}) {
  const { t } = useI18n()
  const { connections } = useConnectionStore()
  const [format, setFormat] = useState<CollectionExportFormat>('json')
  const [filter, setFilter] = useState('')
  const [limit, setLimit] = useState<number | ''>('')
  const [isSuccess, setIsSuccess] = useState(false)
  const { actions } = useModalForm()

  const handleExport = useCallback(async () => {
    actions.setSubmitting(true)
    actions.closeAlert()
    setIsSuccess(false)

    try {
      const connection = connections.find((c) => c.id === connectionId)
      if (!connection) throw new Error(t('common.error.connectionNotFound'))

      const graphqlSchema = resolveSchemaParam(connection.type, databaseName)
      const backendFormat = BACKEND_FORMATS[format]

      const response = await fetch('/api/export', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...addAuthHeader({}, databaseName),
        },
        body: JSON.stringify({
          schema: graphqlSchema,
          storageUnit: collectionName,
          format: backendFormat,
          filter: filter.trim() || undefined,
          limit: typeof limit === 'number' ? limit : undefined,
        }),
      })

      if (!response.ok) {
        const text = await response.text()
        throw new Error(text || t('mongodb.export.failedWithStatus', { status: response.status }))
      }

      const disposition = response.headers.get('Content-Disposition')
      const filenameMatch = disposition?.match(/filename="(.+)"/)
      const filename =
        filenameMatch?.[1] ?? `${collectionName}_export.${FORMAT_EXTENSIONS[format]}`

      const blob = await response.blob()
      downloadBlob(blob, filename)

      setIsSuccess(true)
    } catch (e: any) {
      actions.setAlert({
        type: 'error',
        title: t('mongodb.export.failed'),
        message: e.message || t('mongodb.export.errorOccurred'),
      })
    } finally {
      actions.setSubmitting(false)
    }
  }, [actions, collectionName, connectionId, connections, databaseName, filter, format, limit, t])

  return (
    <ExportCollectionCtx
      value={{ format, setFormat, filter, setFilter, limit, setLimit, isSuccess, handleExport }}
    >
      {children}
    </ExportCollectionCtx>
  )
}

// ---------------------------------------------------------------------------
// Subcomponents
// ---------------------------------------------------------------------------

/** Format selector, filter query, row limit, and progress display. */
function ExportCollectionFields() {
  const { t } = useI18n()
  const { format, setFormat, filter, setFilter, limit, setLimit, isSuccess } =
    useExportCollectionCtx()
  const { state } = useModalForm()
  const disabled = state.isSubmitting || isSuccess

  return (
    <div className="flex flex-col gap-4">
      <FormatSelector options={FORMAT_OPTIONS} value={format} onChange={setFormat} disabled={disabled} />

      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium text-foreground">{t('mongodb.export.filterQuery')}</label>
        <Input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder={t('mongodb.export.filterPlaceholder')}
          className="font-mono text-sm"
          disabled={disabled}
        />
        <p className="text-xs text-muted-foreground">
          {t('mongodb.export.filterHint')}
        </p>
      </div>

      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium text-foreground">{t('mongodb.export.limitRows')}</label>
        <Input
          type="number"
          value={limit}
          onChange={(e) => setLimit(e.target.value ? parseInt(e.target.value) : '')}
          placeholder={t('mongodb.export.limitPlaceholder')}
          className="font-mono text-sm"
          min={1}
          disabled={disabled}
        />
      </div>

      <ExportProgress isExporting={state.isSubmitting} isSuccess={isSuccess} />
    </div>
  )
}

/** Footer bridge: reads isSuccess and handleExport from domain context, delegates to shared ExportFooter. */
function ExportCollectionFooterBridge() {
  const { isSuccess, handleExport } = useExportCollectionCtx()
  return <ExportFooter isSuccess={isSuccess} onClick={handleExport} />
}

// ---------------------------------------------------------------------------
// Modal
// ---------------------------------------------------------------------------

interface ExportCollectionModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  connectionId: string
  databaseName: string
  collectionName: string
}

/** Modal for exporting a MongoDB collection via REST `/api/export` endpoint. */
export function ExportCollectionModal({
  open,
  onOpenChange,
  connectionId,
  databaseName,
  collectionName,
}: ExportCollectionModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <ExportCollectionProvider
          connectionId={connectionId}
          databaseName={databaseName}
          collectionName={collectionName}
        >
          <ModalForm.Header />
          <ExportCollectionFields />
          <ModalForm.Alert />
          <ExportCollectionFooterBridge />
        </ExportCollectionProvider>
      </DialogContent>
    </Dialog>
  )
}
