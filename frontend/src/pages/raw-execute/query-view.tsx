/*
 * Copyright 2026 Clidey, Inc.
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

import {useApolloClient} from "@apollo/client/react";
import type {FC} from "react";
import React, { useEffect} from "react";
import {StorageUnitTable} from "../../components/table";
import {RawExecuteDocument} from "../../generated/graphql";
import type {RowsResult} from "@graphql";
import {CheckCircleIcon} from "../../components/heroicons";
import {useAppSelector} from "../../store/hooks";
import {isDestructiveQuery} from "../../utils/query-utils";

type PromiseFunction = (code: string, signal?: AbortSignal) => Promise<any>;

export type IPluginProps = {
    code: string;
    handleExecuteRef: React.MutableRefObject<PromiseFunction | null>;
    providerId?: string;
    modelType: string;
    token?: string;
    schema: string;
    containerWidth?: number;
    rowsResult?: RowsResult | null;
}

export const QueryView: FC<IPluginProps> = ({ code, handleExecuteRef, containerWidth, rowsResult: data }) => {
    const client = useApolloClient();
    const currentType = useAppSelector(state => state.auth.current?.Type);

    // Set the ref to a function that executes the query and returns a promise
    useEffect(() => {
        handleExecuteRef.current = async (code: string, signal?: AbortSignal) => {
            const result = await client.query({
                query: RawExecuteDocument,
                variables: { query: code },
                context: { fetchOptions: { signal } },
                fetchPolicy: 'network-only',
            });
            if (result.error) {
                throw result.error;
            }
            const data = result.data?.RawExecute ?? null;
            if (!isDestructiveQuery(code, currentType)) {
                return data;
            }
            return null;
        };
    }, [client, handleExecuteRef, currentType]);

    if (data == null) {
        return null;
    }

    if (!isDestructiveQuery(code, currentType)) {
        return (
            <div className="flex flex-col w-full" data-testid="cell-query-output">
                {
                    data.Columns.length > 0 && (
                        <StorageUnitTable
                            key={containerWidth}
                            columns={data.Columns.map((c: any) => c.Name)}
                            columnTypes={data.Columns.map((c: any) => c.Type)}
                            rows={data.Rows}
                            disableEdit={true}
                            limitContextMenu={true}
                            height={250}
                            databaseType={currentType}
                            rawQuery={code}
                            totalCount={data.TotalCount}
                        />
                    )
                }
            </div>
        );
    }

    return (
        <div className="bg-white/10 text-neutral-800 dark:text-neutral-300 rounded-lg p-2 flex gap-sm self-start items-center my-4" data-testid="cell-action-output">
            Action Executed
            <CheckCircleIcon className="w-4 h-4" />
        </div>
    );
};
