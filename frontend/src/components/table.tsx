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
    Sheet,
    SheetContent,
    SheetFooter,
    SheetTitle,
    TableCell,
    Table as TableComponent,
    TableHead,
    TableHeader,
    TableRow,
    toast,
    VirtualizedTableBody
} from "@clidey/ux";
import {
    CalculatorIcon,
    CalendarIcon,
    CheckCircleIcon,
    CircleStackIcon,
    ClockIcon,
    DocumentDuplicateIcon,
    DocumentIcon,
    DocumentTextIcon,
    HashtagIcon,
    KeyIcon,
    ListBulletIcon,
} from "@heroicons/react/24/outline";
import { FC, useCallback, useMemo, useState } from "react";


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

// For delete logic, we need to accept a prop for deleting a row
interface TableProps {
    columns: string[];
    columnTypes?: string[];
    rows: string[][];
    rowHeight?: number;
    height?: number;
    onRowUpdate?: (row: Record<string, string | number>, updatedColumn: string) => Promise<void>;
    onRowDelete?: (rowIndex: number) => Promise<void> | void;
    disableEdit?: boolean;
}

export const StorageUnitTable: FC<TableProps> = ({
    columns,
    columnTypes,
    rows,
    rowHeight = 48,
    height = 500,
    onRowUpdate,
    onRowDelete,
    disableEdit = false,
}) => {
    const [editIndex, setEditIndex] = useState<number | null>(null);
    const [editRow, setEditRow] = useState<string[] | null>(null);
    const [deleting, setDeleting] = useState(false);
    const [checked, setChecked] = useState<number[]>([]);
    const [currentPage, setCurrentPage] = useState(1);
    const pageSize = 20;
    const totalRows = rows.length;
    const totalPages = Math.ceil(totalRows / pageSize);

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

    const handleUpdate = () => {
        if (editIndex !== null && editRow) {
            if (editRow && editIndex !== null) {
                const updatedRow: Record<string, string | number> = {};
                columns.forEach((col, idx) => {
                    updatedRow[col] = editRow[idx];
                });
                onRowUpdate?.(updatedRow, columns[editIndex]).then(() => {
                    setEditIndex(null);
                    setEditRow(null);
                    toast.success("Row updated");
                }).catch(() => {
                    toast.error("Error updating row");
                });
            }
            setEditIndex(null);
            setEditRow(null);
        }
    };

    // Delete logic, adapted from explore-storage-unit.tsx
    const handleDeleteRow = async (rowIndex: number) => {
        if (onRowDelete) {
            setDeleting(true);
            try {
                await onRowDelete(rowIndex);
                toast.success("Row deleted");
            } catch (e: any) {
                toast.error(`Unable to delete the row: ${e?.message || e}`);
            }
            setDeleting(false);
        }
    };

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

    const columnIcons = useMemo(() => {
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
    }, [columns, columnTypes]);

    const handleCellClick = (rowIndex: number, cellIndex: number) => {
        const cell = paginatedRows[rowIndex][cellIndex];
        if (cell !== undefined && cell !== null) {
            if (typeof navigator !== "undefined" && navigator.clipboard) {
                navigator.clipboard.writeText(String(cell));
                toast.success("Copied to clipboard");
            }
        }
    };

    return (
        <div className="flex flex-col grow h-full">
            <TableComponent className="overflow-x-auto">
                <TableHeader>
                    <TableRow>
                        <TableCell className={cn("flex items-center gap-2 w-[20rem]", {
                            "hidden": disableEdit,
                        })}>
                            <Checkbox
                                checked={checked.length === paginatedRows.length}
                                onCheckedChange={() => setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index + (currentPage - 1) * pageSize))}
                            />
                        </TableCell>
                        {columns.map((col, idx) => (
                            <TableHead key={col + idx} icon={columnIcons?.[idx]}>{col}</TableHead>
                        ))}
                    </TableRow>
                </TableHeader>
                <VirtualizedTableBody rowCount={paginatedRows.length} rowHeight={rowHeight} height={height}>
                    {(index) => {
                        const globalIndex = (currentPage - 1) * pageSize + index;
                        return (
                            <ContextMenu key={globalIndex}>
                                <ContextMenuTrigger
                                    asChild
                                    className="contents"
                                >
                                    <tr>
                                        <TableCell className={cn("w-[20rem]", {
                                            "hidden": disableEdit,
                                        })}>
                                            <Checkbox
                                                checked={checked.includes(globalIndex)}
                                                onCheckedChange={() => setChecked(checked.includes(globalIndex) ? checked.filter(i => i !== globalIndex) : [...checked, globalIndex])}
                                            />
                                        </TableCell>
                                        {paginatedRows[index]?.map((cell, cellIdx) => (
                                            <TableCell key={cellIdx} className="cursor-pointer" onClick={() => handleCellClick(globalIndex, cellIdx)}>{cell}</TableCell>
                                        ))}
                                    </tr>
                                </ContextMenuTrigger>
                                <ContextMenuContent className="w-52">
                                    <ContextMenuItem onSelect={() => handleSelectRow(globalIndex)}>
                                        {checked.includes(globalIndex) ? "Deselect Row" : "Select Row"}
                                    </ContextMenuItem>
                                    <ContextMenuItem onSelect={() => handleEdit(globalIndex)} disabled={checked.length > 0}>
                                        Edit Row
                                        <ContextMenuShortcut>⌘E</ContextMenuShortcut>
                                    </ContextMenuItem>
                                    <ContextMenuSub>
                                        <ContextMenuSubTrigger>Export</ContextMenuSubTrigger>
                                        <ContextMenuSubContent className="w-44">
                                            <ContextMenuItem>
                                                <DocumentIcon className="w-4 h-4" />
                                                Export CSV
                                            </ContextMenuItem>
                                            <ContextMenuItem>
                                                <DocumentIcon className="w-4 h-4" />
                                                Export Excel
                                            </ContextMenuItem>
                                        </ContextMenuSubContent>
                                    </ContextMenuSub>
                                    <ContextMenuSub>
                                        <ContextMenuSubTrigger>More Actions</ContextMenuSubTrigger>
                                        <ContextMenuSubContent className="w-44">
                                            <ContextMenuItem
                                                variant="destructive"
                                                disabled={deleting}
                                                onSelect={async () => {
                                                    await handleDeleteRow(globalIndex);
                                                }}
                                            >
                                                Delete Row
                                            </ContextMenuItem>
                                        </ContextMenuSubContent>
                                    </ContextMenuSub>
                                    <ContextMenuSeparator />
                                    <ContextMenuItem onSelect={() => handleEdit(globalIndex)}>
                                        Open in Graph View
                                        <ContextMenuShortcut>⌘G</ContextMenuShortcut>
                                    </ContextMenuItem>
                                </ContextMenuContent>
                            </ContextMenu>
                        );
                    }}
                </VirtualizedTableBody>
            </TableComponent>
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
                                <div key={col} className="flex flex-col gap-2">
                                    <Label>{col}</Label>
                                    <Input
                                        value={editRow[idx] ?? ""}
                                        onChange={e => handleInputChange(e.target.value, idx)}
                                    />
                                </div>
                            ))}
                    </div>
                    <SheetFooter className="flex gap-2">
                        <Button onClick={handleUpdate} disabled={!editRow}>
                            Update
                        </Button>
                    </SheetFooter>
                </SheetContent>
            </Sheet>
        </div>
    );
};
