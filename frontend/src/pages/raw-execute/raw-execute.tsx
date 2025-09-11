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
    Alert,
    AlertDescription,
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
    AlertTitle,
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
    Tabs,
    TabsContent,
    TabsList,
    TabsTrigger
} from "@clidey/ux";
import {DatabaseType, RowsResult} from '@graphql';
import {
    ArrowPathIcon,
    BellAlertIcon,
    CheckCircleIcon,
    CircleStackIcon,
    ClipboardDocumentIcon,
    ClockIcon,
    EllipsisVerticalIcon,
    PencilIcon,
    PlayIcon,
    PlusCircleIcon,
    XMarkIcon
} from "@heroicons/react/24/outline";
import classNames from "classnames";
import {AnimatePresence, motion} from "framer-motion";
import {indexOf} from "lodash";
import {
    ChangeEvent,
    cloneElement,
    FC,
    ReactElement,
    ReactNode,
    Suspense,
    useCallback,
    useEffect,
    useMemo,
    useRef,
    useState
} from "react";
import {v4} from "uuid";
import {AIProvider, useAI} from "../../components/ai";
import {CodeEditor} from "../../components/editor";
import {Loading} from "../../components/loading";
import {InternalPage} from "../../components/page";
import {Tip} from "../../components/tip";
import {InternalRoutes} from "../../config/routes";
import {useAppDispatch, useAppSelector} from "../../store/hooks";
import {isEEFeatureEnabled, loadEEModule} from "../../utils/ee-loader";
import {IPluginProps, QueryView} from "./query-view";

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
}

enum ActionOptions {
    Query="Query",
}

export const ActionOptionIcons: Record<string, ReactElement> = {
    [ActionOptions.Query]: <CircleStackIcon className="w-4 h-4" />,
}

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

const RawExecuteCell: FC<IRawExecuteCellProps> = ({ cellId, onAdd, onDelete, showTools }) => {
    const [mode, setMode] = useState<string>(ActionOptions.Query);
    const [code, setCode] = useState("");
    const [submittedCode, setSubmittedCode] = useState("");
    const [history, setHistory] = useState<{id: string, item: string, status: boolean, date: Date}[]>([]);
    const current = useAppSelector(state => state.auth.current);
    const handleExecute = useRef<(code: string) => Promise<any>>(() => Promise.resolve());
    const [historyOpen, setHistoryOpen] = useState(false);
    const [error, setError] = useState<Error | null>(null);
    const [loading, setLoading] = useState(false);
    const [rows, setRows] = useState<RowsResult | null>(null);
    const { modelType } = useAI();

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
        });
    }, [code, current, mode, allActionOptions, handleExecute]);

    const handleAdd = useCallback(() => {
        onAdd(cellId);
    }, [cellId, onAdd]);

    const handleDelete = useCallback(() => {
        onDelete?.(cellId);
    }, [cellId, onDelete]);

    // Use all plugins
    const output = useMemo(() => {
        const selectedPlugin = allPlugins.find((p: any) => p.type === mode);
        if (selectedPlugin?.component == null || current == null) {
            return null;
        }
        const Component = selectedPlugin.component as FC<IPluginProps>;
        return <div className="flex mt-4 w-full">
            <Suspense fallback={<Loading />}>
                <Component code={code} handleExecuteRef={handleExecute} modelType={modelType?.modelType || ''}
                           schema={current.Database} token={modelType?.token} providerId={current.Id}/>
            </Suspense>
        </div>
    }, [mode, allActionOptions, allPlugins, code, modelType, current]);

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
                <div className="flex grow h-[150px] border border-gray-200 rounded-md overflow-hidden dark:bg-white/10 dark:border-white/5">
                    <CodeEditor language="sql" value={code} setValue={setCode} onRun={(c) => handleRawExecute(c)} />
                </div>
                <div className="absolute top-2 right-2 z-10" data-testid="scratchpad-cell-options">
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
                    <div className="flex gap-2 pointer-events-auto">
                        {actionOptions.length > 1 && <Select
                            value={mode}
                            onValueChange={(val) => setMode(val as string)}
                        >
                            <SelectTrigger style={{
                                background: "var(--secondary)",
                            }}>
                                <div className="flex items-center gap-2 w-full">
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
                                    <XMarkIcon className="w-4 h-4 text-destructive"/>
                                </Button>
                                <p>Delete the cell</p>
                            </Tip>
                        }
                    </div>
                    <div className="flex gap-2 items-center">
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
                    <Alert variant="destructive" title="Error" description={error?.message ?? ""}>
                        <BellAlertIcon className="w-4 h-4" />
                        <AlertTitle>Error</AlertTitle>
                        <AlertDescription>{error?.message ?? ""}</AlertDescription>
                    </Alert>
                </div>
            }
            {loading && <div className="flex justify-center items-center h-full my-16">
                <Loading/>
            </div>}
            {output}
            <Sheet open={historyOpen} onOpenChange={setHistoryOpen}>
                <SheetContent className="w-[350px] max-w-full p-0">
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
                                <div className="flex flex-col gap-4 p-4">
                                    {history.map(({ id, item, status, date }) => (
                                        <Card className="w-full p-4 relative" key={id}>
                                            <Badge
                                                variant={status ? "default" : "destructive"}
                                                className="absolute top-0 -translate-y-1/2 right-2"
                                            >
                                                {status ? "Success" : "Error"}
                                            </Badge>
                                            <div className="flex flex-col min-h-[60px]">
                                                <div className="whitespace-pre-wrap break-words text-sm pr-12">
                                                    {item}
                                                </div>
                                                <div className="flex gap-2 mt-4 justify-between items-center">
                                                    <div className="text-xs text-muted-foreground">
                                                        {formatDate(date)}
                                                    </div>
                                                    <div className="flex gap-2 items-center">
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
                </SheetContent>
            </Sheet>
        </div>
    )
}

const RawExecuteSubPage: FC = () => {
    const [cellIds, setCellIds] = useState<string[]>([v4()]);
    
    const handleAdd = useCallback((id: string) => {
        const index = indexOf(cellIds, id);
        const newCellIds = [...cellIds];
        newCellIds.splice(index+1, 0, v4());
        setCellIds(newCellIds);
    }, [cellIds]);

    const handleDelete = useCallback((cellId: string) => {
        if (cellIds.length <= 1) {
            return;
        }
        setCellIds(ids => ids.filter(id => id !== cellId));
    }, [cellIds.length]);

    return (
        <div className="flex justify-center items-center w-full">
            <div className="w-full flex flex-col gap-2">
                {
                    cellIds.map((cellId, index) => (
                        <div key={cellId} data-testid={`cell-${index}`}>
                            {index > 0 && <Separator className="my-4" />}
                            <RawExecuteCell key={cellId} cellId={cellId} onAdd={handleAdd} onDelete={cellIds.length <= 1 ? undefined : handleDelete}
                                showTools={cellIds.length === 1} />
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
            className="w-full border-b border-gray-400 focus:outline-none focus:border-blue-500 transition-colors text-inherit"
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
    const [pages, setPages] = useState<Page[]>(() => {
        const newId = v4();
        return [{ id: newId, name: "Page 1" }];
    });

    const [activePage, setActivePage] = useState(pages[0].id);
    const [pageStates, setPageStates] = useState<{ [key: string]: ReactNode }>({});

    const aiState = useAI();
    const dispatch = useAppDispatch();

    const handleAdd = useCallback(() => {
        const newId = v4();
        setPages((prevPages) => [
            ...prevPages,
            { id: newId, name: `Page ${prevPages.length + 1}` },
        ]);

        setPageStates((prevStates) => ({
            ...prevStates,
            [newId]: <RawExecuteSubPage key={newId} />,
        }));
    }, []);

    const handleSelect = useCallback((pageId: string) => {
        setActivePage(pageId);
    }, []);

    const handleDelete = useCallback((pageId: string) => {
        setPages((prevPages) => {
            if (prevPages.length <= 1) return prevPages;
            const updatedPages = prevPages.filter((page) => page.id !== pageId);

            if (pageId === activePage) {
                setActivePage(updatedPages[0].id);
            }

            return updatedPages;
        });

        setPageStates((prevStates) => {
            const newStates = { ...prevStates };
            delete newStates[pageId];
            return newStates;
        });
    }, [activePage]);

    const handleUpdatePageName = useCallback((changedPage: Page, newName: string) => {
        setPages(prevPages => {
            const foundPageIndex = prevPages.findIndex(page => page.id === changedPage.id);
            if (foundPageIndex === -1) {
                return prevPages;
            }
            prevPages[foundPageIndex].name = newName;
            return prevPages;
        });
    }, []);

    useEffect(() => {
        setPageStates((prevStates) => {
            const newStates = { ...prevStates };
            pages.forEach((page) => {
                if (newStates[page.id] == null) {
                    newStates[page.id] = <RawExecuteSubPage key={page.id} />;
                }
            });
            return newStates;
        });
    }, [pages]);

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
                            <Tabs defaultValue="buttons" className="w-full h-full" value={activePage}>
                                <div className="flex gap-2 w-full justify-between">
                                    <TabsList className="grid" style={{
                                        gridTemplateColumns: `repeat(${pages.length+1}, minmax(0, 1fr))`
                                    }} defaultValue={activePage} data-testid="page-tabs">
                                        {
                                            pages.map((page, index) => (
                                                <TabsTrigger value={page.id} key={page.id}
                                                             onClick={() => handleSelect(page.id)}
                                                             data-testid={`page-tab-${index}`}>
                                                    <EditableInput page={page} setValue={(newName) => handleUpdatePageName(page, newName)} />
                                                </TabsTrigger>
                                            ))
                                        }
                                        <TabsTrigger value="add" onClick={handleAdd} data-testid="add-page-button">
                                            <PlusCircleIcon className="w-4 h-4" />
                                        </TabsTrigger>
                                    </TabsList>
                                    <AlertDialog>
                                      <AlertDialogTrigger asChild>
                                        <Button
                                          className={classNames({
                                            "hidden": pages.length <= 1,
                                          })}
                                          variant="secondary"
                                          data-testid="delete-page-button"
                                        >
                                          <XMarkIcon className="w-4 h-4" /> Delete page
                                        </Button>
                                      </AlertDialogTrigger>
                                      <AlertDialogContent>
                                        <AlertDialogHeader>
                                          <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
                                          <AlertDialogDescription>
                                            This action cannot be undone. This will permanently delete this page and remove its data.
                                          </AlertDialogDescription>
                                        </AlertDialogHeader>
                                        <AlertDialogFooter>
                                            <AlertDialogCancel
                                                data-testid="delete-page-button-cancel">Cancel</AlertDialogCancel>
                                          <AlertDialogAction asChild>
                                            <Button
                                              variant="destructive"
                                              onClick={() => handleDelete(activePage)}
                                              data-testid="delete-page-button-confirm"
                                            >
                                              Continue
                                            </Button>
                                          </AlertDialogAction>
                                        </AlertDialogFooter>
                                      </AlertDialogContent>
                                    </AlertDialog>
                                </div>
                                <TabsContent value={activePage} className="h-full w-full mt-4">
                                    <AnimatePresence mode="wait">
                                        {Object.entries(pageStates).map(([id, component]) => (
                                            <motion.div
                                                key={id}
                                                className={classNames({
                                                    "hidden": id !== activePage,
                                                })}
                                                // todo this animation
                                                initial={{ opacity: 0, y: 10 }}
                                                animate={{ opacity: 1, y: 0 }}
                                                exit={{ opacity: 0, y: -10 }}
                                            >
                                                {component}
                                            </motion.div>
                                        ))}
                                    </AnimatePresence>
                                </TabsContent>
                            </Tabs>
                        </div>
                    </div>
                </div>
            </div>
        </InternalPage>
    );
};
