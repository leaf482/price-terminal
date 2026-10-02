export const OBSERVATION_RECENT_MS = 24 * 60 * 60 * 1000;
export type HealthListing = {
    listing_id: string; product_id: string; product_name: string; retailer_id: string; retailer_name: string;
    tracking_enabled: boolean; observed_at: string | null; attempted_at: string | null; successful_at: string | null;
    outcome: string; error_summary: string;
};
export type CollectionHealth = { listings: HealthListing[]; truncated: boolean; counts: { total: number; healthy: number; error: number; never: number; disabled: number } };
export function parseCollectionHealth(x: unknown): CollectionHealth {
    if (!x || typeof x !== 'object') throw new Error('Invalid collection health');
    const v = x as CollectionHealth;
    if (!Array.isArray(v.listings) || v.listings.length > 100 || typeof v.truncated !== 'boolean' || !v.counts || !['total', 'healthy', 'error', 'never', 'disabled'].every(k => Number.isInteger(v.counts[k as keyof typeof v.counts]) && v.counts[k as keyof typeof v.counts] >= 0)) throw new Error('Invalid collection health');
    for (const r of v.listings) {
        if (!r || !['listing_id', 'product_id', 'product_name', 'retailer_id', 'retailer_name', 'outcome', 'error_summary'].every(k => typeof r[k as keyof HealthListing] === 'string') || typeof r.tracking_enabled !== 'boolean' || ![r.observed_at, r.attempted_at, r.successful_at].every(t => t === null || (typeof t === 'string' && Number.isFinite(Date.parse(t))))) throw new Error('Invalid collection Listing');
    }
    return v;
}
// Display only: future timestamps are not recent. Nothing is persisted/evaluated.
export function observationFreshness(at: string | null, now: number): string {
    if (at === null) return 'no observation';
    const age = now - Date.parse(at);
    return age >= 0 && age <= OBSERVATION_RECENT_MS ? 'recent' : 'stale';
}
export function collectionRows(rows: HealthListing[], filter: string, sort: string): HealthListing[] {
    const compare = (a: string, b: string) => a < b ? -1 : a > b ? 1 : 0;
    const time = (s: string | null) => s === null ? null : Date.parse(s);
    return rows.filter(r => filter === 'error' ? r.attempted_at !== null && r.outcome !== 'success' : filter === 'never' ? r.attempted_at === null : filter === 'disabled' ? !r.tracking_enabled : filter === 'enabled' ? r.tracking_enabled : true).sort((a, b) => {
        let diff = 0;
        if (sort === 'product') diff = compare((a.product_name || a.product_id).toLowerCase(), (b.product_name || b.product_id).toLowerCase());
        else {
            const x = time(sort === 'success' ? a.successful_at : a.attempted_at), y = time(sort === 'success' ? b.successful_at : b.attempted_at);
            // No success comes first for oldest-success triage; no attempt last for newest.
            diff = x === null ? (y === null ? 0 : sort === 'success' ? -1 : 1) : y === null ? (sort === 'success' ? 1 : -1) : sort === 'success' ? x - y : y - x;
        }
        return diff || compare(a.listing_id, b.listing_id);
    });
}
