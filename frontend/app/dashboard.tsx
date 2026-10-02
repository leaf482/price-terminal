"use client";
import Link from 'next/link';
import { useState } from 'react';
import { filterProducts, type DashboardData, type Filter, type Summary } from '../lib/dashboard';
import { formatPrice } from '../lib/prices';

function counts(values: Record<string, number>) {
    return Object.entries(values).sort(([a], [b]) => a.localeCompare(b)).map(([label, n]) => `${n} ${label.replaceAll('_', ' ')}`).join(' · ') || 'No Listings';
}
export function ProductCard({ row }: { row: Summary }) {
    const p = row.product;
    return <article className="panel">
        <p className="muted">{p.brand || 'Brand not specified'} · {p.model || 'Model not specified'}</p>
        <h2><Link href={`/products/${encodeURIComponent(p.id)}`}>{p.name || p.id}</Link></h2>
        {p.archived && <p>Archived</p>}
        <p>{row.listing_count} Listings</p>
        <p className="price">{row.best_price ? formatPrice(row.best_price.minor_units, row.best_price.currency) : 'No comparable price'}</p>
        <p>{row.best_price ? `Best observed ${row.best_price.basis.replaceAll('_', ' ')}` : row.comparison_status.replaceAll('_', ' ')}</p>
        {!row.has_current_price && <p>Missing current offer / sale price</p>}
        {row.has_current_price && !row.best_price && <p>Known prices exist; no product-wide comparable best price.</p>}
        <p>Latest valid observation: {row.latest_observation_at || 'None'}</p>
        <p>Observation freshness: {counts(row.freshness)}</p>
        <p>Collection status: {counts(row.collection)}</p>
        <p>Latest collection attempt: {row.latest_attempt_at || 'Never'}</p>
        {row.has_collection_error && <p>Collection error — retained observations keep their original times.</p>}
        <p>{row.recent_alert ? 'Triggered alert in the last 7 days' : 'No triggered alerts in the last 7 days'}</p>
    </article>;
}

export default function Dashboard({ data, includeArchived = false }: { data: DashboardData; includeArchived?: boolean }) {
    const [query, setQuery] = useState(''), [filter, setFilter] = useState<Filter>('all');
    const visible = filterProducts(data.products, query, filter);
    return <>
        <div className="controls">
            <Link href={includeArchived ? '/products' : '/products?include_archived=true'}>{includeArchived ? 'Hide archived' : 'Show archived'}</Link>
            <label>Search name, brand or model <input type="search" value={query} onChange={e => setQuery(e.target.value)} /></label>
            <label>Filter <select value={filter} onChange={e => setFilter(e.target.value as Filter)}>
                <option value="all">All products</option><option value="priced">Has current price</option><option value="missing">Missing current price</option><option value="error">Collection error</option><option value="alert">Triggered alert present</option>
            </select></label>
        </div>
        <p>Price presence means a latest valid Listing observation has an offer or sale price, even if stale. Best price requires all Listings to be fresh, in stock and comparable in one currency. Promotions are excluded.</p>
        <p className="muted">Collection status is process-local and resets on restart. Alert window starts {data.recent_alert_since}; retained events may refer to subsequently invalidated observations.</p>
        {!data.products.length ? <p className="panel">{includeArchived ? 'No products yet.' : 'No active products. Use Show archived to include archived Products.'} Open Manage catalog to create a Product, Retailer and Listing.</p> : !visible.length ? <p role="status">No products match this search and filter.</p> : <div className="grid">{visible.map(row => <ProductCard key={row.product.id} row={row} />)}</div>}
        {data.truncated && <p>Showing the first 20 products by ID. Search and filters apply only to these loaded products.</p>}
    </>;
}
