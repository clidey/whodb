import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { indexOf, values } from "lodash";
import { ChangeEvent, cloneElement, FC, ReactElement, ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { v4 } from "uuid";
import { AnimatedButton } from "../../components/button";
import { ClassNames } from "../../components/classes";
import { CodeEditor } from "../../components/editor";
import { Icons } from "../../components/icons";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Table } from "../../components/table";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, useRawExecuteLazyQuery } from "../../generated/graphql";
import { useAppSelector } from "../../store/hooks";

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
    [ActionOptions.Query]: Icons.Database,
}

const actionOptions = values(ActionOptions);

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
    const [history, setHistory] = useState<{id: string, item: string, status: boolean}[]>([]);
    const current = useAppSelector(state => state.auth.current);
    const [showHistory, setShowHistory] = useState(false);

    const handleRawExecute = useCallback((historyCode?: string) => {
        if (current == null) {
            return;
        }
        const currentCode = historyCode ?? code;
        const historyItem = { id: v4(), item: code, status: false };
        setSubmittedCode(currentCode);
        rawExecute({
            variables: {
                query: currentCode,
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
        if (isCodeAQuery || rows.RawExecute.Rows.length > 0) {
            return <div className="flex flex-col w-full h-[250px] mt-4" data-testid="cell-query-output">
                <Table columns={rows.RawExecute.Columns.map(c => c.Name)} columnTags={rows.RawExecute.Columns.map(c => c.Type)}
                rows={rows.RawExecute.Rows} totalPages={1} currentPage={1} disableEdit={true} />
            </div>
        }
        return <div className="bg-white/10 text-neutral-800 dark:text-neutral-300 rounded-lg p-2 flex gap-2 self-start items-center my-4" data-testid="cell-action-output">
            Action Executed
            {Icons.CheckCircle}
        </div>
    }, [rows, isCodeAQuery, mode]);

    const isAnalyzeAvailable = useMemo(() => {
        switch(current?.Type) {
            case DatabaseType.Postgres:
                return true;
        }
        return false;
    }, []);

    const rowLength = useMemo(() => rows?.RawExecute.Rows.length ?? 0, []);

    return <div className="flex flex-col grow group/cell relative">
            <div className="absolute left-0 -translate-x-full pr-2">
                {actionOptions.map((item) => (
                    <motion.div
                        key={item}
                        onClick={() => setMode(item)}
                        className={classNames(
                            ClassNames.Text,
                            "relative text-sm px-2 py-1 rounded-lg rounded-r-none cursor-pointer transition-all w-[150px] whitespace-nowrap text-ellipsis hover:brightness-125 flex gap-1 items-center",
                            {
                                "hidden": !isAnalyzeAvailable,
                            }
                        )}
                        title={item}
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ type: "spring", stiffness: 500, damping: 30 }}>
                        <AnimatePresence>
                            {mode === item && (
                                <motion.div
                                    layoutId={`activeBackground-${cellId}`}
                                    className="absolute inset-0 bg-neutral-800/5 dark:bg-neutral-700 rounded-lg rounded-r-none -z-10"
                                    transition={{ type: "spring", stiffness: 500, damping: 30 }}
                                />
                            )}
                        </AnimatePresence>
                        {cloneElement(ActionOptionIcons[item], {
                            className: "w-4 h-4",
                        })}
                        <span className="relative z-10">{item}</span>
                    </motion.div>
                ))}
            </div>
            {
                showHistory ?
                    <div className="flex flex-col gap-2 grow h-[150px] pb-8 overflow-y-auto">
                        {history.map(({ id, item, status }) => (
                            <motion.div key={id}
                                className={classNames(
                                    ClassNames.Text, 
                                    "text-sm bg-white/5 px-2 rounded-lg rounded-l-none cursor-pointer transition-all w-full group/history-item py-4 h-fit border-l-4 relative",
                                    {
                                        " border-l-green-500": status,
                                        " border-l-red-500": !status,
                                    }
                                )}
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1, transition: { y: { stiffness: 1000, velocity: -100 }}}}
                                exit={{ opacity: 0 }}
                            >
                                {item}
                                <div className="opacity-0 group-hover/history-item:opacity-100 absolute right-4 top-1/2 -translate-y-1/2 px-4 py-2 flex items-center">
                                    <CopyButton text={item} />
                                    <div className="p-2 brightness-75 hover:brightness-100" onClick={() => {
                                        setShowHistory(false);
                                        setCode(item);
                                    }}>{cloneElement(Icons.Edit, {
                                        className: "w-5 h-5",
                                    })}</div>
                                    <div className="p-2 brightness-75 hover:brightness-100" onClick={() => {
                                        handleRawExecute(item);
                                    }}>{Icons.Play}</div>
                                </div>
                            </motion.div>
                        ))}
                    </div>
                : <div className="relative">
                    <div className="flex grow h-[150px] border border-gray-200 rounded-md overflow-hidden dark:bg-white/10 dark:border-white/5">
                        <CodeEditor language="sql" value={code} setValue={setCode} onRun={() => handleRawExecute()} />
                    </div>
                    <div className={classNames("absolute -bottom-3 z-20 flex justify-between px-3 pr-8 w-full opacity-0 transition-all duration-500 group-hover/cell:opacity-100", {
                        "opacity-100": showTools,
                    })}>
                        <div className="flex gap-2">
                            <AnimatedButton icon={Icons.PlusCircle} label="Add" onClick={handleAdd} testId="add-button" />
                            <AnimatedButton icon={Icons.Refresh} label="Clear" onClick={() => setCode("")} testId="clear-button" />
                            {
                                onDelete != null &&
                                <AnimatedButton  iconClassName="stroke-red-800 dark:stroke-red-500" labelClassName="text-red-800 dark:text-red-500"  icon={Icons.Delete} label="Delete" onClick={handleDelete} disabled={loading} testId="delete-button" />
                            }
                        </div>
                        <AnimatedButton iconClassName="stroke-green-800 dark:stroke-green-500" labelClassName="text-green-800 dark:text-green-500" icon={Icons.CheckCircle} label={mode} onClick={() => handleRawExecute()} testId="submit-button" />
                    </div>
                </div>
            }
            {
                error != null &&
                <div className="flex items-center justify-between mt-4" data-testid="cell-error">
                    <div className="text-sm text-red-500 w-[33vw]">{error?.message ?? ""}</div>
                </div>
            }
            {
                loading
                ? <div className="flex justify-center items-center h-[250px]">
                    <Loading />
                </div>
                : rows != null && submittedCode.length > 0 && output
            }
            <div className={classNames("absolute right-0 translate-x-full pl-2 overflow-y-auto overflow-x-hidden", {
                "max-h-[200px]": rowLength === 0,
                "max-h-[400px]": rowLength > 0,
            })}>
                <motion.ul className="flex flex-col gap-1"
                    initial={{opacity: 0, }}
                    animate={{opacity: 1, transition: { staggerChildren: 0.07, delayChildren: 0.05 }}}
                    exit={{opacity: 0, transition: { staggerChildren: 0.05, staggerDirection: -1 }}}>
                    <motion.li
                        className={classNames(
                            ClassNames.Text,
                            "relative text-sm px-2 py-1 rounded-lg rounded-l-none cursor-pointer transition-all w-[150px] whitespace-nowrap text-ellipsis hover:brightness-125 flex gap-1 items-center",
                            {
                                "hidden": history.length === 0,
                            }
                        )}
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ type: "spring", stiffness: 500, damping: 30 }}
                        onClick={() => setShowHistory(!showHistory)}
                    >
                        {cloneElement(Icons.History, {
                            className: classNames("w-4 h-4 transition-all", {
                                "rotate-180": showHistory,
                            })
                        })}
                        <span className="relative z-10">{showHistory ? "Hide" : "Show"} history</span>
                    </motion.li>
                </motion.ul>
            </div>
        </div>
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
            className="w-full bg-transparent border-b border-gray-400 focus:outline-none focus:border-blue-500 transition-colors text-inherit"
          />
        ) : (
          <span className={classNames(ClassNames.Text, "text-sm text-nowrap")}>
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
                    <div className="flex justify-start items-center gap-2">
                        <div className="flex justify-start max-w-[calc(50vw+48px)] items-center gap-2 overflow-x-auto">
                            {pages.map((page, index) => (
                                <div
                                    key={page.id}
                                    data-testid={`page-${index}`}
                                    className={classNames(
                                        ClassNames.Text,
                                        "pl-4 py-1 text-sm rounded-3xl flex items-center gap-1 cursor-pointer transition-all group/page-button",
                                        {
                                            "bg-neutral-300/5 dark:bg-neutral-800": activePage !== page.id,
                                            "bg-neutral-800/5 dark:bg-neutral-700 shadow": activePage === page.id,
                                        }
                                    )}
                                    onClick={() => handleSelect(page.id)}
                                >
                                    <div className="flex items-center gap-1">
                                        {cloneElement(Icons.Console, { className: "w-4 h-4" })}
                                        <EditableInput page={page} setValue={(newName) => handleUpdatePageName(page, newName)} />
                                    </div>
                                    <div className={classNames("ml-1 group-hover/page-button:flex pr-2 opacity-10 group-hover/page-button:opacity-100", {
                                        "opacity-0": pages.length <= 1,
                                    })} onClick={(e) => {
                                        handleDelete(page.id);
                                        e.preventDefault();
                                        e.stopPropagation();
                                    }}>
                                        {cloneElement(Icons.Cancel, {
                                            className: "w-4 h-4",
                                        })}
                                    </div>
                                </div>
                            ))}
                        </div>
                        <div
                            className={classNames(
                                ClassNames.Text,
                                "p-1 rounded-full bg-black/5 dark:bg-white/5 transition-all hover:scale-105 cursor-pointer"
                            )}
                            onClick={handleAdd}
                        >
                            {Icons.Add}
                        </div>
                    </div>
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
                </div>
            </div>
        </InternalPage>
    );
};
