import type { Transport } from './transport.js';
import type { CredentialProvider } from './auth.js';
/** Options for the default GraphQL-over-HTTP transport. */
export interface HttpTransportOptions {
    host: string;
    credentials: CredentialProvider;
    orgId?: string;
    projectId?: string;
}
/**
 * HttpTransport posts operations to `<host>/api/query` with bearer
 * credentials and workspace headers. It retries once on a transient 5xx and
 * once after refreshing credentials on a 401.
 */
export declare class HttpTransport implements Transport {
    private readonly options;
    constructor(options: HttpTransportOptions);
    /** Sets the workspace scope headers used on subsequent requests. */
    setWorkspace(orgId: string | undefined, projectId: string | undefined): void;
    execute(operationName: string, document: string, variables: Record<string, unknown>): Promise<Record<string, unknown>>;
    private post;
}
