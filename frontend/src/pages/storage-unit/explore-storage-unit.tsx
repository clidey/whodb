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

import { Button, Drawer, DrawerContent, DrawerFooter, DrawerHeader, DrawerTitle, Input, Label, SearchInput, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Sheet, SheetContent, SheetFooter, toast } from "@clidey/ux";
import {
    DatabaseType,
    RecordInput,
    RowsResult,
    SortCondition,
    SortDirection,
    StorageUnit,
    useAddRowMutation,
    useGetStorageUnitRowsLazyQuery,
    useRawExecuteLazyQuery,
    useUpdateStorageUnitMutation,
    WhereCondition
} from '@graphql';
import { CheckCircleIcon, CommandLineIcon, PlayIcon, PlusCircleIcon, XMarkIcon } from "@heroicons/react/24/outline";
import { keys } from "lodash";
import { FC, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { CodeEditor } from "../../components/editor";
import { LoadingPage } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { getColumnIcons, StorageUnitTable } from "../../components/table";
import { Tip } from "../../components/tip";
import { InternalRoutes } from "../../config/routes";
import { useAppSelector } from "../../store/hooks";
import { getDatabaseOperators } from "../../utils/database-operators";
import { getDatabaseStorageUnitLabel } from "../../utils/functions";
import { ExploreStorageUnitWhereCondition } from "./explore-storage-unit-where-condition";


export const ExploreStorageUnit: FC<{ scratchpad?: boolean }> = ({ scratchpad }) => {
    const [bufferPageSize, setBufferPageSize] = useState("100");
    const [currentPage, setCurrentPage] = useState(0);
    const [whereCondition, setWhereCondition] = useState<WhereCondition>();
    const [sortConditions, setSortConditions] = useState<SortCondition[]>([]);
    const [pageSize, setPageSize] = useState("");
    const unit: StorageUnit = useLocation().state?.unit;

    let schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const navigate = useNavigate();
    const [rows, setRows] = useState<RowsResult>();
    const [showAdd, setShowAdd] = useState(false);
    const searchRef = useRef<(search: string) => void>(() => {});
    const [search, setSearch] = useState("");
    
    // For add row sheet logic
    const [addRowData, setAddRowData] = useState<Record<string, any>>({});
    const [addRowError, setAddRowError] = useState<string | null>(null);

    const [updateStorageUnit, { loading: updating }] = useUpdateStorageUnitMutation();
    
    // For scratchpad sheet logic
    // todo: is there a different way to do this? clickhouse doesn't have schemas as a table is considered a schema. people mainly switch between DB
    if (current?.Type === DatabaseType.ClickHouse) {
        schema = current.Database
    }

    const [code, setCode] = useState(`SELECT * FROM ${schema}.${unit?.Name}`);

    const [getStorageUnitRows, { loading }] = useGetStorageUnitRowsLazyQuery({
        onCompleted(data) {
            setRows(data.Row);
        },
        fetchPolicy: "no-cache",
    });
    const [addRow, { loading: adding }] = useAddRowMutation();
    const [rawExecute, { data: rawExecuteData }] = useRawExecuteLazyQuery();

    const unitName = useMemo(() => {
        return unit?.Name;
    }, [unit]);

    const handleSubmitRequest = useCallback(() => {
        getStorageUnitRows({
            variables: {
                schema,
                storageUnit: unitName,
                where: whereCondition,
                sort: sortConditions.length > 0 ? sortConditions : undefined,
                pageSize: Number.parseInt(bufferPageSize),
                pageOffset: currentPage,
            },
        });
    }, [getStorageUnitRows, schema, unitName, whereCondition, sortConditions, bufferPageSize, currentPage]);

    const handleQuery = useCallback(() => {
        handleSubmitRequest();
        setCurrentPage(0);
    }, [handleSubmitRequest]);

    const handleColumnSort = useCallback((columnName: string) => {
        setSortConditions(prev => {
            const existingSort = prev.find(s => s.Column === columnName);
            
            if (!existingSort) {
                // Add ascending sort for this column
                return [...prev, { Column: columnName, Direction: SortDirection.Asc }];
            } else if (existingSort.Direction === SortDirection.Asc) {
                // Change to descending
                return prev.map(s => 
                    s.Column === columnName 
                        ? { ...s, Direction: SortDirection.Desc }
                        : s
                );
            } else {
                // Remove sort for this column
                return prev.filter(s => s.Column !== columnName);
            }
        });
    }, []);

    const handleRowUpdate = useCallback(
        (
            row: Record<string, string | number>,
            originalRow?: Record<string, string | number>
        ) => {
            if (!current) {
                return Promise.resolve();
            }
    
            return new Promise<void>((resolve, reject) => {
                // Figure out which columns to update
                const changedColumns = originalRow
                    ? Object.keys(row).filter((col) => row[col] !== originalRow[col])
                    : Object.keys(row);
        
                if (changedColumns.length === 0) {
                    // Nothing changed, skip
                    return Promise.resolve();
                }
        
                // Build values for all columns
                const allColumns = Object.keys(row);
                const values = allColumns.map((col) => ({
                    Key: col,
                    Value: row[col].toString(),
                }));
        
                updateStorageUnit({
                    variables: {
                        schema,
                        storageUnit: unitName,
                        values,
                        updatedColumns: changedColumns,
                    },
                    onCompleted: (data) => {
                        if (!data?.UpdateStorageUnit.Status) {
                            return reject(new Error("Update failed"));
                        }
                        return resolve();
                    },
                    onError: (error) => {
                        return reject(error);
                    },
                });
            });
        },
        [current, schema, unitName]
    );

    const totalCount = useMemo(() => {
        const count = unit?.Attributes.find(attribute => attribute.Key === "Count")?.Value;
        if (count == null || count === "0" || count === "unknown") {
            return rows?.Rows.length?.toString() ?? "unknown";
        }
        return count;
    }, [unit, rows?.Rows.length]);

    useEffect(() => {
        handleSubmitRequest();
        setCode(`SELECT * FROM ${schema}.${unit?.Name}`);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [unit]);

    useEffect(() => {
        if (sortConditions.length > 0) {
            handleSubmitRequest();
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [sortConditions]);

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

    const sortedColumnsMap = useMemo(() => {
        const map = new Map<string, 'asc' | 'desc'>();
        sortConditions.forEach(cond => {
            map.set(cond.Column, cond.Direction === SortDirection.Asc ? 'asc' : 'desc');
        });
        return map;
    }, [sortConditions]);

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

    const columnIcons = useMemo(() => getColumnIcons(columns, columnTypes), [columns, columnTypes]);

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
                <div className="text-sm"><span className="font-semibold">Total Count:</span> {totalCount}</div>
            </div>
            <div className="flex w-full relative">
                <div className="flex justify-between items-end w-full">
                    <div className="flex gap-2">
                        <div className="flex flex-col gap-2">
                            <Label>Search</Label>
                            <SearchInput placeholder="Enter search query" className="w-64" value={search} onChange={e => setSearch(e.target.value)}
                                onKeyDown={e => {
                                    if (e.key === "Enter") {
                                        searchRef.current?.(search);
                                    }
                                }}
                            />
                        </div>
                        <div className="flex flex-col gap-2">
                            <Label>Page Size</Label>
                            <Select value={bufferPageSize} onValueChange={setBufferPageSize}>
                                <SelectTrigger className="w-32" data-testid="table-page-size">
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="10">10</SelectItem>
                                    <SelectItem value="25">25</SelectItem>
                                    <SelectItem value="50">50</SelectItem>
                                    <SelectItem value="100">100</SelectItem>
                                    <SelectItem value="250">250</SelectItem>
                                    <SelectItem value="500">500</SelectItem>
                                    <SelectItem value="1000">1000</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        { current?.Type !== DatabaseType.Redis && <ExploreStorageUnitWhereCondition defaultWhere={whereCondition} columns={columns} operators={validOperators} onChange={handleFilterChange} columnTypes={columnTypes ?? []} /> }
                        <Button className="ml-6 self-end" onClick={handleQuery} data-testid="submit-button">
                            <CheckCircleIcon className="w-4 h-4" /> Query
                        </Button>
                    </div>
                </div>
                <Sheet open={showAdd} onOpenChange={setShowAdd}>
                    <SheetContent side="right" className="p-8">
                        <div className="flex flex-col gap-4 h-full">
                            <div className="text-lg font-semibold mb-2">Add new row</div>
                            <div className="flex flex-col gap-4">
                                {rows?.Columns?.map((col, index) => (
                                    <div key={col.Name} className="flex flex-col gap-2">
                                        <Tip>
                                            <div className="flex items-center gap-1">
                                                {columnIcons[index]}
                                                <Label className="w-fit">
                                                    {col.Name}
                                                </Label> 
                                            </div>
                                            <p className="text-xs">{col.Type}</p>
                                        </Tip>
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
                                <CheckCircleIcon className="w-4 h-4" /> Submit
                            </Button>
                        </SheetFooter>
                    </SheetContent>
                </Sheet>

            </div>
            <div className="grow">
                {
                    rows != null &&
                    <StorageUnitTable
                        columns={columns} 
                        rows={rows.Rows} 
                        onRowUpdate={handleRowUpdate} 
                        columnTypes={columnTypes}
                        schema={schema}
                        storageUnit={unitName}
                        onRefresh={handleSubmitRequest}
                        onColumnSort={handleColumnSort}
                        sortedColumns={sortedColumnsMap}
                        searchRef={searchRef}
                    >
                        <div className="flex gap-2">
                            <Button onClick={handleOpenScratchpad} data-testid="scratchpad-button" variant="secondary">
                                <CommandLineIcon className="w-4 h-4" /> Scratchpad
                            </Button>
                            <Button onClick={handleOpenAddSheet} disabled={adding} data-testid="add-button">
                                <PlusCircleIcon className="w-4 h-4" /> Add Row
                            </Button>
                        </div>
                    </StorageUnitTable>
                }
            </div>
        </div>
        <Drawer open={scratchpad} onOpenChange={handleCloseScratchpad}>
            <DrawerContent className="px-8 max-h-[25vh]">
                <Button variant="ghost" className="absolute top-0 right-0" onClick={handleCloseScratchpad}>
                    <XMarkIcon className="w-4 h-4" />
                </Button>
                <DrawerHeader className="px-0">
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
                        schema={schema}
                        storageUnit={unitName}
                        onRefresh={handleSubmitRequest}
                    />
                </DrawerFooter>
            </DrawerContent>
        </Drawer>
    </InternalPage>
}