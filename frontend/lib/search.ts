import { api, parseProducts, type Product } from './api.ts';
import { parseRetailers, parseListings, type Retailer, type Listing } from './catalog.ts';
export type SearchResults = { products: Product[]; retailers: Retailer[]; listings: Listing[] };
export function parseSearch(value: unknown): SearchResults {
    if (!value || typeof value !== 'object') throw new Error('Invalid search response');
    const v = value as Record<string, unknown>;
    const result = { products: parseProducts(v.products), retailers: parseRetailers(v.retailers), listings: parseListings(v.listings) };
    if (Object.values(result).some(rows => rows.length > 20) || result.products.some(p => typeof p.archived !== 'boolean') || result.listings.some(l => typeof l.tracking_enabled !== 'boolean')) throw new Error('Invalid search bounds/state');
    return result;
}
export function searchCatalog(query: string, signal?: AbortSignal): Promise<SearchResults> {
    const text = query.trim();
    if (!text) return Promise.resolve({ products: [], retailers: [], listings: [] });
    return api(`/search?q=${encodeURIComponent(text)}`, parseSearch, signal);
}
