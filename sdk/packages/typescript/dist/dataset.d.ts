import type { Transport } from './transport.js';
import type { Dataset } from './generated/types.js';
import { type Row } from './hydrate.js';
/** Options for dataset row reads. */
export interface DatasetRowsOptions {
    pageSize?: number;
    pageOffset?: number;
}
/**
 * DatasetHandle is the `whodb.dataset("weekly_kpis")` facade, addressed by
 * dataset name.
 */
export declare class DatasetHandle {
    private readonly transport;
    private readonly projectId;
    private readonly name;
    private datasetCache;
    constructor(transport: Transport, projectId: () => Promise<string>, name: string);
    /** Resolves and caches the dataset metadata backing this handle. */
    meta(): Promise<Dataset>;
    /** Queries dataset rows with optional filter/sort/paging. */
    rows(options?: DatasetRowsOptions): Promise<{
        rows: Row[];
        totalCount: number | null;
    }>;
}
