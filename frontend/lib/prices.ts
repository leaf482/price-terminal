import type { Currency, Observation } from "./api.ts";
// Storage and formatting stay in integer minor units. Only SVG coordinates use floats.
export function formatPrice(amount: number, currency: Currency): string {
    if (!Number.isSafeInteger(amount) || amount < 0)
        throw new Error("Invalid exact amount");
    const value = BigInt(amount), scale = currency === "USD" ? BigInt(100) : BigInt(1);
    return `${currency} ${value / scale}${currency === "USD" ? "." + (value % scale).toString().padStart(2, "0") : ""}`;
}
export function basis(o: Observation | null) { return o?.offer_price !== undefined ? { amount: o.offer_price, label: "Offer price", currency: o.currency! } : o?.sale_price !== undefined ? { amount: o.sale_price, label: "Sale price", currency: o.currency! } : null; }
export function historyPoints(rows: Observation[]) {
    return rows.map(o => ({ time: Date.parse(o.observed_at), timestamp: o.observed_at, price: basis(o) }));
}
// Lines are visual guides only. Never bridge missing prices, currency/basis
// changes, or more than 15 minutes without an observation. No points are added.
export function historySegments(rows: Observation[]) {
    const segments: ReturnType<typeof historyPoints>[] = [];
    let segment: ReturnType<typeof historyPoints> = [];
    for (const point of historyPoints(rows)) {
        const previous = segment.at(-1);
        if (!point.price || (previous && (point.price.currency !== previous.price?.currency || point.price.label !== previous.price?.label || point.time - previous.time > 15 * 60 * 1000))) {
            if (segment.length)
                segments.push(segment);
            segment = [];
        }
        if (point.price)
            segment.push(point);
    }
    if (segment.length)
        segments.push(segment);
    return segments;
}
export function safeSource(url: string) { try {
    const parsed = new URL(url);
    return ["http:", "https:"].includes(parsed.protocol) ? url : null;
}
catch {
    return null;
} }
