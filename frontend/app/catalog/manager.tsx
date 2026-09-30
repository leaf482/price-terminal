"use client";
import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { api, parseProducts, type Product } from '../../lib/api';
import { catalogBody, postCatalog, parseRetailers, parseListings, type CatalogKind, type Retailer, type Listing } from '../../lib/catalog';
export function CatalogForm({ kind, products, retailers, onCreated }: {
    kind: CatalogKind;
    products: Product[];
    retailers: Retailer[];
    onCreated: (body: Record<string, string>) => void;
}) {
    const [busy, setBusy] = useState(false), [message, setMessage] = useState(''), [error, setError] = useState('');
    const pending = useRef(false);
    const fields = kind === 'products' ? ['id', 'name', 'brand', 'model'] : kind === 'retailers' ? ['id', 'name'] : ['id', 'product_id', 'retailer_id', 'url', 'retailer_product_id'];
    return <form className="panel" onSubmit={async (e) => { e.preventDefault(); if (pending.current)
        return; const form = e.currentTarget; pending.current = true; setBusy(true); setError(''); setMessage(''); try {
        const body = catalogBody(kind, new FormData(form));
        await postCatalog(`/${kind}`, body);
        form.reset();
        setMessage(`Created ${body.id}.`);
        onCreated(body);
    }
    catch (err) {
        setError(err instanceof Error ? err.message : 'Could not create record.');
    }
    finally {
        pending.current = false;
        setBusy(false);
    } }}>
 <h2>Create {kind === 'products' ? 'Product' : kind === 'retailers' ? 'Retailer' : 'Listing'}</h2>
 {fields.map(field => <p key={field}><label>{field.replaceAll('_', ' ')} {field === 'retailer_product_id' && '(optional)'}<br /><input name={field} type={field === 'url' ? 'url' : 'text'} required={['id', 'product_id', 'retailer_id', 'url'].includes(field)} list={field === 'product_id' ? 'catalog-products' : field === 'retailer_id' ? 'catalog-retailers' : undefined}/></label></p>)}
 {kind === 'listings' && <><datalist id="catalog-products">{products.map(p => <option key={p.id} value={p.id}>{p.name || p.id}</option>)}</datalist><datalist id="catalog-retailers">{retailers.map(r => <option key={r.id} value={r.id}>{r.name || r.id}</option>)}</datalist><p>Choose an existing ID or enter one exactly. The backend verifies both relationships.</p></>}
 <button disabled={busy}>{busy ? 'Saving…' : 'Create'}</button>{message && <p role="status">{message}</p>}{error && <p role="alert">{error}</p>}
 </form>;
}
export default function CatalogManager() {
    const router = useRouter();
    const [revision, setRevision] = useState(0), [data, setData] = useState<{
        products: Product[];
        retailers: Retailer[];
    } | null>(null), [error, setError] = useState('');
    const [selected, setSelected] = useState(''), [input, setInput] = useState('');
    useEffect(() => { let active = true; const c = new AbortController(); const timer = setTimeout(() => c.abort(), 10000); Promise.all([api('/products?limit=100', parseProducts, c.signal), api('/retailers?limit=100', parseRetailers, c.signal)]).then(([products, retailers]) => { if (active) {
        setData({ products, retailers });
        setError('');
    } }).catch(() => { if (active)
        setError('Could not load catalog. Reload to retry.'); }).finally(() => { clearTimeout(timer); c.abort(); }); return () => { active = false; c.abort(); clearTimeout(timer); }; }, [revision]);
    const reload = () => { setRevision(n => n + 1); router.refresh(); };
    return <><button onClick={reload}>Reload catalog</button>{error && <p role="alert">{error}</p>}{!data && !error && <p role="status">Loading catalog…</p>}
 <div className="grid">{(['products', 'retailers', 'listings'] as CatalogKind[]).map(kind => <CatalogForm key={kind} kind={kind} products={data?.products || []} retailers={data?.retailers || []} onCreated={body => { if (kind === 'listings') {
        setSelected(body.product_id);
        setInput(body.product_id);
    } reload(); }}/>)}</div>
 {data && <><h2>Products</h2>{!data.products.length && <p>No Products yet.</p>}<ul>{data.products.map(p => <li key={p.id}><Link href={`/products/${encodeURIComponent(p.id)}`}>{p.name || p.id}</Link> · ID: {p.id} · {p.brand} {p.model} <button onClick={() => { setSelected(p.id); setInput(p.id); }}>View Listings</button></li>)}</ul>
 <h2>Retailers</h2>{!data.retailers.length && <p>No Retailers yet.</p>}<ul>{data.retailers.map(r => <li key={r.id}>{r.name || r.id} · ID: {r.id}</li>)}</ul><p>Up to 100 Products and Retailers shown. For other records, enter their exact IDs.</p></>}
 <h2>Listings by Product</h2><form onSubmit={e => { e.preventDefault(); setSelected(input); }}><label>Existing Product ID <input required value={input} onChange={e => setInput(e.target.value)}/></label> <button>View Listings</button></form>
 {selected && <ListingBrowser key={`${selected}:${revision}`} id={selected} retailers={data?.retailers || []}/>}
 </>;
}
function ListingBrowser({ id, retailers }: {
    id: string;
    retailers: Retailer[];
}) {
    const [listings, setListings] = useState<Listing[] | null>(null), [error, setError] = useState('');
    useEffect(() => { let active = true; const c = new AbortController(); const timer = setTimeout(() => c.abort(), 10000); api(`/products/${encodeURIComponent(id)}/listings?limit=100`, parseListings, c.signal).then(v => { if (active)
        setListings(v); }).catch(() => { if (active)
        setError('Could not read Listings. Verify the Product ID and reload.'); }).finally(() => clearTimeout(timer)); return () => { active = false; c.abort(); clearTimeout(timer); }; }, [id]);
    return <section><p>Product: <Link href={`/products/${encodeURIComponent(id)}`}>{id}</Link></p>{error && <p role="alert">{error}</p>}{!listings && !error && <p>Loading Listings…</p>}{listings?.length === 0 && <p>No Listings for this Product.</p>}<ul>{listings?.map(l => <li key={l.id}><Link href={`/products/${encodeURIComponent(l.product_id)}#${encodeURIComponent(l.id)}`}>Listing {l.id}</Link> · Product: {l.product_id} · Retailer: {retailers.find(r => r.id === l.retailer_id)?.name || l.retailer_id} ({l.retailer_id})<p>{l.url} · Retailer product ID: {l.retailer_product_id || 'not supplied'}</p></li>)}</ul>{listings?.length === 100 && <p>First 100 Listings shown.</p>}</section>;
}
