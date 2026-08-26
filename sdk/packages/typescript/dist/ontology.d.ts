import type { Transport } from './transport.js';
import type { OntologyDescription, OntologyFastLookup, OntologyObjectType, OntologyStatsResult, OntologySimilarInput, OntologySimilarityResult, OntologyQueryInput, OntologyQuerySortInput, OntologyAggregateMetricInput, WhereCondition, SortCondition, OntologyAddRowsResult } from './generated/types.js';
import { type Row } from './hydrate.js';
import { ListCall } from './pagination.js';
/** Options for list-shaped ontology reads. `where` is a JSON filter object
 * (property → { eq/gt/lt/in/... }) serialized into OntologyQuery.whereJson. */
export interface ListOptions {
    where?: Record<string, unknown>;
    sort?: OntologyQuerySortInput[];
    pageSize?: number;
}
/** Options for the flexible query surface (search, joins, grouping). */
export interface QueryOptions extends Omit<OntologyQueryInput, 'entity'> {
}
/** Options for aggregations. */
export interface AggregateOptions {
    groupBy: string[];
    metrics?: OntologyAggregateMetricInput[];
    where?: WhereCondition;
    sort?: SortCondition[];
    pageSize?: number;
}
/**
 * OntologyHandle is the `whodb.ontology("User")` facade: reads and record
 * writes for one ontology entity, addressed by apiName. Entity metadata is
 * fetched once per handle and reused for pk lookups and row hydration.
 */
export declare class OntologyHandle {
    private readonly transport;
    private readonly projectId;
    private readonly apiName;
    private entityCache;
    constructor(transport: Transport, projectId: () => Promise<string>, apiName: string);
    /** Resolves and caches the entity metadata backing this handle. */
    entityMeta(): Promise<OntologyObjectType>;
    private propertyTypes;
    /** Describes the entity: schema, properties, links, sample queries. */
    describe(): Promise<OntologyDescription>;
    /** Fetches a single record by primary key, or null when absent. */
    get(pk: string | number): Promise<Row | null>;
    /** Lists records with optional filter/sort; supports .pages() iteration. */
    list(options?: ListOptions): ListCall;
    /** Flexible query: text search, joins, grouping, metrics. */
    query(options: QueryOptions): Promise<Row[]>;
    /** Aggregates records grouped by properties with metric functions. */
    aggregate(options: AggregateOptions): Promise<Row[]>;
    /** Statistical summary of one property. */
    stats(property: string, options?: {
        where?: WhereCondition;
    }): Promise<OntologyStatsResult>;
    /** Embedding-based similarity search over this entity's records. */
    similar(input: Omit<OntologySimilarInput, 'entityId'>): Promise<OntologySimilarityResult>;
    /** Follows an outgoing link from one record to its related records. */
    followLink(pk: string | number, linkApiName: string, options?: {
        pageSize?: number;
    }): ListCall;
    /** Follows a link inbound from another entity's records to this record. */
    followIncomingLink(pk: string | number, sourceEntityApiName: string, linkApiName: string, options?: {
        pageSize?: number;
    }): ListCall;
    /** Lists the entity's fast lookups. */
    fastLookups(): Promise<OntologyFastLookup[]>;
    /** Inserts one record. Values are field name/value pairs. */
    create(values: Record<string, unknown>): Promise<void>;
    /** Inserts many records; idempotencyKey makes safe retries possible. */
    createMany(rows: Array<Record<string, unknown>>, options?: {
        idempotencyKey?: string;
    }): Promise<OntologyAddRowsResult>;
    /** Updates one record identified by primary key. */
    update(pk: string | number, values: Record<string, unknown>): Promise<void>;
    /** Deletes one record identified by primary key. */
    delete(pk: string | number): Promise<void>;
}
