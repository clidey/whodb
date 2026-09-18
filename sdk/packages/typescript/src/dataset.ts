import type { Transport } from './transport.js';
import * as ops from './generated/operations.js';
import type { Dataset } from './generated/types.js';
import { hydrateRows, type Row } from './hydrate.js';
import { warnIfFlagged } from './manifest-check.js';
import { NotFoundError } from './errors.js';

/** Options for dataset row reads. */
export interface DatasetRowsOptions {
  pageSize?: number;
  pageOffset?: number;
}

/**
 * DatasetHandle is the `whodb.dataset("weekly_kpis")` facade, addressed by
 * dataset name.
 */
export class DatasetHandle {
  private readonly transport: Transport;
  private readonly projectId: () => Promise<string>;
  private readonly name: string;
  private datasetCache: Dataset | null = null;

  constructor(transport: Transport, projectId: () => Promise<string>, name: string) {
    this.transport = transport;
    this.projectId = projectId;
    this.name = name;
  }

  /** Resolves and caches the dataset metadata backing this handle. */
  async meta(): Promise<Dataset> {
    if (this.datasetCache) return this.datasetCache;
    warnIfFlagged('ProjectDatasets');
    const datasets = await ops.projectDatasets(this.transport, { projectId: await this.projectId() });
    const dataset = datasets.find(d => d.name === this.name);
    if (!dataset) throw new NotFoundError(`dataset "${this.name}" not found in this project`);
    this.datasetCache = dataset;
    return dataset;
  }

  /** Queries dataset rows with optional filter/sort/paging. */
  async rows(options: DatasetRowsOptions = {}): Promise<{ rows: Row[]; totalCount: number | null }> {
    const dataset = await this.meta();
    warnIfFlagged('QueryDataset');
    const result = await ops.queryDataset(this.transport, {
      input: {
        projectId: await this.projectId(),
        datasetId: dataset.id,
        pageSize: options.pageSize ?? 100,
        pageOffset: options.pageOffset ?? 0,
      },
    });
    return hydrateRows(result);
  }
}
