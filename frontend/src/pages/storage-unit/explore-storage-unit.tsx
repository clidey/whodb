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

import { FetchResult } from "@apollo/client";
import { Button, Drawer, DrawerTitle, DrawerContent, DrawerHeader, Input, Label, Sheet, SheetContent, SheetFooter, toast, DrawerFooter, SearchInput } from "@clidey/ux";
import {
    DatabaseType, DeleteRowDocument, DeleteRowMutationResult, RecordInput, RowsResult, StorageUnit,
    UpdateStorageUnitDocument, UpdateStorageUnitMutationResult, useAddRowMutation, useGetStorageUnitRowsLazyQuery,
    useRawExecuteLazyQuery,
    WhereCondition
} from '@graphql';
import { clone, entries, keys, map } from "lodash";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { Icons } from "../../components/icons";
import { Loading, LoadingPage } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { StorageUnitTable } from "../../components/table";
import { graphqlClient } from "../../config/graphql-client";
import { InternalRoutes } from "../../config/routes";
import { useAppSelector } from "../../store/hooks";
import { getDatabaseOperators } from "../../utils/database-operators";
import { getDatabaseStorageUnitLabel } from "../../utils/functions";
import { ExploreStorageUnitWhereCondition } from "./explore-storage-unit-where-condition";
import { CodeEditor } from "../../components/editor";
import { PlayIcon, XMarkIcon } from "@heroicons/react/24/outline";

export const ExploreStorageUnit: FC<{ scratchpad?: boolean }> = ({ scratchpad }) => {
    const [bufferPageSize, setBufferPageSize] = useState("100");
    const [currentPage, setCurrentPage] = useState(0);
    const [whereCondition, setWhereCondition] = useState<WhereCondition>();
    const [pageSize, setPageSize] = useState("");
    const unit: StorageUnit = useLocation().state?.unit;
    const pathname = useLocation().pathname;

    let schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const navigate = useNavigate();
    const [rows, setRows] = useState<RowsResult>();
    const [showAdd, setShowAdd] = useState(false);
    const [checkedRows, setCheckedRows] = useState<Set<number>>(new Set());
    const [deleting, setDeleting] = useState(false);

    // For add row sheet logic
    const [addRowData, setAddRowData] = useState<Record<string, any>>({});
    const [addRowError, setAddRowError] = useState<string | null>(null);
    

    // For scratchpad sheet logic
    // todo: is there a different way to do this? clickhouse doesn't have schemas as a table is considered a schema. people mainly switch between DB
    if (current?.Type === DatabaseType.ClickHouse) {
        schema = current.Database
    }

    const [code, setCode] = useState(`SELECT * FROM ${schema}.${unit?.Name}`);

    const [getStorageUnitRows, { loading }] = useGetStorageUnitRowsLazyQuery({
        onCompleted(data) {
            setRows(data.Row);
            setPageSize(bufferPageSize);
        },
        fetchPolicy: "no-cache",
    });
    const [addRow, { loading: adding }] = useAddRowMutation();
    const [rawExecute, { data: rawExecuteData, called }] = useRawExecuteLazyQuery();

    const unitName = useMemo(() => {
        return unit?.Name;
    }, [unit]);

    const handleSubmitRequest = useCallback(() => {
        getStorageUnitRows({
            variables: {
                schema,
                storageUnit: unitName,
                where: whereCondition,
                pageSize: Number.parseInt(bufferPageSize),
                pageOffset: currentPage,
            },
        });
    }, [getStorageUnitRows, schema, unitName, whereCondition, bufferPageSize, currentPage]);

    const handleQuery = useCallback(() => {
        handleSubmitRequest();
        setCurrentPage(0);
    }, [handleSubmitRequest]);

    const handleRowUpdate = useCallback((row: Record<string, string | number>, updatedColumn: string) => {
        if (current == null) {
            return Promise.reject();
        }
        const values = map(entries(row), ([Key, Value]) => ({
            Key,
            Value,
        }));
        const updatedColumns = [updatedColumn]
        return new Promise<void>(async (res, rej) => {
            try {
                const { data }: FetchResult<UpdateStorageUnitMutationResult["data"]> = await graphqlClient.mutate({
                    mutation: UpdateStorageUnitDocument,
                    variables: {
                        schema,
                        storageUnit: unitName,
                        type: current.Type as DatabaseType,
                        values,
                        updatedColumns,
                    },
                });
                if (data?.UpdateStorageUnit.Status) {
                    return res();
                }
                return rej();
            } catch (err) {
                return rej(err);
            }
        });
    }, [current, schema, unitName]);

    const handleRowDelete = useCallback(async () => {
        if (current == null || rows == null || checkedRows.size === 0) {
            return;
        }
        let unableToDeleteAll = false;
        setDeleting(true);
        const deletedIndexes = [];
        for (const index of [...checkedRows].sort()) {
            const row = rows.Rows[index];
            if (row == null) {
                continue;
            }
            const values = map(rows.Columns, (column, i) => ({
                Key: column.Name,
                Value: row[i],
            }));
            try {
                await new Promise<void>(async (res, rej) => {
                    try {
                        const { data }: FetchResult<DeleteRowMutationResult["data"]> = await graphqlClient.mutate({
                            mutation: DeleteRowDocument,
                            variables: {
                                schema,
                                storageUnit: unitName,
                                type: current.Type as DatabaseType,
                                values,
                            }
                        });
                        if (data?.DeleteRow.Status) {
                            return res();
                        }
                        return rej();
                    } catch (err) {
                        return rej(err);
                    }
                });
                deletedIndexes.push(index);
            } catch (e) {
                if ((checkedRows.size-1) > index) {
                    toast.error(`Unable to delete the row: ${e}. Stopping deleting other selected rows.`);
                } else {
                    toast.error(`Unable to delete the row: ${e}`);
                }
                setDeleting(false);
                unableToDeleteAll=true;
                break;
            }
        }
        const newRows = clone(rows.Rows);
        const newCheckedRows = new Set(checkedRows);
        for (const deletedIndex of deletedIndexes.reverse()) {
            newRows.splice(deletedIndex, 1);
            newCheckedRows.delete(deletedIndex);
        }
        setRows({
            ...rows,
            Rows: newRows,
        });
        setCheckedRows(newCheckedRows);
        if (!unableToDeleteAll) {
            toast.success("Row deleted successfully!");
        }
        setDeleting(false);
    }, [checkedRows, current, rows, schema, unitName]);

    const totalCount = useMemo(() => {
        const count = unit?.Attributes.find(attribute => attribute.Key === "Count")?.Value;
        if (count == null) {
            return "<50";
        }
        if (count == "unknown") {
            return rows?.Rows.length?.toString() ?? "unknown";
        }
        return count;
    }, [unit]);

    useEffect(() => {
        handleSubmitRequest();
        setCode(`SELECT * FROM ${schema}.${unit?.Name}`);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [unit]);

    const routes = useMemo(() => {
        const name = getDatabaseStorageUnitLabel(current?.Type);
        return [
            {
                ...InternalRoutes.Dashboard.StorageUnit,
                name,
            },
            InternalRoutes.Dashboard.ExploreStorageUnit,
            ...(scratchpad ? [InternalRoutes.Dashboard.ExploreStorageUnitWithScratchpad] : []),
        ];
    }, [current]);
    
    const {columns, columnTypes} = useMemo(() => {
        const dataColumns = rows?.Columns.map(c => c.Name) ?? [];
        if (dataColumns.length !== 1 || dataColumns[0] !== "document") {
            return {columns: dataColumns, columnTypes: rows?.Columns.map(column => column.Type)};
        }
        const firstRow = rows?.Rows?.[0];
        if (firstRow == null) {
            return {columns: [], columnTypes: []}
        }
        return  { columns: keys(JSON.parse(firstRow[0])), columnTypes: []}
    }, [rows?.Columns, rows?.Rows]);

    useEffect(() => {
        if (unit == null) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
    }, [navigate, unit]);

    const handleFilterChange = useCallback((filters: WhereCondition) => {
        setWhereCondition(filters);
    }, []);

    const validOperators = useMemo(() => {
        if (!current?.Type) {
            return [];
        }
        return getDatabaseOperators(current.Type);
    }, [current?.Type]);

    // Sheet logic for Add Row (like table.tsx)
    const handleOpenAddSheet = useCallback(() => {
        // Prepare default values for addRowData
        let initialData: Record<string, any> = {};
        if (rows?.Columns) {
            for (const col of rows.Columns) {
                if (col.Name.toLowerCase() === "id" && col.Type === "UUID") {
                    initialData[col.Name] = "gen_random_uuid()";
                } else if (col.Type === "TIMESTAMPTZ") {
                    initialData[col.Name] = "now()";
                } else if (col.Type === "NUMERIC") {
                    initialData[col.Name] = "0";
                } else {
                    initialData[col.Name] = "";
                }
            }
        }
        setAddRowData(initialData);
        setAddRowError(null);
        setShowAdd(true);
    }, [rows?.Columns]);

    const handleAddRowFieldChange = useCallback((key: string, value: string) => {
        setAddRowData(prev => ({
            ...prev,
            [key]: value,
        }));
    }, []);

    const handleAddRowSubmit = useCallback(() => {
        if (!rows?.Columns) return;
        // Prepare values as RecordInput[]
        let values: RecordInput[] = [];
        for (const col of rows.Columns) {
            if (addRowData[col.Name] !== undefined && addRowData[col.Name] !== "") {
                values.push({
                    Key: col.Name,
                    Value: addRowData[col.Name],
                });
            }
        }
        if (values.length === 0) {
            setAddRowError("Please fill at least one value.");
            return;
        }
        addRow({
            variables: {
                schema,
                storageUnit: unit.Name,
                values,
            },
            onCompleted() {
                toast.success("Added data row successfully!");
                setShowAdd(false);
                setTimeout(() => {
                    handleSubmitRequest();
                }, 500);
            },
            onError(e) {
                setAddRowError(e.message);
                toast.error(`Unable to add the data row: ${e.message}`);
            },
        });
    }, [addRow, addRowData, handleSubmitRequest, rows?.Columns, schema, unit?.Name]);

    const handleScratchpad = useCallback(() => {
        if (current == null) {
            return;
        }
        rawExecute({
            variables: {
                query: code,
            },
        });
    }, [code, current, rawExecute]);

    const handleOpenScratchpad = useCallback(() => {
        navigate(InternalRoutes.Dashboard.ExploreStorageUnitWithScratchpad.path, {
            state: {
                unit,
            }
        });
        handleScratchpad();
        setCode(`SELECT * FROM ${schema}.${unit?.Name}`);
        document.body.classList.add("!pointer-events-auto");
    }, [schema, unit]);

    const handleCloseScratchpad = useCallback(() => {
        navigate(InternalRoutes.Dashboard.ExploreStorageUnit.path, {
            state: {
                unit,
            }
        });
    }, [unit]);

    if (unit == null) {
        return <Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />
    }

    if (loading) {
        return <InternalPage routes={routes}>
            <LoadingPage />
        </InternalPage>
    }

    return <InternalPage routes={routes} className="relative">
        <div className="flex flex-col grow gap-4 h-[calc(100%-100px)]">
            <div className="flex items-center justify-between">
                <div className="flex gap-2 items-center">
                    <h1 className="text-xl font-bold mr-4">{unitName}</h1>
                </div>
                <div className="text-sm mr-4"><span className="font-semibold">Total Count:</span> {totalCount}</div>
            </div>
            <div className="flex w-full relative">
                <div className="flex justify-between items-end w-full">
                    <div className="flex gap-2 items-end">
                        <div className="flex flex-col gap-2">
                            <Label>Search</Label>
                            <SearchInput placeholder="Enter search query" className="w-64" />
                        </div>
                        <div className="flex flex-col gap-2">
                            <Label>Page Size</Label>
                            <Input value={bufferPageSize} onChange={e => setBufferPageSize(e.target.value)} data-testid="table-page-size" />
                        </div>
                        { current?.Type !== DatabaseType.Redis && <ExploreStorageUnitWhereCondition defaultWhere={whereCondition} columns={columns} operators={validOperators} onChange={handleFilterChange} columnTypes={columnTypes ?? []} /> }
                        <Button className="ml-6" onClick={handleQuery} data-testid="submit-button">
                            {Icons.CheckCircle} Query
                        </Button>
                    </div>
                    <div className="flex justify-end gap-2">
                        {adding || deleting ? <Loading /> : null}
                        {checkedRows.size > 0 && <Button variant="destructive" onClick={handleRowDelete} disabled={deleting} data-testid="delete-button">
                            {Icons.Delete} {checkedRows.size > 1 ? "Delete rows" : "Delete row"}
                        </Button> }
                        <Button onClick={handleOpenScratchpad} data-testid="scratchpad-button" variant="secondary">
                            {Icons.Code} Scratchpad
                        </Button>
                        <Button onClick={handleOpenAddSheet} disabled={adding} data-testid="add-button">
                            {Icons.Add} Add Row
                        </Button>
                    </div>
                </div>
                <Sheet open={showAdd} onOpenChange={setShowAdd}>
                    <SheetContent side="right" className="p-8">
                        <div className="flex flex-col gap-4 h-full">
                            <div className="text-lg font-semibold mb-2">Add new row</div>
                            <div className="flex flex-col gap-4">
                                {rows?.Columns?.map((col) => (
                                    <div key={col.Name} className="flex flex-col gap-2">
                                        <Label>
                                            {col.Name} <span className="italic">[{col.Type}]</span>
                                        </Label>
                                        <Input
                                            value={addRowData[col.Name] ?? ""}
                                            onChange={e => handleAddRowFieldChange(col.Name, e.target.value)}
                                            placeholder={`Enter value for ${col.Name}`}
                                        />
                                    </div>
                                ))}
                            </div>
                            {addRowError && (
                                <div className="text-red-500 text-xs">{addRowError}</div>
                            )}
                        </div>
                        <SheetFooter className="px-0">
                            <Button onClick={handleAddRowSubmit} data-testid="submit-button" disabled={adding}>
                                {Icons.CheckCircle} Submit
                            </Button>
                        </SheetFooter>
                    </SheetContent>
                </Sheet>
            </div>
            <div className="grow">
                {
                    rows != null &&
                    <StorageUnitTable columns={columns} rows={rows.Rows} onRowUpdate={handleRowUpdate} columnTypes={columnTypes} />
                }
            </div>
        </div>
        <Drawer open={scratchpad} onOpenChange={handleCloseScratchpad}>
            <DrawerContent className="px-8 max-h-[25vh]">
                <DrawerHeader>
                    <DrawerTitle className="flex justify-between items-center">
                        <h2 className="text-lg font-semibold">Scratchpad</h2>
                        <div className="flex gap-2 items-center">
                            <Button onClick={handleScratchpad} data-testid="submit-button">
                                <PlayIcon className="w-4 h-4" />
                                Run
                            </Button>
                        </div>
                    </DrawerTitle>
                </DrawerHeader>
                <div className="flex flex-col gap-2 h-[150px]">
                    <CodeEditor language="sql" value={code} setValue={setCode} onRun={() => handleScratchpad()} />
                </div>
                <DrawerFooter>
                    <StorageUnitTable
                        height={300}
                        columns={rawExecuteData?.RawExecute.Columns.map(c => c.Name) ?? []}
                        columnTypes={rawExecuteData?.RawExecute.Columns.map(c => c.Type) ?? []}
                        rows={rawExecuteData?.RawExecute.Rows ?? []}
                        disableEdit={true}
                    />
                </DrawerFooter>
            </DrawerContent>
        </Drawer>
    </InternalPage>
}
