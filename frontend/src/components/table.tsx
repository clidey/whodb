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
    Input,
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
import { FC, useState } from "react";

interface TableProps {
    columns: string[];
    rows: string[][];
    rowHeight?: number;
    height?: number;
    onRowUpdate?: (row: Record<string, string | number>, updatedColumn: string) => Promise<void>;
    disableEdit?: boolean;
}

export const StorageUnitTable: FC<TableProps> = ({
    columns,
    rows,
    rowHeight = 48,
    height = 500,
    onRowUpdate,
    disableEdit = false,
}) => {
    const [editIndex, setEditIndex] = useState<number | null>(null);
    const [editRow, setEditRow] = useState<string[] | null>(null);

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
                console.log(updatedRow);
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

    return (
        <>
            <TableComponent>
                <TableHeader>
                    <TableRow>
                        <TableCell className="flex items-center gap-2 w-[20rem]">
                            <Checkbox />
                        </TableCell>
                        {columns.map((col, idx) => (
                            <TableHead key={col + idx}>{col}</TableHead>
                        ))}
                        <TableHead>Actions</TableHead>
                    </TableRow>
                </TableHeader>
                <VirtualizedTableBody rowCount={rows.length} rowHeight={rowHeight} height={height}>
                    {(index) => (
                        <>
                            <TableCell className="w-[20rem]">
                                <Checkbox />
                            </TableCell>
                            {rows[index]?.map((cell, cellIdx) => (
                                <TableCell key={cellIdx}>{cell}</TableCell>
                            ))}
                            <TableCell>
                                {!disableEdit && (
                                <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => handleEdit(index)}
                                >
                                        Edit
                                    </Button>
                                )}
                            </TableCell>
                        </>
                    )}
                </VirtualizedTableBody>
            </TableComponent>
            <Sheet open={editIndex !== null} onOpenChange={open => { if (!open) setEditIndex(null); }}>
                <SheetContent side="right" className="w-[400px] max-w-full p-8">
                    <SheetTitle>Edit Row</SheetTitle>
                    <div className="flex flex-col gap-4 mt-4">
                        {editRow &&
                            columns.map((col, idx) => (
                                <div key={col} className="flex flex-col gap-1">
                                    <label className="text-xs font-medium text-neutral-700 dark:text-neutral-300">{col}</label>
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
        </>
    );
};