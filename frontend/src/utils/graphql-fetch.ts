const JSON_REQUEST_LIMIT = 1 << 20;

/** Sends large GraphQL JSON envelopes through the existing multipart transport. */
export const graphqlFetch: typeof fetch = (input, init) => {
    if (init?.method?.toUpperCase() !== 'POST' || typeof init.body !== 'string') {
        return fetch(input, init);
    }
    const headers = new Headers(init.headers);
    const contentType = headers.get('content-type')?.split(';')[0].trim().toLowerCase();
    if (contentType !== 'application/json' || new Blob([init.body]).size <= JSON_REQUEST_LIMIT) {
        return fetch(input, init);
    }
    const body = new FormData();
    body.set('operations', init.body);
    body.set('map', '{}');
    headers.delete('content-type');
    headers.delete('content-length');
    return fetch(input, {...init, headers, body});
};
