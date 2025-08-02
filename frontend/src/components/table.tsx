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
import { Cell, Row, useBlockLayout, useTable } from 'react-table';
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
    cell: Cell<Record<string, string | number>>;
    onCellUpdate?: (row: Cell<Record<string, string | number>>) => Promise<void>;
    disableEdit?: boolean;
    checked?: boolean;
    onRowCheck?: (value: boolean) => void;
}

const TData: FC<ITDataProps> = ({ cell, onCellUpdate, checked, onRowCheck, disableEdit }) => {
    const [changed, setChanged] = useState(false);
    const [editedData, setEditedData] = useState<string>(cell.value);
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
        setEditedData(cell.value);
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
        let previousValue = cell.value;
        cell.value = editedData;
        setUpdating(true);
        onCellUpdate?.(cell).then(() => {
            setEditable(false);
            setCellRect(null);
        }).catch(() => {
            cell.value = previousValue;
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
        setEditedData(cell.value);
    }, [cell.value]);

    const props = useMemo(() => {
        return cell.getCellProps();
    }, [cell]);

    return <div ref={cellRef} {...props} key={props.key}
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
                "group-hover/row:flex": checked != null && cell.column.id === "#",
                "flex": cell.column.id === "#" && checked === true,
                "hidden": checked == null || cell.column.id !== "#" || checked === false,
            })}>
                <CheckBoxInput value={checked ?? false} setValue={onRowCheck} />
            </div>
            <div className={classNames({
                "group-hover/row:hidden": checked != null && cell.column.id === "#",
                "hidden": cell.column.id === "#" && checked === true,
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
    const handleCellUpdate = useCallback((cell: Cell<Record<string, string | number>>) => {
        if (onRowUpdate == null) {
            return Promise.reject();
        }
        const updatedRow = row.cells.reduce((all, one) => {
            all[one.column.id] = one.value;
            return all;
        }, {} as Record<string, string | number>);
        updatedRow[cell.column.id] = cell.value;
        return onRowUpdate?.(updatedRow, cell.column.id);
    }, [onRowUpdate, row.cells]);

    const props = useMemo(() => {
        return row.getRowProps({ style });
    }, [row, style]);

    return (
        <div className="table-row-group text-xs group/row" {...props} key={props.key} data-testid="table-row">
            {
                row.cells.map((cell) => (
                    <TData key={cell.getCellProps().key} cell={cell} onCellUpdate={handleCellUpdate}
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
    schema?: string;
    storageUnit?: string;
}

export const Table: FC<ITableProps> = ({ className, columns: actualColumns, rows: actualRows, columnTags, totalPages, currentPage, onPageChange, onRowUpdate, disableEdit, checkedRows, setCheckedRows, hideActions, schema, storageUnit }) => {
    const fixedTableRef = useRef<FixedSizeList>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const operationsRef = useRef<HTMLDivElement>(null);
    const tableRef = useRef<HTMLTableElement>(null);
    const [direction, setDirection] = useState<"asc" | "dsc">();
    const [sortedColumn, setSortedColumn] = useState<string>();
    const [search, setSearch] = useState("");
    const [searchIndex, setSearchIndex] = useState(0);
    const [height, setHeight] = useState(0);
    const [width, setWidth] = useState(0);
    const [data, setData] = useState<Record<string, string | number>[]>([]);
    const [showImportModal, setShowImportModal] = useState(false);
    const [importMode, setImportMode] = useState<'append' | 'override'>('append');
    const [importFile, setImportFile] = useState<File | null>(null);
    const [importing, setImporting] = useState(false);
    const [showExportConfirm, setShowExportConfirm] = useState(false);
    const [importProgress, setImportProgress] = useState<{processedRows: number; status: string} | null>(null);

    const columns = useMemo(() => {
        const indexWidth = 50;
        const colWidth = Math.max(((width - indexWidth)/actualColumns.length), 150);
        const headerCount: Record<string, number> = {};
        const cols = actualColumns.map((col) => {
            if (headerCount[col] == null) {
                headerCount[col] = 0;
            } else {
                headerCount[col] += 1;
            }

            const id = headerCount[col] > 0 ? `${col}-${headerCount[col]}` : col;

            return {
                id,
                Header: col,
                accessor: id,
                width: colWidth,
            };
        });
        cols.unshift({
            id: "#",
            Header: "#",
            accessor: "#",
            width: indexWidth + 10,
        });
        return cols;
    }, [actualColumns, width]);

    useEffect(() => {
        setData(actualRows.map((row, rowIndex) => {
            const newRow = row.reduce((all, one, colIndex) => {
                if (actualColumns[colIndex] === "#") {
                    all[actualColumns[colIndex]] = one;
                } else {
                    all[columns[colIndex+1].accessor] = one;
                }
                return all;
            }, { "#": (rowIndex+1+(currentPage-1)*actualRows.length).toString() } as Record<string, string | number>);
            newRow.originalIndex = rowIndex;
            return newRow;
        }));
    }, [actualColumns, actualRows]);

    const sortedRows = useMemo(() => {
        if (!sortedColumn) {
            return data;
        }
        const newRows = [...data];
        newRows.sort((a, b) => {
            const aValue = a[sortedColumn];
            const bValue = b[sortedColumn];
            if (isString(aValue) && isString(bValue) && isNumeric(aValue) && isNumeric(bValue)) {
                const aValueNumber = Number.parseFloat(aValue);
                const bValueNumber = Number.parseFloat(bValue);
                return direction === 'asc' ? aValueNumber - bValueNumber : bValueNumber - aValueNumber;
            }

            if (aValue < bValue) {
                return direction === 'asc' ? -1 : 1;
            }
            
            if (aValue > bValue) {
                return direction === 'asc' ? 1 : -1;
            }
            return 0;
        });
        return newRows;
    }, [sortedColumn, direction, data]);

    const {
        getTableProps,
        getTableBodyProps,
        headerGroups,
        rows,
        prepareRow,
    } = useTable(
        {
            columns,
            data: sortedRows,
        },
        useBlockLayout,
    );

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
                for (const value of values(row.values)) {
                    if (value == null) {
                        continue;
                    }
                    const text = value.toLowerCase();
                    if (text != null && searchText != null && text.includes(searchText)) {
                        filteredToOriginalIndex.push(index);
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

    const handleSort = useCallback((columnToSort: string) => {
        const columnSelectedIsDifferent = columnToSort !== sortedColumn;
        if (!columnSelectedIsDifferent && direction === "dsc") {
            setDirection(undefined);
            return setSortedColumn(undefined);
        }
        setSortedColumn(columnToSort);
        if (direction == null || columnSelectedIsDifferent) {
            return setDirection("asc");
        }
        setDirection("dsc");
    }, [sortedColumn, direction]);

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
        prepareRow(row);
        const originalIndex = row.original.originalIndex as number;
        return <TableRow key={`row-${row.values[actualColumns[0]]}`} row={row} style={style}
            onRowUpdate={(row, updatedColumn) => handleRowUpdate(index, row, updatedColumn)}
            checked={checkedRows?.has(originalIndex)}
            onRowCheck={(value) => handleRowCheck(originalIndex, value)}
            disableEdit={disableEdit} />;
    }, [rows, prepareRow, actualColumns, checkedRows, disableEdit, handleRowUpdate, handleRowCheck]);

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

    const hasSelectedRows = (checkedRows?.size ?? 0) > 0;

    // Always call the hook, but use conditional logic inside
    const backendExport = useExportToCSV(schema || '', storageUnit || '', hasSelectedRows);

    const handleExportConfirm = useCallback(() => {
        if (!schema || !storageUnit) {
            // Fallback to client-side export if schema/storageUnit not provided
            const selectedRows = hasSelectedRows 
                ? [...checkedRows!].map(index => sortedRows[index])
                : sortedRows;
            
            const csvContent = [
                actualColumns.join('|'), 
                ...selectedRows.map(row => actualColumns.map(col => row[col] || '').join("|"))
            ].join('\n'); 
        
            const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        
            const link = document.createElement('a');
            if (link.download !== undefined) {
                const url = URL.createObjectURL(blob);
                link.setAttribute('href', url);
                link.setAttribute('download', `${storageUnit || 'data'}.csv`);
                link.style.visibility = 'hidden';
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);
            }
        } else {
            // Use backend export for full data
            backendExport();
        }
        setShowExportConfirm(false);
    }, [schema, storageUnit, hasSelectedRows, actualColumns, sortedRows, checkedRows, backendExport]);

    const handleImport = useCallback(async () => {
        if (!importFile || !schema || !storageUnit) {
            notify("Missing file or table information", "error");
            return;
        }

        setImporting(true);
        
        try {
            // Read file content
            const text = await importFile.text();
            
            // Convert to base64 for GraphQL transport
            const base64Data = btoa(text);
            
            // Call import mutation
            const response = await fetch('/graphql', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    query: `
                        mutation ImportCSV($schema: String!, $storageUnit: String!, $csvData: String!, $mode: ImportMode!) {
                            ImportCSV(schema: $schema, storageUnit: $storageUnit, csvData: $csvData, mode: $mode) {
                                totalRows
                                processedRows
                                status
                                error
                            }
                        }
                    `,
                    variables: {
                        schema,
                        storageUnit,
                        csvData: base64Data,
                        mode: importMode === 'append' ? 'Append' : 'Override'
                    }
                })
            });

            const result = await response.json();
            
            if (result.errors) {
                throw new Error(result.errors[0].message);
            }
            
            const importResult = result.data.ImportCSV;
            
            if (importResult.error) {
                throw new Error(importResult.error);
            }
            
            notify(`Successfully imported ${importResult.processedRows} rows`, "success");
            setShowImportModal(false);
            setImportFile(null);
            setImportProgress(null);
            
            // Refresh the table
            window.location.reload();
        } catch (error) {
            notify(`Import failed: ${error}`, "error");
        } finally {
            setImporting(false);
        }
    }, [importFile, schema, storageUnit, importMode]);

    const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (file && file.type === 'text/csv') {
            setImportFile(file);
        } else {
            notify("Please select a CSV file", "error");
        }
    }, []);

    return (
        <>
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
                    {schema && storageUnit && (
                        <AnimatedButton icon={Icons.Upload} label="Import" type="lg" onClick={() => setShowImportModal(true)} />
                    )}
                    <AnimatedButton icon={Icons.Download} label={hasSelectedRows ? `Export ${checkedRows?.size} selected` : "Export all"} type="lg" onClick={() => setShowExportConfirm(true)} />
                </div>
            </div>
            <div className={twMerge(classNames("flex overflow-x-auto h-full", className))} style={{
                width,
            }} data-testid="table">
                <div className="border-separate border-spacing-0 h-fit" ref={tableRef} {...getTableProps()}>
                    <div>
                        {headerGroups.map(headerGroup => (
                            <div className="group/header-row" {...headerGroup.getHeaderGroupProps()} key={headerGroup.getHeaderGroupProps().key}>
                                {headerGroup.headers.map((column, i) => (
                                    <>
                                        <div {...column.getHeaderProps()} key={column.getHeaderProps().key} className="text-xs border-t border-l last:border-r border-gray-200 dark:border-white/5 p-2 text-left bg-gray-500 dark:bg-white/20 text-white first:rounded-tl-lg last:rounded-tr-lg relative group/header cursor-pointer select-none">
                                            <div className={classNames({
                                                "group-hover/header-row:hidden": checkedRows != null && column.id === "#",
                                                "hidden": column.id === "#" && allChecked,
                                            })} onClick={() => handleSort(column.id)} data-testid="table-header">
                                                {column.render('Header')} {i > 0 && columnTags?.[i-1] != null && columnTags?.[i-1].length > 0 && <span className="text-[11px]">[{columnTags?.[i-1]}]</span>}
                                            </div>
                                            <div className={classNames("absolute top-0 left-0 h-full w-full justify-center items-center bg-transparent z-[1] hover:scale-110 transition-all", {
                                                "group-hover/header-row:flex": checkedRows != null && column.id === "#",
                                                "flex": column.id === "#" && allChecked,
                                                "hidden": checkedRows == null || column.id !== "#" || !allChecked,
                                            })}>
                                                <CheckBoxInput value={allChecked} setValue={handleCheckAll} />
                                            </div>
                                            <div className={twMerge(classNames("transition-all absolute top-2 right-2 opacity-0", {
                                                "opacity-100": sortedColumn === column.id,
                                                "rotate-180": direction === "dsc",
                                            }))}>
                                                {Icons.ArrowUp}
                                            </div>
                                        </div>
                                    </>
                                ))}
                            </div>
                        ))}
                    </div>
                    <div className="tbody" {...getTableBodyProps()}>
                        <FixedSizeList
                            ref={fixedTableRef}
                            height={height}
                            itemCount={sortedRows.length}
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
        <AnimatePresence>
            {showImportModal && (
                <Portal>
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
                        onClick={() => setShowImportModal(false)}
                    >
                        <motion.div
                            initial={{ scale: 0.9 }}
                            animate={{ scale: 1 }}
                            exit={{ scale: 0.9 }}
                            className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4"
                            onClick={(e) => e.stopPropagation()}
                        >
                            <h2 className="text-xl font-semibold mb-4 dark:text-white">Import CSV</h2>
                            
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium mb-2 dark:text-gray-300">
                                        Select CSV File
                                    </label>
                                    <input
                                        type="file"
                                        accept=".csv"
                                        onChange={handleFileSelect}
                                        className="block w-full text-sm text-gray-900 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg cursor-pointer bg-gray-50 dark:bg-gray-700 focus:outline-none"
                                    />
                                    {importFile && (
                                        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                                            Selected: {importFile.name}
                                        </p>
                                    )}
                                </div>
                                
                                <div>
                                    <label className="block text-sm font-medium mb-2 dark:text-gray-300">
                                        Import Mode
                                    </label>
                                    <div className="space-y-2">
                                        <label className="flex items-center">
                                            <input
                                                type="radio"
                                                value="append"
                                                checked={importMode === 'append'}
                                                onChange={(e) => setImportMode(e.target.value as 'append')}
                                                className="mr-2"
                                            />
                                            <span className="text-sm dark:text-gray-300">Append to existing data</span>
                                        </label>
                                        <label className="flex items-center">
                                            <input
                                                type="radio"
                                                value="override"
                                                checked={importMode === 'override'}
                                                onChange={(e) => setImportMode(e.target.value as 'override')}
                                                className="mr-2"
                                            />
                                            <span className="text-sm dark:text-gray-300">Override existing data</span>
                                        </label>
                                    </div>
                                    {importMode === 'override' && (
                                        <p className="mt-2 text-sm text-yellow-600 dark:text-yellow-400">
                                            ⚠️ Warning: This will delete all existing data in the table
                                        </p>
                                    )}
                                </div>
                                
                                <div className="text-sm text-gray-600 dark:text-gray-400">
                                    <p className="font-medium mb-1">CSV Format Requirements:</p>
                                    <ul className="list-disc list-inside space-y-1">
                                        <li>First row must contain column headers with format: column_name:data_type</li>
                                        <li>Use pipe (|) as delimiter</li>
                                        <li>All required columns must be present</li>
                                    </ul>
                                </div>
                                
                                {importing && (
                                    <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
                                        <div className="flex items-center gap-2">
                                            <Loading hideText={true} />
                                            <span className="text-sm text-blue-700 dark:text-blue-300">
                                                Importing data...
                                            </span>
                                        </div>
                                    </div>
                                )}
                            </div>
                            
                            <div className="flex justify-end gap-2 mt-6">
                                <AnimatedButton
                                    icon={Icons.Cancel}
                                    label="Cancel"
                                    onClick={() => {
                                        setShowImportModal(false);
                                        setImportFile(null);
                                    }}
                                    disabled={importing}
                                />
                                <AnimatedButton
                                    icon={Icons.Upload}
                                    label={importing ? "Importing..." : "Import"}
                                    onClick={handleImport}
                                    disabled={!importFile || importing}
                                    className="bg-blue-500 text-white hover:bg-blue-600"
                                />
                            </div>
                        </motion.div>
                    </motion.div>
                </Portal>
            )}
        </AnimatePresence>
        <AnimatePresence>
            {showExportConfirm && (
                <Portal>
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
                        onClick={() => setShowExportConfirm(false)}
                    >
                        <motion.div
                            initial={{ scale: 0.9 }}
                            animate={{ scale: 1 }}
                            exit={{ scale: 0.9 }}
                            className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4"
                            onClick={(e) => e.stopPropagation()}
                        >
                            <h2 className="text-xl font-semibold mb-4 dark:text-white">Export CSV</h2>
                            
                            <div className="space-y-4">
                                <p className="text-gray-600 dark:text-gray-300">
                                    {hasSelectedRows 
                                        ? `You are about to export ${checkedRows?.size} selected rows.`
                                        : `You are about to export all data from the table. This may take some time for large tables.`}
                                </p>
                                
                                <div className="text-sm text-gray-600 dark:text-gray-400">
                                    <p className="font-medium mb-1">Export Format:</p>
                                    <ul className="list-disc list-inside space-y-1">
                                        <li>CSV format with pipe (|) delimiter</li>
                                        <li>Headers include column names and data types</li>
                                        <li>UTF-8 encoding with BOM for Excel compatibility</li>
                                    </ul>
                                </div>
                            </div>
                            
                            <div className="flex justify-end gap-2 mt-6">
                                <AnimatedButton
                                    icon={Icons.Cancel}
                                    label="Cancel"
                                    onClick={() => setShowExportConfirm(false)}
                                />
                                <AnimatedButton
                                    icon={Icons.Download}
                                    label="Export"
                                    onClick={handleExportConfirm}
                                    className="bg-blue-500 text-white hover:bg-blue-600"
                                />
                            </div>
                        </motion.div>
                    </motion.div>
                </Portal>
            )}
        </AnimatePresence>
    </>
    )
}
