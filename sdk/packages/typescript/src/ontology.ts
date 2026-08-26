import type { Transport } from './transport.js';
import * as ops from './generated/operations.js';
import type {
  OntologyDescription, OntologyFastLookup, OntologyObjectType, OntologyStatsResult,
  OntologySimilarInput, OntologySimilarityResult, OntologyQueryInput,
  OntologyQuerySortInput, OntologyAggregateMetricInput, WhereCondition,
  SortCondition, RecordInput, OntologyCreateRowInput, OntologyAddRowsResult,
} from './generated/types.js';
import { hydrateRows, propertyTypesOf, type PropertyTypes, type Row } from './hydrate.js';
import { ListCall, type Page } from './pagination.js';
import { warnIfFlagged } from './manifest-check.js';
import { NotFoundError, ValidationError } from './errors.js';

const DEFAULT_PAGE_SIZE = 100;

/** Options for list-shaped ontology reads. `where` is a JSON filter object
 * (property → { eq/gt/lt/in/... }) serialized into OntologyQuery.whereJson. */
export interface ListOptions {
  where?: Record<string, unknown>;
  sort?: OntologyQuerySortInput[];
  pageSize?: number;
}

/** Options for the flexible query surface (search, joins, grouping). */
export interface QueryOptions extends Omit<OntologyQueryInput, 'entity'> {}

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
export class OntologyHandle {
  private readonly transport: Transport;
  private readonly projectId: () => Promise<string>;
  private readonly apiName: string;
  private entityCache: OntologyObjectType | null = null;

  constructor(transport: Transport, projectId: () => Promise<string>, apiName: string) {
    this.transport = transport;
    this.projectId = projectId;
    this.apiName = apiName;
  }

  /** Resolves and caches the entity metadata backing this handle. */
  async entityMeta(): Promise<OntologyObjectType> {
    if (this.entityCache) return this.entityCache;
    warnIfFlagged('OntologyEntities');
    const entities = await ops.ontologyEntities(this.transport, { projectId: await this.projectId() });
    const entity = entities.find(e => e.apiName === this.apiName);
    if (!entity) {
      throw new NotFoundError(`ontology entity "${this.apiName}" not found in this project`);
    }
    this.entityCache = entity;
    return entity;
  }

  private async propertyTypes(): Promise<PropertyTypes> {
    return propertyTypesOf(await this.entityMeta());
  }

  /** Describes the entity: schema, properties, links, sample queries. */
  async describe(): Promise<OntologyDescription> {
    await this.entityMeta(); // NotFoundError for unknown entities
    warnIfFlagged('OntologyDescribe');
    return ops.ontologyDescribe(this.transport, {
      projectId: await this.projectId(),
      input: { entities: [this.apiName], includeInferred: true },
    });
  }

  /** Fetches a single record by primary key, or null when absent. */
  async get(pk: string | number): Promise<Row | null> {
    const entity = await this.entityMeta();
    if (!entity.primaryKey) {
      throw new ValidationError(`entity "${this.apiName}" has no primary key — use list() with a where filter`);
    }
    warnIfFlagged('OntologyQuery');
    const result = await ops.ontologyQuery(this.transport, {
      projectId: await this.projectId(),
      input: {
        entity: this.apiName,
        whereJson: JSON.stringify({ [entity.primaryKey]: { eq: String(pk) } }),
        pageSize: 1,
        offset: 0,
      },
    });
    const { rows } = hydrateRows(result, await this.propertyTypes());
    return rows[0] ?? null;
  }

  /** Lists records with optional filter/sort; supports .pages() iteration. */
  list(options: ListOptions = {}): ListCall {
    const pageSize = options.pageSize ?? DEFAULT_PAGE_SIZE;
    return new ListCall(async (pageOffset): Promise<Page> => {
      // Resolve entity metadata first: hydration types come from it, and an
      // unknown entity should fail with NotFoundError before any query runs.
      const propertyTypes = await this.propertyTypes();
      warnIfFlagged('OntologyQuery');
      // OntologyQuery (not OntologyRows) is the list path: it supports
      // filter + sort, and addresses the entity by apiName directly.
      const result = await ops.ontologyQuery(this.transport, {
        projectId: await this.projectId(),
        input: {
          entity: this.apiName,
          whereJson: options.where ? JSON.stringify(options.where) : null,
          sort: options.sort ?? null,
          pageSize,
          offset: pageOffset,
        },
      });
      const { rows, totalCount } = hydrateRows(result, propertyTypes);
      return { rows, totalCount, pageOffset };
    }, pageSize);
  }

  /** Flexible query: text search, joins, grouping, metrics. */
  async query(options: QueryOptions): Promise<Row[]> {
    warnIfFlagged('OntologyQuery');
    const result = await ops.ontologyQuery(this.transport, {
      projectId: await this.projectId(),
      input: { ...options, entity: this.apiName },
    });
    return hydrateRows(result, await this.propertyTypes()).rows;
  }

  /** Aggregates records grouped by properties with metric functions. */
  async aggregate(options: AggregateOptions): Promise<Row[]> {
    const entity = await this.entityMeta();
    warnIfFlagged('OntologyAggregate');
    const result = await ops.ontologyAggregate(this.transport, {
      projectId: await this.projectId(),
      id: entity.id,
      groupBy: options.groupBy,
      metrics: options.metrics ?? [],
      where: options.where ?? null,
      sort: options.sort ?? null,
      pageSize: options.pageSize ?? DEFAULT_PAGE_SIZE,
      pageOffset: 0,
    });
    return hydrateRows(result).rows;
  }

  /** Statistical summary of one property. */
  async stats(property: string, options: { where?: WhereCondition } = {}): Promise<OntologyStatsResult> {
    const entity = await this.entityMeta();
    warnIfFlagged('OntologyStats');
    return ops.ontologyStats(this.transport, {
      projectId: await this.projectId(),
      id: entity.id,
      property,
      where: options.where ?? null,
    });
  }

  /** Embedding-based similarity search over this entity's records. */
  async similar(input: Omit<OntologySimilarInput, 'entityId'>): Promise<OntologySimilarityResult> {
    const entity = await this.entityMeta();
    warnIfFlagged('OntologySimilar');
    return ops.ontologySimilar(this.transport, {
      projectId: await this.projectId(),
      input: { ...input, entityId: entity.id } as OntologySimilarInput,
    });
  }

  /** Follows an outgoing link from one record to its related records. */
  followLink(pk: string | number, linkApiName: string, options: { pageSize?: number } = {}): ListCall {
    const pageSize = options.pageSize ?? DEFAULT_PAGE_SIZE;
    return new ListCall(async (pageOffset): Promise<Page> => {
      const entity = await this.entityMeta();
      warnIfFlagged('OntologyFollowLink');
      const result = await ops.ontologyFollowLink(this.transport, {
        projectId: await this.projectId(),
        entityId: entity.id,
        pk: String(pk),
        linkApiName,
        pageSize,
        pageOffset,
      });
      const { rows, totalCount } = hydrateRows(result);
      return { rows, totalCount, pageOffset };
    }, pageSize);
  }

  /** Follows a link inbound from another entity's records to this record. */
  followIncomingLink(
    pk: string | number,
    sourceEntityApiName: string,
    linkApiName: string,
    options: { pageSize?: number } = {},
  ): ListCall {
    const pageSize = options.pageSize ?? DEFAULT_PAGE_SIZE;
    return new ListCall(async (pageOffset): Promise<Page> => {
      const entity = await this.entityMeta();
      warnIfFlagged('OntologyEntities');
      const sourceEntities = await ops.ontologyEntities(this.transport, { projectId: await this.projectId() });
      const source = sourceEntities.find(e => e.apiName === sourceEntityApiName);
      if (!source) throw new NotFoundError(`ontology entity "${sourceEntityApiName}" not found in this project`);
      warnIfFlagged('OntologyFollowIncomingLink');
      const result = await ops.ontologyFollowIncomingLink(this.transport, {
        projectId: await this.projectId(),
        entityId: entity.id,
        pk: String(pk),
        sourceEntityId: source.id,
        linkApiName,
        pageSize,
        pageOffset,
      });
      const { rows, totalCount } = hydrateRows(result);
      return { rows, totalCount, pageOffset };
    }, pageSize);
  }

  /** Lists the entity's fast lookups. */
  async fastLookups(): Promise<OntologyFastLookup[]> {
    const entity = await this.entityMeta();
    warnIfFlagged('OntologyFastLookups');
    return ops.ontologyFastLookups(this.transport, {
      projectId: await this.projectId(),
      entityId: entity.id,
    });
  }

  /** Inserts one record. Values are field name/value pairs. */
  async create(values: Record<string, unknown>): Promise<void> {
    const entity = await this.entityMeta();
    warnIfFlagged('OntologyAddRow');
    await ops.ontologyAddRow(this.transport, {
      projectId: await this.projectId(),
      entityId: entity.id,
      values: toRecordInputs(values),
    });
  }

  /** Inserts many records; idempotencyKey makes safe retries possible. */
  async createMany(
    rows: Array<Record<string, unknown>>,
    options: { idempotencyKey?: string } = {},
  ): Promise<OntologyAddRowsResult> {
    const entity = await this.entityMeta();
    warnIfFlagged('OntologyAddRows');
    return ops.ontologyAddRows(this.transport, {
      projectId: await this.projectId(),
      entityId: entity.id,
      rows: rows.map(row => ({ values: toRecordInputs(row) }) as OntologyCreateRowInput),
      idempotencyKey: options.idempotencyKey ?? null,
    });
  }

  /** Updates one record identified by primary key. */
  async update(pk: string | number, values: Record<string, unknown>): Promise<void> {
    const entity = await this.entityMeta();
    if (!entity.primaryKey) {
      throw new ValidationError(`entity "${this.apiName}" has no primary key — updates are not supported`);
    }
    warnIfFlagged('OntologyUpdateRow');
    await ops.ontologyUpdateRow(this.transport, {
      projectId: await this.projectId(),
      entityId: entity.id,
      values: toRecordInputs({ ...values, [entity.primaryKey]: String(pk) }),
      updatedColumns: Object.keys(values),
    });
  }

  /** Deletes one record identified by primary key. */
  async delete(pk: string | number): Promise<void> {
    const entity = await this.entityMeta();
    if (!entity.primaryKey) {
      throw new ValidationError(`entity "${this.apiName}" has no primary key — deletes are not supported`);
    }
    warnIfFlagged('OntologyDeleteRow');
    await ops.ontologyDeleteRow(this.transport, {
      projectId: await this.projectId(),
      entityId: entity.id,
      values: toRecordInputs({ [entity.primaryKey]: String(pk) }),
    });
  }
}

function toRecordInputs(values: Record<string, unknown>): RecordInput[] {
  return Object.entries(values).map(([key, value]) => ({
    Key: key,
    Value: value === null || value === undefined ? '' : typeof value === 'object' ? JSON.stringify(value) : String(value),
  }));
}
