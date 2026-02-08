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

import {isEEMode} from "@/config/ee-imports";
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
    EmptyState,
    Input,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    toast
} from "@clidey/ux";
import {GetAiChatQuery, useExecuteConfirmedSqlMutation, useGetAiChatLazyQuery} from '@graphql';
import {
    ArrowUpCircleIcon,
    CheckCircleIcon,
    CodeBracketIcon,
    CommandLineIcon,
    SparklesIcon,
    TableCellsIcon
} from "../../components/heroicons";
import classNames from "classnames";
import {cloneElement, FC, KeyboardEventHandler, useCallback, useEffect, useMemo, useRef, useState} from "react";
import ReactMarkdown from 'react-markdown';
import logoImage from "../../../public/images/logo.svg";
import {AIProvider, useAI} from "../../components/ai";
import {CodeEditor} from "../../components/editor";
import {ErrorState} from "../../components/error-state";
import {Loading} from "../../components/loading";
import {InternalPage} from "../../components/page";
import {StorageUnitTable} from "../../components/table";
import {extensions} from "../../config/features";
import {InternalRoutes} from "../../config/routes";
import {HoudiniActions} from "../../store/chat";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {ScratchpadActions} from "../../store/scratchpad";
import {isEEFeatureEnabled, loadEEComponent} from "../../utils/ee-loader";
import {chooseRandomItems} from "../../utils/functions";
import {databaseSupportsScratchpad, databaseTypesThatUseDatabaseInsteadOfSchema} from "../../utils/database-features";
import {useNavigate} from "react-router-dom";
import {useChatExamples} from "./examples";
import {useTranslation} from '@/hooks/use-translation';
import {addAuthHeader, isDesktopScheme} from "../../utils/auth-headers";

// Lazy load chart components if EE is enabled
const LineChart = isEEFeatureEnabled('dataVisualization') ? loadEEComponent(
    () => import('@ee/components/charts/line-chart').then(m => ({ default: m.LineChart })),
    () => null
) : () => null;

const PieChart = isEEFeatureEnabled('dataVisualization') ? loadEEComponent(
    () => import('@ee/components/charts/pie-chart').then(m => ({ default: m.PieChart })),
    () => null
) : () => null;

const THINKING_PHRASES_COUNT = 25;


type TableData = GetAiChatQuery["AIChat"][0]["Result"];

const TablePreview: FC<{ type: string, data: TableData, text: string }> = ({ type, data, text }) => {
    const { t } = useTranslation('pages/chat');
    const dispatch = useAppDispatch();
    const [showSQL, setShowSQL] = useState(false);
    const [showScratchpadDialog, setShowScratchpadDialog] = useState(false);
    const [selectedPage, setSelectedPage] = useState<string>("new");
    const [newPageName, setNewPageName] = useState<string>("");
    const navigate = useNavigate();
    const current = useAppSelector(state => state.auth.current);
    const { pages, activePageId } = useAppSelector(state => state.scratchpad);

    const handleCodeToggle = useCallback(() => {
        setShowSQL(status => !status);
    }, []);

    // Create page options excluding current page
    const pageOptions = useMemo(() => {
        return [
            ...pages.map(page => ({ value: page.id, label: page.name })),
            { value: "new", label: t('createNewPage') }
        ];
    }, [pages, activePageId, t]);

    const handleMoveToScratchpad = useCallback(() => {
        if (!databaseSupportsScratchpad(current?.Type)) {
            toast.error(t('scratchpadNotSupported'));
            return;
        }
        // Initialize scratchpad if needed
        if (pages.length === 0) {
            dispatch(ScratchpadActions.ensurePagesHaveCells());
        }
        setShowScratchpadDialog(true);
    }, [current?.Type, pages.length, dispatch, t]);

    const handleScratchpadConfirm = useCallback(() => {
        if (selectedPage === "new") {
            // Create new page with the query
            const pageName = newPageName.trim() || `Page ${pages.length + 1}`;
            dispatch(ScratchpadActions.addPage({ name: pageName, initialQuery: text }));
            // Navigate to scratchpad - the new page will be created with the query
            navigate(InternalRoutes.RawExecute.path, {
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
            navigate(InternalRoutes.RawExecute.path, {
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

    const canMoveToScratchpad = useMemo(() => {
        return databaseSupportsScratchpad(current?.Type) && type.startsWith("sql:");
    }, [current?.Type, type]);

    return <div className="flex flex-col w-[calc(100%-50px)] group/table-preview">
        <div className="opacity-0 group-hover/table-preview:opacity-100 focus-within:opacity-100 transition-all flex gap-1 -transalte-y-full absolute top-0 z-[10]">
            <Button onClick={handleCodeToggle} data-testid="icon-button" variant="outline" aria-label={showSQL ? t('showTable') : t('showCode')}>
                {cloneElement(showSQL ? <TableCellsIcon className="w-6 h-6" /> : <CodeBracketIcon className="w-6 h-6" />, {
                    className: "w-6 h-6",
                    "aria-hidden": true,
                })}
            </Button>
            {canMoveToScratchpad && (
                <Button
                    variant="outline"
                    onClick={handleMoveToScratchpad}
                    data-testid="icon-button"
                    title={t('moveToScratchpad')}
                    aria-label={t('moveToScratchpad')}
                >
                    <CommandLineIcon className="w-6 h-6" aria-hidden="true" />
                </Button>
            )}
        </div>
        <div className="flex items-center gap-lg overflow-hidden break-all leading-6 shrink-0 h-full w-full">
            {
                showSQL
                ? <div className="h-[150px] w-full">
                    <CodeEditor value={text} />
                </div>
                :  (data != null && data.Rows.length > 0) || type === "sql:get"
                    ? <div className="w-full">
                        <StorageUnitTable
                            columns={data?.Columns?.map(c => c.Name) ?? []}
                            columnTypes={data?.Columns?.map(c => c.Type) ?? []}
                            rows={data?.Rows ?? []}
                            disableEdit={true}
                            limitContextMenu={true}
                            databaseType={current?.Type}
                            height={250}
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
                    <DialogTitle>{t('dialogTitle')}</DialogTitle>
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
                                onChange={(e) => setNewPageName(e.target.value)}
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
}

export const ChatPage: FC = () => {
    const { t } = useTranslation('pages/chat');
    const [query, setQuery] = useState("");
    const chats = useAppSelector(state => state.houdini.chats);
    const [getAIChat, { loading: getAIChatLoading }] = useGetAiChatLazyQuery();
    const [executeConfirmedSql] = useExecuteConfirmedSqlMutation();
    const scrollContainerRef = useRef<HTMLDivElement>(null);
    const schemaFromState = useAppSelector(state => state.database.schema);
    const authProfile = useAppSelector(state => state.auth.current);
    const [executingConfirmedId, setExecutingConfirmedId] = useState<number | null>(null);
    const [showQueryForId, setShowQueryForId] = useState<number | null>(null);
    const messageIdCounter = useRef(0);

    // For databases that use "database" instead of "schema" (MySQL, MariaDB, etc.),
    // we need to pass the database value where the backend expects "schema"
    const schema = useMemo(() => {
        if (databaseTypesThatUseDatabaseInsteadOfSchema(authProfile?.Type)) {
            return authProfile?.Database || '';
        }
        return schemaFromState;
    }, [authProfile?.Type, authProfile?.Database, schemaFromState]);
    const [currentSearchIndex, setCurrentSearchIndex] = useState<number>();

    const dispatch = useAppDispatch();

    // Generate unique message IDs to prevent collisions
    const getUniqueMessageId = useCallback(() => {
        messageIdCounter.current += 1;
        return Date.now() * 1000 + messageIdCounter.current;
    }, []);

    const aiState = useAI();
    const { modelType, currentModel, modelAvailable, models } = aiState;

    const chatExamples = useChatExamples();

    const thinkingPhrases = useMemo(() => {
        return Array.from({ length: THINKING_PHRASES_COUNT }, (_, i) => t(`thinking${i}`));
    }, [t]);

    const [loading, setLoading] = useState(false);
    const loadingPhraseRef = useRef<string>("");

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

    // Apply stable indices to current examples (allows localization changes to work)
    const examples = useMemo(() => {
        if (exampleIndicesRef.current === null) {
            return chatExamples.slice(0, 3);
        }
        return exampleIndicesRef.current.map(i => chatExamples[i]);
    }, [chatExamples]);

    const handleSubmitQuery = useCallback(async () => {
        const sanitizedQuery = query.trim();
        if (modelType == null || sanitizedQuery.length === 0) {
            return;
        }

        setLoading(true);
        loadingPhraseRef.current = isEEMode ? thinkingPhrases[0] : chooseRandomItems(thinkingPhrases)[0];
        dispatch(HoudiniActions.addChatMessage({ Type: "message", Text: sanitizedQuery, isUserInput: true, RequiresConfirmation: false }));
        setQuery("");

        // Add a placeholder for streaming text
        const streamingMessageId = getUniqueMessageId();
        dispatch(HoudiniActions.addChatMessage({
            Type: "message",
            Text: "",
            isStreaming: true,
            id: streamingMessageId,
            RequiresConfirmation: false
        }));

        setTimeout(() => {
            if (scrollContainerRef.current != null) {
                scrollContainerRef.current.scroll({
                    top: scrollContainerRef.current.scrollHeight,
                    behavior: "smooth",
                });
            }
        }, 250);

        try {
            const isDesktop = isDesktopScheme();
            const endpoint = '/api/ai-chat/stream';

            const requestBody = {
                schema,
                modelType: modelType.modelType,
                providerId: modelType.id || '',
                token: modelType.token || '',
                model: currentModel ?? '',
                input: {
                    Query: sanitizedQuery,
                    PreviousConversation: chats.map(chat =>
                        `${chat.isUserInput ? "<User>" : "<System>"}${chat.Text}${chat.isUserInput ? "</User>" : "</System>"}`
                    ).join("\n"),
                },
            };

            const response = await fetch(endpoint, {
                method: 'POST',
                credentials: 'include',
                headers: addAuthHeader({
                    'Content-Type': 'application/json',
                }),
                body: JSON.stringify(requestBody),
            });

            if (!response.ok) {
                setLoading(false);
                return;
            }

            // Check response type - server returns JSON for non-streaming (Wails), SSE for streaming
            const contentType = response.headers.get('Content-Type') || '';
            const isNonStreaming = contentType.includes('application/json') || isDesktop;

            // Non-streaming mode (desktop/Wails) - server returns JSON
            if (isNonStreaming) {
                const data = await response.json();
                // Server returns { messages: [...], done: true }
                const messages = data.messages || data;

                // Remove the placeholder streaming message
                dispatch(HoudiniActions.removeChatMessage(streamingMessageId));

                // Add all messages from the response
                if (Array.isArray(messages)) {
                    for (const msg of messages) {
                        const messageId = getUniqueMessageId();
                        dispatch(HoudiniActions.addChatMessage({
                            Type: msg.Type,
                            Text: msg.Text,
                            Result: msg.Result,
                            RequiresConfirmation: msg.RequiresConfirmation || false,
                            id: messageId,
                        }));
                    }
                }

                setLoading(false);

                // Scroll to bottom
                setTimeout(() => {
                    if (scrollContainerRef.current != null) {
                        scrollContainerRef.current.scroll({
                            top: scrollContainerRef.current.scrollHeight,
                            behavior: "smooth",
                        });
                    }
                }, 100);
                return;
            }

            // SSE streaming mode (browser)
            if (!response.body) {
                setLoading(false);
                return;
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let streamingText = '';
            let currentEventType = '';
            let addedSqlMessages = new Set<string>(); // Track added SQL to avoid duplicates

            while (true) {
                const { done, value } = await reader.read();
                if (done) {
                    break;
                }

                const chunk = decoder.decode(value, { stream: true });
                const lines = chunk.split('\n');

                for (const line of lines) {
                    if (line.startsWith('event: ')) {
                        currentEventType = line.slice(7).trim();
                    } else if (line.startsWith('data: ')) {
                        const data = line.slice(6);
                        if (!data.trim()) continue;

                        try {
                            const parsed = JSON.parse(data);

                            if (currentEventType === 'chunk') {
                                const text = parsed.text || '';
                                const chunkType = parsed.type || '';

                                // Only update message text if it's longer (ignore SQL/error chunks)
                                if (chunkType !== 'sql' && chunkType !== 'error' && text && text.length > streamingText.length) {
                                    streamingText = text;
                                    dispatch(HoudiniActions.updateChatMessage({
                                        id: streamingMessageId,
                                        Text: streamingText,
                                    }));
                                }

                                // Auto-scroll
                                if (scrollContainerRef.current != null) {
                                    scrollContainerRef.current.scroll({
                                        top: scrollContainerRef.current.scrollHeight,
                                        behavior: "smooth",
                                    });
                                }
                            } else if (currentEventType === 'message') {
                                // Handle complete messages (SQL responses and errors after streaming)
                                if (parsed.Type?.startsWith("sql") || parsed.Type === "error") {
                                    // Create a unique key for this message to avoid duplicates
                                    const messageKey = `${parsed.Type}:${parsed.Text}`;

                                    // Only add if we haven't seen this message before
                                    if (!addedSqlMessages.has(messageKey)) {
                                        addedSqlMessages.add(messageKey);

                                        const messageId = getUniqueMessageId();
                                        dispatch(HoudiniActions.addChatMessage({
                                            Type: parsed.Type,
                                            Text: parsed.Text,
                                            Result: parsed.Result,
                                            RequiresConfirmation: parsed.RequiresConfirmation || false,
                                            id: messageId,
                                        }));

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
                            } else if (currentEventType === 'done') {
                                // Stream complete - finalize the streaming message
                                if (streamingText === '' || streamingText.trim() === '') {
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
                            } else if (currentEventType === 'error') {
                                dispatch(HoudiniActions.removeChatMessage(streamingMessageId));
                                const errorMessage = typeof parsed.error === 'string'
                                    ? parsed.error
                                    : parsed.error?.message || parsed.message || 'Unknown error';
                                toast.error(t('unableToQuery') + " " + errorMessage);
                                setLoading(false);
                            }
                        } catch (e) {
                            console.error('Failed to parse SSE data:', e);
                        }
                    }
                }
            }
        } catch (error) {
            dispatch(HoudiniActions.removeChatMessage(streamingMessageId));
            const errorMessage = error instanceof Error
                ? error.message
                : typeof error === 'string'
                ? error
                : 'Unknown error';
            toast.error(t('unableToQuery') + " " + errorMessage);
            setLoading(false);
        }
    }, [chats, currentModel, modelType, query, schema, dispatch, t, scrollContainerRef, getUniqueMessageId]);

    const disableChat = useMemo(() => {
        return loading || models.length === 0 || (!modelAvailable && !currentModel) || query.trim().length === 0;
    }, [loading, modelAvailable, models.length, currentModel, query]);

    const handleKeyUp: KeyboardEventHandler<HTMLInputElement> = useCallback((e) => {
        if (e.key === "Enter") {
            if (query.trim().length > 0 && !disableChat) {
                handleSubmitQuery();
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
    }, [chats, currentSearchIndex, query, handleSubmitQuery, disableChat]);

    const handleSelectExample = useCallback((example: string) => {
        setQuery(example);
    }, []);

    const handleClear = useCallback(() => {
        dispatch(HoudiniActions.clear());
        setQuery("");
        setCurrentSearchIndex(undefined);
    }, [dispatch]);

    const handleConfirmSQL = useCallback(async (messageId: number, sql: string, operationType: string) => {
        setExecutingConfirmedId(messageId);
        try {
            const { data, errors } = await executeConfirmedSql({
                variables: {
                    query: sql,
                    operationType: operationType,
                },
            });

            if (errors || !data) {
                toast.error(t('unableToQuery') + " " + (errors?.[0]?.message || t('failedToExecuteSQL')));
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

            // Scroll to bottom
            setTimeout(() => {
                if (scrollContainerRef.current != null) {
                    scrollContainerRef.current.scroll({
                        top: scrollContainerRef.current.scrollHeight,
                        behavior: "smooth",
                    });
                }
            }, 100);

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
    }, [executeConfirmedSql, dispatch, t, scrollContainerRef]);

    const handleCancelSQL = useCallback((messageId: number) => {
        dispatch(HoudiniActions.removeChatMessage(messageId));
        toast.info(t('queryCancelled') || 'Query cancelled');
    }, [dispatch, t]);

    const disableAll = useMemo(() => {
        return models.length === 0 || (!modelAvailable && !currentModel);
    }, [modelAvailable, models.length, currentModel]);

    // Auto-scroll to bottom when chats change or component mounts
    useEffect(() => {
        if (scrollContainerRef.current != null && chats.length > 0) {
            scrollContainerRef.current.scrollTop = scrollContainerRef.current.scrollHeight;
        }
    }, [chats.length]);

    return (
        <InternalPage routes={[InternalRoutes.Chat]} className="h-full">
            <div className="flex flex-col w-full h-full gap-2">
                <AIProvider
                    {...aiState}
                    onClear={handleClear}
                />
                <div className={classNames("flex grow w-full rounded-xl overflow-hidden", {
                    "hidden": disableAll,
                })}>
                    {
                        chats.length === 0
                        ? <div className="flex flex-col justify-center items-center w-full gap-8" data-testid="chat-empty-state-container">
                            {/* {extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-16" />} */}
                            <EmptyState title={t('emptyStateTitle')} description="" icon={<SparklesIcon className="w-16 h-16" data-testid="empty-state-sparkles-icon" />} />
                            <div className="flex flex-wrap justify-center items-center gap-4" data-testid="chat-examples-list">
                                {
                                    examples.map((example, i) => (
                                        <Card key={`chat-${i}`} className="flex flex-col gap-sm w-[250px] h-[120px] p-4 text-sm cursor-pointer hover:opacity-80 transition-all"
                                            onClick={() => handleSelectExample(example.description)}>
                                            {example.icon}
                                            {example.description}
                                        </Card>
                                    ))
                                }
                            </div>
                        </div>
                        : <div className="h-full w-full py-8 max-h-[calc(75vh-25px)] overflow-y-auto" ref={scrollContainerRef}>
                            <div className="flex justify-center w-full h-full">
                                <div className="flex w-full flex-col gap-2">
                                    {
                                        chats.map((chat, i) => {
                                            if (chat.Type === "message" || chat.Type === "text") {
                                                return <div key={`chat-${i}`} className={classNames("flex gap-lg overflow-hidden break-words leading-6 shrink-0 relative", {
                                                    "self-end ml-3": chat.isUserInput,
                                                    "self-start": !chat.isUserInput,
                                                })} data-testid={chat.isUserInput ? "user-message" : "system-message"}>
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput
                                                        ? extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8" />
                                                        : <div className="pl-4" />}
                                                    {chat.isUserInput ? (
                                                        <p className={classNames("py-2 rounded-xl whitespace-pre-wrap bg-neutral-600/5 dark:bg-[#2C2F33] px-4", {
                                                            "animate-fade-in": chat.isStreaming,
                                                        })} data-input-message="user">
                                                            {chat.Text}
                                                            {chat.isStreaming && <span className="inline-block w-2 h-4 ml-1 bg-current animate-pulse" />}
                                                        </p>
                                                    ) : (
                                                        <div className={classNames("py-2 rounded-xl markdown-content ml-12", {
                                                            "animate-fade-in": chat.isStreaming,
                                                        })} data-input-message="system">
                                                            <ReactMarkdown
                                                                components={{
                                                                    p: ({node, ...props}) => <p className="mb-2 last:mb-0" {...props} />,
                                                                    strong: ({node, ...props}) => <strong className="font-semibold" {...props} />,
                                                                    ul: ({node, ...props}) => <ul className="list-disc list-inside mb-2 space-y-1" {...props} />,
                                                                    ol: ({node, ...props}) => <ol className="list-decimal list-inside mb-2 space-y-1" {...props} />,
                                                                    li: ({node, ...props}) => <li className="ml-2" {...props} />,
                                                                    h1: ({node, ...props}) => <h1 className="text-xl font-bold mb-2 mt-4 first:mt-0" {...props} />,
                                                                    h2: ({node, ...props}) => <h2 className="text-lg font-semibold mb-2 mt-3 first:mt-0" {...props} />,
                                                                    h3: ({node, ...props}) => <h3 className="text-md font-semibold mb-1 mt-2 first:mt-0" {...props} />,
                                                                    code: ({node, ...props}) => {
                                                                        const isInline = !String(props.className || '').includes('language-');
                                                                        return isInline
                                                                            ? <code className="bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded text-sm" {...props} />
                                                                            : <code className="block bg-neutral-100 dark:bg-neutral-800 p-2 rounded my-2 text-sm overflow-x-auto" {...props} />;
                                                                    },
                                                                    blockquote: ({node, ...props}) => <blockquote className="border-l-4 border-neutral-300 dark:border-neutral-700 pl-4 my-2 italic" {...props} />,
                                                                }}
                                                            >
                                                                {chat.Text}
                                                            </ReactMarkdown>
                                                            {chat.isStreaming && <span className="inline-block w-2 h-4 ml-1 bg-current animate-pulse" />}
                                                        </div>
                                                    )}
                                                </div>
                                            } else if (chat.Type === "error") {
                                                return (
                                                    <div key={`chat-${i}`} className="flex gap-lg overflow-hidden break-words leading-6 shrink-0 self-start pt-6 relative" data-testid="error-message">
                                                        {!chat.isUserInput && chats[i-1]?.isUserInput
                                                            ? extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8" />
                                                            : <div className="pl-4" />}
                                                        <ErrorState error={chat.Text.replace(/^ERROR:\s*/i, "")} />
                                                    </div>
                                                );
                                            } else if (isEEFeatureEnabled('dataVisualization') && (chat.Type === "sql:pie-chart" || chat.Type === "sql:line-chart")) {
                                                return <div key={`chat-${i}`} className="flex items-center self-start relative" data-testid="visual-message">
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput && (extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8" />)}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:pie-chart" && PieChart && <PieChart columns={chat.Result?.Columns?.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} />}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:line-chart" && LineChart && <LineChart columns={chat.Result?.Columns?.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} />}
                                                </div>
                                            } else if (chat.RequiresConfirmation) {
                                                // Show confirmation UI inline
                                                const isExecuting = executingConfirmedId === chat.id;
                                                const showQuery = showQueryForId === chat.id;

                                                return <div key={`chat-${i}`} className="flex gap-lg w-full pt-4 relative" data-testid="confirmation-message">
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput
                                                        ? (extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8" />)
                                                        : <div className="pl-4" />}
                                                    <div className="flex flex-col gap-3 w-[calc(100%-50px)]">
                                                        <Alert className="w-full">
                                                            <SparklesIcon className="w-4 h-4" />
                                                            <AlertTitle>{t('confirmExecutionTitle') || 'Confirm Execution'}</AlertTitle>
                                                            <AlertDescription>
                                                                {t('confirmExecutionDescription') || 'This operation will modify your database. Review and confirm to proceed.'}
                                                            </AlertDescription>
                                                        </Alert>

                                                        {/* SQL Query Toggle */}
                                                        <Button
                                                            variant="ghost"
                                                            size="sm"
                                                            onClick={() => setShowQueryForId(showQuery ? null : (chat.id ?? null))}
                                                            className="w-fit"
                                                        >
                                                            <CodeBracketIcon className="w-4 h-4 mr-2" />
                                                            {showQuery ? t('hideQuery') || 'Hide Query' : t('showQuery') || 'Show Query'}
                                                        </Button>

                                                        {/* SQL Query Display */}
                                                        {showQuery && (
                                                            <div className="h-[200px] w-full rounded-lg overflow-hidden">
                                                                <CodeEditor value={chat.Text} language="sql" />
                                                            </div>
                                                        )}

                                                        {/* Action Buttons */}
                                                        <div className="flex gap-2">
                                                            <Button
                                                                variant="outline"
                                                                onClick={() => chat.id && handleCancelSQL(chat.id)}
                                                                disabled={isExecuting}
                                                                size="sm"
                                                            >
                                                                {t('no') || 'No'}
                                                            </Button>
                                                            <Button
                                                                onClick={() => chat.id && handleConfirmSQL(chat.id, chat.Text, chat.Type)}
                                                                disabled={isExecuting}
                                                                size="sm"
                                                            >
                                                                {isExecuting ? (t('executing') || 'Executing...') : (t('yes') || 'Yes')}
                                                            </Button>
                                                        </div>
                                                    </div>
                                                </div>
                                            }
                                            return <div key={`chat-${i}`} className="flex gap-lg w-full pt-4 relative" data-testid="table-message">
                                                {!chat.isUserInput && chats[i-1]?.isUserInput && (extensions.Logo ?? <img src={logoImage} alt="clidey logo" className="w-auto h-8" />)}
                                                <TablePreview type={chat.Type} text={chat.Text} data={chat.Result} />
                                            </div>
                                        })
                                    }
                                    { loading &&  <div className="flex w-full mt-4">
                                        <Loading loadingText={loadingPhraseRef.current} size="sm" />
                                    </div> }
                                </div>
                            </div>
                        </div>
                    }
                </div>
                {
                    (models.length === 0 || (!modelAvailable && !currentModel)) &&
                    <EmptyState title={t('noModelTitle')} description={t('noModelDescription')} icon={<SparklesIcon className="w-16 h-16" data-testid="empty-state-sparkles-icon" />} />
                }
                <div className={classNames("flex justify-between items-center gap-2", {
                    "opacity-80": disableChat,
                    "opacity-10": disableAll,
                })}>
                    <Input
                        value={query}
                        onChange={e => setQuery(e.target.value)}
                        placeholder={t('placeholder')}
                        onSubmit={handleSubmitQuery}
                        disabled={disableAll}
                        onKeyUp={handleKeyUp}
                        autoComplete="off"
                        data-testid="chat-input"
                    />
                    <Button tabIndex={0} onClick={loading ? undefined : handleSubmitQuery} className={cn("rounded-full", {
                        "opacity-50": loading,
                    })} disabled={disableChat} variant={disableChat ? "secondary" : undefined} data-testid="icon-button" aria-label={t('sendMessage')}>
                        <ArrowUpCircleIcon className="w-8 h-8" aria-hidden="true" />
                    </Button>
                </div>
            </div>
        </InternalPage>
    )
}
