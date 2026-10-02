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

import {useLazyQuery, useMutation, useQuery} from "@apollo/client/react";
import {
    Alert,
    AlertDescription,
    AlertDialog,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
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
    DataPagination,
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
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableHeadRow,
    TableRow,
    TextArea,
    toast
} from "@clidey/ux";
import { Spinner } from '@/components/loading';
import {
    AnalyzeMockDataDependenciesDocument,
    AddRowDocument,
    DeleteRowDocument,
    GenerateMockDataDocument,
    MockDataMaxRowCountDocument,
    SourceAction,
    type SourceObjectRefInput,
} from '@graphql';
import type {FC} from "react";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState} from "react";
import { createPortal } from "react-dom";
import {Export} from "./export";
import {ImportData} from "./import-data";
import {useTranslation} from '@/hooks/use-translation';
import {copyToClipboard} from '@/services/clipboard';
import {useSourceContract} from "@/hooks/useSourceContract";
import {sourceObjectSupportsAction} from "@/config/source-types";
import {ph} from '@/utils/privacy';
import {
    ArrowDownCircleIcon,
    ArrowDownTrayIcon,
    ArrowUpCircleIcon,
    CalculatorIcon,
    CheckCircleIcon,
    ChevronDownIcon,
    ChevronUpIcon,
    CircleStackIcon,
    CursorArrowRaysIcon,
    DocumentDuplicateIcon,
    DocumentIcon,
    DocumentTextIcon,
    EllipsisVerticalIcon,
    MagnifyingGlassIcon,
    PencilSquareIcon,
    TrashIcon,
    XMarkIcon
} from "./heroicons";
import {Tip} from "./tip";
import {WhoDBChatIcon} from "./whodb-chat-icon";
import {formatShortcut} from "@/utils/platform";
import {matchesShortcut, SHORTCUTS} from "@/utils/shortcuts";
import {formatNumber} from "@/utils/functions";


// Dynamically load Export component
// const EEExport = loadEEComponent(
//     null,
// );

const EEExport = null;

// Dynamic Export component
const DynamicExport: FC<{
    open: boolean;
    onOpenChange: (open: boolean) => void;
    storageUnit: string;
    objectRef?: SourceObjectRefInput;
    hasSelectedRows: boolean;
    selectedRowsData?: Record<string, any>[];
    checkedRowsCount: number;
    databaseType?: string;
    rawQuery?: string;
    preselectedFormat?: 'csv' | 'excel' | 'ndjson';
    forceExportAll?: boolean;
}> = (props) => {
    // Use registered Export if available, otherwise fall back to default
    const ExportComponent = EEExport ?? Export;
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


/** Returns the WhoDB icon for each column's data type. */
export function getColumnIcons(columns: string[], columnTypes?: string[]) {
    return columns.map((col, idx) => {
        const rawType = columnTypes?.[idx] ?? "";
        const type = stripTypeSuffix(rawType).toUpperCase();
        const key = `${col}-${idx}`;

        if (intTypes.has(type) || uintTypes.has(type)) return <WhoDBChatIcon key={key} name="hash" />;
        if (floatTypes.has(type)) return <WhoDBChatIcon key={key} name="banknotes" />;
        if (boolTypes.has(type)) return <WhoDBChatIcon key={key} name="check-circle" />;
        if (dateTypes.has(type)) return <WhoDBChatIcon key={key} name="calendar" />;
        if (dateTimeTypes.has(type)) return <WhoDBChatIcon key={key} name="clock" />;
        if (uuidTypes.has(type)) return <WhoDBChatIcon key={key} name="secret" />;
        if (binaryTypes.has(type)) return <WhoDBChatIcon key={key} name="copy" />;
        if (jsonTypes.has(type) || xmlTypes.has(type)) return <WhoDBChatIcon key={key} name="code" />;
        if (networkTypes.has(type)) return <WhoDBChatIcon key={key} name="globe" />;
        if (geometryTypes.has(type)) return <WhoDBChatIcon key={key} name="grid" />;
        if (type.startsWith("ARRAY")) return <WhoDBChatIcon key={key} name="list" />;
        if (stringTypes.has(type)) return <WhoDBChatIcon key={key} name="file-text" />;
        return <WhoDBChatIcon key={key} name="source" />;
    });
}

/**
 * Maps column data types to HTML5 input attributes for native validation.
 * Leverages browser-native validation and appropriate input types.
 *
 * @param rawType - The column type string (e.g., "INTEGER", "VARCHAR(255)", "TIMESTAMP")
 * @returns Object with HTML input attributes (type, step, min, inputMode)
 */
export function getInputPropsForColumnType(rawType: string): {
    type?: React.HTMLInputTypeAttribute;
    step?: string;
    min?: string;
    inputMode?: 'text' | 'numeric' | 'decimal',
    onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
} {
    const type = stripTypeSuffix(rawType).toUpperCase();

    // the html5 spec for numbers allows "e" to be used to mean exponent, so 2e2 => 2*10^2 => 200.
    // that requires extra backend handling and databases do not usually show nums like that.
    // so we avoid "e" as well as "+" because if a number doesn't have "-", it's already positive.
    const numOnKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {if (e.key === "e" || e.key === "E" || e.key === "+") e.preventDefault();}

    // Integer types - use number input with step=1
    if (intTypes.has(type)) {
        return { type: 'number', step: '1', inputMode: 'numeric',  onKeyDown: numOnKeyDown};
    }

    // Unsigned integer types - use number input with min=0 and step=1
    if (uintTypes.has(type)) {
        return { type: 'number', step: '1', min: '0', inputMode: 'numeric', onKeyDown: numOnKeyDown };
    }

    // Float/decimal types - use number input with step=any
    if (floatTypes.has(type)) {
        return { type: 'number', step: 'any', inputMode: 'decimal', onKeyDown: numOnKeyDown };
    }

    // Default to text input with text keyboard
    return { type: 'text', inputMode: 'text' };
}

interface FormatOptions {
    dates?: boolean;
    booleans?: boolean;
}

function formatCellDisplay(value: string, columnType: string | undefined, options: FormatOptions): string {
    if (!columnType || !value) return value;
    const type = stripTypeSuffix(columnType).toUpperCase();

    if (options.dates && (dateTypes.has(type) || dateTimeTypes.has(type))) {
        const parsed = Date.parse(value);
        if (!Number.isNaN(parsed)) {
            const date = new Date(parsed);
            return dateTypes.has(type) ? date.toLocaleDateString() : date.toLocaleString();
        }
    }

    if (options.booleans && boolTypes.has(type)) {
        const lower = value.toLowerCase();
        if (lower === 't' || lower === '1' || lower === 'true') return 'true';
        if (lower === 'f' || lower === '0' || lower === 'false') return 'false';
    }

    return value;
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
    cellEditing?: boolean;
    disableEdit?: boolean;
    allowRowUpdate?: boolean;
    allowRowDelete?: boolean;
    limitContextMenu?: boolean;
    schema?: string;
    storageUnit?: string;
    sourceLabel?: string;
    objectRef?: SourceObjectRefInput;
    onRefresh?: () => void;
    children?: React.ReactNode;
    actionsTarget?: HTMLElement | null;
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
    // Mock data generation control - set to false for views/materialized views
    isMockDataGenerationAllowed?: boolean;
    // Import control - set to true to enable import functionality
    allowImport?: boolean;
    rawQuery?: string;
    // Enforce minimum height - when true, always uses passed height; when false, shrinks to content if smaller
    enforceMinHeight?: boolean;
    // Enable keyboard shortcuts - should only be true on the explore-storage-unit page
    enableKeyboardShortcuts?: boolean;
    formatDatesLocale?: boolean;
    formatBooleansReadable?: boolean;
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
    cellEditing = false,
    disableEdit = false,
    allowRowUpdate = true,
    allowRowDelete = true,
    limitContextMenu = false,
    schema: _schema,
    storageUnit,
    sourceLabel,
    objectRef,
    onRefresh,
    children,
    actionsTarget,
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
    isValidForeignKey: _isValidForeignKey,
    onEntitySearch,
    databaseType,
    // Mock data generation control
    isMockDataGenerationAllowed = true,
    // Import control
    allowImport = false,
    rawQuery,
    enforceMinHeight = false,
    enableKeyboardShortcuts = false,
    formatDatesLocale = false,
    formatBooleansReadable = false,
}) => {
    const { t, language } = useTranslation('components/table');
    const [editIndex, setEditIndex] = useState<number | null>(null);
    const [editRow, setEditRow] = useState<string[] | null>(null);
    const [editRowInitialLengths, setEditRowInitialLengths] = useState<number[]>([]);
    const [editingCell, setEditingCell] = useState<{ row: number; column: number } | null>(null);
    const [pendingCells, setPendingCells] = useState<Record<string, string>>({});
    const [savingCells, setSavingCells] = useState(false);
    const [pendingDeleteIndexes, setPendingDeleteIndexes] = useState<number[] | null>(null);
    const [deletingRows, setDeletingRows] = useState(false);
    const [checked, setChecked] = useState<number[]>([]);
    const [showExportConfirm, setShowExportConfirm] = useState(false);
    const [showImport, setShowImport] = useState(false);
    const [preselectedFormat, setPreselectedFormat] = useState<'csv' | 'excel' | 'ndjson' | undefined>(undefined);
    const [forceExportAll, setForceExportAll] = useState(false);
    const tableRef = useRef<HTMLDivElement>(null);
    const [contextMenuCellIdx, setContextMenuCellIdx] = useState<number | null>(null);
    const [contextMenuRowIdx, setContextMenuRowIdx] = useState<number | null>(null);

    // Keyboard navigation state
    const [focusedRowIndex, setFocusedRowIndex] = useState<number | null>(null);
    // Track focused column header for focus restoration after sort/refresh
    const focusedColumnRef = useRef<string | null>(null);
    
    // Mock data state
    const [showMockDataSheet, setShowMockDataSheet] = useState(false);
    const [mockDataRowCount, setMockDataRowCount] = useState("100");
    const [mockDataMethod, setMockDataMethod] = useState("Normal");
    const [mockDataOverwriteExisting, setMockDataOverwriteExisting] = useState("append");
    const [mockDataFkDensityRatio, setMockDataFkDensityRatio] = useState("20");
    const [showMockDataConfirmation, setShowMockDataConfirmation] = useState(false);
    const { item, supportsMockDataRelations, sequentialPaginationOnly } = useSourceContract(databaseType);
    const isMockDataSupported =
        sourceObjectSupportsAction(item, objectRef?.Kind, SourceAction.GenerateMockData) && isMockDataGenerationAllowed;
    const isImportSupported = sourceObjectSupportsAction(item, objectRef?.Kind, SourceAction.ImportData);
    const isExportSupported = rawQuery != null || sourceObjectSupportsAction(item, objectRef?.Kind, SourceAction.ViewRows);
    const isRowUpdateSupported = sourceObjectSupportsAction(item, objectRef?.Kind, SourceAction.UpdateData);
    const isRowDeleteSupported = sourceObjectSupportsAction(item, objectRef?.Kind, SourceAction.DeleteData);
    const isRowInsertSupported = sourceObjectSupportsAction(item, objectRef?.Kind, SourceAction.InsertData);
    const { data: maxRowData } = useQuery(MockDataMaxRowCountDocument);
    const maxRowCount = maxRowData?.MockDataMaxRowCount ?? 200;

    // Use server-side pagination
    const currentPage = serverCurrentPage ?? 1;
    const totalRows = totalCount ?? 0;
    const totalPages = Math.ceil(totalRows / pageSize);

    const [generateMockData, { loading: generatingMockData }] = useMutation(GenerateMockDataDocument);
    const [analyzeDependencies, { data: depAnalysis, loading: analyzingDeps }] = useLazyQuery(AnalyzeMockDataDependenciesDocument);
    const [deleteRow, ] = useMutation(DeleteRowDocument);
    const [restoreRow] = useMutation(AddRowDocument);
    const [containerWidth, setContainerWidth] = useState<number>(0);
    const lastSearchState = useRef<{ search: string; matchIdx: number }>({ search: '', matchIdx: 0 });
    const canEditRows = !disableEdit && allowRowUpdate && isRowUpdateSupported && onRowUpdate != null;
    const canDeleteRows = !disableEdit && allowRowDelete && isRowDeleteSupported && objectRef != null;
    const pendingCellCount = Object.keys(pendingCells).length;
    const pendingRowIndexes = [...new Set(Object.keys(pendingCells).map(key => Number(key.split(':')[0])))];

    useEffect(() => {
        setEditingCell(null);
        setPendingCells({});
    }, [currentPage, storageUnit]);

    const updatePendingCell = useCallback((rowIndex: number, columnIndex: number, value: string) => {
        const key = `${rowIndex}:${columnIndex}`;
        setPendingCells(previous => {
            const next = {...previous};
            if (value === rows[rowIndex]?.[columnIndex]) delete next[key];
            else next[key] = value;
            return next;
        });
    }, [rows]);

    const savePendingCells = useCallback(async () => {
        if (!onRowUpdate || savingCells || pendingCellCount === 0) return;
        setEditingCell(null);
        setSavingCells(true);
        try {
            const rowIndexes = [...new Set(Object.keys(pendingCells).map(key => Number(key.split(':')[0])))];
            for (const rowIndex of rowIndexes) {
                const original: Record<string, string | number> = {};
                const updated: Record<string, string | number> = {};
                columns.forEach((column, columnIndex) => {
                    const value = rows[rowIndex][columnIndex];
                    original[column] = value;
                    updated[column] = pendingCells[`${rowIndex}:${columnIndex}`] ?? value;
                });
                await onRowUpdate(updated, original);
            }
            setPendingCells({});
            toast.success(t('rowUpdated'));
            onRefresh?.();
        } catch {
            toast.error(t('errorUpdatingRow'));
        } finally {
            setSavingCells(false);
        }
    }, [columns, onRefresh, onRowUpdate, pendingCellCount, pendingCells, rows, savingCells, t]);

    const handleEdit = (index: number) => {
        if (!canEditRows) {
            return;
        }
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
        if (!canEditRows) {
            return;
        }
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
    }, [canEditRows, editIndex, editRow, columns, onRowUpdate, rows, onRefresh, t]);

    // --- Export logic ---
    const hasSelectedRows = checked.length > 0;
    const primaryColumnIndex = columnIsPrimary?.findIndex(Boolean) ?? -1;
    const selectedIds = checked.map(index => primaryColumnIndex >= 0 ? rows[index]?.[primaryColumnIndex] : String(index + 1)).filter(Boolean);
    const selectedRowsData = useMemo(() => {
        if (hasSelectedRows) {
            const validChecked = checked.filter(idx => idx < rows.length);
            return validChecked.map(idx => {
                const row = rows[idx];
                const rowObj: Record<string, any> = {};
                columns.forEach((col, colIdx) => {
                    rowObj[col] = row[colIdx];
                });
                return rowObj;
            });
        }
        if (rawQuery) {
            return rows.map(row => {
                const rowObj: Record<string, any> = {};
                columns.forEach((col, colIdx) => {
                    rowObj[col] = row[colIdx];
                });
                return rowObj;
            });
        }
        return undefined;
    }, [hasSelectedRows, checked, rows, columns, rawQuery]);

    const openExport = useCallback((format?: 'csv' | 'excel' | 'ndjson', exportAll?: boolean) => {
        if (!isExportSupported) {
            return;
        }
        setPreselectedFormat(format);
        setForceExportAll(exportAll ?? false);
        setShowExportConfirm(true);
    }, [isExportSupported]);

    // Delete logic, adapted from explore-storage-unit.tsx
    const doDeleteRows = useCallback(async (indexesToDelete: number[]) => {
        if (!canDeleteRows) {
            return;
        }
        if (!objectRef) {
            toast.error(t('storageUnitRequired'));
            return;
        }
        const deletedRows: string[][] = [];
        for (const index of indexesToDelete) {
            const row = rows[index];
            if (!row) continue;
            const values = columns.map((col, i) => ({
                Key: col,
                Value: row[i],
            }));
            try {
                const result = await deleteRow({
                    variables: {
                        ref: objectRef,
                        values,
                    },
                });
                if (!result.data?.DeleteRow.Status) throw new Error(t('unableToDeleteRowStatus'));
                deletedRows.push([...row]);
            } catch (e: any) {
                toast.error(t('unableToDeleteRow', { message: e?.message ?? e }));
                break;
            }
        }
        if (deletedRows.length === indexesToDelete.length) {
            const toastOptions = isRowInsertSupported ? {
                description: t('rowsDeletedFrom', {table: storageUnit ?? '', source: sourceLabel ?? ''}),
                duration: 7000,
                className: 'ce-delete-toast',
                icon: <span className="ce-delete-success-icon"><span className="ce-whodb-icon ce-whodb-icon-check" /></span>,
                action: {
                    label: t('undo'),
                    onClick: () => {
                        void (async () => {
                            try {
                                for (const deleted of deletedRows) {
                                    const result = await restoreRow({variables: {
                                        ref: objectRef,
                                        values: columns.map((column, i) => ({Key: column, Value: deleted[i]})),
                                    }});
                                    if (!result.data?.AddRow.Status) throw new Error(t('undoFailed'));
                                }
                                toast.success(t('deleteUndone'));
                                onRefresh?.();
                            } catch {
                                toast.error(t('undoFailed'));
                            }
                        })();
                    },
                },
            } : {description: t('rowsDeletedFrom', {table: storageUnit ?? '', source: sourceLabel ?? ''}), duration: 7000, className: 'ce-delete-toast', icon: <span className="ce-delete-success-icon"><span className="ce-whodb-icon ce-whodb-icon-check" /></span>};
            toast.success(t('rowsDeleted', {count: deletedRows.length}), toastOptions);
        }
        onRefresh?.();
    }, [canDeleteRows, columns, deleteRow, isRowInsertSupported, objectRef, onRefresh, restoreRow, rows, sourceLabel, storageUnit, t]);

    const handleDeleteRow = useCallback((rowIndex: number) => {
        if (!canDeleteRows) {
            return;
        }
        if (!rows || !columns) return;
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
        setPendingDeleteIndexes(indexesToDelete);
    }, [canDeleteRows, rows, columns, checked]);

    const handleConfirmDelete = useCallback(async () => {
        if (pendingDeleteIndexes && !deletingRows) {
            const indexes = pendingDeleteIndexes;
            setDeletingRows(true);
            try {
                await doDeleteRows(indexes);
            } finally {
                setDeletingRows(false);
                setPendingDeleteIndexes(null);
            }
        }
    }, [pendingDeleteIndexes, deletingRows, doDeleteRows]);

    const handleCancelDelete = useCallback(() => {
        setPendingDeleteIndexes(null);
    }, []);

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
            return () => { clearTimeout(timeoutId); };
        }
    }, [rows]);

    const handlePageChange = useCallback((newPage: number) => {
        if (cellEditing && pendingCellCount > 0) {
            toast.info(t('saveOrDiscardBeforePaging'));
            return;
        }
        // Sources paginating via keyset cursors (e.g. PostHog, which rejects
        // OFFSET for personal-API-key queries) can only resolve the page
        // immediately before or after the current one, so arbitrary jumps
        // are silently ignored rather than sent as a doomed request.
        if (sequentialPaginationOnly && Math.abs(newPage - currentPage) !== 1) {
            return;
        }
        onPageChange?.(newPage);
    }, [cellEditing, currentPage, onPageChange, pendingCellCount, sequentialPaginationOnly, t]);

    const handleSelectRow = useCallback((rowIndex: number) => {
        const isCurrentlySelected = checked.includes(rowIndex);
        const newChecked = isCurrentlySelected ? checked.filter(i => i !== rowIndex) : [...checked, rowIndex];
        setChecked(newChecked);
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
        const timeout = window.setTimeout(() => {
            const cell = paginatedRows[rowIndex][cellIndex];
            if (cell !== undefined && cell !== null) {
                void copyToClipboard(String(cell)).then(success => {
                    if (success) toast.success(t('copiedToClipboard'));
                });
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
            void copyToClipboard(rowString).then(success => {
                if (success) toast.success(t('rowCopiedToClipboard'));
            });
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
        if (!storageUnit || !objectRef) {
            toast.error(t('storageUnitRequired'));
            return;
        }

        if (mockDataOverwriteExisting === "overwrite" && !showMockDataConfirmation) {
            setShowMockDataConfirmation(true);
            return;
        }

        const count = parseInt(mockDataRowCount);

        // Validate row count
        if (isNaN(count) || count < 1) {
            toast.error(t('rowCountMustBePositive'));
            return;
        }

        if (count > maxRowCount) {
            toast.error(t('rowCountExceedsMax', { max: maxRowCount }));
            return;
        }
        
        try {
            const result = await generateMockData({
                variables: {
                    input: {
                        Ref: objectRef,
                        RowCount: count,
                        Method: mockDataMethod,
                        OverwriteExisting: mockDataOverwriteExisting === "overwrite",
                        FkDensityRatio: parseInt(mockDataFkDensityRatio) || 20,
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
    }, [generateMockData, maxRowCount, mockDataFkDensityRatio, mockDataMethod, mockDataOverwriteExisting, mockDataRowCount, objectRef, onRefresh, showMockDataConfirmation, storageUnit, t]);

    const columnIcons = useMemo(() => getColumnIcons(columns, columnTypes), [columns, columnTypes]);

    // Cleanup click timeouts on unmount
    useEffect(() => {
        return () => {
            // Clear all pending timeouts
            clickTimeouts.current.forEach(timeout => { clearTimeout(timeout); });
            clickTimeouts.current.clear();
        };
    }, []);

    useEffect(() => {
        // Note: schema can be empty for SQLite which doesn't use schemas
        if (showMockDataSheet && objectRef) {
            const rowCount = parseInt(mockDataRowCount) || 100;
            if (rowCount > 0 && rowCount <= maxRowCount) {
                void analyzeDependencies({
                    variables: {
                        ref: objectRef,
                        rowCount,
                        fkDensityRatio: null,
                    },
                });
            }
        }
    }, [analyzeDependencies, maxRowCount, mockDataRowCount, objectRef, showMockDataSheet]);

    const adjustedDepAnalysis = useMemo(() => {
        const analysis = depAnalysis?.AnalyzeMockDataDependencies;
        if (!analysis || analysis.Error || !analysis.Tables || analysis.Tables.length <= 1) {
            return analysis;
        }

        const ratio = parseInt(mockDataFkDensityRatio) || 20;
        const requestedRows = parseInt(mockDataRowCount) || 100;
        const tables = [...analysis.Tables];

        // Tables are in generation order (parents first, target last)
        // Recalculate: target gets requested count, parents get child/ratio
        const recalculated = tables.map((t) => ({ ...t }));

        // Start from the end (target table) and work backwards
        let childRowCount = requestedRows;
        for (let i = recalculated.length - 1; i >= 0; i--) {
            if (i === recalculated.length - 1) {
                // Target table gets the requested row count
                recalculated[i] = { ...recalculated[i], RowsToGenerate: requestedRows };
            } else {
                // Parent tables get childCount/ratio (min 1)
                const parentRows = Math.max(1, Math.floor(childRowCount / ratio));
                recalculated[i] = { ...recalculated[i], RowsToGenerate: parentRows };
                childRowCount = parentRows;
            }
        }

        const totalRows = recalculated.reduce((sum, t) => sum + t.RowsToGenerate, 0);

        return {
            ...analysis,
            Tables: recalculated,
            TotalRows: totalRows,
        };
    }, [depAnalysis, mockDataFkDensityRatio, mockDataRowCount]);

    // Listen for menu export trigger
    useEffect(() => {
        const handleExportTrigger = () => {
            openExport();
        };

        window.addEventListener('menu:trigger-export', handleExportTrigger);
        return () => {
            window.removeEventListener('menu:trigger-export', handleExportTrigger);
        };
    }, []);

    // Listen for menu import trigger
    useEffect(() => {
        const handleImportTrigger = () => {
            if (isImportSupported && allowImport) {
                setShowImport(true);
            }
        };

        window.addEventListener('menu:trigger-import', handleImportTrigger);
        return () => {
            window.removeEventListener('menu:trigger-import', handleImportTrigger);
        };
    }, [isImportSupported, allowImport]);

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

    // Helper to scroll the focused row into view inside the table's own scroll container.
    // Uses smooth scrolling for small jumps, auto (instant) for large ones.
    const scrollRowIntoView = useCallback((rowIndex: number) => {
        const container = tableRef.current?.querySelector<HTMLElement>('[data-slot="table-container"]');
        const row = tableRef.current?.querySelector<HTMLElement>(`[data-row-idx="${rowIndex}"]`);
        if (!container || !row) return;

        const rowTop = row.offsetTop;
        const rowBottom = rowTop + row.offsetHeight;
        const viewTop = container.scrollTop;
        const viewBottom = viewTop + container.clientHeight;

        let target: number | null = null;
        if (rowTop < viewTop) {
            target = rowTop;
        } else if (rowBottom > viewBottom) {
            target = rowBottom - container.clientHeight;
        }

        if (target === null) return;

        const distance = Math.abs(target - viewTop);
        const behavior = distance > container.clientHeight ? 'auto' : 'smooth';
        container.scrollTo({ top: target, behavior });
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
        if (!enableKeyboardShortcuts) return;

        const handleKeyDown = (event: KeyboardEvent) => {
            // Only handle shortcuts when not in input fields
            if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) {
                return;
            }

            // Mod+Shift combos (table-level: available with zero rows)
            if (matchesShortcut(event, SHORTCUTS.exportData)) {
                if (isExportSupported) {
                    event.preventDefault();
                    openExport();
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.mockData)) {
                if (isMockDataSupported) {
                    event.preventDefault();
                    setShowMockDataSheet(true);
                }
                return;
            }

            // Mod combos (no Shift)
            if (matchesShortcut(event, SHORTCUTS.importData)) {
                if (isImportSupported && allowImport) {
                    event.preventDefault();
                    setShowImport(true);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.refresh)) {
                event.preventDefault();
                onRefresh?.();
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.selectAll)) {
                event.preventDefault();
                setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index));
                return;
            }

            if (paginatedRows.length === 0) return;

            // Extend-select variants (Shift+Arrow)
            if (matchesShortcut(event, SHORTCUTS.extendSelectDown)) {
                event.preventDefault();
                if (focusedRowIndex === null) {
                    moveFocus(0, true);
                } else {
                    moveFocus(Math.min(focusedRowIndex + 1, paginatedRows.length - 1), true);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.extendSelectUp)) {
                event.preventDefault();
                if (focusedRowIndex === null) {
                    moveFocus(paginatedRows.length - 1, true);
                } else {
                    moveFocus(Math.max(focusedRowIndex - 1, 0), true);
                }
                return;
            }

            if (matchesShortcut(event, SHORTCUTS.editRowAlt)) {
                if (focusedRowIndex !== null && canEditRows) {
                    event.preventDefault();
                    handleEdit(focusedRowIndex);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.deleteRow) || matchesShortcut(event, SHORTCUTS.deleteRowAlt)) {
                if (focusedRowIndex !== null && canDeleteRows) {
                    event.preventDefault();
                    handleDeleteRow(focusedRowIndex);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.nextPage)) {
                event.preventDefault();
                if (onPageChange && currentPage < totalPages) {
                    onPageChange(currentPage + 1);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.prevPage)) {
                event.preventDefault();
                if (onPageChange && currentPage > 1) {
                    onPageChange(currentPage - 1);
                }
                return;
            }

            // Plain key navigation
            if (matchesShortcut(event, SHORTCUTS.moveDown)) {
                event.preventDefault();
                if (focusedRowIndex === null) {
                    moveFocus(0, false);
                } else {
                    moveFocus(Math.min(focusedRowIndex + 1, paginatedRows.length - 1), false);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.moveUp)) {
                event.preventDefault();
                if (focusedRowIndex === null) {
                    moveFocus(paginatedRows.length - 1, false);
                } else {
                    moveFocus(Math.max(focusedRowIndex - 1, 0), false);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.moveFirst)) {
                event.preventDefault();
                moveFocus(0, false);
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.moveLast)) {
                event.preventDefault();
                moveFocus(paginatedRows.length - 1, false);
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.pageDown)) {
                event.preventDefault();
                if (focusedRowIndex === null) {
                    moveFocus(Math.min(visibleRowCount - 1, paginatedRows.length - 1), false);
                } else {
                    moveFocus(Math.min(focusedRowIndex + visibleRowCount, paginatedRows.length - 1), false);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.pageUp)) {
                event.preventDefault();
                if (focusedRowIndex === null) {
                    moveFocus(0, false);
                } else {
                    moveFocus(Math.max(focusedRowIndex - visibleRowCount, 0), false);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.toggleSelect)) {
                if (focusedRowIndex !== null) {
                    event.preventDefault();
                    handleSelectRow(focusedRowIndex);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.editRow)) {
                if (focusedRowIndex !== null && canEditRows) {
                    event.preventDefault();
                    handleEdit(focusedRowIndex);
                }
                return;
            }
            if (matchesShortcut(event, SHORTCUTS.closeDialogs)) {
                event.preventDefault();
                setFocusedRowIndex(null);
                return;
            }
        };

        window.addEventListener('keydown', handleKeyDown);
        return () => { window.removeEventListener('keydown', handleKeyDown); };
    }, [enableKeyboardShortcuts, onRefresh, checked, paginatedRows, handleDeleteRow, handleEdit, focusedRowIndex, moveFocus, visibleRowCount, handleSelectRow, canEditRows, canDeleteRows, onPageChange, currentPage, totalPages, openExport, isExportSupported]);



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
                el.classList.remove('bg-highlight/35', 'table-search-highlight', 'bg-muted');
            });

            // Remove highlight from the last highlighted cell if it exists
            if (lastHighlightedCell) {
                lastHighlightedCell.classList.remove('bg-muted', 'table-search-highlight', 'bg-highlight/35');
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
                        const searchValue = String(cellValue);
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

                const { rowIdx, colIdx } = matches[matchIdx];
                // Compose a unique selector for the cell
                const selector = `[data-row-idx="${rowIdx}"] [data-col-idx="${colIdx}"]`;
                const cell = document.querySelector(selector) as HTMLElement | null;
                if (cell) {
                    // Remove highlight from any previously highlighted cell
                    document.querySelectorAll('.table-search-highlight').forEach(el => {
                        el.classList.remove('bg-muted', 'table-search-highlight', 'bg-highlight/35');
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
                lastHighlightedCell.classList.remove('bg-muted', 'table-search-highlight', 'bg-highlight/35');
                lastHighlightedCell = null;
            }
        };
    }, [searchRef, rows, columns]);

    // Calculate actual height needed for the table content
    // Add small buffer to account for borders/padding to prevent unnecessary scrollbar
    const actualTableHeight = useMemo(() => {
        if (enforceMinHeight) {
            // Always use the passed height when enforceMinHeight is true
            return height;
        }

        // Original behavior: shrink to content if content is smaller than height
        if (paginatedRows.length === 0) return Math.min(500, height);
        const contentHeight = paginatedRows.length * rowHeight;
        return Math.min(contentHeight + 1, height);
    }, [paginatedRows.length, rowHeight, height, enforceMinHeight]);

    const contextMenu = useCallback((index: number) => {
        const isFocused = focusedRowIndex === index;
        const isSelected = checked.includes(index);

        const tableRow = (
            <TableRow
                key={index}
                data-row-idx={index}
                role="row"
                aria-rowindex={index + 1}
                aria-selected={isSelected}
                data-focused={isFocused || undefined}
                tabIndex={isFocused ? 0 : -1}
                className={cn(
                    "group relative cursor-pointer",
                    // Focus styling - visible ring around focused row
                    isFocused && "bg-primary/5",
                    // Selected styling
                    isSelected && "bg-muted"
                )}
                onClick={() => { setFocusedRowIndex(index); }}
                onFocus={() => { setFocusedRowIndex(index); }}
                onContextMenu={() => { setContextMenuRowIdx(index); }}
            >
                <TableCell
                    role="gridcell"
                    className={cn("ce-table-selection-cell min-w-[40px] w-[40px]", {
                        "hidden": disableEdit,
                    })}
                >
                    <div className="ce-table-selection-control"><Checkbox
                        checked={isSelected}
                        onCheckedChange={() => { setChecked(isSelected ? checked.filter(i => i !== index) : [...checked, index]); }}
                        aria-label={isSelected ? t('deselectRow') : t('selectRow')}
                    /></div>
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
                    const displayValue = (formatDatesLocale || formatBooleansReadable)
                        ? formatCellDisplay(cell, columnTypes?.[cellIdx], { dates: formatDatesLocale, booleans: formatBooleansReadable })
                        : cell;
                    return (
                        <TableCell
                            key={columns[cellIdx]}
                            role="gridcell"
                            className={cn(ph.mask, "cursor-pointer")}
                            title={displayValue !== cell ? cell : t(cellEditing && canEditRows ? 'cellEditingHint' : 'cellInteractionHint')}
                            onClick={(e) => {
                                e.stopPropagation();
                                setFocusedRowIndex(index);
                                handleCellClick(index, cellIdx);
                            }}
                            onDoubleClick={() => {
                                if (cellEditing && canEditRows) {
                                    for (const timeout of clickTimeouts.current.values()) clearTimeout(timeout);
                                    clickTimeouts.current.clear();
                                    setEditingCell({row: index, column: cellIdx});
                                } else handleCellDoubleClick(index);
                            }}
                            onContextMenu={() => { if (!limitContextMenu) { setContextMenuCellIdx(cellIdx); } }}
                            data-col-idx={cellIdx}
                        >
                            {cellEditing && canEditRows && editingCell?.row === index && editingCell.column === cellIdx ? (
                                <Input
                                    autoFocus
                                    className="ce-cell-editor"
                                    value={pendingCells[`${index}:${cellIdx}`] ?? cell}
                                    onChange={event => { updatePendingCell(index, cellIdx, event.target.value); }}
                                    onClick={event => { event.stopPropagation(); }}
                                    onDoubleClick={event => { event.stopPropagation(); }}
                                    onBlur={() => { setEditingCell(null); }}
                                    onKeyDown={event => {
                                        if (event.key === 'Enter') setEditingCell(null);
                                        if (event.key === 'Escape') {
                                            updatePendingCell(index, cellIdx, rows[index][cellIdx]);
                                            setEditingCell(null);
                                        }
                                    }}
                                    data-testid="cell-editor"
                                />
                            ) : (
                                <span className={cn(pendingCells[`${index}:${cellIdx}`] !== undefined && 'ce-cell-pending')}>
                                    {pendingCells[`${index}:${cellIdx}`] ?? displayValue}
                                </span>
                            )}
                        </TableCell>
                    );
                })}
            </TableRow>
        );

        return tableRow;
    }, [
	checked,
	handleCellClick,
	handleEdit,
	handleSelectRow,
	handleDeleteRow,
	paginatedRows,
	disableEdit,
	limitContextMenu,
	onRefresh,
	t,
	contextMenuCellIdx,
	columns,
	columnIsForeignKey,
	columnIsPrimary,
	onEntitySearch,
	focusedRowIndex,
	isMockDataSupported,
	openExport,
	canEditRows,
	canDeleteRows,
	isExportSupported,
    cellEditing,
    editingCell,
    pendingCells,
    updatePendingCell,
    rows,
    handleCellDoubleClick,
    formatDatesLocale,
    formatBooleansReadable,
    columnTypes
]);

    const rowContextMenuContent = (index: number) => (
        <ContextMenuContent
            className="w-52 max-h-[calc(100vh-2rem)] overflow-y-auto"
            collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
        >
            <ContextMenuItem
                onSelect={() => {
                    if (contextMenuCellIdx == null) return;
                    const cell = paginatedRows[index]?.[contextMenuCellIdx];
                    if (cell !== undefined && cell !== null) {
                        void copyToClipboard(String(cell)).then(success => {
                            if (success) toast.success(t('copiedCellToClipboard'));
                        });
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
                        void copyToClipboard(rowString).then(success => {
                            if (success) toast.success(t('rowCopiedToClipboard'));
                        });
                    }
                }}
                className="[&>[data-slot='context-menu-shortcut']]:flex"
            >
                <DocumentTextIcon className="w-4 h-4" />
                {t('copyRow')}
                <ContextMenuShortcut><CursorArrowRaysIcon className="w-4 h-4" /><CursorArrowRaysIcon className="w-4 h-4" /></ContextMenuShortcut>
            </ContextMenuItem>
            {!limitContextMenu && (
                <ContextMenuItem onSelect={() => { handleSelectRow(index); }}>
                    <CheckCircleIcon className="w-4 h-4 text-primary" />
                    {checked.includes(index) ? t('deselectRow') : t('selectRow')}
                    <ContextMenuShortcut>Space</ContextMenuShortcut>
                </ContextMenuItem>
            )}
            {!limitContextMenu && canEditRows && (
                <ContextMenuItem onSelect={() => { handleEdit(index); }} disabled={checked.length > 1} data-testid="context-menu-edit-row">
                    <PencilSquareIcon className="w-4 h-4" />
                    {t('editRow')}
                    <ContextMenuShortcut>Enter</ContextMenuShortcut>
                </ContextMenuItem>
            )}
            {isExportSupported && (
                <ContextMenuSub>
                    <ContextMenuSubTrigger>
                        <ArrowDownTrayIcon className="w-4 h-4 mr-2" />
                        {t('export')}
                    </ContextMenuSubTrigger>
                    <ContextMenuSubContent collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}>
                        <ContextMenuItem onSelect={() => { openExport('csv', true); }}>
                            <DocumentIcon className="w-4 h-4" />
                            {t('exportAllAsCsv')}
                            <ContextMenuShortcut>{formatShortcut(SHORTCUTS.exportData.displayKeys)}</ContextMenuShortcut>
                        </ContextMenuItem>
                        <ContextMenuItem onSelect={() => { openExport('excel', true); }}>
                            <DocumentIcon className="w-4 h-4" />
                            {t('exportAllAsExcel')}
                        </ContextMenuItem>
                        {!disableEdit && (
                            <>
                                <ContextMenuSeparator />
                                <ContextMenuItem onSelect={() => { openExport('csv'); }} disabled={checked.length === 0}>
                                    <DocumentIcon className="w-4 h-4" />
                                    {t('exportSelectedAsCsv')}
                                </ContextMenuItem>
                                <ContextMenuItem onSelect={() => { openExport('excel'); }} disabled={checked.length === 0}>
                                    <DocumentIcon className="w-4 h-4" />
                                    {t('exportSelectedAsExcel')}
                                </ContextMenuItem>
                            </>
                        )}
                    </ContextMenuSubContent>
                </ContextMenuSub>
            )}
            {!limitContextMenu && isMockDataSupported && (
                <ContextMenuItem onSelect={() => { setShowMockDataSheet(true); }}>
                    <DocumentDuplicateIcon className="w-4 h-4" />
                    {t('mockData')}
                    <ContextMenuShortcut>{formatShortcut(SHORTCUTS.mockData.displayKeys)}</ContextMenuShortcut>
                </ContextMenuItem>
            )}
            {!limitContextMenu && canDeleteRows && (
                <ContextMenuItem
                    variant="destructive"
                    onSelect={() => { handleDeleteRow(index); }}
                    data-testid="context-menu-delete-row"
                >
                    <TrashIcon className="w-4 h-4 text-destructive" />
                    {t('deleteRow')}
                    <ContextMenuShortcut>{formatShortcut(SHORTCUTS.deleteRow.displayKeys)}</ContextMenuShortcut>
                </ContextMenuItem>
            )}
        </ContextMenuContent>
    );

    const actions = <>
        {isImportSupported && allowImport && (
            <Button variant="secondary" onClick={() => { setShowImport(true); }} className="flex gap-sm" data-testid="import-button">
                {cellEditing ? <WhoDBChatIcon name="upload" /> : <ArrowUpCircleIcon className="w-4 h-4" />}{t('importAction')}
            </Button>
        )}
        {isExportSupported && (
            <Button variant="secondary" onClick={() => { openExport(); }} className="flex gap-sm" data-testid="export-all-button">
                {cellEditing ? <WhoDBChatIcon name="download" /> : <ArrowDownCircleIcon className="w-4 h-4" />}{cellEditing ? t('exportAction') : hasSelectedRows ? t('exportSelected', { count: checked.length }) : t('exportAll')}
            </Button>
        )}
        {actionsTarget && children}
    </>;

    return (
        <div ref={tableRef} className="flex min-w-0 w-full">
            <div className="flex flex-col space-y-4 min-w-0 w-full" data-testid="table-container">
                <div className="sr-only" aria-live="polite" aria-atomic="true">
                    {paginatedRows.length > 0
                        ? `${paginatedRows.length} rows loaded${totalCount != null ? ` of ${totalCount} total` : ''}`
                        : ''}
                </div>
                <div style={{ width: enableKeyboardShortcuts ? '100%' : `${containerWidth}px` }}>
                    <TableComponent
                        role="grid"
                        aria-label={storageUnit ? `${storageUnit} data table` : 'Data table'}
                        aria-rowcount={paginatedRows.length}
                        aria-multiselectable={true}
                        maxHeight={actualTableHeight + 48}
                    >
                    <TableHeader data-column-count={columns.length}>
                        <ContextMenu>
                            <ContextMenuTrigger asChild>
                                    <TableHeadRow role="row" aria-rowindex={0} className="group relative cursor-context-menu hover:bg-muted/50 transition-colors" title={t('rightClickForOptions')}>
                                        <TableHead className={cn("ce-table-selection-cell min-w-[40px] w-[40px] relative", {
                                            "hidden": disableEdit,
                                        })}>
                                            <div className="ce-table-selection-control"><Checkbox
                                                checked={checked.length === paginatedRows.length && paginatedRows.length > 0}
                                                onCheckedChange={() => {
                                                    setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index));
                                                }}
                                                aria-label={checked.length === paginatedRows.length ? t('deselectAll') : t('selectAll')}
                                            /></div>
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
                                                key={col}
                                                icon={columnIsPrimary?.[idx] ? <WhoDBChatIcon name="secret" /> : columnIsForeignKey?.[idx] ? <WhoDBChatIcon name="relation" /> : columnIcons?.[idx]}
                                                className={cn(ph.mask, {
                                                    "cursor-pointer select-none": onColumnSort,
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
                                                data-column-name={col}
                                                data-sort-direction={sortedColumns?.get(col) ?? undefined}
                                            >
                                                <Tip>
                                                    <p className={cn("flex items-center gap-xs", {
                                                        "font-bold": columnIsPrimary?.[idx],
                                                        "italic": columnIsForeignKey?.[idx] && !columnIsPrimary?.[idx],
                                                    })}>
                                                        {col}
                                                        {enableKeyboardShortcuts && columnIsPrimary?.[idx] && <span className="ce-column-key">{t('primaryKeyShort')}</span>}
                                                        {enableKeyboardShortcuts && columnIsForeignKey?.[idx] && <span className="ce-column-key">{t('foreignKeyShort')}</span>}
                                                        {onColumnSort && sortedColumns?.has(col) && (
                                                            sortedColumns.get(col) === 'asc'
                                                                ? <ChevronUpIcon className="w-4 h-4" data-testid="sort-indicator" />
                                                                : <ChevronDownIcon className="w-4 h-4" data-testid="sort-indicator" />
                                                        )}
                                                    </p>
                                                    <p className="text-xs">{columnTypes?.[idx]?.toLowerCase()}</p>
                                                </Tip>
                                                {enableKeyboardShortcuts && <span className="ce-column-type">{columnTypes?.[idx]?.toLowerCase()}</span>}
                                            </TableHead>
                                        ))}
                                    </TableHeadRow>
                                </ContextMenuTrigger>
                                <ContextMenuContent
                    className="w-64 max-h-[calc(100vh-2rem)] overflow-y-auto"
                    collisionPadding={{ top: 16, right: 16, bottom: 16, left: 16 }}
                >
                                    {!limitContextMenu && isMockDataSupported && (
                                        <ContextMenuItem onSelect={() => { setShowMockDataSheet(true); }} data-testid="context-menu-mock-data">
                                            <CalculatorIcon className="w-4 h-4" />
                                            {t('mockData')}
                                            <ContextMenuShortcut>{formatShortcut(SHORTCUTS.mockData.displayKeys)}</ContextMenuShortcut>
                                        </ContextMenuItem>
                                    )}
                                    {!limitContextMenu && isMockDataSupported && isExportSupported && <ContextMenuSeparator />}
                                    {isExportSupported && (
                                        <ContextMenuSub>
                                            <ContextMenuSubTrigger>
                                                <ArrowDownCircleIcon className="w-4 h-4 mr-2" />
                                                {t('exportData')}
                                            </ContextMenuSubTrigger>
                                            <ContextMenuSubContent
                                                collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
                                            >
                                                <ContextMenuItem
                                                    onSelect={() => { openExport('csv', true); }}
                                                >
                                                    <DocumentIcon className="w-4 h-4" />
                                                    {t('exportAllAsCsv')}
                                                    <ContextMenuShortcut>{formatShortcut(SHORTCUTS.exportData.displayKeys)}</ContextMenuShortcut>
                                                </ContextMenuItem>
                                                <ContextMenuItem
                                                    onSelect={() => { openExport('excel', true); }}
                                                >
                                                    <DocumentIcon className="w-4 h-4" />
                                                    {t('exportAllAsExcel')}
                                                </ContextMenuItem>
                                                {!disableEdit && (
                                                    <>
                                                        <ContextMenuSeparator />
                                                        <ContextMenuItem
                                                            onSelect={() => { openExport('csv'); }}
                                                            disabled={checked.length === 0}
                                                        >
                                                            <DocumentIcon className="w-4 h-4" />
                                                            {t('exportSelectedAsCsv')}
                                                        </ContextMenuItem>
                                                        <ContextMenuItem
                                                            onSelect={() => { openExport('excel'); }}
                                                            disabled={checked.length === 0}
                                                        >
                                                            <DocumentIcon className="w-4 h-4" />
                                                            {t('exportSelectedAsExcel')}
                                                        </ContextMenuItem>
                                                    </>
                                                )}
                                            </ContextMenuSubContent>
                                        </ContextMenuSub>
                                    )}
                                    {(isExportSupported || (!limitContextMenu && isMockDataSupported)) && <ContextMenuSeparator />}
                                    {!limitContextMenu && (
                                        <ContextMenuItem onSelect={() => onRefresh?.()}>
                                            <CircleStackIcon className="w-4 h-4" />
                                            {t('refreshData')}
                                            <ContextMenuShortcut>{formatShortcut(SHORTCUTS.refresh.displayKeys)}</ContextMenuShortcut>
                                        </ContextMenuItem>
                                    )}
                                    {!limitContextMenu && (
                                        <ContextMenuItem
                                            onSelect={() => {
                                                setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index));
                                            }}
                                        >
                                            <CheckCircleIcon className="w-4 h-4" />
                                            {checked.length === paginatedRows.length ? t('deselectAll') : t('selectAll')}
                                            <ContextMenuShortcut>{formatShortcut(SHORTCUTS.selectAll.displayKeys)}</ContextMenuShortcut>
                                        </ContextMenuItem>
                                    )}
                                </ContextMenuContent>
                        </ContextMenu>
                    </TableHeader>
                    {paginatedRows.length > 0 && (
                        <ContextMenu>
                            <ContextMenuTrigger asChild>
                                <TableBody rowHeight={rowHeight}>
                                    {paginatedRows.map((_, rowIdx) => contextMenu(rowIdx))}
                                </TableBody>
                            </ContextMenuTrigger>
                            {contextMenuRowIdx !== null && rowContextMenuContent(contextMenuRowIdx)}
                        </ContextMenu>
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
                            <ContextMenuItem onSelect={() => { setShowMockDataSheet(true); }} className={cn({
                                "hidden": disableEdit || !isMockDataSupported,
                            })}>
                                <CalculatorIcon className="w-4 h-4" />
                                {t('mockData')}
                                <ContextMenuShortcut>{formatShortcut(SHORTCUTS.mockData.displayKeys)}</ContextMenuShortcut>
                            </ContextMenuItem>
                            {isExportSupported && (
                                <ContextMenuSub>
                                    <ContextMenuSubTrigger>
                                        <ArrowDownCircleIcon className="w-4 h-4 mr-2" />
                                        {t('export')}
                                    </ContextMenuSubTrigger>
                                    <ContextMenuSubContent
                                        collisionPadding={{ top: 20, right: 20, bottom: 20, left: 20 }}
                                    >
                                        <ContextMenuItem
                                            onSelect={() => { openExport('csv', true); }}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            {t('exportAllAsCsv')}
                                            <ContextMenuShortcut>{formatShortcut(SHORTCUTS.exportData.displayKeys)}</ContextMenuShortcut>
                                        </ContextMenuItem>
                                        <ContextMenuItem
                                            onSelect={() => { openExport('excel', true); }}
                                        >
                                            <DocumentIcon className="w-4 h-4" />
                                            {t('exportAllAsExcel')}
                                        </ContextMenuItem>
                                    </ContextMenuSubContent>
                                </ContextMenuSub>
                            )}
                        </ContextMenuContent>
                    </ContextMenu>
                )}
                </div>
                <div className={cn("flex justify-between items-center", {
                    "justify-end": children == null,
                    "mt-4": children != null && !actionsTarget,
                    "hidden": actionsTarget != null && totalPages <= 1 && !cellEditing,
                })}>
                    {!actionsTarget && children}
                    {cellEditing && totalCount != null && totalCount > 0 && <span className="ce-explore-row-summary" data-testid="total-count-bottom">{t('rowsSummary', {count: totalCount})}</span>}
                    {cellEditing && showPagination ? <nav className="ce-explore-pagination" aria-label={t('tablePagination')}>
                        <Button variant="outline" size="icon" disabled={currentPage <= 1} onClick={() => { handlePageChange(currentPage - 1); }} aria-label={t('previousPage')}>
                            <WhoDBChatIcon name="chevron-left" />
                        </Button>
                        <span>{t('pageOf', {page: currentPage, total: Math.max(1, totalPages)})}</span>
                        <Button variant="outline" size="icon" disabled={currentPage >= totalPages} onClick={() => { handlePageChange(currentPage + 1); }} aria-label={t('nextPage')}>
                            <WhoDBChatIcon name="chevron-right" />
                        </Button>
                    </nav> : <DataPagination
                        totalPages={totalPages}
                        currentPage={currentPage}
                        onPageChange={handlePageChange}
                        className={cn("flex justify-end", {
                            "hidden": !showPagination,
                        })}
                    />}
                </div>
                <div className="flex justify-end items-center mb-2 gap-4">
                    {!cellEditing && totalCount != null && totalCount > 0 && (
                        <div className="text-sm" data-testid="total-count-bottom">
                            {enableKeyboardShortcuts
                                ? t('rowsSummary', { count: totalCount })
                                : <><span className="font-semibold">{t('totalCount')}</span> {formatNumber(totalCount, language)}</>}
                        </div>
                    )}
                    {!actionsTarget && actions}
                </div>
                {actionsTarget && createPortal(actions, actionsTarget)}
                {cellEditing && hasSelectedRows && pendingDeleteIndexes == null && createPortal(
                    <div className={cn('ce-selection-bar', pendingCellCount > 0 && 'ce-selection-bar-above')} role="status" data-testid="selected-rows-bar">
                        <strong>{t('rowsSelected', {count: checked.length})}</strong>
                        {primaryColumnIndex >= 0 && <span className="ce-selection-ids">{t('selectedIds', {ids: selectedIds.join(', ')})}</span>}
                        <span className="ce-cell-save-divider" />
                        {isExportSupported && <Button variant="ghost" size="sm" onClick={() => {openExport();}}>{t('exportAction')}</Button>}
                        {canDeleteRows && <Button variant="destructive" size="sm" onClick={() => {handleDeleteRow(checked[0]);}} data-testid="delete-selected-rows">
                            <span className="ce-whodb-icon ce-whodb-icon-trash" aria-hidden="true" />{t('deleteRow', {count: checked.length})}
                        </Button>}
                    </div>, document.body
                )}
                {cellEditing && pendingCellCount > 0 && pendingDeleteIndexes == null && createPortal(
                    <div className="ce-cell-save-bar" role="status" data-testid="cell-save-bar">
                        <strong>{t('unsavedChanges', {count: pendingCellCount})}</strong>
                        <span className="ce-cell-save-context">{storageUnit} · {pendingRowIndexes.length === 1 ? t('rowNumber', {number: (currentPage - 1) * pageSize + pendingRowIndexes[0] + 1}) : t('rowsSummary', {count: pendingRowIndexes.length})}</span>
                        <span className="ce-cell-save-divider" />
                        <Button variant="ghost" size="sm" disabled={savingCells} onClick={() => {setPendingCells({}); setEditingCell(null);}}>{t('discard')}</Button>
                        <Button size="sm" disabled={savingCells} onClick={() => {void savePendingCells();}} data-testid="save-cell-changes">
                            {savingCells && <Spinner className="size-4" />}
                            {savingCells ? t('saving') : t('save')}
                        </Button>
                    </div>, document.body
                )}
                <Sheet open={editIndex !== null} onOpenChange={open => {
                    if (!open) {
                        setEditIndex(null);
                        setEditRow(null);
                        setEditRowInitialLengths([]);
                    }
                }}>
                    <SheetContent side="right" className="w-[400px] max-w-full p-8 flex flex-col" data-testid="edit-row-dialog" footer={
                        <SheetFooter className="flex gap-sm px-0">
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
                    }>
                        <SheetTitle>{t('editRow')}</SheetTitle>
                        <div className="flex flex-col gap-lg mt-4">
                                {editRow &&
                                    columns.map((col, idx) => (
                                        <div key={col} className={cn("flex flex-col gap-2", ph.noCapture)}>
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
                                                        key={`input-${col}`}
                                                        value={editRow[idx] ?? ""}
                                                        onChange={e => { handleInputChange(e.target.value, idx); }}
                                                        data-testid={`editable-field-${idx}`}
                                                        {...getInputPropsForColumnType(columnTypes?.[idx] ?? '')}
                                                    />
                                                    : <TextArea
                                                        key={`textarea-${col}`}
                                                        value={editRow[idx] ?? ""}
                                                        onChange={e => { handleInputChange(e.target.value, idx); }}
                                                        rows={5}
                                                        className="min-h-[100px]"
                                                        data-testid={`editable-field-${idx}`}
                                                    />
                                            }
                                        </div>
                                    ))}
                        </div>
                    </SheetContent>
                </Sheet>
            </div>
            <Sheet open={showMockDataSheet} onOpenChange={(open) => {
                setShowMockDataSheet(open);
                    if (!open) {
                        setShowMockDataConfirmation(false);
                    }
                }}>
                <SheetContent side="right" className="p-8" data-testid="mock-data-sheet" footer={
                    <SheetFooter className="flex gap-sm px-0">
                        <Alert variant={supportsMockDataRelations ? "info" : "default"} className="mb-4">
                            <AlertTitle>{t('mockDataNote')}</AlertTitle>
                            <AlertDescription>
                                {supportsMockDataRelations ? t('mockDataWarning') : t('mockDataWarningClickHouse')}
                            </AlertDescription>
                        </Alert>
                        <Button
                            className="flex-1"
                            variant="secondary"
                            onClick={() => { setShowMockDataSheet(false); }}
                            data-testid="cancel-mock-data"
                        >
                            {t('cancel')}
                        </Button>
                        {!showMockDataConfirmation ? (
                            <Button className="flex-1" onClick={() => { void handleMockDataGenerate(); }} disabled={generatingMockData || !mockDataRowCount || parseInt(mockDataRowCount) < 1} data-testid="mock-data-generate-button">
                                {t('generate')}
                            </Button>
                        ) : (
                            <Button className="flex-1" onClick={() => { void handleMockDataGenerate(); }} disabled={generatingMockData || !mockDataRowCount || parseInt(mockDataRowCount) < 1} variant="destructive" data-testid="mock-data-overwrite-button">
                                {t('yesOverwrite')}
                            </Button>
                        )}
                    </SheetFooter>
                }>
                    <div className="flex flex-col gap-lg">
                        <SheetTitle className="flex items-center gap-2"><CalculatorIcon className="w-4 h-4" /> {t('mockData')}</SheetTitle>
                        {!showMockDataConfirmation ? (
                            <div className="space-y-4">
                                <Label>{t('numberOfRows', { max: maxRowCount })}</Label>
                                <Input
                                    value={mockDataRowCount}
                                    onChange={e => {handleMockDataRowCountChange(e.target.value); }}
                                    type="text"
                                    inputMode="numeric"
                                    pattern="[0-9]*"
                                    max={maxRowCount.toString()}
                                    placeholder={t('enterNumberOfRows', { max: maxRowCount })}
                                    data-testid="mock-data-rows-input"
                                />
                                <Label>{t('method')}</Label>
                                <Select value={mockDataMethod} onValueChange={setMockDataMethod}>
                                    <SelectTrigger className="w-full" data-testid="mock-data-method-select">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="Normal" data-value="Normal">{t('methodNormal')}</SelectItem>
                                    </SelectContent>
                                </Select>
                                <Label>{t('dataHandling')}</Label>
                                <Select value={mockDataOverwriteExisting} onValueChange={setMockDataOverwriteExisting}>
                                    <SelectTrigger className="w-full" data-testid="mock-data-handling-select">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="append" data-value="append">{t('appendToExisting')}</SelectItem>
                                        <SelectItem value="overwrite" data-value="overwrite">{t('overwriteExisting')}</SelectItem>
                                    </SelectContent>
                                </Select>
                                {supportsMockDataRelations && (
                                    <>
                                        <div>
                                            <Label>{t('fkVariety')}</Label>
                                            <p className="text-sm text-muted-foreground mb-2">{t('fkVarietyDescription')}</p>
                                        </div>
                                        <Select value={mockDataFkDensityRatio} onValueChange={setMockDataFkDensityRatio}>
                                            <SelectTrigger className="w-full" data-testid="mock-data-fk-variety-select">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="5" data-value="5">{t('fkVarietyHigh')}</SelectItem>
                                                <SelectItem value="10" data-value="10">{t('fkVarietyMedium')}</SelectItem>
                                                <SelectItem value="20" data-value="20">{t('fkVarietyNormal')}</SelectItem>
                                                <SelectItem value="50" data-value="50">{t('fkVarietyLow')}</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </>
                                )}
                                {/* Dependency preview when FK tables will be populated */}
                                {adjustedDepAnalysis?.Error && (
                                    <Alert variant="destructive" className="mt-4">
                                        <AlertTitle>{t('dependencyError')}</AlertTitle>
                                        <AlertDescription>
                                            {adjustedDepAnalysis.Error}
                                        </AlertDescription>
                                    </Alert>
                                )}
                                {adjustedDepAnalysis && !adjustedDepAnalysis.Error && adjustedDepAnalysis.Tables && adjustedDepAnalysis.Tables.length > 1 && (
                                    <div className="mt-4 p-3 border rounded-md bg-muted/50">
                                        <p className="text-sm font-medium mb-2">{t('tablesToPopulate')}</p>
                                        <ul className="text-sm space-y-1">
                                            {adjustedDepAnalysis.Tables.map((tbl) => (
                                                <li key={tbl.Table} className="flex items-center gap-2">
                                                    <span className="font-mono">{tbl.Table}</span>
                                                    <span className="text-muted-foreground">
                                                        ({tbl.RowsToGenerate} {t('rows')})
                                                    </span>
                                                    {tbl.UsesExistingData && (
                                                        <span className="text-xs px-1.5 py-0.5 rounded bg-secondary text-secondary-foreground">
                                                            {t('usingExisting')}
                                                        </span>
                                                    )}
                                                </li>
                                            ))}
                                        </ul>
                                        <p className="text-sm text-muted-foreground mt-2">
                                            {t('totalRows', { count: adjustedDepAnalysis.TotalRows })}
                                        </p>
                                    </div>
                                )}
                                {analyzingDeps && (
                                    <div className="mt-4 flex justify-center">
                                        <Spinner />
                                    </div>
                                )}
                                {generatingMockData && (
                                    <div className="mt-8 flex justify-center">
                                        <Spinner />
                                    </div>
                                )}
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <div className="flex items-center justify-center mb-4">
                                    <div className="w-16 h-16 bg-highlight/25 rounded-full flex items-center justify-center">
                                        <XMarkIcon className="w-8 h-8 text-[var(--brand-readable)]" />
                                    </div>
                                </div>
                                <p className="text-center text-gray-700 dark:text-gray-300">
                                    {t('overwriteConfirmation', { storageUnit })}
                                </p>
                            </div>
                        )}
                    </div>
                </SheetContent>
            </Sheet>
            {isExportSupported && (
                <Suspense fallback={<Spinner />}>
                    <DynamicExport
                        open={showExportConfirm}
                        onOpenChange={setShowExportConfirm}
                        storageUnit={rawQuery ? 'query_export' : (storageUnit ?? '')}
                        objectRef={objectRef}
                        hasSelectedRows={hasSelectedRows}
                        selectedRowsData={selectedRowsData}
                        checkedRowsCount={checked.length}
                        databaseType={databaseType}
                        rawQuery={rawQuery}
                        preselectedFormat={preselectedFormat}
                        forceExportAll={forceExportAll}
                    />
                </Suspense>
            )}
            {isImportSupported && allowImport && (
                <ImportData
                    open={showImport}
                    onOpenChange={setShowImport}
                    objectRef={objectRef}
                    databaseType={databaseType}
                    onImportSuccess={onRefresh}
                />
            )}
            <AlertDialog open={pendingDeleteIndexes != null} onOpenChange={(open) => { if (!open && !deletingRows) handleCancelDelete(); }}>
                <AlertDialogContent className={cn(cellEditing && 'ce-delete-dialog')}>
                    <AlertDialogHeader>
                        {cellEditing && <span className="ce-delete-icon"><span className="ce-whodb-icon ce-whodb-icon-trash" aria-hidden="true" /></span>}
                        <AlertDialogTitle>{cellEditing ? t('deleteFromTableTitle', {count: pendingDeleteIndexes?.length ?? 1, table: storageUnit ?? ''}) : t('deleteRowConfirmTitle', { count: pendingDeleteIndexes?.length ?? 1 })}</AlertDialogTitle>
                        <AlertDialogDescription>
                            {cellEditing ? t('deleteRowsDetails', {ids: pendingDeleteIndexes?.map(index => rows[index]?.[primaryColumnIndex >= 0 ? primaryColumnIndex : 0]).join(', ') ?? '', source: sourceLabel ?? ''}) : t('deleteRowConfirmDescription', { count: pendingDeleteIndexes?.length ?? 1 })}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    {cellEditing && primaryColumnIndex >= 0 && pendingDeleteIndexes && <code className="ce-delete-preview">
                        DELETE FROM {storageUnit} WHERE {columns[primaryColumnIndex]} IN ({pendingDeleteIndexes.map(index => rows[index]?.[primaryColumnIndex]).join(', ')});
                    </code>}
                    <AlertDialogFooter>
                        <AlertDialogCancel onClick={handleCancelDelete} disabled={deletingRows}>{t('cancel')}</AlertDialogCancel>
                        <Button variant="destructive" onClick={() => { void handleConfirmDelete(); }} disabled={deletingRows} data-testid="confirm-delete-row-button">
                            {deletingRows && <Spinner className="size-4" aria-hidden="true" />}
                            {deletingRows && cellEditing ? t('deleting') : deletingRows ? t('deletingRows', {count: pendingDeleteIndexes?.length ?? 1}) : t('deleteRow', { count: pendingDeleteIndexes?.length ?? 1 })}
                        </Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
};
