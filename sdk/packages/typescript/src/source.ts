import type { Transport } from './transport.js';
import * as ops from './generated/operations.js';
import type {
  Column, PlatformSource, SourceObject, SourceObjectRefInput,
  WhereCondition, SortCondition,
} from './generated/types.js';
import { hydrateRows } from './hydrate.js';
import { ListCall, type Page } from './pagination.js';
import { warnIfFlagged } from './manifest-check.js';

const DEFAULT_PAGE_SIZE = 100;

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
export class SourceHandle {
  private readonly transport: Transport;
  private readonly projectId: () => Promise<string>;
  private readonly sourceId: string;

  constructor(transport: Transport, projectId: () => Promise<string>, sourceId: string) {
    this.transport = transport;
    this.projectId = projectId;
    this.sourceId = sourceId;
  }

  /** Lists browsable objects (schemas, tables, collections...). */
  async objects(options: { parent?: SourceObjectRefInput; pageSize?: number; pageOffset?: number } = {}): Promise<SourceObject[]> {
    warnIfFlagged('PlatformSourceObjects');
    return ops.platformSourceObjects(this.transport, {
      projectId: await this.projectId(),
      sourceId: this.sourceId,
      parent: options.parent ?? null,
      kinds: null,
      pageSize: options.pageSize ?? null,
      pageOffset: options.pageOffset ?? null,
    });
  }

  /** Lists the columns of one object (table/collection). */
  async columns(ref: SourceObjectRefInput): Promise<Column[]> {
    warnIfFlagged('PlatformSourceColumns');
    return ops.platformSourceColumns(this.transport, {
      projectId: await this.projectId(),
      sourceId: this.sourceId,
      ref,
    });
  }

  /** Reads rows from one object; supports .pages() iteration. */
  rows(ref: SourceObjectRefInput, options: SourceRowsOptions = {}): ListCall {
    const pageSize = options.pageSize ?? DEFAULT_PAGE_SIZE;
    return new ListCall(async (pageOffset): Promise<Page> => {
      warnIfFlagged('PlatformSourceRows');
      const result = await ops.platformSourceRows(this.transport, {
        projectId: await this.projectId(),
        sourceId: this.sourceId,
        ref,
        where: options.where ?? null,
        sort: options.sort ?? null,
        pageSize,
        pageOffset,
      });
      const { rows, totalCount } = hydrateRows(result);
      return { rows, totalCount, pageOffset };
    }, pageSize);
  }
}

/** Lists the project's sources. */
export async function listSources(transport: Transport, projectId: string): Promise<PlatformSource[]> {
  warnIfFlagged('ProjectSources');
  return ops.projectSources(transport, { projectId });
}
