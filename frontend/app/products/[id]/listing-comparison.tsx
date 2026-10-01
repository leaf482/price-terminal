"use client";
import { useState } from "react";
import type { Prices } from "../../../lib/api";
import { sortListings, type ListingSort } from "../../../lib/comparison";
import { basis, formatPrice, safeSource } from "../../../lib/prices";
import PromotionView from "./promotion-view";
import AlertView from "./alert-view";
import CollectionControls from "./collection-controls";
import QualityView from "./quality-view";
import RecordPrice from "./record-price";
import TrackingControls from "./tracking-controls";
import CSVImport from "./csv-import";
import CSVExport from "./csv-export";
export default function ListingComparison({ prices, retailers, revision }: { prices: Prices; retailers: Record<string, string>; revision: string }) {
 const [sort, setSort] = useState<ListingSort>('price_asc');
 const best = prices.best_price;
 return <section aria-label="Listing comparison">
 <h2>Compare Listings</h2>
 <p><strong>Best comparable observed price: </strong>{best ? `${formatPrice(best.minor_units, best.currency)} · ${best.basis.replaceAll('_', ' ')} · Listing ${best.listing_id}` : `Unavailable — ${prices.comparison_status.replaceAll('_', ' ')}`}</p>
 <p>Best price uses the backend observed offer price, otherwise sale price. All Listings must have fresh, in-stock, comparable observations in one currency. Promotions and EffectivePrice are excluded.</p>
 <label>Sort Listings <select value={sort} onChange={e => setSort(e.target.value as ListingSort)}><option value="price_asc">Observed price: low to high</option><option value="price_desc">Observed price: high to low</option><option value="newest">Newest observation</option><option value="retailer">Retailer name</option></select></label>
 <p>Price sorts group currencies alphabetically, with missing prices last. Sorting includes stale and out-of-stock observations; it does not imply eligibility for best price. Conditional savings are not guaranteed.</p>
 <div className="grid">{sortListings(prices.listings, sort, retailers).map(row => { const o = row.observation, p = basis(o), url = safeSource(row.listing.url); return <article className="panel" id={row.listing.id} key={row.listing.id}><h3>{retailers[row.listing.retailer_id] || row.listing.retailer_id}</h3><p>Retailer ID: {row.listing.retailer_id}</p><p>Listing: {row.listing.id}{row.listing.retailer_product_id && ` · SKU: ${row.listing.retailer_product_id}`}</p><p><strong>Observed current price</strong></p><p className="price">{p ? formatPrice(p.amount, p.currency) : "Price unavailable"}</p><p>{p?.label || "No offer or sale price"} · {o?.stock.replaceAll("_", " ") || "Stock unknown"}</p><p>Tracking: {row.listing.tracking_enabled === false ? "disabled" : "enabled"}</p><p className="muted">Observation freshness: {row.freshness} · Latest valid observation: {o?.observed_at || "No observation"}</p><p className="muted">Collection: {row.collection.state} · Source: {o?.source || "unavailable"}</p>{o?.msrp !== undefined && <p>MSRP: {formatPrice(o.msrp, o.currency!)} · {o.msrp_source}</p>}{o?.retailer_list_price !== undefined && <p>Retailer list: {formatPrice(o.retailer_list_price, o.currency!)}</p>}{o?.sale_price !== undefined && o.offer_price !== undefined && <p>Sale: {formatPrice(o.sale_price, o.currency!)}</p>}{url && <a href={url} target="_blank" rel="noreferrer">Visit listing ↗</a>}<RecordPrice listingID={row.listing.id}/><CSVImport listingID={row.listing.id}/><CSVExport listingID={row.listing.id}/><QualityView key={revision} listingID={row.listing.id}/><TrackingControls id={row.listing.id} enabled={row.listing.tracking_enabled !== false}/><CollectionControls id={row.listing.id} status={row.collection} trackingEnabled={row.listing.tracking_enabled !== false}/><PromotionView key={row.listing.id} listingID={row.listing.id}/><AlertView key={`alerts-${row.listing.id}`} listingID={row.listing.id}/></article>; })}</div>
</section>; }