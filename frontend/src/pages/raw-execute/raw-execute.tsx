/*
 * Copyright 2025 Clidey, Inc.
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

import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
    Badge,
    Button,
    Card,
    cn,
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
    EmptyState,
    formatDate,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    Separator,
    Sheet,
    SheetContent,
    SheetFooter,
    Tabs,
    TabsContent,
    TabsList,
    TabsTrigger,
    toast
} from "@clidey/ux";
import { DatabaseType, RowsResult } from '@graphql';
import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import {
    ChangeEvent,
    cloneElement,
    FC,
    ReactElement,
    Suspense,
    useCallback,
    useEffect,
    useMemo,
    useRef,
    useState
} from "react";
import { useLocation } from "react-router-dom";
import { v4 } from "uuid";
import { AIProvider, useAI } from "../../components/ai";
import { CodeEditor } from "../../components/editor";
import { ErrorState } from "../../components/error-state";
import {
    ArrowPathIcon,
    CheckCircleIcon,
    CircleStackIcon,
    ClipboardDocumentIcon,
    ClockIcon,
    EllipsisHorizontalIcon,
    EllipsisVerticalIcon,
    PencilIcon,
    PlayIcon,
    PlusCircleIcon,
    XCircleIcon,
    XMarkIcon
} from "../../components/heroicons";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Tip } from "../../components/tip";
import { InternalRoutes } from "../../config/routes";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { ScratchpadActions } from "../../store/scratchpad";
import { isEEFeatureEnabled, loadEEModule } from "../../utils/ee-loader";
import { isDesktopApp } from "../../utils/external-links";
import { IPluginProps, QueryView } from "./query-view";

type EEExports = {
    plugins: any[];
    ActionOptions: Record<string, string>;
    ActionOptionIcons: Record<string, ReactElement>;
};

type IRawExecuteCellProps = {
    cellId: string;
    onAdd: (cellId: string) => void;
    onDelete?: (cellId: string) => void;
    showTools?: boolean;
    cellData?: any;
}

enum ActionOptions {
    Query="Query",
}

export const ActionOptionIcons: Record<string, ReactElement> = {
    [ActionOptions.Query]: <CircleStackIcon className="w-4 h-4" />,
}

// Lightweight, dependency-free SQL highlighter

// Safe SQL syntax highlighter component that renders React elements
const SQLHighlighter: FC<{ code: string }> = ({ code }) => {
    const keywords = [
        'SELECT','FROM','WHERE','AND','OR','NOT','INSERT','INTO','VALUES','UPDATE','SET','DELETE','CREATE','TABLE','PRIMARY','KEY','FOREIGN','REFERENCES','DROP','ALTER','ADD','COLUMN','JOIN','LEFT','RIGHT','FULL','OUTER','INNER','ON','GROUP','BY','ORDER','HAVING','LIMIT','OFFSET','DISTINCT','AS','IN','IS','NULL','LIKE','BETWEEN','UNION','ALL','EXISTS','CASE','WHEN','THEN','ELSE','END','WITH','EXPLAIN','DESCRIBE','SHOW'
    ];

    const parseSQL = (sql: string): React.ReactNode[] => {
        const tokens: Array<{ type: string; value: string; className?: string }> = [];
        let remaining = sql;
        let position = 0;

        while (remaining.length > 0) {
            let matched = false;

            // Block comments
            const blockCommentMatch = remaining.match(/^\/\*[\s\S]*?\*\//);
            if (blockCommentMatch) {
                tokens.push({
                    type: 'comment',
                    value: blockCommentMatch[0],
                    className: 'text-muted-foreground'
                });
                remaining = remaining.slice(blockCommentMatch[0].length);
                matched = true;
            }
            // Line comments
            else if (remaining.match(/^--/)) {
                const lineEnd = remaining.indexOf('\n');
                const comment = lineEnd === -1 ? remaining : remaining.slice(0, lineEnd);
                tokens.push({
                    type: 'comment',
                    value: comment,
                    className: 'text-muted-foreground'
                });
                remaining = remaining.slice(comment.length);
                matched = true;
            }
            // Single quoted strings
            else if (remaining.startsWith("'")) {
                let stringValue = "'";
                let i = 1;
                while (i < remaining.length) {
                    if (remaining[i] === "'" && remaining[i-1] !== '\\') {
                        stringValue += "'";
                        break;
                    }
                    stringValue += remaining[i];
                    i++;
                }
                tokens.push({
                    type: 'string',
                    value: stringValue,
                    className: 'text-amber-600 dark:text-amber-400'
                });
                remaining = remaining.slice(stringValue.length);
                matched = true;
            }
            // Double quoted strings
            else if (remaining.startsWith('"')) {
                let stringValue = '"';
                let i = 1;
                while (i < remaining.length) {
                    if (remaining[i] === '"' && remaining[i-1] !== '\\') {
                        stringValue += '"';
                        break;
                    }
                    stringValue += remaining[i];
                    i++;
                }
                tokens.push({
                    type: 'string',
                    value: stringValue,
                    className: 'text-amber-600 dark:text-amber-400'
                });
                remaining = remaining.slice(stringValue.length);
                matched = true;
            }
            // Numbers
            else if (remaining.match(/^\d+(?:\.\d+)?/)) {
                const numberMatch = remaining.match(/^\d+(?:\.\d+)?/);
                if (numberMatch) {
                    tokens.push({
                        type: 'number',
                        value: numberMatch[0],
                        className: 'text-blue-600 dark:text-blue-400'
                    });
                    remaining = remaining.slice(numberMatch[0].length);
                    matched = true;
                }
            }
            // Keywords and identifiers
            else if (remaining.match(/^[a-zA-Z_][a-zA-Z0-9_]*/)) {
                const wordMatch = remaining.match(/^[a-zA-Z_][a-zA-Z0-9_]*/);
                if (wordMatch) {
                    const word = wordMatch[0];
                    const upperWord = word.toUpperCase();
                    if (keywords.includes(upperWord)) {
                        tokens.push({
                            type: 'keyword',
                            value: word,
                            className: 'text-purple-700 dark:text-purple-400 font-medium'
                        });
                    } else {
                        tokens.push({
                            type: 'identifier',
                            value: word
                        });
                    }
                    remaining = remaining.slice(word.length);
                    matched = true;
                }
            }
            // Whitespace and other characters
            else {
                const char = remaining[0];
                tokens.push({
                    type: 'text',
                    value: char
                });
                remaining = remaining.slice(1);
                matched = true;
            }

            if (!matched) {
                // Fallback to prevent infinite loop
                const char = remaining[0];
                tokens.push({
                    type: 'text',
                    value: char
                });
                remaining = remaining.slice(1);
            }
        }

        return tokens.map((token, index) => {
            if (token.className) {
                return (
                    <span key={index} className={token.className}>
                        {token.value}
                    </span>
                );
            }
            return token.value;
        });
    };

    return <>{parseSQL(code)}</>;
};

const CopyButton: FC<{ text: string }> = ({text}) => {
    const [copied, setCopied] = useState(false);

    const handleCopyToClipboard = useCallback(() => {
        navigator.clipboard.writeText(text).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        });
    }, [text]);

    return (
        <Button
            size="icon"
            variant="secondary"
            className="border border-input"
            onClick={handleCopyToClipboard}
            title={copied ? "Copied!" : "Copy to clipboard"}
            type="button"
            data-testid="copy-to-clipboard-button"
        >
            {copied ? <CheckCircleIcon className="w-4 h-4"/> : <ClipboardDocumentIcon className="w-4 h-4"/>}
        </Button>
    );
}

const RawExecuteCell: FC<IRawExecuteCellProps> = ({ cellId, onAdd, onDelete, showTools, cellData }) => {
    const dispatch = useAppDispatch();
    const [mode, setMode] = useState<string>(cellData?.mode || ActionOptions.Query);
    const [code, setCode] = useState(cellData?.code || "");
    const [submittedCode, setSubmittedCode] = useState("");
    const [history, setHistory] = useState<{id: string, item: string, status: boolean, date: Date}[]>(() => {
        if (!cellData?.history) return [];
        // Ensure all dates are proper Date objects
        return cellData.history.map((item: any) => ({
            ...item,
            date: item.date instanceof Date ? item.date : new Date(item.date)
        }));
    });
    const current = useAppSelector(state => state.auth.current);
    const handleExecute = useRef<(code: string) => Promise<any>>(() => Promise.resolve());
    const [historyOpen, setHistoryOpen] = useState(false);
    const [error, setError] = useState<Error | null>(null);
    const [loading, setLoading] = useState(false);
    const [rows, setRows] = useState<RowsResult | null>(null);
    const { modelType } = useAI();    
    const [editorHeight, setEditorHeight] = useState(150);
    const [resultsHeight, setResultsHeight] = useState(250);
    const [isResizing, setIsResizing] = useState(false);
    const [isResizingResults, setIsResizingResults] = useState(false);
    const [allowResultsResize, setAllowResultsResize] = useState(false);
    const resultsContainerRef = useRef<HTMLDivElement | null>(null);

    // Sync local state with Redux state
    useEffect(() => {
        if (cellData) {
            setCode(cellData.code || "");
            setMode(cellData.mode || ActionOptions.Query);
            // Ensure all dates are proper Date objects
            const processedHistory = (cellData.history || []).map((item: any) => ({
                ...item,
                date: item.date instanceof Date ? item.date : new Date(item.date)
            }));
            setHistory(processedHistory);
        }
    }, [cellData]);

    // Update Redux when code changes (but not on initial load)
    const isInitialLoad = useRef(true);
    useEffect(() => {
        if (isInitialLoad.current) {
            isInitialLoad.current = false;
            return;
        }
        if (cellData && code !== cellData.code) {
            dispatch(ScratchpadActions.updateCellCode({ cellId, code }));
        }
    }, [code, cellId, dispatch, cellData]);

    // Update Redux when mode changes (but not on initial load)
    const isInitialModeLoad = useRef(true);
    useEffect(() => {
        if (isInitialModeLoad.current) {
            isInitialModeLoad.current = false;
            return;
        }
        if (cellData && mode !== cellData.mode) {
            dispatch(ScratchpadActions.updateCellMode({ cellId, mode }));
        }
    }, [mode, cellId, dispatch, cellData]);

    // State for all plugins, action options, and action option icons (not just EE)
    const [allPlugins, setAllPlugins] = useState<{ type: string, component: FC<IPluginProps> }[]>([
        {
            type: ActionOptions.Query,
            component: QueryView,
        },
    ]);
    const [allActionOptions, setAllActionOptions] = useState<Record<string, string>>({ ...ActionOptions });
    const [allActionOptionIcons, setAllActionOptionIcons] = useState<Record<string, ReactElement>>({ ...ActionOptionIcons });

    // Load EE module on mount and merge with base
    useEffect(() => {
        let mounted = true;
        loadEEModule<EEExports>(
            () => import('@ee/pages/raw-execute/index'),
            { plugins: [], ActionOptions: {}, ActionOptionIcons: {} }
        ).then((mod) => {
            if (mod && mounted) {
                const { default: defaultMod } = mod as any;
                if (defaultMod == null || defaultMod.plugins == null) {
                    return;
                }
                // Merge plugins
                setAllPlugins(prev => [
                    ...prev,
                    ...(defaultMod.plugins || []).map((p: any) => ({
                        type: p.type,
                        component: p.component,
                    })),
                ]);
                // Merge action options
                setAllActionOptions(prev => ({
                    ...prev,
                    ...(defaultMod.ActionOptions || {})
                }));
                // Merge action option icons
                setAllActionOptionIcons(prev => ({
                    ...prev,
                    ...(defaultMod.ActionOptionIcons || {})
                }));
            }
        });
        return () => { mounted = false; }
    }, []);

    const handleRawExecute = useCallback((historyCode?: string) => {
        if (current == null) {
            setLoading(false);
            return;
        }
        const currentCode = historyCode ?? code;
        const historyItem = {id: v4(), item: currentCode, status: false, date: new Date()};
        setSubmittedCode(currentCode);
        setError(null);
        setLoading(true);
        handleExecute.current(currentCode).then((data) => {
            historyItem.status = true;
            setRows(data);
        }).catch((err) => {
            setError(err);
        }).finally(() => {
            setLoading(false);
            setHistory(h => [historyItem, ...h]);
            dispatch(ScratchpadActions.addCellHistory({ 
                cellId, 
                item: currentCode, 
                status: historyItem.status 
            }));
        });
    }, [code, current, mode, allActionOptions, handleExecute, cellId, dispatch]);

    const handleAdd = useCallback(() => {
        onAdd(cellId);
    }, [cellId, onAdd]);

    const handleDelete = useCallback(() => {
        onDelete?.(cellId);
    }, [cellId, onDelete]);

    const handleEditorResize = useCallback((e: React.MouseEvent) => {
        e.preventDefault();
        setIsResizing(true);
        
        const startY = e.clientY;
        const startHeight = editorHeight;
        
        const handleMouseMove = (e: MouseEvent) => {
            const deltaY = e.clientY - startY;
            const newHeight = Math.max(100, Math.min(500, startHeight + deltaY));
            setEditorHeight(newHeight);
        };
        
        const handleMouseUp = () => {
            setIsResizing(false);
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', handleMouseUp);
        };
        
        document.addEventListener('mousemove', handleMouseMove);
        document.addEventListener('mouseup', handleMouseUp);
    }, [editorHeight]);

    const handleResultsResize = useCallback((e: React.MouseEvent) => {
        e.preventDefault();
        setIsResizingResults(true);
        
        const startY = e.clientY;
        const startHeight = resultsHeight;
        
        const handleMouseMove = (e: MouseEvent) => {
            const deltaY = e.clientY - startY;
            const newHeight = Math.max(100, Math.min(800, startHeight + deltaY));
            setResultsHeight(newHeight);
        };
        
        const handleMouseUp = () => {
            setIsResizingResults(false);
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', handleMouseUp);
        };
        
        document.addEventListener('mousemove', handleMouseMove);
        document.addEventListener('mouseup', handleMouseUp);
    }, [resultsHeight]);

    // Use all plugins
    const output = useMemo(() => {
        const selectedPlugin = allPlugins.find((p: any) => p.type === mode);
        if (selectedPlugin?.component == null || current == null) {
            return null;
        }
        const Component = selectedPlugin.component as FC<IPluginProps>;
        return (
            <div className="flex flex-col mt-4 w-full group relative">
                <div
                    className={cn("h-2 cursor-row-resize transition-all duration-200 group-hover:border-b border-muted", {
                        "hidden": rows == null || !allowResultsResize,
                    })}
                    onMouseDown={handleResultsResize}
                    data-testid="output-resize-button"
                >
                    <div className="absolute bottom-0 left-1/2 -translate-x-1/2 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-200 z-10">
                        <EllipsisHorizontalIcon className="w-4 h-4 text-gray-400" />
                    </div>
                </div>
                <div 
                    ref={resultsContainerRef}
                    className={cn({
                        "overflow-auto": allowResultsResize,
                        "overflow-visible": !allowResultsResize,
                    })}
                    style={{
                        minHeight: "fit-content",
                        height: allowResultsResize ? `${resultsHeight}px` : "auto",
                    }}
                >
                    <Suspense fallback={<Loading />}>
                        <Component code={code} handleExecuteRef={handleExecute} modelType={modelType?.modelType || ''}
                                   schema={current.Database} token={modelType?.token} providerId={current.Id}/>
                    </Suspense>
                </div>
            </div>
        );
    }, [mode, allActionOptions, allPlugins, code, modelType, current, resultsHeight, isResizingResults, handleResultsResize, rows, allowResultsResize]);

    // Measure results on first mount to fit content
    useEffect(() => {
        if (allowResultsResize) {
            return;
        }
        const raf = requestAnimationFrame(() => {
            const el = resultsContainerRef.current;
            if (el == null) {
                return;
            }
            const measured = el.scrollHeight;
            if (measured > 0) {
                setResultsHeight(Math.min(800, Math.max(100, measured)));
                setAllowResultsResize(true);
            }
        });
        return () => cancelAnimationFrame(raf);
    }, [allowResultsResize]);

    // Re-measure whenever results change (e.g., query re-run) to fit new content
    useEffect(() => {
        if (rows == null) {
            return;
        }
        // Temporarily allow content to define height, then measure and lock
        setAllowResultsResize(false);
        const raf = requestAnimationFrame(() => {
            const el = resultsContainerRef.current;
            if (el == null) {
                setAllowResultsResize(true);
                return;
            }
            const measured = el.scrollHeight;
            if (measured > 0) {
                setResultsHeight(Math.min(800, Math.max(100, measured)));
            }
            setAllowResultsResize(true);
        });
        return () => cancelAnimationFrame(raf);
    }, [rows, mode]);

    const isAnalyzeAvailable = useMemo(() => {
        if (!isEEFeatureEnabled('analyzeView')) {
            return false;
        }
        switch(current?.Type) {
            case DatabaseType.Postgres:
                return !!allActionOptions.Analyze;
        }
        return false;
    }, [current?.Type, allActionOptions]);

    // Merge icons from all sources
    const mergedActionOptionIcons = useMemo(() => {
        return {
            ...ActionOptionIcons,
            ...allActionOptionIcons,
        };
    }, [allActionOptionIcons]);

    const actionOptions = useMemo(() => {
        return Object.keys(allActionOptions);
    }, [allActionOptions]);

    return (
        <div className="flex flex-col grow group/cell relative">
            <div className="relative">
                <div 
                    className="flex grow border border-gray-200 rounded-md overflow-hidden dark:bg-white/10 dark:border-white/5"
                    style={{ height: `${editorHeight}px` }}
                >
                    <CodeEditor language="sql" value={code} setValue={setCode} onRun={(c) => handleRawExecute(c)} />
                </div>
                <div 
                    className="h-2 cursor-row-resize transition-all duration-200 relative group"
                    onMouseDown={handleEditorResize}
                    style={{ cursor: isResizing ? 'row-resize' : 'row-resize' }}
                    data-testid="editor-resize-button"
                >
                    <div className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-200 z-10">
                        <EllipsisHorizontalIcon className="w-4 h-4 text-gray-400" />
                    </div>
                </div>
                <div className="absolute top-1 right-1 z-10" data-testid="scratchpad-cell-options">
                    <DropdownMenu>
                        <DropdownMenuTrigger>
                            <Button
                                variant="ghost"
                                className="flex justify-center items-center"
                                data-testid="icon-button">
                                <EllipsisVerticalIcon className="w-4 h-4"/>
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => setCode("")}>
                                <ArrowPathIcon className="w-4 h-4"/>
                                Clear
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => {
                                navigator.clipboard.writeText(code).then(() => {
                                    toast.success("Code copied to clipboard");
                                });
                            }}>
                                <ClipboardDocumentIcon className="w-4 h-4"/>
                                Copy Code
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => setHistoryOpen(true)}>
                                <ClockIcon className="w-4 h-4"/>
                                Query History
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
                <div className={classNames("absolute -bottom-3 z-20 flex justify-between px-3 pr-8 w-full opacity-0 transition-all duration-500 group-hover/cell:opacity-100 pointer-events-none", {
                    "opacity-100": showTools,
                })}>
                    <div className="flex gap-sm pointer-events-auto">
                        {actionOptions.length > 1 && <Select
                            value={mode}
                            onValueChange={(val) => setMode(val as string)}
                        >
                            <SelectTrigger style={{
                                background: "var(--secondary)",
                            }}>
                                <div className="flex items-center gap-sm w-full">
                                    {mergedActionOptionIcons[mode] && cloneElement(mergedActionOptionIcons[mode], { className: "w-4 h-4" })}
                                    <span>{mode}</span>
                                </div>
                            </SelectTrigger>
                            <SelectContent>
                                {actionOptions.map((item) => (
                                    <SelectItem
                                        key={item}
                                        value={item}
                                        className={classNames({
                                            "hidden": !isAnalyzeAvailable && (item === allActionOptions.Analyze),
                                        })}
                                    >
                                        <div className="flex items-center gap-2">
                                            {mergedActionOptionIcons[item] && cloneElement(mergedActionOptionIcons[item], { className: "w-4 h-4" })}
                                            <span>{item}</span>
                                        </div>
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>}
                        <Tip>
                            <Button onClick={handleAdd} data-testid="add-cell-button" variant="secondary"
                                    className="border border-input">
                                <PlusCircleIcon className="w-4 h-4" />
                            </Button>
                                <p>Add a new cell</p>
                        </Tip>
                        <Tip>
                            <Button onClick={() => setCode("")} data-testid="clear-cell-button" variant="secondary"
                                    className="border border-input">
                                <ArrowPathIcon className="w-4 h-4" />
                            </Button>
                            <p>Clear the editor</p>
                        </Tip>
                        {
                            onDelete != null &&
                            <Tip>
                                <Button variant="destructive" onClick={handleDelete} data-testid="delete-cell-button"
                                        className="border border-input bg-white hover:bg-white/95">
                                    <XCircleIcon className="w-4 h-4 text-destructive"/>
                                </Button>
                                <p>Delete the cell</p>
                            </Tip>
                        }
                    </div>
                    <div className="flex gap-sm items-center">
                        <Button
                            onClick={() => setHistoryOpen(true)}
                            data-testid="history-button"
                            className={cn("pointer-events-auto border border-input", {
                                "hidden": history.length === 0,
                            })}
                            variant="secondary"
                            disabled={history.length === 0}
                        >
                            <ClockIcon className="w-4 h-4" />
                        </Button>
                        <Button onClick={() => handleRawExecute()} data-testid="query-cell-button"
                                className={cn("pointer-events-auto", {
                            "hidden": code.length === 0,
                        })} disabled={code.length === 0}>
                            {<CheckCircleIcon className="w-4 h-4" />}
                        </Button>
                    </div>
                </div>
            </div>
            {
                error != null &&
                <div className="flex items-center justify-between mt-8" data-testid="cell-error">
                    <ErrorState error={error} />
                </div>
            }
            {loading && <div className="flex justify-center items-center h-full my-16">
                <Loading/>
            </div>}
            {output}
            <Sheet open={historyOpen} onOpenChange={setHistoryOpen}>
                <SheetContent className="min-w-[50vw] max-w-[50vw] p-0">
                    <div className="flex flex-col h-full">
                        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
                            <div className="flex items-center gap-2">
                                <ClockIcon className="w-5 h-5" />
                                <span className="font-semibold text-lg">Query History</span>
                            </div>
                        </div>
                        <div className="flex-1 px-2 py-4 overflow-y-auto">
                            {history.length === 0 ? (
                                <EmptyState title="No history yet" description="Run a query to see your history" icon={<ClockIcon className="w-10 h-10" />} />
                            ) : (
                                <div className="flex flex-col gap-lg p-4">
                                    {history.map(({ id, item, status, date }) => (
                                        <Card className="w-full p-4 relative" key={id}>
                                            <Badge
                                                variant={status ? "default" : "destructive"}
                                                className="absolute top-0 -translate-y-1/2 right-2"
                                            >
                                                {status ? "Success" : "Error"}
                                            </Badge>
                                            <div className="flex flex-col min-h-[60px]">
                                                <div className="pr-12">
                                                    <div className="rounded-md overflow-hidden bg-neutral-50 dark:bg-[#1f1f1f]">
                                                        <pre className="text-xs p-3 whitespace-pre-wrap">
                                                            <code>
                                                                <SQLHighlighter code={item} />
                                                            </code>
                                                        </pre>
                                                    </div>
                                                </div>
                                                <div className="flex gap-sm mt-4 justify-between items-center">
                                                    <div className="text-xs text-muted-foreground">
                                                        {date instanceof Date ? formatDate(date) : 'Invalid date'}
                                                    </div>
                                                    <div className="flex gap-sm items-center">
                                                        <CopyButton text={item} />
                                                        <Button
                                                            size="icon"
                                                            variant="secondary"
                                                            className="border border-input"
                                                            onClick={() => {
                                                                setHistoryOpen(false);
                                                                setCode(item);
                                                            }}
                                                            title="Clone to editor"
                                                            data-testid="clone-to-editor-button"
                                                        >
                                                            <PencilIcon className="w-4 h-4" />
                                                        </Button>
                                                        <Button
                                                            size="icon"
                                                            variant="secondary"
                                                            className="border border-input"
                                                            onClick={() => handleRawExecute(item)}
                                                            title="Run"
                                                            data-testid="run-history-button"
                                                        >
                                                            <PlayIcon className="w-4 h-4" />
                                                        </Button>
                                                    </div>
                                                </div>
                                            </div>
                                        </Card>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>
                    <SheetFooter>
                        {history.length > 0 && (
                            <AlertDialog>
                                <AlertDialogTrigger asChild>
                                    <Button
                                        variant="outline"
                                        data-testid="clear-history-button"
                                        className="self-end"
                                    >
                                        <ArrowPathIcon className="w-4 h-4 mr-1" />
                                        Clear History
                                    </Button>
                                </AlertDialogTrigger>
                                <AlertDialogContent>
                                    <AlertDialogHeader>
                                        <AlertDialogTitle>Clear Query History</AlertDialogTitle>
                                        <AlertDialogDescription>
                                            Are you sure you want to clear all query history? This action cannot be undone.
                                        </AlertDialogDescription>
                                    </AlertDialogHeader>
                                    <AlertDialogFooter>
                                        <AlertDialogCancel data-testid="clear-history-cancel">Cancel</AlertDialogCancel>
                                        <AlertDialogAction asChild>
                                            <Button
                                                variant="destructive"
                                                onClick={() => {
                                                    setHistory([]);
                                                    dispatch(ScratchpadActions.clearCellHistory({ cellId }));
                                                }}
                                                data-testid="clear-history-confirm"
                                            >
                                                Clear History
                                            </Button>
                                        </AlertDialogAction>
                                    </AlertDialogFooter>
                                </AlertDialogContent>
                            </AlertDialog>
                        )}
                    </SheetFooter>
                </SheetContent>
            </Sheet>
        </div>
    )
}

const RawExecuteSubPage: FC<{ 
    pageId: string; 
    cellIds: string[]; 
    cells: Record<string, any>;
}> = ({ pageId, cellIds = [], cells = {} }) => {
    // Ensure cellIds is always an array
    const safeCellIds = cellIds || [];
    const dispatch = useAppDispatch();
    
    const handleAdd = useCallback((id: string) => {
        dispatch(ScratchpadActions.addCell({ pageId, afterCellId: id }));
    }, [dispatch, pageId]);

    const handleDelete = useCallback((cellId: string) => {
        if (safeCellIds.length <= 1) {
            return;
        }
        dispatch(ScratchpadActions.deleteCell({ pageId, cellId }));
    }, [safeCellIds.length, dispatch, pageId]);


    return (
        <div className="flex justify-center items-center w-full">
            <div className="w-full flex flex-col gap-2">
                {
                    safeCellIds.map((cellId, index) => (
                        <div key={cellId} data-testid={`cell-${index}`}>
                            {index > 0 && <Separator className="my-4" />}
                            <RawExecuteCell 
                                key={cellId} 
                                cellId={cellId} 
                                onAdd={handleAdd} 
                                onDelete={safeCellIds.length <= 1 ? undefined : handleDelete}
                                showTools={safeCellIds.length === 1}
                                cellData={cells[cellId]}
                            />
                        </div>
                    ))
                }
            </div>
        </div>
    )
}

const EditableInput: FC<{ page: Page; setValue: (value: string) => void }> = ({ page, setValue }) => {
    const [currentContent, setCurrentContent] = useState(page.name);
    const [isEditing, setIsEditing] = useState(false);
    const inputRef = useRef<HTMLInputElement | null>(null);
  
    const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
      setCurrentContent(event.target.value);
    };
  
    const handleBlur = () => {
      if (currentContent !== page.name) {
        setValue(currentContent);
      }
      setIsEditing(false);
    };
  
    const handleDoubleClick = () => {
      setIsEditing(true);
      setTimeout(() => inputRef.current?.focus(), 0);
    };
  
    return (
      <div className="inline-block" onClick={() => inputRef.current?.focus()} onDoubleClick={handleDoubleClick}>
        {isEditing ? (
          <input
            ref={inputRef}
            type="text"
            value={currentContent}
            onChange={handleChange}
            onBlur={handleBlur}
            autoFocus
            className="w-auto max-w-[40ch] border-b border-gray-400 focus:outline-none focus:border-blue-500 transition-colors text-inherit"
          />
        ) : (
          <span className="text-sm text-nowrap">
            {currentContent || "Double click to edit"}
          </span>
        )}
      </div>
    );
};


type Page = {
    id: string;
    name: string;
}

export const RawExecutePage: FC = () => {
    const location = useLocation();
    const dispatch = useAppDispatch();
    const { pages = [], cells = {}, activePageId } = useAppSelector(state => state.scratchpad);
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [pageToDelete, setPageToDelete] = useState<string | null>(null);

    const aiState = useAI();

    // Initialize scratchpad and ensure all pages have cells
    const hasInitialized = useRef(false);
    useEffect(() => {
        console.log('Initialization check:', { pagesLength: pages.length, hasInitialized: hasInitialized.current });
        if (!hasInitialized.current) {
            console.log('Ensuring scratchpad has proper structure...');
            hasInitialized.current = true;
            dispatch(ScratchpadActions.ensurePagesHaveCells());
        }
    }, [dispatch]);

    // Handle navigation state from chat
    const hasProcessedInitialQuery = useRef(false);
    useEffect(() => {
        const state = location.state as { initialQuery?: string; targetPage?: string } | null;
        if (state?.initialQuery && !hasProcessedInitialQuery.current && activePageId) {
            hasProcessedInitialQuery.current = true;
            if (state.targetPage === "new") {
                // Page was already created with the query in chat.tsx, no need to add anything
                // The new page should already be active and contain the query
            } else if (state.targetPage && state.targetPage !== "new") {
                // Add to specific existing page and set it as active
                dispatch(ScratchpadActions.addCellToPageAndActivate({ 
                    pageId: state.targetPage, 
                    initialQuery: state.initialQuery 
                }));
            } else {
                // Add to current page
                dispatch(ScratchpadActions.addCell({ 
                    pageId: activePageId, 
                    initialQuery: state.initialQuery 
                }));
            }
        }
    }, [location.state, activePageId, dispatch, pages.length]);

    // Handle target page highlighting (when no initial query, just highlighting)
    useEffect(() => {
        const state = location.state as { initialQuery?: string; targetPage?: string } | null;
        if (state?.targetPage && !state.initialQuery && state.targetPage !== "new") {
            // Just highlight the target page without adding content
            dispatch(ScratchpadActions.setActivePage({ pageId: state.targetPage }));
        }
    }, [location.state, dispatch]);

    // Listen for menu event to create new Scratchpad page
    useEffect(() => {
        if (!isDesktopApp()) return;

        const handleNewScratchpadPage = () => {
            dispatch(ScratchpadActions.addPage({ name: `Page ${pages.length + 1}` }));
        };

        window.addEventListener('menu:new-scratchpad-page', handleNewScratchpadPage);

        return () => {
            window.removeEventListener('menu:new-scratchpad-page', handleNewScratchpadPage);
        };
    }, [dispatch, pages.length]);

    const handleAdd = useCallback(() => {
        dispatch(ScratchpadActions.addPage({ name: `Page ${pages.length + 1}` }));
    }, [dispatch, pages.length]);

    const handleSelect = useCallback((pageId: string) => {
        dispatch(ScratchpadActions.setActivePage({ pageId }));
    }, [dispatch]);

    const handleDelete = useCallback((pageId: string) => {
        dispatch(ScratchpadActions.deletePage({ pageId }));
    }, [dispatch]);

    const promptDelete = useCallback((pageId: string) => {
        setPageToDelete(pageId);
        setConfirmOpen(true);
    }, []);

    const handleUpdatePageName = useCallback((changedPage: Page, newName: string) => {
        dispatch(ScratchpadActions.updatePageName({ pageId: changedPage.id, name: newName }));
    }, [dispatch]);


    return (
        <InternalPage routes={[InternalRoutes.RawExecute]}>
            <div className="flex flex-col w-full gap-2" data-testid="raw-execute-page">
                {isEEFeatureEnabled('analyzeView') && <AIProvider 
                    {...aiState}
                    disableNewChat={true}
                />}
                <div className="flex justify-center items-center w-full mt-4">
                    <div className="w-full flex flex-col gap-4">
                        <div className="flex justify-between items-center">
                            <Tabs className="w-full h-full" value={activePageId || ""}>
                                <div className="flex gap-sm w-full justify-between">
                                    <TabsList className="flex flex-wrap gap-sm" data-testid="page-tabs">
                                        {
                                            pages.map((page, index) => (
                                                    <TabsTrigger value={page.id} key={page.id}
                                                                 onClick={() => handleSelect(page.id)}
                                                                 data-testid={`page-tab-${index}`}>
                                                        <div className="flex items-center gap-2 group">
                                                            <EditableInput page={page} setValue={(newName) => handleUpdatePageName(page, newName)} />
                                                            <button
                                                                type="button"
                                                                title="Delete page"
                                                                onClick={(e) => {
                                                                    e.preventDefault();
                                                                    e.stopPropagation();
                                                                    promptDelete(page.id);
                                                                }}
                                                                className={cn("opacity-0 group-hover:opacity-100 transition-opacity", {
                                                                    "hidden": pages.length <= 1,
                                                                })}
                                                                aria-label="Delete page"
                                                                data-testid={`delete-page-tab-${index}`}
                                                            >
                                                                <XMarkIcon className="w-3 h-3" />
                                                            </button>
                                                        </div>
                                                    </TabsTrigger>
                                            ))
                                        }
                                        <TabsTrigger value="add" onClick={handleAdd} data-testid="add-page-button">
                                            <PlusCircleIcon className="w-4 h-4" />
                                        </TabsTrigger>
                                    </TabsList>
                                </div>
                                <TabsContent value={activePageId || ""} className="h-full w-full mt-4">
                                    <AnimatePresence mode="wait">
                                        {pages.map((page) => (
                                            <motion.div
                                                key={page.id}
                                                className={classNames({
                                                    "hidden": page.id !== activePageId,
                                                })}
                                                // todo this animation
                                                initial={{ opacity: 0, y: 10 }}
                                                animate={{ opacity: 1, y: 0 }}
                                                exit={{ opacity: 0, y: -10 }}
                                            >
                                                <RawExecuteSubPage 
                                                    key={page.id} 
                                                    pageId={page.id}
                                                    cellIds={page.cellIds || []}
                                                    cells={cells}
                                                />
                                            </motion.div>
                                        ))}
                                    </AnimatePresence>
                                </TabsContent>
                            </Tabs>
                        </div>
                    </div>
                </div>
            </div>
            <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    {`Delete ${pages.find(p => p.id === pageToDelete)?.name ?? 'page'}?`}
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    {`This action cannot be undone. This will permanently delete "${pages.find(p => p.id === pageToDelete)?.name ?? 'this page'}" and remove its data.`}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel data-testid="delete-page-button-cancel">Cancel</AlertDialogCancel>
                  <AlertDialogAction asChild>
                    <Button
                      variant="destructive"
                      onClick={() => {
                        if (pageToDelete) {
                          handleDelete(pageToDelete);
                        }
                        setConfirmOpen(false);
                        setPageToDelete(null);
                      }}
                      data-testid="delete-page-button-confirm"
                    >
                      Continue
                    </Button>
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
        </InternalPage>
    );
};
