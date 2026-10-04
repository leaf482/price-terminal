import Link from "next/link";
import { displayPrice } from "../../../lib/prices";
import ListingComparison from "./listing-comparison";
import { parseRetailers } from "../../../lib/catalog";
import ArchiveControls from "./archive-controls";
import MetadataEditor from "../../catalog/metadata-editor";
import { api, parseProduct, parsePrices } from "../../../lib/api";

import HistoryView from "./history-view";

import { randomUUID } from "node:crypto";
export default async function ProductPage({ params }: {
    params: Promise<{
        id: string;
    }>;
}) {
    const { id } = await params, path = encodeURIComponent(id);
	// A server refresh after import must also reload client history/audit reads,
	// including imports older than the current observation.
	const revision = randomUUID();
    const [product, prices] = await Promise.all([api(`/products/${path}`, parseProduct), api(`/products/${path}/prices`, parsePrices)]);
    const retailers = await api('/retailers?limit=100', parseRetailers);
    const retailerNames: Record<string, string> = Object.fromEntries(retailers.map(r => [r.id, r.name]));
    const missing = [...new Set(prices.listings.map(r => r.listing.retailer_id))].filter(id => !Object.hasOwn(retailerNames, id));
    await Promise.all(missing.map(async id => { const r = await api(`/retailers/${encodeURIComponent(id)}`, value => parseRetailers([value])[0]); Object.defineProperty(retailerNames, id, { value: r.name, enumerable: true, configurable: true, writable: true }); }));
    return <main className="product-detail"><nav><Link href="/products">← Products</Link> <Link href="/catalog">Manage catalog</Link></nav><header><p className="eyebrow">{product.brand || "PRODUCT"}</p><h1>{product.name || product.id}</h1><p className="muted">{product.model} · ID: {product.id}</p></header>
 <section className="price-hero" aria-label="Best observed price"><p className="eyebrow">Best comparable observed price</p><p className="price">{prices.best_price ? displayPrice(prices.best_price.minor_units, prices.best_price.currency) : "No comparable price"}</p><p className="muted">{prices.best_price ? `Observed ${prices.best_price.basis.replaceAll('_', ' ')} · Fresh, in-stock observations in one currency` : prices.comparison_status.replaceAll('_', ' ')}. Promotions and EffectivePrice are excluded.</p></section>
 <HistoryView key={`${product.id}:${revision}`} listings={prices.listings.map(x => ({ id: x.listing.id, retailer: retailerNames[x.listing.retailer_id] || x.listing.retailer_id }))}/>
 {!prices.listings.length && <p>No Listings for this product yet.</p>}
 <ListingComparison prices={prices} retailers={retailerNames} revision={revision}/>
 <section className="maintenance"><h2>Product management</h2><ArchiveControls id={product.id} archived={product.archived === true}/><MetadataEditor kind="products" record={product}/><p className="muted">Listing source fields are read-only. To change Product, Retailer, URL or retailer product ID, disable the old Listing if needed and create a new Listing.</p></section></main>;
}
