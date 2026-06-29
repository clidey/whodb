import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react'
import { Database, Loader2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { ImportFilePicker } from '@/components/database/shared/import/ImportFilePicker'
import { ImportModeSelect } from '@/components/database/shared/import/ImportModeSelect'
import { ImportNotice } from '@/components/database/shared/import/ImportNotice'
import { ImportPreviewTable } from '@/components/database/shared/import/ImportPreviewTable'
import { ImportSourceOptions } from '@/components/database/shared/import/ImportSourceOptions'
import { ImportTargetModeToggle } from '@/components/database/shared/import/ImportTargetModeToggle'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useI18n } from '@/i18n/useI18n'
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
  const [sheetOptions, setSheetOptions] = useState<string[]>([])
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
    setSheetOptions([])
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
      setResult(null)
      setPreview(null)
      if (nextFormat !== CollectionImportFormat.Excel) {
        setSheetOptions([])
      }

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
      setSheetOptions(nextPreview?.Sheets ?? [])
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
        setSheet('')
        setSheetOptions([])
        return
      }
      const inferred = inferCollectionFormat(selected.name)
      if (!inferred) {
        setFile(null)
        setFormat(null)
        setSheet('')
        setSheetOptions([])
        setFileError(t('database.import.collection.invalidFile'))
        return
      }
      setFileError(null)
      setSheet('')
      setSheetOptions([])
      setFile(selected)
      setFormat(inferred)
    },
    [t],
  )

  useEffect(() => {
    if (!file || !format) return
    void loadPreview(file, format)
  }, [delimiter, file, format, loadPreview, sheet])

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
    preview != null &&
    targetCollection.length > 0 &&
    !preview?.ValidationError &&
    !previewLoading &&
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
  const showSheet = format === CollectionImportFormat.Excel && sheetOptions.length > 0
  const selectedSheet = sheet || preview?.Sheet || sheetOptions[0] || ''

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
          <ImportFilePicker
            label={t('database.import.collection.file')}
            accept=".json,.ndjson,.jsonl,.csv,.xls,.xlsx"
            ariaLabel={t('database.import.collection.uploadFile')}
            onChange={handleFileChange}
            selectedFileName={file ? t('database.import.collection.selectedFile', { filename: file.name }) : null}
            error={fileError}
            disabled={importing}
            inputRef={fileInputRef}
          />

          {!lockedCollection && (
            <div className="flex flex-col gap-2">
              <ImportTargetModeToggle
                label={t('database.import.collection.targetMode')}
                mode={targetMode}
                onModeChange={setTargetMode}
                existingLabel={t('database.import.collection.targetModeExisting')}
                newLabel={t('database.import.collection.targetModeNew')}
                disabled={importing}
              />
              {targetMode === 'existing' ? (
                <Select value={selectedCollection || undefined} onValueChange={setSelectedCollection} disabled={importing}>
                  <SelectTrigger className="w-full" aria-label={t('database.import.collection.target')}>
                    <SelectValue
                      placeholder={existingCollections.length === 0
                        ? t('database.import.collection.noCollections')
                        : t('database.import.collection.selectCollection')}
                    />
                  </SelectTrigger>
                  <SelectContent>
                    {existingCollections.map((name) => (
                      <SelectItem key={name} value={name}>{name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
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

          <ImportModeSelect
            label={t('database.import.collection.mode')}
            value={effectiveMode}
            onChange={setMode}
            disabled={importing || targetMode === 'new'}
            note={targetMode === 'new' ? t('database.import.collection.newCollectionAppendOnly') : undefined}
            options={[
              { value: ImportMode.Append, label: t('database.import.collection.mode.append') },
              { value: ImportMode.Overwrite, label: t('database.import.collection.mode.overwrite') },
              { value: ImportMode.Upsert, label: t('database.import.collection.mode.upsert') },
            ]}
          />

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

          <ImportSourceOptions
            showDelimiter={showDelimiter}
            showSheet={showSheet}
            delimiterLabel={t('database.import.collection.csvDelimiter')}
            delimiterOptions={CSV_DELIMITER_OPTIONS.map((option) => ({ value: option.value, label: t(option.labelKey) }))}
            delimiter={delimiter}
            onDelimiterChange={setDelimiter}
            sheetLabel={t('database.import.collection.excelSheet')}
            sheetPlaceholder={t('database.import.collection.excelSheetPlaceholder')}
            sheetOptions={sheetOptions}
            sheet={selectedSheet}
            onSheetChange={setSheet}
            disabled={importing}
          />

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

          {effectiveMode === ImportMode.Overwrite && (
            <ImportNotice tone="warning">{t('database.import.collection.overwriteWarning')}</ImportNotice>
          )}

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
    return <ImportNotice tone="error">{formatCollectionImportError(t, preview.ValidationError)}</ImportNotice>
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
    <ImportPreviewTable
      caption={t('database.import.collection.skipColumns')}
      columns={preview.Columns}
      rows={preview.Rows}
      skippable
      skipped={skippedColumns}
      onToggleColumn={onToggleColumn}
      truncated={preview.Truncated}
      truncatedText={t('database.import.collection.truncated')}
    />
  )
}

/** Renders the import outcome: success counts or a failure message. */
function CollectionImportResultView({ result }: { result: CollectionImportResult }) {
  const { t } = useI18n()

  if (!result.Status) {
    return (
      <ImportNotice tone="error">
        <span>{formatCollectionImportError(t, result.Detail)}</span>
        {result.Message && (
          <pre className="mt-2 whitespace-pre-wrap break-words font-mono text-xs leading-relaxed">{result.Message}</pre>
        )}
      </ImportNotice>
    )
  }

  const summary =
    result.SkippedCount > 0
      ? t('database.import.collection.successWithSkipped', { imported: result.ImportedCount, skipped: result.SkippedCount })
      : t('database.import.collection.successMessage', { imported: result.ImportedCount })

  return (
    <ImportNotice tone="success">
      <div className="flex flex-col gap-2">
        <span>{summary}</span>
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
    </ImportNotice>
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
