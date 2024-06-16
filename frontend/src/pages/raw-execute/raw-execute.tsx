import { FC, KeyboardEvent, useCallback, useMemo, useRef, useState } from "react";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { CodeEditor } from "../../components/editor";
import { AnimatedButton } from "../../components/button";
import { Icons } from "../../components/icons";
import { Table } from "../../components/table";
import { useLazyQuery } from "@apollo/client";
import { DatabaseType, RawExecuteDocument, RawExecuteQuery, RawExecuteQueryVariables } from "../../generated/graphql";
import { Loading } from "../../components/loading";
import { SearchInput } from "../../components/search";

export const RawExecutePage: FC = () => {
    const [code, setCode] = useState("");
    const [search, setSearch] = useState("");
    const [searchIndex, setSearchIndex] = useState(0);
    const tableRef = useRef<HTMLTableElement>(null);

    const [rawExecute, { data: rows, loading, error }] = useLazyQuery<RawExecuteQuery, RawExecuteQueryVariables>(RawExecuteDocument);

    const handleRawExecute = useCallback(() => {
        rawExecute({
            variables: {
                type: DatabaseType.Postgres,
                query: code,
            },
        })
    }, [code, rawExecute]);

    const rowCount = useMemo(() => {
        return rows?.RawExecute.Rows.length ?? 0;
    }, [rows?.RawExecute.Rows.length]);

    const handleKeyUp = useCallback((e: KeyboardEvent<HTMLInputElement>) => {
        if (tableRef.current == null) {
            return;
        }
        let interval: NodeJS.Timeout;
        if (e.key === "Enter") {
            let newSearchIndex = (searchIndex+1) % rowCount;
            setSearchIndex(newSearchIndex);
            const searchText = search.toLowerCase();
            let index = 0;
            const tbody = tableRef.current.querySelector("tbody");
            if (tbody == null) {
                return;
            }
            for (const childNode of tbody.childNodes) {
                if (childNode instanceof HTMLTableRowElement) {
                    const text = childNode.textContent?.toLowerCase();
                    if (text != null && searchText != null && text.includes(searchText)) {
                        if (index === newSearchIndex) {
                            childNode.scrollIntoView({
                                behavior: "smooth",
                                block: "center",
                                inline: "center",
                            });
                            for (const cell of childNode.querySelectorAll("input")) {
                                if (cell instanceof HTMLInputElement) {
                                    cell.classList.add("!bg-yellow-100");
                                    interval = setTimeout(() => {
                                        cell.classList.remove("!bg-yellow-100");
                                    }, 3000);
                                }
                            }
                            return;
                        }
                        index++;
                    }
                }
            };
        }
        
        return () => {
            if (interval != null) {
                clearInterval(interval);
            }
        }
    }, [search, rowCount, searchIndex]);

    const handleSearchChange = useCallback((newValue: string) => {
        setSearchIndex(-1);
        setSearch(newValue);
    }, []);

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
            </div>
            {
                rows == null
                ? <div className="flex grow h-[50vh] text-sm text-gray-600 justify-center items-center">
                    No Results
                </div>
                : <div className="flex flex-col w-full">
                    <Table tableRef={tableRef} className="h-[40vh] overflow-y-scroll" columns={rows.RawExecute.Columns.map(c => c.Name)} columnTags={rows.RawExecute.Columns.map(c => c.Type)}
                        rows={rows.RawExecute.Rows} totalPages={1} currentPage={1}>
                        <div className="flex justify-between items-center w-full">
                            <div>
                                <SearchInput search={search} setSearch={handleSearchChange} placeholder="Search through rows     [Press Enter]" inputProps={{
                                    className: "w-[300px]",
                                    onKeyUp: handleKeyUp,
                                }} />
                            </div>
                            <div className="flex gap-4 items-center">
                                <div className="text-sm text-gray-600"><span className="font-semibold">Count:</span> {rowCount}</div>
                                <AnimatedButton icon={Icons.Download} label="Export" type="lg" />
                            </div>
                        </div>
                    </Table>
                </div>
            }
        </InternalPage>
    )
}