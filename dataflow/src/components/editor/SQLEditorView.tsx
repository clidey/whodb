import { useState, useEffect, useRef, useCallback } from "react";
import { Play, AlignLeft, CheckCircle, AlertCircle, FileText, Loader2, XCircle, CheckCircle2, GalleryVerticalEnd, Database, Network, BarChart3 } from "lucide-react";
import { format } from 'sql-formatter';
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/Button";
import { Tooltip, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { ChartCreateModal } from "@/components/analysis/chart-create";
import MonacoEditor from "./MonacoEditorWrapper";
import type { editor } from 'monaco-editor';
import type * as Monaco from 'monaco-editor';
import { useConnectionStore } from "@/stores/useConnectionStore";
import { useRawExecuteLazyQuery, useGetStorageUnitsLazyQuery, useGetColumnsBatchLazyQuery } from '@graphql';
import { getEditorLanguage, getUnsupportedRedisCommand, isReadOperation, resolveSchemaParam, supportsSchema } from "@/utils/database-features";
import { registerSQLCompletionProvider } from './sql-completion';
import type { SQLCompletionData, ColumnInfo } from './sql-completion';
import {
    splitRedisCommands,
    splitSQLStatements,
    splitMongoStatements,
    isLikelyMongoCommand,
    isStandaloneTransactionStatement,
} from '@/utils/sql-split';
import { useTabStore } from "@/stores/useTabStore";
import { useI18n } from "@/i18n/useI18n";

const IS_MAC = typeof navigator !== 'undefined' && /Mac/.test(navigator.platform);

interface SQLEditorViewProps {
    tabId: string;
    context?: {
        connectionId: string;
        databaseName?: string;
        schemaName?: string;
    } | null;
    initialSql?: string;
    onSqlChange?: (sql: string) => void;
    /** Called after a successful read query with the result columns, rows, and execution context. */
    onQueryResults?: (
        columns: string[],
        rows: Record<string, any>[],
        context: { database?: string; schema?: string },
    ) => void;
    /** When true, shows a "Create Chart" button in the toolbar that opens ChartCreateModal with the last successful read result. Defaults to true. */
    showChartCreate?: boolean;
}

interface StatementResult {
    columns: string[];
    rows: Record<string, string>[];
    info: string;
    isError?: boolean;
    sql: string;
    database?: string;
    schema?: string;
}

/** SQL editor with integrated database/schema selectors and query execution. */
export function SQLEditorView({ tabId, context, initialSql, onSqlChange, onQueryResults, showChartCreate = true }: SQLEditorViewProps) {
    const { t } = useI18n();
    const { connections } = useConnectionStore();
    const connectionType = connections.find((c) => c.id === context?.connectionId)?.type ?? 'POSTGRES';
    const [rawExecute] = useRawExecuteLazyQuery({ fetchPolicy: 'no-cache' });
    const [fetchStorageUnits] = useGetStorageUnitsLazyQuery({ fetchPolicy: 'no-cache' });
    const [fetchColumnsBatch] = useGetColumnsBatchLazyQuery({ fetchPolicy: 'no-cache' });
    const [activeResultTab, setActiveResultTab] = useState<'result' | 'message'>('result');
    const [query, setQuery] = useState(initialSql || "");
    const [isExecuting, setIsExecuting] = useState(false);
    const [queryResults, setQueryResults] = useState<StatementResult[] | null>(null);
    const [executionTime, setExecutionTime] = useState<number | null>(null);
    const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);
    const [monacoInstance, setMonacoInstance] = useState<typeof Monaco | null>(null);
    const { updateTab } = useTabStore();
    const { fetchDatabases, fetchSchemas } = useConnectionStore();
    const [databases, setDatabases] = useState<string[]>([]);
    const [schemas, setSchemas] = useState<string[]>([]);
    const [selectedDatabase, setSelectedDatabase] = useState(context?.databaseName ?? '');
    const [selectedSchema, setSelectedSchema] = useState(context?.schemaName ?? '');
    const [isChartModalOpen, setIsChartModalOpen] = useState(false);

    // Fetch databases on mount
    useEffect(() => {
        if (!context?.connectionId) return;
        fetchDatabases(context.connectionId).then(setDatabases).catch(console.error);
    }, [context?.connectionId, fetchDatabases]);

    // Fetch schemas when database changes (Postgres only), default to "public" or first available
    useEffect(() => {
        if (!context?.connectionId || !selectedDatabase || !supportsSchema(connectionType)) return;
        fetchSchemas(context.connectionId, selectedDatabase).then((result) => {
            setSchemas(result);
            if (!selectedSchema && result.length > 0) {
                const defaultSchema = result.includes('public') ? 'public' : result[0];
                setSelectedSchema(defaultSchema);
            }
        }).catch(console.error);
    }, [context?.connectionId, selectedDatabase, connectionType, fetchSchemas]);

    // Register SQL completion provider when schema metadata is available
    useEffect(() => {
        if (getEditorLanguage(connectionType) !== 'sql') return;
        if (!monacoInstance) return;

        const schemaParam = resolveSchemaParam(connectionType, selectedDatabase, selectedSchema);
        if (!schemaParam) return;

        let disposed = false;
        let disposable: Monaco.IDisposable | null = null;

        (async () => {
            const database = selectedDatabase || context?.databaseName;
            const { data: storageData } = await fetchStorageUnits({
                variables: { schema: schemaParam },
                context: { database },
            });
            if (disposed || !storageData?.StorageUnit) return;

            const tableNames = storageData.StorageUnit.map((u) => u.Name);
            if (tableNames.length === 0) return;

            const { data: columnsData } = await fetchColumnsBatch({
                variables: { schema: schemaParam, storageUnits: tableNames },
                context: { database },
            });
            if (disposed) return;

            const columns = new Map<string, ColumnInfo[]>();
            if (columnsData?.ColumnsBatch) {
                for (const batch of columnsData.ColumnsBatch) {
                    columns.set(
                        batch.StorageUnit,
                        batch.Columns.map((c) => ({
                            name: c.Name,
                            type: c.Type,
                            isPrimary: c.IsPrimary,
                            isForeignKey: c.IsForeignKey,
                        })),
                    );
                }
            }

            const completionData: SQLCompletionData = { tables: tableNames, columns };
            disposable = registerSQLCompletionProvider(monacoInstance, completionData);
        })();

        return () => {
            disposed = true;
            disposable?.dispose();
        };
    }, [connectionType, selectedDatabase, selectedSchema, monacoInstance, fetchStorageUnits, fetchColumnsBatch]);

    const handleDatabaseChange = (db: string) => {
        setSelectedDatabase(db);
        if (supportsSchema(connectionType)) {
            setSelectedSchema('');
            updateTab(tabId, { databaseName: db, schemaName: undefined });
        } else {
            updateTab(tabId, { databaseName: db });
        }
    };

    const handleSchemaChange = (schema: string) => {
        setSelectedSchema(schema);
        updateTab(tabId, { schemaName: schema });
    };

    const handleRun = async () => {
        const upperType = connectionType.toUpperCase();
        const statements = upperType === 'REDIS'
            ? splitRedisCommands(query)
            : upperType === 'MONGODB'
                ? splitMongoStatements(query)
                : splitSQLStatements(query);
        if (statements.length === 0) return;

        setIsExecuting(true);
        setQueryResults(null);
        const executionDatabase = selectedDatabase || context?.databaseName;
        const executionSchema = selectedSchema || context?.schemaName;
        const startTime = Date.now();
        const results: StatementResult[] = [];

        for (let idx = 0; idx < statements.length; idx++) {
            const sql = statements[idx];

            // Block unsupported Redis commands with a clear message
            if (upperType === 'REDIS') {
                const unsupportedKey = getUnsupportedRedisCommand(sql);
                if (unsupportedKey) {
                    results.push({ columns: [], rows: [], info: t(unsupportedKey as import('@/i18n/messages').MessageKey), isError: true, sql, database: executionDatabase, schema: executionSchema });
                    break;
                }
            }

            // Reject non-command MongoDB statements early with a localized, targeted error
            if (upperType === 'MONGODB' && !isLikelyMongoCommand(sql)) {
                results.push({
                    columns: [],
                    rows: [],
                    info: t('mongodb.editor.unsupportedStatement', { statement: sql }),
                    isError: true,
                    sql,
                    database: executionDatabase,
                    schema: executionSchema,
                });
                continue;
            }

            // Block standalone transaction statements with a warning
            if (upperType !== 'REDIS' && upperType !== 'MONGODB' && isStandaloneTransactionStatement(sql)) {
                results.push({ columns: [], rows: [], info: t('sql.editor.transactionWarning'), isError: true, sql, database: executionDatabase, schema: executionSchema });
                continue;
            }

            try {
                const { data, error } = await rawExecute({
                    variables: { query: sql },
                    context: { database: selectedDatabase || context?.databaseName },
                });

                if (error) {
                    results.push({
                        columns: [],
                        rows: [],
                        info: error.message,
                        isError: true,
                        sql,
                        database: executionDatabase,
                        schema: executionSchema,
                    });
                    break;
                }

                if (data?.RawExecute) {
                    const raw = data.RawExecute;
                    const columns = raw.Columns.map((c) => c.Name);

                    if (isReadOperation(connectionType, sql) || raw.Rows.length > 0) {
                        const rows = raw.Rows.map((row) =>
                            Object.fromEntries(columns.map((col, i) => [col, row[i]]))
                        );
                        results.push({ columns, rows, info: t('sql.editor.rows', { count: raw.TotalCount }), sql, database: executionDatabase, schema: executionSchema });
                    } else {
                        results.push({ columns: [], rows: [], info: t('sql.editor.actionExecuted'), sql, database: executionDatabase, schema: executionSchema });
                    }
                }
            } catch (err: any) {
                results.push({
                    columns: [],
                    rows: [],
                    info: err.message,
                    isError: true,
                    sql,
                    database: executionDatabase,
                    schema: executionSchema,
                });
            }
        }

        const endTime = Date.now();
        setExecutionTime((endTime - startTime) / 1000);

        setQueryResults(results);
        setActiveResultTab(results.some((r) => r.isError) ? 'message' : 'result');

        // Notify parent of the last successful read result
        const lastRead = [...results].reverse().find((r) => !r.isError && r.rows.length > 0);
        if (lastRead) {
            onQueryResults?.(lastRead.columns, lastRead.rows, {
                database: lastRead.database ?? executionDatabase,
                schema: lastRead.schema ?? executionSchema,
            });
        }

        // Refresh sidebar tree when a write operation succeeded (DDL/DML may change schema objects)
        const hasSuccessfulWrite = results.some((r) => !r.isError && !isReadOperation(connectionType, r.sql));
        if (hasSuccessfulWrite) {
            useConnectionStore.getState().triggerSidebarRefresh();
        }

        setIsExecuting(false);
    };

    const handleFormat = () => {
        if (!query.trim()) return;
        try {
            const formatted = format(query);
            setQuery(formatted);
            onSqlChange?.(formatted);
            if (editorRef.current) {
                editorRef.current.setValue(formatted);
            }
        } catch {
            // sql-formatter can't parse the query — leave it as-is
        }
    };

    // Keep refs in sync so Monaco keybindings always call the latest handlers
    const handleRunRef = useRef(handleRun);
    handleRunRef.current = handleRun;
    const handleFormatRef = useRef(handleFormat);
    handleFormatRef.current = handleFormat;
    const isExecutingRef = useRef(isExecuting);
    isExecutingRef.current = isExecuting;

    const [resultsHeight, setResultsHeight] = useState(400);
    const isResizing = useRef(false);
    const containerRef = useRef<HTMLDivElement>(null);

    const handleResizeMouseDown = useCallback((e: React.MouseEvent) => {
        e.preventDefault();
        isResizing.current = true;
        document.body.style.cursor = 'row-resize';
        document.body.style.userSelect = 'none';
    }, []);

    useEffect(() => {
        const handleMouseMove = (e: MouseEvent) => {
            if (!isResizing.current || !containerRef.current) return;
            const containerRect = containerRef.current.getBoundingClientRect();
            const newHeight = containerRect.bottom - e.clientY;
            const minHeight = 40;
            const maxHeight = containerRect.height - 100;
            setResultsHeight(Math.min(maxHeight, Math.max(minHeight, newHeight)));
        };

        const handleMouseUp = () => {
            if (!isResizing.current) return;
            isResizing.current = false;
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        };

        document.addEventListener('mousemove', handleMouseMove);
        document.addEventListener('mouseup', handleMouseUp);
        return () => {
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', handleMouseUp);
        };
    }, []);

    const lastReadResult = queryResults?.slice().reverse().find((r) => !r.isError && r.rows.length > 0) ?? null;
    const canCreateChart = showChartCreate && !!lastReadResult && !!context?.connectionId;

    return (
        <div
            className="flex h-full flex-col bg-background overflow-hidden"
            ref={containerRef}
            data-testid="sql.editor.view"
            data-qa-module="sql"
            data-qa-object="editor"
            data-qa-state={isExecuting ? 'executing' : queryResults?.some((r) => r.isError) ? 'error' : queryResults ? 'completed' : 'ready'}
            data-qa-loading={isExecuting ? 'true' : 'false'}
            data-qa-connection-id={context?.connectionId}
            data-qa-database={selectedDatabase || context?.databaseName}
            data-qa-schema={selectedSchema || context?.schemaName}
        >
            {/* Toolbar */}
            <div
                className="flex h-12 items-center justify-between border-b px-2 shrink-0"
                data-testid="sql.editor.toolbar"
                data-qa-module="sql"
                data-qa-object="editor-toolbar"
                data-qa-state={isExecuting ? 'executing' : 'ready'}
            >
                {/* Left: Action Buttons */}
                <div className="flex items-center">
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                variant="ghost"
                                size="icon"
                                onClick={handleRun}
                                disabled={isExecuting}
                                data-testid="sql.editor.run-button"
                                data-qa-module="sql"
                                data-qa-object="query"
                                data-qa-action="execute"
                                data-qa-state={isExecuting ? 'executing' : 'ready'}
                                data-qa-disabled-reason={isExecuting ? 'executing' : undefined}
                                data-qa-risk="query_execution"
                            >
                                {isExecuting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent>{t('sql.actions.run')} ({IS_MAC ? '⌘↩' : 'Ctrl+Enter'})</TooltipContent>
                    </Tooltip>
                    {getEditorLanguage(connectionType) === 'sql' && (
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={handleFormat}
                                    disabled={!query.trim()}
                                    data-testid="sql.editor.format-button"
                                    data-qa-module="sql"
                                    data-qa-object="query"
                                    data-qa-action="format"
                                    data-qa-disabled-reason={!query.trim() ? 'empty_query' : undefined}
                                >
                                    <AlignLeft className="h-4 w-4" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>{t('sql.actions.format')} ({IS_MAC ? '⇧⌥F' : 'Shift+Alt+F'})</TooltipContent>
                        </Tooltip>
                    )}
                    {showChartCreate && (
                        <>
                            <Separator orientation="vertical" className="mx-1 data-[orientation=vertical]:h-4" />
                            <Tooltip>
                                <TooltipTrigger asChild>
                                    <span>
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            onClick={() => setIsChartModalOpen(true)}
                                            disabled={!canCreateChart}
                                            data-testid="sql.editor.create-chart-button"
                                            data-qa-module="sql"
                                            data-qa-object="query-result"
                                            data-qa-action="create-chart"
                                            data-qa-disabled-reason={!canCreateChart ? 'not_ready' : undefined}
                                        >
                                            <BarChart3 className="h-4 w-4" />
                                        </Button>
                                    </span>
                                </TooltipTrigger>
                                <TooltipContent>{t('analysis.chart.create')}</TooltipContent>
                            </Tooltip>
                        </>
                    )}
                </div>

                {/* Right: Database/Schema Selectors */}
                <div className="flex items-center gap-2">
                    {/* Database Selector */}
                    <Select
                        value={selectedDatabase || undefined}
                        onValueChange={handleDatabaseChange}
                        disabled={databases.length === 0}
                    >
                        <SelectTrigger
                            className="gap-1.5 border-0 bg-transparent shadow-none"
                            data-testid="sql.editor.database-select"
                            data-qa-module="sql"
                            data-qa-object="execution-context"
                            data-qa-field="database"
                            data-qa-state={databases.length === 0 ? 'empty' : 'ready'}
                            data-qa-disabled-reason={databases.length === 0 ? 'no_databases' : undefined}
                        >
                            <Database className="h-4 w-4 text-muted-foreground" />
                            <SelectValue placeholder={t('sql.editor.selectDatabase')} />
                        </SelectTrigger>
                        <SelectContent
                            align="end"
                            data-testid="sql.editor.database-select-content"
                            data-qa-module="sql"
                            data-qa-object="execution-context-options"
                            data-qa-field="database"
                        >
                            {databases.map((db) => (
                                <SelectItem
                                    key={db}
                                    value={db}
                                    data-testid="sql.editor.database-option"
                                    data-qa-module="sql"
                                    data-qa-object="database"
                                    data-qa-action="select"
                                    data-qa-resource-type="database"
                                    data-qa-resource-id={db}
                                >
                                    {db}
                                </SelectItem>
                            ))}
                            {databases.length === 0 && (
                                <div className="px-3 py-2 text-sm text-muted-foreground">{t('sql.editor.noDatabases')}</div>
                            )}
                        </SelectContent>
                    </Select>

                    {/* Schema Selector (Postgres only) */}
                    {supportsSchema(connectionType) && (
                        <Select
                            value={selectedSchema || undefined}
                            onValueChange={handleSchemaChange}
                            disabled={!selectedDatabase || schemas.length === 0}
                        >
                            <SelectTrigger
                                className="gap-1.5 border-0 bg-transparent shadow-none"
                                data-testid="sql.editor.schema-select"
                                data-qa-module="sql"
                                data-qa-object="execution-context"
                                data-qa-field="schema"
                                data-qa-state={schemas.length === 0 ? 'empty' : 'ready'}
                                data-qa-disabled-reason={!selectedDatabase ? 'database_required' : schemas.length === 0 ? 'no_schemas' : undefined}
                            >
                                <Network className="h-4 w-4 text-muted-foreground" />
                                <SelectValue placeholder={t('sql.editor.selectSchema')} />
                            </SelectTrigger>
                            <SelectContent
                                align="end"
                                data-testid="sql.editor.schema-select-content"
                                data-qa-module="sql"
                                data-qa-object="execution-context-options"
                                data-qa-field="schema"
                            >
                                {schemas.map((schema) => (
                                    <SelectItem
                                        key={schema}
                                        value={schema}
                                        data-testid="sql.editor.schema-option"
                                        data-qa-module="sql"
                                        data-qa-object="schema"
                                        data-qa-action="select"
                                        data-qa-resource-type="schema"
                                        data-qa-resource-id={schema}
                                    >
                                        {schema}
                                    </SelectItem>
                                ))}
                                {schemas.length === 0 && (
                                    <div className="px-3 py-2 text-sm text-muted-foreground">{t('sql.editor.noSchemas')}</div>
                                )}
                            </SelectContent>
                        </Select>
                    )}
                </div>
            </div>

            {/* Main Content Area (Split View) */}
            <div className="flex-1 flex flex-col overflow-hidden relative">
                {/* Editor Area */}
                <div
                    className="flex-1 overflow-hidden"
                    style={{ marginBottom: isResizing ? 0 : 0 }}
                    data-testid="sql.editor.input-region"
                    data-qa-module="sql"
                    data-qa-object="query-input"
                    data-qa-field="query"
                >
                    <MonacoEditor
                        height="100%"
                        language={getEditorLanguage(connectionType)}
                        value={query}
                        onChange={(value: string | undefined) => {
                            const v = value || '';
                            setQuery(v);
                            onSqlChange?.(v);
                        }}
                        theme="vs-light"
                        options={{
                            minimap: { enabled: false },
                            fontSize: 14,
                            lineNumbers: 'on',
                            roundedSelection: false,
                            scrollBeyondLastLine: false,
                            readOnly: false,
                            automaticLayout: true,
                            suggestOnTriggerCharacters: true,
                            quickSuggestions: true,
                            wordBasedSuggestions: 'off',
                        }}
                        onMount={(editorInstance: editor.IStandaloneCodeEditor, monacoInstance: typeof Monaco) => {
                            editorRef.current = editorInstance;
                            setMonacoInstance(monacoInstance);

                            editorInstance.addAction({
                                id: 'run-query',
                                label: 'Run Query',
                                keybindings: [monacoInstance.KeyMod.CtrlCmd | monacoInstance.KeyCode.Enter],
                                run: () => { if (!isExecutingRef.current) handleRunRef.current(); },
                            });

                            editorInstance.addAction({
                                id: 'format-sql',
                                label: 'Format SQL',
                                keybindings: [monacoInstance.KeyMod.Shift | monacoInstance.KeyMod.Alt | monacoInstance.KeyCode.KeyF],
                                run: () => { handleFormatRef.current(); },
                            });
                        }}
                    />
                </div>

                {/* Resize Handle */}
                <div
                    className="w-full h-1 cursor-row-resize hover:bg-primary/30 active:bg-primary/50 z-10"
                    onMouseDown={handleResizeMouseDown}
                    data-testid="sql.editor.result-resize-handle"
                    data-qa-module="sql"
                    data-qa-object="result-pane"
                    data-qa-action="resize"
                />

                {/* Results Pane */}
                <div
                    className="border-t flex flex-col bg-background transition-[height] ease-out duration-75"
                    style={{ height: resultsHeight, maxHeight: '80%' }}
                    data-testid="sql.editor.result-pane"
                    data-qa-module="sql"
                    data-qa-object="query-result"
                    data-qa-state={isExecuting ? 'loading' : queryResults?.some((r) => r.isError) ? 'error' : queryResults ? 'success' : 'empty'}
                    data-qa-loading={isExecuting ? 'true' : 'false'}
                >
                    {/* Result Tabs */}
                    <div className="flex items-center border-b bg-muted/10 h-10">
                        <Button
                            onClick={() => setActiveResultTab('result')}
                            variant="ghost"
                            size="sm"
                            data-testid="sql.editor.result-tab"
                            data-qa-module="sql"
                            data-qa-object="result-pane-tab"
                            data-qa-action="show-result"
                            data-qa-state={activeResultTab === 'result' ? 'active' : 'inactive'}
                            className={cn(
                                "h-full w-25 rounded-none border-b-2 px-4 py-2 text-sm font-normal",
                                activeResultTab === 'result' ? "border-primary text-primary bg-background" : "border-transparent text-muted-foreground hover:text-foreground"
                            )}
                        >
                            <FileText className="h-4 w-4" />
                            {t('sql.editor.results')}
                        </Button>
                        <Button
                            onClick={() => setActiveResultTab('message')}
                            variant="ghost"
                            size="sm"
                            data-testid="sql.editor.message-tab"
                            data-qa-module="sql"
                            data-qa-object="result-pane-tab"
                            data-qa-action="show-message"
                            data-qa-state={activeResultTab === 'message' ? 'active' : 'inactive'}
                            className={cn(
                                "h-full w-25 rounded-none border-b-2 px-4 py-2 text-sm font-normal",
                                activeResultTab === 'message' ? "border-primary text-primary bg-background" : "border-transparent text-muted-foreground hover:text-foreground"
                            )}
                        >
                            {queryResults?.some(r => r.isError) ? (
                                <AlertCircle className="h-4 w-4 text-destructive" />
                            ) : (
                                <CheckCircle className="h-4 w-4" />
                            )}
                            {t('sql.editor.message')}
                        </Button>
                    </div>

                    {/* Result Content */}
                    <div className="flex-1 overflow-auto bg-background/50 p-0">
                        {activeResultTab === 'result' && (
                            <div className="w-full text-sm">
                                {queryResults && queryResults.length > 0 ? (
                                    <div className="divide-y divide-border">
                                        {queryResults.map((result, resultIndex) => (
                                            <div
                                                key={resultIndex}
                                                className="flex flex-col"
                                                data-testid="sql.editor.result-set"
                                                data-qa-module="sql"
                                                data-qa-object="statement-result"
                                                data-qa-state={result.isError ? 'error' : 'success'}
                                                data-qa-error-code={result.isError ? 'query_execution_failed' : undefined}
                                                data-qa-result-index={resultIndex}
                                                data-qa-row-count={result.rows.length}
                                            >
                                                {/* Result Header */}
                                                <div className="flex flex-col border-b border-border/50">
                                                    <div className={cn(
                                                        "px-4 py-2.5 flex items-center justify-between",
                                                        result.isError ? 'bg-destructive/5' : 'bg-muted/30'
                                                    )}
                                                        data-testid="sql.editor.result-set-header"
                                                        data-qa-module="sql"
                                                        data-qa-object="statement-result"
                                                        data-qa-state={result.isError ? 'error' : 'success'}
                                                    >
                                                        <div className="flex items-center gap-3">
                                                            <div className={cn(
                                                                "flex items-center justify-center w-5 h-5 rounded-full",
                                                                result.isError ? 'bg-destructive/10 text-destructive' : 'bg-success/10 text-success'
                                                            )}>
                                                                {result.isError ? (
                                                                    <XCircle className="h-3.5 w-3.5" />
                                                                ) : (
                                                                    <CheckCircle2 className="h-3.5 w-3.5" />
                                                                )}
                                                            </div>
                                                            <span className="font-medium text-sm text-foreground">
                                                                {t('sql.editor.resultNumber', { index: resultIndex + 1 })}
                                                            </span>
                                                            <span className={cn(
                                                                "text-xs px-2 py-0.5 rounded-full font-medium",
                                                                result.isError
                                                                    ? 'bg-destructive/10 text-destructive'
                                                                    : 'bg-success/10 text-success border border-success/20'
                                                            )}>
                                                                {result.isError ? t('common.alert.error') : (result.info || t('sql.editor.success'))}
                                                            </span>
                                                        </div>
                                                        <div className="flex items-center gap-4 text-xs text-muted-foreground">
                                                            {!result.isError && (
                                                                <span className="flex items-center gap-1.5">
                                                                    <GalleryVerticalEnd className="h-3.5 w-3.5" />
                                                                    {t('sql.editor.rows', { count: result.rows.length })}
                                                                </span>
                                                            )}
                                                        </div>
                                                    </div>

                                                    {/* SQL Statement Display */}
                                                    {result.sql && queryResults.length > 1 && (
                                                        <div className="px-4 py-2 bg-muted/30 border-t border-b border-border/50">
                                                            <code className={cn(
                                                                "text-sm font-mono block whitespace-pre-wrap break-all pl-2 border-l-2",
                                                                result.isError
                                                                    ? 'border-destructive text-destructive bg-destructive/5'
                                                                    : 'border-primary/30 text-muted-foreground'
                                                            )}>
                                                                {result.sql}
                                                            </code>
                                                        </div>
                                                    )}
                                                </div>

                                                {/* Result Body */}
                                                {result.isError ? (
                                                    <div className="px-4 py-3">
                                                        <div className="flex items-start gap-2 text-destructive text-sm">
                                                            <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
                                                            <pre className="font-mono whitespace-pre-wrap break-all text-xs">{result.info}</pre>
                                                        </div>
                                                    </div>
                                                ) : (
                                                    <div className="overflow-x-auto">
                                                        <table
                                                            className="w-full border-collapse text-left"
                                                            data-testid="sql.editor.result-table"
                                                            data-qa-module="sql"
                                                            data-qa-object="result-table"
                                                            data-qa-state={result.rows.length > 0 ? 'ready' : 'empty'}
                                                            data-qa-row-count={result.rows.length}
                                                        >
                                                            <thead className="bg-muted sticky top-0 z-10">
                                                                <tr>
                                                                    <th className="border-b border-r px-4 py-2 font-medium text-muted-foreground w-16 text-center bg-muted">#</th>
                                                                    {result.columns.map((col, i) => (
                                                                        <th key={i} className="border-b border-r px-4 py-2 font-medium text-muted-foreground bg-muted whitespace-nowrap">
                                                                            {col}
                                                                        </th>
                                                                    ))}
                                                                </tr>
                                                            </thead>
                                                            <tbody>
                                                                {result.rows.length > 0 ? (
                                                                    result.rows.map((row, i) => (
                                                                        <tr
                                                                            key={i}
                                                                            className="hover:bg-muted/10"
                                                                            data-testid="sql.editor.result-row"
                                                                            data-qa-module="sql"
                                                                            data-qa-object="result-row"
                                                                            data-qa-state="ready"
                                                                            data-qa-row-index={i}
                                                                        >
                                                                            <td className="border-b border-r px-4 py-1.5 text-muted-foreground text-center bg-muted/5 font-mono text-xs">
                                                                                {i + 1}
                                                                            </td>
                                                                            {result.columns.map((col, j) => (
                                                                                <td
                                                                                    key={j}
                                                                                    className="border-b border-r px-4 py-1.5 whitespace-nowrap max-w-[300px] truncate"
                                                                                    data-testid="sql.editor.result-cell"
                                                                                    data-qa-module="sql"
                                                                                    data-qa-object="result-cell"
                                                                                    data-qa-field={col}
                                                                                    data-qa-row-index={i}
                                                                                >
                                                                                    {typeof row[col] === 'object' ? JSON.stringify(row[col]) : String(row[col] ?? '')}
                                                                                </td>
                                                                            ))}
                                                                        </tr>
                                                                    ))
                                                                ) : (
                                                                    <tr>
                                                                        <td
                                                                            colSpan={result.columns.length + 1}
                                                                            className="px-4 py-8 text-center text-muted-foreground"
                                                                            data-testid="sql.editor.result-empty"
                                                                            data-qa-module="sql"
                                                                            data-qa-object="result-table"
                                                                            data-qa-state="empty"
                                                                        >
                                                                            {t('sql.editor.noRowsReturned')}
                                                                        </td>
                                                                    </tr>
                                                                )}
                                                            </tbody>
                                                        </table>
                                                    </div>
                                                )}
                                            </div>
                                        ))}
                                    </div>
                                ) : isExecuting ? (
                                    <div className="flex items-center justify-center h-full min-h-[200px] text-muted-foreground">
                                        <Loader2 className="h-6 w-6 animate-spin" />
                                    </div>
                                ) : (
                                    <div className="flex items-center justify-center h-full min-h-[200px] text-muted-foreground flex-col gap-2">
                                        <Play className="h-8 w-8 opacity-20" />
                                        <span>{t('sql.editor.runToSeeResults')}</span>
                                    </div>
                                )}
                            </div>
                        )}
                        {activeResultTab === 'message' && (
                            <div className="p-4 text-sm font-mono space-y-2">
                                {queryResults && queryResults.length > 0 ? (
                                    <>
                                        {queryResults.map((result, idx) => (
                                            <div key={idx} className={cn(
                                                "flex flex-col gap-1 rounded px-3 py-2",
                                                result.isError ? 'bg-destructive/5' : 'bg-muted/30'
                                            )}
                                                data-testid="sql.editor.message-item"
                                                data-qa-module="sql"
                                                data-qa-object="statement-message"
                                                data-qa-state={result.isError ? 'error' : 'success'}
                                                data-qa-error-code={result.isError ? 'query_execution_failed' : undefined}
                                                data-qa-result-index={idx}
                                            >
                                                <div className="flex items-center gap-2">
                                                    {result.isError ? (
                                                        <XCircle className="h-3.5 w-3.5 text-destructive shrink-0" />
                                                    ) : (
                                                        <CheckCircle2 className="h-3.5 w-3.5 text-success shrink-0" />
                                                    )}
                                                    <span className="font-medium text-xs">
                                                        {t('sql.editor.resultNumber', { index: idx + 1 })}
                                                    </span>
                                                    {queryResults.length > 1 && (
                                                        <code className="text-xs text-muted-foreground truncate max-w-[400px]">
                                                            {result.sql}
                                                        </code>
                                                    )}
                                                </div>
                                                <div className={cn(
                                                    "pl-6 text-xs",
                                                    result.isError ? 'text-destructive' : 'text-muted-foreground'
                                                )}>
                                                    {result.info}
                                                </div>
                                            </div>
                                        ))}
                                        <div className="flex items-center gap-2 pt-2 border-t text-xs text-muted-foreground">
                                            <span>{t('sql.editor.executionSummary', {
                                                total: queryResults.length,
                                                success: queryResults.filter(r => !r.isError).length,
                                                failed: queryResults.filter(r => r.isError).length,
                                                time: executionTime?.toFixed(3) ?? '0',
                                            })}</span>
                                        </div>
                                    </>
                                ) : isExecuting ? (
                                    <div className="flex items-center justify-center min-h-[160px] text-muted-foreground">
                                        <Loader2 className="h-6 w-6 animate-spin" />
                                    </div>
                                ) : (
                                    <div className="text-muted-foreground">{t('sql.editor.noQueryExecuted')}</div>
                                )}
                            </div>
                        )}
                    </div>
                </div>
            </div>

            {showChartCreate && lastReadResult && context?.connectionId && (
                <ChartCreateModal
                    open={isChartModalOpen}
                    onOpenChange={setIsChartModalOpen}
                    initialData={{
                        connectionId: context.connectionId,
                        databaseName: lastReadResult.database ?? context.databaseName ?? '',
                        schemaName: lastReadResult.schema ?? context.schemaName,
                        query: lastReadResult.sql,
                        columns: lastReadResult.columns,
                        rows: lastReadResult.rows,
                    }}
                />
            )}
        </div>
    );
}
