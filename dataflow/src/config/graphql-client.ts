/**
 * Apollo Client configuration for WhoDB Core GraphQL API.
 *
 * Link chain: errorLink → authLink → uploadOrHttpLink
 *
 * - uploadOrHttpLink: POST to /api/query as JSON or GraphQL multipart upload
 * - authLink: injects Authorization Bearer header from auth-store
 * - errorLink: logs network errors
 *
 * Shared GraphQL client wiring for DataFlow.
 */

import { ApolloClient, ApolloLink, createHttpLink, InMemoryCache, Observable } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import { onError } from '@apollo/client/link/error';
import { print } from 'graphql';
import { addAuthHeader } from './auth-headers';
import {
  getBootstrapDescriptor,
  triggerRebootstrap,
  triggerStandaloneUnauthorized,
} from './auth-store';

const RETRY_HEADER = 'X-WhoDB-Retry-Attempt';
const GRAPHQL_ENDPOINT = '/api/query';

interface UploadEntry {
  path: string;
  file: Blob;
}

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
  uri: GRAPHQL_ENDPOINT,
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

const uploadLink = new ApolloLink((operation) => new Observable((observer) => {
  const controller = new AbortController();

  void (async () => {
    try {
      const formData = createGraphQLMultipartForm({
        query: print(operation.query),
        variables: operation.variables,
        operationName: operation.operationName,
      });
      const response = await authFetch(GRAPHQL_ENDPOINT, {
        method: 'POST',
        credentials: 'include',
        headers: removeContentTypeHeader(operation.getContext().headers),
        body: formData,
        signal: controller.signal,
      });
      const payload = await readGraphQLResponse(response);

      if (!response.ok && !payload.errors) {
        throw new Error(`GraphQL upload failed (${response.status})`);
      }

      observer.next(payload);
      observer.complete();
    } catch (error) {
      if (!controller.signal.aborted) observer.error(error);
    }
  })();

  return () => controller.abort();
}));

const uploadOrHttpLink = ApolloLink.split(
  (operation) => hasUploadVariable(operation.variables),
  uploadLink,
  httpLink,
);

export const graphqlClient = new ApolloClient({
  link: errorLink.concat(authLink.concat(uploadOrHttpLink)),
  cache: new InMemoryCache(),
  defaultOptions: {
    query: { fetchPolicy: 'no-cache' },
    watchQuery: { fetchPolicy: 'no-cache' },
    mutate: { fetchPolicy: 'no-cache' },
  },
});

/** Returns whether GraphQL variables contain a browser file/blob upload. */
export function hasUploadVariable(value: unknown): boolean {
  if (isUploadFile(value)) return true;
  if (Array.isArray(value)) return value.some(hasUploadVariable);
  if (!value || typeof value !== 'object') return false;
  return Object.values(value).some(hasUploadVariable);
}

/** Builds a GraphQL multipart request body for operations containing uploads. */
export function createGraphQLMultipartForm(operation: {
  query: string;
  variables?: Record<string, unknown>;
  operationName?: string;
}): FormData {
  const files: UploadEntry[] = [];
  const variables = extractUploadVariables(operation.variables ?? {}, 'variables', files);
  const formData = new FormData();

  formData.append('operations', JSON.stringify({
    query: operation.query,
    variables,
    operationName: operation.operationName,
  }));
  formData.append('map', JSON.stringify(Object.fromEntries(
    files.map((entry, index) => [String(index), [entry.path]]),
  )));
  files.forEach((entry, index) => {
    formData.append(String(index), entry.file);
  });

  return formData;
}

async function readGraphQLResponse(response: Response): Promise<Record<string, unknown>> {
  const text = await response.text();
  if (!text) return {};
  return JSON.parse(text);
}

function extractUploadVariables(value: unknown, path: string, files: UploadEntry[]): unknown {
  if (isUploadFile(value)) {
    files.push({ path, file: value });
    return null;
  }

  if (Array.isArray(value)) {
    return value.map((item, index) => extractUploadVariables(item, `${path}.${index}`, files));
  }

  if (!value || typeof value !== 'object') return value;

  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>).map(([key, child]) => [
      key,
      extractUploadVariables(child, `${path}.${key}`, files),
    ]),
  );
}

function isUploadFile(value: unknown): value is Blob {
  return typeof Blob !== 'undefined' && value instanceof Blob;
}

function removeContentTypeHeader(headers: HeadersInit | undefined): Record<string, string> {
  const normalized = normalizeHeaders(headers);
  return Object.fromEntries(
    Object.entries(normalized).filter(([key]) => key.toLowerCase() !== 'content-type'),
  );
}

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
