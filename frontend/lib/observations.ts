import { jsonRequest } from './json-request.ts';
import { decimalUnits } from './alerts.ts';

export function observationBody(form: FormData, id: string) {
    const text = (name: string) => String(form.get(name) ?? '').trim();
    const currency = text('currency');
    // Preserve evidence exactly apart from surrounding whitespace. Date would
    // roll invalid calendar dates forward and discard sub-millisecond precision.
    // The backend validates RFC3339/calendar semantics.
    const observedAt = text('observed_at');
    if (!observedAt || !/T.*(Z|[+-]\d{2}:\d{2})$/.test(observedAt)) throw new Error('Observation time must include an explicit timezone.');
    if (!text('source')) throw new Error('Source evidence is required.');
    const prices: Record<string, { minor_units: number; currency: string }> = {};
    for (const field of ['offer_price', 'sale_price', 'retailer_list_price', 'msrp']) {
        if (!text(field)) continue;
        if (!['USD', 'JPY'].includes(currency)) throw new Error('Select USD or JPY for monetary values.');
        prices[field] = { minor_units: decimalUnits(text(field), currency === 'USD' ? 2 : 0), currency };
    }
    if (prices.msrp && !text('msrp_source')) throw new Error('MSRP requires explicit manufacturer-price evidence.');
    return { id, observed_at: observedAt, source: text('source'), stock: text('stock'), ...prices, msrp_source: text('msrp_source') };
}

export async function recordObservation(listing: string, body: ReturnType<typeof observationBody>) {
    const response = await jsonRequest(`/api/listings/${encodeURIComponent(listing)}/observations`, 'POST', body);
    if (response.ok) return;
    const payload = await response.json().catch(() => null);
    throw new Error(payload?.error?.message || `Unable to record observation (${response.status}).`);
}
