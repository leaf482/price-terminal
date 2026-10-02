import { parseAlert, type Alert } from './alerts.ts';
import type { Currency } from './api.ts';
type Context = { product_id: string; product_name: string; retailer_id: string; retailer_name: string };
export type OverviewAlert = Context & { alert: Alert; last_triggered_at: string | null };
export type OverviewEvent = Context & { alert_id: string; listing_id: string; observation_id: string; kind: Alert['kind']; minor_units: number; currency: Currency; triggered_at: string };
export type AlertOverview = { alerts: OverviewAlert[]; events: OverviewEvent[]; alerts_truncated: boolean; events_truncated: boolean };
function object(v: unknown): Record<string, unknown> { if (!v || typeof v !== 'object' || Array.isArray(v)) throw new Error('Invalid overview'); return v as Record<string, unknown>; }
function context(v: Record<string, unknown>) { for (const key of ['product_id','product_name','retailer_id','retailer_name']) if (typeof v[key] !== 'string') throw new Error('Invalid overview context'); }
function timestamp(v: unknown) { if (typeof v !== 'string' || !Number.isFinite(Date.parse(v))) throw new Error('Invalid overview time'); }
export function parseAlertOverview(value: unknown): AlertOverview {
 const v=object(value);if (!Array.isArray(v.alerts)||v.alerts.length>100||!Array.isArray(v.events)||v.events.length>20||typeof v.alerts_truncated!=='boolean'||typeof v.events_truncated!=='boolean') throw new Error('Invalid overview bounds');
 for (const row of v.alerts) { const r=object(row); context(r);parseAlert(r.alert);if(r.last_triggered_at!==null)timestamp(r.last_triggered_at); }
 for(const row of v.events){const r=object(row);context(r);for(const key of ['alert_id','listing_id','observation_id'])if(typeof r[key]!=='string')throw new Error('Invalid event identity');timestamp(r.triggered_at);if(!['target','drop','historical_low'].includes(String(r.kind))||!['USD','JPY'].includes(String(r.currency))||!Number.isSafeInteger(r.minor_units)||(r.minor_units as number)<0)throw new Error('Invalid event value');}
 return v as AlertOverview;
}
export type AlertFilter = 'all'|'enabled'|'disabled'|'triggered'|'never';
export function filterAlerts(rows: OverviewAlert[],filter: AlertFilter,kind: string) {return rows.filter(r=>(kind==='all'||r.alert.kind===kind)&&(filter==='all'||(filter==='enabled'&&r.alert.enabled)||(filter==='disabled'&&!r.alert.enabled)||(filter==='triggered'&&r.last_triggered_at!==null)||(filter==='never'&&r.last_triggered_at===null)));}
