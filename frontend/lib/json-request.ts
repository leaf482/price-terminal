// Transport only: feature modules own status/error mapping and whether a
// successful response needs a body. Some mutations intentionally accept an
// empty success body, while others require validated JSON.
export function jsonRequest(url: string, method: 'POST' | 'PUT' | 'PATCH', body: unknown, timeoutMs = 10000, signal?: AbortSignal): Promise<Response> {
    return fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: signal || AbortSignal.timeout(timeoutMs),
    });
}
