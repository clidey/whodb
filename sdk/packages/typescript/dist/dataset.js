import * as ops from './generated/operations.js';
import { hydrateRows } from './hydrate.js';
import { warnIfFlagged } from './manifest-check.js';
import { NotFoundError } from './errors.js';
/**
 * DatasetHandle is the `whodb.dataset("weekly_kpis")` facade, addressed by
 * dataset name.
 */
export class DatasetHandle {
    transport;
    projectId;
    name;
    datasetCache = null;
    constructor(transport, projectId, name) {
        this.transport = transport;
        this.projectId = projectId;
        this.name = name;
    }
    /** Resolves and caches the dataset metadata backing this handle. */
    async meta() {
        if (this.datasetCache)
            return this.datasetCache;
        warnIfFlagged('ProjectDatasets');
        const datasets = await ops.projectDatasets(this.transport, { projectId: await this.projectId() });
        const dataset = datasets.find(d => d.name === this.name);
        if (!dataset)
            throw new NotFoundError(`dataset "${this.name}" not found in this project`);
        this.datasetCache = dataset;
        return dataset;
    }
    /** Queries dataset rows with optional filter/sort/paging. */
    async rows(options = {}) {
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
