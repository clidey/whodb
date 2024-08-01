import { FetchResult } from "@apollo/client";
import classNames from "classnames";
import { motion } from "framer-motion";
import { clone, entries, keys, map } from "lodash";
import { cloneElement, FC, useCallback, useEffect, useMemo, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { AnimatedButton } from "../../components/button";
import { Dropdown } from "../../components/dropdown";
import { Icons } from "../../components/icons";
import { Input, InputWithlabel } from "../../components/input";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Table } from "../../components/table";
import { graphqlClient } from "../../config/graphql-client";
import { InternalRoutes } from "../../config/routes";
import { Column, DatabaseType, RecordInput, RowsResult, StorageUnit, UpdateStorageUnitDocument, UpdateStorageUnitMutationResult, useAddRowMutation, useGetStorageUnitRowsLazyQuery } from "../../generated/graphql";
import { notify } from "../../store/function";
import { useAppSelector } from "../../store/hooks";
import { getDatabaseStorageUnitLabel, isNoSQL, isNumeric } from "../../utils/functions";
import { ExploreStorageUnitWhereCondition, IExploreStorageUnitWhereConditionFilter } from "./explore-storage-unit-where-condition";

export const ExploreStorageUnit: FC = () => {
    const [bufferPageSize, setBufferPageSize] = useState("100");
    const [currentPage, setCurrentPage] = useState(0);
    const [whereCondition, setWhereCondition] = useState("");
    const [currentFilters, setCurrentFilters] = useState<IExploreStorageUnitWhereConditionFilter[]>([]);
    const [pageSize, setPageSize] = useState("");
    const unit: StorageUnit = useLocation().state?.unit;
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const navigate = useNavigate();
    const [rows, setRows] = useState<RowsResult>();
    const [showAdd, setShowAdd] = useState(false);
    const [newRowForm, setNewRowForm] = useState<RecordInput[]>([]);

    const [getStorageUnitRows, { loading }] = useGetStorageUnitRowsLazyQuery();
    const [addRow,] = useAddRowMutation();

    const unitName = useMemo(() => {
        return unit?.Name;
    }, [unit]);

    const handleSubmitRequest = useCallback(() => {
        getStorageUnitRows({
            variables: {
                type: current?.Type as DatabaseType,
                schema,
                storageUnit: unitName,
                where: whereCondition,
                pageSize: Number.parseInt(bufferPageSize),
                pageOffset: currentPage,
            },
            onCompleted(data) {
                setRows(data.Row);
                setPageSize(bufferPageSize);
            },
            fetchPolicy: "no-cache",
        });
    }, [getStorageUnitRows, current?.Type, schema, unitName, whereCondition, bufferPageSize, currentPage]);

    const handlePageChange = useCallback((page: number) => {
        setCurrentPage(page-1);
        getStorageUnitRows({
            variables: {
                type: current?.Type as DatabaseType,
                schema,
                storageUnit: unitName,
                where: whereCondition,
                pageSize: Number.parseInt(bufferPageSize),
                pageOffset: currentPage,
            }
        });
    }, [getStorageUnitRows, current?.Type, schema, unitName, whereCondition, bufferPageSize, currentPage]);

    const handleQuery = useCallback(() => {
        handleSubmitRequest();
        setCurrentPage(0);
    }, [handleSubmitRequest]);

    const handleRowUpdate = useCallback((row: Record<string, string>) => {
        if (current == null) {
            return Promise.reject();
        }
        const values = map(entries(row), ([Key, Value]) => ({
            Key,
            Value,
        }));
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

    const totalCount = useMemo(() => {
        return unit?.Attributes.find(attribute => attribute.Key === "Count")?.Value ?? "unknown";
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
    
    const columns = useMemo(() => {
        const dataColumns = rows?.Columns.map(c => c.Name) ?? [];
        if (dataColumns.length !== 1 || dataColumns[0] !== "document") {
            return dataColumns;
        }
        const firstRow = rows?.Rows?.[0];
        if (firstRow == null) {
            return [];
        }
        return keys(JSON.parse(firstRow[0]));
    }, [rows?.Columns, rows?.Rows]);

    useEffect(() => {
        if (unitName == null) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
    }, [navigate, unitName]);

    const handleFilterChange = useCallback((filters: IExploreStorageUnitWhereConditionFilter[]) => {
        setCurrentFilters(filters);
        if (!current?.Type) {
            return;
        }
    
        const databaseType = current.Type;
        let whereClause = "";
    
        switch (databaseType) {
            case DatabaseType.Postgres:
            case DatabaseType.MySql:
            case DatabaseType.Sqlite3:
            case DatabaseType.MariaDb:
                whereClause = filters.map(filter => `${filter.field} ${filter.operator} '${filter.value}'`).join(' AND ');
                break;
            case DatabaseType.ElasticSearch:
                const elasticSearchConditions: Record<string, Record<string, any>> = {};
                filters.forEach(filter => {
                    elasticSearchConditions[filter.operator] = { [filter.field]: filter.value };
                });
                whereClause = JSON.stringify({ query: { bool: { must: Object.entries(elasticSearchConditions).map(([field, condition]) => ({ [field]: condition })) } } });
                break;
            case DatabaseType.MongoDb:
                const mongoDbConditions: Record<string, Record<string, any>> = {};
                filters.forEach(filter => {
                    mongoDbConditions[filter.field] = { [filter.operator]: filter.value };
                });
                whereClause = JSON.stringify(mongoDbConditions);
                break;
            default:
                throw new Error(`Unsupported database type: ${databaseType}`);
        }
        
        setWhereCondition(whereClause);
    }, [current?.Type]);

    const validOperators = useMemo(() => {
        if (!current?.Type) {
            return [];
        }
    
        switch (current.Type) {
            case DatabaseType.Postgres:
            case DatabaseType.MySql:
            case DatabaseType.Sqlite3:
            case DatabaseType.MariaDb:
                return [
                    "=", ">=", ">", "<=", "<", "<>", "!=", "!>", "!<", "BETWEEN", "NOT BETWEEN", 
                    "LIKE", "NOT LIKE", "IN", "NOT IN", "IS NULL", "IS NOT NULL", "AND", "OR", 
                    "NOT"
                ];
            case DatabaseType.ElasticSearch:
                return [
                    "match", "match_phrase", "match_phrase_prefix", "multi_match", "bool", 
                    "term", "terms", "range", "exists", "prefix", "wildcard", "regexp", 
                    "fuzzy", "ids", "constant_score", "function_score", "dis_max", "nested", 
                    "has_child", "has_parent"
                ];
            case DatabaseType.MongoDb:
                return ["eq", "ne", "gt", "gte", "lt", "lte", "in", "nin", "and", "or", 
                        "not", "nor", "exists", "type", "regex", "expr", "mod", "all", 
                        "elemMatch", "size", "bitsAllClear", "bitsAllSet", "bitsAnyClear", 
                        "bitsAnySet", "geoIntersects", "geoWithin", "near", "nearSphere"];
        }
        return [];
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
        if (isNoSQL(current?.Type as DatabaseType) && rows?.Rows != null && rows.Rows.length === 0) {
            try {
                values = entries(JSON.parse(newRowForm[0].Value)).map(([Key, Value]) => ({ Key, Value } as RecordInput));
            } catch {
                values = [];
            }
        }
        addRow({
            variables: {
                type: current?.Type as DatabaseType,
                schema,
                storageUnit: unit.Name,
                values,
            },
            onCompleted() {
                notify("Added data row successfully!", "success");
                setShowAdd(false);
                handleSubmitRequest();
            },
            onError(e) {
                notify(`Unable to add the data row: ${e.message}`, "error");
            },
        });
    }, [addRow, current?.Type, handleSubmitRequest, newRowForm, rows?.Rows, schema, unit.Name]);

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
            <Loading />
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
                    <InputWithlabel label="Page Size" value={bufferPageSize} setValue={setBufferPageSize} />
                    { current?.Type !== DatabaseType.Redis && <ExploreStorageUnitWhereCondition defaultFilters={currentFilters} options={columns} operators={validOperators} onChange={handleFilterChange} /> }
                    <AnimatedButton className="mt-5" type="lg" icon={Icons.CheckCircle} label="Query" onClick={handleQuery} />
                </div>
                <motion.div className="flex flex-col absolute z-10 right-0 top-0 backdrop-blur-xl" variants={{
                    "open": {
                        height: "500px",
                        width: "800px",
                    },
                    "close": {
                        height: "35px",
                        width: "150px",
                    },
                }} animate={showAdd ? "open" : "close"}>
                    <div className="flex w-full justify-end">
                        <AnimatedButton type="lg" icon={Icons.Add} label={showAdd ? "Cancel" : "Add Row"} onClick={handleToggleShowAdd} />
                    </div>
                    <div className={classNames("flex flex-col gap-2 overflow-y-auto h-full p-8 mt-2", {
                        "flex border border-white/5 rounded-lg": showAdd,
                        "hidden": !showAdd,
                    })}>
                        <div className="flex justify-between gap-4">
                            <div className="text-lg text-neutral-800 dark:text-neutral-300 w-full border-b border-white/10 pb-2 mb-2">
                                New row
                            </div>
                            <AnimatedButton type="lg" icon={Icons.CheckCircle} label="Submit" onClick={handleAddSubmitRequest} />
                        </div>
                        {newRowForm.map((col, i) => <>
                            <div key={`add-row-${col.Key}`} className="flex gap-2 items-center">
                                <div className="text-xs text-neutral-800 dark:text-neutral-300 w-[150px]">{col.Key} [{col.Extra?.at(1)?.Value}]</div>
                                <Dropdown className={classNames({
                                    "hidden": isNoSQL(current?.Type as DatabaseType),
                                })} value={configDropdown.find(item => item.id === col.Extra?.at(0)?.Value)}
                                    onChange={item => handleNewFormChange("config", i, item.id)}
                                    items={configDropdown}
                                    showIconOnly={true} />
                                <Input value={col.Value} inputProps={{
                                    placeholder: `Enter value for ${col.Key}`,
                                }} setValue={(value) => handleNewFormChange("value", i, value)} />
                            </div>
                        </>)}
                    </div>
                </motion.div>
            </div>
            <div className="grow">
                {
                    rows != null &&
                    <Table columns={rows.Columns.map(c => c.Name)} columnTags={rows.Columns.map(c => c.Type)}
                        rows={rows.Rows} totalPages={totalPages} currentPage={currentPage+1} onPageChange={handlePageChange}
                        onRowUpdate={handleRowUpdate} disableEdit={rows.DisableUpdate} />
                }
            </div>
        </div>
    </InternalPage>
}