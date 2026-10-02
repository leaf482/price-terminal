import Link from 'next/link';
import { countLabels, type HomeOverview } from '../lib/home';
import { changePrice } from '../lib/price-changes';
import { formatPrice } from '../lib/prices';
export function HomeView({ data }: { data: HomeOverview }) {
    const context = (r: {product_id:string;product_name:string;listing_id:string}) => <><Link href={`/products/${encodeURIComponent(r.product_id)}`}>{r.product_name || r.product_id}</Link> · <Link href={`/listings/${encodeURIComponent(r.listing_id)}`}>Listing {r.listing_id}</Link></>;
    return <main><h1>Tracker overview</h1><p><Link href="/products">Tracked Products</Link> · <Link href="/catalog">Manage catalog</Link></p>
        <h2>Summary</h2>{data.counts === null ? <p role="alert">Summary unavailable.</p> : <div className="grid">{Object.entries(countLabels).map(([key,label]) => <article className="panel" key={key}><h3>{label}</h3><p>{data.counts![key as keyof typeof countLabels]}</p></article>)}</div>}
        <p>Counts cover the whole catalog. Collection errors count enabled Listings whose latest recorded attempt failed. Recent alerts count distinct alerts triggered in the last 7 days, including disabled alerts. Audit gaps may exist.</p>
        <section><h2><Link href="/price-changes">Recent Price Changes</Link></h2><p>Latest five observed-price comparisons; offer price otherwise sale, same Listing and currency.</p>
            {data.price_changes === null ? <p role="alert">Price changes unavailable.</p> : !data.price_changes.length ? <p>No price changes yet.</p> : data.price_changes.map(c => <article key={c.current.id}>{context(c)}<p>{c.retailer_name || c.retailer_id} · {changePrice(c.previous.minor_units,c.currency)} → {changePrice(c.current.minor_units,c.currency)} · {c.direction} · {c.percentage === null ? 'Percentage unavailable' : c.percentage+'%'}</p><p>Observed: {c.current.observed_at}</p></article>)}</section>
        <section><h2><Link href="/alerts">Recent Alert Triggers</Link></h2><p>Latest five stored alert events, independent of the seven-day count.</p>
            {data.alert_events === null ? <p role="alert">Alert events unavailable.</p> : !data.alert_events.length ? <p>No triggered alerts yet.</p> : data.alert_events.map(e => <article key={JSON.stringify([e.alert_id,e.observation_id])}>{context(e)}<p>{e.retailer_name || e.retailer_id} · {e.kind} · {formatPrice(e.minor_units,e.currency)}</p><p>Triggered: {e.triggered_at}</p></article>)}</section>
        <section><h2><Link href="/collection">Recent Collection Failures</Link></h2><p>Latest five unsuccessful attempts, including cancellations/unavailable outcomes and Listings since recovered or disabled. Attempt time is not observation freshness.</p>
            {data.collection_failures === null ? <p role="alert">Collection failures unavailable.</p> : !data.collection_failures.length ? <p>No recorded collection failures.</p> : data.collection_failures.map(f => <article key={f.id}>{context(f)}<p>{f.retailer_name || f.retailer_id} · {f.outcome} · {f.error_summary}</p><p>Attempt started: {f.started_at}</p></article>)}</section>
    </main>;
}
