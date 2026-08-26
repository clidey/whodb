/**
 * Transport executes one platform operation and returns the GraphQL `data`
 * object. Implementations: HttpTransport (default, GraphQL over HTTP) and
 * IpcTransport (inside the WhoDB Functions runtime). The generated core and
 * the facade never speak HTTP directly — everything routes through this seam.
 */
export interface Transport {
    execute(operationName: string, document: string, variables: Record<string, unknown>): Promise<Record<string, unknown>>;
}
