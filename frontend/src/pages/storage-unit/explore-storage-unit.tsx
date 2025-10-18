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
    Badge,
    Button,
    cn,
    Drawer,
    DrawerContent,
    DrawerHeader,
    DrawerTitle,
    Input,
    Label,
    SearchInput,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Sheet,
    SheetContent,
    SheetFooter,
    SheetTitle,
    StackList,
    StackListItem,
    toast
} from "@clidey/ux";
import {
    DatabaseType,
    RecordInput,
    RowsResult,
    SortCondition,
    SortDirection,
    StorageUnit,
    useAddRowMutation,
    useColumnsLazyQuery,
    useGetStorageUnitRowsLazyQuery,
    useGetStorageUnitsLazyQuery,
    useRawExecuteLazyQuery,
    useUpdateStorageUnitMutation,
    WhereCondition,
    WhereConditionType
} from '@graphql';
import keys from "lodash/keys";
import { FC, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { CodeEditor } from "../../components/editor";
import { ErrorState } from "../../components/error-state";
import { CheckCircleIcon, CommandLineIcon, PlayIcon, PlusCircleIcon, TableCellsIcon, XMarkIcon } from "../../components/heroicons";
import { LoadingPage } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { SchemaViewer } from "../../components/schema-viewer";
import { getColumnIcons, StorageUnitTable } from "../../components/table";
import { Tip } from "../../components/tip";
import { BUILD_EDITION } from "../../config/edition";
import { InternalRoutes } from "../../config/routes";
import { useAppSelector } from "../../store/hooks";
import { databaseSupportsScratchpad, databaseTypesThatUseDatabaseInsteadOfSchema } from "../../utils/database-features";
import { getDatabaseOperators } from "../../utils/database-operators";
import { getDatabaseStorageUnitLabel, isNoSQL } from "../../utils/functions";
import { ExploreStorageUnitWhereCondition } from "./explore-storage-unit-where-condition";
import { ExploreStorageUnitWhereConditionSheet } from "./explore-storage-unit-where-condition-sheet";

// Conditionally import EE query utilities
let generateInitialQuery: ((databaseType: string | undefined, schema: string | undefined, tableName: string | undefined) => string) | undefined;

if (BUILD_EDITION === 'ee') {
    // Dynamically import EE query utilities when in EE mode
    import('@ee/pages/storage-unit/query-utils').then(module => {
        generateInitialQuery = module.generateInitialQuery;
    }).catch(() => {
        // EE module not available, use default
        generateInitialQuery = undefined;
    });
}

export const ExploreStorageUnit: FC<{ scratchpad?: boolean }> = ({ scratchpad }) => {

    const [bufferPageSize, setBufferPageSize] = useState("100");
    const [currentPage, setCurrentPage] = useState(1);
    const [whereCondition, setWhereCondition] = useState<WhereCondition>();
    const [sortConditions, setSortConditions] = useState<SortCondition[]>([]);
    const unit: StorageUnit = useLocation().state?.unit;

    let schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const whereConditionMode = useAppSelector(state => state.settings.whereConditionMode);
    const navigate = useNavigate();
    const [rows, setRows] = useState<RowsResult>();
    const [showAdd, setShowAdd] = useState(false);
    const searchRef = useRef<(search: string) => void>(() => {});
    const [search, setSearch] = useState("");
    const [currentTableName, setCurrentTableName] = useState<string>("");
    
    // For add row sheet logic
    const [addRowData, setAddRowData] = useState<Record<string, any>>({});
    const [addRowError, setAddRowError] = useState<string | null>(null);

    // Entity search sheet state
    const [showEntitySearchSheet, setShowEntitySearchSheet] = useState(false);
    const [entitySearchData, setEntitySearchData] = useState<{
        columnName: string;
        value: string;
        targetTable: string;
    } | null>(null);
    const [entitySearchResults, setEntitySearchResults] = useState<RowsResult | null>(null);

    const [updateStorageUnit, {loading: updating}] = useUpdateStorageUnitMutation();

    // For databases that don't have schemas (MongoDB, ClickHouse), pass the database name as the schema parameter
    // todo: is there a different way to do this? clickhouse doesn't have schemas as a table is considered a schema. people mainly switch between DB
    if (databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type) && current?.Database) {
        schema = current.Database
    }

    const [getStorageUnitRows, { loading }] = useGetStorageUnitRowsLazyQuery({
        onCompleted(data) {
            setRows(data.Row);
        },
        fetchPolicy: "no-cache",
    });
    const [getStorageUnits] = useGetStorageUnitsLazyQuery();
    const [getColumns] = useColumnsLazyQuery();
    const [addRow, { loading: adding }] = useAddRowMutation();
    const [rawExecute, { data: rawExecuteData }] = useRawExecuteLazyQuery();

    const unitName = useMemo(() => {
        return unit?.Name;
    }, [unit]);

    const initialScratchpadQuery = useMemo(() => {
        if (generateInitialQuery && current?.Type) {
            return generateInitialQuery(current?.Type, schema, unitName);
        }
        const qualified = schema ? `${schema}.${unitName}` : unitName;
        return `SELECT * FROM ${qualified} LIMIT 5`;
    }, [schema, unitName, current?.Type, generateInitialQuery]);

    const [code, setCode] = useState(initialScratchpadQuery);

    const handleSubmitRequest = useCallback((pageOffset: number | null = null) => {
        const tableNameToUse = unitName || currentTableName;
        if (tableNameToUse) {
            setCurrentTableName(tableNameToUse);
        }
        getStorageUnitRows({
            variables: {
                schema,
                storageUnit: tableNameToUse,
                where: whereCondition,
                sort: sortConditions.length > 0 ? sortConditions : undefined,
                pageSize: Number.parseInt(bufferPageSize),
                pageOffset: pageOffset ?? currentPage - 1,
            },
        });
    }, [getStorageUnitRows, schema, unitName, currentTableName, whereCondition, sortConditions, bufferPageSize, currentPage]);

    const handleQuery = useCallback(() => {
        handleSubmitRequest();
        setCurrentPage(1);
    }, [handleSubmitRequest]);

    const handlePageChange = useCallback((page: number) => {
        setCurrentPage(page);
        handleSubmitRequest(page - 1);
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
        setCode(initialScratchpadQuery);
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
    
    const {columns, columnTypes, columnIsPrimary, columnIsForeignKey} = useMemo(() => {
        const dataColumns = rows?.Columns.map(c => c.Name) ?? [];
        return {
            columns: dataColumns,
            columnTypes: rows?.Columns.map(column => column.Type),
            columnIsPrimary: rows?.Columns.map(column => column.IsPrimary),
            columnIsForeignKey: rows?.Columns.map(column => column.IsForeignKey)
        };
    }, [rows?.Columns, rows?.Rows]);

    useEffect(() => {
        if (unit == null && !currentTableName) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
    }, [navigate, unit, currentTableName]);

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
        if (rows?.Columns == null) return;
        let values: RecordInput[] = [];
        if (isNoSQL(current?.Type as DatabaseType) && rows.Columns.length === 1 && rows.Columns[0].Type === "Document") {
            try {
                const json = JSON.parse(addRowData.document);
                for (const key of keys(json)) {
                    values.push({
                        Key: key,
                        Value: json[key],
                    });
                }
            } catch (e) {
                setAddRowError("Invalid JSON.");
                return;
            }
        } else {
            for (const col of rows.Columns) {
                if (addRowData[col.Name] !== undefined && addRowData[col.Name] !== "") {
                    values.push({
                        Key: col.Name,
                        Value: addRowData[col.Name],
                    });
                }
            }
        }

        if (values.length === 0) {
            setAddRowError("Please fill at least one value.");
            return;
        }
        addRow({
            variables: {
                schema,
                storageUnit: unit?.Name || unitName || currentTableName || "",
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
    }, [addRow, addRowData, handleSubmitRequest, rows?.Columns, schema, unit?.Name, current?.Type]);

    const handleScratchpad = useCallback((specificCode?: string) => {
        if (current == null) {
            return;
        }
        rawExecute({
            variables: {
                query: specificCode ?? code,
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
        setCode(initialScratchpadQuery);
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

    const {whereColumns, whereColumnTypes} = useMemo(() => {
        if (rows?.Columns == null || rows?.Columns.length === 0 || rows == null || rows.Rows.length === 0) {
            return {whereColumns: [], whereColumnTypes: []};
        }
        if (rows?.Columns.length === 1 && rows?.Columns[0].Type === "Document" && isNoSQL(current?.Type as DatabaseType)) {
            const whereColumns = keys(JSON.parse(rows?.Rows[0][0]));
            const whereColumnTypes = whereColumns.map(() => "string");
            return {whereColumns, whereColumnTypes}
        }
        return {whereColumns: columns, whereColumnTypes: columnTypes}
    }, [rows?.Columns, rows?.Rows, current?.Type])

    // Foreign key detection logic
    const [allTableNames, setAllTableNames] = useState<Set<string>>(new Set());
    
    const isValidForeignKey = useCallback((columnName: string) => {
        // Check for both singular and plural table names
        if (columnName.endsWith("_id")) {
            const base = columnName.slice(0, -3);
            return allTableNames.has(base) || allTableNames.has(base + "s");
        }
        return false;
    }, [allTableNames]);

    const getTargetTableName = useCallback((columnName: string) => {
        if (columnName.endsWith("_id")) {
            const base = columnName.slice(0, -3);
            if (allTableNames.has(base)) {
                return base;
            } else if (allTableNames.has(base + "s")) {
                return base + "s";
            }
        }
        return null;
    }, [allTableNames]);

    // Load all table names for foreign key detection
    useEffect(() => {
        if (schema) {
            getStorageUnits({
                variables: { schema },
                onCompleted: (data) => {
                    const tableNames = new Set(data.StorageUnit?.map(unit => unit.Name) || []);
                    setAllTableNames(tableNames);
                }
            });
        }
    }, [schema, getStorageUnits]);

    // Get primary key column name from rows
    const getPrimaryKeyColumn = useCallback(() => {
        if (!rows?.Columns) {
            return null;
        }
        const primaryColumn = rows.Columns.find(col => col.IsPrimary);
        return primaryColumn?.Name || null;
    }, [rows?.Columns]);

    // Entity search functionality
    const handleEntitySearch = useCallback((columnName: string, value: string) => {
        const targetTable = getTargetTableName(columnName);
        if (!targetTable) {
            toast.error("Could not determine target table for foreign key");
            return;
        }

        setEntitySearchData({
            columnName,
            value,
            targetTable
        });

        // First, fetch the target table's column metadata to find its primary key
        getColumns({
            variables: {
                schema,
                storageUnit: targetTable
            },
            onCompleted: (columnsData) => {
                // Find the primary key column in the target table
                const targetPrimaryKey = columnsData.Columns?.find(col => col.IsPrimary);

                if (!targetPrimaryKey) {
                    toast.error(`No primary key found for table ${targetTable}`);
                    return;
                }

                const primaryKeyName = targetPrimaryKey.Name;

                // Now search for the entity using the correct primary key
                getStorageUnitRows({
                    variables: {
                        schema,
                        storageUnit: targetTable,
                        where: {
                            Type: WhereConditionType.Atomic,
                            Atomic: {
                                Key: primaryKeyName,
                                Operator: "=",
                                Value: value,
                                ColumnType: "string"
                            }
                        },
                        pageSize: 1,
                        pageOffset: 0
                    },
                    onCompleted: (data) => {
                        setEntitySearchResults(data.Row);
                        setShowEntitySearchSheet(true);
                    },
                    onError: (error) => {
                        toast.error(`Failed to search for entity: ${error.message}`);
                    }
                });
            },
            onError: (error) => {
                toast.error(`Failed to get target table structure: ${error.message}`);
            }
        });
    }, [getColumns, getStorageUnitRows, getTargetTableName, schema]);

    const handleCloseEntitySearchSheet = useCallback(() => {
        setShowEntitySearchSheet(false);
        setEntitySearchData(null);
        setEntitySearchResults(null);
    }, []);

    if (unit == null) {
        return <Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />
    }

    if (loading) {
        return <InternalPage routes={routes}>
            <LoadingPage />
        </InternalPage>
    }

    // Prevent rendering if unit is not available and we don't have a table name
    if (!unit && !currentTableName) {
        return <InternalPage routes={routes}>
            <LoadingPage/>
        </InternalPage>
    }

    return <InternalPage routes={routes} className="relative" sidebar={<SchemaViewer />}>
        <div className="flex flex-col grow gap-lg h-[calc(100%-100px)]">
            <div className="flex items-center justify-between">
                <div className="flex gap-sm items-center">
                    <h1 className="text-xl font-bold mr-4">{unitName}</h1>
                </div>
                <div className="text-sm"><span className="font-semibold">Total Count:</span> {totalCount}</div>
            </div>
            <div className="flex w-full relative" data-testid="explore-storage-unit-options">
                <div className="flex justify-between items-end w-full">
                    <div className="flex gap-2">
                        <div className="flex flex-col gap-2">
                            <Label>Search</Label>
                            <SearchInput placeholder="Search query..." className="w-64" value={search} onChange={e => setSearch(e.target.value)}
                                onKeyDown={e => {
                                    if (e.key === "Enter") {
                                        searchRef.current?.(search);
                                    }
                                }}
                                         data-testid="table-search"
                            />
                        </div>
                        <div className="flex flex-col gap-2">
                            <Label>Page Size</Label>
                            <Select value={bufferPageSize} onValueChange={setBufferPageSize}>
                                <SelectTrigger className="w-32" data-testid="table-page-size">
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    {import.meta.env.VITE_E2E_TEST === "true" &&
                                        <SelectItem value="1" data-value="1">1</SelectItem>
                                    }
                                    {import.meta.env.VITE_E2E_TEST === "true" &&
                                        <SelectItem value="2" data-value="2">2</SelectItem>
                                    }
                                    <SelectItem value="10" data-value="10">10</SelectItem>
                                    <SelectItem value="25" data-value="25">25</SelectItem>
                                    <SelectItem value="50" data-value="50">50</SelectItem>
                                    <SelectItem value="100" data-value="100">100</SelectItem>
                                    <SelectItem value="250" data-value="250">250</SelectItem>
                                    <SelectItem value="500" data-value="500">500</SelectItem>
                                    <SelectItem value="1000" data-value="1000">1000</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        {current?.Type !== DatabaseType.Redis && (
                            whereConditionMode === 'sheet' ? (
                                <ExploreStorageUnitWhereConditionSheet 
                                    defaultWhere={whereCondition} 
                                    columns={whereColumns}
                                    operators={validOperators} 
                                    onChange={handleFilterChange}
                                    columnTypes={whereColumnTypes ?? []}
                                />
                            ) : (
                                <ExploreStorageUnitWhereCondition 
                                    defaultWhere={whereCondition} 
                                    columns={whereColumns}
                                    operators={validOperators} 
                                    onChange={handleFilterChange}
                                    columnTypes={whereColumnTypes ?? []}
                                />
                            )
                        )}
                        <Button className="ml-6 mt-[22px]" onClick={handleQuery} data-testid="submit-button">
                            <CheckCircleIcon className="w-4 h-4" /> Query
                        </Button>
                    </div>
                    <Button onClick={handleOpenScratchpad} data-testid="scratchpad-button" variant="secondary"
                        className={cn({
                            "hidden": !databaseSupportsScratchpad(current?.Type),
                        })}>
                        <CommandLineIcon className="w-4 h-4" /> Scratchpad
                    </Button>
                </div>
                <Sheet open={showAdd} onOpenChange={setShowAdd}>
                    <SheetContent side="right" className="flex flex-col p-8">
                        <SheetTitle className="flex items-center gap-2"><TableCellsIcon className="w-5 h-5" /> Add new row</SheetTitle>
                        <div className="flex-1 overflow-y-auto pr-2">
                            <div className="flex flex-col gap-4">
                                {rows?.Columns?.map((col, index) => (
                                    <div key={col.Name} className="flex flex-col gap-2"
                                         data-testid={`add-row-field-${col.Name}`}>
                                        <Tip>
                                            <div className="flex items-center gap-xs">
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
                                <ErrorState error={addRowError} />
                            )}
                        </div>
                        <SheetFooter className="flex flex-row gap-sm px-0 pt-4 border-t">
                            <Button
                                className="flex-1"
                                variant="secondary"
                                onClick={() => setShowAdd(false)}
                                data-testid="cancel-add-row"
                            >
                                Cancel
                            </Button>
                            <Button className="flex-1" onClick={handleAddRowSubmit} data-testid="submit-add-row-button" disabled={adding}>
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
                        columnIsPrimary={columnIsPrimary}
                        columnIsForeignKey={columnIsForeignKey}
                        schema={schema}
                        storageUnit={unitName}
                        onRefresh={handleSubmitRequest}
                        onColumnSort={handleColumnSort}
                        sortedColumns={sortedColumnsMap}
                        searchRef={searchRef}
                        pageSize={Number.parseInt(bufferPageSize)}
                        // Server-side pagination props
                        totalCount={Number.parseInt(totalCount)}
                        currentPage={currentPage}
                        onPageChange={handlePageChange}
                        showPagination={true}
                        // Foreign key functionality
                        isValidForeignKey={isValidForeignKey}
                        onEntitySearch={handleEntitySearch}
                    >
                        <div className="flex gap-2">
                            <Button onClick={handleOpenAddSheet} disabled={adding} data-testid="add-row-button">
                                <PlusCircleIcon className="w-4 h-4" /> Add Row
                            </Button>
                        </div>
                    </StorageUnitTable>
                }
            </div>
        </div>
        <Drawer open={scratchpad} onOpenChange={handleCloseScratchpad}>
            <DrawerContent className="px-8 min-h-[65vh]">
                <Button variant="ghost" className="absolute top-0 right-0" onClick={handleCloseScratchpad} data-testid="icon-button">
                    <XMarkIcon className="w-4 h-4" />
                </Button>
                <DrawerHeader className="px-0">
                    <DrawerTitle className="flex justify-between items-center">
                        <h2 className="text-lg font-semibold">Scratchpad</h2>
                        <div className="flex gap-sm items-center">
                            <Button onClick={() => handleScratchpad()} data-testid="run-submit-button">
                                <PlayIcon className="w-4 h-4" />
                                Run
                            </Button>
                        </div>
                    </DrawerTitle>
                </DrawerHeader>
                <div className="flex flex-col gap-sm h-[150px] mb-4">
                    <CodeEditor language="sql" value={code} setValue={setCode} onRun={handleScratchpad} />
                </div>
                <StorageUnitTable
                    columns={rawExecuteData?.RawExecute.Columns.map(c => c.Name) ?? []}
                    columnTypes={rawExecuteData?.RawExecute.Columns.map(c => c.Type) ?? []}
                    rows={rawExecuteData?.RawExecute.Rows ?? []}
                    disableEdit={true}
                    schema={schema}
                    storageUnit={unitName}
                    onRefresh={handleSubmitRequest}
                    showPagination={false}
                />
            </DrawerContent>
        </Drawer>
        <Sheet open={showEntitySearchSheet} onOpenChange={setShowEntitySearchSheet}>
            <SheetContent side="right" className="flex flex-col p-8">
                <div className="text-lg font-semibold mb-4">
                    Entity Details
                    {entitySearchData && (
                        <div className="text-sm text-gray-500 mt-1">
                            {entitySearchData.targetTable} (ID: {entitySearchData.value})
                        </div>
                    )}
                </div>
                <div className="flex-1 overflow-y-auto pr-2">
                    {entitySearchResults && entitySearchResults.Rows.length > 0 ? (
                        <div className="flex flex-col gap-4">
                            <StackList>
                                {entitySearchResults.Columns.map((column, index) => (
                                    <StackListItem 
                                        key={column.Name} 
                                        item={
                                            isValidForeignKey(column.Name) ? (
                                                <Badge className="text-lg" data-testid="foreign-key-attribute">
                                                    {column.Name}
                                                </Badge>
                                            ) : column.Name
                                        }
                                    >
                                        {entitySearchResults.Rows[0][index]}
                                    </StackListItem>
                                ))}
                            </StackList>
                        </div>
                    ) : (
                        <div className="text-center text-gray-500 py-8">
                            No entity found with the specified ID
                        </div>
                    )}
                </div>
                <SheetFooter className="px-0 pt-4 border-t">
                    <Button onClick={handleCloseEntitySearchSheet} variant="outline">
                        Close
                    </Button>
                </SheetFooter>
            </SheetContent>
        </Sheet>
    </InternalPage>
}