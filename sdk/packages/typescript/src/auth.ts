import { execFile } from 'node:child_process';
import { AuthError, CliCredentialsError } from './errors.js';

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
export function apiKeyProvider(apiKey: string): CredentialProvider {
  return {
    token: async () => apiKey,
  };
}

/** Static OIDC token or caller-managed token callback. */
export function tokenProvider(source: string | (() => Promise<string>)): CredentialProvider {
  if (typeof source === 'string') {
    return { token: async () => source };
  }
  return { token: source };
}

interface PrintTokenOutput {
  access_token: string;
  expires_at?: string;
  host?: string;
  org_id?: string;
  project_id?: string;
}

const CLI_REFRESH_SKEW_MS = 60_000;

/**
 * CLI credentials: exec `whodb auth print-token --format json` and cache the
 * token until shortly before its expiry — the gcloud-ADC pattern for local
 * development. Requires the whodb CLI on PATH and a prior `whodb login`.
 */
export function cliProvider(command = 'whodb'): CredentialProvider {
  let cached: PrintTokenOutput | null = null;

  const exec = (): Promise<PrintTokenOutput> =>
    new Promise((resolvePromise, rejectPromise) => {
      execFile(command, ['auth', 'print-token', '--format', 'json'], { timeout: 15_000 }, (error, stdout, stderr) => {
        if (error) {
          if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
            rejectPromise(new CliCredentialsError(
              `whodb CLI not found — install it or set WHODB_API_KEY`,
            ));
            return;
          }
          rejectPromise(new CliCredentialsError(
            `whodb auth print-token failed: ${stderr.trim() || error.message}`,
          ));
          return;
        }
        try {
          resolvePromise(JSON.parse(stdout) as PrintTokenOutput);
        } catch {
          rejectPromise(new CliCredentialsError('whodb auth print-token returned invalid JSON'));
        }
      });
    });

  const isFresh = (entry: PrintTokenOutput): boolean => {
    if (!entry.expires_at) return false; // no expiry info — re-exec every call
    return new Date(entry.expires_at).getTime() - Date.now() > CLI_REFRESH_SKEW_MS;
  };

  return {
    async token() {
      if (cached && isFresh(cached)) return cached.access_token;
      cached = await exec();
      if (!cached.access_token) throw new AuthError('whodb CLI returned an empty access token');
      return cached.access_token;
    },
    async refresh() {
      cached = null;
    },
    async defaults() {
      if (!cached) cached = await exec();
      return { host: cached.host, orgId: cached.org_id, projectId: cached.project_id };
    },
  };
}
