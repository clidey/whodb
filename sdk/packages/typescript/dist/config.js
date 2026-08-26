import { apiKeyProvider, cliProvider, tokenProvider } from './auth.js';
export const DEFAULT_HOST = 'https://app.whodb.com';
/**
 * Applies the credential precedence from SDK_DESIGN.md §5.3:
 * constructor args → WHODB_API_KEY env → CLI helper (local-dev default).
 * Workspace/host: explicit args → WHODB_ORG/WHODB_PROJECT/WHODB_HOST env →
 * CLI helper's saved defaults (only when using CLI credentials).
 */
export function resolveConfig(config, env = process.env) {
    let credentials;
    let usingCliCredentials = false;
    let usingApiKey = false;
    if (config.credentials) {
        credentials = config.credentials;
    }
    else if (config.apiKey) {
        credentials = apiKeyProvider(config.apiKey);
        usingApiKey = true;
    }
    else if (config.token) {
        credentials = tokenProvider(config.token);
    }
    else if (env.WHODB_API_KEY) {
        credentials = apiKeyProvider(env.WHODB_API_KEY);
        usingApiKey = true;
    }
    else {
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
