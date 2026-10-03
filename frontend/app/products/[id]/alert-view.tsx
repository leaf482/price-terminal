"use client";
import { useEffect, useState } from 'react';
import { api, type Currency } from '../../../lib/api';
import { formatPrice } from '../../../lib/prices';
import { alertBody, parseAlerts, parseEvents, saveAlert, type Alert, type Events } from '../../../lib/alerts';
export default function AlertView({ listingID }: {
    listingID: string;
}) {
    const [data, setData] = useState<{
        alerts: Alert[];
        events: Events;
    } | null>(null), [error, setError] = useState(''), [busy, setBusy] = useState(false), [refresh, setRefresh] = useState(0);
    const [kind, setKind] = useState<Alert['kind']>('target'), [currency, setCurrency] = useState<Currency>('USD'), [value, setValue] = useState(''), [stock, setStock] = useState(true);
    useEffect(() => { let active = true; const controller = new AbortController(); const timer = setTimeout(() => controller.abort(), 10000); const path = `/listings/${encodeURIComponent(listingID)}`; Promise.all([api(`${path}/alerts`, parseAlerts, controller.signal), api(`${path}/alert-events`, parseEvents, controller.signal)]).then(([alerts, events]) => { if (active) {
        setData({ alerts, events });
        setError('');
    } }).catch(() => { if (active)
        setError('Could not load alerts/events. Refresh to retry.'); }).finally(() => clearTimeout(timer)); return () => { active = false; controller.abort(); clearTimeout(timer); }; }, [listingID, refresh]);
    async function save(body: unknown, id?: string) { setBusy(true); setError(''); try {
        await saveAlert(listingID, body, id);
        setRefresh(v => v + 1);
    }
    catch (e) {
        setError(e instanceof Error ? e.message : 'Could not save alert');
    }
    finally {
        setBusy(false);
    } }
    return <section><h4>Observed-price alerts</h4><p>Based on offer price, otherwise sale price. Promotions and EffectivePrice are excluded. Events are stored here; nothing is sent externally.</p>
 <form onSubmit={e => { e.preventDefault(); try {
        void save(alertBody(kind, currency, value, stock));
    }
    catch (err) {
        setError(err instanceof Error ? err.message : 'Invalid alert');
    } }}>
 <label>Alert type <select value={kind} onChange={e => setKind(e.target.value as Alert['kind'])}><option value="target">Target price ≤ threshold</option><option value="drop">Price drop from previous comparable observation</option><option value="historical_low">New historical low</option></select></label><br />
 <label>Currency <select value={currency} onChange={e => setCurrency(e.target.value as Currency)}><option>USD</option><option>JPY</option></select></label><br />
 {kind !== 'historical_low' && <label>{kind === 'target' ? 'Target amount' : 'Drop percentage'} <input required inputMode="decimal" value={value} onChange={e => setValue(e.target.value)}/></label>}<br />
 <label><input type="checkbox" checked={stock} onChange={e => setStock(e.target.checked)}/>Require in-stock observations</label><p><button disabled={busy || !data}>Create alert</button> <button type="button" disabled={busy} onClick={() => setRefresh(v => v + 1)}>Refresh alerts/events</button></p></form>
 {error && <p role="alert">{error}</p>}{!data && !error && <p role="status">Loading alerts…</p>}
 {data && <><h5>Configured alerts</h5>{!data.alerts.length && <p>No alerts configured.</p>}<ul>{data.alerts.map(a => <li key={a.id}><AlertConditions alert={a}/> · {a.enabled ? 'Enabled' : 'Disabled'} <button aria-label={`${a.enabled ? "Disable" : "Enable"} alert ${a.id}`} disabled={busy} onClick={() => void save({ enabled: !a.enabled }, a.id)}>{a.enabled ? 'Disable' : 'Enable'}</button></li>)}</ul><EventList data={data.events}/></>}
 </section>;
}
export function AlertConditions({ alert: a }: {
    alert: Alert;
}) { return <span>{a.kind === 'target' ? `Price ≤ ${formatPrice(a.threshold_minor_units!, a.currency)}` : a.kind === 'drop' ? `Drop ≥ ${Math.trunc(a.drop_basis_points! / 100)}.${String(a.drop_basis_points! % 100).padStart(2, '0')}% (${a.currency})` : `New historical low (${a.currency})`} · {a.require_in_stock ? 'In stock required' : 'Any stock state'} · ID: {a.id}</span>; }
export function EventList({ data }: {
    data: Events;
}) { return <div><h5>Triggered events</h5>{data.truncated && <p>Newest 100 events shown.</p>}{!data.events.length ? <p>No triggered events yet.</p> : <ul>{data.events.map(e => <li key={`${e.alert_id}:${e.observation_id}`}><strong>{e.kind.replaceAll('_', ' ')}: {formatPrice(e.minor_units, e.currency)}</strong> ({e.price_basis})<br />Alert: {e.alert_id} · Observation: {e.observation_id}<br />Observed: {e.observed_at}<br />Triggered: {e.triggered_at}{e.comparison_minor_units !== null && <p>Earlier comparable value: {formatPrice(e.comparison_minor_units, e.currency)}</p>}</li>)}</ul>}</div>; }
