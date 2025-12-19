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
    Table as TableComponent,
    TableCell,
    TableHead,
    TableHeader,
    TableHeadRow,
    TableRow,
    TextArea,
    toast,
    VirtualizedTableBody
} from "@clidey/ux";
import {useDeleteRowMutation, useGenerateMockDataMutation, useMockDataMaxRowCountQuery} from '@graphql';
import {FC, Suspense, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {Export} from "./export";
import {useTranslation} from '@/hooks/use-translation';
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
    CodeBracketIcon,
    CursorArrowRaysIcon,
    DocumentDuplicateIcon,
    DocumentIcon,
    DocumentTextIcon,
    EllipsisVerticalIcon,
    GlobeAltIcon,
    HashtagIcon,
    KeyIcon,
    ListBulletIcon,
    MagnifyingGlassIcon,
    PencilSquareIcon,
    ShareIcon,
    Squares2X2Icon,
    TrashIcon,
    XMarkIcon
} from "./heroicons";
import {Tip} from "./tip";
import {formatShortcut, isModKeyPressed} from "@/utils/platform";

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
    databaseType?: string;
}> = (props) => {
    // Use EE Export if available, otherwise fall back to CE Export
    const ExportComponent = EEExport || Export;
    return <ExportComponent {...props} />;
};

// Type sets for icon mapping
// Includes both canonical forms and common aliases for broad matching
const stringTypes = new Set([
    "TEXT", "STRING", "VARCHAR", "CHAR",
    "CHARACTER VARYING", "CHARACTER",
    "FIXEDSTRING",
]);
const intTypes = new Set([
    "INTEGER", "SMALLINT", "BIGINT", "INT", "TINYINT", "MEDIUMINT",
    "INT2", "INT4", "INT8",
    "INT16", "INT32", "INT64", "INT128", "INT256",
    "SERIAL", "BIGSERIAL", "SMALLSERIAL",
]);
const uintTypes = new Set([
    "TINYINT UNSIGNED", "SMALLINT UNSIGNED", "MEDIUMINT UNSIGNED", "BIGINT UNSIGNED",
    "UINT8", "UINT16", "UINT32", "UINT64", "UINT128", "UINT256",
]);
const floatTypes = new Set([
    "REAL", "NUMERIC", "DOUBLE PRECISION", "FLOAT", "NUMBER", "DOUBLE", "DECIMAL",
    "FLOAT4", "FLOAT8",
    "FLOAT32", "FLOAT64",
    "DECIMAL32", "DECIMAL64", "DECIMAL128", "DECIMAL256",
    "MONEY",
]);
const boolTypes = new Set([
    "BOOLEAN", "BIT", "BOOL",
]);
const dateTypes = new Set([
    "DATE",
    "DATE32",
]);
const dateTimeTypes = new Set([
    "DATETIME", "TIMESTAMP", "TIME",
    "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE",
    "TIME WITH TIME ZONE", "TIME WITHOUT TIME ZONE",
    "DATETIME2", "SMALLDATETIME",
    "TIMETZ", "TIMESTAMPTZ",
    "INTERVAL",
    "DATETIME64",
    "YEAR",
]);
const uuidTypes = new Set([
    "UUID",
]);
const binaryTypes = new Set([
    "BLOB", "BYTEA", "VARBINARY", "BINARY", "IMAGE",
    "TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
]);
const jsonTypes = new Set([
    "JSON", "JSONB",
]);
const networkTypes = new Set([
    "CIDR", "INET", "MACADDR", "MACADDR8",
    "IPV4", "IPV6",
]);
const geometryTypes = new Set([
    "POINT", "LINE", "LSEG", "BOX", "PATH", "POLYGON", "CIRCLE",
    "GEOMETRY", "GEOGRAPHY",
    "LINESTRING", "MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION",
]);
const xmlTypes = new Set([
    "XML",
]);

/**
 * Strips length/precision suffix from a type string.
 * e.g., "VARCHAR(255)" -> "VARCHAR", "DECIMAL(10,2)" -> "DECIMAL"
 */
function stripTypeSuffix(type: string): string {
    return type.replace(/\(.*\)$/, '').trim();
}


export function getColumnIcons(columns: string[], columnTypes?: string[], t?: (key: string) => string) {
    return columns.map((col, idx) => {
        const rawType = columnTypes?.[idx] || "";
        // Strip length/precision suffix and uppercase for matching
        const type = stripTypeSuffix(rawType).toUpperCase();

        if (intTypes.has(type) || uintTypes.has(type)) return <HashtagIcon className="w-4 h-4" aria-label={t?.('integerType') ?? 'Integer type'} />;
        if (floatTypes.has(type)) return <CalculatorIcon className="w-4 h-4" aria-label={t?.('decimalType') ?? 'Decimal type'} />;
        if (boolTypes.has(type)) return <CheckCircleIcon className="w-4 h-4" aria-label={t?.('booleanType') ?? 'Boolean type'} />;
        if (dateTypes.has(type)) return <CalendarIcon className="w-4 h-4" aria-label={t?.('dateType') ?? 'Date type'} />;
        if (dateTimeTypes.has(type)) return <ClockIcon className="w-4 h-4" aria-label={t?.('dateTimeType') ?? 'DateTime type'} />;
        if (uuidTypes.has(type)) return <KeyIcon className="w-4 h-4" aria-label={t?.('uuidType') ?? 'UUID type'} />;
        if (binaryTypes.has(type)) return <DocumentDuplicateIcon className="w-4 h-4" aria-label={t?.('binaryType') ?? 'Binary type'} />;
        if (jsonTypes.has(type)) return <CodeBracketIcon className="w-4 h-4" aria-label={t?.('jsonType') ?? 'JSON type'} />;
        if (networkTypes.has(type)) return <GlobeAltIcon className="w-4 h-4" aria-label={t?.('networkType') ?? 'Network type'} />;
        if (geometryTypes.has(type)) return <Squares2X2Icon className="w-4 h-4" aria-label={t?.('geometryType') ?? 'Geometry type'} />;
        if (xmlTypes.has(type)) return <CodeBracketIcon className="w-4 h-4" aria-label={t?.('xmlType') ?? 'XML type'} />;
        if (type.startsWith("ARRAY")) return <ListBulletIcon className="w-4 h-4" aria-label={t?.('arrayType') ?? 'Array type'} />;
        if (stringTypes.has(type)) return <DocumentTextIcon className="w-4 h-4" aria-label={t?.('textType') ?? 'Text type'} />;
        return <CircleStackIcon className="w-4 h-4" aria-label={t?.('dataType') ?? 'Data type'} />;
    });
}


interface TableProps {
    columns: string[];
    columnTypes?: string[];
    columnIsPrimary?: boolean[];
    columnIsForeignKey?: boolean[];
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
    // Foreign key functionality
    isValidForeignKey?: (columnName: string) => boolean;
    onEntitySearch?: (columnName: string, value: string) => void;
    databaseType?: string;
}

export const StorageUnitTable: FC<TableProps> = ({
    columns,
    columnTypes,
    columnIsPrimary,
    columnIsForeignKey,
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
    // Foreign key functionality
    isValidForeignKey,
    onEntitySearch,
    databaseType,
}) => {
    const { t } = useTranslation('components/table');
    const [editIndex, setEditIndex] = useState<number | null>(null);
    const [editRow, setEditRow] = useState<string[] | null>(null);
    const [editRowInitialLengths, setEditRowInitialLengths] = useState<number[]>([]);
    const [deleting, setDeleting] = useState(false);
    const [checked, setChecked] = useState<number[]>([]);
    const [showExportConfirm, setShowExportConfirm] = useState(false);
    const tableRef = useRef<HTMLDivElement>(null);
    const [contextMenuCellIdx, setContextMenuCellIdx] = useState<number | null>(null);

    // Keyboard navigation state
    const [focusedRowIndex, setFocusedRowIndex] = useState<number | null>(null);
    // Track focused column header for focus restoration after sort/refresh
    const focusedColumnRef = useRef<string | null>(null);
    
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

    // Accessibility: live region for screen reader announcements
    const [liveAnnouncement, setLiveAnnouncement] = useState<string>('');

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
                    toast.success(t('rowUpdated'));
                    onRefresh?.();
                })
                .catch(() => {
                    toast.error(t('errorUpdatingRow'));
                });
        }
    }, [editIndex, editRow, columns, onRowUpdate, rows, onRefresh, t]);

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
        if (checked.length > 0) {
            indexesToDelete = [...checked];
        }
        if (indexesToDelete.length === 0) return;
        toast.info(indexesToDelete.length === 1 ? t('deletingRow') : t('deletingRows'));
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
                toast.error(t('unableToDeleteRow', { message: e?.message || e }));
                unableToDeleteAll = true;
                break;
            }
        }
        if (!unableToDeleteAll) {
            toast.success(t('rowDeleted'));
        }
        onRefresh?.();
    }, [deleteRow, schema, storageUnit, rows, columns, checked, onRefresh, t]);

    const paginatedRows = useMemo(() => {
        // For server-side pagination, rows are already paginated
        return rows;
    }, [rows]);

    // Reset row focus and selection when rows change (page change, refresh, etc.)
    // But restore column header focus if one was focused (for keyboard sorting)
    useEffect(() => {
        setFocusedRowIndex(null);
        setChecked([]);

        // Restore column header focus after data refresh (for keyboard sorting UX)
        const columnToFocus = focusedColumnRef.current;
        if (columnToFocus) {
            // Delay needed for DOM to update after React render
            const timeoutId = setTimeout(() => {
                const header = document.querySelector(
                    `[data-testid="column-header-${columnToFocus}"]`
                ) as HTMLElement;
                if (header) {
                    header.focus();
                }
            }, 50);
            return () => clearTimeout(timeoutId);
        }
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
                    <PaginationLink href="#" onClick={e => { e.preventDefault(); handlePageChange(1); }} size="sm" data-testid="table-page-number" data-page="1" data-active={currentPage === 1} aria-label={t('goToPage', { page: 1 })}>1</PaginationLink>
                </PaginationItem>
            );
            if (start > 2) {
                links.push(<PaginationEllipsis key="start-ellipsis" aria-hidden="true" />);
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
                        data-testid="table-page-number"
                        data-page={i}
                        data-active={i === currentPage}
                        aria-label={i === currentPage ? t('currentPage', { page: i }) : t('goToPage', { page: i })}
                        aria-current={i === currentPage ? 'page' : undefined}
                    >
                        {i}
                    </PaginationLink>
                </PaginationItem>
            );
        }

        if (end < totalPages) {
            if (end < totalPages - 1) {
                links.push(<PaginationEllipsis key="end-ellipsis" aria-hidden="true" />);
            }
            links.push(
                <PaginationItem key={totalPages}>
                    <PaginationLink href="#" onClick={e => { e.preventDefault(); handlePageChange(totalPages); }} size="sm" data-testid="table-page-number" data-page={totalPages} data-active={currentPage === totalPages} aria-label={t('goToPage', { page: totalPages })}>{totalPages}</PaginationLink>
                </PaginationItem>
            );
        }

        return links;
    };

    const handleSelectRow = useCallback((rowIndex: number) => {
        const isCurrentlySelected = checked.includes(rowIndex);
        const newChecked = isCurrentlySelected ? checked.filter(i => i !== rowIndex) : [...checked, rowIndex];
        setChecked(newChecked);
        // Announce selection change to screen readers
        const selectedCount = newChecked.length;
        if (selectedCount === 0) {
            setLiveAnnouncement(t('noRowsSelected'));
        } else if (selectedCount === 1) {
            setLiveAnnouncement(t('oneRowSelected'));
        } else {
            setLiveAnnouncement(t('rowsSelected', { count: selectedCount }));
        }
    }, [checked, t]);

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
                    toast.success(t('copiedToClipboard'));
                }
            }
            clickTimeouts.current.delete(cellKey);
        }, 200); // 200ms delay to detect double-click

        clickTimeouts.current.set(cellKey, timeout);
    }, [paginatedRows, t]);

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
                toast.success(t('rowCopiedToClipboard'));
            }
        }
    }, [paginatedRows, columns.length, t]);


    // --- End export logic ---

    // Mock data handlers
    const handleMockDataRowCountChange = useCallback((value: string) => {
        // Only allow numeric input
        const numericValue = value.replace(/[^0-9]/g, '');
        const parsedValue = parseInt(numericValue) || 0;
        
        // Enforce max limit
        if (parsedValue > maxRowCount) {
            setMockDataRowCount(maxRowCount.toString());
            toast.error(t('maximumRowCount', { max: maxRowCount }));
        } else {
            setMockDataRowCount(numericValue);
        }
    }, [maxRowCount, t]);

    const handleMockDataGenerate = useCallback(async () => {
        // For databases without schemas (like SQLite), only storageUnit is required
        if (!storageUnit) {
            toast.error(t('storageUnitRequired'));
            return;
        }

        if (mockDataOverwriteExisting === "overwrite" && !showMockDataConfirmation) {
            setShowMockDataConfirmation(true);
            return;
        }

        const count = parseInt(mockDataRowCount) || 100;

        // Double-check the limit
        if (count > maxRowCount) {
            toast.error(t('rowCountExceedsMax', { max: maxRowCount }));
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
                toast.success(t('successfullyGenerated', { count: data.AmountGenerated }));
                setShowMockDataSheet(false);
                setShowMockDataConfirmation(false);
                // Trigger a refresh by calling the onRefresh callback if provided
                if (onRefresh) {
                    onRefresh();
                }
            } else {
                toast.error(t('failedToMockData'));
            }
        } catch (error: any) {
            if (error.message === "mock data generation is not allowed for this table") {
                toast.error(t('mockDataNotAllowed'));
            } else {
                toast.error(t('mockDataFailed', { message: error.message }));
            }
        }
    }, [generateMockData, schema, storageUnit, mockDataRowCount, mockDataMethod, mockDataOverwriteExisting, showMockDataConfirmation, maxRowCount, onRefresh, t]);

    const columnIcons = useMemo(() => getColumnIcons(columns, columnTypes, t), [columns, columnTypes, t]);

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

    // Helper to scroll focused row into view
    const scrollRowIntoView = useCallback((rowIndex: number) => {
        const rowElement = document.querySelector(`[data-row-idx="${rowIndex}"]`);
        if (rowElement) {
            rowElement.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }
    }, []);

    // Helper to move focus and optionally extend selection
    const moveFocus = useCallback((newIndex: number, extendSelection: boolean = false) => {
        if (newIndex < 0 || newIndex >= paginatedRows.length) return;

        if (extendSelection && focusedRowIndex !== null) {
            // Extend selection from current focus to new index
            const start = Math.min(focusedRowIndex, newIndex);
            const end = Math.max(focusedRowIndex, newIndex);
            const newChecked = new Set(checked);
            for (let i = start; i <= end; i++) {
                newChecked.add(i);
            }
            setChecked(Array.from(newChecked));
        }

        setFocusedRowIndex(newIndex);
        scrollRowIntoView(newIndex);
    }, [paginatedRows.length, focusedRowIndex, checked, scrollRowIntoView]);

    // Calculate visible rows for PageUp/PageDown
    const visibleRowCount = useMemo(() => {
        return Math.floor(height / rowHeight);
    }, [height, rowHeight]);

    // Keyboard navigation and shortcuts
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            // Only handle shortcuts when not in input fields
            if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) {
                return;
            }

            // Skip if no rows
            if (paginatedRows.length === 0) return;

            const cmdKey = isModKeyPressed(event);

            // Arrow key navigation (no modifier required)
            switch (event.key) {
                case 'ArrowDown':
                    event.preventDefault();
                    if (focusedRowIndex === null) {
                        moveFocus(0, event.shiftKey);
                    } else {
                        moveFocus(Math.min(focusedRowIndex + 1, paginatedRows.length - 1), event.shiftKey);
                    }
                    return;

                case 'ArrowUp':
                    event.preventDefault();
                    if (focusedRowIndex === null) {
                        moveFocus(paginatedRows.length - 1, event.shiftKey);
                    } else {
                        moveFocus(Math.max(focusedRowIndex - 1, 0), event.shiftKey);
                    }
                    return;

                case 'Home':
                    event.preventDefault();
                    moveFocus(0, event.shiftKey);
                    return;

                case 'End':
                    event.preventDefault();
                    moveFocus(paginatedRows.length - 1, event.shiftKey);
                    return;

                case 'PageDown':
                    event.preventDefault();
                    if (focusedRowIndex === null) {
                        moveFocus(Math.min(visibleRowCount - 1, paginatedRows.length - 1), event.shiftKey);
                    } else {
                        moveFocus(Math.min(focusedRowIndex + visibleRowCount, paginatedRows.length - 1), event.shiftKey);
                    }
                    return;

                case 'PageUp':
                    event.preventDefault();
                    if (focusedRowIndex === null) {
                        moveFocus(0, event.shiftKey);
                    } else {
                        moveFocus(Math.max(focusedRowIndex - visibleRowCount, 0), event.shiftKey);
                    }
                    return;

                case ' ': // Space - toggle selection of focused row
                    if (focusedRowIndex !== null) {
                        event.preventDefault();
                        handleSelectRow(focusedRowIndex);
                    }
                    return;

                case 'Enter': // Enter - edit focused row
                    if (focusedRowIndex !== null && !disableEdit) {
                        event.preventDefault();
                        handleEdit(focusedRowIndex);
                    }
                    return;

                case 'Escape':
                    // Clear focus and selection
                    event.preventDefault();
                    setFocusedRowIndex(null);
                    return;
            }

            // Modifier key combinations (Cmd/Ctrl)
            if (cmdKey) {
                // Handle Shift+Key combinations first
                if (event.shiftKey) {
                    switch (event.key.toLowerCase()) {
                        case 'e':
                            // Mod+Shift+E: Export (opens export dialog)
                            event.preventDefault();
                            setShowExportConfirm(true);
                            break;
                    }
                    return;
                }

                // Handle Cmd/Ctrl+Key combinations (without Shift)
                switch (event.key.toLowerCase()) {
                    case 'm':
                        // Mod+M: Mock data
                        event.preventDefault();
                        setShowMockDataSheet(true);
                        break;
                    case 'r':
                        // Mod+R: Refresh table
                        event.preventDefault();
                        onRefresh?.();
                        break;
                    case 'a':
                        // Mod+A: Select/deselect all visible rows
                        event.preventDefault();
                        setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index));
                        break;
                    case 'e':
                        // Mod+E: Edit focused row
                        if (focusedRowIndex !== null && !disableEdit) {
                            event.preventDefault();
                            handleEdit(focusedRowIndex);
                        }
                        break;
                    case 'backspace':
                    case 'delete':
                        // Mod+Delete/Backspace: Delete focused row
                        if (focusedRowIndex !== null && !disableEdit) {
                            event.preventDefault();
                            handleDeleteRow(focusedRowIndex);
                        }
                        break;
                    case 'arrowright':
                        // Mod+ArrowRight: Next page
                        if (onPageChange && currentPage < totalPages) {
                            event.preventDefault();
                            onPageChange(currentPage + 1);
                        }
                        break;
                    case 'arrowleft':
                        // Mod+ArrowLeft: Previous page
                        if (onPageChange && currentPage > 1) {
                            event.preventDefault();
                            onPageChange(currentPage - 1);
                        }
                        break;
                }
            }
        };

        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [onRefresh, checked, paginatedRows, handleDeleteRow, handleEdit, focusedRowIndex, moveFocus, visibleRowCount, handleSelectRow, disableEdit, onPageChange, currentPage, totalPages]);



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
                    if (cellValue !== undefined && cellValue !== null) {
                        // Trim trailing null characters for comparison (e.g., ClickHouse FixedString)
                        const searchValue = String(cellValue).replace(/\0+$/, '');
                        if (searchValue.toLowerCase().includes(search.toLowerCase())) {
                            matches.push({ rowIdx, colIdx });
                        }
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

                // Announce search result to screen readers
                setLiveAnnouncement(t('searchMatchFound', { current: matchIdx + 1, total: matches.length }));

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
                setLiveAnnouncement(t('noSearchMatches'));
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
        const isFocused = focusedRowIndex === index;
        const isSelected = checked.includes(index);

        return <ContextMenu key={index}>
            <ContextMenuTrigger className="contents">
                <TableRow
                    data-row-idx={index}
                    role="row"
                    aria-rowindex={index + 1}
                    aria-selected={isSelected}
                    data-focused={isFocused || undefined}
                    tabIndex={isFocused ? 0 : -1}
                    className={cn(
                        "group relative cursor-pointer outline-none",
                        // Focus styling - visible ring around focused row
                        isFocused && "ring-2 ring-primary ring-inset bg-primary/5",
                        // Selected styling
                        isSelected && "bg-muted"
                    )}
                    style={style}
                    onClick={() => setFocusedRowIndex(index)}
                    onFocus={() => setFocusedRowIndex(index)}
                >
                    <TableCell
                        role="gridcell"
                        className={cn("min-w-[40px] w-[40px]", {
                            "hidden": disableEdit,
                        })}
                    >
                        <Checkbox
                            checked={isSelected}
                            onCheckedChange={() => setChecked(isSelected ? checked.filter(i => i !== index) : [...checked, index])}
                            aria-label={isSelected ? t('deselectRow') : t('selectRow')}
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
                            }} data-testid="icon-button" aria-label={t('moreActions')}>
                            <EllipsisVerticalIcon className="w-4 h-4" />
                        </Button>
                    </TableCell>
                    {paginatedRows[index]?.map((cell, cellIdx) => {
                        // Trim trailing null characters (e.g., from ClickHouse FixedString)
                        const displayValue = typeof cell === 'string' ? cell.replace(/\0+$/, '') : cell;
                        return (
                            <TableCell
                                key={cellIdx}
                                role="gridcell"
                                className="cursor-pointer"
                                title={t('cellInteractionHint')}
                                onClick={(e) => {
                                    e.stopPropagation();
                                    setFocusedRowIndex(index);
                                    handleCellClick(index, cellIdx);
                                }}
                                onDoubleClick={() => handleCellDoubleClick(index)}
                                onContextMenu={() => setContextMenuCellIdx(cellIdx)}
                                data-col-idx={cellIdx}
                            >
                                {displayValue}
                            </TableCell>
                        );
                    })}
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
                                const copyValue = typeof cell === 'string' ? cell.replace(/\0+$/, '') : String(cell);
                                navigator.clipboard.writeText(copyValue);
                                toast.success(t('copiedCellToClipboard'));
                            }
                        }
                    }}
                    disabled={contextMenuCellIdx == null}
                >
                    <DocumentDuplicateIcon className="w-4 h-4" />
                    {t('copyCell')}
                    <ContextMenuShortcut><CursorArrowRaysIcon className="w-4 h-4" /></ContextMenuShortcut>
                </ContextMenuItem>
                {onEntitySearch && contextMenuCellIdx !== null && columnIsForeignKey?.[contextMenuCellIdx] && !columnIsPrimary?.[contextMenuCellIdx] && (
                    <ContextMenuItem
                        onSelect={() => {
                            if (contextMenuCellIdx == null) return;
                            const cell = paginatedRows[index]?.[contextMenuCellIdx];
                            const columnName = columns[contextMenuCellIdx];
                            if (cell !== undefined && cell !== null && columnName) {
                                onEntitySearch(columnName, String(cell));
                            }
                        }}
                    >
                        <MagnifyingGlassIcon className="w-4 h-4" />
                        {t('searchForEntity')}
                    </ContextMenuItem>
                )}
                <ContextMenuItem
                    onSelect={() => {
                        const row = paginatedRows[index];
                        if (row && Array.isArray(row)) {
                            const rowString = row.map(cell => cell ?? "").join("\t");
                            if (typeof navigator !== "undefined" && navigator.clipboard) {
                                navigator.clipboard.writeText(rowString);
                                toast.success(t('rowCopiedToClipboard'));
                            }
                        }
                    }}
                    className="[&>[data-slot='context-menu-shortcut']]:flex"
                >
                    <DocumentTextIcon className="w-4 h-4" />
                    {t('copyRow')}
                    <ContextMenuShortcut><CursorArrowRaysIcon className="w-4 h-4" /><CursorArrowRaysIcon className="w-4 h-4" /></ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuItem onSelect={() => handleSelectRow(index)}>
                    <CheckCircleIcon className="w-4 h-4 text-primary" />
                    {checked.includes(index) ? t('deselectRow') : t('selectRow')}
                    <ContextMenuShortcut>Space</ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuItem onSelect={() => handleEdit(index)} disabled={checked.length > 0} data-testid="context-menu-edit-row">
                    <PencilSquareIcon className="w-4 h-4" />
                    {t('editRow')}
                    <ContextMenuShortcut>Enter</ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuSub>
                    <ContextMenuSubTrigger>
                        <ArrowDownTrayIcon className="w-4 h-4 mr-2" />
                        {t('export')}
                    </ContextMenuSubTrigger>
                    <ContextMenuSubContent
                        collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
                    >
                        <ContextMenuItem
                            onSelect={() => setShowExportConfirm(true)}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            {t('exportAllAsCsv')}
                            <ContextMenuShortcut>{formatShortcut(["Mod", "Shift", "E"])}</ContextMenuShortcut>
                        </ContextMenuItem>
                        <ContextMenuItem
                            onSelect={() => setShowExportConfirm(true)}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            {t('exportAllAsExcel')}
                        </ContextMenuItem>
                        <ContextMenuSeparator />
                        <ContextMenuItem
                            onSelect={() => setShowExportConfirm(true)}
                            disabled={checked.length === 0}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            {t('exportSelectedAsCsv')}
                        </ContextMenuItem>
                        <ContextMenuItem
                            onSelect={() => setShowExportConfirm(true)}
                            disabled={checked.length === 0}
                        >
                            <DocumentIcon className="w-4 h-4" />
                            {t('exportSelectedAsExcel')}
                        </ContextMenuItem>
                    </ContextMenuSubContent>
                </ContextMenuSub>
                <ContextMenuItem
                    onSelect={() => setShowMockDataSheet(true)}
                >
                    <DocumentDuplicateIcon className="w-4 h-4" />
                    {t('mockData')}
                    <ContextMenuShortcut>{formatShortcut(["Mod", "M"])}</ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuItem
                    variant="destructive"
                    disabled={deleting}
                    onSelect={async () => {
                        await handleDeleteRow(index);
                    }}
                    data-testid="context-menu-delete-row"
                >
                    <TrashIcon className="w-4 h-4 text-destructive" />
                    {t('deleteRow')}
                    <ContextMenuShortcut>Delete</ContextMenuShortcut>
                </ContextMenuItem>
            </ContextMenuContent>
        </ContextMenu>
    }, [checked, handleCellClick, handleEdit, handleSelectRow, handleDeleteRow, paginatedRows, disableEdit, onRefresh, t, contextMenuCellIdx, columns, columnIsForeignKey, columnIsPrimary, onEntitySearch, deleting, focusedRowIndex]);

    return (
        <div ref={tableRef} className="h-full flex">
            {/* Screen reader live region for announcements */}
            <div
                role="status"
                aria-live="polite"
                aria-atomic="true"
                className="sr-only"
            >
                {liveAnnouncement}
            </div>
            <div className="flex flex-col h-full space-y-4 w-0" style={{
                width: `${containerWidth}px`,
            }} data-testid="table-container">
                <TableComponent
                    role="grid"
                    aria-label={storageUnit ? `${storageUnit} data table` : 'Data table'}
                    aria-rowcount={paginatedRows.length}
                    aria-multiselectable={true}
                >
                    <TableHeader>
                        <ContextMenu>
                            <ContextMenuTrigger asChild>
                                <TableHeadRow role="row" aria-rowindex={0} className="group relative cursor-context-menu hover:bg-muted/50 transition-colors" title={t('rightClickForOptions')}>
                                    <TableHead className={cn("min-w-[40px] w-[40px] relative", {
                                        "hidden": disableEdit,
                                    })}>
                                        <Checkbox
                                            checked={checked.length === paginatedRows.length && paginatedRows.length > 0}
                                            onCheckedChange={() => {
                                                setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index));
                                            }}
                                            aria-label={checked.length === paginatedRows.length ? t('deselectAll') : t('selectAll')}
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
                                            }} data-testid="icon-button" aria-label={t('moreActions')}>
                                            <EllipsisVerticalIcon className="w-4 h-4" aria-hidden="true" />
                                        </Button>
                                    </TableHead>
                                    {columns.map((col, idx) => (
                                        <TableHead
                                            key={col + idx}
                                            icon={columnIsPrimary?.[idx] ? <KeyIcon className="w-4 h-4" aria-label="Primary key" /> : columnIsForeignKey?.[idx] ? <ShareIcon className="w-4 h-4" aria-label="Foreign key" /> : columnIcons?.[idx]}
                                            className={cn({
                                                "cursor-pointer select-none": onColumnSort,
                                                "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-inset": onColumnSort,
                                            })}
                                            tabIndex={onColumnSort ? 0 : undefined}
                                            onClick={() => onColumnSort?.(col)}
                                            onKeyDown={(e) => {
                                                if (onColumnSort && (e.key === 'Enter' || e.key === ' ')) {
                                                    e.preventDefault();
                                                    onColumnSort(col);
                                                }
                                            }}
                                            onFocus={() => { focusedColumnRef.current = col; }}
                                            data-testid={`column-header-${col}`}
                                        >
                                            <Tip>
                                                <p className={cn("flex items-center gap-xs", {
                                                    "font-bold": columnIsPrimary?.[idx],
                                                    "italic": columnIsForeignKey?.[idx] && !columnIsPrimary?.[idx],
                                                })}>
                                                    {col}
                                                    {onColumnSort && sortedColumns?.has(col) && (
                                                        sortedColumns.get(col) === 'asc'
                                                            ? <ChevronUpIcon className="w-4 h-4" />
                                                            : <ChevronDownIcon className="w-4 h-4" />
                                                    )}
                                                </p>
                                                <p className="text-xs">{columnTypes?.[idx]?.toLowerCase()}</p>
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
                                    {t('mockData')}
                                    <ContextMenuShortcut>{formatShortcut(["Mod", "M"])}</ContextMenuShortcut>
                                </ContextMenuItem>
                                <ContextMenuSeparator />
                                <ContextMenuSub>
                                    <ContextMenuSubTrigger>
                                        <ArrowDownCircleIcon className="w-4 h-4 mr-2" />
                                        {t('exportData')}
                                    </ContextMenuSubTrigger>
                                    <ContextMenuSubContent
                                        collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
                                    >
                                        <ContextMenuItem
                                            onSelect={() => setShowExportConfirm(true)}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            {t('exportAllAsCsv')}
                                            <ContextMenuShortcut>{formatShortcut(["Mod", "Shift", "E"])}</ContextMenuShortcut>
                                        </ContextMenuItem>
                                        <ContextMenuItem
                                            onSelect={() => setShowExportConfirm(true)}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            {t('exportAllAsExcel')}
                                        </ContextMenuItem>
                                        <ContextMenuSeparator />
                                        <ContextMenuItem
                                            onSelect={() => setShowExportConfirm(true)}
                                            disabled={checked.length === 0}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            {t('exportSelectedAsCsv')}
                                        </ContextMenuItem>
                                        <ContextMenuItem
                                            onSelect={() => setShowExportConfirm(true)}
                                            disabled={checked.length === 0}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            {t('exportSelectedAsExcel')}
                                        </ContextMenuItem>
                                    </ContextMenuSubContent>
                                </ContextMenuSub>
                                <ContextMenuSeparator />
                                <ContextMenuItem onSelect={() => onRefresh?.()}>
                                    <CircleStackIcon className="w-4 h-4" />
                                    {t('refreshData')}
                                    <ContextMenuShortcut>{formatShortcut(["Mod", "R"])}</ContextMenuShortcut>
                                </ContextMenuItem>
                                <ContextMenuItem
                                    onSelect={() => {
                                        setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index));
                                    }}
                                >
                                    <CheckCircleIcon className="w-4 h-4" />
                                    {checked.length === paginatedRows.length ? t('deselectAll') : t('selectAll')}
                                    <ContextMenuShortcut>{formatShortcut(["Mod", "A"])}</ContextMenuShortcut>
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
                            <div className="flex items-center justify-center cursor-pointer border rounded-lg h-[200px]">
                                <EmptyState className="table-empty-state" title={t('noDataAvailable')} description={t('noDataAvailable')} icon={<DocumentTextIcon className="w-4 h-4" />} />
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
                                {t('mockData')}
                                <ContextMenuShortcut>{formatShortcut(["Mod", "M"])}</ContextMenuShortcut>
                            </ContextMenuItem>
                            <ContextMenuSub>
                                <ContextMenuSubTrigger>
                                    <ArrowDownCircleIcon className="w-4 h-4 mr-2" />
                                    {t('export')}
                                </ContextMenuSubTrigger>
                                <ContextMenuSubContent
                                    collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
                                >
                                    <ContextMenuItem
                                        onSelect={() => setShowExportConfirm(true)}
                                    >
                                        <DocumentIcon className="w-4 h-4" />
                                        {t('exportAllAsCsv')}
                                        <ContextMenuShortcut>{formatShortcut(["Mod", "Shift", "E"])}</ContextMenuShortcut>
                                    </ContextMenuItem>
                                    <ContextMenuItem
                                        onSelect={() => setShowExportConfirm(true)}
                                    >
                                        <DocumentIcon className="w-4 h-4" />
                                        {t('exportAllAsExcel')}
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
                    <Pagination
                        className={cn("flex justify-end", {
                            "hidden": !showPagination,
                        })}
                        aria-label={t('tablePagination')}
                    >
                        <PaginationContent>
                            <PaginationItem>
                                <PaginationPrevious
                                    href="#"
                                    onClick={e => {
                                        e.preventDefault();
                                        if (currentPage > 1) handlePageChange(currentPage - 1);
                                    }}
                                    aria-disabled={currentPage === 1}
                                    aria-label={t('previousPage')}
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
                                    aria-label={t('nextPage')}
                                    size="sm"
                                    className={cn({
                                        "opacity-50 pointer-events-none": currentPage === totalPages,
                                    })}
                                />
                            </PaginationItem>
                        </PaginationContent>
                    </Pagination>
                </div>
                <div className="flex justify-end items-center mb-2 gap-4">
                    <div className="text-sm hidden" data-testid="total-count-bottom"><span className="font-semibold">{t('totalCount')}</span> {totalCount}</div>
                    <Button
                        variant="secondary"
                        onClick={() => setShowExportConfirm(true)}
                        className="flex gap-sm"
                    >
                        <ArrowDownCircleIcon className="w-4 h-4" />
                        {hasSelectedRows ? t('exportSelected', { count: checked.length }) : t('exportAll')}
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
                        <SheetTitle>{t('editRowTitle')}</SheetTitle>
                        <div className="flex-1 overflow-y-auto mt-4">
                            <div className="flex flex-col gap-lg pr-2">
                                {editRow &&
                                    columns.map((col, idx) => (
                                        <div key={col} className="flex flex-col gap-2">
                                            <div className="flex flex-col gap-0.5">
                                                <Label>{col}</Label>
                                                {columnTypes?.[idx] && (
                                                    <span className="text-xs text-muted-foreground">
                                                        {t('typeHint', { type: columnTypes[idx] })}
                                                    </span>
                                                )}
                                            </div>
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
                                {t('cancel')}
                            </Button>
                            <Button className="flex-1" onClick={handleUpdate} disabled={!editRow} data-testid="update-button">
                                {t('update')}
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
                        <SheetTitle className="flex items-center gap-2"><CalculatorIcon className="w-4 h-4" /> {t('mockDataTitle')}</SheetTitle>
                        {!showMockDataConfirmation ? (
                            <div className="space-y-4">
                                <Label>{t('numberOfRows', { max: maxRowCount })}</Label>
                                <Input
                                    value={mockDataRowCount}
                                    onChange={e => handleMockDataRowCountChange(e.target.value)}
                                    type="text"
                                    inputMode="numeric"
                                    pattern="[0-9]*"
                                    max={maxRowCount.toString()}
                                    placeholder={t('enterNumberOfRows', { max: maxRowCount })}
                                />
                                <Label>{t('method')}</Label>
                                <Select value={mockDataMethod} onValueChange={setMockDataMethod}>
                                    <SelectTrigger className="w-full">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="Normal">{t('methodNormal')}</SelectItem>
                                    </SelectContent>
                                </Select>
                                <Label>{t('dataHandling')}</Label>
                                <Select value={mockDataOverwriteExisting} onValueChange={setMockDataOverwriteExisting}>
                                    <SelectTrigger className="w-full">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="append">{t('appendToExisting')}</SelectItem>
                                        <SelectItem value="overwrite">{t('overwriteExisting')}</SelectItem>
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
                                    {t('overwriteConfirmation', { storageUnit })}
                                </p>
                            </div>
                        )}
                    </div>
                    <SheetFooter className="flex gap-sm px-0">
                        <Alert variant="info" className="mb-4">
                            <AlertTitle>{t('mockDataNote')}</AlertTitle>
                            <AlertDescription>
                                {t('mockDataWarning')}
                            </AlertDescription>
                        </Alert>
                        <Button
                            className="flex-1"
                            variant="secondary"
                            onClick={() => setShowMockDataSheet(false)}
                            data-testid="cancel-mock-data"
                        >
                            {t('cancel')}
                        </Button>
                        {!showMockDataConfirmation ? (
                            <Button className="flex-1" onClick={handleMockDataGenerate} disabled={generatingMockData}>
                                {t('generate')}
                            </Button>
                        ) : (
                            <Button className="flex-1" onClick={handleMockDataGenerate} disabled={generatingMockData} variant="destructive">
                                {t('yesOverwrite')}
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
                    databaseType={databaseType}
                />
            </Suspense>
        </div>
    );
};
