"use client";
import Link from 'next/link';
import { useState } from 'react';
import { filterRetailerListings, type RetailerOverview, type RetailerFilter } from '../../../lib/retailer-overview';
import { basis, formatPrice, safeSource } from '../../../lib/prices';
export default function Overview({ data }: { data: RetailerOverview }) {
    const [query, setQuery] = useState(''), [filter, setFilter] = useState<RetailerFilter>('all');
    const rows = filterRetailerListings(data.listings, query, filter);
    return <section><label>Search Product name, brand or model <input value={query} onChange={e => setQuery(e.target.value)}/></label>
        <label>Filter <select value={filter} onChange={e => setFilter(e.target.value as RetailerFilter)}><option value="all">All Listings</option><option value="enabled">Tracking enabled</option><option value="disabled">Tracking disabled</option><option value="priced">Has current price</option><option value="error">Collection error</option></select></label>
        <p>{rows.length} of {data.listings.length} loaded Listings. Prices are observed offer prices, otherwise sale prices; Products are not compared against each other.</p>
        {data.truncated && <p>Showing the first 100 Listings by ID. Search and filters apply to these loaded Listings only.</p>}
        {!data.listings.length ? <p>No Listings for this Retailer.</p> : !rows.length ? <p>No Listings match this search and filter.</p> : <div className="grid">{rows.map(({ product: p, current: c }) => {
            const price = basis(c.observation), url = safeSource(c.listing.url);
            return <article className="panel" key={c.listing.id}><h2><Link href={`/products/${encodeURIComponent(p.id)}#${encodeURIComponent(c.listing.id)}`}>{p.name || p.id}</Link></h2>
                <p>{p.brand} · {p.model} · Product ID: {p.id}{p.archived && ' · Archived Product'}</p><p><Link href={`/listings/${encodeURIComponent(c.listing.id)}`}>Listing: {c.listing.id}</Link></p>
                {url && <a href={url} target="_blank" rel="noreferrer">{c.listing.url}</a>}
                <p>Tracking: {c.listing.tracking_enabled === false ? 'disabled' : 'enabled'}</p>
                <p>Observed price: {price ? formatPrice(price.amount, price.currency) : 'Price unavailable'} · {price?.label || 'No offer or sale price'}</p>
                <p>Stock: {c.observation?.stock.replaceAll('_', ' ') || 'unknown'}</p>
                <p>Latest valid observation: {c.observation?.observed_at || 'None'} · {c.freshness}</p>
                <p>Collection: {c.collection.state}{c.collection.error && ` · ${c.collection.error}`}</p>
                <p>Last attempt: {c.collection.last_attempted_at || 'Never'} · Last success: {c.collection.last_successful_at || 'Never'}</p>
            </article>;
        })}</div>}
    </section>;
}
