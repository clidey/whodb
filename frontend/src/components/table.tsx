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

import {
    Alert,
    AlertDescription,
    AlertTitle,
    Button,
    Checkbox,
    cn,
    ContextMenu,
    ContextMenuContent,
    ContextMenuItem,
    ContextMenuSeparator,
    ContextMenuShortcut,
    ContextMenuSub,
    ContextMenuSubContent,
    ContextMenuSubTrigger,
    ContextMenuTrigger,
    Input,
    Label,
    Pagination,
    PaginationContent,
    PaginationEllipsis,
    PaginationItem,
    PaginationLink,
    PaginationNext,
    PaginationPrevious,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Sheet,
    SheetContent,
    SheetFooter,
    SheetTitle,
    Table as TableComponent,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
    toast,
    VirtualizedTableBody,
    EmptyState
} from "@clidey/ux";
import {
    ArrowDownCircleIcon,
    CalculatorIcon,
    CalendarIcon,
    CheckCircleIcon,
    ChevronDownIcon,
    ChevronUpIcon,
    CircleStackIcon,
    ClockIcon,
    DocumentDuplicateIcon,
    DocumentIcon,
    DocumentTextIcon,
    EllipsisVerticalIcon,
    HashtagIcon,
    KeyIcon,
    ListBulletIcon,
    XMarkIcon,
} from "@heroicons/react/24/outline";
import {FC, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {useExportToCSV} from "./hooks"; // You may need to adjust this import
import {useDeleteRowMutation, useGenerateMockDataMutation, useMockDataMaxRowCountQuery} from '@graphql';
import {Tip} from "./tip";

// Type sets based on core/src/plugins/gorm/utils.go
const stringTypes = new Set([
    "TEXT", "STRING", "VARCHAR", "CHAR"
]);
const intTypes = new Set([
    "INTEGER", "SMALLINT", "BIGINT", "INT", "TINYINT", "MEDIUMINT", "INT4", "INT8", "INT16", "INT32", "INT64"
]);
const uintTypes = new Set([
    "TINYINT UNSIGNED", "SMALLINT UNSIGNED", "MEDIUMINT UNSIGNED", "BIGINT UNSIGNED", "UINT8", "UINT16", "UINT32", "UINT64"
]);
const floatTypes = new Set([
    "REAL", "NUMERIC", "DOUBLE PRECISION", "FLOAT", "NUMBER", "DOUBLE", "DECIMAL"
]);
const boolTypes = new Set([
    "BOOLEAN", "BIT", "BOOL"
]);
const dateTypes = new Set([
    "DATE"
]);
const dateTimeTypes = new Set([
    "DATETIME", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE", "DATETIME2", "SMALLDATETIME", "TIMETZ", "TIMESTAMPTZ"
]);
const uuidTypes = new Set([
    "UUID"
]);
const binaryTypes = new Set([
    "BLOB", "BYTEA", "VARBINARY", "BINARY", "IMAGE", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB"
]);


export function getColumnIcons(columns: string[], columnTypes?: string[]) {
    return columns.map((col, idx) => {
        const type = columnTypes?.[idx]?.toUpperCase?.() || "";
        if (intTypes.has(type) || uintTypes.has(type)) return <HashtagIcon className="w-4 h-4" />;
        if (floatTypes.has(type)) return <CalculatorIcon className="w-4 h-4" />;
        if (boolTypes.has(type)) return <CheckCircleIcon className="w-4 h-4" />;
        if (dateTypes.has(type)) return <CalendarIcon className="w-4 h-4" />;
        if (dateTimeTypes.has(type)) return <ClockIcon className="w-4 h-4" />;
        if (uuidTypes.has(type)) return <KeyIcon className="w-4 h-4" />;
        if (binaryTypes.has(type)) return <DocumentDuplicateIcon className="w-4 h-4" />;
        if (type.startsWith("ARRAY")) return <ListBulletIcon className="w-4 h-4" />;
        if (stringTypes.has(type)) return <DocumentTextIcon className="w-4 h-4" />;
        return <CircleStackIcon className="w-4 h-4" />;
    });
}

interface TableProps {
    columns: string[];
    columnTypes?: string[];
    rows: string[][];
    rowHeight?: number;
    height?: number;
    onRowUpdate?: (row: Record<string, string | number>, originalRow?: Record<string, string | number>) => Promise<void>;
    disableEdit?: boolean;
    schema?: string;
    storageUnit?: string;
    onRefresh?: () => void;
    children?: React.ReactNode;
    onColumnSort?: (column: string) => void;
    sortedColumns?: Map<string, 'asc' | 'desc'>;
    searchRef?: React.MutableRefObject<(search: string) => void>;
}

export const StorageUnitTable: FC<TableProps> = ({
    columns,
    columnTypes,
    rows,
    rowHeight = 48,
    height = 500,
    onRowUpdate,
    disableEdit = false,
    schema,
    storageUnit,
    onRefresh,
    children,
    onColumnSort,
    sortedColumns,
    searchRef,
}) => {
    const [editIndex, setEditIndex] = useState<number | null>(null);
    const [editRow, setEditRow] = useState<string[] | null>(null);
    const [deleting, setDeleting] = useState(false);
    const [checked, setChecked] = useState<number[]>([]);
    const [currentPage, setCurrentPage] = useState(1);
    const [showExportConfirm, setShowExportConfirm] = useState(false);
    const [exportDelimiter, setExportDelimiter] = useState(',');
    const [exportFormat, setExportFormat] = useState<'csv' | 'excel'>('csv');
    const tableRef = useRef<HTMLDivElement>(null);
    
    // Mock data state
    const [showMockDataSheet, setShowMockDataSheet] = useState(false);
    const [mockDataRowCount, setMockDataRowCount] = useState("100");
    const [mockDataMethod, setMockDataMethod] = useState("Normal");
    const [mockDataOverwriteExisting, setMockDataOverwriteExisting] = useState("append");
    const [showMockDataConfirmation, setShowMockDataConfirmation] = useState(false);
    const { data: maxRowData } = useMockDataMaxRowCountQuery();
    const maxRowCount = maxRowData?.MockDataMaxRowCount || 200;
    
    const pageSize = 20;
    const totalRows = rows.length;
    const totalPages = Math.ceil(totalRows / pageSize);
    
    const [generateMockData, { loading: generatingMockData }] = useGenerateMockDataMutation();
    const [deleteRow, ] = useDeleteRowMutation();
    const [containerWidth, setContainerWidth] = useState<number>(0);
    const lastSearchState = useRef<{ search: string; matchIdx: number }>({ search: '', matchIdx: 0 });



    // Export options as lists
    const exportFormatOptions = [
        { value: 'csv', label: 'CSV - Comma Separated Values' },
        { value: 'excel', label: 'Excel - XLSX Format' },
    ] as const;

    const exportDelimiterOptions = [
        { value: ',', label: 'Comma (,) - Standard CSV' },
        { value: ';', label: 'Semicolon (;) - Excel in some locales' },
        { value: '|', label: 'Pipe (|) - Less common in data' },
    ] as const;

    const handleEdit = (index: number) => {
        setEditIndex(index);
        setEditRow([...rows[index]]);
    };

    const handleInputChange = (value: string, idx: number) => {
        if (editRow) {
            const updated = [...editRow];
            updated[idx] = value;
            setEditRow(updated);
        }
    };

    const handleUpdate = useCallback(() => {
        if (editIndex !== null && editRow) {
            const updatedRow: Record<string, string | number> = {};
            columns.forEach((col, idx) => {
                updatedRow[col] = editRow[idx];
            });
            // Pass the original row as the second argument
            const originalRow: Record<string, string | number> = {};
            if (rows[editIndex]) {
                columns.forEach((col, idx) => {
                    originalRow[col] = rows[editIndex][idx];
                });
            }
            onRowUpdate?.(updatedRow, originalRow)
                .then(() => {
                    setEditIndex(null);
                    setEditRow(null);
                    toast.success("Row updated");
                    onRefresh?.();
                })
                .catch(() => {
                    toast.error("Error updating row");
                });
        }
    }, [editIndex, editRow, columns, onRowUpdate, rows, onRefresh]);

    // --- Export logic ---
    const hasSelectedRows = checked.length > 0;
    const selectedRowsData = useMemo(() => {
        if (hasSelectedRows) {
            return checked.map(idx => rows[idx]);
        }
        return undefined;
    }, [hasSelectedRows, checked, rows]);

    // Delete logic, adapted from explore-storage-unit.tsx
    const handleDeleteRow = useCallback(async (rowIndex: number) => {
        if (!rows || !columns) return;
        let unableToDeleteAll = false;
        const deletedIndexes: number[] = [];
        let indexesToDelete: number[] = [];
        if (Array.isArray(rowIndex)) {
            indexesToDelete = rowIndex;
        } else if (typeof rowIndex === "number") {
            indexesToDelete = [rowIndex];
        }
        if (selectedRowsData && selectedRowsData.length > 0) {
            indexesToDelete = selectedRowsData.map((_, idx) => idx);
        }
        if (indexesToDelete.length === 0) return;
        toast.info(indexesToDelete.length === 1 ? "Deleting row..." : "Deleting rows...");
        for (const index of indexesToDelete) {
            const row = rows[index];
            if (!row) continue;
            const values = columns.map((col, i) => ({
                Key: col,
                Value: row[i],
            }));
            try {
                await deleteRow({
                    variables: {
                        schema: schema || '',
                        storageUnit: storageUnit || '',
                        values,
                    },
                });
                deletedIndexes.push(index);
            } catch (e: any) {
                toast.error(`Unable to delete the row: ${e?.message || e}`);
                unableToDeleteAll = true;
                break;
            }
        }
        if (!unableToDeleteAll) {
            toast.success("Row deleted");
        }
        onRefresh?.();
    }, [deleteRow, schema, storageUnit, rows, columns, selectedRowsData, onRefresh]);

    const paginatedRows = useMemo(() => rows.slice((currentPage - 1) * pageSize, currentPage * pageSize), [rows, currentPage, pageSize]);

    const renderPaginationLinks = () => {
        const links = [];
        // Show up to 3 pages before and after current
        const start = Math.max(1, currentPage - 2);
        const end = Math.min(totalPages, currentPage + 2);

        if (start > 1) {
            links.push(
                <PaginationItem key={1}>
                    <PaginationLink href="#" onClick={e => { e.preventDefault(); setCurrentPage(1); }} size="sm">1</PaginationLink>
                </PaginationItem>
            );
            if (start > 2) {
                links.push(<PaginationEllipsis key="start-ellipsis" />);
            }
        }

        for (let i = start; i <= end; i++) {
            links.push(
                <PaginationItem key={i}>
                    <PaginationLink
                        href="#"
                        isActive={i === currentPage}
                        onClick={e => { e.preventDefault(); setCurrentPage(i); }}
                        size="sm"
                    >
                        {i}
                    </PaginationLink>
                </PaginationItem>
            );
        }

        if (end < totalPages) {
            if (end < totalPages - 1) {
                links.push(<PaginationEllipsis key="end-ellipsis" />);
            }
            links.push(
                <PaginationItem key={totalPages}>
                    <PaginationLink href="#" onClick={e => { e.preventDefault(); setCurrentPage(totalPages); }} size="sm">{totalPages}</PaginationLink>
                </PaginationItem>
            );
        }

        return links;
    };

    const handleSelectRow = useCallback((rowIndex: number) => {
        setChecked(checked.includes(rowIndex) ? checked.filter(i => i !== rowIndex) : [...checked, rowIndex]);
    }, [checked]);

    const handleCellClick = (rowIndex: number, cellIndex: number) => {
        const cell = paginatedRows[rowIndex][cellIndex];
        if (cell !== undefined && cell !== null) {
            if (typeof navigator !== "undefined" && navigator.clipboard) {
                navigator.clipboard.writeText(String(cell));
                toast.success("Copied to clipboard");
            }
        }
    };



    // Always call the hook, but use conditional logic inside
    const backendExport = useExportToCSV(schema || '', storageUnit || '', hasSelectedRows, exportDelimiter, selectedRowsData, exportFormat);

    const handleExportConfirm = useCallback(async () => {
        try {
            await backendExport();
            setShowExportConfirm(false);
        } catch (error: any) {
            toast.error(error.message || 'Export failed');
        }
    }, [backendExport]);

    // --- End export logic ---

    // Mock data handlers
    const handleMockDataRowCountChange = useCallback((value: string) => {
        // Only allow numeric input
        const numericValue = value.replace(/[^0-9]/g, '');
        const parsedValue = parseInt(numericValue) || 0;
        
        // Enforce max limit
        if (parsedValue > maxRowCount) {
            setMockDataRowCount(maxRowCount.toString());
            toast.error(`Maximum row count is ${maxRowCount}`);
        } else {
            setMockDataRowCount(numericValue);
        }
    }, [maxRowCount]);

    const handleMockDataGenerate = useCallback(async () => {
        if (!schema || !storageUnit) {
            toast.error("Schema and storage unit are required for mock data generation");
            return;
        }

        if (mockDataOverwriteExisting === "overwrite" && !showMockDataConfirmation) {
            setShowMockDataConfirmation(true);
            return;
        }

        const count = parseInt(mockDataRowCount) || 100;
        
        // Double-check the limit
        if (count > maxRowCount) {
            toast.error(`Row count cannot exceed ${maxRowCount}`);
            return;
        }
        
        try {
            const result = await generateMockData({
                variables: {
                    input: {
                        Schema: schema,
                        StorageUnit: storageUnit,
                        RowCount: count,
                        Method: mockDataMethod,
                        OverwriteExisting: mockDataOverwriteExisting === "overwrite",
                    }
                }
            });

            const data = result.data?.GenerateMockData;
            if (data?.AmountGenerated) {
                toast.success(`Successfully generated ${data.AmountGenerated} rows`);
                setShowMockDataSheet(false);
                setShowMockDataConfirmation(false);
                // Trigger a refresh by calling the onRefresh callback if provided
                if (onRefresh) {
                    onRefresh();
                }
            } else {
                toast.error(`Failed to generate mock data`);
            }
        } catch (error: any) {
            if (error.message === "mock data generation is not allowed for this table") {
                toast.error("Mock data generation is not allowed for this table");
            } else {
                toast.error(`Failed to generate mock data: ${error.message}`);
            }
        }
    }, [generateMockData, schema, storageUnit, mockDataRowCount, mockDataMethod, mockDataOverwriteExisting, showMockDataConfirmation, maxRowCount]);

    const columnIcons = useMemo(() => getColumnIcons(columns, columnTypes), [columns, columnTypes]);

    // Refresh page when it is resized and it settles
    useEffect(() => {
        let resizeTimeout: ReturnType<typeof setTimeout> | null = null;

        const handleResize = () => {
            if (resizeTimeout) clearTimeout(resizeTimeout);
            resizeTimeout = setTimeout(() => {
                if (onRefresh) {
                    onRefresh();
                }
            }, 300);
        };

        window.addEventListener('resize', handleResize);

        return () => {
            window.removeEventListener('resize', handleResize);
            if (resizeTimeout) clearTimeout(resizeTimeout);
        };
    }, [onRefresh]);



    useEffect(() => {
        if (tableRef.current) {
            setContainerWidth(tableRef.current.offsetWidth);
        }
    }, [tableRef]);

    // Highlight and scroll to the searched cell using document querySelector, no state needed

    useEffect(() => {
        if (!searchRef) return;

        let lastHighlightedCell: HTMLElement | null = null;

        searchRef.current = (search: string) => {
            // Remove any previous highlight
            document.querySelectorAll('.table-search-highlight').forEach(el => {
                el.classList.remove('bg-yellow-200', 'table-search-highlight', 'bg-muted');
            });

            // Remove highlight from the last highlighted cell if it exists
            if (lastHighlightedCell) {
                lastHighlightedCell.classList.remove('bg-muted', 'table-search-highlight', 'bg-yellow-200');
                lastHighlightedCell = null;
            }

            if (!search || !rows || !columns) {
                lastSearchState.current = { search: '', matchIdx: 0 };
                return;
            }

            // Find all matching cells
            const matches: { rowIdx: number; colIdx: number }[] = [];
            rows.forEach((row, rowIdx) => {
                row.forEach((cellValue, colIdx) => {
                    if (
                        cellValue !== undefined &&
                        cellValue !== null &&
                        cellValue.toString().toLowerCase().includes(search.toLowerCase())
                    ) {
                        matches.push({ rowIdx, colIdx });
                    }
                });
            });

            if (matches.length > 0) {
                // Determine which match to highlight
                let matchIdx = 0;
                if (lastSearchState.current.search === search) {
                    // Advance to next match, wrap around
                    matchIdx = (lastSearchState.current.matchIdx + 1) % matches.length;
                }
                // Update last search state
                lastSearchState.current = { search, matchIdx };

                const { rowIdx, colIdx } = matches[matchIdx];
                // Compose a unique selector for the cell
                const selector = `[data-row-idx="${rowIdx}"] [data-col-idx="${colIdx}"]`;
                const cell = document.querySelector(selector) as HTMLElement | null;
                if (cell) {
                    // Remove highlight from any previously highlighted cell
                    document.querySelectorAll('.table-search-highlight').forEach(el => {
                        el.classList.remove('bg-muted', 'table-search-highlight', 'bg-yellow-200');
                    });
                    cell.classList.add('bg-muted', 'table-search-highlight');
                    lastHighlightedCell = cell;
                    cell.scrollIntoView({ behavior: 'smooth', block: 'center', inline: 'center' });
                    setTimeout(() => {
                        if (cell === lastHighlightedCell) {
                            cell.classList.remove('bg-muted', 'table-search-highlight');
                            lastHighlightedCell = null;
                        }
                    }, 3000);
                }
            } else {
                // No matches, reset state
                lastSearchState.current = { search, matchIdx: 0 };
            }
        };

        // Cleanup on unmount
        return () => {
            if (lastHighlightedCell) {
                lastHighlightedCell.classList.remove('bg-muted', 'table-search-highlight', 'bg-yellow-200');
                lastHighlightedCell = null;
            }
        };
    }, [searchRef, rows, columns]);

    const contextMenu = useCallback((globalIndex: number, index: number) => {
        return <ContextMenu key={globalIndex}>
            <ContextMenuTrigger className="contents">
                <TableRow data-row-idx={index} className="group">
                    <TableCell className={cn({
                        "hidden": disableEdit,
                    })}>
                        <Checkbox
                            checked={checked.includes(globalIndex)}
                            onCheckedChange={() => setChecked(checked.includes(globalIndex) ? checked.filter(i => i !== globalIndex) : [...checked, globalIndex])}
                        />
                    </TableCell>
                    {paginatedRows[index]?.map((cell, cellIdx) => (
                        <TableCell key={cellIdx} className="cursor-pointer" onClick={() => handleCellClick(globalIndex, cellIdx)} data-col-idx={cellIdx}>{cell}</TableCell>
                    ))}
                    <Button variant="secondary" className="absolute right-6 opacity-0 group-hover:opacity-100 border border-input" onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        // Manually trigger context menu on this row
                        const event = new MouseEvent("contextmenu", {
                            bubbles: true,
                            clientX: e.clientX,
                            clientY: e.clientY,
                        });
                        e.currentTarget.dispatchEvent(event);
                        }} data-testid="icon-button">
                        <EllipsisVerticalIcon className="w-4 h-4" />
                    </Button>
                </TableRow>
            </ContextMenuTrigger>
            <ContextMenuContent className="w-52">
                <ContextMenuItem onSelect={() => handleSelectRow(globalIndex)}>
                    {checked.includes(globalIndex) ? "Deselect Row" : "Select Row"}
                </ContextMenuItem>
                <ContextMenuItem onSelect={() => handleEdit(globalIndex)} disabled={checked.length > 0} data-testid="context-menu-edit-row">
                    Edit Row
                    <ContextMenuShortcut>⌘E</ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuSub>
                    <ContextMenuSubTrigger>Export</ContextMenuSubTrigger>
                    <ContextMenuSubContent>
                        <ContextMenuItem
                            onSelect={() => { setShowExportConfirm(true); setExportFormat('csv'); }}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            Export All as CSV
                        </ContextMenuItem>
                        <ContextMenuItem
                            onSelect={() => { setShowExportConfirm(true); setExportFormat('excel'); }}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            Export All as Excel
                        </ContextMenuItem>
                        <ContextMenuSeparator />
                        <ContextMenuItem
                            onSelect={() => { setShowExportConfirm(true); setExportFormat('csv'); }}
                            disabled={checked.length === 0}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            Export Selected as CSV
                        </ContextMenuItem>
                        <ContextMenuItem
                            onSelect={() => { setShowExportConfirm(true); setExportFormat('excel'); }}
                            disabled={checked.length === 0}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            Export Selected as Excel
                        </ContextMenuItem>
                    </ContextMenuSubContent>
                </ContextMenuSub>
                <ContextMenuItem
                    onSelect={() => setShowMockDataSheet(true)}
                >
                    Generate Mock Data
                </ContextMenuItem>
                <ContextMenuSub>
                    <ContextMenuSubTrigger data-testid="context-menu-more-actions">More Actions</ContextMenuSubTrigger>
                    <ContextMenuSubContent className="w-44">
                        <ContextMenuItem
                            variant="destructive"
                            disabled={deleting}
                            onSelect={async () => {
                                await handleDeleteRow(globalIndex);
                            }}
                            data-testid="context-menu-delete-row"
                        >
                            Delete Row
                        </ContextMenuItem>
                    </ContextMenuSubContent>
                </ContextMenuSub>
                <ContextMenuSeparator />
                <ContextMenuItem disabled={true}>
                    Open in Graph View
                    <ContextMenuShortcut>⌘G</ContextMenuShortcut>
                </ContextMenuItem>
            </ContextMenuContent>
        </ContextMenu>
    }, []);

    return (
        <div ref={tableRef} className="h-full">
            <div className="flex flex-col h-full space-y-4 w-0" style={{
                width: `${containerWidth}px`,
            }}>
                <TableComponent className="overflow-x-auto">
                    <TableHeader>
                        <TableRow>
                            <TableHead className={cn({
                                "hidden": disableEdit,
                            })}>
                                <Checkbox
                                    checked={checked.length === paginatedRows.length}
                                    onCheckedChange={() => setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index + (currentPage - 1) * pageSize))}
                                />
                            </TableHead>
                            {columns.map((col, idx) => (
                                <TableHead
                                    key={col + idx} 
                                    icon={columnIcons?.[idx]}
                                    className={cn({
                                        "cursor-pointer select-none": onColumnSort,
                                    })}
                                    onClick={() => onColumnSort?.(col)}
                                >
                                    <Tip>
                                        <p className="flex items-center gap-1">
                                            {col}
                                            {onColumnSort && sortedColumns?.has(col) && (
                                                sortedColumns.get(col) === 'asc' 
                                                    ? <ChevronUpIcon className="w-4 h-4" />
                                                    : <ChevronDownIcon className="w-4 h-4" />
                                            )}
                                        </p>
                                        <p className="text-xs">{columnTypes?.[idx]}</p>
                                    </Tip>
                                </TableHead>
                            ))}
                        </TableRow>
                    </TableHeader>
                    {paginatedRows.length > 0 &&
                    <VirtualizedTableBody rowCount={paginatedRows.length} rowHeight={rowHeight} height={height}>
                        {(index) => {
                            const globalIndex = (currentPage - 1) * pageSize + index;
                            return contextMenu(globalIndex, index);
                        }}
                    </VirtualizedTableBody>}
                </TableComponent>
                {paginatedRows.length === 0 && (
                    <ContextMenu>
                        <ContextMenuTrigger asChild>
                            <div className="flex items-center justify-center h-full min-h-[500px] cursor-pointer">
                                <EmptyState title="No data available" description="No data available" icon={<DocumentTextIcon className="w-4 h-4" />} />
                            </div>
                        </ContextMenuTrigger>
                        <ContextMenuContent className="w-52">
                            <ContextMenuItem onSelect={() => setShowMockDataSheet(true)}>
                                Generate Mock Data
                            </ContextMenuItem>
                            <ContextMenuSub>
                                <ContextMenuSubTrigger>Export</ContextMenuSubTrigger>
                                <ContextMenuSubContent>
                                    <ContextMenuItem
                                        onSelect={() => { setShowExportConfirm(true); setExportFormat('csv'); }}
                                    >
                                        <DocumentIcon className="w-4 h-4" />
                                        Export All as CSV
                                    </ContextMenuItem>
                                    <ContextMenuItem
                                        onSelect={() => { setShowExportConfirm(true); setExportFormat('excel'); }}
                                    >
                                        <DocumentIcon className="w-4 h-4" />
                                        Export All as Excel
                                    </ContextMenuItem>
                                </ContextMenuSubContent>
                            </ContextMenuSub>
                        </ContextMenuContent>
                    </ContextMenu>
                )}

                <div className={cn("flex justify-between items-center mb-2", {
                    "justify-end": children == null,
                })}>
                    {children}
                    <Button
                        variant="secondary"
                        onClick={() => setShowExportConfirm(true)}
                        className="flex gap-2"
                    >
                        <ArrowDownCircleIcon className="w-4 h-4" />
                        {hasSelectedRows ? `Export ${checked.length} selected` : "Export all"}
                    </Button>
                </div>
                <div className="flex mt-4">
                    <Pagination className={cn("flex justify-end", {
                        "hidden": totalPages <= 1,
                    })}>
                        <PaginationContent>
                            <PaginationItem>
                                <PaginationPrevious
                                    href="#"
                                    onClick={e => {
                                        e.preventDefault();
                                        if (currentPage > 1) setCurrentPage(currentPage - 1);
                                    }}
                                    aria-disabled={currentPage === 1}
                                    size="sm"
                                />
                            </PaginationItem>
                            {renderPaginationLinks()}
                            <PaginationItem>
                                <PaginationNext
                                    href="#"
                                    onClick={e => {
                                        e.preventDefault();
                                        if (currentPage < totalPages) setCurrentPage(currentPage + 1);
                                    }}
                                    aria-disabled={currentPage === totalPages}
                                    size="sm"
                                />
                            </PaginationItem>
                        </PaginationContent>
                    </Pagination>
                </div>
                <Sheet open={editIndex !== null} onOpenChange={open => { if (!open) setEditIndex(null); }}>
                    <SheetContent side="right" className="w-[400px] max-w-full p-8">
                        <SheetTitle>Edit Row</SheetTitle>
                        <div className="flex flex-col gap-4 mt-4">
                            {editRow &&
                                columns.map((col, idx) => (
                                    <div key={col} className="flex flex-col gap-2" data-testid={`editable-field-${idx}`}>
                                        <Label>{col}</Label>
                                        <Input
                                            value={editRow[idx] ?? ""}
                                            onChange={e => handleInputChange(e.target.value, idx)}
                                        />
                                    </div>
                                ))}
                        </div>
                        <SheetFooter className="flex gap-2 px-0">
                            <Button onClick={handleUpdate} disabled={!editRow} data-testid="update-button">
                                Update
                            </Button>
                        </SheetFooter>
                    </SheetContent>
                </Sheet>
            </div>
            <Sheet open={showExportConfirm} onOpenChange={setShowExportConfirm}>
                <SheetContent side="right" className="max-w-md w-full p-8">
                    <div className="flex flex-col gap-4 grow">
                        <h2 className="text-xl font-semibold mb-4">Export Data</h2>
                        <div className="space-y-4 grow">
                            <p>
                                {hasSelectedRows
                                    ? `You are about to export ${checked.length} selected rows.`
                                    : `You are about to export all data from the table. This may take some time for large tables.`}
                            </p>
                            <div className="mb-4 flex flex-col gap-2">
                                <Label>
                                    Format
                                </Label>
                                <Select
                                    value={exportFormat}
                                    onValueChange={(value) => setExportFormat(value as 'csv' | 'excel')}
                                >
                                    <SelectTrigger className="w-full">
                                        <SelectValue>
                                            {
                                                exportFormatOptions.find(opt => opt.value === exportFormat)?.label
                                            }
                                        </SelectValue>
                                    </SelectTrigger>
                                    <SelectContent>
                                        {exportFormatOptions.map(opt => (
                                            <SelectItem key={opt.value} value={opt.value}>
                                                {opt.label}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                                {exportFormat === 'csv' && (
                                    <>
                                        <Label>
                                            Delimiter
                                        </Label>
                                        <Select
                                            value={exportDelimiter}
                                            onValueChange={(value) => setExportDelimiter(value)}
                                        >
                                            <SelectTrigger className="w-full">
                                                <SelectValue>
                                                    {
                                                        exportDelimiterOptions.find(opt => opt.value === exportDelimiter)?.label
                                                    }
                                                </SelectValue>
                                            </SelectTrigger>
                                            <SelectContent>
                                                {exportDelimiterOptions.map(opt => (
                                                    <SelectItem key={opt.value} value={opt.value}>
                                                        {opt.label}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                        <p className="text-sm mt-2">Choose a delimiter that doesn't appear in your data</p>
                                    </>
                                )}
                            </div>
                        </div>
                        <SheetFooter className="px-0">
                            <div className="text-xs text-muted-foreground mb-8">
                                <p className="font-medium mb-1">Export Details:</p>
                                <ul className="list-disc list-inside space-y-1">
                                    {exportFormat === 'csv' ? (
                                        <>
                                            <li><p className="inline-block">Headers include column names and data types</p></li>
                                            <li><p className="inline-block">UTF-8 encoding</p></li>
                                            <li><p className="inline-block">Customizable delimiter</p></li>
                                        </>
                                    ) : (
                                        <>
                                            <li><p className="inline-block">Excel XLSX format</p></li>
                                            <li><p className="inline-block">Formatted headers with styling</p></li>
                                            <li><p className="inline-block">Auto-sized columns</p></li>
                                        </>
                                    )}
                                </ul>
                            </div>
                            <Button onClick={handleExportConfirm}>
                                Export
                            </Button>
                        </SheetFooter>
                    </div>
                </SheetContent>
            </Sheet>
            <Sheet open={showMockDataSheet} onOpenChange={setShowMockDataSheet}>
                <SheetContent side="right" className="p-8">
                    <div className="flex flex-col gap-4 h-full">
                        <div className="text-lg font-semibold mb-2">Generate Mock Data for {storageUnit}</div>
                        {!showMockDataConfirmation ? (
                            <div className="space-y-4">
                                <Label>Number of Rows (max: {maxRowCount})</Label>
                                <Input
                                    value={mockDataRowCount}
                                    onChange={e => handleMockDataRowCountChange(e.target.value)}
                                    type="text"
                                    inputMode="numeric"
                                    pattern="[0-9]*"
                                    max={maxRowCount.toString()}
                                    placeholder={`Enter number of rows (1-${maxRowCount})`}
                                />
                                <Label>Method</Label>
                                <Select value={mockDataMethod} onValueChange={setMockDataMethod}>
                                    <SelectTrigger className="w-full">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="Normal">Normal</SelectItem>
                                    </SelectContent>
                                </Select>
                                <Label>Data Handling</Label>
                                <Select value={mockDataOverwriteExisting} onValueChange={setMockDataOverwriteExisting}>
                                    <SelectTrigger className="w-full">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="append">Append to existing data</SelectItem>
                                        <SelectItem value="overwrite">Overwrite existing data</SelectItem>
                                    </SelectContent>
                                </Select>
                                {generatingMockData && (
                                    <div className="mt-4">
                                        <div className="flex justify-center">
                                            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                                        </div>
                                        <p className="text-center text-sm text-gray-600 dark:text-gray-400 mt-2">
                                            Generating mock data...
                                        </p>
                                    </div>
                                )}
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <div className="flex items-center justify-center mb-4">
                                    <div className="w-16 h-16 bg-yellow-100 dark:bg-yellow-900 rounded-full flex items-center justify-center">
                                        <XMarkIcon className="w-8 h-8 text-yellow-600 dark:text-yellow-400" />
                                    </div>
                                </div>
                                <p className="text-center text-gray-700 dark:text-gray-300">
                                    Are you sure you want to overwrite all existing data in {storageUnit}? This action cannot be undone.
                                </p>
                            </div>
                        )}
                    </div>
                    <SheetFooter className="px-0">
                        <Alert variant="info" className="mb-4">
                            <AlertTitle>Note</AlertTitle>
                            <AlertDescription>
                                Mock data generation does not yet fully support foreign keys and all constraints. You may experience some errors or missing data.
                            </AlertDescription>
                        </Alert>
                        {!showMockDataConfirmation ? (
                            <Button onClick={handleMockDataGenerate} disabled={generatingMockData}>
                                Generate
                            </Button>
                        ) : (
                            <Button onClick={handleMockDataGenerate} disabled={generatingMockData} variant="destructive">
                                Yes, Overwrite
                            </Button>
                        )}
                    </SheetFooter>
                </SheetContent>
            </Sheet>
        </div>
    );
};
