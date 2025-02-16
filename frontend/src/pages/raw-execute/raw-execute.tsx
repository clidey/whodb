// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

import classNames from "classnames";
import { indexOf } from "lodash";
import { FC, useCallback, useMemo, useState } from "react";
import { v4 } from "uuid";
import { AnimatedButton } from "../../components/button";
import { CodeEditor } from "../../components/editor";
import { Icons } from "../../components/icons";
import { Loading, LoadingPage } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Table } from "../../components/table";
import { InternalRoutes } from "../../config/routes";
import { useRawExecuteLazyQuery } from "../../generated/graphql";

type IRawExecuteCellProps = {
    cellId: string;
    onAdd: (cellId: string) => void;
    onDelete?: (cellId: string) => void;
    showTools?: boolean;
}

const RawExecuteCell: FC<IRawExecuteCellProps> = ({ cellId, onAdd, onDelete, showTools }) => {
    const [code, setCode] = useState("");
    const [submittedCode, setSubmittedCode] = useState("");
    const [rawExecute, { data: rows, loading, error }] = useRawExecuteLazyQuery();

    const handleRawExecute = useCallback(() => {
        setSubmittedCode(code);
        rawExecute({
            variables: {
                query: code,
            },
        })
    }, [code, rawExecute]);

    const handleAdd = useCallback(() => {
        onAdd(cellId);
    }, [cellId, onAdd]);

    const handleDelete = useCallback(() => {
        onDelete?.(cellId);
    }, [cellId, onDelete]);

    const isCodeAQuery = useMemo(() => {
        if (submittedCode == null) {
            return true;
        }
        return submittedCode.split("\n").filter(text => !text.startsWith("--")).join("\n").trim().toLowerCase().startsWith("select");
    }, [submittedCode]);

    return <div className="flex flex-col grow group/cell">
            <div className="relative">
                <div className="flex grow h-[150px] border border-gray-200 rounded-md overflow-hidden dark:bg-white/10 dark:border-white/5">
                    <CodeEditor language="sql" value={code} setValue={setCode} onRun={handleRawExecute} />
                </div>
                <div className={classNames("absolute -bottom-3 z-20 flex justify-between px-3 pr-8 w-full opacity-0 transition-all duration-500 group-hover/cell:opacity-100", {
                    "opacity-100": showTools,
                })}>
                    <div className="flex gap-2">
                        <AnimatedButton icon={Icons.PlusCircle} label="Add" onClick={handleAdd} />
                        {
                            onDelete != null &&
                            <AnimatedButton className="bg-red-100/80 hover:bg-red-200 dark:hover:bg-red-900" iconClassName="stroke-red-800" labelClassName="text-red-800"  icon={Icons.Delete} label="Delete" onClick={handleDelete} />
                        }
                    </div>
                    <AnimatedButton className="bg-green-200 hover:bg-green-400 dark:hover:bg-green-900" iconClassName="stroke-green-800" labelClassName="text-green-800" icon={Icons.CheckCircle} label="Submit query" onClick={handleRawExecute} disabled={loading} />
                </div>
            </div>
            {
                error != null &&
                <div className="flex items-center justify-between mt-4">
                    <div className="text-sm text-red-500 w-[33vw]">{error?.message ?? ""}</div>
                </div>
            }
            {
                loading
                ? <div className="flex justify-center items-center h-[250px]">
                    <Loading />
                </div>
                : rows != null && submittedCode.length > 0 && 
                (isCodeAQuery
                    ?
                        <div className="flex flex-col w-full h-[250px] mt-4">
                            <Table columns={rows.RawExecute.Columns.map(c => c.Name)} columnTags={rows.RawExecute.Columns.map(c => c.Type)}
                                rows={rows.RawExecute.Rows} totalPages={1} currentPage={1} disableEdit={true} />
                        </div>
                :   <div className="bg-white/10 text-neutral-800 dark:text-neutral-300 rounded-lg p-2 flex gap-2 self-start items-center my-4">
                        Action Executed
                        {Icons.CheckCircle}
                    </div>)
            }
        </div>
}

export const RawExecutePage: FC = () => {
    const [cellIds, setCellIds] = useState<string[]>([v4()]);
    
    const handleAdd = useCallback((id: string) => {
        const index = indexOf(cellIds, id);
        const newCellIds = [...cellIds];
        newCellIds.splice(index+1, 0, v4());
        setCellIds(newCellIds);
    }, [cellIds]);

    const handleDelete = useCallback((cellId: string) => {
        if (cellIds.length <= 1) {
            return;
        }
        setCellIds(ids => ids.filter(id => id !== cellId));
    }, [cellIds.length]);

    return (
        <InternalPage routes={[InternalRoutes.RawExecute]}>
            <div className="flex justify-center items-center w-full">
                <div className="w-full max-w-[1000px] flex flex-col gap-4">
                    {
                        cellIds.map((cellId, index) => (
                            <div key={cellId}>
                                {index > 0 && <div className="border-dashed border-t border-gray-300 my-2 dark:border-neutral-600"></div>}
                                <RawExecuteCell key={cellId} cellId={cellId} onAdd={handleAdd} onDelete={cellIds.length <= 1 ? undefined : handleDelete}
                                    showTools={cellIds.length === 1} />
                            </div>
                        ))
                    }
                </div>
            </div>
        </InternalPage>
    )
}   