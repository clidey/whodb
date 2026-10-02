/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {useLazyQuery, useMutation} from "@apollo/client/react";
import {
    Alert,
    AlertDescription,
    AlertTitle,
    Button,
    Card,
    cn,
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
    EmptyState,
    Input,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    toast
} from "@clidey/ux";
import {ChatHistorySidebar} from "./chat-history-sidebar";
import type {RowsResult} from '@graphql';
import {ExecuteConfirmedSqlDocument, GenerateChatTitleDocument, GetDatabaseQuerySuggestionsDocument} from '@graphql';
import {
    CheckCircleIcon,
    CodeBracketIcon,
    CommandLineIcon,
    DocumentDuplicateIcon,
    EllipsisHorizontalIcon,
    SparklesIcon,
    TableCellsIcon
} from "../../components/heroicons";
import classNames from "classnames";
import type { FC, KeyboardEventHandler} from "react";
import {memo, useCallback, useEffect, useMemo, useRef, useState} from "react";
import ReactMarkdown from 'react-markdown';
import logoImage from "../../../public/images/logo.svg";
import {AIProvider, useAI} from "../../components/ai";
import {Tip} from "../../components/tip";
import {CodeEditor} from "../../components/editor";
import {ErrorState} from "../../components/error-state";
import {Loading, Spinner} from "../../components/loading";
import {WhoDBChatIcon} from "../../components/whodb-chat-icon";
import {InternalPage} from "../../components/page";
import {StorageUnitTable} from "../../components/table";
import {copyToClipboard} from "../../services/clipboard";
import {MessageCopyAction} from "../../components/message-copy-action";
import {extensions, featureFlags} from "../../config/features";
import {InternalRoutes} from "../../config/routes";
import {HoudiniActions, type IChatMessage} from "../../store/chat";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {ScratchpadActions} from "../../store/scratchpad";
import {getComponent} from "../../config/component-registry";
import {useSourceContract} from "../../hooks/useSourceContract";
import {useNavigate} from "react-router-dom";
import {useChatExamples} from "./examples";
import {useTranslation} from '@/hooks/use-translation';
import {addAuthHeader} from "../../utils/auth-headers";
import {withBasePath} from "../../utils/base-path";
import {matchesShortcut, SHORTCUTS} from "../../utils/shortcuts";
import {useContainerWidth} from "../../hooks/use-container-width";
import {buildSourceScopeRef} from "../../utils/source-refs";
import {ph} from "../../utils/privacy";
import {
    countBucket,
    frontendAnalyticsErrorCode,
    textLengthBucket,
    trackFrontendIntent,
    trackScreenViewed,
} from "../../config/frontend-analytics";

// Chart components from the component registry
const LineChart = getComponent('line-chart');
const PieChart = getComponent('pie-chart');

function unsupportedToolName(text: string): string | null {
    if (!text.trimStart().startsWith('{')) return null;
    try {
        const payload: unknown = JSON.parse(text);
        if (payload && typeof payload === 'object' && 'toolName' in payload && typeof payload.toolName === 'string') {
            return payload.toolName;
        }
    } catch {
        return null;
    }
    return null;
}

function parseSQLFailure(text: string): {sql: string; error: string} | null {
    try {
        const value: unknown = JSON.parse(text);
        if (value && typeof value === 'object' && 'sql' in value && 'error' in value && typeof value.sql === 'string' && typeof value.error === 'string') {
            return {sql: value.sql, error: value.error};
        }
    } catch {
        return null;
    }
    return null;
}

function failedSQLLine(sql: string, error: string): number | null {
    const lines = sql.split('\n');
    const lineNumber = /\bline\s+(\d+)\b/i.exec(error);
    if (lineNumber) {
        const index = Number(lineNumber[1]) - 1;
        if (index >= 0 && index < lines.length) return index;
    }
    const objectName = /(?:no such table:|relation|unknown column|no such column:)\s*["'`]?([\w.]+)/i.exec(error)?.[1];
    if (objectName) {
        const index = lines.findIndex(line => line.toLowerCase().includes(objectName.toLowerCase()));
        if (index >= 0) return index;
    }
    return null;
}

const markdownComponents = {
    p: ({_node, ...props}: any) => <p className="mb-2 last:mb-0" {...props} />,
    strong: ({_node, ...props}: any) => <strong className="font-semibold" {...props} />,
    ul: ({_node, ...props}: any) => <ul className="list-disc list-inside mb-2 space-y-1" {...props} />,
    ol: ({_node, ...props}: any) => <ol className="list-decimal list-inside mb-2 space-y-1" {...props} />,
    li: ({_node, ...props}: any) => <li className="ml-2" {...props} />,
    h1: ({_node, ...props}: any) => <h1 className="text-xl font-bold mb-2 mt-4 first:mt-0" {...props} />,
    h2: ({_node, ...props}: any) => <h2 className="text-lg font-semibold mb-2 mt-3 first:mt-0" {...props} />,
    h3: ({_node, ...props}: any) => <h3 className="text-md font-semibold mb-1 mt-2 first:mt-0" {...props} />,
    a: ({_node, ...props}: any) => <a className="text-primary underline underline-offset-2" {...props} />,
    code: ({_node, children, ...props}: any) => {
        const isInline = !String(props.className ?? '').includes('language-');
        return isInline
            ? <code className="rounded bg-muted px-1 py-0.5 text-[0.9em] text-foreground" {...props}>{children}</code>
            : <CodeBlock>{String(children)}</CodeBlock>;
    },
    blockquote: ({_node, ...props}: any) => <blockquote className="my-2 border-l-4 border-border pl-4 italic" {...props} />,
};

const CodeBlock: FC<{ children: string }> = ({ children }) => {
    const [copied, setCopied] = useState(false);

    const handleCopy = useCallback(() => {
        void copyToClipboard(children).then(success => {
            if (success) {
                setCopied(true);
                setTimeout(() =>{  setCopied(false); }, 2000);
            }
        });
    }, [children]);

    return (
        <div className="relative group/code-block my-2">
            <code className="block overflow-x-auto rounded bg-muted p-2 pr-16 text-[0.9em] text-foreground">
                {children}
            </code>
            <button
                onClick={handleCopy}
                className="absolute top-1.5 right-1.5 flex items-center gap-1 rounded bg-secondary px-1.5 py-0.5 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
                {copied
                    ? <><CheckCircleIcon className="w-3.5 h-3.5 text-green-500" /> <span className="text-green-500">Copied!</span></>
                    : <><DocumentDuplicateIcon className="w-3.5 h-3.5" /> <span>Copy</span></>
                }
            </button>
        </div>
    );
};

type TableData = RowsResult | null | undefined;

const TablePreview: FC<{ type: string, data: TableData, text: string, containerWidth?: number }> = memo(({ type, data, text, containerWidth }) => {
    const { t } = useTranslation('pages/chat');
    const dispatch = useAppDispatch();
    const [showSQL, setShowSQL] = useState(false);
    const [showScratchpadDialog, setShowScratchpadDialog] = useState(false);
    const [selectedPage, setSelectedPage] = useState<string>("new");
    const [newPageName, setNewPageName] = useState<string>("");
    const [dropdownOpen, setDropdownOpen] = useState(false);
    const navigate = useNavigate();
    const currentType = useAppSelector(state => state.auth.current?.Type);
    const { pages, activePageId } = useAppSelector(state => state.scratchpad);
    const { supportsScratchpad } = useSourceContract(currentType);

    const handleCodeToggle = useCallback(() => {
        setShowSQL(status => !status);
    }, []);

    const handleExport = useCallback(() => {
        if (!data) return;
        const escapeCell = (value: string) => {
            const safeValue = /^[=+@\-\t\r]/.test(value) ? `'${value}` : value;
            return `"${safeValue.replaceAll('"', '""')}"`;
        };
        const csv = [data.Columns.map(column => column.Name), ...data.Rows].map(row => row.map(escapeCell).join(',')).join('\n');
        const url = URL.createObjectURL(new Blob([csv], {type: 'text/csv;charset=utf-8'}));
        const link = document.createElement('a');
        link.href = url;
        link.download = 'chat-result.csv';
        document.body.appendChild(link);
        link.click();
        link.remove();
        window.setTimeout(() => { URL.revokeObjectURL(url); }, 0);
    }, [data]);

    // Create page options excluding current page
    const pageOptions = useMemo(() => {
        return [
            ...pages.map(page => ({ value: page.id, label: page.name })),
            { value: "new", label: t('createNewPage') }
        ];
    }, [pages, activePageId, t]);

    const handleMoveToScratchpad = useCallback(() => {
        if (!supportsScratchpad) {
            toast.error(t('scratchpadNotSupported'));
            return;
        }
        // Initialize scratchpad if needed
        if (pages.length === 0) {
            dispatch(ScratchpadActions.ensurePagesHaveCells());
        }
        setShowScratchpadDialog(true);
    }, [dispatch, pages.length, supportsScratchpad, t]);

    const handleScratchpadConfirm = useCallback(() => {
        if (selectedPage === "new") {
            // Create new page with the query
            const pageName = newPageName.trim() || `Page ${pages.length + 1}`;
            dispatch(ScratchpadActions.addPage({ name: pageName, initialQuery: text }));
            // Navigate to scratchpad - the new page will be created with the query
            void navigate(InternalRoutes.RawExecute.path, {
                state: {
                    targetPage: "new"
                }
            });
        } else {
            // Add to existing page and set it as active
            dispatch(ScratchpadActions.addCellToPageAndActivate({
                pageId: selectedPage,
                initialQuery: text
            }));
            // Navigate to scratchpad and highlight the target page
            void navigate(InternalRoutes.RawExecute.path, {
                state: {
                    targetPage: selectedPage
                }
            });
        }
        setShowScratchpadDialog(false);
        setSelectedPage("new");
        setNewPageName("");
        toast.success(t('queryMoved'));
    }, [navigate, text, selectedPage, newPageName, pages.length, dispatch, t]);

    const previewResult = useMemo(() => {
        if (data == null || data.Rows.length === 0) {
            return t('noDataReturned');
        }
        return type.toUpperCase().split(":")?.[1];
    }, [data, type, t]);

    const unhandledTool = data == null ? unsupportedToolName(text) : null;

    const canMoveToScratchpad = useMemo(() => {
        return supportsScratchpad && type.startsWith("sql:");
    }, [supportsScratchpad, type]);

    const keyedRows = useMemo(() => {
        const seen = new Map<string, number>();
        return (data?.Rows ?? []).map(row => {
            const signature = JSON.stringify(row);
            const occurrence = seen.get(signature) ?? 0;
            seen.set(signature, occurrence + 1);
            return {row, key: `${signature}:${occurrence}`};
        });
    }, [data]);

    return <div className="ce-chat-result-card flex gap-2 w-[calc(100%-50px)] max-w-full min-w-0 group/table-preview">
        <div className={cn("transition-all shrink-0 pt-1", {
            "opacity-0 group-hover/table-preview:opacity-100 focus-within:opacity-100": !dropdownOpen,
            "opacity-100": dropdownOpen,
        }, __WHODB_EDITION__ === 'ce' && 'ce-chat-result-menu')}>
            <DropdownMenu open={dropdownOpen} onOpenChange={setDropdownOpen}>
                <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="sm" data-testid="icon-button" aria-label={t('actions')}>
                        <EllipsisHorizontalIcon className="w-5 h-5" aria-hidden="true" />
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={handleCodeToggle} data-testid="toggle-view-option">
                        {showSQL ? (
                            <>
                                <TableCellsIcon className="w-4 h-4 mr-2" />
                                {t('showTable')}
                            </>
                        ) : (
                            <>
                                <CodeBracketIcon className="w-4 h-4 mr-2" />
                                {t('showCode')}
                            </>
                        )}
                    </DropdownMenuItem>
                    {canMoveToScratchpad && (
                        <DropdownMenuItem onClick={handleMoveToScratchpad} data-testid="move-to-scratchpad-option">
                            <CommandLineIcon className="w-4 h-4 mr-2" />
                            {t('moveToScratchpad')}
                        </DropdownMenuItem>
                    )}
                </DropdownMenuContent>
            </DropdownMenu>
        </div>
        <div className="ce-chat-result-body flex flex-col gap-lg overflow-hidden break-all leading-6 shrink-0 w-full max-w-full min-w-0">
            {data != null && <div className="ce-chat-result-heading"><strong>{t('result')}</strong><span>{t('rowCount', {count: data.Rows.length})}</span><span className="ce-chat-result-complete">{t('complete')}</span></div>}
            {
                __WHODB_EDITION__ === 'ce' && data != null
                ? <>
                    {showSQL
                        ? <pre className="ce-chat-result-sql">{text}</pre>
                        : data.Rows.length > 0
                            ? <div className="ce-chat-result-grid"><table><thead><tr>{data.Columns.map(column => <th key={column.Name}>{column.Name}</th>)}</tr></thead><tbody>{keyedRows.map(({row, key}) => <tr key={key}>{data.Columns.map((column, index) => <td key={column.Name}>{row[index]}</td>)}</tr>)}</tbody></table></div>
                            : <p className="ce-chat-result-empty">{t('noDataReturned')}</p>}
                    <div className="ce-chat-result-actions"><button type="button" onClick={handleCodeToggle}><WhoDBChatIcon name="code" />{showSQL ? t('showTable') : t('sql')}</button><button type="button" onClick={handleExport}><WhoDBChatIcon name="download" />{t('export')}</button></div>
                </>
                : __WHODB_EDITION__ === 'ce' && unhandledTool
                ? <div className="ce-chat-result-empty"><strong>{t('toolResponseNotExecuted')}</strong><span>{unhandledTool}</span></div>
                : __WHODB_EDITION__ === 'ce' && data == null
                ? <div className="ce-chat-result-empty"><strong>{t('resultUnavailable')}</strong>{text && <pre className="ce-chat-result-sql">{text}</pre>}</div>
                : showSQL
                ? <div className={cn("h-[300px] w-full", ph.mask)}>
                    <CodeEditor value={text} language="sql" />
                </div>
                :  (data != null && data.Rows.length > 0) || type === "sql:get"
                    ? <div className="w-full">
                        <StorageUnitTable
                            key={containerWidth}
                            columns={data?.Columns?.map(c => c.Name) ?? []}
                            columnTypes={data?.Columns?.map(c => c.Type) ?? []}
                            rows={data?.Rows ?? []}
                            disableEdit={true}
                            limitContextMenu={true}
                            databaseType={currentType}
                            rawQuery={text}
                            height={__WHODB_EDITION__ === 'ce' ? Math.max(76, (data?.Rows?.length ?? 0) * 32 + 40) : 200}
                            enforceMinHeight={__WHODB_EDITION__ !== 'ce'}
                            totalCount={data?.Rows?.length ?? 0}
                        />
                    </div>
                    : (type.startsWith("sql:") && (type === "sql:insert" || type === "sql:update" || type === "sql:delete" || type === "sql:create" || type === "sql:alter" || type === "sql:drop"))
                    ? <Alert title={t('actionExecuted')} className="w-fit">
                        <CheckCircleIcon className="w-4 h-4" />
                        <AlertTitle>{t('actionExecuted')}</AlertTitle>
                        <AlertDescription>
                            {previewResult}
                        </AlertDescription>
                    </Alert>
                    : null
            }
        </div>
        
        <Dialog open={showScratchpadDialog} onOpenChange={setShowScratchpadDialog}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>{t('moveToScratchpad')}</DialogTitle>
                    <DialogDescription>
                        {t('dialogDescription')}
                    </DialogDescription>
                </DialogHeader>
                <div className="py-4 space-y-4">
                    <div>
                        <label className="text-sm font-medium mb-2 block">{t('selectPageLabel')}</label>
                        <Select value={selectedPage} onValueChange={setSelectedPage}>
                            <SelectTrigger className="w-full">
                                <SelectValue placeholder={t('selectPagePlaceholder')} />
                            </SelectTrigger>
                            <SelectContent>
                                {pageOptions.map((option) => (
                                    <SelectItem key={option.value} value={option.value}>
                                        {option.label}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                    {selectedPage === "new" && (
                        <div>
                            <label className="text-sm font-medium mb-2 block">{t('newPageLabel')}</label>
                            <Input
                                value={newPageName}
                                onChange={(e) =>{  setNewPageName(e.target.value); }}
                                placeholder={t('newPagePlaceholder')}
                            />
                        </div>
                    )}
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={() => {
                        setShowScratchpadDialog(false);
                        setSelectedPage("new");
                        setNewPageName("");
                    }}>
                        {t('cancel')}
                    </Button>
                    <Button onClick={handleScratchpadConfirm}>
                        {t('moveToScratchpad')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    </div>
});

/** Renders CE chat in the shared full-height conversation layout. */
export const ChatPage: FC = () => {
    const { t } = useTranslation('pages/chat');
    const [query, setQuery] = useState("");
    const { sessions, activeSessionId } = useAppSelector(state => state.houdini);
    const activeSession = useMemo(() => {
        return sessions.length > 0 && activeSessionId
            ? sessions.find(s => s.id === activeSessionId)
            : undefined;
    }, [sessions, activeSessionId]);
    const chats = useMemo(() => {
        return activeSession?.messages ?? [];
    }, [activeSession]);
    const displayChats = useMemo(() => {
        const visible: IChatMessage[] = [];
        for (let start = 0; start < chats.length;) {
            if (!chats[start].isUserInput) {
                visible.push(chats[start]);
                start++;
                continue;
            }
            let end = start + 1;
            while (end < chats.length && !chats[end].isUserInput) end++;
            const turn = chats.slice(start, end);
            const invalidTool = turn.find(chat => !chat.isUserInput && unsupportedToolName(chat.Text));
            const hasDatabaseResult = turn.some(chat => chat.Result != null || chat.RequiresConfirmation);
            if (invalidTool && !hasDatabaseResult) {
                visible.push(chats[start], {
                    id: invalidTool.id,
                    Type: 'error',
                    Text: t('unsupportedToolRequest'),
                    RequiresConfirmation: false,
                });
            } else if (invalidTool) {
                visible.push(...turn.filter(chat => !unsupportedToolName(chat.Text)));
            } else {
                visible.push(...turn);
            }
            start = end;
        }
        return visible;
    }, [chats, t]);
    const autoScrollEnabled = activeSession?.autoScrollEnabled ?? true;
    const [executeConfirmedSql] = useMutation(ExecuteConfirmedSqlDocument);
    const [generateChatTitleMutation] = useMutation(GenerateChatTitleDocument);
    const scrollContainerRef = useRef<HTMLDivElement>(null);
    const containerWidth = useContainerWidth(scrollContainerRef);
    const schemaFromState = useAppSelector(state => state.database.schema);
    const authProfile = useAppSelector(state => state.auth.current);
    const authProfileType = authProfile?.Type;
    const authProfileDatabase = authProfile?.Database;
    const { item, supportsScripts, supportsScratchpad } = useSourceContract(authProfileType);
    const scratchpadPageId = useAppSelector(state => state.scratchpad.activePageId);
    const [executingConfirmedId, setExecutingConfirmedId] = useState<number | null>(null);
    const [copiedSqlId, setCopiedSqlId] = useState<number | null>(null);
    const messageIdCounter = useRef(0);
    const sourceScopeRef = useMemo(() => buildSourceScopeRef(item, authProfile, schemaFromState), [authProfileDatabase, item, schemaFromState]);
    const [currentSearchIndex, setCurrentSearchIndex] = useState<number>();
    const draftAbandonedRef = useRef(false);
    const chatRequest = useRef<AbortController | null>(null);
    const streamingMessage = useRef<number | null>(null);
    const draftAnalyticsPropsRef = useRef<Record<string, unknown>>({});

    const dispatch = useAppDispatch();
    const navigate = useNavigate();

    // Generate unique message IDs to prevent collisions
    const getUniqueMessageId = useCallback(() => {
        messageIdCounter.current += 1;
        return Date.now() * 1000 + messageIdCounter.current;
    }, []);

    const aiState = useAI();
    const { modelType, currentModel, modelAvailable, models } = aiState;

    const chatExamples = useChatExamples();

    const [loading, setLoading] = useState(false);
    const [progressSteps, setProgressSteps] = useState<Array<{id: string; step: string; status: string}>>([]);

    // Database-specific suggestions
    const [getDatabaseSuggestions, { loading: suggestionsLoading }] = useLazyQuery(GetDatabaseQuerySuggestionsDocument, {
        fetchPolicy: 'network-only',
        errorPolicy: 'all',
    });
    const [databaseSuggestions, setDatabaseSuggestions] = useState<Array<{ description: string; category: string }>>([]);
    const [useDatabaseSuggestions, setUseDatabaseSuggestions] = useState(false);
    const hasFetchedSuggestionsRef = useRef(false);

    useEffect(() => {
        trackScreenViewed('chat', {
            has_model: currentModel != null,
            has_provider: modelType?.id != null,
            model_type: modelType?.modelType ?? 'unknown',
            message_count_bucket: countBucket(chats.length),
        });
    }, []);

    useEffect(() => {
        draftAbandonedRef.current = query.trim().length > 0;
        draftAnalyticsPropsRef.current = {
            input_length_bucket: textLengthBucket(query),
            message_count_bucket: countBucket(chats.length),
            model_type: modelType?.modelType ?? 'unknown',
            has_model: currentModel != null,
        };
    }, [chats.length, currentModel, modelType?.modelType, query]);

    useEffect(() => {
        return () => {
            if (draftAbandonedRef.current) {
                trackFrontendIntent('chat.message_draft_abandoned', draftAnalyticsPropsRef.current);
            }
        };
    }, []);

    // Store random indices in a ref so they remain stable across re-renders
    const exampleIndicesRef = useRef<number[] | null>(null);

    // Initialize random indices once
    useEffect(() => {
        if (exampleIndicesRef.current === null && chatExamples.length > 0) {
            const indices: number[] = [];
            const available = [...Array(chatExamples.length).keys()];
            const count = Math.min(3, chatExamples.length);
            for (let i = 0; i < count; i++) {
                const randomIndex = Math.floor(Math.random() * available.length);
                indices.push(available[randomIndex]);
                available.splice(randomIndex, 1);
            }
            exampleIndicesRef.current = indices;
        }
    }, [chatExamples.length]);

    // Map category to icon
    const getCategoryIcon = useCallback((category: string) => {
        switch (category) {
            case 'SELECT':
                return <WhoDBChatIcon name="table" />;
            case 'AGGREGATE':
                return <WhoDBChatIcon name="banknotes" />;
            default:
                return <WhoDBChatIcon name="code" />;
        }
    }, []);

    // Use database suggestions if available, otherwise fall back to generic examples
    const examples = useMemo(() => {
        if (useDatabaseSuggestions && databaseSuggestions.length > 0) {
            return databaseSuggestions.map(s => ({
                icon: getCategoryIcon(s.category),
                description: s.description,
            }));
        }

        // Fallback to generic examples
        if (exampleIndicesRef.current === null) {
            return chatExamples.slice(0, 3);
        }
        return exampleIndicesRef.current.map(i => chatExamples[i]);
    }, [useDatabaseSuggestions, databaseSuggestions, chatExamples, getCategoryIcon]);

    const handleSubmitQuery = useCallback(async (queryOverride?: string, retryMessageId?: number) => {
        const sanitizedQuery = (queryOverride ?? query).trim();
        if (modelType == null || sanitizedQuery.length === 0) {
            return;
        }
        draftAbandonedRef.current = false;
        trackFrontendIntent('chat.message_submitted', {
            input_length_bucket: textLengthBucket(sanitizedQuery),
            message_count_bucket: countBucket(chats.length),
            model_type: modelType.modelType,
            has_model: currentModel != null,
            has_provider: modelType.id != null,
            has_source: sourceScopeRef != null,
        });

        // Check if we should try to generate a title:
        // - Session still has default name (matches "Chat X" pattern)
        // This will keep trying on each message until we get a meaningful title
        const hasDefaultName = activeSession?.name?.match(/^Chat \d+$/);
        const shouldTryTitle = hasDefaultName;

        setLoading(true);
        setProgressSteps([]);
        if (retryMessageId != null) dispatch(HoudiniActions.removeChatMessage(retryMessageId));
        else dispatch(HoudiniActions.addChatMessage({ Type: "message", Text: sanitizedQuery, isUserInput: true, RequiresConfirmation: false }));
        setQuery("");

        // Add a placeholder for streaming text
        const streamingMessageId = getUniqueMessageId();
        streamingMessage.current = streamingMessageId;
        dispatch(HoudiniActions.addChatMessage({
            Type: "message",
            Text: "",
            isStreaming: true,
            id: streamingMessageId,
            RequiresConfirmation: false
        }));

        if (autoScrollEnabled) {
            setTimeout(() => {
                if (scrollContainerRef.current != null) {
                    scrollContainerRef.current.scroll({
                        top: scrollContainerRef.current.scrollHeight,
                        behavior: "smooth",
                    });
                }
            }, 250);
        }

        const controller = new AbortController();
        chatRequest.current = controller;
        try {
            const response = await fetch(withBasePath('/api/ai-chat/stream'), {
                method: 'POST',
                credentials: 'include',
                signal: controller.signal,
                headers: addAuthHeader({
                    'Content-Type': 'application/json',
                }),
                body: JSON.stringify({
                    ref: sourceScopeRef,
                    modelType: modelType.modelType,
                    providerId: modelType.id ?? '',
                    token: modelType.token ?? '',
                    model: currentModel ?? '',
                    input: {
                        Query: sanitizedQuery,
                        PreviousConversation: displayChats.filter(chat => chat.Type !== 'activity' && chat.id !== retryMessageId).map(chat =>
                            `${chat.isUserInput ? "<User>" : "<System>"}${chat.Text}${chat.isUserInput ? "</User>" : "</System>"}`
                        ).join("\n"),
                    },
                }),
            });

            if (!response.ok) {
                trackFrontendIntent('chat.message_failed', {
                    input_length_bucket: textLengthBucket(sanitizedQuery),
                    message_count_bucket: countBucket(chats.length),
                    model_type: modelType.modelType,
                    error_code: 'connection_failed',
                });
                setLoading(false);
                return;
            }

            if (!response.body) {
                trackFrontendIntent('chat.message_failed', {
                    input_length_bucket: textLengthBucket(sanitizedQuery),
                    message_count_bucket: countBucket(chats.length),
                    model_type: modelType.modelType,
                    error_code: 'unknown',
                });
                setLoading(false);
                return;
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let buffer = '';
            let streamingText = '';
            let currentEventType = '';
            const addedSqlMessages = new Set<string>(); // Track added SQL to avoid duplicates
            let streamDone = false;
            let rejectedTool = false;
            let firstSqlMessageId: number | null = null;
            let progressTrace: Array<{id: string; step: string; status: string}> = [];

            while (true) {
                const { done, value } = await reader.read();
                if (done) {
                    break;
                }

                // Buffer partial SSE lines across reads so frames split across
                // chunk boundaries are reassembled before parsing.
                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop() ?? '';

                for (const line of lines) {
                    if (line.startsWith('event: ')) {
                        currentEventType = line.slice(7).trim();
                    } else if (line.startsWith('data: ')) {
                        const data = line.slice(6);
                        if (!data.trim()) continue;

                        try {
                            const parsed = JSON.parse(data);

                            if (currentEventType === 'progress') {
                                if (typeof parsed.step === 'string' && typeof parsed.status === 'string') {
                                    const index = progressTrace.findIndex(item => item.step === parsed.step && item.status !== 'completed');
                                    progressTrace = index >= 0
                                        ? progressTrace.map((item, position) => position === index ? {...item, status: parsed.status} : item)
                                        : [...progressTrace, {id: `${parsed.step}:${progressTrace.length + 1}`, step: parsed.step, status: parsed.status}];
                                    setProgressSteps(progressTrace);
                                }
                            } else if (currentEventType === 'chunk') {
                                const text = parsed.text ?? '';
                                const chunkType = parsed.type ?? '';

                                // Only update message text if it's longer (ignore SQL/error chunks)
                                if (chunkType !== 'sql' && chunkType !== 'error' && text && text.length > streamingText.length) {
                                    streamingText = text;
                                    dispatch(HoudiniActions.updateChatMessage({
                                        id: streamingMessageId,
                                        Text: streamingText,
                                    }));
                                }

                                // Auto-scroll
                                if (autoScrollEnabled && scrollContainerRef.current != null) {
                                    scrollContainerRef.current.scroll({
                                        top: scrollContainerRef.current.scrollHeight,
                                        behavior: "smooth",
                                    });
                                }
                            } else if (currentEventType === 'message') {
                                // Handle complete messages (SQL responses and errors after streaming)
                                if ((parsed.Type === 'message' || parsed.Type === 'text') && typeof parsed.Text === 'string') {
                                    streamingText = parsed.Text;
                                    dispatch(HoudiniActions.updateChatMessage({id: streamingMessageId, Text: streamingText}));
                                    continue;
                                }
                                if (parsed.Type?.startsWith("sql") || parsed.Type === "error" || parsed.Type === 'provider:error' || parsed.Type === 'scope:error') {
                                    if (parsed.Type?.startsWith("sql") && unsupportedToolName(parsed.Text ?? '')) {
                                        rejectedTool = true;
                                        continue;
                                    }
                                    // Create a unique key for this message to avoid duplicates
                                    const messageKey = `${parsed.Type}:${parsed.Text}`;

                                    // Only add if we haven't seen this message before
                                    if (!addedSqlMessages.has(messageKey)) {
                                        addedSqlMessages.add(messageKey);

                                        const messageId = getUniqueMessageId();
                                        if (parsed.Type.startsWith('sql:') && firstSqlMessageId == null) firstSqlMessageId = messageId;
                                        dispatch(HoudiniActions.addChatMessage({
                                            Type: parsed.Type,
                                            Text: parsed.Text,
                                            Result: parsed.Result,
                                            RequiresConfirmation: parsed.RequiresConfirmation ?? false,
                                            id: messageId,
                                        }));

                                        if (autoScrollEnabled) {
                                            setTimeout(() => {
                                                if (scrollContainerRef.current != null) {
                                                    scrollContainerRef.current.scroll({
                                                        top: scrollContainerRef.current.scrollHeight,
                                                        behavior: "smooth",
                                                    });
                                                }
                                            }, 100);
                                        }
                                    }
                                }
                            } else if (currentEventType === 'done') {
                                if (firstSqlMessageId != null) {
                                    dispatch(HoudiniActions.setChatActivity({id: firstSqlMessageId, activity: progressTrace.filter(item => item.status === 'completed').map(item => item.id)}));
                                }
                                // Stream complete - finalize the streaming message
                                if (rejectedTool) {
                                    dispatch(HoudiniActions.completeStreamingMessage({
                                        id: streamingMessageId,
                                        message: { Type: 'error', Text: t('unsupportedToolRequest') },
                                    }));
                                } else if (streamingText === '' || streamingText.trim() === '') {
                                    // No message text was streamed, remove placeholder
                                    dispatch(HoudiniActions.removeChatMessage(streamingMessageId));
                                } else {
                                    // Complete the streaming message with final text
                                    dispatch(HoudiniActions.completeStreamingMessage({
                                        id: streamingMessageId,
                                        message: { Type: "message", Text: streamingText },
                                    }));
                                }
                                setLoading(false);
                                trackFrontendIntent('chat.message_completed', {
                                    input_length_bucket: textLengthBucket(sanitizedQuery),
                                    response_length_bucket: textLengthBucket(streamingText),
                                    message_count_bucket: countBucket(chats.length + 1),
                                    model_type: modelType.modelType,
                                    has_model: currentModel != null,
                                });

                                // Try to generate title if session still has default name
                                if (shouldTryTitle) {
                                    void generateChatTitle(sanitizedQuery);
                                }
                                streamDone = true;
                            } else if (currentEventType === 'error') {
                                dispatch(HoudiniActions.removeChatMessage(streamingMessageId));
                                const errorMessage = typeof parsed.error === 'string'
                                    ? parsed.error
                                    : parsed.error?.message ?? parsed.message ?? 'Unknown error';
                                toast.error(t('unableToQuery') + " " + errorMessage);
                                trackFrontendIntent('chat.message_failed', {
                                    input_length_bucket: textLengthBucket(sanitizedQuery),
                                    message_count_bucket: countBucket(chats.length),
                                    model_type: modelType.modelType,
                                    error_code: frontendAnalyticsErrorCode(errorMessage),
                                });
                                setLoading(false);
                                streamDone = true;
                            }
                        } catch (e) {
                            console.error('Failed to parse SSE data:', e);
                        }
                    }
                }

                // Stop reading after done/error — avoids WebKit throwing on post-EOF read in Wails
                if (streamDone) {
                    void reader.cancel();
                    break;
                }
            }
            if (!streamDone && !controller.signal.aborted) {
                if (rejectedTool) {
                    dispatch(HoudiniActions.completeStreamingMessage({
                        id: streamingMessageId,
                        message: { Type: 'error', Text: t('unsupportedToolRequest') },
                    }));
                } else if (streamingText.trim()) {
                    dispatch(HoudiniActions.completeStreamingMessage({
                        id: streamingMessageId,
                        message: { Type: "message", Text: streamingText },
                    }));
                } else {
                    dispatch(HoudiniActions.removeChatMessage(streamingMessageId));
                }
                toast.error(t('streamInterrupted'));
                setLoading(false);
            }
        } catch (error) {
            if (controller.signal.aborted) return;
            dispatch(HoudiniActions.removeChatMessage(streamingMessageId));
            const errorMessage = error instanceof Error
                ? error.message
                : typeof error === 'string'
                ? error
                : 'Unknown error';
            toast.error(t('unableToQuery') + " " + errorMessage);
            trackFrontendIntent('chat.message_failed', {
                input_length_bucket: textLengthBucket(sanitizedQuery),
                message_count_bucket: countBucket(chats.length),
                model_type: modelType.modelType,
                error_code: frontendAnalyticsErrorCode(error),
            });
            setLoading(false);
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [chats, currentModel, modelType, query, sourceScopeRef, dispatch, t, scrollContainerRef, getUniqueMessageId, activeSession, activeSessionId, autoScrollEnabled, displayChats]);

    // Helper function to generate and update chat title
    const generateChatTitle = useCallback(async (userQuery: string) => {
        if (!modelType || !activeSessionId || !currentModel) {
            return;
        }

        try {
            const result = await generateChatTitleMutation({
                variables: {
                    input: {
                        Query: userQuery,
                        ModelType: modelType.modelType,
                        ProviderId: modelType.id ?? undefined,
                        Token: modelType.token ?? undefined,
                        Model: currentModel,
                        Endpoint: undefined,
                    }
                }
            });

            const title = result.data?.GenerateChatTitle?.Title;
            if (title && title.trim() !== '') {
                dispatch(HoudiniActions.updateSessionName({
                    sessionId: activeSessionId,
                    name: title,
                }));
            }
        } catch (error) {
            console.error('[Chat Title] Failed to generate chat title:', error);
            // Non-critical error, don't show to user
        }
    }, [modelType, currentModel, activeSessionId, dispatch, generateChatTitleMutation]);

    const disableChat = useMemo(() => {
        return loading || modelType == null || currentModel == null || models.length === 0 || !modelAvailable || query.trim().length === 0;
    }, [loading, modelType, modelAvailable, models.length, currentModel, query]);

    const handleKeyDown: KeyboardEventHandler<HTMLInputElement> = useCallback((e) => {
        if (matchesShortcut(e, SHORTCUTS.clearEditor)) {
            e.preventDefault();
            setQuery('');
            setCurrentSearchIndex(undefined);
            return;
        }
    }, []);

    const handleKeyUp: KeyboardEventHandler<HTMLInputElement> = useCallback((e) => {
        if (e.key === "Enter") {
            if (query.trim().length > 0 && !disableChat) {
                void handleSubmitQuery();
            }
            return;
        }
        if (e.key === "ArrowUp") {
          const foundSearchIndex = currentSearchIndex != null ? currentSearchIndex - 1 : chats.length - 1;
          let searchIndex = foundSearchIndex;

          while (searchIndex >= 0) {
            if (chats[searchIndex].isUserInput) {
              setCurrentSearchIndex(searchIndex);
              setQuery(chats[searchIndex].Text);
              return;
            }
            searchIndex--;
          }

          // If we've exhausted the history (searchIndex < 0), clear the input
          if (searchIndex < 0) {
            setCurrentSearchIndex(undefined);
            setQuery('');
            return;
          }

          if (currentSearchIndex !== chats.length - 1) {
            searchIndex = chats.length - 1;
            while (searchIndex > foundSearchIndex) {
              if (chats[searchIndex].isUserInput) {
                setCurrentSearchIndex(searchIndex);
                setQuery(chats[searchIndex].Text);
                return;
              }
              searchIndex--;
            }
          }
        }
        if (e.key === "ArrowDown") {
          if (currentSearchIndex == null) return;

          let searchIndex = currentSearchIndex + 1;
          while (searchIndex < chats.length) {
            if (chats[searchIndex].isUserInput) {
              setCurrentSearchIndex(searchIndex);
              setQuery(chats[searchIndex].Text);
              return;
            }
            searchIndex++;
          }

          // Past the end of history — clear input
          setCurrentSearchIndex(undefined);
          setQuery('');
        }
    }, [chats, currentSearchIndex, query, handleSubmitQuery, disableChat]);

    const handleSelectExample = useCallback((example: string) => {
        draftAbandonedRef.current = false;
        trackFrontendIntent('chat.example_selected', {
            input_length_bucket: textLengthBucket(example),
            source: useDatabaseSuggestions ? 'database' : 'generic',
        });
        setQuery(example);
    }, [useDatabaseSuggestions]);

    const handleStop = useCallback(() => {
        chatRequest.current?.abort();
        if (streamingMessage.current != null) dispatch(HoudiniActions.removeChatMessage(streamingMessage.current));
        streamingMessage.current = null;
        setLoading(false);
    }, [dispatch]);

    const handleClear = useCallback(() => {
        trackFrontendIntent('chat.cleared', {
            message_count_bucket: countBucket(chats.length),
        });
        dispatch(HoudiniActions.clear());
        setQuery("");
        setCurrentSearchIndex(undefined);
        // Reset suggestions fetch flag to allow fetching again
        hasFetchedSuggestionsRef.current = false;
        setDatabaseSuggestions([]);
        setUseDatabaseSuggestions(false);
    }, [chats.length, dispatch]);

    const handleConfirmSQL = useCallback(async (messageId: number, sql: string, operationType: string) => {
        setExecutingConfirmedId(messageId);
        trackFrontendIntent('chat.sql_confirmation_accepted', {
            operation_type: operationType,
        });
        try {
            const { data, error } = await executeConfirmedSql({
                variables: {
                    query: sql,
                    operationType: operationType,
                },
            });

            if (error || !data) {
                toast.error(t('unableToQuery') + " " + (error?.message ?? t('failedToExecuteSQL')));
                setLoading(false);
                return;
            }

            const result = data.ExecuteConfirmedSQL;

            // Update the confirmation message in place with the result
            dispatch(HoudiniActions.completeStreamingMessage({
                id: messageId,
                message: {
                    Type: result.Type,
                    Text: result.Text,
                    Result: result.Result,
                    RequiresConfirmation: false,
                },
            }));

            if (autoScrollEnabled) {
                setTimeout(() => {
                    if (scrollContainerRef.current != null) {
                        scrollContainerRef.current.scroll({
                            top: scrollContainerRef.current.scrollHeight,
                            behavior: "smooth",
                        });
                    }
                }, 100);
            }

        } catch (error) {
            const errorMessage = error instanceof Error
                ? error.message
                : typeof error === 'string'
                ? error
                : 'Unknown error';
            toast.error(t('unableToQuery') + " " + errorMessage);
        } finally {
            setExecutingConfirmedId(null);
        }
    }, [autoScrollEnabled, executeConfirmedSql, dispatch, t, scrollContainerRef]);

    const handleCancelSQL = useCallback((messageId: number) => {
        trackFrontendIntent('chat.sql_confirmation_declined');
        dispatch(HoudiniActions.removeChatMessage(messageId));
        toast.info(t('queryCancelled') || 'Query cancelled');
    }, [dispatch, t]);

    const disableAll = useMemo(() => {
        return modelType == null || currentModel == null || models.length === 0 || !modelAvailable;
    }, [modelType, modelAvailable, models.length, currentModel]);

    // Initialize chat sessions on mount
    const hasInitialized = useRef(false);
    useEffect(() => {
        if (!hasInitialized.current) {
            hasInitialized.current = true;
            dispatch(HoudiniActions.initializeChatSessions());
        }
    }, [dispatch]);

    // Auto-scroll to bottom when chats change or component mounts
    useEffect(() => {
        if (autoScrollEnabled && scrollContainerRef.current != null && chats.length > 0) {
            scrollContainerRef.current.scrollTop = scrollContainerRef.current.scrollHeight;
        }
    }, [activeSessionId, autoScrollEnabled, chats.length]);

    // Fetch database-specific suggestions when AI is available and chat is empty
    useEffect(() => {
        // Only fetch if:
        // 1. AI model is available
        // 2. Chat is empty (showing examples)
        // 3. Haven't already attempted to fetch
        const shouldFetch =
            modelAvailable &&
            currentModel &&
            chats.length === 0 &&
            !hasFetchedSuggestionsRef.current;

        if (shouldFetch) {
            hasFetchedSuggestionsRef.current = true;
            getDatabaseSuggestions({
                variables: {
                    ref: sourceScopeRef,
                },
            }).then(({ data }) => {
                if (data?.DatabaseQuerySuggestions && data.DatabaseQuerySuggestions.length > 0) {
                    setDatabaseSuggestions(data.DatabaseQuerySuggestions.map(s => ({
                        description: s.description,
                        category: s.category,
                    })));
                    setUseDatabaseSuggestions(true);
                } else {
                    // Fallback to generic examples
                    setUseDatabaseSuggestions(false);
                }
            }).catch((error) => {
                console.error('[Database Suggestions] Error fetching suggestions:', error);
                // On error, fallback to generic examples
                setUseDatabaseSuggestions(false);
            });
        }
    }, [chats.length, currentModel, getDatabaseSuggestions, modelAvailable, sourceScopeRef]);

    const assistantAvatar = <span className="ce-chat-avatar">{extensions.MetaIcon ?? <img src={logoImage} alt="" />}</span>;

    return (
        <InternalPage routes={[InternalRoutes.Chat]} className="h-full min-w-0 !overflow-hidden" sidebar={__WHODB_EDITION__ === 'ce' ? undefined : <ChatHistorySidebar />} subSidebarWidth="18rem" fullHeight>
            <div className="ce-chat-page flex h-full min-h-0 w-full min-w-0 flex-col">
                <div className="ce-chat-toolbar flex min-h-11 w-full items-center border-b border-border px-3 pb-2">
                    <AIProvider
                        {...aiState}
                        onClear={handleClear}
                        disableNewChat
                    />
                    <Button size="sm" variant="outline" onClick={handleClear} data-testid="chat-new-chat"><WhoDBChatIcon name="plus" />{t('newChat')}</Button>
                </div>
                <div className="flex min-h-0 w-full flex-1 overflow-hidden">
                    {
                        chats.length === 0
                        ? disableAll
                            ? <div className="flex h-full w-full items-center justify-center"><EmptyState title={t('noModelTitle')} description={t('noModelDescription')} icon={<SparklesIcon className="w-16 h-16" data-testid="empty-state-sparkles-icon" />} /></div>
                            : <div className="ce-chat-empty flex flex-col justify-center items-center w-full gap-8" data-testid="chat-empty-state-container">
                            <img src={logoImage} alt="" className="ce-chat-empty-logo" />
                            <div className="ce-chat-empty-heading"><h1>{t('askAboutSource', {source: authProfile?.Database === 'whodb-sample' ? t('sampleSQLite') : (authProfile?.DisplayName ?? authProfile?.Database ?? '')})}</h1><p>{t('emptyStateDescription')}</p></div>
                            {suggestionsLoading ? (
                                <Loading loadingText={t('loadingSuggestions')} size="sm" />
                            ) : (
                                <div className="flex flex-col gap-2 items-center">
                                    {!useDatabaseSuggestions && examples.length > 0 && (
                                        <p className="text-xs text-muted-foreground">{t('genericExamplesLabel')}</p>
                                    )}
                                    {useDatabaseSuggestions && databaseSuggestions.length > 0 && (
                                        <p className="text-xs text-muted-foreground">{t('databaseSpecificSuggestionsLabel')}</p>
                                    )}
                                    <div className="ce-chat-examples flex flex-wrap justify-center items-center gap-4" data-testid="chat-examples-list">
                                        {
                                            examples.slice(0, 3).map((example) => (
                                                <Card key={example.description} className="flex flex-col gap-sm w-[250px] h-[120px] p-4 text-sm cursor-pointer hover:opacity-80 transition-all"
                                                    onClick={() =>{  handleSelectExample(example.description); }}>
                                                    {example.icon}
                                                    {example.description}
                                                </Card>
                                            ))
                                        }
                                    </div>
                                </div>
                            )}
                        </div>
                        : <div className="ce-chat-conversation h-full w-full overflow-y-auto overflow-x-hidden px-3 pt-6 pb-12" ref={scrollContainerRef}>
                            <div className="flex w-full min-h-full max-w-full justify-center">
                                <div className="mx-auto flex w-full max-w-4xl min-w-0 flex-col gap-3">
                                    {
                                        displayChats.map((chat, i) => {
                                            const firstReply = !chat.isUserInput && displayChats[i-1]?.isUserInput;
                                            const nextUser = firstReply ? displayChats.findIndex((message, index) => index > i && message.isUserInput) : -1;
                                            const turn = firstReply ? displayChats.slice(i, nextUser < 0 ? undefined : nextUser) : [];
                                            const recordedSteps = turn.find(message => message.activity?.length)?.activity ?? [];
                                            const activitySteps = recordedSteps.map(id => ({id, label: id.startsWith('schema:') ? t('readSchema') : id.startsWith('plan:') ? t('preparingQuery') : id.startsWith('query:') ? t('runQuery') : id.startsWith('draft:') ? t('draftChange') : t('retryingQuery')}));
                                            const activity = activitySteps.length > 0 && <details className="ce-chat-activity"><summary><WhoDBChatIcon name="check-circle" /><strong>{t('activitySteps', {count: activitySteps.length})}</strong>{activitySteps.map(step => <span key={step.id}>{step.label}</span>)}<b>{t('activityDetails')}</b></summary><ol>{activitySteps.map(step => <li key={step.id}>{step.label}</li>)}</ol></details>;
                                            if (chat.Type === 'activity') return null;
                                            const priorUserText = displayChats.slice(0, i).reverse().find(message => message.isUserInput)?.Text ?? '';
                                            const isOntologyQuestion = /ontolog/i.test(priorUserText);
                                            const isSampleDatabase = authProfile?.Database === 'whodb-sample';
                                            const sourceName = isSampleDatabase ? t('sampleDatabase') : (authProfile?.Database ?? '');
                                            const setSuggestedQuery = (suggestion: string) => {
                                                setQuery(suggestion);
                                                document.querySelector<HTMLInputElement>('[data-testid="chat-input"]')?.focus();
                                            };
                                            const renderScopeCard = () => <div className="ce-chat-error-card ce-chat-scope-card">
                                                <div className="ce-chat-error-heading"><WhoDBChatIcon name="help" /><div><strong>{t('actionUnavailable')}</strong><span>{t('toolActionNotRun')}</span></div></div>
                                                <div className="ce-chat-error-body"><p>{chat.Type === 'scope:error' ? t('unsupportedToolRequest') : chat.Text}</p><div className="ce-chat-error-actions"><button type="button" className="ce-chat-primary-action" onClick={() => { setSuggestedQuery(isOntologyQuestion ? t('tableConnectionsPrompt') : t('listTablesPrompt')); }}>{isOntologyQuestion ? t('showTableConnections') : t('tryAvailableTables')}</button><button type="button" onClick={() => { void navigate(InternalRoutes.Graph.path); }}><WhoDBChatIcon name="relation" />{t('openGraph')}</button></div></div>
                                            </div>;
                                            if (chat.Type === "message" || chat.Type === "text") {
                                                if (chat.isStreaming && !chat.Text) return null;
                                                return <div key={`chat-${chat.id}`} className={classNames("flex gap-lg overflow-hidden break-words leading-6 shrink-0 relative group/msg", {
                                                    "self-end ml-3": chat.isUserInput,
                                                    "self-start": !chat.isUserInput,
                                                })} data-testid={chat.isUserInput ? "user-message" : "system-message"}>
                                                    {!chat.isUserInput && displayChats[i-1]?.isUserInput
                                                        ? assistantAvatar
                                                        : <div className="ce-chat-avatar-spacer" />}
                                                    {chat.isUserInput ? (
                                                        <div className="flex flex-col items-end">
                                                            <p className={classNames("rounded-xl bg-primary/10 px-4 py-2 whitespace-pre-wrap dark:bg-primary/25", {
                                                                "animate-fade-in": chat.isStreaming,
                                                            })} data-input-message="user">
                                                                {chat.Text}
                                                                {chat.isStreaming && <span className="inline-block w-2 h-4 ml-1 bg-current animate-pulse" />}
                                                            </p>
                                                            <MessageCopyAction text={chat.Text} />
                                                        </div>
                                                    ) : (
                                                        <div className="flex flex-col">
                                                            {activity}
                                                            <div className={classNames("py-2 rounded-xl markdown-content", {
                                                                "animate-fade-in": chat.isStreaming,
                                                            })} data-input-message="system">
                                                                <ReactMarkdown components={markdownComponents}>
                                                                    {chat.Text}
                                                                </ReactMarkdown>
                                                                {chat.isStreaming && <span className="inline-block w-2 h-4 ml-1 bg-current animate-pulse" />}
                                                            </div>
                                                            {!chat.isStreaming && <MessageCopyAction text={chat.Text} />}
                                                        </div>
                                                    )}
                                                </div>
                                            } else if (chat.Type === 'scope:error') {
                                                return <div key={`chat-${chat.id}`} className="ce-chat-error-row" data-testid="scope-error-message">{firstReply ? assistantAvatar : <div className="ce-chat-avatar-spacer" />}{renderScopeCard()}</div>;
                                            } else if (chat.Type === 'provider:error' || (chat.Type === 'error' && /connection refused|could not connect/i.test(chat.Text))) {
                                                return <div key={`chat-${chat.id}`} className="ce-chat-error-row" data-testid="provider-error-message">{firstReply ? assistantAvatar : <div className="ce-chat-avatar-spacer" />}<div className="ce-chat-provider-error"><WhoDBChatIcon name="help" /><div><strong>{t('cantReachProvider', {provider: modelType?.name ?? modelType?.modelType ?? t('modelProvider')})}</strong><span>{chat.Text}</span></div><button type="button" onClick={() => { document.querySelector<HTMLButtonElement>('[data-testid="ai-model-select"]')?.click(); }}>{t('changeModel')}</button><button type="button" className="ce-chat-primary-action" disabled={loading || !priorUserText} onClick={() => { void handleSubmitQuery(priorUserText, chat.id); }}><WhoDBChatIcon name="play" />{t('retry')}</button></div></div>;
                                            } else if (chat.Type === 'sql:error') {
                                                const failure = parseSQLFailure(chat.Text);
                                                if (!failure) return null;
                                                const badLine = failedSQLLine(failure.sql, failure.error);
                                                const missingTable = /no such table:\s*["'`]?([\w.]+)/i.exec(failure.error)?.[1];
                                                const lines = failure.sql.split('\n').map((line, index) => ({id: `${index + 1}:${line}`, number: index + 1, text: line}));
                                                const editInScratchpad = () => {
                                                    if (scratchpadPageId) dispatch(ScratchpadActions.addCellToPageAndActivate({pageId: scratchpadPageId, initialQuery: failure.sql}));
                                                    else dispatch(ScratchpadActions.addPage({name: t('failedQueryPageName'), initialQuery: failure.sql}));
                                                    void navigate(InternalRoutes.RawExecute.path, {state: {targetPage: scratchpadPageId ?? 'new'}});
                                                };
                                                return <div key={`chat-${chat.id}`} className="ce-chat-error-row" data-testid="sql-error-message">{firstReply ? assistantAvatar : <div className="ce-chat-avatar-spacer" />}<div className="ce-chat-error-card ce-chat-sql-error-card"><div className="ce-chat-error-heading"><WhoDBChatIcon name="help" /><div><strong>{t('queryFailed')}</strong><span>{t('queryErrorSource', {source: sourceName})}</span></div><b>{missingTable ? t('noSuchTable') : t('queryError')}</b></div><div className="ce-chat-error-body">{activity}<p>{missingTable ? t('missingTableExplanation', {table: missingTable}) : failure.error}</p><pre className="ce-chat-failed-sql">{lines.map(line => <span key={line.id} className={badLine === line.number - 1 ? 'ce-chat-failed-line' : undefined}><small>{line.number}</small>{line.text}</span>)}</pre><div className="ce-chat-error-actions"><button type="button" className="ce-chat-primary-action" onClick={() => { setSuggestedQuery(isSampleDatabase ? t('countOrdersByUserPrompt') : t('listTablesPrompt')); }}>{isSampleDatabase ? t('countOrdersByUserInstead') : t('tryAvailableTables')}</button><button type="button" disabled={!supportsScratchpad} onClick={editInScratchpad}><WhoDBChatIcon name="code" />{t('editInScratchpad')}</button></div></div></div></div>;
                                            } else if (chat.Type === "error") {
                                                const errorText = chat.Text.replace(/^ERROR:\s*/i, "");
                                                const unsupportedAction = errorText === t('unsupportedToolRequest');
                                                return (
                                                    <div key={`chat-${chat.id}`} className="flex gap-2 overflow-hidden break-words leading-6 shrink-0 pt-6 relative self-start" data-testid="error-message">
                                                        {!chat.isUserInput && displayChats[i-1]?.isUserInput
                                                            ? assistantAvatar
                                                            : null}
                                                        {unsupportedAction
                                                            ? renderScopeCard()
                                                            : <ErrorState error={errorText} />}
                                                    </div>
                                                );
                                            } else if (featureFlags.dataVisualization && (chat.Type === "sql:pie-chart" || chat.Type === "sql:line-chart")) {
                                                return <div key={`chat-${chat.id}`} className={cn("flex gap-lg w-full max-w-full min-w-0 pt-4 relative", ph.mask)} data-testid="visual-message">
                                                    {!chat.isUserInput && displayChats[i-1]?.isUserInput && assistantAvatar}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:pie-chart" && PieChart && <PieChart columns={chat.Result?.Columns?.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} text={chat.Text} />}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:line-chart" && LineChart && <LineChart columns={chat.Result?.Columns?.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} text={chat.Text} />}
                                                </div>
                                            } else if (chat.RequiresConfirmation) {
                                                // Show confirmation UI inline
                                                const isExecuting = executingConfirmedId === chat.id;
                                                return <div key={`chat-${chat.id}`} className="flex gap-lg w-full max-w-full min-w-0 pt-4 relative" data-testid="confirmation-message">
                                                    {!chat.isUserInput && displayChats[i-1]?.isUserInput
                                                        ? assistantAvatar
                                                        : <div className="ce-chat-avatar-spacer" />}
                                                    <div className="ce-chat-change-card flex flex-col gap-3 w-[calc(100%-50px)] max-w-full min-w-0">
                                                        {activity}
                                                        <div className="ce-chat-change-heading"><WhoDBChatIcon name="code" /><div><strong>{t('reviewChange')}</strong><span>{t('changeNeedsConfirmation')}</span></div></div>
                                                        <div className="ce-chat-change-sql relative w-full rounded-lg overflow-hidden">
                                                                <code>{chat.Text}</code>
                                                                <button
                                                                    onClick={() => {
                                                                        void copyToClipboard(chat.Text).then(success => {
                                                                            if (success) {
                                                                                setCopiedSqlId(chat.id ?? null);
                                                                                setTimeout(() =>{  setCopiedSqlId(null); }, 2000);
                                                                            }
                                                                        });
                                                                    }}
                                                                    className="absolute top-2 right-2 flex items-center gap-1 px-1.5 py-0.5 rounded text-xs text-neutral-500 dark:text-neutral-400 hover:text-neutral-800 dark:hover:text-neutral-100 bg-neutral-200 dark:bg-neutral-700 hover:bg-neutral-300 dark:hover:bg-neutral-600 transition-colors z-10"
                                                                >
                                                                    <WhoDBChatIcon name={copiedSqlId === chat.id ? 'check-circle' : 'copy'} />
                                                                    <span>{copiedSqlId === chat.id ? t('copied') : t('copy')}</span>
                                                                </button>
                                                        </div>

                                                        {/* Action Buttons */}
                                                        <div className="flex gap-2 justify-end">
                                                            <Button
                                                                variant="outline"
                                                                onClick={() => { if (chat.id) handleCancelSQL(chat.id); }}
                                                                disabled={isExecuting}
                                                                size="sm"
                                                            >
                                                                {t('dontRun')}
                                                            </Button>
                                                            <Button
                                                                onClick={() => { if (chat.id) void handleConfirmSQL(chat.id, chat.Text, chat.Type); }}
                                                                disabled={isExecuting || !supportsScripts}
                                                                size="sm"
                                                            >
                                                                {isExecuting && <Spinner />}{isExecuting ? t('running') : t('runStatement')}
                                                            </Button>
                                                        </div>
                                                    </div>
                                                </div>
                                            }
                                            return <div key={`chat-${chat.id}`} className="flex gap-lg w-full max-w-full min-w-0 pt-4 relative" data-testid="table-message">
                                                {firstReply ? assistantAvatar : <div className="ce-chat-avatar-spacer" />}
                                                <div className="flex min-w-0 flex-1 flex-col gap-2">{activity}<TablePreview type={chat.Type} text={chat.Text} data={chat.Result} containerWidth={containerWidth} /></div>
                                            </div>
                                        })
                                    }
                                </div>
                            </div>
                        </div>
                    }
                </div>
                {loading && <div className="ce-chat-pinned-progress">
                    <div className="ce-chat-thinking" role="status">
                        <div className="ce-chat-thinking-heading"><div><img src={logoImage} alt="" className="ce-chat-working-avatar" /><strong>{t('workingOnIt')}</strong><span className="ce-chat-thinking-count">{progressSteps.filter(item => item.status === 'completed').length}/{progressSteps.length || 1}</span></div><span>{t('runOnSource', {source: authProfile?.Database === 'whodb-sample' ? t('sampleDatabase') : (authProfile?.Database ?? '')})}</span></div>
                        <div className="ce-chat-thinking-list">
                            {(progressSteps.length ? progressSteps : [{id: 'thinking', step: 'thinking', status: 'started'}]).map(item => {
                                const label = item.step === 'schema' ? t('readingSchema') : item.step === 'plan' ? t('preparingQuery') : item.step === 'query' ? t('runQuery') : item.step === 'draft' ? t('draftChange') : item.step === 'retry' ? t('retryingQuery') : t('thinking');
                                return <div key={item.id} className="ce-chat-thinking-step"><span className="ce-chat-step-icon">{item.status === 'completed' ? <WhoDBChatIcon name="check-circle" /> : <Spinner />}</span><span>{label}</span></div>;
                            })}
                        </div>
                    </div>
                    <p className="ce-chat-stop-hint">{t('pressStopToCancel')}</p>
                </div>}
                <div className={classNames("ce-chat-composer flex shrink-0 items-center gap-2 border-t border-border px-3 py-4", {
                    "opacity-80": disableChat,
                    "opacity-10": disableAll,
                })}>
                    <div className="mx-auto flex w-full max-w-4xl items-center gap-2 rounded-lg border border-border bg-card p-1.5 shadow-sm">
                        <Input
                            value={query}
                            onChange={e =>{  setQuery(e.target.value); }}
                            placeholder={t('placeholder')}
                            onSubmit={() => { void handleSubmitQuery(); }}
                            disabled={disableAll}
                            onKeyDown={handleKeyDown}
                            onKeyUp={handleKeyUp}
                            autoComplete="off"
                            data-testid="chat-input"
                            className="!border-0 !bg-transparent !shadow-none !ring-0"
                        />
                        <Tip className="w-fit">
                            <Button tabIndex={0} onClick={loading ? handleStop : () => { void handleSubmitQuery(); }} className="size-9 rounded-md p-0" disabled={disableChat && !loading} variant={disableChat && !loading ? "secondary" : undefined} data-testid="icon-button" aria-label={loading ? t('stop') : t('sendMessage')}>
                                {loading ? <span className="size-2 rounded-sm bg-current" /> : <WhoDBChatIcon name="send" className="size-4" />}
                            </Button>
                            <p>{loading ? t('stop') : t('sendMessage')}</p>
                        </Tip>
                    </div>
                </div>
            </div>
        </InternalPage>
    )
}
