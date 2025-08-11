/**
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

import { Button, Select, SelectContent, SelectItem, SelectValue, SelectTrigger, Sheet, SheetContent, Tabs, TabsContent, TabsList, TabsTrigger, EmptyState, Card, Badge, formatDate, Alert, AlertTitle, AlertDescription, ScrollArea } from "@clidey/ux";
import { DatabaseType, useRawExecuteLazyQuery } from '@graphql';
import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { indexOf } from "lodash";
import { ChangeEvent, cloneElement, FC, ReactElement, ReactNode, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { v4 } from "uuid";
import { CodeEditor } from "../../components/editor";
import { AnalyzeGraphFallback } from "../../components/ee-fallbacks";
import { Icons } from "../../components/icons";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { StorageUnitTable } from "../../components/table";
import { InternalRoutes } from "../../config/routes";
import { LocalLoginProfile } from "../../store/auth";
import { useAppSelector } from "../../store/hooks";
import { isEEFeatureEnabled, loadEEComponent } from "../../utils/ee-loader";
import { BellAlertIcon, ClockIcon, XMarkIcon } from "@heroicons/react/24/outline";

// Conditionally load the AnalyzeGraph component from EE
const AnalyzeGraph = loadEEComponent(
    () => import('@ee/pages/raw-execute/analyze-view').then(m => ({ default: m.AnalyzeGraph })),
    AnalyzeGraphFallback
);

type IRawExecuteCellProps = {
    cellId: string;
    onAdd: (cellId: string) => void;
    onDelete?: (cellId: string) => void;
    showTools?: boolean;
}

enum ActionOptions {
    Query="Query",
    Analyze="Analyze",
}

// Only include Analyze option if EE is available
const getActionOptions = (): ActionOptions[] => {
    const options = [ActionOptions.Query];
    if (isEEFeatureEnabled('analyzeView')) {
        options.push(ActionOptions.Analyze);
    }
    return options;
};

export const ActionOptionIcons: Record<string, ReactElement> = {
    [ActionOptions.Query]: Icons.Database,
    [ActionOptions.Analyze]: Icons.Code,
}

const actionOptions = getActionOptions();

function getModeCommand(mode: ActionOptions, current?: LocalLoginProfile) {
    if (current == null || mode !== ActionOptions.Analyze) {
         return "";
    }
    if (current.Type === DatabaseType.Postgres) {
        return "EXPLAIN (ANALYZE, FORMAT JSON)"
    }
    return "";
}

const CopyButton: FC<{ text: string }> = (props) => {
    const [copied, setCopied] = useState(false);

    const handleCopyToClibpoard = useCallback(() => {
        navigator.clipboard.writeText(props.text).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        });
    }, []);

    return <div className="p-2 brightness-75 hover:brightness-100" onClick={handleCopyToClibpoard}>{copied ? Icons.CheckCircle : Icons.Clipboard}</div>;
}

const RawExecuteCell: FC<IRawExecuteCellProps> = ({ cellId, onAdd, onDelete, showTools }) => {
    const [mode, setMode] = useState<ActionOptions>(ActionOptions.Query);
    const [code, setCode] = useState("");
    const [submittedCode, setSubmittedCode] = useState("");
    const [rawExecute, { data: rows, loading, error }] = useRawExecuteLazyQuery();
    const [history, setHistory] = useState<{id: string, item: string, status: boolean, date: Date}[]>([]);
    const current = useAppSelector(state => state.auth.current);
    const [showHistory, setShowHistory] = useState(false);

    const handleRawExecute = useCallback((historyCode?: string) => {
        console.log("handleRawExecute", historyCode);
        if (current == null) {
            return;
        }
        const currentCode = historyCode ?? code;
        const historyItem = { id: v4(), item: code, status: false, date: new Date() };
        setSubmittedCode(currentCode);
        rawExecute({
            variables: {
                query: getModeCommand(mode, current) + currentCode,
            },
            onCompleted() {
                historyItem.status = true;
            },
        }).finally(() => {
            if (historyCode == null) setHistory(h => [historyItem , ...h]);
        });
    }, [code, rawExecute, current, mode]);

    const handleAdd = useCallback(() => {
        onAdd(cellId);
    }, [cellId, onAdd]);

    const handleDelete = useCallback(() => {
        onDelete?.(cellId);
    }, [cellId, onDelete]);

    const isCodeAQuery = useMemo(() => {
        if (submittedCode == null) {
            return true;
        }
        return submittedCode.split("\n").filter(text => !text.startsWith("--")).join("\n").trim().toLowerCase().startsWith("select");
    }, [submittedCode]);

    const output = useMemo(() => {
        if (rows == null) {
            return null;
        }
        if (mode === ActionOptions.Analyze && isEEFeatureEnabled('analyzeView') && AnalyzeGraph) {
            let data;
            try {
                data = JSON.parse(rows.RawExecute.Rows[0][0])[0];
            } catch {
                return <div className="text-red-500 mt-4">Unable to analyze the query</div>
            }
            return <div className="flex mt-4 h-[350px] w-full">
                <Suspense fallback={<Loading />}>
                {/* @ts-ignore */}
                    <AnalyzeGraph data={data} />
                </Suspense>
            </div>
        }
        if (isCodeAQuery || rows.RawExecute.Rows.length > 0) {
            return <div className="flex flex-col w-full h-[250px] mt-4" data-testid="cell-query-output">
                <StorageUnitTable
                    columns={rows.RawExecute.Columns.map(c => c.Name)}
                    columnTypes={rows.RawExecute.Columns.map(c => c.Type)}
                    rows={rows.RawExecute.Rows}
                    disableEdit={true}
                />
            </div>
        }
        return <div className="bg-white/10 text-neutral-800 dark:text-neutral-300 rounded-lg p-2 flex gap-2 self-start items-center my-4" data-testid="cell-action-output">
            Action Executed
            {Icons.CheckCircle}
        </div>
    }, [rows, isCodeAQuery, mode]);

    const isAnalyzeAvailable = useMemo(() => {
        if (!isEEFeatureEnabled('analyzeView')) {
            return false;
        }
        switch(current?.Type) {
            case DatabaseType.Postgres:
                return true;
        }
        return false;
    }, [current?.Type]);

    const rowLength = useMemo(() => rows?.RawExecute.Rows.length ?? 0, []);

    // Sheet for history (right side, like sidebar)
    const [historyOpen, setHistoryOpen] = useState(false);

    return (
        <div className="flex flex-col grow group/cell relative">
            <div className="relative">
                <div className="flex grow h-[150px] border border-gray-200 rounded-md overflow-hidden dark:bg-white/10 dark:border-white/5">
                    <CodeEditor language="sql" value={code} setValue={setCode} onRun={(c) => handleRawExecute(c)} />
                </div>
                <div className={classNames("absolute -bottom-3 z-20 flex justify-between px-3 pr-8 w-full opacity-0 transition-all duration-500 group-hover/cell:opacity-100 pointer-events-none", {
                    "opacity-100": showTools,
                })}>
                    <div className="flex gap-2 pointer-events-auto">
                        <Select
                            value={mode}
                            onValueChange={(val) => setMode(val as ActionOptions)}
                        >
                            <SelectTrigger style={{
                                background: "var(--secondary)",
                            }}>
                                <div className="flex items-center gap-2 w-full">
                                    {cloneElement(ActionOptionIcons[mode], { className: "w-4 h-4" })}
                                    <span>{mode}</span>
                                </div>
                            </SelectTrigger>
                            <SelectContent>
                                {actionOptions.map((item) => (
                                    <SelectItem
                                        key={item}
                                        value={item}
                                        className={classNames({
                                            "hidden": !isAnalyzeAvailable || (item === ActionOptions.Analyze && !isEEFeatureEnabled('analyzeView')),
                                        })}
                                    >
                                        <div className="flex items-center gap-2">
                                            {cloneElement(ActionOptionIcons[item], { className: "w-4 h-4" })}
                                            <span>{item}</span>
                                        </div>
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Button onClick={handleAdd} data-testid="add-button" variant="secondary" className="border border-input">
                            {Icons.PlusCircle}
                        </Button>
                        <Button onClick={() => setCode("")} data-testid="clear-button" variant="secondary" className="border border-input">
                            {Icons.Refresh}
                        </Button>
                        {
                            onDelete != null &&
                            <Button variant="destructive" onClick={handleDelete} disabled={loading} data-testid="delete-button">
                                {Icons.Delete}
                            </Button>
                        }
                    </div>
                    <div className="flex gap-2 items-center">
                        <Button
                            onClick={() => setHistoryOpen(true)}
                            data-testid="history-button"
                            className="pointer-events-auto"
                            variant="secondary"
                        >
                            <ClockIcon className="w-4 h-4" />
                        </Button>
                        <Button onClick={() => handleRawExecute()} data-testid="submit-button" className="pointer-events-auto">
                            {Icons.CheckCircle}
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
            {
                loading
                ? <div className="flex justify-center items-center h-[250px]">
                    <Loading />
                </div>
                : rows != null && submittedCode.length > 0 && output
            }
            <Sheet open={historyOpen} onOpenChange={setHistoryOpen}>
                <SheetContent className="w-[350px] max-w-full p-0">
                    <div className="flex flex-col h-full">
                        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
                            <div className="flex items-center gap-2">
                                <ClockIcon className="w-5 h-5" />
                                <span className="font-semibold text-lg">Query History</span>
                            </div>
                        </div>
                        <ScrollArea className="flex-1 px-2 py-4">
                            {history.length === 0 ? (
                                <EmptyState title="No history yet" description="Run a query to see your history" icon={<ClockIcon className="w-10 h-10" />} />
                            ) : (
                                <div className="flex flex-col gap-4 p-4">
                                    {history.map(({ id, item, status, date }) => (
                                        <Card className="w-full p-4 relative" key={id}>
                                            <Badge
                                                variant={status ? "success" : "destructive"}
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
                                                        >
                                                            {cloneElement(Icons.Edit, { className: "w-5 h-5" })}
                                                        </Button>
                                                        <Button
                                                            size="icon"
                                                            variant="secondary"
                                                            className="border border-input"
                                                            onClick={() => handleRawExecute(item)}
                                                            title="Run"
                                                        >
                                                            {Icons.Play}
                                                        </Button>
                                                    </div>
                                                </div>
                                            </div>
                                        </Card>
                                    ))}
                                </div>
                            )}
                        </ScrollArea>
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
            <div className="w-full max-w-[1000px] flex flex-col gap-4">
                {
                    cellIds.map((cellId, index) => (
                        <div key={cellId} data-testid={`cell-${index}`}>
                            {index > 0 && <div className="border-dashed border-t border-gray-300 my-2 dark:border-neutral-600"></div>}
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
            <div className="flex justify-center items-center w-full">
                <div className="w-full max-w-[1000px] flex flex-col gap-4">
                    <div className="flex justify-between items-center">
                        <Tabs defaultValue="buttons" className="w-full h-full" value={activePage}>
                            <div className="flex gap-2 w-full justify-between">
                                <TabsList className="grid" style={{
                                    gridTemplateColumns: `repeat(${pages.length+1}, minmax(0, 1fr))`
                                }} defaultValue={activePage}>
                                    {
                                        pages.map(page => (
                                            <TabsTrigger value={page.id} key={page.id} onClick={() => handleSelect(page.id)}>
                                                <EditableInput page={page} setValue={(newName) => handleUpdatePageName(page, newName)} />
                                            </TabsTrigger>
                                        ))
                                    }
                                    <TabsTrigger value="add" onClick={handleAdd}>
                                        {Icons.Add}
                                    </TabsTrigger>
                                </TabsList>
                                <Button className={classNames({
                                    "hidden": pages.length <= 1,
                                })} variant="secondary" onClick={() => handleDelete(activePage)}>{Icons.Delete} Delete page</Button>
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
        </InternalPage>
    );
};
