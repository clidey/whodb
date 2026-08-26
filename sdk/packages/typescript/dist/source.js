import * as ops from './generated/operations.js';
import { hydrateRows } from './hydrate.js';
import { ListCall } from './pagination.js';
import { warnIfFlagged } from './manifest-check.js';
const DEFAULT_PAGE_SIZE = 100;
/**
 * SourceHandle is the `whodb.source("src_...")` facade: browse and read a
 * connected data source by ID.
 */
export class SourceHandle {
    transport;
    projectId;
    sourceId;
    constructor(transport, projectId, sourceId) {
        this.transport = transport;
        this.projectId = projectId;
        this.sourceId = sourceId;
    }
    /** Lists browsable objects (schemas, tables, collections...). */
    async objects(options = {}) {
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
    async columns(ref) {
        warnIfFlagged('PlatformSourceColumns');
        return ops.platformSourceColumns(this.transport, {
            projectId: await this.projectId(),
            sourceId: this.sourceId,
            ref,
        });
    }
    /** Reads rows from one object; supports .pages() iteration. */
    rows(ref, options = {}) {
        const pageSize = options.pageSize ?? DEFAULT_PAGE_SIZE;
        return new ListCall(async (pageOffset) => {
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
export async function listSources(transport, projectId) {
    warnIfFlagged('ProjectSources');
    return ops.projectSources(transport, { projectId });
}
