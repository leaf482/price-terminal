export type ChangePoint = { id: string; observed_at: string; minor_units: string };
export type PriceChange = { listing_id: string; product_id: string; product_name: string; retailer_id: string; retailer_name: string; previous: ChangePoint; current: ChangePoint; currency: 'USD' | 'JPY'; change_minor: string; percentage: string | null; direction: 'increased' | 'decreased' | 'unchanged' };
export type PriceChanges = { changes: PriceChange[]; truncated: boolean };
export function parsePriceChanges(x: unknown): PriceChanges {
    if (!x || typeof x !== 'object') throw new Error('Invalid price changes');
    const v = x as PriceChanges;
    if (!Array.isArray(v.changes) || v.changes.length > 100 || typeof v.truncated !== 'boolean') throw new Error('Invalid price changes');
    for (const c of v.changes) {
        if (!c || !['listing_id','product_id','product_name','retailer_id','retailer_name'].every(k => typeof c[k as keyof PriceChange] === 'string') || !['USD','JPY'].includes(c.currency) || !['increased','decreased','unchanged'].includes(c.direction) || typeof c.change_minor !== 'string' || !/^-?\d+$/.test(c.change_minor) || !(c.percentage === null || typeof c.percentage === 'string' && /^-?\d+\.\d{2}$/.test(c.percentage))) throw new Error('Invalid price change');
        for (const p of [c.previous,c.current]) if (!p || typeof p.id !== 'string' || typeof p.observed_at !== 'string' || !Number.isFinite(Date.parse(p.observed_at)) || typeof p.minor_units !== 'string' || !/^\d+$/.test(p.minor_units)) throw new Error('Invalid change point');
    }
    return v;
}
export function changePrice(value: string, currency: 'USD' | 'JPY'): string {
    const n = BigInt(value), magnitude = n < BigInt(0) ? -n : n, scale = currency === 'USD' ? BigInt(100) : BigInt(1);
    return `${currency} ${n < BigInt(0) ? '-' : ''}${magnitude / scale}${currency === 'USD' ? '.' + (magnitude % scale).toString().padStart(2,'0') : ''}`;
}
// Input is already newest-first at full backend timestamp precision. Stable
// percentage ties retain that order. Cross multiplication avoids float rounding
// and sorts exact ratios rather than their rounded display percentages.
export function changeRows(rows: PriceChange[], direction: string, query: string, sort: string): PriceChange[] {
    const q = query.trim().toLowerCase();
    const filtered = rows.filter(c => (direction === 'all' || c.direction === direction) && [c.product_id,c.product_name,c.retailer_id,c.retailer_name].some(v => v.toLowerCase().includes(q)));
    if (sort === 'newest') return filtered;
    return filtered.sort((a,b) => {
        const ap = BigInt(a.previous.minor_units), bp = BigInt(b.previous.minor_units);
        if (ap === BigInt(0)) return bp === BigInt(0) ? 0 : 1;
        if (bp === BigInt(0)) return -1;
        const x = (BigInt(a.current.minor_units) - ap) * bp, y = (BigInt(b.current.minor_units) - bp) * ap;
        const result = x < y ? -1 : x > y ? 1 : 0;
        return sort === 'decrease' ? result : -result;
    });
}
