import { apiKeyProvider, cliProvider, tokenProvider, type CredentialProvider } from './auth.js';

/** Constructor options for the WhoDB client. */
export interface WhoDBConfig {
  /** Platform API key (whodb_sk_...). Highest-precedence credential. */
  apiKey?: string;
  /** Raw OIDC access token or an async token callback. */
  token?: string | (() => Promise<string>);
  /** Fully custom credential provider. */
  credentials?: CredentialProvider;
  /** Organization slug or ID. */
  org?: string;
  /** Project slug or ID. */
  project?: string;
  /** Platform host. Defaults to https://app.whodb.com. */
  host?: string;
}

export const DEFAULT_HOST = 'https://app.whodb.com';

/** Resolved client configuration after applying the precedence rules. */
export interface ResolvedConfig {
  credentials: CredentialProvider;
  /** True when credentials came from the CLI helper (workspace defaults apply). */
  usingCliCredentials: boolean;
  /** True for API keys: the key carries its org, and the platform
   * auto-resolves the project when the key has exactly one grant — so
   * org/project config is optional. */
  usingApiKey: boolean;
  host?: string;
  org?: string;
  project?: string;
}

/**
 * Applies the credential precedence from SDK_DESIGN.md §5.3:
 * constructor args → WHODB_API_KEY env → CLI helper (local-dev default).
 * Workspace/host: explicit args → WHODB_ORG/WHODB_PROJECT/WHODB_HOST env →
 * CLI helper's saved defaults (only when using CLI credentials).
 */
export function resolveConfig(config: WhoDBConfig, env: NodeJS.ProcessEnv = process.env): ResolvedConfig {
  let credentials: CredentialProvider;
  let usingCliCredentials = false;
  let usingApiKey = false;
  if (config.credentials) {
    credentials = config.credentials;
  } else if (config.apiKey) {
    credentials = apiKeyProvider(config.apiKey);
    usingApiKey = true;
  } else if (config.token) {
    credentials = tokenProvider(config.token);
  } else if (env.WHODB_API_KEY) {
    credentials = apiKeyProvider(env.WHODB_API_KEY);
    usingApiKey = true;
  } else {
    credentials = cliProvider();
    usingCliCredentials = true;
  }
  return {
    credentials,
    usingCliCredentials,
    usingApiKey,
    host: config.host ?? env.WHODB_HOST,
    org: config.org ?? env.WHODB_ORG,
    project: config.project ?? env.WHODB_PROJECT,
  };
}
