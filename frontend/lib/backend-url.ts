export function backendURL(value: string | undefined): string {
    const input = value === undefined ? 'http://127.0.0.1:8080' : value;
    try {
        const url = new URL(input);
        if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash || url.pathname !== '/') throw new Error();
        return url.origin;
    } catch { throw new Error('Invalid BACKEND_URL: expected an http(s) origin without credentials, path, query or fragment.'); }
}
