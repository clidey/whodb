import { createContext, use, useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent, type ReactNode } from 'react'
import { AlertTriangle, CheckCircle, Database, FileCode, FileSpreadsheet, Loader2, Upload } from 'lucide-react'
import { Dialog, DialogClose, DialogContent, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Textarea } from '@/components/ui/Textarea'
import { ModalForm, useModalForm } from '@/components/ui/ModalForm'
import { useI18n } from '@/i18n/useI18n'
import { cn } from '@/lib/utils'
import { useConnectionStore } from '@/stores/useConnectionStore'
import { buildImportSqlInput, hasSqlScriptSource, isAcceptedSqlFile, type SqlScriptSource } from './sql-import-utils'
import { useImportSqlMutation } from '@graphql'

type ImportMethod = 'sql' | 'tableFile'
type SourceKind = 'file' | 'text'

interface DatabaseImportContextValue {
  method: ImportMethod
  sourceKind: SourceKind
  source: SqlScriptSource | null
  targetDatabase: string
  databaseOptions: string[]
  result: ImportResultState | null
  fileInputRef: React.RefObject<HTMLInputElement | null>
  setMethod: (method: ImportMethod) => void
  setSourceKind: (kind: SourceKind) => void
  setTargetDatabase: (database: string) => void
  handleFileChange: (event: ChangeEvent<HTMLInputElement>) => void
  handleScriptChange: (script: string) => void
  convertFileToText: () => void
  executeImport: () => Promise<void>
}

interface ImportResultState {
  status: boolean
  detail?: string | null
  message?: string | null
}

const DatabaseImportContext = createContext<DatabaseImportContextValue | null>(null)

function useDatabaseImport(): DatabaseImportContextValue {
  const context = use(DatabaseImportContext)
  if (!context) throw new Error('useDatabaseImport must be used within DatabaseImportProvider')
  return context
}

interface DatabaseImportProviderProps {
  connectionId: string
  databaseName: string
  schema?: string | null
  tableName?: string | null
  onSuccess?: (context: { databaseName: string; schema?: string | null; tableName?: string | null }) => void
  children: ReactNode
}

/** Provides state and execution behavior for the shared Database Import modal. */
function DatabaseImportProvider({
  connectionId,
  databaseName,
  schema,
  tableName,
  onSuccess,
  children,
}: DatabaseImportProviderProps) {
  const { t } = useI18n()
  const [method, setMethod] = useState<ImportMethod>('sql')
  const [sourceKind, setSourceKindState] = useState<SourceKind>('file')
  const [source, setSource] = useState<SqlScriptSource | null>(null)
  const [targetDatabase, setTargetDatabase] = useState(databaseName)
  const [databaseOptions, setDatabaseOptions] = useState<string[]>([databaseName])
  const [result, setResult] = useState<ImportResultState | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const fetchDatabases = useConnectionStore((state) => state.fetchDatabases)
  const { actions } = useModalForm()
  const [importSql] = useImportSqlMutation()

  useEffect(() => {
    setTargetDatabase(databaseName)
    setDatabaseOptions([databaseName])
  }, [databaseName])

  useEffect(() => {
    let cancelled = false

    void fetchDatabases(connectionId)
      .then((databases) => {
        if (cancelled) return
        const nextOptions = databases.includes(databaseName)
          ? databases
          : [databaseName, ...databases]
        setDatabaseOptions([...new Set(nextOptions)])
      })
      .catch(() => {
        if (!cancelled) {
          actions.setAlert({
            type: 'error',
            title: t('database.import.loadDatabasesFailed'),
            message: t('database.import.loadDatabasesFailedMessage'),
          })
        }
      })

    return () => { cancelled = true }
  }, [actions, connectionId, databaseName, fetchDatabases, t])

  const setSourceKind = useCallback((kind: SourceKind) => {
    setSourceKindState(kind)
    setSource(null)
    setResult(null)
    actions.closeAlert()
    if (kind === 'file' && fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }, [actions])

  const handleFileChange = useCallback(async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    actions.closeAlert()
    setResult(null)

    if (!isAcceptedSqlFile(file)) {
      setSource(null)
      actions.setAlert({
        type: 'error',
        title: t('database.import.invalidFileTitle'),
        message: t('database.import.invalidFileMessage'),
      })
      event.target.value = ''
      return
    }

    try {
      const preview = await file.text()
      setSourceKindState('file')
      setSource({ kind: 'file', file, filename: file.name, preview })
    } catch {
      setSource(null)
      actions.setAlert({
        type: 'error',
        title: t('database.import.readFileFailedTitle'),
        message: t('database.import.readFileFailedMessage'),
      })
      event.target.value = ''
    }
  }, [actions, t])

  const handleScriptChange = useCallback((script: string) => {
    setSourceKindState('text')
    setSource({ kind: 'text', script })
    setResult(null)
    actions.closeAlert()
  }, [actions])

  const convertFileToText = useCallback(() => {
    if (!source || source.kind !== 'file') return
    setSourceKindState('text')
    setSource({ kind: 'text', script: source.preview })
    setResult(null)
    actions.closeAlert()
    if (fileInputRef.current) fileInputRef.current.value = ''
  }, [actions, source])

  const executeImport = useCallback(async () => {
    if (!source || !hasSqlScriptSource(source)) return

    actions.setSubmitting(true)
    actions.closeAlert()
    setResult(null)

    try {
      const { data, errors } = await importSql({
        variables: { input: buildImportSqlInput(source) },
        context: { database: targetDatabase },
      })

      if (errors?.length) {
        actions.setAlert({
          type: 'error',
          title: t('database.import.failedTitle'),
          message: errors[0].message,
        })
        return
      }

      const importResult = data?.ImportSQL
      if (!importResult) {
        actions.setAlert({
          type: 'error',
          title: t('database.import.failedTitle'),
          message: t('database.import.failedMessage'),
        })
        return
      }

      setResult({ status: importResult.Status, detail: importResult.Detail, message: importResult.Message })
      if (importResult.Status) {
        onSuccess?.(targetDatabase === databaseName
          ? { databaseName: targetDatabase, schema, tableName }
          : { databaseName: targetDatabase, schema: null, tableName: null })
      }
    } catch (error) {
      actions.setAlert({
        type: 'error',
        title: t('database.import.failedTitle'),
        message: error instanceof Error ? error.message : t('database.import.failedMessage'),
      })
    } finally {
      actions.setSubmitting(false)
    }
  }, [actions, databaseName, importSql, onSuccess, schema, source, t, tableName, targetDatabase])

  const contextValue = useMemo<DatabaseImportContextValue>(() => ({
    method,
    sourceKind,
    source,
    targetDatabase,
    databaseOptions,
    result,
    fileInputRef,
    setMethod,
    setSourceKind,
    setTargetDatabase,
    handleFileChange,
    handleScriptChange,
    convertFileToText,
    executeImport,
  }), [
    convertFileToText,
    databaseOptions,
    executeImport,
    handleFileChange,
    handleScriptChange,
    method,
    result,
    source,
    sourceKind,
    targetDatabase,
  ])

  return (
    <DatabaseImportContext value={contextValue}>
      {children}
    </DatabaseImportContext>
  )
}

function DatabaseImportFields() {
  return (
    <div className="flex max-h-[68vh] flex-col gap-4 overflow-y-auto pr-1">
      <ImportMethodSelector />
      <TargetDatabaseSelect />
      <SqlSourceSelector />
      <ImportResultFeedback />
    </div>
  )
}

function ImportMethodSelector() {
  const { t } = useI18n()
  const { method, setMethod } = useDatabaseImport()
  const { state } = useModalForm()
  const disabled = state.isSubmitting

  return (
    <div className="flex flex-col gap-2">
      <span className="text-sm font-medium text-foreground">{t('database.import.method')}</span>
      <div className="grid gap-2 sm:grid-cols-2">
        <button
          type="button"
          onClick={() => setMethod('sql')}
          disabled={disabled}
          aria-pressed={method === 'sql'}
          className={cn(
            'flex min-h-14 items-center gap-3 rounded-md border px-3 py-2 text-left transition-colors',
            method === 'sql'
              ? 'border-primary bg-primary/5'
              : 'border-input bg-background hover:bg-muted/30',
          )}
        >
          <FileCode className="h-4 w-4 text-primary" />
          <span className="flex flex-col gap-0.5">
            <span className="text-sm font-medium">{t('database.import.method.sqlFile')}</span>
            <span className="text-xs text-muted-foreground">{t('database.import.method.sqlFileDescription')}</span>
          </span>
        </button>
        <button
          type="button"
          disabled
          aria-disabled="true"
          className="flex min-h-14 items-center gap-3 rounded-md border border-input bg-muted/20 px-3 py-2 text-left opacity-60"
        >
          <FileSpreadsheet className="h-4 w-4 text-muted-foreground" />
          <span className="flex flex-col gap-0.5">
            <span className="text-sm font-medium">{t('database.import.method.tableFile')}</span>
            <span className="text-xs text-muted-foreground">{t('database.import.method.comingSoon')}</span>
          </span>
        </button>
      </div>
    </div>
  )
}

function TargetDatabaseSelect() {
  const { t } = useI18n()
  const { targetDatabase, setTargetDatabase, databaseOptions } = useDatabaseImport()
  const { state } = useModalForm()

  return (
    <div className="flex flex-col gap-2">
      <label className="text-sm font-medium text-foreground" htmlFor="database-import-target">
        {t('database.import.targetDatabase')}
      </label>
      <select
        id="database-import-target"
        value={targetDatabase}
        onChange={(event) => setTargetDatabase(event.target.value)}
        disabled={state.isSubmitting}
        className="h-9 w-full rounded-md border border-input bg-background px-3 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:opacity-50"
      >
        {databaseOptions.map((database) => (
          <option key={database} value={database}>{database}</option>
        ))}
      </select>
    </div>
  )
}

function SqlSourceSelector() {
  const { t } = useI18n()
  const { sourceKind, setSourceKind, source } = useDatabaseImport()
  const { state } = useModalForm()

  return (
    <div className="flex flex-col gap-3">
      <span className="text-sm font-medium text-foreground">{t('database.import.source')}</span>
      <div className="inline-flex w-fit rounded-md border border-input p-1">
        <button
          type="button"
          onClick={() => setSourceKind('file')}
          disabled={state.isSubmitting}
          aria-pressed={sourceKind === 'file'}
          className={cn(
            'rounded-sm px-3 py-1.5 text-sm transition-colors',
            sourceKind === 'file' ? 'bg-highlight-background text-foreground' : 'text-muted-foreground hover:text-foreground',
          )}
        >
          {t('database.import.source.file')}
        </button>
        <button
          type="button"
          onClick={() => setSourceKind('text')}
          disabled={state.isSubmitting}
          aria-pressed={sourceKind === 'text'}
          className={cn(
            'rounded-sm px-3 py-1.5 text-sm transition-colors',
            sourceKind === 'text' ? 'bg-highlight-background text-foreground' : 'text-muted-foreground hover:text-foreground',
          )}
        >
          {t('database.import.source.text')}
        </button>
      </div>
      {sourceKind === 'file' ? <SqlFileSource /> : <SqlTextSource />}
      {!hasSqlScriptSource(source) && (
        <p className="text-xs text-muted-foreground">{t('database.import.emptySourceHint')}</p>
      )}
    </div>
  )
}

function SqlFileSource() {
  const { t } = useI18n()
  const { source, fileInputRef, handleFileChange, convertFileToText } = useDatabaseImport()
  const { state } = useModalForm()
  const fileSource = source?.kind === 'file' ? source : null

  return (
    <div className="flex flex-col gap-3">
      <Input
        ref={fileInputRef}
        type="file"
        accept=".sql"
        onChange={handleFileChange}
        disabled={state.isSubmitting}
        aria-label={t('database.import.uploadSqlFile')}
      />
      {fileSource && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between gap-3">
            <span className="min-w-0 truncate text-sm text-muted-foreground">
              {t('database.import.selectedFile', { filename: fileSource.filename })}
            </span>
            <Button type="button" variant="outline" size="sm" onClick={convertFileToText} disabled={state.isSubmitting}>
              {t('database.import.editAsText')}
            </Button>
          </div>
          <Textarea
            value={fileSource.preview}
            readOnly
            aria-readonly="true"
            aria-label={t('database.import.sqlFilePreview')}
            className="h-52 resize-none font-mono text-xs"
          />
        </div>
      )}
    </div>
  )
}

function SqlTextSource() {
  const { t } = useI18n()
  const { source, handleScriptChange } = useDatabaseImport()
  const { state } = useModalForm()
  const script = source?.kind === 'text' ? source.script : ''

  return (
    <Textarea
      value={script}
      onChange={(event) => handleScriptChange(event.target.value)}
      disabled={state.isSubmitting}
      aria-label={t('database.import.sqlTextInput')}
      placeholder={t('database.import.pastePlaceholder')}
      className="h-60 resize-none font-mono text-xs"
    />
  )
}

function ImportResultFeedback() {
  const { t } = useI18n()
  const { result } = useDatabaseImport()
  const { state } = useModalForm()

  if (state.isSubmitting) {
    return (
      <div role="status" className="flex items-center gap-2 rounded-md border border-primary/20 bg-primary/5 p-3 text-sm text-primary">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>{t('database.import.running')}</span>
      </div>
    )
  }

  if (!result) {
    return (
      <div className="flex items-start gap-2 rounded-md border border-warning/20 bg-warning/5 p-3 text-sm text-warning">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        <span>{t('database.import.reviewWarning')}</span>
      </div>
    )
  }

  if (result.status) {
    return (
      <div role="status" className="flex items-center gap-2 rounded-md border border-success/20 bg-success/5 p-3 text-sm text-success">
        <CheckCircle className="h-4 w-4" />
        <span>{t('database.import.successMessage')}</span>
      </div>
    )
  }

  return (
    <div role="alert" className="flex items-start gap-2 rounded-md border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
      <div className="min-w-0">
        <span>{formatImportFailure(t, result.detail)}</span>
        {result.detail === 'import.error.sql_failed' && result.message && (
          <pre className="mt-2 whitespace-pre-wrap break-words font-mono text-xs leading-relaxed">{result.message}</pre>
        )}
      </div>
    </div>
  )
}

function formatImportFailure(t: ReturnType<typeof useI18n>['t'], detail?: string | null): string {
  switch (detail) {
    case 'import.error.sql_source_both':
      return t('database.import.error.sqlSourceBoth')
    case 'import.error.sql_source_missing':
      return t('database.import.error.sqlSourceMissing')
    case 'import.error.sql_too_large':
      return t('database.import.error.sqlTooLarge')
    case 'import.error.sql_file_failed':
      return t('database.import.error.sqlFileFailed')
    case 'import.error.sql_multi_statement_unsupported':
      return t('database.import.error.sqlMultiStatementUnsupported')
    case 'import.error.sql_failed':
      return t('database.import.error.sqlFailed')
    default:
      return t('database.import.failedMessage')
  }
}

function DatabaseImportFooter() {
  const { t } = useI18n()
  const { source, result, executeImport } = useDatabaseImport()
  const { state } = useModalForm()
  const canRun = hasSqlScriptSource(source)

  if (result?.status) {
    return (
      <DialogFooter>
        <DialogClose asChild>
          <Button variant="outline">{t('common.actions.close')}</Button>
        </DialogClose>
      </DialogFooter>
    )
  }

  return (
    <DialogFooter>
      <DialogClose asChild>
        <Button type="button" variant="outline" disabled={state.isSubmitting}>
          {t('common.actions.cancel')}
        </Button>
      </DialogClose>
      <Button type="button" onClick={executeImport} disabled={!canRun || state.isSubmitting} className="gap-2">
        {state.isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
        {state.isSubmitting ? t('database.import.running') : t('database.import.runImport')}
      </Button>
    </DialogFooter>
  )
}

export interface DatabaseImportModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  connectionId: string
  databaseName: string
  schema?: string | null
  tableName?: string | null
  onSuccess?: (context: { databaseName: string; schema?: string | null; tableName?: string | null }) => void
}

/** Shared Database Import modal for supported SQL import entry points. */
export function DatabaseImportModal({
  open,
  onOpenChange,
  connectionId,
  databaseName,
  schema,
  tableName,
  onSuccess,
}: DatabaseImportModalProps) {
  const { t } = useI18n()
  const description = tableName
    ? schema ? `${databaseName}.${schema}.${tableName}` : `${databaseName}.${tableName}`
    : schema ? `${databaseName}.${schema}` : databaseName

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <ModalForm.Provider meta={{ title: t('database.import.title'), description, icon: Database }}>
          <DatabaseImportProvider
            connectionId={connectionId}
            databaseName={databaseName}
            schema={schema}
            tableName={tableName}
            onSuccess={onSuccess}
          >
            <ModalForm.Header />
            <DatabaseImportFields />
            <ModalForm.Alert />
            <DatabaseImportFooter />
          </DatabaseImportProvider>
        </ModalForm.Provider>
      </DialogContent>
    </Dialog>
  )
}
