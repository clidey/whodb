/**
 * Apollo Client configuration for WhoDB Core GraphQL API.
 *
 * Link chain: errorLink → authLink → httpLink
 *
 * - httpLink: POST to /api/query (proxied to Core in dev, same-origin in prod)
 * - authLink: injects Authorization Bearer header from auth-store
 * - errorLink: logs network errors
 *
 * Shared GraphQL client wiring for DataFlow.
 */

import { ApolloClient, createHttpLink, InMemoryCache } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import { onError } from '@apollo/client/link/error';
import { addAuthHeader } from './auth-headers';
import {
  getBootstrapDescriptor,
  triggerRebootstrap,
  triggerStandaloneUnauthorized,
} from './auth-store';

const RETRY_HEADER = 'X-WhoDB-Retry-Attempt';

/** Fetch wrapper that handles expired server-side auth sessions. */
export async function authFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const response = await fetch(input, init);
  if (response.status !== 401) return response;

  const retryAttempt = readHeader(init?.headers, RETRY_HEADER);
  if (retryAttempt === '1') return response;

  if (!getBootstrapDescriptor()) {
    await triggerStandaloneUnauthorized();
    return response;
  }

  const rebootstrapSucceeded = await triggerRebootstrap();
  if (!rebootstrapSucceeded) return response;

  const existingHeaders = normalizeHeaders(init?.headers);
  const databaseOverride = existingHeaders['X-WhoDB-Database'] ?? existingHeaders['x-whodb-database'];

  return fetch(input, {
    ...init,
    headers: addAuthHeader({
      ...existingHeaders,
      [RETRY_HEADER]: '1',
    }, databaseOverride),
  });
}

const httpLink = createHttpLink({
  uri: '/api/query',
  credentials: 'include',
  fetch: authFetch,
});

const authLink = setContext((_, previousContext) => ({
  headers: addAuthHeader(
    previousContext.headers,
    previousContext.database,
  ),
}));

const errorLink = onError(({ networkError }) => {
  if (networkError) {
    const status = 'statusCode' in networkError ? networkError.statusCode : undefined;
    console.error(`GraphQL network error (${status ?? 'unknown'}):`, networkError);
  }
});

export const graphqlClient = new ApolloClient({
  link: errorLink.concat(authLink.concat(httpLink)),
  cache: new InMemoryCache(),
  defaultOptions: {
    query: { fetchPolicy: 'no-cache' },
    watchQuery: { fetchPolicy: 'no-cache' },
    mutate: { fetchPolicy: 'no-cache' },
  },
});

function normalizeHeaders(headers: HeadersInit | undefined): Record<string, string> {
  if (!headers) return {};
  if (headers instanceof Headers) {
    return Object.fromEntries(headers.entries());
  }
  if (Array.isArray(headers)) {
    return Object.fromEntries(headers);
  }
  return headers;
}

function readHeader(headers: HeadersInit | undefined, key: string): string | undefined {
  const normalized = normalizeHeaders(headers);
  return normalized[key] ?? normalized[key.toLowerCase()];
}
