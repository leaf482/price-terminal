import { api, APIError, parsePrices, parseProduct } from './api.ts';
import { parseRetailers } from './catalog.ts';

async function optionalRecord<T>(read: Promise<T>): Promise<T | null> {
    try { return await read; } catch (error) { if (error instanceof APIError && error.status === 404) return null; throw error; }
}
export async function loadListingDetail(id: string) {
    const current = await optionalRecord(api(`/listings/${encodeURIComponent(id)}/price`, value => parsePrices({ product_id: '', listings: [value], best_price: null, comparison_status: 'not_compared' }).listings[0]));
    if (!current) return null;
    const [product, retailer] = await Promise.all([
        optionalRecord(api(`/products/${encodeURIComponent(current.listing.product_id)}`, parseProduct)),
        optionalRecord(api(`/retailers/${encodeURIComponent(current.listing.retailer_id)}`, value => parseRetailers([value])[0])),
    ]);
    return { current, product, retailer };
}
