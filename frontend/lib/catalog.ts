export type Retailer = {
    id: string;
    name: string;
};
export type Listing = {
    id: string;
    product_id: string;
    retailer_id: string;
    url: string;
    retailer_product_id?: string;
	tracking_enabled?: boolean;
};
function records(value: unknown, fields: string[]): Record<string, unknown>[] {
    if (!Array.isArray(value))
        throw new Error('Invalid catalog response');
    return value.map(item => { if (!item || typeof item !== 'object' || fields.some(f => typeof item[f] !== 'string'))
        throw new Error('Invalid catalog record'); return item; });
}
export function parseRetailers(value: unknown): Retailer[] { return records(value, ['id', 'name']) as Retailer[]; }
export function parseListings(value: unknown): Listing[] { const result = records(value, ['id', 'product_id', 'retailer_id', 'url']); for (const v of result) {
    if (v.retailer_product_id !== undefined && typeof v.retailer_product_id !== 'string')
        throw new Error('Invalid retailer product ID');
} return result as Listing[]; }
export type CatalogKind = 'products' | 'retailers' | 'listings';
export function catalogBody(kind: CatalogKind, form: FormData): Record<string, string> {
    const fields = kind === 'products' ? ['id', 'name', 'brand', 'model'] : kind === 'retailers' ? ['id', 'name'] : ['id', 'product_id', 'retailer_id', 'url', 'retailer_product_id'];
    const body: Record<string, string> = {};
    for (const field of fields) {
        body[field] = String(form.get(field) || '');
    }
    if (!body.id.trim())
        throw new Error('ID is required.');
    if (kind === 'listings' && (!body.product_id.trim() || !body.retailer_id.trim() || !body.url.trim()))
        throw new Error('Product, Retailer and URL are required.');
    return body;
}
export async function postCatalog(path: string, body?: Record<string, string>): Promise<void> {
    const response = await fetch(`/api${path}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(30000) });
    if (response.ok)
        return;
    const payload = await response.json().catch(() => null);
    const code = payload?.error?.code;
	if (code === 'tracking_disabled') throw new Error('Tracking is disabled for this Listing.');
    if (code === 'collection_busy')
        throw new Error('Collection is already running. Wait, then reload status.');
    if (code === 'collection_unavailable')
        throw new Error('No provider is configured for this Listing.');
    if (code === 'collection_failed')
        throw new Error('Collection failed. Previous observations are preserved.');
    if (response.status === 409)
        throw new Error('This ID or Listing source identity already exists. Use a different ID or check existing Listings.');
    if (response.status === 404)
        throw new Error('The requested record or linked Product/Retailer was not found.');
    if (response.status === 400)
        throw new Error('Invalid fields. Check IDs, linked records and the full http(s) URL.');
    throw new Error('Request failed. Check the backend/database and try again.');
}
