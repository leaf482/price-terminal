import type { Current } from './api.ts';
import { basis } from './prices.ts';
export type ListingSort = 'price_asc' | 'price_desc' | 'newest' | 'retailer';
const compare = (a: string | number | bigint, b: typeof a) => a < b ? -1 : a > b ? 1 : 0;
// Sort timestamps by instant with all nine fractional digits, not Date's
// millisecond precision. This does not change submitted or stored facts.
function instant(text: string): bigint {
    const fraction = text.match(/\.(\d+)/)?.[1] ?? '';
    return BigInt(Date.parse(text.replace(/\.\d+/, ''))) * BigInt(1000000) + BigInt(fraction.padEnd(9, '0').slice(0, 9));
}
export function sortListings(rows: Current[], mode: ListingSort, names: Record<string, string>): Current[] {
    return [...rows].sort((a, b) => {
        let order = 0;
        if (mode === 'price_asc' || mode === 'price_desc') {
            const x = basis(a.observation), y = basis(b.observation);
            if (!x || !y) order = x ? -1 : y ? 1 : 0;
            else order = compare(x.currency, y.currency) || compare(x.amount, y.amount) * (mode === 'price_asc' ? 1 : -1);
        } else if (mode === 'newest') {
            const x = a.observation, y = b.observation;
            order = x && y ? compare(instant(y.observed_at), instant(x.observed_at)) : x ? -1 : y ? 1 : 0;
        } else {
            order = compare((names[a.listing.retailer_id] || a.listing.retailer_id).toLowerCase(), (names[b.listing.retailer_id] || b.listing.retailer_id).toLowerCase());
        }
        return order || compare(a.listing.id, b.listing.id);
    });
}
