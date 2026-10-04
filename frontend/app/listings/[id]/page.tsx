import Link from 'next/link';
import AttemptHistory from './attempt-history';
import { notFound } from 'next/navigation';
import { randomUUID } from 'node:crypto';
import type { Current, Product } from '../../../lib/api';
import type { Retailer } from '../../../lib/catalog';
import { loadListingDetail } from '../../../lib/listing-detail';
import { basis, displayPrice as formatPrice, safeSource } from '../../../lib/prices';
import CollectionControls from '../../products/[id]/collection-controls';
import TrackingControls from '../../products/[id]/tracking-controls';
import RecordPrice from '../../products/[id]/record-price';
import CSVImport from '../../products/[id]/csv-import';
import CSVExport from '../../products/[id]/csv-export';
import HistoryView from '../../products/[id]/history-view';
import PromotionView from '../../products/[id]/promotion-view';
import AlertView from '../../products/[id]/alert-view';
import QualityView from '../../products/[id]/quality-view';

export default async function ListingPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = await params;
    const data = await loadListingDetail(id);
    if (!data) notFound();
    const { current: c, product, retailer } = data, l = c.listing;
    const revision = randomUUID();
    return <main><nav><Link href={`/products/${encodeURIComponent(l.product_id)}`}>← Product detail</Link> · <Link href={`/retailers/${encodeURIComponent(l.retailer_id)}`}>Retailer detail</Link></nav>
        <ListingIdentity listing={l} product={product} retailer={retailer}/>
        <ObservedPrice current={c}/>
        <HistoryView key={`history-${revision}`} listings={[{ id: l.id, retailer: retailer?.name || l.retailer_id }]}/>
        <PromotionView key={`promotion-${revision}`} listingID={l.id}/>
        <AlertView key={`alerts-${revision}`} listingID={l.id}/>
        <section className="maintenance"><h2>Listing tools &amp; audit</h2>
        <CollectionControls id={l.id} status={c.collection} trackingEnabled={l.tracking_enabled !== false}/>
        <TrackingControls id={l.id} enabled={l.tracking_enabled !== false}/>
        <RecordPrice listingID={l.id}/><CSVImport listingID={l.id}/><CSVExport listingID={l.id}/>
        <AttemptHistory key={`attempts-${revision}`} listingID={l.id}/>
        <QualityView key={`quality-${revision}`} listingID={l.id} table/>
        </section>
    </main>;
}

// Presentation only: catalog fallbacks and source identity are separate from
// observed facts. Loading and all interactive tools remain owned by the page.
function ListingIdentity({ listing: l, product, retailer }: {
    listing: Current['listing'];
    product: Product | null;
    retailer: Retailer | null;
}) {
    const url = safeSource(l.url);
    return <header className="listing-identity">
        <h1>Listing {l.id}</h1>
        <p>Product: {product?.name || l.product_id} · {product?.brand} · {product?.model}</p>
        {!product && <p role="alert">Product metadata is unavailable. Stored Product ID: {l.product_id}</p>}
        <p>Retailer: {retailer?.name || l.retailer_id} · ID: {l.retailer_id}</p>
        {!retailer && <p role="alert">Retailer metadata is unavailable. Stored Retailer ID: {l.retailer_id}</p>}
        <p>Source URL: {url ? <a href={url} target="_blank" rel="noreferrer">{l.url}</a> : l.url}</p>
        <p>Retailer product ID: {l.retailer_product_id || 'Not supplied'}</p>
        <p>Source identity is read-only. Create a new Listing for source changes.</p>
        <p className="status-label">Tracking: {l.tracking_enabled === false ? 'disabled' : 'enabled'}</p>
    </header>;
}

function ObservedPrice({ current: c }: { current: Current }) {
    const o = c.observation, price = basis(o);
    return <section className="price-hero">
        <h2>Observed current price</h2><p className="price">{price ? formatPrice(price.amount, price.currency) : 'Price unavailable'}</p>
        <p>{price?.label || 'No offer or sale price'} · Stock: {o?.stock.replaceAll('_', ' ') || 'unknown'}</p>
        <p>Latest valid observation: {o?.observed_at || 'None'} · Freshness: {c.freshness}</p><p>Observation source: {o?.source || 'Unavailable'}</p>
        {o?.msrp !== undefined && <p>MSRP: {formatPrice(o.msrp, o.currency!)} · {o.msrp_source}</p>}
        {o?.retailer_list_price !== undefined && <p>Retailer list: {formatPrice(o.retailer_list_price, o.currency!)}</p>}
        {o?.sale_price !== undefined && o.offer_price !== undefined && <p>Sale price: {formatPrice(o.sale_price, o.currency!)}</p>}
    </section>;
}
