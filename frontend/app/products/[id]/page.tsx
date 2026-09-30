import Link from "next/link";
import { api, parseProduct, parsePrices } from "../../../lib/api";
import { basis, formatPrice, safeSource } from "../../../lib/prices";
import HistoryView from "./history-view";
import PromotionView from "./promotion-view";
import AlertView from "./alert-view";
export default async function ProductPage({ params }: {
    params: Promise<{
        id: string;
    }>;
}) {
    const { id } = await params, path = encodeURIComponent(id);
    const [product, prices] = await Promise.all([api(`/products/${path}`, parseProduct), api(`/products/${path}/prices`, parsePrices)]);
    return <main><Link href="/">← Products</Link><header><p className="eyebrow">{product.brand || "PRODUCT"}</p><h1>{product.name || product.id}</h1><p>{product.model} · ID: {product.id}</p></header>
 <h2>Current observations</h2>{!prices.listings.length && <p>No Listings for this product yet.</p>}
 <div className="grid">{prices.listings.map(row => { const o = row.observation, p = basis(o), url = safeSource(row.listing.url); return <article className="panel" key={row.listing.id}><h3>{row.listing.retailer_id}</h3><p>Listing: {row.listing.id}{row.listing.retailer_product_id && ` · SKU: ${row.listing.retailer_product_id}`}</p><p className="price">{p ? formatPrice(p.amount, p.currency) : "Price unavailable"}</p><p>{p?.label || "No offer or sale price"} · {o?.stock.replaceAll("_", " ") || "Stock unknown"}</p><p className="muted">{row.freshness} · {o?.observed_at || "No observation"}</p><p className="muted">Collection: {row.collection.state} · Source: {o?.source || "unavailable"}</p>{o?.msrp !== undefined && <p>MSRP: {formatPrice(o.msrp, o.currency!)} · {o.msrp_source}</p>}{o?.retailer_list_price !== undefined && <p>Retailer list: {formatPrice(o.retailer_list_price, o.currency!)}</p>}{o?.sale_price !== undefined && o.offer_price !== undefined && <p>Sale: {formatPrice(o.sale_price, o.currency!)}</p>}{url && <a href={url} target="_blank" rel="noreferrer">Visit listing ↗</a>}<PromotionView key={row.listing.id} listingID={row.listing.id}/><AlertView key={`alerts-${row.listing.id}`} listingID={row.listing.id}/></article>; })}</div>
 <HistoryView key={product.id} listings={prices.listings.map(x => ({ id: x.listing.id, retailer: x.listing.retailer_id }))}/></main>;
}
