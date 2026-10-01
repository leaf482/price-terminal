import Link from "next/link";
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
    const missing = [...new Set(prices.listings.map(r => r.listing.retailer_id))].filter(id => !(id in retailerNames));
    await Promise.all(missing.map(async id => { const r = await api(`/retailers/${encodeURIComponent(id)}`, value => parseRetailers([value])[0]); retailerNames[id] = r.name; }));
    return <main><Link href="/">← Products</Link> <Link href="/catalog">Manage catalog</Link><header><p className="eyebrow">{product.brand || "PRODUCT"}</p><h1>{product.name || product.id}</h1><p>{product.model} · ID: {product.id}</p></header>
 <ArchiveControls id={product.id} archived={product.archived === true}/><MetadataEditor kind="products" record={product}/><h2>Current observations</h2><p>Listing source fields are read-only. To change Product, Retailer, URL or retailer product ID, disable the old Listing if needed and create a new Listing.</p>{!prices.listings.length && <p>No Listings for this product yet.</p>}
 <ListingComparison prices={prices} retailers={retailerNames} revision={revision}/>
 <HistoryView key={`${product.id}:${revision}`} listings={prices.listings.map(x => ({ id: x.listing.id, retailer: retailerNames[x.listing.retailer_id] || x.listing.retailer_id }))}/></main>;
}
