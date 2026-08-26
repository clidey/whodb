/**
 * CredentialProvider yields a bearer credential for platform requests.
 * refresh() is invoked once after a 401 before the request is retried.
 */
export interface CredentialProvider {
    token(): Promise<string>;
    refresh?(): Promise<void>;
    /** Workspace defaults carried by the credential source, if any. */
    defaults?(): Promise<WorkspaceDefaults>;
}
/** Workspace defaults resolved from a credential source (CLI helper). */
export interface WorkspaceDefaults {
    host?: string;
    orgId?: string;
    projectId?: string;
}
/** Static API-key credentials (production/headless usage). */
export declare function apiKeyProvider(apiKey: string): CredentialProvider;
/** Static OIDC token or caller-managed token callback. */
export declare function tokenProvider(source: string | (() => Promise<string>)): CredentialProvider;
/**
 * CLI credentials: exec `whodb auth print-token --format json` and cache the
 * token until shortly before its expiry — the gcloud-ADC pattern for local
 * development. Requires the whodb CLI on PATH and a prior `whodb login`.
 */
export declare function cliProvider(command?: string): CredentialProvider;
