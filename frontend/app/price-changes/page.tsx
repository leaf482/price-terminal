'use client';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import { api } from '../../lib/api';
import { parsePriceChanges, changeRows, changePrice, type PriceChanges } from '../../lib/price-changes';
export default function PriceChangesPage() {
    const [data,setData] = useState<PriceChanges | null>(null), [error,setError] = useState(false), [direction,setDirection] = useState('all'), [query,setQuery] = useState(''), [sort,setSort] = useState('newest'), [revision,setRevision] = useState(0);
    useEffect(() => { let active = true; const c = new AbortController(); const timer = setTimeout(() => c.abort(),10000);
        api('/price-changes',parsePriceChanges,c.signal).then(v => { if (active) { setData(v); setError(false); } }).catch(() => { if (active) setError(true); }).finally(() => clearTimeout(timer));
        return () => { active = false; c.abort(); clearTimeout(timer); };
    },[revision]);
    const rows = data ? changeRows(data.changes,direction,query,sort) : [];
    return <main><h1>Recent Price Changes</h1><p>Observed offer price, otherwise sale price. Each point compares with the previous valid price in the same Listing and currency, skipping missing prices and other currencies. Promotions are excluded.</p>
        <p>Percentage is unavailable when the previous price is zero, including zero to zero. Changed-at means observation time, not collection time.</p>
        <button onClick={() => setRevision(n => n + 1)}>Reload changes</button>
        <label>Direction <select value={direction} onChange={e => setDirection(e.target.value)}>{['all','decreased','increased','unchanged'].map(x => <option key={x}>{x}</option>)}</select></label>
        <label>Product / Retailer <input value={query} onChange={e => setQuery(e.target.value)}/></label>
        <label>Sort <select value={sort} onChange={e => setSort(e.target.value)}><option value="newest">Newest change</option><option value="decrease">Largest percentage decrease</option><option value="increase">Largest percentage increase</option></select></label>
        {error && <p role="alert">Could not load price changes. Retry.</p>}{!data && !error && <p role="status">Loading price changes…</p>}
        {data && <><p>Newest 100 comparisons at most. Filters apply to loaded results; unavailable percentages sort last.</p>{data.truncated && <p>Older comparisons are omitted.</p>}
            {!rows.length ? <p>No price changes match.</p> : <div className="table-scroll" role="region" aria-label="Price changes table" tabIndex={0}><table><thead><tr>{['Product','Retailer','Previous price / time','Current price','Absolute change','Percentage','Direction','Changed at','Listing'].map(x => <th scope="col" key={x}>{x}</th>)}</tr></thead><tbody>{rows.map(c => <tr key={c.current.id}>
                <td>{c.product_name || c.product_id}</td><td>{c.retailer_name || c.retailer_id}</td><td>{changePrice(c.previous.minor_units,c.currency)} · {c.previous.observed_at}</td><td>{changePrice(c.current.minor_units,c.currency)}</td><td>{changePrice(c.change_minor,c.currency)}</td><td>{c.percentage === null ? 'Unavailable (previous zero)' : `${c.percentage}%`}</td><td><span className="status-label" data-state={c.direction}>{c.direction}</span></td><td>{c.current.observed_at}</td><td><Link href={`/listings/${encodeURIComponent(c.listing_id)}`}>{c.listing_id}</Link></td>
            </tr>)}</tbody></table></div>}</>}
    </main>;
}
