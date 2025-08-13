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
import classNames from "classnames";
import { motion } from "framer-motion";
import { clone, entries, keys, map } from "lodash";
import {cloneElement, FC, useCallback, useEffect, useMemo, useRef, useState} from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { AnimatedButton } from "../../components/button";
import { Dropdown } from "../../components/dropdown";
import { Icons } from "../../components/icons";
import { Input, InputWithlabel } from "../../components/input";
import { Loading, LoadingPage } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Table } from "../../components/table";
import { graphqlClient } from "../../config/graphql-client";
import { InternalRoutes } from "../../config/routes";
import {
    Column, DatabaseType, DeleteRowDocument, DeleteRowMutationResult, RecordInput, RowsResult, StorageUnit,
    UpdateStorageUnitDocument, UpdateStorageUnitMutationResult, useAddRowMutation, useGetStorageUnitRowsLazyQuery,
    WhereCondition
} from '@graphql';
import { notify } from "../../store/function";
import { useAppSelector } from "../../store/hooks";
import { getDatabaseStorageUnitLabel, isNoSQL, isNumeric } from "../../utils/functions";
import { getDatabaseOperators } from "../../utils/database-operators";
import { ExploreStorageUnitWhereCondition } from "./explore-storage-unit-where-condition";
import { MockDataDialog } from "../../components/mock-data-dialog";


export const ExploreStorageUnit: FC = () => {
    const [bufferPageSize, setBufferPageSize] = useState("100");
    const [currentPage, setCurrentPage] = useState(0);
    const [whereCondition, setWhereCondition] = useState<WhereCondition>();
    const [pageSize, setPageSize] = useState("");
    const unit: StorageUnit = useLocation().state?.unit;
    let schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const navigate = useNavigate();
    const [rows, setRows] = useState<RowsResult>();
    const [showAdd, setShowAdd] = useState(false);
    const [newRowForm, setNewRowForm] = useState<RecordInput[]>([]);
    const [checkedRows, setCheckedRows] = useState<Set<number>>(new Set());
    const [deleting, setDeleting] = useState(false);
    const [showMockDataDialog, setShowMockDataDialog] = useState(false);
    const addRowRef = useRef<HTMLDivElement>(null);

    const hasFormContent = useCallback(() => {
        // Check if any form field has been modified from default values
        return newRowForm.some(field => {
            // If it's an ID or date field with default values, don't count it
            const isDefault =
                (field.Key.toLowerCase() === "id" && field.Value === "gen_random_uuid()") ||
                (field.Extra?.at(1)?.Value === "TIMESTAMPTZ" && field.Value === "now()") ||
                (field.Extra?.at(1)?.Value === "NUMERIC" && field.Value === "0") ||
                field.Value === "";

            return !isDefault;
        });
    }, [newRowForm]);

    const handleKeyDown = useCallback((e: KeyboardEvent) => {
        if (e.key === 'Escape' && showAdd) {
            // Only close if no content has been entered
            if (!hasFormContent()) {
                setShowAdd(false);
            }
        }
    }, [showAdd, hasFormContent]);

    useEffect(() => {
        if (showAdd) {
            document.addEventListener('keydown', handleKeyDown);
            return () => {
                document.removeEventListener('keydown', handleKeyDown);
            };
        }
    }, [showAdd, handleKeyDown]);

    const handleClickOutside = useCallback((e: MouseEvent) => {
        if (showAdd && addRowRef.current && !addRowRef.current.contains(e.target as Node)) {
            // Only close if no content has been entered
            if (!hasFormContent()) {
                setShowAdd(false);
            }
        }
    }, [showAdd, hasFormContent]);

    useEffect(() => {
        if (showAdd) {
            document.addEventListener('mousedown', handleClickOutside);
            return () => {
                document.removeEventListener('mousedown', handleClickOutside);
            };
        }
    }, [showAdd, handleClickOutside]);

    // todo: is there a different way to do this? clickhouse doesn't have schemas as a table is considered a schema. people mainly switch between DB
    if (current?.Type === DatabaseType.ClickHouse) {
        schema = current.Database
    }

    const [getStorageUnitRows, { loading }] = useGetStorageUnitRowsLazyQuery({
        onCompleted(data) {
            setRows(data.Row);
            setPageSize(bufferPageSize);
        },
        fetchPolicy: "no-cache",
    });
    const [addRow, { loading: adding }] = useAddRowMutation();

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

    const handlePageChange = useCallback((page: number) => {
        setCurrentPage(page-1);
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
            // this method ensures that the component is not rerendered
            // hence, the edited cache in the table would stay intact & performant
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
                    notify("Row updated successfully!", "success");
                    return res();
                }
                notify("Unable to update the row!", "error");
            } catch (err) {
                notify(`Unable to update the row: ${err}`, "error");
            }
            return rej();
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
                    notify(`Unable to delete the row: ${e}. Stopping deleting other selected rows.`, "error");
                } else {
                    notify(`Unable to delete the row: ${e}`, "error");
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
            notify("Row deleted successfully!", "success");
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

    const totalPages = useMemo(() => {
        if (!isNumeric(totalCount) || !isNumeric(pageSize)) {
            return 1;
        }
        return Math.max(Math.round(Number.parseInt(totalCount)/(Number.parseInt(pageSize)+1)), 1);
    }, [pageSize, totalCount]);

    useEffect(() => {
        handleSubmitRequest();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const routes = useMemo(() => {
        const name = getDatabaseStorageUnitLabel(current?.Type);
        return [
            {
                ...InternalRoutes.Dashboard.StorageUnit,
                name,
            },
            InternalRoutes.Dashboard.ExploreStorageUnit
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
        if (unitName == null) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
    }, [navigate, unitName]);

    const handleFilterChange = useCallback((filters: WhereCondition) => {
        setWhereCondition(filters);
    }, []);

    const validOperators = useMemo(() => {
        if (!current?.Type) {
            return [];
        }
        
        return getDatabaseOperators(current.Type);
    }, [current?.Type]);

    const handleToggleShowAdd = useCallback(() => {
        const showAddStatus = !showAdd;
        if (showAddStatus) {
            if (newRowForm.length === 0) {
                let columns: Column[] = [];
                if (isNoSQL(current?.Type as DatabaseType)) {
                    if (rows?.Rows != null && rows.Rows.length > 0) {
                        columns = entries(JSON.parse(rows.Rows[0][0])).filter(([col,]) => col !== "_id").map(([col, value]) => ({
                            Name: col,
                            Type: typeof value,
                        }));
                    }
                }
                if (columns.length === 0) {
                    columns = rows?.Columns ?? [];
                }
                setNewRowForm((columns.map(col => {
                    const colName = col.Name.toLowerCase();
                    const isId = colName === "id" && col.Type === "UUID";
                    const isDate = col.Type === "TIMESTAMPTZ";
                    const isNumeric = col.Type === "NUMERIC";
                    const isCode = isId || isDate;
                    return {
                        Key: col.Name,
                        Value: isId ? "gen_random_uuid()" : isDate ? "now()" : isNumeric ? "0" : "",
                        Extra: [
                            {
                                Key: "Config",
                                Value: isCode ? "sql" : "text",
                            },
                            {
                                Key: "Type",
                                Value: col.Type,
                            },
                        ],
                    }
                })));
            }
        }
        setShowAdd(showAddStatus);
    }, [current?.Type, newRowForm.length, rows?.Columns, rows?.Rows, showAdd]);

    const handleAddSubmitRequest = useCallback(() => {
        let values = newRowForm;
        values = values.filter(item => item.Value != '')  // remove empty fields
        if (isNoSQL(current?.Type as DatabaseType) && rows?.Rows != null && rows.Rows.length === 0) {
            try {
                values = entries(JSON.parse(newRowForm[0].Value)).map(([Key, Value]) => ({ Key, Value } as RecordInput));
            } catch {
                values = [];
            }
        }
        addRow({
            variables: {
                schema,
                storageUnit: unit.Name,
                values,
            },
            onCompleted() {
                notify("Added data row successfully!", "success");
                setShowAdd(false);
                setTimeout(() => {
                    handleSubmitRequest();
                }, 500);
            },
            onError(e) {
                notify(`Unable to add the data row: ${e.message}`, "error");
            },
        });
    }, [addRow, current?.Type, handleSubmitRequest, newRowForm, rows?.Rows, schema, unit?.Name]);

    const handleNewFormChange = useCallback((type: "value" | "config", index: number, value: string) => {
        setNewRowForm(rowForm => {
            const newFormClone = clone(rowForm);
            if (type === "value") {
                newFormClone[index].Value = value;
            } else {
                newFormClone[index].Extra![0].Value = value;
            }
                
            return newFormClone;
        });
    }, []);

    const configDropdown = useMemo(() => {
        return [{id: "text", label: "Text", icon: cloneElement(Icons.Text, {
            className: "w-4 h-4",
        })}, { id: "sql", label: "SQL", icon: cloneElement(Icons.Code, {
            className: "w-4 h-4",
        })}];
    }, []);

    if (unit == null) {
        return <Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />
    }

    if (loading) {
        return <InternalPage routes={routes}>
            <LoadingPage />
        </InternalPage>
    }

    return <InternalPage routes={routes}>
        <div className="flex flex-col grow gap-4 h-[calc(100%-100px)]">
            <div className="flex items-center justify-between">
                <div className="flex gap-2 items-center">
                    <div className="text-xl font-bold mr-4 dark:text-neutral-300">{unitName}</div>
                </div>
                <div className="text-sm mr-4 dark:text-neutral-300"><span className="font-semibold">Total Count:</span> {totalCount}</div>
            </div>
            <div className="flex w-full relative">
                <div className="flex gap-2">
                    <InputWithlabel label="Page Size" value={bufferPageSize} setValue={setBufferPageSize} testId="table-page-size" />
                    { current?.Type !== DatabaseType.Redis && <ExploreStorageUnitWhereCondition defaultWhere={whereCondition} columns={columns} operators={validOperators} onChange={handleFilterChange} columnTypes={columnTypes ?? []} /> }
                    <AnimatedButton className="mt-5" type="lg" icon={Icons.CheckCircle} label="Query" onClick={handleQuery} testId="submit-button" />
                </div>
                <motion.div tabIndex={0} ref={addRowRef} className={classNames("flex flex-col absolute z-10 right-0 top-0 backdrop-blur-xl", {
                        "hidden": current?.Type === DatabaseType.Redis,
                    })} variants={{
                    "open": {
                        height: "500px",
                        width: "800px",
                    },
                    "close": {
                        height: "35px",
                        width: "fit-content",
                    },
                }} animate={showAdd ? "open" : "close"}>
                    <div className="flex w-full justify-end gap-2">
                        {adding || deleting && <Loading />}
                        {checkedRows.size > 0 && <AnimatedButton type="lg" icon={Icons.Delete} label={checkedRows.size > 1 ? "Delete rows" : "Delete row"} iconClassName="stroke-red-500 dark:stroke-red-500" labelClassName="text-red-500 dark:text-red-500" onClick={handleRowDelete} disabled={deleting} /> }
                        <AnimatedButton type="lg" icon={Icons.Database} label="Generate Mock Data" onClick={() => setShowMockDataDialog(true)} disabled={adding || deleting} />
                        <AnimatedButton type="lg" icon={Icons.Add} label={showAdd ? "Cancel" : "Add Row"} onClick={handleToggleShowAdd} disabled={adding} />
                    </div>
                    <div className={classNames("flex flex-col gap-2 overflow-y-auto h-full p-8 mt-2", {
                        "flex border border-white/5 rounded-lg": showAdd,
                        "hidden": !showAdd,
                    })}>
                        <div className="flex justify-between gap-4">
                            <div
                                className="text-lg text-neutral-800 dark:text-neutral-300 w-full border-b border-white/10 pb-2 mb-2">
                                Add new row
                            </div>
                        </div>
                        {newRowForm.map((col, i) => <>
                            <div key={`add-row-${col.Key}`} className="flex gap-2 items-center">
                                <div
                                    className="text-xs text-neutral-800 dark:text-neutral-300 w-[150px]">{col.Key} [{col.Extra?.at(1)?.Value}]
                                </div>
                                <Dropdown className={classNames({
                                    "hidden": isNoSQL(current?.Type as DatabaseType),
                                })} value={configDropdown.find(item => item.id === col.Extra?.at(0)?.Value)}
                                          onChange={item => handleNewFormChange("config", i, item.id)}
                                          items={configDropdown}
                                          showIconOnly={true}/>
                                <Input value={col.Value} inputProps={{
                                    placeholder: `Enter value for ${col.Key}`,
                                }} setValue={(value) => handleNewFormChange("value", i, value)}/>
                            </div>
                        </>)}
                        <div className="flex justify-end gap-4 mt-2">
                            <AnimatedButton className="px-3" type="lg" icon={Icons.CheckCircle} label="Submit"
                                            onClick={handleAddSubmitRequest}/>
                        </div>
                    </div>
                </motion.div>
            </div>
            <div className="grow">
                {
                    rows != null &&
                    <Table columns={rows.Columns.map(c => c.Name)} columnTags={rows.Columns.map(c => c.Type)}
                           rows={rows.Rows} totalPages={totalPages} currentPage={currentPage + 1}
                           onPageChange={handlePageChange}
                           onRowUpdate={handleRowUpdate} disableEdit={rows.DisableUpdate}
                        checkedRows={checkedRows} setCheckedRows={setCheckedRows}
                        schema={schema} storageUnit={unitName} />
                }
            </div>
        </div>
        {unit && (
            <MockDataDialog
                isOpen={showMockDataDialog}
                onClose={() => setShowMockDataDialog(false)}
                storageUnit={unit}
                onSuccess={handleQuery}
            />
        )}
    </InternalPage>
}