import { request as httpRequest } from 'node:http';
import { NotFoundError, PlatformError, TransportCapabilityError } from './errors.js';
/**
 * IpcTransport runs the same facades inside the WhoDB Functions runtime,
 * over the in-container IPC server (unix socket in Docker, TCP in K8s).
 *
 * Two impedance differences are absorbed here:
 * - GraphQL operations address entities by ID; IPC endpoints address them by
 *   apiName. IDs resolve via a cached /entities call.
 * - GraphQL write mutations carry RecordInput pairs; IPC takes plain objects.
 *
 * Operations outside the ontology surface (datasets, sources, files,
 * workspace) throw TransportCapabilityError in v1.
 */
export class IpcTransport {
    address;
    jobId;
    token;
    entitiesCache = null;
    constructor(options = {}) {
        this.address = options.address ?? process.env.WHODB_IPC_ADDRESS ?? '';
        this.jobId = options.jobId ?? process.env.WHODB_JOB_ID ?? '';
        this.token = options.token ?? process.env.WHODB_IPC_TOKEN ?? '';
    }
    async execute(operationName, _document, variables) {
        const handler = this.handlers[operationName];
        if (!handler) {
            throw new TransportCapabilityError(`${operationName} is not available inside the function runtime in v1`);
        }
        return { [operationName]: await handler(variables) };
    }
    post(path, body) {
        return new Promise((resolve, reject) => {
            const payload = JSON.stringify(body ?? {});
            const isSocket = this.address.startsWith('/');
            const options = {
                method: 'POST',
                path,
                headers: {
                    'Content-Type': 'application/json',
                    'Content-Length': Buffer.byteLength(payload),
                    'X-Job-ID': this.jobId,
                    Authorization: this.token,
                },
                ...(isSocket
                    ? { socketPath: this.address }
                    : {
                        host: this.address.split(':')[0],
                        port: Number(this.address.split(':')[1] ?? 80),
                    }),
            };
            const req = httpRequest(options, (res) => {
                const chunks = [];
                res.on('data', (chunk) => chunks.push(chunk));
                res.on('end', () => {
                    const text = Buffer.concat(chunks).toString('utf8');
                    if ((res.statusCode ?? 500) >= 400) {
                        reject(new PlatformError(`IPC request ${path} failed with HTTP ${res.statusCode}`, `IPC_${res.statusCode}`));
                        return;
                    }
                    try {
                        resolve(text ? JSON.parse(text) : null);
                    }
                    catch {
                        reject(new PlatformError(`IPC request ${path} returned invalid JSON`, 'IPC_INVALID_JSON'));
                    }
                });
            });
            req.on('error', reject);
            req.end(payload);
        });
    }
    async entities() {
        this.entitiesCache ??= (await this.post('/entities', {})) ?? [];
        return this.entitiesCache;
    }
    async entityName(entityId) {
        const entity = (await this.entities()).find(e => e.id === entityId);
        if (!entity)
            throw new NotFoundError(`ontology entity ${entityId} not found in this function's scope`);
        return String(entity.apiName ?? '');
    }
    async entityPk(entityId) {
        const entity = (await this.entities()).find(e => e.id === entityId);
        return String(entity?.primaryKey ?? '');
    }
    recordInputsToData(values) {
        const data = {};
        for (const record of values ?? []) {
            data[record.Key] = record.Value;
        }
        return data;
    }
    handlers = {
        OntologyEntities: async () => this.entities(),
        OntologyQuery: async (variables) => {
            const input = { ...(variables.input ?? {}) };
            const body = {};
            for (const [key, value] of Object.entries(input)) {
                if (value !== null && value !== undefined)
                    body[key] = value;
            }
            const whereJson = body.whereJson;
            delete body.whereJson;
            if (whereJson)
                body.where = JSON.parse(whereJson);
            return this.post('/query', body);
        },
        OntologyDescribe: async (variables) => this.post('/describe', variables.input ?? {}),
        OntologyStats: async (variables) => this.post('/stats', {
            entity: await this.entityName(String(variables.id)),
            property: variables.property,
            where: variables.where,
        }),
        OntologySimilar: async (variables) => {
            const input = variables.input ?? {};
            return this.post('/similar', {
                entity: await this.entityName(String(input.entityId)),
                rowId: input.rowId,
                topK: input.topK,
                properties: input.properties,
                where: input.where,
            });
        },
        OntologyAggregate: async (variables) => this.post('/query', {
            entity: await this.entityName(String(variables.id)),
            where: variables.where,
            groupBy: variables.groupBy,
            metrics: variables.metrics,
            sort: variables.sort,
            pageSize: variables.pageSize,
            offset: variables.pageOffset,
        }),
        OntologyAddRow: async (variables) => {
            await this.post('/create', {
                entity: await this.entityName(String(variables.entityId)),
                data: this.recordInputsToData(variables.values),
            });
            return { Status: true };
        },
        OntologyAddRows: async (variables) => {
            const ids = (await this.post('/create_many', {
                entity: await this.entityName(String(variables.entityId)),
                rows: (variables.rows ?? []).map(row => this.recordInputsToData(row.values)),
                idempotencyKey: variables.idempotencyKey,
            })) ?? [];
            return { inserted: ids.length, ids };
        },
        OntologyUpdateRow: async (variables) => {
            const entityId = String(variables.entityId);
            const data = this.recordInputsToData(variables.values);
            const pkKey = await this.entityPk(entityId);
            const pk = pkKey ? String(data[pkKey] ?? '') : '';
            if (pkKey)
                delete data[pkKey];
            await this.post('/update', { entity: await this.entityName(entityId), pk, data });
            return { Status: true };
        },
        OntologyDeleteRow: async (variables) => {
            const entityId = String(variables.entityId);
            const data = this.recordInputsToData(variables.values);
            const pkKey = await this.entityPk(entityId);
            const pk = pkKey ? String(data[pkKey] ?? '') : String(Object.values(data)[0] ?? '');
            await this.post('/delete', { entity: await this.entityName(entityId), pk });
            return { Status: true };
        },
        OntologyFollowLink: async (variables) => this.post('/follow_link', {
            entity: await this.entityName(String(variables.entityId)),
            pk: variables.pk,
            link: variables.linkApiName,
            pageSize: variables.pageSize,
            offset: variables.pageOffset,
        }),
        OntologyFollowIncomingLink: async (variables) => this.post('/follow_incoming_link', {
            entity: await this.entityName(String(variables.entityId)),
            pk: variables.pk,
            sourceEntity: await this.entityName(String(variables.sourceEntityId)),
            link: variables.linkApiName,
            pageSize: variables.pageSize,
            offset: variables.pageOffset,
        }),
    };
}
