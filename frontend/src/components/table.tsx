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

import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { clone, isString, values } from "lodash";
import { CSSProperties, FC, KeyboardEvent, MouseEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { flexRender, getCoreRowModel, getSortedRowModel, SortingState, useReactTable, ColumnDef, Row, Cell } from '@tanstack/react-table';
import { FixedSizeList, ListChildComponentProps } from "react-window";
import { twMerge } from "tailwind-merge";
import { notify } from "../store/function";
import { isMarkdown, isNumeric, isValidJSON } from "../utils/functions";
import { ActionButton, AnimatedButton } from "./button";
import { Portal } from "./common";
import { CodeEditor } from "./editor";
import { useExportToCSV, useLongPress } from "./hooks";
import { Icons } from "./icons";
import { CheckBoxInput } from "./input";
import { Loading } from "./loading";
import { SearchInput } from "./search";

type IPaginationProps = {
    pageCount: number;
    currentPage: number;
    onPageChange?: (page: number) => void;
}

const Pagination: FC<IPaginationProps> = ({ pageCount, currentPage, onPageChange }) => {
    const paginationRef = useRef<HTMLDivElement>(null);
    const [focusedPage, setFocusedPage] = useState<number | null>(null);

    const handlePageChange = useCallback((page: number) => {
        setFocusedPage(page);
        onPageChange?.(page);
        
        // Set focus to the new current page button after page change
        setTimeout(() => {
            if (paginationRef.current) {
                const pageButton = paginationRef.current.querySelector(`button[aria-current="page"]`) as HTMLButtonElement;
                if (pageButton) {
                    pageButton.focus();
                } else {
                    // Fallback: focus first available page button
                    const firstButton = paginationRef.current.querySelector('button') as HTMLButtonElement;
                    firstButton?.focus();
                }
            }
        }, 100);
    }, [onPageChange]);

    const renderPageNumbers = () => {
        const pageNumbers = [];
        const maxVisiblePages = 5;

        if (pageCount <= maxVisiblePages) {
            for (let i = 1; i <= pageCount; i++) {
                pageNumbers.push(
                    <button
                        key={i}
                        className={`cursor-pointer p-2 text-sm hover:scale-110 hover:bg-gray-200 rounded-md text-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500 ${currentPage === i ? 'bg-gray-300' : ''}`}
                        onClick={() => handlePageChange(i)}
                        onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                                e.preventDefault();
                                handlePageChange(i);
                            }
                        }}
                        aria-label={`Go to page ${i}`}
                        aria-current={currentPage === i ? 'page' : undefined}
                        data-testid="table-page-number">
                        {i}
                    </button>
                );
            }
        } else {
            const createPageItem = (i: number) => (
                <button
                    key={i}
                    className={classNames("cursor-pointer p-2 text-sm hover:scale-110 hover:bg-gray-200 dark:hover:bg-white/15 rounded-md text-gray-600 dark:text-neutral-300 focus:outline-none focus:ring-2 focus:ring-blue-500", {
                        "bg-gray-300 dark:bg-white/10": currentPage === i,
                    })}
                    onClick={() => handlePageChange(i)}
                    onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault();
                            handlePageChange(i);
                        }
                    }}
                    aria-label={`Go to page ${i}`}
                    aria-current={currentPage === i ? 'page' : undefined}
                    data-testid="table-page-number">
                    {i}
                </button>
            );

            pageNumbers.push(createPageItem(1));

            if (currentPage > 3) {
                pageNumbers.push(
                    <div key="start-ellipsis" className="cursor-default p-2 text-sm text-gray-600 dark:text-neutral-300">...</div>
                );
            }

            const startPage = Math.max(2, currentPage - 1);
            const endPage = Math.min(pageCount - 1, currentPage + 1);

            for (let i = startPage; i <= endPage; i++) {
                pageNumbers.push(createPageItem(i));
            }

            if (currentPage < pageCount - 2) {
                pageNumbers.push(
                    <div key="end-ellipsis" className="cursor-default p-2 text-sm text-gray-600 dark:text-neutral-300">...</div>
                );
            }

            pageNumbers.push(createPageItem(pageCount));
        }

        return pageNumbers;
    };

    return (
        <nav aria-label="Table pagination" role="navigation">
            <div ref={paginationRef} className="flex space-x-2">
                {renderPageNumbers()}
            </div>
        </nav>
    );
};

type ITDataProps = {
    cell: Cell<Record<string, string | number>, unknown>;
    onCellUpdate?: (cell: Cell<Record<string, string | number>, unknown>) => Promise<void>;
    disableEdit?: boolean;
    checked?: boolean;
    onRowCheck?: (value: boolean) => void;
}

const TData: FC<ITDataProps> = ({ cell, onCellUpdate, checked, onRowCheck, disableEdit }) => {
    const [changed, setChanged] = useState(false);
    const [editedData, setEditedData] = useState<string>(cell.getValue() as string);
    const [editable, setEditable] = useState(false);
    const [preview, setPreview] = useState(false);
    const [cellRect, setCellRect] = useState<DOMRect | null>(null);
    const cellRef = useRef<HTMLDivElement>(null);
    const [copied, setCopied] = useState(false);
    const [updating, setUpdating] = useState(false);
    const [escapeAttempted, setEscapeAttempted] = useState(false);

    const handleChange = useCallback((value: string) => {
        setEditedData(value);
        if (!changed) setChanged(true);
    }, [changed]);

    const handleCancel = useCallback(() => {
        setEditedData(cell.getValue() as string);
        setEditable(false);
        setCellRect(null);
    }, [cell]);

    const handleEdit = useCallback((e: MouseEvent) => {
        e.stopPropagation();
        e.preventDefault();
        if (cellRef.current) {
            setCellRect(cellRef.current.getBoundingClientRect());
            setEditable(true);
        }
    }, []);

    const handlePreview = useCallback(() => {
        if (cellRef.current) {
            setCellRect(cellRef.current.getBoundingClientRect());
            setPreview(true);
        }
    }, []);

    const handleLongPress = useCallback(() => {
        handlePreview();
        return () => {
            setCellRect(null);
            setPreview(false);
        }
    }, [handlePreview]);

    const handleCopy = useCallback(() => {
        navigator.clipboard.writeText(editedData).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        });
    }, [editedData]);

    const longPressProps = useLongPress({
        onLongPress: handleLongPress,
        onClick: handleCopy,
    });

    const handleUpdate = useCallback(() => {
        setUpdating(true);
        // Create a cell context with the edited data
        const cellWithData = { ...cell, editedData };
        onCellUpdate?.(cellWithData as any).then(() => {
            setEditable(false);
            setCellRect(null);
        }).catch(() => {
            // Revert value on error
            setEditedData(cell.getValue() as string);
        }).finally(() => {
            setUpdating(false); 
        });
    }, [cell, editedData, onCellUpdate]);

    const handleEditorEscapeButton = useCallback((e: KeyboardEvent) => {
        if (e.key === "Escape" && !changed) {
            handleCancel();
        } else if (e.key === "Escape" && changed) {
            if (escapeAttempted) {
                setEscapeAttempted(false);
                handleCancel();
            } else {
                setEscapeAttempted(true);
                notify("You have unsaved changes, please save or cancel. Pressing Escape again will close without saving.", "warning");
                setTimeout(() => setEscapeAttempted(false), 2000); // reset it in case
            }
        }
    }, [changed, handleCancel, escapeAttempted]);

    const language = useMemo(() => {
        if (editedData == null) {
            return;
        }
        if (isValidJSON(editedData)) {
            return "json";
        }
        if (isMarkdown(editedData)) {
            return "markdown";
        }
    }, [editedData]);

    useEffect(() => {
        setEditedData(cell.getValue() as string);
    }, [cell]);

    const columnId = cell.column.id;

    return <div ref={cellRef}
        className={classNames("relative group/data cursor-pointer transition-all text-xs table-cell border-t border-l last:border-r group-last/row:border-b first:group-last/row:rounded-bl-lg last:group-last/row:rounded-br-lg border-gray-200 dark:border-white/5 p-0", {
            "bg-gray-200 dark:bg-white/10 blur-[2px]": editable || preview,
        })} data-testid="table-row-data">
        <span className="cell-data hidden">{editedData}</span>
        <div 
            className={classNames("w-full h-full p-2 leading-tight focus:outline-hidden focus:shadow-outline appearance-none transition-all duration-300 border-solid border-gray-200 dark:border-white/5 overflow-hidden whitespace-nowrap select-none text-gray-600 dark:text-neutral-300", {
                "group-even/row:bg-gray-100 hover:bg-gray-300 hover:group-even/row:bg-gray-300 dark:group-even/row:bg-white/10 dark:group-odd/row:bg-white/5 dark:hover:group-even/row:bg-white/15 dark:hover:group-odd/row:bg-white/15": !editable,
                "bg-transparent": editable,
            })}>
            <div className={classNames("absolute top-0 left-0 h-full w-full justify-center items-center bg-transparent z-1 hover:scale-110 transition-all", {
                "group-hover/row:flex": checked != null && columnId === "#",
                "flex": columnId === "#" && checked === true,
                "hidden": checked == null || columnId !== "#" || checked === false,
            })}>
                <CheckBoxInput value={checked ?? false} setValue={onRowCheck} />
            </div>
            <div className={classNames({
                "group-hover/row:hidden": checked != null && columnId === "#",
                "hidden": columnId === "#" && checked === true,
            })} {...longPressProps}>
                {editedData}
            </div>
        </div>
        <div className={classNames("transition-all hidden absolute right-2 top-1/2 -translate-y-1/2 hover:scale-125 p-1", {
            "hidden": copied || disableEdit,
            "group-hover/data:flex": !copied && !disableEdit,
        })} onClick={handleEdit} data-testid="edit-button">
            {Icons.Edit}
        </div>
         <AnimatePresence mode="wait">
            {cellRect != null && (
                <Portal>
                    <motion.div
                        initial={{ opacity: 0, }}
                        animate={{ opacity: 1, }}
                        exit={{ opacity: 0, }}
                        transition={{ duration: 0.3 }}
                        className={classNames("fixed top-0 left-0 w-screen h-screen flex items-center justify-center z-50 bg-gray-500/40", {
                            "select-none": preview,
                        })}
                        onMouseUp={preview ? longPressProps.onMouseUp : undefined} onTouchEnd={preview ? longPressProps.onTouchEnd : undefined}
                        data-testid="edit-dialog">
                        <motion.div
                            initial={{
                                top: cellRect.top,
                                left: cellRect.left,
                                width: cellRect.width,
                                height: cellRect.height,
                                transform: "unset",
                            }}
                            animate={{
                                top: "20vh",
                                left: "20vw",
                                height: "60vh",
                                width: "60vw",
                            }}
                            exit={{
                                top: cellRect.top,
                                left: cellRect.left,
                                width: cellRect.width,
                                height: cellRect.height,
                                transform: "unset",
                            }}
                            transition={{ duration: 0.3 }}
                            className="absolute flex flex-col h-full justify-between gap-4">
                            <div className="rounded-lg shadow-lg overflow-hidden grow" onKeyDown={handleEditorEscapeButton}>
                                <CodeEditor
                                    defaultShowPreview={preview}
                                    disabled={preview}
                                    language={language}
                                    value={editedData}
                                    setValue={handleChange}
                                />
                            </div>
                            <motion.div
                                initial={{ opacity: 0, }}
                                animate={{ opacity: 1, }}
                                exit={{ opacity: 0, }}
                                transition={{ duration: 0.1 }}
                                className={classNames("flex gap-2 justify-center w-full", {
                                    "hidden": preview,
                                })}>
                                <ActionButton icon={Icons.Cancel} onClick={handleCancel} disabled={updating} testId="cancel-update-button" />
                                {
                                    updating
                                    ? <div className="bg-white rounded-full p-2"><Loading hideText={true} /></div>
                                    : <ActionButton icon={Icons.CheckCircle} className={changed ? "stroke-green-500" : undefined} onClick={handleUpdate} disabled={!changed} testId="update-button" />
                                }
                            </motion.div>
                        </motion.div>
                    </motion.div>
                </Portal>
            )}
        </AnimatePresence>
        <AnimatePresence>
            {copied && (
                <motion.div
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: 10 }}
                    transition={{ duration: 0.5 }}
                    className="absolute top-0 h-full right-2 flex justify-center items-center pointer-events-none">
                    <div className="text-xs rounded-md px-2 bg-green-200 text-green-800">
                        Copied!
                    </div>
                </motion.div>
            )}
        </AnimatePresence>
    </div>
}

type ITableRow = {
    row: Row<Record<string, string | number>>;
    style: CSSProperties;
    onRowUpdate?: (row: Record<string, string | number>, updatedColumn: string) => Promise<void>;
    checked?: boolean;
    onRowCheck?: (value: boolean) => void;
    disableEdit?: boolean;
}

const TableRow: FC<ITableRow> = ({ row, style, onRowUpdate, checked, onRowCheck, disableEdit }) => {
    const handleCellUpdate = useCallback((cell: Cell<Record<string, string | number>, unknown> & { editedData?: string }) => {
        if (onRowUpdate == null) {
            return Promise.reject();
        }
        const updatedRow = row.getAllCells().reduce((all, one) => {
            all[one.column.id] = one.getValue() as string | number;
            return all;
        }, {} as Record<string, string | number>);
        if (cell.editedData !== undefined) {
            updatedRow[cell.column.id] = cell.editedData;
        }
        return onRowUpdate?.(updatedRow, cell.column.id);
    }, [onRowUpdate, row]);

    return (
        <div className="table-row-group text-xs group/row" style={style} data-testid="table-row">
            {
                row.getVisibleCells().map((cell) => (
                    <TData key={cell.id} cell={cell} onCellUpdate={handleCellUpdate}
                        disableEdit={disableEdit || cell.column.id === "#"}
                        checked={checked}
                        onRowCheck={onRowCheck} />
                ))
            }
        </div>
    )
}

type ITableProps = {
    className?: string;
    columns: string[];
    columnTags?: string[];
    rows: string[][];
    totalPages: number;
    currentPage: number;
    onPageChange?: (page: number) => void;
    onRowUpdate?: (row: Record<string, string | number>, updatedColumn: string) => Promise<void>;
    disableEdit?: boolean;
    checkedRows?: Set<number>;
    setCheckedRows?: (checkedRows: Set<number>) => void;
    hideActions?: boolean;
}

export const Table: FC<ITableProps> = ({ className, columns: actualColumns, rows: actualRows, columnTags, totalPages, currentPage, onPageChange, onRowUpdate, disableEdit, checkedRows, setCheckedRows, hideActions }) => {
    const fixedTableRef = useRef<FixedSizeList>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const operationsRef = useRef<HTMLDivElement>(null);
    const tableRef = useRef<HTMLTableElement>(null);
    const [sorting, setSorting] = useState<SortingState>([]);
    const [search, setSearch] = useState("");
    const [searchIndex, setSearchIndex] = useState(0);
    const [height, setHeight] = useState(0);
    const [width, setWidth] = useState(0);
    const [data, setData] = useState<Record<string, string | number>[]>([]);

    const columns = useMemo<ColumnDef<Record<string, string | number>>[]>(() => {
        const indexWidth = 50;
        const colWidth = Math.max(((width - indexWidth)/actualColumns.length), 150);
        const headerCount: Record<string, number> = {};
        const cols: ColumnDef<Record<string, string | number>>[] = actualColumns.map((col) => {
            if (headerCount[col] == null) {
                headerCount[col] = 0;
            } else {
                headerCount[col] += 1;
            }

            const id = headerCount[col] > 0 ? `${col}-${headerCount[col]}` : col;

            return {
                id,
                header: col,
                accessorKey: id,
                size: colWidth,
            };
        });
        cols.unshift({
            id: "#",
            header: "#",
            accessorKey: "#",
            size: indexWidth + 10,
        });
        return cols;
    }, [actualColumns, width]);

    useEffect(() => {
        setData(actualRows.map((row, rowIndex) => {
            const newRow = row.reduce((all, one, colIndex) => {
                if (actualColumns[colIndex] === "#") {
                    all[actualColumns[colIndex]] = one;
                } else if (columns[colIndex+1]) {
                    all[columns[colIndex+1].id || columns[colIndex+1].accessorKey as string] = one;
                }
                return all;
            }, { "#": (rowIndex+1+(currentPage-1)*actualRows.length).toString() } as Record<string, string | number>);
            newRow.originalIndex = rowIndex;
            return newRow;
        }));
    }, [actualColumns, actualRows, currentPage, columns]);

    const table = useReactTable({
        columns,
        data,
        state: {
            sorting,
        },
        onSortingChange: setSorting,
        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
        sortingFns: {
            auto: (rowA, rowB, columnId) => {
                const aValue = rowA.getValue<string | number>(columnId);
                const bValue = rowB.getValue<string | number>(columnId);
                if (isString(aValue) && isString(bValue) && isNumeric(aValue) && isNumeric(bValue)) {
                    const aValueNumber = Number.parseFloat(aValue);
                    const bValueNumber = Number.parseFloat(bValue);
                    return aValueNumber - bValueNumber;
                }
                if (aValue < bValue) return -1;
                if (aValue > bValue) return 1;
                return 0;
            },
        },
    });

    const rows = table.getRowModel().rows;

    const rowCount = useMemo(() => {
        return rows.length ?? 0;
    }, [rows]);

    const handleKeyUp = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
        if (tableRef.current == null || search.length === 0) {
            return;
        }
        // @ts-ignore
        let interval: NodeJS.Timeout;
        if (e.key === "Enter") {
            const searchText = search.toLowerCase();
            const filteredToOriginalIndex = [];
            for (const [index, row] of rows.entries()) {
                const rowValues = row.getAllCells().map(cell => cell.getValue());
                for (const value of rowValues) {
                    if (value == null) {
                        continue;
                    }
                    const text = String(value).toLowerCase();
                    if (text != null && searchText != null && text.includes(searchText)) {
                        filteredToOriginalIndex.push(index);
                        break;
                    }
                }
            }
            
            if (rows.length > 0 &&  filteredToOriginalIndex.length > 0) {
                const newSearchIndex = (searchIndex + 1) % filteredToOriginalIndex.length;
                setSearchIndex(newSearchIndex);
                const originalIndex = filteredToOriginalIndex[newSearchIndex] + 1;
                fixedTableRef.current?.scrollToItem(originalIndex, "center");
                setTimeout(() => {
                    const currentVisibleRows = tableRef.current?.querySelectorAll(".table-row-group") ?? [];
                    for (const currentVisibleRow of currentVisibleRows) {
                        const text = currentVisibleRow.querySelector("div > span")?.textContent ?? "";
                        if (isNumeric(text)) {
                            const id = parseInt(text);
                            if (id === originalIndex) {
                                currentVisibleRow.classList.add("!bg-yellow-100", "dark:!bg-yellow-800");
                                interval = setTimeout(() => {
                                    currentVisibleRow.classList.remove("!bg-yellow-100", "dark:!bg-yellow-800");
                                }, 3000);
                            }
                        }
                    }
                }, 100);
            }
        }

        return () => {
            if (interval != null) {
                clearInterval(interval);
            }
        }
    }, [rows, search, searchIndex]);

    const handleSearchChange = useCallback((newValue: string) => {
        setSearchIndex(-1);
        setSearch(newValue);
    }, []);

    const handleSort = useCallback((columnId: string) => {
        setSorting((old) => {
            const existingSort = old.find(d => d.id === columnId);
            if (!existingSort) {
                return [{ id: columnId, desc: false }];
            }
            if (!existingSort.desc) {
                return [{ id: columnId, desc: true }];
            }
            return [];
        });
    }, []);

    const handleRowUpdate = useCallback((index: number, row: Record<string, string | number>, updatedColumn: string) => {
        if (onRowUpdate == null) {
            return Promise.resolve();
        }
        delete row["#"];
        return onRowUpdate(row, updatedColumn).then(() => {
            setData(value => {
                const newValue = clone(value);
                newValue[index] = clone(row);
                return newValue;
            });
        });
    }, [onRowUpdate]);

    const handleRowCheck = useCallback((index: number, value: boolean) => {
        const newCheckedRows = new Set(checkedRows);
        if (value) {
            newCheckedRows.add(index);
        } else {
            newCheckedRows.delete(index);
        }
        setCheckedRows?.(newCheckedRows);
    }, [checkedRows, setCheckedRows]);

    const handleRenderRow = useCallback(({ index, style }: ListChildComponentProps) => {
        const row = rows[index];
        const originalIndex = row.original.originalIndex as number;
        return <TableRow key={`row-${row.id}`} row={row} style={style}
            onRowUpdate={(row, updatedColumn) => handleRowUpdate(index, row, updatedColumn)}
            checked={checkedRows?.has(originalIndex)}
            onRowCheck={(value) => handleRowCheck(originalIndex, value)}
            disableEdit={disableEdit} />;
    }, [rows, checkedRows, disableEdit, handleRowUpdate, handleRowCheck]);

    useEffect(() => {
        if (containerRef.current == null || operationsRef.current == null) {
            return;
        }
        const { height, width } = containerRef.current.getBoundingClientRect();
        const padding = 60;
        setHeight(height - operationsRef.current.getBoundingClientRect().height - padding); 
        setWidth(width);
    }, []);

    const allChecked = useMemo(() => {
        return (checkedRows?.size ?? 0) === rows.length;
    }, [checkedRows?.size, rows.length]);;

    const handleCheckAll = useCallback(() => {
        if (setCheckedRows == null) {
            return;
        }
        if (allChecked) {
            return setCheckedRows(new Set<number>());
        }
        setCheckedRows(new Set(rows.map((_, i) => i)));
    }, [allChecked, rows, setCheckedRows]);

    const specificIndexes = useMemo(() => {
        return  [...checkedRows ?? []];
    }, [checkedRows]);

    const exportToCSV = useExportToCSV(actualColumns, rows.map(r => r.original), specificIndexes);

    return (
        <div className="flex flex-col grow gap-4 items-center w-full h-full" ref={containerRef}>
            <div className={classNames("flex justify-between items-center w-full", {
                "hidden": hideActions,
            })} ref={operationsRef}>
                <div>
                    <SearchInput search={search} setSearch={handleSearchChange} placeholder="Search through rows     [Press Enter]" inputProps={{
                        className: "w-[300px]",
                        onKeyUp: handleKeyUp,
                    }} testId="table-search" />
                </div>
                <div className="flex gap-4 items-center">
                    <div className="text-sm text-gray-600 dark:text-neutral-300"><span className="font-semibold">Count:</span> {rowCount}</div>
                    <AnimatedButton icon={Icons.Download} label={checkedRows != null && checkedRows?.size > 0 ? "Export selected" : "Export"} type="lg" onClick={exportToCSV} />
                </div>
            </div>
            <div className={twMerge(classNames("flex overflow-x-auto h-full", className))} style={{
                width,
            }} data-testid="table">
                <div className="border-separate border-spacing-0 h-fit" ref={tableRef}>
                    <div>
                        {table.getHeaderGroups().map(headerGroup => (
                            <div className="group/header-row" key={headerGroup.id}>
                                {headerGroup.headers.map((header, i) => {
                                    const sortingState = sorting.find(s => s.id === header.column.id);
                                    return (
                                        <div key={header.id} 
                                            style={{ width: header.column.getSize() }}
                                            className="inline-block text-xs border-t border-l last:border-r border-gray-200 dark:border-white/5 p-2 text-left bg-gray-500 dark:bg-white/20 text-white first:rounded-tl-lg last:rounded-tr-lg relative group/header cursor-pointer select-none">
                                            <div className={classNames({
                                                "group-hover/header-row:hidden": checkedRows != null && header.column.id === "#",
                                                "hidden": header.column.id === "#" && allChecked,
                                            })} onClick={() => handleSort(header.column.id)} data-testid="table-header">
                                                {flexRender(header.column.columnDef.header, header.getContext())} 
                                                {i > 0 && columnTags?.[i-1] != null && columnTags?.[i-1].length > 0 && <span className="text-[11px]">[{columnTags?.[i-1]}]</span>}
                                            </div>
                                            <div className={classNames("absolute top-0 left-0 h-full w-full justify-center items-center bg-transparent z-[1] hover:scale-110 transition-all", {
                                                "group-hover/header-row:flex": checkedRows != null && header.column.id === "#",
                                                "flex": header.column.id === "#" && allChecked,
                                                "hidden": checkedRows == null || header.column.id !== "#" || !allChecked,
                                            })}>
                                                <CheckBoxInput value={allChecked} setValue={handleCheckAll} />
                                            </div>
                                            <div className={twMerge(classNames("transition-all absolute top-2 right-2 opacity-0", {
                                                "opacity-100": sortingState !== undefined,
                                                "rotate-180": sortingState?.desc,
                                            }))}>
                                                {Icons.ArrowUp}
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        ))}
                    </div>
                    <div className="tbody">
                        <FixedSizeList
                            ref={fixedTableRef}
                            height={height}
                            itemCount={rows.length}
                            itemSize={31}
                            width="100%"
                        >
                            {handleRenderRow}
                        </FixedSizeList>
                    </div>
                </div>
            </div>
            {
                totalPages > 1 &&
                <div className="flex justify-center items-center">
                    <Pagination pageCount={totalPages} currentPage={currentPage} onPageChange={onPageChange} />
                </div>
            }
        </div>
    )
}
