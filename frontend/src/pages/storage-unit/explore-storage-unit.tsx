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
    toast,
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
import {FC, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {Navigate, useLocation, useNavigate} from "react-router-dom";
import {CodeEditor} from "../../components/editor";
import {ErrorState} from "../../components/error-state";
import {
    CheckCircleIcon,
    CommandLineIcon,
    MagnifyingGlassIcon,
    PlayIcon,
    PlusCircleIcon,
    TableCellsIcon,
    XMarkIcon
} from "../../components/heroicons";
import {LoadingPage} from "../../components/loading";
import {InternalPage} from "../../components/page";
import {SchemaViewer} from "../../components/schema-viewer";
import {getColumnIcons, StorageUnitTable} from "../../components/table";
import {Tip} from "../../components/tip";
import {BUILD_EDITION} from "../../config/edition";
import {InternalRoutes} from "../../config/routes";
import {useAppSelector} from "../../store/hooks";
import {databaseSupportsScratchpad, databaseTypesThatUseDatabaseInsteadOfSchema} from "../../utils/database-features";
import {getDatabaseOperators} from "../../utils/database-operators";
import {getDatabaseStorageUnitLabel, isNoSQL} from "../../utils/functions";
import {usePageSize} from "../../hooks/use-page-size";
import {ExploreStorageUnitWhereCondition} from "./explore-storage-unit-where-condition";
import {ExploreStorageUnitWhereConditionSheet} from "./explore-storage-unit-where-condition-sheet";
import {useTranslation} from "../../hooks/use-translation";

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
    const defaultPageSize = useAppSelector(state => state.settings.defaultPageSize);
    const {
        pageSize,
        pageSizeString,
        isCustom: isCustomPageSize,
        customInput: customPageSizeInput,
        setCustomInput: setCustomPageSizeInput,
        handleSelectChange: handlePageSizeChange,
        handleCustomApply: handleCustomPageSizeApply,
    } = usePageSize(defaultPageSize);
    const { t } = useTranslation('pages/explore-storage-unit');
    const { t: tTable } = useTranslation('components/table');

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

    // Request counter to prevent race conditions - only the latest query's results should be used
    const latestRequestIdRef = useRef(0);
    // Ref to always have the latest whereCondition (avoids stale closure issues)
    const whereConditionRef = useRef<WhereCondition | undefined>(whereCondition);
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

    // Keep whereConditionRef in sync with whereCondition state
    useEffect(() => {
        whereConditionRef.current = whereCondition;
    }, [whereCondition]);

    // TODO: ClickHouse/MongoDB use database name as schema parameter since they lack traditional schemas
    if (databaseTypesThatUseDatabaseInsteadOfSchema(current?.Type) && current?.Database) {
        schema = current.Database
    }

    const [getStorageUnitRows, { loading }] = useGetStorageUnitRowsLazyQuery({
        fetchPolicy: "no-cache",
    });
    const [getStorageUnits] = useGetStorageUnitsLazyQuery();
    const [getColumns] = useColumnsLazyQuery({
        fetchPolicy: "network-only",
    });
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
        // Increment request counter and capture this request's ID to prevent race conditions
        latestRequestIdRef.current += 1;
        const thisRequestId = latestRequestIdRef.current;
        // Use ref to always get the latest whereCondition (avoids stale closure issues)
        const currentWhereCondition = whereConditionRef.current;

        getStorageUnitRows({
            variables: {
                schema,
                storageUnit: tableNameToUse,
                where: currentWhereCondition,
                sort: sortConditions.length > 0 ? sortConditions : undefined,
                pageSize,
                pageOffset: pageOffset ?? currentPage - 1,
            },
        }).then(result => {
            const isLatest = thisRequestId === latestRequestIdRef.current;
            if (isLatest && result.data) {
                setRows(result.data.Row);
            }
        });
    }, [getStorageUnitRows, schema, unitName, currentTableName, sortConditions, pageSize, currentPage]);

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
                return [...prev, { Column: columnName, Direction: SortDirection.Asc }];
            } else if (existingSort.Direction === SortDirection.Asc) {
                return prev.map(s =>
                    s.Column === columnName
                        ? { ...s, Direction: SortDirection.Desc }
                        : s
                );
            } else {
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
        if (rows?.TotalCount != null && rows.TotalCount > 0) {
            return rows.TotalCount.toString();
        }
        const count = unit?.Attributes.find(attribute => attribute.Key === "Count")?.Value;
        if (count != null && count !== "0" && count !== "unknown") {
            return count;
        }
        return rows?.Rows.length?.toString() ?? "unknown";
    }, [unit, rows?.TotalCount, rows?.Rows.length]);

    useEffect(() => {
        // Reset all state when switching to a new table to prevent cached data
        setCurrentPage(1);
        setWhereCondition(undefined);
        whereConditionRef.current = undefined;
        setSortConditions([]);
        setSearch("");
        setRows(undefined);
        setShowAdd(false);
        setAddRowData({});
        setAddRowError(null);
        setShowEntitySearchSheet(false);
        setEntitySearchData(null);
        setEntitySearchResults(null);

        // Fetch fresh data for the new table
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

    // Broadcast available columns for Command Palette sorting
    useEffect(() => {
        window.dispatchEvent(new CustomEvent('table:columns-available', {
            detail: { columns }
        }));
        return () => {
            window.dispatchEvent(new CustomEvent('table:columns-available', {
                detail: { columns: [] }
            }));
        };
    }, [columns]);

    // Listen for sort requests from Command Palette
    useEffect(() => {
        const handleSortColumn = (event: CustomEvent<{ column: string }>) => {
            handleColumnSort(event.detail.column);
        };
        window.addEventListener('table:sort-column', handleSortColumn as EventListener);
        return () => {
            window.removeEventListener('table:sort-column', handleSortColumn as EventListener);
        };
    }, [handleColumnSort]);

    useEffect(() => {
        if (unit == null && !currentTableName) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
    }, [navigate, unit, currentTableName]);

    const handleFilterChange = useCallback((filters: WhereCondition) => {
        // Update ref synchronously to avoid stale closure issues
        whereConditionRef.current = filters;
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
        // Prepare empty values for empty addRowData values
        let initialData: Record<string, any> = {};
        if (rows?.Columns) {
            // todo: add support for different functions for defaults like now(), gen_random_uuid(), etc
            for (const col of rows.Columns) {
                initialData[col.Name] = "";
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
                setAddRowError(t('invalidJson'));
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
            setAddRowError(t('fillAtLeastOneValue'));
            return;
        }
        addRow({
            variables: {
                schema,
                storageUnit: unit?.Name || unitName || currentTableName || "",
                values,
            },
            onCompleted() {
                toast.success(t('addRowSuccess'));
                setShowAdd(false);
                setTimeout(() => {
                    handleSubmitRequest();
                }, 500);
            },
            onError(e) {
                setAddRowError(e.message);
                const errorMessage = t('addRowError').replace('{error}', e.message);
                toast.error(errorMessage);
            },
        });
    }, [addRow, addRowData, handleSubmitRequest, rows?.Columns, schema, t, unit?.Name, current?.Type]);

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

    const columnIcons = useMemo(() => getColumnIcons(columns, columnTypes, tTable), [columns, columnTypes, tTable]);

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

    // Foreign key detection using actual column metadata
    const getColumnByName = useCallback((columnName: string) => {
        return rows?.Columns?.find(col => col.Name === columnName);
    }, [rows?.Columns]);

    const isValidForeignKey = useCallback((columnName: string) => {
        const column = getColumnByName(columnName);
        return Boolean(column?.IsForeignKey && column?.ReferencedTable != null);
    }, [getColumnByName]);

    const getTargetTableName = useCallback((columnName: string) => {
        const column = getColumnByName(columnName);
        return column?.ReferencedTable || null;
    }, [getColumnByName]);

    // Entity search functionality
    const handleEntitySearch = useCallback((columnName: string, value: string) => {
        const targetTable = getTargetTableName(columnName);
        if (!targetTable) {
            toast.error(t('couldNotDetermineTargetTable'));
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
                    const errorMessage = t('noPrimaryKeyFound').replace('{table}', targetTable);
                    toast.error(errorMessage);
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
                        const errorMessage = t('failedToSearchEntity').replace('{error}', error.message);
                        toast.error(errorMessage);
                    }
                });
            },
            onError: (error) => {
                const errorMessage = t('failedToGetTargetTableStructure').replace('{error}', error.message);
                toast.error(errorMessage);
            }
        });
    }, [getColumns, getStorageUnitRows, getTargetTableName, schema, t]);

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
                <div className="text-sm" data-testid="total-count-top"><span className="font-semibold">{t('totalCount')}</span> {totalCount}</div>
            </div>
            <div className="flex w-full relative" data-testid="explore-storage-unit-options">
                <div className="flex justify-between items-end w-full">
                    <div className="flex gap-2">
                        <div className="flex flex-col gap-2">
                            <Label>{t('searchLabel')}</Label>
                            <SearchInput placeholder={t('searchPlaceholder')} className="w-64" value={search} onChange={e => setSearch(e.target.value)}
                                onKeyDown={e => {
                                    if (e.key === "Enter") {
                                        searchRef.current?.(search);
                                    }
                                }}
                                         data-testid="table-search"
                            />
                        </div>
                        <div className="flex flex-col gap-2">
                            <Label>{t('pageSizeLabel')}</Label>
                            <div className="flex gap-2">
                                <Select
                                    value={isCustomPageSize ? "custom" : pageSizeString}
                                    onValueChange={handlePageSizeChange}
                                >
                                    <SelectTrigger className="w-32" data-testid="table-page-size">
                                        <SelectValue/>
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
                                        <SelectItem value="custom" data-value="custom">Custom</SelectItem>
                                    </SelectContent>
                                </Select>
                                {isCustomPageSize && (
                                    <Input
                                        type="number"
                                        min={1}
                                        className="w-24"
                                        value={customPageSizeInput}
                                        onChange={(e) => setCustomPageSizeInput(e.target.value)}
                                        onBlur={handleCustomPageSizeApply}
                                        onKeyDown={(e) => {
                                            if (e.key === "Enter") {
                                                handleCustomPageSizeApply();
                                            }
                                        }}
                                        data-testid="table-page-size-custom"
                                    />
                                )}
                            </div>
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
                            <CheckCircleIcon className="w-4 h-4" /> {t('queryButton')}
                        </Button>
                    </div>
                    <Button onClick={handleOpenScratchpad} data-testid="embedded-scratchpad-button" variant="secondary"
                        className={cn({
                            "hidden": !databaseSupportsScratchpad(current?.Type),
                        })}>
                        <CommandLineIcon className="w-4 h-4" /> {t('scratchpad')}
                    </Button>
                </div>
                <Sheet open={showAdd} onOpenChange={setShowAdd}>
                    <SheetContent
                        side="right"
                        className="flex flex-col p-8"
                        onKeyDown={(e) => {
                            if (e.key === 'Escape') {
                                e.preventDefault();
                                e.stopPropagation();
                                setShowAdd(false);
                            }
                        }}
                    >
                        <SheetTitle className="flex items-center gap-2"><TableCellsIcon className="w-5 h-5" /> {t('addRowTitle')}</SheetTitle>
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
                                            <p className="text-xs">{col.Type?.toLowerCase()}</p>
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
                                {t('cancel')}
                            </Button>
                            <Button className="flex-1" onClick={handleAddRowSubmit} data-testid="submit-add-row-button" disabled={adding}>
                                <CheckCircleIcon className="w-4 h-4" /> {t('submit')}
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
                        pageSize={pageSize}
                        // Server-side pagination props
                        totalCount={Number.parseInt(totalCount, 10)}
                        currentPage={currentPage}
                        onPageChange={handlePageChange}
                        showPagination={true}
                        // Foreign key functionality
                        isValidForeignKey={isValidForeignKey}
                        onEntitySearch={handleEntitySearch}
                        databaseType={current?.Type}
                    >
                        <div className="flex gap-2">
                            <Button onClick={handleOpenAddSheet} disabled={adding} data-testid="add-row-button">
                                <PlusCircleIcon className="w-4 h-4" /> {t('addRowButton')}
                            </Button>
                        </div>
                    </StorageUnitTable>
                }
            </div>
        </div>
        <Drawer open={scratchpad} onOpenChange={handleCloseScratchpad}>
            <DrawerContent className="px-8 min-h-[65vh]" data-testid="scratchpad-drawer">
                <Button variant="ghost" className="absolute top-0 right-0" onClick={handleCloseScratchpad} data-testid="icon-button">
                    <XMarkIcon className="w-4 h-4" />
                </Button>
                <DrawerHeader className="px-0">
                    <DrawerTitle className="flex justify-between items-center">
                        <h2 className="text-lg font-semibold">{t('scratchpadTitle')}</h2>
                        <div className="flex gap-sm items-center">
                            <Button onClick={() => handleScratchpad()} data-testid="run-submit-button">
                                <PlayIcon className="w-4 h-4" />
                                {t('run')}
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
                    limitContextMenu={true}
                    schema={schema}
                    storageUnit={unitName}
                    onRefresh={handleSubmitRequest}
                    showPagination={false}
                    databaseType={current?.Type}
                />
            </DrawerContent>
        </Drawer>
        <Sheet open={showEntitySearchSheet} onOpenChange={setShowEntitySearchSheet}>
            <SheetContent side="right" className="flex flex-col p-8 min-w-[600px]">
                <div className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <MagnifyingGlassIcon className="w-5 h-5" />
                    {t('searchAround')}
                </div>
                {entitySearchData && (
                    <div className="text-sm text-gray-500">
                        {t('query').replace('{table}', entitySearchData.targetTable).replace('{id}', entitySearchData.value)}
                    </div>
                )}
                <div className="flex-1 overflow-y-auto pr-2">
                    {entitySearchResults && entitySearchResults.Rows.length > 0 ? (
                        <div className="flex flex-col gap-4">
                            <StackList>
                                {entitySearchData && (
                                    <StackListItem key="entity-id" item="ID">
                                        {entitySearchData.value}
                                    </StackListItem>
                                )}
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
                            {t('noEntityFound')}
                        </div>
                    )}
                </div>
                <SheetFooter>
                    <Button onClick={handleCloseEntitySearchSheet} variant="outline">
                        {t('close')}
                    </Button>
                </SheetFooter>
            </SheetContent>
        </Sheet>
    </InternalPage>
}
