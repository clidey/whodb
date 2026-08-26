import type { Transport } from './transport.js';
import type { Column, PlatformSource, SourceObject, SourceObjectRefInput, WhereCondition, SortCondition } from './generated/types.js';
import { ListCall } from './pagination.js';
/** Options for source row reads. */
export interface SourceRowsOptions {
    where?: WhereCondition;
    sort?: SortCondition[];
    pageSize?: number;
}
/**
 * SourceHandle is the `whodb.source("src_...")` facade: browse and read a
 * connected data source by ID.
 */
export declare class SourceHandle {
    private readonly transport;
    private readonly projectId;
    private readonly sourceId;
    constructor(transport: Transport, projectId: () => Promise<string>, sourceId: string);
    /** Lists browsable objects (schemas, tables, collections...). */
    objects(options?: {
        parent?: SourceObjectRefInput;
        pageSize?: number;
        pageOffset?: number;
    }): Promise<SourceObject[]>;
    /** Lists the columns of one object (table/collection). */
    columns(ref: SourceObjectRefInput): Promise<Column[]>;
    /** Reads rows from one object; supports .pages() iteration. */
    rows(ref: SourceObjectRefInput, options?: SourceRowsOptions): ListCall;
}
/** Lists the project's sources. */
export declare function listSources(transport: Transport, projectId: string): Promise<PlatformSource[]>;
