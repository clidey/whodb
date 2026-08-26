import type { Transport } from './transport.js';
import type { CredentialProvider } from './auth.js';
import { mapGraphQLErrors, AuthError, PlatformError } from './errors.js';
import { SDK_VERSION } from './version.js';

/** Options for the default GraphQL-over-HTTP transport. */
export interface HttpTransportOptions {
  host: string;
  credentials: CredentialProvider;
  orgId?: string;
  projectId?: string;
}

const RETRYABLE_STATUS = new Set([502, 503, 504]);

/**
 * HttpTransport posts operations to `<host>/api/query` with bearer
 * credentials and workspace headers. It retries once on a transient 5xx and
 * once after refreshing credentials on a 401.
 */
export class HttpTransport implements Transport {
  private readonly options: HttpTransportOptions;

  constructor(options: HttpTransportOptions) {
    this.options = options;
  }

  /** Sets the workspace scope headers used on subsequent requests. */
  setWorkspace(orgId: string | undefined, projectId: string | undefined): void {
    this.options.orgId = orgId;
    this.options.projectId = projectId;
  }

  async execute(
    operationName: string,
    document: string,
    variables: Record<string, unknown>,
  ): Promise<Record<string, unknown>> {
    let response = await this.post(document, variables);
    if (response.status === 401 && this.options.credentials.refresh) {
      await this.options.credentials.refresh();
      response = await this.post(document, variables);
    } else if (RETRYABLE_STATUS.has(response.status)) {
      response = await this.post(document, variables);
    }
    if (response.status === 401) {
      throw new AuthError('authentication failed — check your API key or run: whodb login');
    }
    if (!response.ok) {
      throw new PlatformError(`platform request failed with HTTP ${response.status}`, `HTTP_${response.status}`);
    }
    const payload = (await response.json()) as {
      data?: Record<string, unknown>;
      errors?: Array<{ message: string; extensions?: { code?: string } }>;
    };
    if (payload.errors && payload.errors.length > 0) {
      throw mapGraphQLErrors(payload.errors);
    }
    if (!payload.data) {
      throw new PlatformError(`empty response for ${operationName}`, 'EMPTY_RESPONSE');
    }
    return payload.data;
  }

  private async post(document: string, variables: Record<string, unknown>): Promise<Response> {
    const token = await this.options.credentials.token();
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      'User-Agent': `clidey-whodb-ts/${SDK_VERSION}`,
    };
    if (this.options.orgId) headers['X-Whodb-Org-Id'] = this.options.orgId;
    if (this.options.projectId) headers['X-Whodb-Project-Id'] = this.options.projectId;
    return fetch(`${this.options.host.replace(/\/$/, '')}/api/query`, {
      method: 'POST',
      headers,
      body: JSON.stringify({ query: document, variables }),
    });
  }
}
