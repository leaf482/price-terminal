import { parseProduct, type Product, type Prices } from './api.ts';

export type Summary = {
    product: Product; listing_count: number; has_current_price: boolean;
    best_price: Prices['best_price']; comparison_status: string;
    latest_observation_at: string | null; latest_attempt_at: string | null;
    freshness: Record<string, number>; collection: Record<string, number>;
    has_collection_error: boolean; recent_alert: boolean;
};
export type DashboardData = { products: Summary[]; truncated: boolean; recent_alert_since: string };
export type Filter = 'all' | 'priced' | 'missing' | 'error' | 'alert';

function object(x: unknown): Record<string, unknown> {
    if (!x || typeof x !== 'object' || Array.isArray(x)) throw new Error('Invalid dashboard response');
    return x as Record<string, unknown>;
}
function count(x: unknown) { if (typeof x !== 'number' || !Number.isSafeInteger(x) || x < 0) throw new Error('Invalid dashboard count/amount'); }
function timestamp(x: unknown) { if (typeof x !== 'string' || !Number.isFinite(Date.parse(x))) throw new Error('Invalid dashboard time'); }
export function parseDashboard(value: unknown): DashboardData {
    const data = object(value);
    if (!Array.isArray(data.products) || data.products.length > 20 || typeof data.truncated !== 'boolean') throw new Error('Invalid dashboard products');
    timestamp(data.recent_alert_since);
    for (const entry of data.products) {
        const row = object(entry); parseProduct(row.product); count(row.listing_count);
        for (const key of ['has_current_price', 'has_collection_error', 'recent_alert']) if (typeof row[key] !== 'boolean') throw new Error('Invalid dashboard status');
        if (typeof row.comparison_status !== 'string') throw new Error('Invalid comparison');
        for (const key of ['latest_observation_at', 'latest_attempt_at']) if (row[key] !== null) timestamp(row[key]);
        for (const key of ['freshness', 'collection']) Object.values(object(row[key])).forEach(count);
        if (row.best_price !== null) {
            const best = object(row.best_price); count(best.minor_units);
            if (!['USD', 'JPY'].includes(String(best.currency)) || !['offer_price', 'sale_price'].includes(String(best.basis)) || typeof best.listing_id !== 'string' || row.comparison_status !== 'comparable' || row.has_current_price !== true) throw new Error('Invalid comparable price');
        }
    }
    return data as DashboardData;
}

export function filterProducts(products: Summary[], query: string, filter: Filter): Summary[] {
    const search = query.trim().toLowerCase();
    return products.filter(row => {
        const p = row.product;
        if (![p.name, p.brand, p.model].some(value => value.toLowerCase().includes(search))) return false;
        switch (filter) {
            case 'priced': return row.has_current_price;
            case 'missing': return !row.has_current_price;
            case 'error': return row.has_collection_error;
            case 'alert': return row.recent_alert;
            default: return true;
        }
    });
}
