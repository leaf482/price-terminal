import { jsonRequest } from './json-request.ts';
import type { Currency } from './api.ts';
export type Alert = {
    id: string;
    listing_id: string;
    kind: 'target' | 'drop' | 'historical_low';
    currency: Currency;
    threshold_minor_units?: number;
    drop_basis_points?: number;
    enabled: boolean;
    require_in_stock: boolean;
};
export type AlertEvent = {
    alert_id: string;
    listing_id: string;
    observation_id: string;
    kind: string;
    triggered_at: string;
    observed_at: string;
    minor_units: number;
    currency: Currency;
    price_basis: string;
    comparison_minor_units: number | null;
};
export type Events = {
    events: AlertEvent[];
    truncated: boolean;
};
function obj(v: unknown): Record<string, unknown> { if (!v || typeof v !== 'object' || Array.isArray(v))
    throw new Error('Invalid alert response'); return v as Record<string, unknown>; }
function text(v: unknown) { if (typeof v !== 'string')
    throw new Error('Invalid alert text'); }
function amount(v: unknown) { if (!Number.isSafeInteger(v) || (v as number) < 0)
    throw new Error('Invalid exact alert amount'); }
function currency(v: unknown) { if (v !== 'USD' && v !== 'JPY')
    throw new Error('Unsupported alert currency'); }
export function parseAlert(v: unknown): Alert { const a = obj(v); text(a.id); text(a.listing_id); currency(a.currency); if (!['target', 'drop', 'historical_low'].includes(String(a.kind)) || typeof a.enabled !== 'boolean' || typeof a.require_in_stock !== 'boolean')
    throw new Error('Invalid alert configuration'); if (a.kind === 'target')
    amount(a.threshold_minor_units); if (a.kind === 'drop') {
    amount(a.drop_basis_points);
    if ((a.drop_basis_points as number) < 1 || (a.drop_basis_points as number) > 10000)
        throw new Error('Invalid drop');
} ; return a as Alert; }
export function parseAlerts(v: unknown): Alert[] { if (!Array.isArray(v))
    throw new Error('Invalid alert list'); return v.map(parseAlert); }
export function parseEvents(v: unknown): Events { const data = obj(v); if (!Array.isArray(data.events) || typeof data.truncated !== 'boolean')
    throw new Error('Invalid events'); for (const event of data.events) {
    const e = obj(event);
    for (const key of ['alert_id', 'listing_id', 'observation_id', 'kind', 'triggered_at', 'observed_at', 'price_basis'])
        text(e[key]);
    amount(e.minor_units);
    currency(e.currency);
    if (e.comparison_minor_units !== null)
        amount(e.comparison_minor_units);
} return data as Events; }
// Decimal input is converted by string/BigInt arithmetic, never floating money.
export function decimalUnits(text: string, digits: number): number { if (!/^\d+(\.\d+)?$/.test(text))
    throw new Error('Enter a nonnegative decimal amount'); const [whole, fraction = ''] = text.split('.'); if (fraction.length > digits)
    throw new Error(`Use at most ${digits} decimal places`); const n = BigInt(whole + fraction.padEnd(digits, '0')); if (n > BigInt(Number.MAX_SAFE_INTEGER))
    throw new Error('Amount exceeds exact UI range'); return Number(n); }
export function alertBody(kind: Alert['kind'], currency: Currency, value: string, stock: boolean) { const body: {
    kind: Alert['kind'];
    currency: Currency;
    enabled: boolean;
    require_in_stock: boolean;
    threshold_minor_units?: number;
    drop_basis_points?: number;
} = { kind, currency, enabled: true, require_in_stock: stock }; if (kind === 'target')
    body.threshold_minor_units = decimalUnits(value, currency === 'USD' ? 2 : 0); if (kind === 'drop') {
    body.drop_basis_points = decimalUnits(value, 2);
    if (body.drop_basis_points < 1 || body.drop_basis_points > 10000)
        throw new Error('Drop must be 0.01–100%');
} ; return body; }
export async function saveAlert(listing: string, body: unknown, id?: string): Promise<Alert> { const path = `/api/listings/${encodeURIComponent(listing)}/alerts${id ? '/' + encodeURIComponent(id) : ''}`; const response = await jsonRequest(path, id ? 'PATCH' : 'POST', body); if (!response.ok)
    throw new Error(`Could not save alert (${response.status})`); return parseAlert(obj(await response.json()).data); }
