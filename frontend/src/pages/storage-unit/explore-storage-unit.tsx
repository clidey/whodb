import { useLazyQuery } from "@apollo/client";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { AnimatedButton } from "../../components/button";
import { Icons } from "../../components/icons";
import { InputWithlabel } from "../../components/input";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Table } from "../../components/table";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, GetStorageUnitRowsDocument, GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables, StorageUnit } from "../../generated/graphql";
import { isNumeric } from "../../utils/functions";
import { useAppSelector } from "../../store/hooks";

export const ExploreStorageUnit: FC = () => {
    const [pageSize, setPageSize] = useState("10");
    const [currentPage, setCurrentPage] = useState(0);
    const [whereCondition, setWhereCondition] = useState("");
    const unit: StorageUnit = useLocation().state?.unit;
    const schema = useAppSelector(state => state.database.schema);
    const current = useAppSelector(state => state.auth.current);

    const [getStorageUnitRows, { data: rows, loading }] = useLazyQuery<GetStorageUnitRowsQuery, GetStorageUnitRowsQueryVariables>(GetStorageUnitRowsDocument);

    const handleSubmitRequest = useCallback(() => {
        getStorageUnitRows({
            variables: {
                type: current?.Type as DatabaseType,
                schema,
                storageUnit: unit.Name,
                where: whereCondition,
                pageSize: Number.parseInt(pageSize),
                pageOffset: currentPage,
            },
        });
    }, [getStorageUnitRows, current?.Type, schema, unit.Name, whereCondition, pageSize, currentPage]);

    const handlePageChange = useCallback((page: number) => {
        setCurrentPage(page-1);
        getStorageUnitRows({
            variables: {
                type: current?.Type as DatabaseType,
                schema,
                storageUnit: unit.Name,
                where: whereCondition,
                pageSize: Number.parseInt(pageSize),
                pageOffset: currentPage,
            }
        });
    }, [current?.Type, currentPage, getStorageUnitRows, pageSize, schema, unit.Name, whereCondition]);

    const handleQuery = useCallback(() => {
        handleSubmitRequest();
        setCurrentPage(0);
    }, [handleSubmitRequest]);

    const totalCount: number = useMemo(() => {
        const rowCount = unit?.Attributes.find(attribute => attribute.Key === "Row Count")?.Value ?? "0";
        if (isNumeric(rowCount)) {
            return Number.parseInt(rowCount);
        }
        return 0;
    }, [unit]);

    const totalPages = useMemo(() => {
        if (!isNumeric(pageSize)) {
            return 1;
        }
        return Math.max(Math.round(totalCount/(Number.parseInt(pageSize)+1)), 1);
    }, [pageSize, totalCount]);

    useEffect(() => {
        handleSubmitRequest();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    if (unit == null) {
        return <Navigate to={InternalRoutes.Dashboard.StorageUnit.path} />
    }

    if (loading) {
        return <InternalPage routes={[InternalRoutes.Dashboard.StorageUnit, InternalRoutes.Dashboard.ExploreStorageUnit]}>
            <Loading />
        </InternalPage>
    }

    return <InternalPage routes={[InternalRoutes.Dashboard.StorageUnit, InternalRoutes.Dashboard.ExploreStorageUnit]}>
        <div className="flex flex-col grow gap-4">
            <div className="flex items-center justify-between">
                <div className="flex gap-2 items-center">
                    <div className="text-xl font-bold mr-4">{unit.Name}</div>
                </div>
                <div className="text-sm mr-4"><span className="font-semibold">Count:</span> {totalCount}</div>
            </div>
            <div className="flex gap-2 items-end">
                <InputWithlabel label="Page Size" value={pageSize} setValue={setPageSize} />
                <InputWithlabel label="Where Condition" value={whereCondition} setValue={setWhereCondition} />
                <AnimatedButton type="lg" icon={Icons.CheckCircle} label="Query" onClick={handleQuery} />
            </div>
            {
                rows != null &&
                <Table columns={rows.Row.Columns.map(c => c.Name)} columnTags={rows.Row.Columns.map(c => c.Type)}
                    rows={rows.Row.Rows} totalPages={totalPages} currentPage={currentPage+1} onPageChange={handlePageChange} />
            }
        </div>
    </InternalPage>
}