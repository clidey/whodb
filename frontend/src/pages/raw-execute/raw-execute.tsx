import { useLazyQuery } from "@apollo/client";
import { FC, useCallback, useState } from "react";
import { AnimatedButton } from "../../components/button";
import { CodeEditor } from "../../components/editor";
import { Icons } from "../../components/icons";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Table } from "../../components/table";
import { InternalRoutes } from "../../config/routes";
import { DatabaseType, RawExecuteDocument, RawExecuteQuery, RawExecuteQueryVariables } from "../../generated/graphql";

export const RawExecutePage: FC = () => {
    const [code, setCode] = useState("");
    const [rawExecute, { data: rows, loading, error }] = useLazyQuery<RawExecuteQuery, RawExecuteQueryVariables>(RawExecuteDocument);

    const handleRawExecute = useCallback(() => {
        rawExecute({
            variables: {
                type: DatabaseType.Postgres,
                query: code,
            },
        })
    }, [code, rawExecute]);

    if (loading) {
        return <InternalPage routes={[InternalRoutes.Dashboard.StorageUnit, InternalRoutes.Dashboard.ExploreStorageUnit]}>
            <Loading />
        </InternalPage>
    }

    return (
        <InternalPage routes={[InternalRoutes.RawExecute]}>
            <div className="flex flex-col grow gap-2">
                <div className="flex grow h-[25vh] border border-gray-200 rounded-md overflow-hidden">
                    <CodeEditor language="sql" value={code} setValue={setCode} onRun={handleRawExecute} />
                </div>
                <div className="flex items-center justify-between">
                    <div className="text-sm text-red-500">{error?.message ?? ""}</div>
                    <AnimatedButton icon={Icons.CheckCircle} label="Submit query" onClick={handleRawExecute} type="lg" />
                </div>
                {
                    rows == null
                    ? <div className="flex grow h-[50vh] text-sm text-gray-600 justify-center items-center">
                        No Results
                    </div>
                    : <div className="flex flex-col w-full">
                        <Table className="h-[40vh] overflow-y-scroll" columns={rows.RawExecute.Columns.map(c => c.Name)} columnTags={rows.RawExecute.Columns.map(c => c.Type)}
                            rows={rows.RawExecute.Rows} totalPages={1} currentPage={1} />
                    </div>
                }
            </div>
        </InternalPage>
    )
}   