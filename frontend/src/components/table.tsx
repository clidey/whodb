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
    EmptyState,
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
    Spinner,
    TableCell,
    Table as TableComponent,
    TableHead,
    TableHeader,
    TableHeadRow,
    TableRow,
    TextArea,
    toast,
    VirtualizedTableBody
} from "@clidey/ux";
import { useDeleteRowMutation, useGenerateMockDataMutation, useMockDataMaxRowCountQuery } from '@graphql';
import { FC, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Export } from "./export";
import {
    ArrowDownCircleIcon,
    ArrowDownTrayIcon,
    CalculatorIcon,
    CalendarIcon,
    CheckCircleIcon,
    ChevronDownIcon,
    ChevronUpIcon,
    CircleStackIcon,
    ClockIcon,
    CursorArrowRaysIcon,
    DocumentDuplicateIcon,
    DocumentIcon,
    DocumentTextIcon,
    EllipsisHorizontalIcon,
    EllipsisVerticalIcon,
    HashtagIcon,
    KeyIcon,
    ListBulletIcon,
    PencilSquareIcon,
    ShareIcon,
    TrashIcon,
    XMarkIcon
} from "./heroicons";
import { Tip } from "./tip";

// Dynamically load EE Export component
// const EEExport = loadEEComponent(
//     () => import('@ee/components/export').then(mod => ({ default: mod.Export })),
//     null,
// );

const EEExport = null;

// Dynamic Export component that uses EE version if available, otherwise CE version
const DynamicExport: FC<{
    open: boolean;
    onOpenChange: (open: boolean) => void;
    schema: string;
    storageUnit: string;
    hasSelectedRows: boolean;
    selectedRowsData?: Record<string, any>[];
    checkedRowsCount: number;
}> = (props) => {
    // Use EE Export if available, otherwise fall back to CE Export
    const ExportComponent = EEExport || Export;
    return <ExportComponent {...props} />;
};

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

// Render platform-specific shortcut labels: ⌘ on macOS, Ctrl on others
const isMacPlatform = typeof navigator !== 'undefined' && /Mac|iPhone|iPad|iPod/.test(navigator.platform);
function renderShortcut(parts: ("Mod" | "Shift" | "Delete" | string)[]) {
    const mapMac: Record<string, string> = { Mod: "⌘", Shift: "⇧", Delete: "⌫" };
    const mapWin: Record<string, string> = { Mod: "Ctrl", Shift: "Shift", Delete: "Del" };
    const map = isMacPlatform ? mapMac : mapWin;
    if (isMacPlatform) {
        return parts.map(p => map[p] || p).join("");
    }
    return parts.map(p => map[p] || p).join("+");
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
    pageSize?: number;
    // Server-side pagination props
    totalCount?: number;
    currentPage?: number;
    onPageChange?: (page: number) => void;
    showPagination?: boolean;
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
    pageSize = 100,
    // Server-side pagination props
    totalCount,
    currentPage: serverCurrentPage,
    onPageChange,
    showPagination = false,
}) => {
    const [editIndex, setEditIndex] = useState<number | null>(null);
    const [editRow, setEditRow] = useState<string[] | null>(null);
    const [editRowInitialLengths, setEditRowInitialLengths] = useState<number[]>([]);
    const [deleting, setDeleting] = useState(false);
    const [checked, setChecked] = useState<number[]>([]);
    const [showExportConfirm, setShowExportConfirm] = useState(false);
    const tableRef = useRef<HTMLDivElement>(null);
    const [contextMenuCellIdx, setContextMenuCellIdx] = useState<number | null>(null);
    
    // Mock data state
    const [showMockDataSheet, setShowMockDataSheet] = useState(false);
    const [mockDataRowCount, setMockDataRowCount] = useState("100");
    const [mockDataMethod, setMockDataMethod] = useState("Normal");
    const [mockDataOverwriteExisting, setMockDataOverwriteExisting] = useState("append");
    const [showMockDataConfirmation, setShowMockDataConfirmation] = useState(false);
    const { data: maxRowData } = useMockDataMaxRowCountQuery();
    const maxRowCount = maxRowData?.MockDataMaxRowCount || 200;
    
    // Use server-side pagination
    const currentPage = serverCurrentPage || 1;
    const totalRows = totalCount || 0;
    const totalPages = Math.ceil(totalRows / pageSize);

    const [generateMockData, { loading: generatingMockData }] = useGenerateMockDataMutation();
    const [deleteRow, ] = useDeleteRowMutation();
    const [containerWidth, setContainerWidth] = useState<number>(0);
    const lastSearchState = useRef<{ search: string; matchIdx: number }>({ search: '', matchIdx: 0 });

    const handleEdit = (index: number) => {
        setEditIndex(index);
        const rowData = [...rows[index]];
        setEditRow(rowData);
        // Store initial lengths to prevent input/textarea switching
        setEditRowInitialLengths(rowData.map(cell => cell?.length || 0));
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
                    setEditRowInitialLengths([]);
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
            // Convert array of arrays to array of objects with column names as keys
            return checked.map(idx => {
                const row = rows[idx];
                const rowObj: Record<string, any> = {};
                columns.forEach((col, colIdx) => {
                    rowObj[col] = row[colIdx];
                });
                return rowObj;
            });
        }
        return undefined;
    }, [hasSelectedRows, checked, rows, columns]);

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

    const paginatedRows = useMemo(() => {
        // For server-side pagination, rows are already paginated
        return rows;
    }, [rows]);

    const handlePageChange = useCallback((newPage: number) => {
        onPageChange?.(newPage);
    }, [onPageChange]);

    const renderPaginationLinks = () => {
        const links = [];
        // Show up to 3 pages before and after current
        const start = Math.max(1, currentPage - 2);
        const end = Math.min(totalPages, currentPage + 2);

        if (start > 1) {
            links.push(
                <PaginationItem key={1}>
                    <PaginationLink href="#" onClick={e => { e.preventDefault(); handlePageChange(1); }} size="sm">1</PaginationLink>
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
                        onClick={e => { e.preventDefault(); handlePageChange(i); }}
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
                    <PaginationLink href="#" onClick={e => { e.preventDefault(); handlePageChange(totalPages); }} size="sm">{totalPages}</PaginationLink>
                </PaginationItem>
            );
        }

        return links;
    };

    const handleSelectRow = useCallback((rowIndex: number) => {
        setChecked(checked.includes(rowIndex) ? checked.filter(i => i !== rowIndex) : [...checked, rowIndex]);
    }, [checked]);

    // Track click timeouts to prevent single-click when double-click occurs
    const clickTimeouts = useRef<Map<string, number>>(new Map());

    const handleCellClick = useCallback((rowIndex: number, cellIndex: number) => {
        const cellKey = `${rowIndex}-${cellIndex}`;
        
        // Clear any existing timeout for this cell
        const existingTimeout = clickTimeouts.current.get(cellKey);
        if (existingTimeout) {
            clearTimeout(existingTimeout);
        }
        
        // Set a new timeout for the single-click action
        const timeout = setTimeout(() => {
            const cell = paginatedRows[rowIndex][cellIndex];
            if (cell !== undefined && cell !== null) {
                if (typeof navigator !== "undefined" && navigator.clipboard) {
                    navigator.clipboard.writeText(String(cell));
                    toast.success("Copied to clipboard");
                }
            }
            clickTimeouts.current.delete(cellKey);
        }, 200); // 200ms delay to detect double-click
        
        clickTimeouts.current.set(cellKey, timeout);
    }, [paginatedRows]);

    const handleCellDoubleClick = useCallback((rowIndex: number) => {
        // Clear any pending single-click timeouts for all cells in this row
        for (let cellIdx = 0; cellIdx < columns.length; cellIdx++) {
            const cellKey = `${rowIndex}-${cellIdx}`;
            const timeout = clickTimeouts.current.get(cellKey);
            if (timeout) {
                clearTimeout(timeout);
                clickTimeouts.current.delete(cellKey);
            }
        }
        
        const row = paginatedRows[rowIndex];
        if (row && Array.isArray(row)) {
            const rowString = row.map(cell => cell ?? "").join("\t");
            if (typeof navigator !== "undefined" && navigator.clipboard) {
                navigator.clipboard.writeText(rowString);
                toast.success("Row copied to clipboard");
            }
        }
    }, [paginatedRows, columns.length]);


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
        // For databases without schemas (like SQLite), only storageUnit is required
        if (!storageUnit) {
            toast.error("Storage unit is required for mock data generation");
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
                        Schema: schema || "",  // Use empty string if schema is null/undefined (SQLite case)
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
                toast.error(`Failed to mock data`);
            }
        } catch (error: any) {
            if (error.message === "mock data generation is not allowed for this table") {
                toast.error("Mock data generation is not allowed for this table");
            } else {
                toast.error(`Failed to mock data: ${error.message}`);
            }
        }
    }, [generateMockData, schema, storageUnit, mockDataRowCount, mockDataMethod, mockDataOverwriteExisting, showMockDataConfirmation, maxRowCount]);

    const columnIcons = useMemo(() => getColumnIcons(columns, columnTypes), [columns, columnTypes]);

    // Cleanup click timeouts on unmount
    useEffect(() => {
        return () => {
            // Clear all pending timeouts
            clickTimeouts.current.forEach(timeout => clearTimeout(timeout));
            clickTimeouts.current.clear();
        };
    }, []);

    // Listen for menu export trigger
    useEffect(() => {
        const handleExportTrigger = () => {
            setShowExportConfirm(true);
        };

        window.addEventListener('menu:trigger-export', handleExportTrigger);
        return () => {
            window.removeEventListener('menu:trigger-export', handleExportTrigger);
        };
    }, []);

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

    // Keyboard shortcuts for table operations
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            // Only handle shortcuts when not in input fields
            if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) {
                return;
            }

            const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
            const cmdKey = isMac ? event.metaKey : event.ctrlKey;

            if (cmdKey) {
                switch (event.key.toLowerCase()) {
                    case 'm':
                        event.preventDefault();
                        setShowMockDataSheet(true);
                        break;
                    case 'r':
                        event.preventDefault();
                        onRefresh?.();
                        break;
                    case 'a':
                        event.preventDefault();
                        // For server-side pagination, select all visible rows
                        setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index));
                        break;
                    case 'e':
                        event.preventDefault();
                        setShowExportConfirm(true);
                        break;
                }
            }
        };

        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [onRefresh, checked, paginatedRows, currentPage, pageSize]);



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

    const contextMenu = useCallback((index: number, style: React.CSSProperties) => {
        return <ContextMenu key={index}>
            <ContextMenuTrigger className="contents">
                <TableRow data-row-idx={index} className="group relative" style={style}>
                    <TableCell className={cn("min-w-[40px] w-[40px]", {
                        "hidden": disableEdit,
                    })}>
                        <Checkbox
                            checked={checked.includes(index)}
                            onCheckedChange={() => setChecked(checked.includes(index) ? checked.filter(i => i !== index) : [...checked, index])}
                        />
                        <Button variant="secondary" className="opacity-0 group-hover:opacity-100 absolute right-2 w-0 top-1.5" onClick={(e) => {
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
                    </TableCell>
                    {paginatedRows[index]?.map((cell, cellIdx) => (
                        <TableCell
                            key={cellIdx}
                            className="cursor-pointer"
                            onClick={() => handleCellClick(index, cellIdx)}
                            onDoubleClick={() => handleCellDoubleClick(index)}
                            onContextMenu={() => setContextMenuCellIdx(cellIdx)}
                            data-col-idx={cellIdx}
                        >
                            {cell}
                        </TableCell>
                    ))}
                </TableRow>
            </ContextMenuTrigger>
            <ContextMenuContent
                className="w-52 max-h-[calc(100vh-2rem)] overflow-y-auto"
                collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
            >
                <ContextMenuItem
                    onSelect={() => {
                        if (contextMenuCellIdx == null) return;
                        const cell = paginatedRows[index]?.[contextMenuCellIdx];
                        if (cell !== undefined && cell !== null) {
                            if (typeof navigator !== "undefined" && navigator.clipboard) {
                                navigator.clipboard.writeText(String(cell));
                                toast.success("Copied cell to clipboard");
                            }
                        }
                    }}
                    disabled={contextMenuCellIdx == null}
                >
                    <DocumentDuplicateIcon className="w-4 h-4" />
                    Copy Cell
                    <ContextMenuShortcut><CursorArrowRaysIcon className="w-4 h-4" /></ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuItem
                    onSelect={() => {
                        const row = paginatedRows[index];
                        if (row && Array.isArray(row)) {
                            const rowString = row.map(cell => cell ?? "").join("\t");
                            if (typeof navigator !== "undefined" && navigator.clipboard) {
                                navigator.clipboard.writeText(rowString);
                                toast.success("Copied row to clipboard");
                            }
                        }
                    }}
                    className="[&>[data-slot='context-menu-shortcut']]:flex"
                >
                    <DocumentTextIcon className="w-4 h-4" />
                    Copy Row
                    <ContextMenuShortcut><CursorArrowRaysIcon className="w-4 h-4" /><CursorArrowRaysIcon className="w-4 h-4" /></ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuItem onSelect={() => handleSelectRow(index)}>
                    {checked.includes(index) ? (
                        <>
                            <CheckCircleIcon className="w-4 h-4 text-primary" />
                            Deselect Row
                            <ContextMenuShortcut>{renderShortcut(["Mod", "D"])}</ContextMenuShortcut>
                        </>
                    ) : (
                        <>
                            <CheckCircleIcon className="w-4 h-4 text-primary" />
                            Select Row
                            <ContextMenuShortcut>{renderShortcut(["Mod", "S"])}</ContextMenuShortcut>
                        </>
                    )}
                </ContextMenuItem>
                <ContextMenuItem onSelect={() => handleEdit(index)} disabled={checked.length > 0} data-testid="context-menu-edit-row">
                    <PencilSquareIcon className="w-4 h-4" />
                    Edit Row
                    <ContextMenuShortcut>{renderShortcut(["Mod", "E"])}</ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuSub>
                    <ContextMenuSubTrigger>
                        <ArrowDownTrayIcon className="w-4 h-4 mr-2" />
                        Export
                    </ContextMenuSubTrigger>
                    <ContextMenuSubContent
                        collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
                    >
                        <ContextMenuItem
                            onSelect={() => setShowExportConfirm(true)}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            Export All as CSV
                            <ContextMenuShortcut>{renderShortcut(["Mod", "Shift", "C"])}</ContextMenuShortcut>
                        </ContextMenuItem>
                        <ContextMenuItem
                            onSelect={() => setShowExportConfirm(true)}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            Export All as Excel
                            <ContextMenuShortcut>{renderShortcut(["Mod", "Shift", "X"])}</ContextMenuShortcut>
                        </ContextMenuItem>
                        <ContextMenuSeparator />
                        <ContextMenuItem
                            onSelect={() => setShowExportConfirm(true)}
                            disabled={checked.length === 0}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            Export Selected as CSV
                            <ContextMenuShortcut>{renderShortcut(["Mod", "C"])}</ContextMenuShortcut>
                        </ContextMenuItem>
                        <ContextMenuItem
                            onSelect={() => setShowExportConfirm(true)}
                            disabled={checked.length === 0}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            Export Selected as Excel
                            <ContextMenuShortcut>{renderShortcut(["Mod", "X"])}</ContextMenuShortcut>
                        </ContextMenuItem>
                    </ContextMenuSubContent>
                </ContextMenuSub>
                <ContextMenuItem
                    onSelect={() => setShowMockDataSheet(true)}
                >
                    <DocumentDuplicateIcon className="w-4 h-4" />
                    Mock Data
                    <ContextMenuShortcut>{renderShortcut(["Mod", "M"])}</ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuSub>
                    <ContextMenuSubTrigger data-testid="context-menu-more-actions">
                        <EllipsisHorizontalIcon className="w-4 h-4 mr-2" />
                        More Actions
                    </ContextMenuSubTrigger>
                    <ContextMenuSubContent
                        className="w-44"
                        collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
                    >
                        <ContextMenuItem
                            variant="destructive"
                            disabled={deleting}
                            onSelect={async () => {
                                await handleDeleteRow(index);
                            }}
                            data-testid="context-menu-delete-row"
                        >
                            <TrashIcon className="w-4 h-4 text-destructive" />
                            Delete Row
                            <ContextMenuShortcut>{renderShortcut(["Mod", "Delete"])}</ContextMenuShortcut>
                        </ContextMenuItem>
                    </ContextMenuSubContent>
                </ContextMenuSub>
                <ContextMenuSeparator />
                <ContextMenuItem disabled={true}>
                    <ShareIcon className="w-4 h-4" />
                    Open in Graph
                    <ContextMenuShortcut>{renderShortcut(["Mod", "G"])}</ContextMenuShortcut>
                </ContextMenuItem>
            </ContextMenuContent>
        </ContextMenu>
    }, [checked, handleCellClick, handleEdit, handleSelectRow, handleDeleteRow, paginatedRows, disableEdit, onRefresh]);

    return (
        <div ref={tableRef} className="h-full flex">
            <div className="flex flex-col h-full space-y-4 w-0" style={{
                width: `${containerWidth}px`,
            }}>
                <TableComponent>
                    <TableHeader>
                        <ContextMenu>
                            <ContextMenuTrigger asChild>
                                <TableHeadRow className="group relative cursor-context-menu hover:bg-muted/50 transition-colors" title="Right-click for table options">
                                    <TableHead className={cn("min-w-[40px] w-[40px] relative", {
                                        "hidden": disableEdit,
                                    })}>
                                        <Checkbox
                                            checked={checked.length === paginatedRows.length}
                                            onCheckedChange={() => {
                                                setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index));
                                            }}
                                        />
                                        <Button variant="secondary" className="opacity-0 group-hover:opacity-100 absolute right-2 top-1.5 w-0" onClick={(e) => {
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
                                                <p className="flex items-center gap-xs">
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
                                </TableHeadRow>
                            </ContextMenuTrigger>
                            <ContextMenuContent
                className="w-64 max-h-[calc(100vh-2rem)] overflow-y-auto"
                collisionPadding={{ top: 16, right: 16, bottom: 16, left: 16 }}
            >
                                <ContextMenuItem onSelect={() => setShowMockDataSheet(true)} data-testid="context-menu-mock-data">
                                    <CalculatorIcon className="w-4 h-4" />
                                    Mock Data
                                    <ContextMenuShortcut>{renderShortcut(["Mod", "M"])}</ContextMenuShortcut>
                                </ContextMenuItem>
                                <ContextMenuSeparator />
                                <ContextMenuSub>
                                    <ContextMenuSubTrigger>
                                        <ArrowDownCircleIcon className="w-4 h-4 mr-2" />
                                        Export Data
                                    </ContextMenuSubTrigger>
                                    <ContextMenuSubContent
                        collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
                    >
                                        <ContextMenuItem
                                            onSelect={() => setShowExportConfirm(true)}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            Export All as CSV
                                        </ContextMenuItem>
                                        <ContextMenuItem
                                            onSelect={() => setShowExportConfirm(true)}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            Export All as Excel
                                        </ContextMenuItem>
                                        <ContextMenuSeparator />
                                        <ContextMenuItem
                                            onSelect={() => setShowExportConfirm(true)}
                                            disabled={checked.length === 0}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            Export Selected as CSV
                                        </ContextMenuItem>
                                        <ContextMenuItem
                                            onSelect={() => setShowExportConfirm(true)}
                                            disabled={checked.length === 0}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            Export Selected as Excel
                                        </ContextMenuItem>
                                    </ContextMenuSubContent>
                                </ContextMenuSub>
                                <ContextMenuSeparator />
                                <ContextMenuItem onSelect={() => onRefresh?.()}>
                                    <CircleStackIcon className="w-4 h-4" />
                                    Refresh Data
                                    <ContextMenuShortcut>{renderShortcut(["Mod", "R"])}</ContextMenuShortcut>
                                </ContextMenuItem>
                                <ContextMenuItem 
                                    onSelect={() => {
                                        setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index));
                                    }}
                                >
                                    <CheckCircleIcon className="w-4 h-4" />
                                    {checked.length === paginatedRows.length ? "Deselect All" : "Select All"}
                                    <ContextMenuShortcut>{renderShortcut(["Mod", "A"])}</ContextMenuShortcut>
                                </ContextMenuItem>
                                <ContextMenuSeparator />
                                <ContextMenuItem disabled={true}>
                                    <CalculatorIcon className="w-4 h-4" />
                                    Column Statistics
                                    <ContextMenuShortcut>{renderShortcut(["Mod", "S"])}</ContextMenuShortcut>
                                </ContextMenuItem>
                                <ContextMenuItem disabled={true}>
                                    <DocumentTextIcon className="w-4 h-4" />
                                    Schema Information
                                </ContextMenuItem>
                            </ContextMenuContent>
                        </ContextMenu>
                    </TableHeader>
                    {paginatedRows.length > 0 && (
                        <VirtualizedTableBody
                            rowCount={paginatedRows.length}
                            rowHeight={rowHeight}
                            height={Math.min(Math.min(height, window.innerHeight * 0.5), paginatedRows.length * rowHeight)}
                            overscan={10}
                        >
                            {(rowIdx: number, rowStyle: React.CSSProperties) => contextMenu(rowIdx, rowStyle)}
                        </VirtualizedTableBody>
                    )}
                </TableComponent>
                {paginatedRows.length === 0 && (
                    <ContextMenu>
                        <ContextMenuTrigger asChild>
                            <div className="flex items-center justify-center h-full min-h-[500px] cursor-pointer">
                                <EmptyState title="No data available" description="No data available" icon={<DocumentTextIcon className="w-4 h-4" />} />
                            </div>
                        </ContextMenuTrigger>
                        <ContextMenuContent
                className="w-52 max-h-[calc(100vh-2rem)] overflow-y-auto"
                collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
            >
                            <ContextMenuItem onSelect={() => setShowMockDataSheet(true)} className={cn({
                                "hidden": disableEdit,
                            })}>
                                <CalculatorIcon className="w-4 h-4" />
                                Mock Data
                                <ContextMenuShortcut>{renderShortcut(["Mod", "G"])}</ContextMenuShortcut>
                            </ContextMenuItem>
                            <ContextMenuSub>
                                <ContextMenuSubTrigger>
                                    <ArrowDownCircleIcon className="w-4 h-4 mr-2" />
                                    Export
                                </ContextMenuSubTrigger>
                                <ContextMenuSubContent
                        collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
                    >
                                    <ContextMenuItem
                                        onSelect={() => setShowExportConfirm(true)}
                                    >
                                        <DocumentIcon className="w-4 h-4" />
                                        Export All as CSV
                                        <ContextMenuShortcut>⌘C</ContextMenuShortcut>
                                    </ContextMenuItem>
                                    <ContextMenuItem
                                        onSelect={() => setShowExportConfirm(true)}
                                    >
                                        <DocumentIcon className="w-4 h-4" />
                                        Export All as Excel
                                        <ContextMenuShortcut>⌘E</ContextMenuShortcut>
                                    </ContextMenuItem>
                                </ContextMenuSubContent>
                            </ContextMenuSub>
                        </ContextMenuContent>
                    </ContextMenu>
                )}
                <div className={cn("flex justify-between items-center", {
                    "justify-end": children == null,
                    "mt-4": children != null,
                })}>
                    {children}
                    <Pagination className={cn("flex justify-end", {
                        "hidden": !showPagination,
                    })}>
                        <PaginationContent>
                            <PaginationItem>
                                <PaginationPrevious
                                    href="#"
                                    onClick={e => {
                                        e.preventDefault();
                                        if (currentPage > 1) handlePageChange(currentPage - 1);
                                    }}
                                    aria-disabled={currentPage === 1}
                                    size="sm"
                                    className={cn({
                                        "opacity-50 pointer-events-none": currentPage === 1,
                                    })}
                                />
                            </PaginationItem>
                            {renderPaginationLinks()}
                            <PaginationItem>
                                <PaginationNext
                                    href="#"
                                    onClick={e => {
                                        e.preventDefault();
                                        if (currentPage < totalPages) handlePageChange(currentPage + 1);
                                    }}
                                    aria-disabled={currentPage === totalPages}
                                    size="sm"
                                    className={cn({
                                        "opacity-50 pointer-events-none": currentPage === totalPages,
                                    })}
                                />
                            </PaginationItem>
                        </PaginationContent>
                    </Pagination>
                </div>
                <div className="flex justify-end items-center mb-2">
                    <Button
                        variant="secondary"
                        onClick={() => setShowExportConfirm(true)}
                        className="flex gap-sm"
                    >
                        <ArrowDownCircleIcon className="w-4 h-4" />
                        {hasSelectedRows ? `Export ${checked.length} selected` : "Export all"}
                    </Button>
                </div>
                <Sheet open={editIndex !== null} onOpenChange={open => { 
                    if (!open) {
                        setEditIndex(null);
                        setEditRow(null);
                        setEditRowInitialLengths([]);
                    }
                }}>
                    <SheetContent side="right" className="w-[400px] max-w-full p-8 flex flex-col">
                        <SheetTitle>Edit Row</SheetTitle>
                        <div className="flex-1 overflow-y-auto mt-4">
                            <div className="flex flex-col gap-lg pr-2">
                                {editRow &&
                                    columns.map((col, idx) => (
                                        <div key={col} className="flex flex-col gap-2">
                                            <Label>{col}</Label>
                                            {
                                                editRowInitialLengths[idx] < 50 ?
                                                    <Input
                                                        key={`input-${idx}`}
                                                        value={editRow[idx] ?? ""}
                                                        onChange={e => handleInputChange(e.target.value, idx)}
                                                        data-testid={`editable-field-${idx}`}
                                                    />
                                                    : <TextArea
                                                        key={`textarea-${idx}`}
                                                        value={editRow[idx] ?? ""}
                                                        onChange={e => handleInputChange(e.target.value, idx)}
                                                        rows={5}
                                                        className="min-h-[100px]"
                                                        data-testid={`editable-field-${idx}`}
                                                    />
                                            }
                                        </div>
                                    ))}
                            </div>
                        </div>
                        <SheetFooter className="flex gap-sm px-0 mt-4">
                            <Button
                                className="flex-1"
                                variant="secondary"
                                onClick={() => {
                                    setEditIndex(null);
                                    setEditRow(null);
                                    setEditRowInitialLengths([]);
                                }}
                                data-testid="cancel-edit-row"
                            >
                                Cancel
                            </Button>
                            <Button className="flex-1" onClick={handleUpdate} disabled={!editRow} data-testid="update-button">
                                Update
                            </Button>
                        </SheetFooter>
                    </SheetContent>
                </Sheet>
            </div>
            <Sheet open={showMockDataSheet} onOpenChange={(open) => {
                setShowMockDataSheet(open);
                    if (!open) {
                        setShowMockDataConfirmation(false);
                    }
                }}>
                <SheetContent side="right" className="p-8">
                    <div className="flex flex-col gap-lg h-full">
                        <SheetTitle className="flex items-center gap-2"><CalculatorIcon className="w-4 h-4" /> Mock Data</SheetTitle>
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
                                    <div className="mt-8 flex justify-center">
                                        <Spinner />
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
                    <SheetFooter className="flex gap-sm px-0">
                        <Alert variant="info" className="mb-4">
                            <AlertTitle>Note</AlertTitle>
                            <AlertDescription>
                                Mock data generation does not yet fully support foreign keys and all constraints. You may experience some errors or missing data.
                            </AlertDescription>
                        </Alert>
                        <Button
                            className="flex-1"
                            variant="secondary"
                            onClick={() => setShowMockDataSheet(false)}
                            data-testid="cancel-mock-data"
                        >
                            Cancel
                        </Button>
                        {!showMockDataConfirmation ? (
                            <Button className="flex-1" onClick={handleMockDataGenerate} disabled={generatingMockData}>
                                Generate
                            </Button>
                        ) : (
                            <Button className="flex-1" onClick={handleMockDataGenerate} disabled={generatingMockData} variant="destructive">
                                Yes, Overwrite
                            </Button>
                        )}
                    </SheetFooter>
                </SheetContent>
            </Sheet>
            <Suspense fallback={<Spinner />}>
                <DynamicExport
                    open={showExportConfirm}
                    onOpenChange={setShowExportConfirm}
                    schema={schema || ''}
                    storageUnit={storageUnit || ''}
                    hasSelectedRows={hasSelectedRows}
                    selectedRowsData={selectedRowsData}
                    checkedRowsCount={checked.length}
                />
            </Suspense>
        </div>
    );
};
