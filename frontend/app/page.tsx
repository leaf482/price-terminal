import Link from "next/link";
import { api, parseProducts, parsePrices } from "../lib/api";
import { formatPrice } from "../lib/prices";
export const dynamic = "force-dynamic";
export default async function Home() {
    const products = await api("/products?limit=20", parseProducts);
    const cards = await Promise.all(products.map(async (product) => ({ product, prices: await api(`/products/${encodeURIComponent(product.id)}/prices`, parsePrices).catch(() => null) })));
    return <main><header><p className="eyebrow">PRICE TERMINAL</p><h1>Your tracked products</h1><p>Observed prices, with their history and context.</p></header>
 {!products.length && <p className="panel">No products yet. Create products and Listings through the backend APIs to begin.</p>}
 <div className="grid">{cards.map(({ product: p, prices }) => <article className="panel" key={p.id}><p className="muted">{p.brand || "Brand not specified"} · {p.model || p.id}</p><h2><Link href={`/products/${encodeURIComponent(p.id)}`}>{p.name || p.id}</Link></h2><p className="price">{prices?.best_price ? formatPrice(prices.best_price.minor_units, prices.best_price.currency) : "No comparable price"}</p><p className="muted">{prices ? prices.comparison_status.replaceAll("_", " ") : "Current prices could not be loaded"}</p></article>)}</div>
 {products.length === 20 && <p>Showing the first 20 products by ID. This MVP catalog is bounded.</p>}</main>;
}
