import { FetchResult } from "@apollo/client";
import { entries, keys, map } from "lodash";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { AnimatedButton } from "../../components/button";
import { Icons } from "../../components/icons";
import { InputWithlabel } from "../../components/input";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Table } from "../../components/table";
import { graphqlClient } from "../../config/graphql-client";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, StorageUnit, UpdateStorageUnitDocument, UpdateStorageUnitMutationResult, useGetStorageUnitRowsLazyQuery } from "../../generated/graphql";
import { notify } from "../../store/function";
import { useAppSelector } from "../../store/hooks";
import { getDatabaseStorageUnitLabel, isNumeric } from "../../utils/functions";
import { ExploreStorageUnitWhereCondition, IExploreStorageUnitWhereConditionFilter } from "./explore-storage-unit-where-condition";

export const ExploreStorageUnit: FC = () => {
    const [bufferPageSize, setBufferPageSize] = useState("10");
    const [currentPage, setCurrentPage] = useState(0);
    const [whereCondition, setWhereCondition] = useState("");
    const [pageSize, setPageSize] = useState("");
    const unit: StorageUnit = useLocation().state?.unit;
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);
    const navigate = useNavigate();

    const [getStorageUnitRows, { data: rows, loading }] = useGetStorageUnitRowsLazyQuery();

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
            onCompleted() {
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
        const dataColumns = rows?.Row.Columns.map(c => c.Name) ?? [];
        if (dataColumns.length !== 1 || dataColumns[0] !== "document") {
            return dataColumns;
        }
        const firstRow = rows?.Row.Rows?.[0];
        if (firstRow == null) {
            return dataColumns;
        }
        return keys(JSON.parse(firstRow[0]));
    }, [rows?.Row.Columns, rows?.Row.Rows]);

    useEffect(() => {
        if (unitName == null) {
            navigate(InternalRoutes.Dashboard.StorageUnit.path);
        }
    }, [navigate, unitName]);
    const handleFilterChange = useCallback((filters: IExploreStorageUnitWhereConditionFilter[]) => {
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
                    elasticSearchConditions[filter.field] = { [filter.operator]: filter.value };
                });
                whereClause = JSON.stringify({ query: { bool: { must: Object.values(elasticSearchConditions) } } });
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
                return ["=", ">=", ">", "<=", "<"];
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
            <div className="flex gap-2">
                <InputWithlabel label="Page Size" value={bufferPageSize} setValue={setBufferPageSize} />
                { current?.Type !== DatabaseType.Redis && <ExploreStorageUnitWhereCondition options={columns} operators={validOperators} onChange={handleFilterChange} /> }
                <AnimatedButton className="mt-5" type="lg" icon={Icons.CheckCircle} label="Query" onClick={handleQuery} />
            </div>
            <div className="grow">
                {
                    rows != null &&
                    <Table columns={rows.Row.Columns.map(c => c.Name)} columnTags={rows.Row.Columns.map(c => c.Type)}
                        rows={rows.Row.Rows} totalPages={totalPages} currentPage={currentPage+1} onPageChange={handlePageChange}
                        onRowUpdate={handleRowUpdate} disableEdit={rows.Row.DisableUpdate} />
                }
            </div>
        </div>
    </InternalPage>
}