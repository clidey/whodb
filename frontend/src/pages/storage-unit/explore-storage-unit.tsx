import { entries, map } from "lodash";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { Navigate, useLocation } from "react-router-dom";
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
import { isNoSQL, isNumeric } from "../../utils/functions";
import { FetchResult } from "@apollo/client";

export const ExploreStorageUnit: FC = () => {
    const [bufferPageSize, setBufferPageSize] = useState("10");
    const [currentPage, setCurrentPage] = useState(0);
    const [whereCondition, setWhereCondition] = useState("");
    const [pageSize, setPageSize] = useState("");
    const unit: StorageUnit = useLocation().state?.unit;
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);

    const [getStorageUnitRows, { data: rows, loading }] = useGetStorageUnitRowsLazyQuery();

    const handleSubmitRequest = useCallback(() => {
        getStorageUnitRows({
            variables: {
                type: current?.Type as DatabaseType,
                schema,
                storageUnit: unit.Name,
                where: whereCondition,
                pageSize: Number.parseInt(bufferPageSize),
                pageOffset: currentPage,
            },
            onCompleted() {
                setPageSize(bufferPageSize);
            },
        });
    }, [getStorageUnitRows, current?.Type, schema, unit.Name, whereCondition, bufferPageSize, currentPage]);

    const handlePageChange = useCallback((page: number) => {
        setCurrentPage(page-1);
        getStorageUnitRows({
            variables: {
                type: current?.Type as DatabaseType,
                schema,
                storageUnit: unit.Name,
                where: whereCondition,
                pageSize: Number.parseInt(bufferPageSize),
                pageOffset: currentPage,
            }
        });
    }, [current?.Type, currentPage, getStorageUnitRows, bufferPageSize, schema, unit.Name, whereCondition]);

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
                        storageUnit: unit.Name,
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
    }, [current, schema, unit.Name]);

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
        const name = current == null || !isNoSQL(current.Type) ? "Tables" : "Collections";
        return [
            {
                ...InternalRoutes.Dashboard.StorageUnit,
                name,
            },
            InternalRoutes.Dashboard.ExploreStorageUnit
        ];
    }, [current]);

    if (unit == null) {
        return <Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />
    }

    if (loading) {
        return <InternalPage routes={routes}>
            <Loading />
        </InternalPage>
    }

    return <InternalPage routes={routes}>
        <div className="flex flex-col grow gap-4">
            <div className="flex items-center justify-between">
                <div className="flex gap-2 items-center">
                    <div className="text-xl font-bold mr-4">{unit.Name}</div>
                </div>
                <div className="text-sm mr-4"><span className="font-semibold">Total Count:</span> {totalCount}</div>
            </div>
            <div className="flex gap-2 items-end">
                <InputWithlabel label="Page Size" value={bufferPageSize} setValue={setBufferPageSize} />
                <InputWithlabel label="Where Condition" value={whereCondition} setValue={setWhereCondition} />
                <AnimatedButton type="lg" icon={Icons.CheckCircle} label="Query" onClick={handleQuery} />
            </div>
            {
                rows != null &&
                <Table columns={rows.Row.Columns.map(c => c.Name)} columnTags={rows.Row.Columns.map(c => c.Type)}
                    rows={rows.Row.Rows} totalPages={totalPages} currentPage={currentPage+1} onPageChange={handlePageChange}
                    onRowUpdate={handleRowUpdate} disableEdit={rows.Row.DisableUpdate} />
            }
        </div>
    </InternalPage>
}