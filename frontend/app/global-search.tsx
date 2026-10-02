"use client";
import Link from 'next/link';
import { useEffect, useRef, useState } from 'react';
import { searchCatalog, type SearchResults } from '../lib/search';
export default function GlobalSearch() {
    const [query, setQuery] = useState(''), [data, setData] = useState<SearchResults | null>(null), [busy, setBusy] = useState(false), [error, setError] = useState('');
    const request = useRef<AbortController | null>(null);
    useEffect(() => () => { request.current?.abort(); request.current = null; }, []);
    return <nav aria-label="Global catalog search" className="panel"><form onSubmit={async e => {
        e.preventDefault(); request.current?.abort(); const controller = new AbortController(); request.current = controller;
        setData(null); setError('');
        if (!query.trim()) { setBusy(false); setError('Enter a search query.'); return; }
        setBusy(true); const timer = setTimeout(() => controller.abort(), 10000);
        try { const result = await searchCatalog(query, controller.signal); if (request.current === controller) setData(result); }
        catch { if (request.current === controller) setError('Search failed or timed out. Try again.'); }
        finally { clearTimeout(timer); if (request.current === controller) setBusy(false); }
    }}><label>Search catalog <input value={query} maxLength={200} onChange={e => { request.current?.abort(); request.current = null; setBusy(false); setData(null); setError(''); setQuery(e.target.value); }} placeholder="Product, Retailer or Listing"/></label><button disabled={busy}>Search</button></form>
        <p>Up to 20 results per type, ordered by ID. Refine your query for other matches.</p>
        {busy && <p role="status">Searching…</p>}{error && <p role="alert">{error}</p>}
        {data && <>
            {!data.products.length && !data.retailers.length && !data.listings.length && <p role="status">No matching catalog records.</p>}
            <h2>Products</h2><ul>{data.products.map(p => <li key={p.id}><Link href={`/products/${encodeURIComponent(p.id)}`}>{p.name || p.id}</Link> · ID: {p.id} · {p.brand} {p.model}{p.archived && ' · Archived'}</li>)}</ul>
            <h2>Retailers</h2><ul>{data.retailers.map(r => <li key={r.id}><Link href={`/retailers/${encodeURIComponent(r.id)}`}>{r.name || r.id}</Link> · ID: {r.id}</li>)}</ul>
            <h2>Listings</h2><ul>{data.listings.map(l => <li key={l.id}><Link href={`/listings/${encodeURIComponent(l.id)}`}>{l.id}</Link> · Product: {l.product_id} · Retailer: {l.retailer_id} · {l.url} · SKU: {l.retailer_product_id || 'not supplied'}{!l.tracking_enabled && ' · Tracking disabled'}</li>)}</ul>
        </>}
    </nav>;
}
