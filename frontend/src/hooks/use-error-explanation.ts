import { useCallback, useEffect, useRef, useState } from 'react';
import { v4 as uuidv4 } from 'uuid';
import { useAppSelector } from '../store/hooks';
import { graphqlClient } from '../config/graphql-client';
import { useSourceContract } from './useSourceContract';
import { buildSourceParentRef, buildSourceObjectRef } from '../utils/source-refs';
import {
    GetStorageUnitsDocument,
    ColumnsDocument,
    SourceObjectKind,
} from '@graphql';
import {
    addListener,
    isModelReady,
    requestExplanation,
    type ErrorExplainerListener,
} from '../services/error-explainer';

export type ExplainerStatus = 'idle' | 'loading-model' | 'generating' | 'done' | 'error';

type ExplanationResult = {
    explanation: string;
    fix: string;
};

function parseExplanation(text: string): ExplanationResult {
    const explanationMatch = text.match(/Explanation:\s*([\s\S]*?)(?=\n\nFix:|$)/);
    const fixMatch = text.match(/Fix:\s*([\s\S]*)/);
    return {
        explanation: explanationMatch?.[1]?.trim() ?? text.trim(),
        fix: fixMatch?.[1]?.trim() ?? '',
    };
}

function extractTableNames(query: string): string[] {
    const pattern = /(?:FROM|JOIN|INTO|UPDATE|TABLE)\s+([`"']?[\w.]+[`"']?)/gi;
    const names: string[] = [];
    let match;
    while ((match = pattern.exec(query)) !== null) {
        names.push(match[1].replace(/[`"']/g, ''));
    }
    return [...new Set(names)];
}

function formatColumnDef(col: { Name: string; Type: string; IsPrimary: boolean; IsForeignKey: boolean }): string {
    let def = `${col.Name} ${col.Type}`;
    if (col.IsPrimary) def += ' PK';
    if (col.IsForeignKey) def += ' FK';
    return def;
}

export function useErrorExplanation(params: {
    error: string | null;
    query: string;
    dbType: string;
    schema: string;
}) {
    const enabled = useAppSelector(state => state.settings.errorExplainerEnabled);
    const current = useAppSelector(state => state.auth.current);
    const selectedSchema = useAppSelector(state => state.database.schema);
    const { item } = useSourceContract(current?.Type);

    const [status, setStatus] = useState<ExplainerStatus>('idle');
    const [result, setResult] = useState<ExplanationResult | null>(null);
    const [progress, setProgress] = useState<number | undefined>(undefined);
    const [errorMessage, setErrorMessage] = useState<string | null>(null);
    const [retryCount, setRetryCount] = useState(0);
    const requestIdRef = useRef<string | null>(null);

    useEffect(() => {
        if (!enabled || !params.error) {
            setStatus('idle');
            setResult(null);
            setProgress(undefined);
            setErrorMessage(null);
            requestIdRef.current = null;
            return;
        }

        const id = uuidv4();
        requestIdRef.current = id;
        setResult(null);
        setErrorMessage(null);
        setProgress(undefined);

        setStatus(isModelReady() ? 'generating' : 'loading-model');

        const listener: ErrorExplainerListener = (event) => {
            if (event.type === 'status') {
                if (event.status === 'loading-model') setStatus('loading-model');
                else if (event.status === 'generating' && 'id' in event && event.id === id) setStatus('generating');
            } else if (event.type === 'progress') {
                setProgress(event.progress);
            } else if (event.type === 'result' && event.id === id) {
                setResult(parseExplanation(event.text));
                setStatus('done');
            } else if (event.type === 'error') {
                if (!('id' in event) || event.id === id) {
                    setErrorMessage(event.message);
                    setStatus('error');
                }
            }
        };

        const unsubscribe = addListener(listener);

        const fetchSchemaAndRequest = async () => {
            let schemaContext = params.schema;

            try {
                const tableNames = extractTableNames(params.query);
                if (tableNames.length > 0 && item && current) {
                    const columnResults: string[] = [];

                    for (const tableName of tableNames.slice(0, 5)) {
                        try {
                            const ref = buildSourceObjectRef(item, current, selectedSchema, tableName);
                            const { data } = await graphqlClient.query({
                                query: ColumnsDocument,
                                variables: { ref },
                                fetchPolicy: 'cache-first',
                            });
                            if (data?.Columns?.length) {
                                const cols = data.Columns.map(formatColumnDef).join(', ');
                                columnResults.push(`${tableName}(${cols})`);
                            }
                        } catch {
                            // Table might not exist — that could be the error itself
                        }
                    }

                    if (columnResults.length > 0) {
                        schemaContext = columnResults.join('; ');
                    } else {
                        const parentRef = buildSourceParentRef(item, current, selectedSchema);
                        if (parentRef) {
                            try {
                                const { data } = await graphqlClient.query({
                                    query: GetStorageUnitsDocument,
                                    variables: { parent: parentRef },
                                    fetchPolicy: 'cache-first',
                                });
                                if (data?.StorageUnit?.length) {
                                    const names = data.StorageUnit.map(u => u.Name).slice(0, 20);
                                    schemaContext = `Tables: ${names.join(', ')}`;
                                }
                            } catch {
                                // Fall through with basic schema
                            }
                        }
                    }
                }
            } catch {
                // Schema fetch failed — use basic context
            }

            requestExplanation({
                id,
                error: params.error!,
                query: params.query,
                dbType: params.dbType,
                schema: schemaContext,
            });
        };

        fetchSchemaAndRequest();

        return () => {
            unsubscribe();
            requestIdRef.current = null;
        };
    }, [enabled, params.error, params.query, params.dbType, params.schema, item, current, selectedSchema, retryCount]);

    const retry = useCallback(() => {
        setRetryCount(c => c + 1);
    }, []);

    return { enabled, status, result, progress, errorMessage, retry };
}
