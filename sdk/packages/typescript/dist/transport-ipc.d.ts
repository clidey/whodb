import type { Transport } from './transport.js';
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
export declare class IpcTransport implements Transport {
    private readonly address;
    private readonly jobId;
    private readonly token;
    private entitiesCache;
    constructor(options?: {
        address?: string;
        jobId?: string;
        token?: string;
    });
    execute(operationName: string, _document: string, variables: Record<string, unknown>): Promise<Record<string, unknown>>;
    private post;
    private entities;
    private entityName;
    private entityPk;
    private recordInputsToData;
    private readonly handlers;
}
