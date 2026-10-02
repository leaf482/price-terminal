'use client';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import { api } from '../../lib/api';
import { parseCollectionHealth, collectionRows, observationFreshness, type CollectionHealth } from '../../lib/collection-health';

export default function CollectionPage() {
    const [data, setData] = useState<CollectionHealth | null>(null), [error, setError] = useState(false);
    const [filter, setFilter] = useState('all'), [sort, setSort] = useState('attempt'), [revision, setRevision] = useState(0), [now, setNow] = useState(0);
    useEffect(() => {
        let active = true; const c = new AbortController(); const timer = setTimeout(() => c.abort(), 10000);
        api('/collection/overview', parseCollectionHealth, c.signal).then(v => { if (active) { setData(v); setNow(Date.now()); setError(false); } }).catch(() => { if (active) setError(true); }).finally(() => clearTimeout(timer));
        return () => { active = false; c.abort(); clearTimeout(timer); };
    }, [revision]);
    return <main><h1>Collection</h1><button onClick={() => setRevision(n => n + 1)}>Reload collection</button>
        <p>Observation freshness is display-only: recent means within the last 24 hours. Collection attempts do not refresh observations. Future timestamps are shown as stale.</p>
        {error && <p role="alert">Could not load collection health. Retry.</p>}{!data && !error && <p role="status">Loading collection health…</p>}
        <label>Filter <select value={filter} onChange={e => setFilter(e.target.value)}>{[['all','All'],['error','Collection error'],['never','Never collected'],['disabled','Tracking disabled'],['enabled','Tracking enabled']].map(([value,label]) => <option key={value} value={value}>{label}</option>)}</select></label>
        <label>Sort <select value={sort} onChange={e => setSort(e.target.value)}><option value="attempt">Most recent attempt</option><option value="success">Oldest successful collection (none first)</option><option value="product">Product name</option></select></label>
        {data && <><p>Total shown: {data.counts.total} · Healthy/latest success: {data.counts.healthy} · Collection error: {data.counts.error} · Never collected: {data.counts.never} · Tracking disabled: {data.counts.disabled}</p>
            <p>Counts cover loaded Listings before filters. Disabled takes priority; healthy means latest attempt succeeded, regardless of observation age. Never collected means no recorded attempt; audit gaps are possible.</p>
            {data.truncated && <p>First 100 Listings by ID shown. Filters and sorting apply only to this bounded set.</p>}
            {!collectionRows(data.listings,filter,sort).length ? <p>No Listings match these filters.</p> : <table><thead><tr>{['Product','Retailer','Tracking','Latest observation / freshness','Latest attempt','Latest success','Outcome / error','Listing'].map(x => <th key={x}>{x}</th>)}</tr></thead>
                <tbody>{collectionRows(data.listings,filter,sort).map(r => <tr key={r.listing_id}>
                    <td>{r.product_name || r.product_id}<br/>{r.product_id}</td><td>{r.retailer_name || r.retailer_id}<br/>{r.retailer_id}</td><td>{r.tracking_enabled ? 'enabled' : 'disabled'}</td>
                    <td>{r.observed_at || 'None'} · {observationFreshness(r.observed_at,now)}</td><td>{r.attempted_at || 'Never'}</td><td>{r.successful_at || 'Never'}</td><td>{r.outcome || 'No recorded attempt'}{r.error_summary && ` · ${r.error_summary}`}</td><td><Link href={`/listings/${encodeURIComponent(r.listing_id)}`}>{r.listing_id}</Link></td>
                </tr>)}</tbody></table>}</>}
    </main>;
}
