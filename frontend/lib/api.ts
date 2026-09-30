export type Currency = "USD" | "JPY";
export type Product = {
    id: string;
    name: string;
    brand: string;
    model: string;
};
export type Observation = {
    observed_at: string;
    source: string;
    stock: "unknown" | "in_stock" | "out_of_stock";
    currency?: Currency;
    msrp?: number;
    msrp_source?: string;
    retailer_list_price?: number;
    sale_price?: number;
    offer_price?: number;
};
export type Current = {
    listing: {
        id: string;
        product_id: string;
        retailer_id: string;
        url: string;
        retailer_product_id?: string;
    };
    observation: Observation | null;
    freshness: string;
    collection: {
        state: string;
        active?: boolean;
        last_attempted_at?: string;
        last_successful_at?: string;
        error?: string;
    };
};
export type Prices = {
    product_id: string;
    listings: Current[];
    best_price: {
        minor_units: number;
        currency: Currency;
        basis: string;
        listing_id: string;
    } | null;
    comparison_status: string;
};
export type History = {
    listing_id: string;
    observations: Observation[];
    truncated: boolean;
    historical_low: {
        minor_units: number;
        currency: Currency;
    } | null;
    period_change_percent: string | null;
};
function object(x: unknown): Record<string, unknown> {
    if (!x || typeof x !== "object" || Array.isArray(x))
        throw new Error("Invalid API response");
    return x as Record<string, unknown>;
}
function text(x: unknown): asserts x is string { if (typeof x !== "string")
    throw new Error("Invalid API text"); }
function amount(x: unknown) { if (typeof x !== "number" || !Number.isSafeInteger(x) || x < 0)
    throw new Error("Price exceeds supported exact integer range or is invalid"); }
function currency(x: unknown) { if (x !== "USD" && x !== "JPY")
    throw new Error("Unsupported currency"); }
function money(x: unknown) { const v = object(x); amount(v.minor_units); currency(v.currency); }
export function parseObservation(x: unknown): Observation {
    const v = object(x);
    text(v.observed_at);
    if (!Number.isFinite(Date.parse(v.observed_at)))
        throw new Error("Invalid observation time");
    text(v.source);
    if (!["unknown", "in_stock", "out_of_stock"].includes(String(v.stock)))
        throw new Error("Invalid stock");
    let hasPrice = false;
    for (const key of ["msrp", "retailer_list_price", "sale_price", "offer_price"]) {
        if (v[key] !== undefined) {
            amount(v[key]);
            hasPrice = true;
        }
    }
    if (hasPrice)
        currency(v.currency);
    else if (v.currency !== undefined)
        throw new Error("Stock-only currency");
    if (v.msrp !== undefined)
        text(v.msrp_source);
    return v as Observation;
}
export function parseProduct(x: unknown): Product { const v = object(x); for (const key of ["id", "name", "brand", "model"])
    text(v[key]); return v as Product; }
export function parseProducts(x: unknown): Product[] { if (!Array.isArray(x))
    throw new Error("Invalid product list"); return x.map(parseProduct); }
export function parsePrices(x: unknown): Prices {
    const v = object(x);
    text(v.product_id);
    text(v.comparison_status);
    if (!Array.isArray(v.listings))
        throw new Error("Invalid listings");
    for (const item of v.listings) {
        const row = object(item), l = object(row.listing);
        for (const k of ["id", "product_id", "retailer_id", "url"])
            text(l[k]);
        if (l.retailer_product_id !== undefined)
            text(l.retailer_product_id);
        text(row.freshness);
        text(object(row.collection).state);
        const status=object(row.collection);
        for(const k of ['last_attempted_at','last_successful_at','error']) if(status[k]!==undefined) text(status[k]);
        if (row.observation !== null)
            parseObservation(row.observation);
    }
    if (v.best_price !== null) {
        money(v.best_price);
        const best = object(v.best_price);
        text(best.basis);
        text(best.listing_id);
    }
    return v as Prices;
}
export function parseHistory(x: unknown): History {
    const v = object(x);
    text(v.listing_id);
    if (!Array.isArray(v.observations) || typeof v.truncated !== "boolean")
        throw new Error("Invalid history");
    v.observations.forEach(parseObservation);
    if (v.historical_low !== null)
        money(v.historical_low);
    if (v.period_change_percent !== null && (typeof v.period_change_percent !== "string" || !/^-?\d+\.\d{2}$/.test(v.period_change_percent)))
        throw new Error("Invalid change");
    return v as History;
}
export async function api<T>(path: string, parse: (x: unknown) => T, signal?: AbortSignal): Promise<T> {
    const base = typeof window === "undefined" ? (process.env.BACKEND_URL || "http://127.0.0.1:8080") : "/api";
    const response = await fetch(`${base}${path}`, { cache: "no-store", signal: signal || AbortSignal.timeout(10000) });
    if (!response.ok)
        throw new Error(`Request failed (${response.status}). Check the backend and database.`);
    return parse(object(await response.json()).data);
}
