import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react'
import { AlertTriangle, CheckCircle, Database, Loader2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/Button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/Input'
import { useI18n } from '@/i18n/useI18n'
import { cn } from '@/lib/utils'
import { useConnectionStore } from '@/stores/useConnectionStore'
import { resolveSchemaParam } from '@/utils/database-features'
import {
  CollectionImportFormat,
  ImportMode,
  useGetStorageUnitsQuery,
  useImportCollectionFileMutation,
  useImportCollectionPreviewMutation,
  type CollectionImportError,
  type CollectionImportPreview,
  type CollectionImportResult,
} from '@graphql'

const SELECT_CLASS =
  'h-9 w-full rounded-md border border-input bg-background px-3 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:opacity-50'

const CSV_DELIMITER_OPTIONS = [
  { value: '', labelKey: 'database.import.collection.delimiter.auto' },
  { value: ',', labelKey: 'database.import.collection.delimiter.comma' },
  { value: ';', labelKey: 'database.import.collection.delimiter.semicolon' },
  { value: '|', labelKey: 'database.import.collection.delimiter.pipe' },
  { value: '\t', labelKey: 'database.import.collection.delimiter.tab' },
] as const

type TargetMode = 'existing' | 'new'

/** Infers the import format from a file name, or null when unsupported. */
function inferCollectionFormat(filename: string): CollectionImportFormat | null {
  const lower = filename.toLowerCase()
  if (lower.endsWith('.json') || lower.endsWith('.ndjson') || lower.endsWith('.jsonl')) return CollectionImportFormat.Json
  if (lower.endsWith('.csv')) return CollectionImportFormat.Csv
  if (lower.endsWith('.xls') || lower.endsWith('.xlsx')) return CollectionImportFormat.Excel
  return null
}

export interface CollectionImportModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  connectionId: string
  databaseName: string
  /** When provided, the import targets this existing collection and the target is locked. */
  collectionName?: string | null
  onSuccess?: () => void
}

/**
 * Modal for Collection File Import: loads a JSON, CSV, or Excel file into one
 * existing or newly created MongoDB collection.
 */
export function CollectionImportModal({
  open,
  onOpenChange,
  connectionId,
  databaseName,
  collectionName,
  onSuccess,
}: CollectionImportModalProps) {
  const { t } = useI18n()
  const { connections } = useConnectionStore()
  const connectionType = useMemo(
    () => connections.find((item) => item.id === connectionId)?.type,
    [connections, connectionId],
  )
  const schemaParam = useMemo(
    () => resolveSchemaParam(connectionType, databaseName),
    [connectionType, databaseName],
  )

  const lockedCollection = collectionName ?? null
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const [file, setFile] = useState<File | null>(null)
  const [format, setFormat] = useState<CollectionImportFormat | null>(null)
  const [fileError, setFileError] = useState<string | null>(null)
  const [targetMode, setTargetMode] = useState<TargetMode>('existing')
  const [selectedCollection, setSelectedCollection] = useState(lockedCollection ?? '')
  const [newCollectionName, setNewCollectionName] = useState('')
  const [mode, setMode] = useState<ImportMode>(ImportMode.Append)
  const [upsertKeys, setUpsertKeys] = useState('_id')
  const [delimiter, setDelimiter] = useState('')
  const [sheet, setSheet] = useState('')
  const [skippedColumns, setSkippedColumns] = useState<Set<string>>(new Set())
  const [preview, setPreview] = useState<CollectionImportPreview | null>(null)
  const [result, setResult] = useState<CollectionImportResult | null>(null)

  const { data: storageUnitsData } = useGetStorageUnitsQuery({
    variables: { schema: schemaParam },
    context: { database: databaseName },
    fetchPolicy: 'no-cache',
    skip: !open || lockedCollection != null,
  })
  const existingCollections = useMemo(
    () => storageUnitsData?.StorageUnit.map((unit) => unit.Name) ?? [],
    [storageUnitsData],
  )

  const [runPreview, { loading: previewLoading }] = useImportCollectionPreviewMutation()
  const [runImport, { loading: importing }] = useImportCollectionFileMutation()

  const resetState = useCallback(() => {
    setFile(null)
    setFormat(null)
    setFileError(null)
    setTargetMode('existing')
    setSelectedCollection(lockedCollection ?? '')
    setNewCollectionName('')
    setMode(ImportMode.Append)
    setUpsertKeys('_id')
    setDelimiter('')
    setSheet('')
    setSkippedColumns(new Set())
    setPreview(null)
    setResult(null)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }, [lockedCollection])

  useEffect(() => {
    if (!open) resetState()
  }, [open, resetState])

  const loadPreview = useCallback(
    async (nextFile: File, nextFormat: CollectionImportFormat) => {
      const response = await runPreview({
        variables: {
          input: {
            Format: nextFormat,
            File: nextFile,
            Delimiter: nextFormat === CollectionImportFormat.Csv && delimiter ? delimiter : undefined,
            Sheet: nextFormat === CollectionImportFormat.Excel && sheet.trim() ? sheet.trim() : undefined,
          },
        },
        context: { database: databaseName },
      })
      const nextPreview = response.data?.ImportCollectionPreview ?? null
      setPreview(nextPreview)
      setSkippedColumns(new Set())
    },
    [runPreview, delimiter, sheet, databaseName],
  )

  const handleFileChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const selected = event.target.files?.[0] ?? null
      setResult(null)
      setPreview(null)
      if (!selected) {
        setFile(null)
        setFormat(null)
        return
      }
      const inferred = inferCollectionFormat(selected.name)
      if (!inferred) {
        setFile(null)
        setFormat(null)
        setFileError(t('database.import.collection.invalidFile'))
        return
      }
      setFileError(null)
      setFile(selected)
      setFormat(inferred)
      void loadPreview(selected, inferred)
    },
    [loadPreview, t],
  )

  const toggleColumn = useCallback((column: string) => {
    setSkippedColumns((current) => {
      const next = new Set(current)
      if (next.has(column)) next.delete(column)
      else next.add(column)
      return next
    })
  }, [])

  const effectiveMode = targetMode === 'new' ? ImportMode.Append : mode
  const targetCollection = lockedCollection ?? (targetMode === 'existing' ? selectedCollection : newCollectionName.trim())

  const canRun =
    file != null &&
    format != null &&
    targetCollection.length > 0 &&
    !preview?.ValidationError &&
    !importing

  const executeImport = useCallback(async () => {
    if (!file || !format || !targetCollection) return
    const skipColumns =
      format === CollectionImportFormat.Json ? undefined : Array.from(skippedColumns)
    const response = await runImport({
      variables: {
        input: {
          Schema: schemaParam,
          Collection: targetCollection,
          Format: format,
          Mode: effectiveMode,
          UpsertKeys:
            effectiveMode === ImportMode.Upsert
              ? upsertKeys.split(',').map((key) => key.trim()).filter(Boolean)
              : undefined,
          SkipColumns: skipColumns && skipColumns.length > 0 ? skipColumns : undefined,
          Delimiter: format === CollectionImportFormat.Csv && delimiter ? delimiter : undefined,
          Sheet: format === CollectionImportFormat.Excel && sheet.trim() ? sheet.trim() : undefined,
          File: file,
        },
      },
      context: { database: databaseName },
    })
    const importResult = response.data?.ImportCollectionFile ?? null
    setResult(importResult)
    if (importResult?.Status) onSuccess?.()
  }, [
    file, format, targetCollection, skippedColumns, runImport, schemaParam,
    effectiveMode, upsertKeys, delimiter, sheet, databaseName, onSuccess,
  ])

  const showDelimiter = format === CollectionImportFormat.Csv
  const showSheet = format === CollectionImportFormat.Excel

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Database className="h-4 w-4" />
            {t('database.import.collection.title')}
          </DialogTitle>
          <DialogDescription>
            {lockedCollection ? `${databaseName} / ${lockedCollection}` : databaseName}
          </DialogDescription>
        </DialogHeader>

        <div className="flex max-h-[60vh] flex-col gap-4 overflow-y-auto pr-1">
          {/* File */}
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium text-foreground">{t('database.import.collection.file')}</label>
            <Input
              ref={fileInputRef}
              type="file"
              accept=".json,.ndjson,.jsonl,.csv,.xls,.xlsx"
              onChange={handleFileChange}
              disabled={importing}
              aria-label={t('database.import.collection.uploadFile')}
            />
            {file && (
              <span className="truncate text-xs text-muted-foreground">
                {t('database.import.collection.selectedFile', { filename: file.name })}
              </span>
            )}
            {fileError && <span className="text-xs text-destructive">{fileError}</span>}
          </div>

          {/* Target */}
          {!lockedCollection && (
            <div className="flex flex-col gap-2">
              <span className="text-sm font-medium text-foreground">{t('database.import.collection.targetMode')}</span>
              <div className="inline-flex w-fit rounded-md border border-input p-1">
                <button
                  type="button"
                  onClick={() => setTargetMode('existing')}
                  aria-pressed={targetMode === 'existing'}
                  className={cn(
                    'rounded-sm px-3 py-1.5 text-sm transition-colors',
                    targetMode === 'existing' ? 'bg-highlight-background text-foreground' : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {t('database.import.collection.targetModeExisting')}
                </button>
                <button
                  type="button"
                  onClick={() => setTargetMode('new')}
                  aria-pressed={targetMode === 'new'}
                  className={cn(
                    'rounded-sm px-3 py-1.5 text-sm transition-colors',
                    targetMode === 'new' ? 'bg-highlight-background text-foreground' : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {t('database.import.collection.targetModeNew')}
                </button>
              </div>
              {targetMode === 'existing' ? (
                <select
                  value={selectedCollection}
                  onChange={(event) => setSelectedCollection(event.target.value)}
                  disabled={importing}
                  className={SELECT_CLASS}
                  aria-label={t('database.import.collection.target')}
                >
                  <option value="" disabled>
                    {existingCollections.length === 0
                      ? t('database.import.collection.noCollections')
                      : t('database.import.collection.selectCollection')}
                  </option>
                  {existingCollections.map((name) => (
                    <option key={name} value={name}>{name}</option>
                  ))}
                </select>
              ) : (
                <Input
                  value={newCollectionName}
                  onChange={(event) => setNewCollectionName(event.target.value)}
                  placeholder={t('database.import.collection.newCollectionNamePlaceholder')}
                  disabled={importing}
                  aria-label={t('database.import.collection.newCollectionName')}
                />
              )}
            </div>
          )}

          {/* Mode */}
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium text-foreground">{t('database.import.collection.mode')}</label>
            <select
              value={effectiveMode}
              onChange={(event) => setMode(event.target.value as ImportMode)}
              disabled={importing || targetMode === 'new'}
              className={SELECT_CLASS}
            >
              <option value={ImportMode.Append}>{t('database.import.collection.mode.append')}</option>
              <option value={ImportMode.Overwrite}>{t('database.import.collection.mode.overwrite')}</option>
              <option value={ImportMode.Upsert}>{t('database.import.collection.mode.upsert')}</option>
            </select>
            {targetMode === 'new' && (
              <span className="text-xs text-muted-foreground">{t('database.import.collection.newCollectionAppendOnly')}</span>
            )}
          </div>

          {/* Upsert keys */}
          {effectiveMode === ImportMode.Upsert && (
            <div className="flex flex-col gap-2">
              <label className="text-sm font-medium text-foreground">{t('database.import.collection.upsertKeys')}</label>
              <Input
                value={upsertKeys}
                onChange={(event) => setUpsertKeys(event.target.value)}
                placeholder="_id"
                disabled={importing}
              />
              <span className="text-xs text-muted-foreground">{t('database.import.collection.upsertKeysHint')}</span>
            </div>
          )}

          {/* CSV delimiter / Excel sheet */}
          {(showDelimiter || showSheet) && (
            <div className="flex flex-wrap items-end gap-3">
              {showDelimiter && (
                <div className="flex flex-1 flex-col gap-2">
                  <label className="text-sm font-medium text-foreground">{t('database.import.collection.csvDelimiter')}</label>
                  <select value={delimiter} onChange={(event) => setDelimiter(event.target.value)} disabled={importing} className={SELECT_CLASS}>
                    {CSV_DELIMITER_OPTIONS.map((option) => (
                      <option key={option.labelKey} value={option.value}>{t(option.labelKey)}</option>
                    ))}
                  </select>
                </div>
              )}
              {showSheet && (
                <div className="flex flex-1 flex-col gap-2">
                  <label className="text-sm font-medium text-foreground">{t('database.import.collection.excelSheet')}</label>
                  <Input
                    value={sheet}
                    onChange={(event) => setSheet(event.target.value)}
                    placeholder={t('database.import.collection.excelSheetPlaceholder')}
                    disabled={importing}
                  />
                </div>
              )}
              {file && format && (
                <Button type="button" variant="outline" size="sm" onClick={() => void loadPreview(file, format)} disabled={previewLoading || importing}>
                  {t('database.import.collection.refreshPreview')}
                </Button>
              )}
            </div>
          )}

          {/* Preview */}
          {previewLoading && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t('database.import.collection.previewLoading')}
            </div>
          )}
          {preview && !previewLoading && (
            <CollectionImportPreviewView
              preview={preview}
              skippedColumns={skippedColumns}
              onToggleColumn={toggleColumn}
            />
          )}

          {/* Overwrite warning */}
          {effectiveMode === ImportMode.Overwrite && (
            <div className="flex items-start gap-2 rounded-md border border-warning/20 bg-warning/5 p-3 text-sm text-warning">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{t('database.import.collection.overwriteWarning')}</span>
            </div>
          )}

          {/* Result */}
          {result && <CollectionImportResultView result={result} />}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={importing}>
            {t('common.actions.cancel')}
          </Button>
          <Button type="button" onClick={() => void executeImport()} disabled={!canRun}>
            {importing && <Loader2 className="h-4 w-4 animate-spin" />}
            {importing ? t('database.import.collection.running') : t('database.import.collection.runImport')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Renders the parsed sample: a tabular grid for CSV/Excel or a document list for JSON. */
function CollectionImportPreviewView({
  preview,
  skippedColumns,
  onToggleColumn,
}: {
  preview: CollectionImportPreview
  skippedColumns: Set<string>
  onToggleColumn: (column: string) => void
}) {
  const { t } = useI18n()

  if (preview.ValidationError) {
    return (
      <div role="alert" className="flex items-start gap-2 rounded-md border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        <span>{formatCollectionImportError(t, preview.ValidationError)}</span>
      </div>
    )
  }

  if (preview.Format === CollectionImportFormat.Json) {
    return (
      <div className="flex flex-col gap-2">
        <span className="text-sm font-medium text-foreground">
          {preview.Count != null
            ? t('database.import.collection.detected', { count: preview.Count })
            : t('database.import.collection.previewDocuments')}
        </span>
        <div className="flex max-h-60 flex-col gap-2 overflow-y-auto rounded-md border border-input p-2">
          {preview.Documents.map((document, index) => (
            <pre key={index} className="whitespace-pre-wrap break-words font-mono text-xs leading-relaxed">{document}</pre>
          ))}
        </div>
        {preview.Truncated && <span className="text-xs text-muted-foreground">{t('database.import.collection.truncated')}</span>}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <span className="text-sm font-medium text-foreground">{t('database.import.collection.skipColumns')}</span>
      <div className="overflow-x-auto rounded-md border border-input">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-input">
              {preview.Columns.map((column) => (
                <th key={column} className="px-2 py-1.5 text-left font-medium">
                  <label className="flex items-center gap-1.5">
                    <Checkbox
                      checked={!skippedColumns.has(column)}
                      onCheckedChange={() => onToggleColumn(column)}
                    />
                    <span className={cn('truncate', skippedColumns.has(column) && 'text-muted-foreground line-through')}>{column}</span>
                  </label>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {preview.Rows.map((row, rowIndex) => (
              <tr key={rowIndex} className="border-b border-input/50 last:border-0">
                {preview.Columns.map((column, columnIndex) => (
                  <td key={column} className={cn('px-2 py-1', skippedColumns.has(column) && 'text-muted-foreground/50')}>
                    {row[columnIndex] ?? ''}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {preview.Truncated && <span className="text-xs text-muted-foreground">{t('database.import.collection.truncated')}</span>}
    </div>
  )
}

/** Renders the import outcome: success counts or a failure message. */
function CollectionImportResultView({ result }: { result: CollectionImportResult }) {
  const { t } = useI18n()

  if (!result.Status) {
    return (
      <div role="alert" className="flex items-start gap-2 rounded-md border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        <div className="min-w-0">
          <span>{formatCollectionImportError(t, result.Detail)}</span>
          {result.Message && (
            <pre className="mt-2 whitespace-pre-wrap break-words font-mono text-xs leading-relaxed">{result.Message}</pre>
          )}
        </div>
      </div>
    )
  }

  const summary =
    result.SkippedCount > 0
      ? t('database.import.collection.successWithSkipped', { imported: result.ImportedCount, skipped: result.SkippedCount })
      : t('database.import.collection.successMessage', { imported: result.ImportedCount })

  return (
    <div role="status" className="flex flex-col gap-2 rounded-md border border-success/20 bg-success/5 p-3 text-sm text-success">
      <div className="flex items-center gap-2">
        <CheckCircle className="h-4 w-4 shrink-0" />
        <span>{summary}</span>
      </div>
      {result.MatchedCount != null && (
        <span className="text-xs">
          {t('database.import.collection.upsertSummary', {
            matched: result.MatchedCount,
            modified: result.ModifiedCount ?? 0,
            upserted: result.UpsertedCount ?? 0,
          })}
        </span>
      )}
      {result.Errors.length > 0 && (
        <div className="mt-1 flex flex-col gap-1 text-xs text-warning">
          <span>{t('database.import.collection.skippedErrors')}</span>
          {result.Errors.map((error: CollectionImportError) => (
            <span key={error.Index} className="font-mono">
              {t('database.import.collection.skippedError', { index: error.Index, reason: error.Reason })}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}

/** Maps a backend import i18n key to a localized message. */
function formatCollectionImportError(t: ReturnType<typeof useI18n>['t'], detail?: string | null): string {
  switch (detail) {
    case 'import.error.collection_unsupported':
      return t('database.import.error.collectionUnsupported')
    case 'import.validation.missing_file':
      return t('database.import.error.missingFile')
    case 'import.validation.file_too_large':
      return t('database.import.error.fileTooLarge')
    case 'import.validation.unsupported_format':
      return t('database.import.error.unsupportedFormat')
    case 'import.validation.parse_failed':
      return t('database.import.error.parseFailed')
    case 'import.validation.ambiguous_delimiter':
      return t('database.import.error.ambiguousDelimiter')
    case 'import.validation.invalid_delimiter':
      return t('database.import.error.invalidDelimiter')
    case 'import.validation.no_columns':
      return t('database.import.error.noColumns')
    case 'import.validation.row_too_many_columns':
      return t('database.import.error.rowTooManyColumns')
    case 'import.validation.row_limit_exceeded':
      return t('database.import.error.rowLimitExceeded')
    case 'import.validation.duplicate_header':
      return t('database.import.error.duplicateHeader')
    case 'import.validation.empty_header':
      return t('database.import.error.emptyHeader')
    case 'import.error.no_rows':
      return t('database.import.error.noRows')
    case 'import.error.clear_failed':
      return t('database.import.error.clearFailed')
    case 'import.error.import_failed':
      return t('database.import.error.importFailed')
    default:
      return t('database.import.error.parseFailed')
  }
}
