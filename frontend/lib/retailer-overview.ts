import { parseProduct, parsePrices, type Product, type Current } from './api.ts';
import { parseRetailers, type Retailer } from './catalog.ts';
import { basis } from './prices.ts';
export type RetailerRow = { product: Product; current: Current };
export type RetailerOverview = { retailer: Retailer; listings: RetailerRow[]; truncated: boolean };
export type RetailerFilter = 'all' | 'enabled' | 'disabled' | 'priced' | 'error';
export function parseRetailerOverview(value: unknown): RetailerOverview {
    if (!value || typeof value !== 'object') throw new Error('Invalid Retailer overview');
    const v = value as Record<string, unknown>;
    parseRetailers([v.retailer]);
    if (!Array.isArray(v.listings) || v.listings.length > 100 || typeof v.truncated !== 'boolean') throw new Error('Invalid Retailer Listings');
    for (const item of v.listings) {
        const p = parseProduct(item.product);
        parsePrices({ product_id: p.id, listings: [item.current], best_price: null, comparison_status: 'not_compared' });
    }
    return v as RetailerOverview;
}
export function filterRetailerListings(rows: RetailerRow[], query: string, filter: RetailerFilter): RetailerRow[] {
    const search = query.trim().toLowerCase();
    return rows.filter(({ product: p, current: c }) => {
        if (![p.name, p.brand, p.model].some(text => text.toLowerCase().includes(search))) return false;
        switch (filter) {
            case 'enabled': return c.listing.tracking_enabled !== false;
            case 'disabled': return c.listing.tracking_enabled === false;
            case 'priced': return basis(c.observation) !== null;
            case 'error': return c.collection.state === 'failed' || !!c.collection.error;
            default: return true;
        }
    });
}
