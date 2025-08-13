/**
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

import React, { FC, useEffect, useMemo } from "react";
import { Icons } from "../../components/icons";
import { StorageUnitTable } from "../../components/table";
import { useRawExecuteLazyQuery } from "../../generated/graphql";

type PromiseFunction = (code: string) => Promise<any>;

export type IPluginProps = {
    code: string;
    handleExecuteRef: React.MutableRefObject<PromiseFunction | null>;
}

function isSQLQueryAction(code?: string): boolean {
    if (code == null) {
        return true;
    }
    // Remove comments and trim
    const cleaned = code
        .split("\n")
        .filter((text: string) => !text.trim().startsWith("--"))
        .join("\n")
        .trim()
        .toLowerCase();

    // Match common SQL query starting keywords
    // Accepts: select, with, values, show, explain, describe, etc.
    // (add more as needed)
    return /^(select|with|values|show|explain|describe)\b/.test(cleaned);
}

export const QueryView: FC<IPluginProps> = ({ code, handleExecuteRef }) => {
    const [rawExecute, { data }] = useRawExecuteLazyQuery();

    // Set the ref to a function that executes the query and returns a promise
    useEffect(() => {
        handleExecuteRef.current = (code: string) => {
            console.log("code", code);
            return new Promise((resolve, reject) => {
                rawExecute({
                    variables: {
                        query: code,
                    },
                    onCompleted: (data) => {
                        resolve(data.RawExecute);
                    },
                    onError: (error) => {
                        reject(error);
                    },
                });
            });
        };
    }, [rawExecute, handleExecuteRef]);

    if (data == null) {
        return null;
    }

    if (isSQLQueryAction(code) || data.RawExecute.Rows.length > 0) {
        return (
            <div className="flex flex-col w-full" data-testid="cell-query-output">
                {
                    data.RawExecute.Columns.length > 0 && (
                        <StorageUnitTable
                            columns={data.RawExecute.Columns.map((c: any) => c.Name)}
                            columnTypes={data.RawExecute.Columns.map((c: any) => c.Type)}
                            rows={data.RawExecute.Rows}
                            disableEdit={true}
                            height={250}
                        />
                    )
                }
            </div>
        );
    }

    return (
        <div className="bg-white/10 text-neutral-800 dark:text-neutral-300 rounded-lg p-2 flex gap-2 self-start items-center my-4" data-testid="cell-action-output">
            Action Executed
            {Icons.CheckCircle}
        </div>
    );
};
