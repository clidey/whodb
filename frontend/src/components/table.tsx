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
import { FC, useCallback, useMemo, useState } from "react";

// For delete logic, we need to accept a prop for deleting a row
interface TableProps {
    columns: string[];
    rows: string[][];
    rowHeight?: number;
    height?: number;
    onRowUpdate?: (row: Record<string, string | number>, updatedColumn: string) => Promise<void>;
    onRowDelete?: (rowIndex: number) => Promise<void> | void;
    disableEdit?: boolean;
}

export const StorageUnitTable: FC<TableProps> = ({
    columns,
    rows,
    rowHeight = 48,
    height = 500,
    onRowUpdate,
    onRowDelete,
    disableEdit = false,
}) => {
    const [editIndex, setEditIndex] = useState<number | null>(null);
    const [editRow, setEditRow] = useState<string[] | null>(null);
    const [contextMenuRow, setContextMenuRow] = useState<number | null>(null);
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

    return (
        <div className="flex flex-col grow h-full">
            <TableComponent>
                <TableHeader>
                    <TableRow>
                        <TableCell className="flex items-center gap-2 w-[20rem]">
                            <Checkbox
                                checked={checked.length === paginatedRows.length}
                                onCheckedChange={() => setChecked(checked.length === paginatedRows.length ? [] : paginatedRows.map((_, index) => index + (currentPage - 1) * pageSize))}
                            />
                        </TableCell>
                        {columns.map((col, idx) => (
                            <TableHead key={col + idx}>{col}</TableHead>
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
                                    onContextMenu={() => setContextMenuRow(globalIndex)}
                                    className="contents"
                                >
                                    <tr>
                                        <TableCell className="w-[20rem]">
                                            <Checkbox
                                                checked={checked.includes(globalIndex)}
                                                onCheckedChange={() => setChecked(checked.includes(globalIndex) ? checked.filter(i => i !== globalIndex) : [...checked, globalIndex])}
                                            />
                                        </TableCell>
                                        {paginatedRows[index]?.map((cell, cellIdx) => (
                                            <TableCell key={cellIdx} className="cursor-pointer">{cell}</TableCell>
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
                <Pagination className="flex justify-end">
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