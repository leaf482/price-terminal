"use client";
import Link from 'next/link';
import { useEffect, useRef, useState } from 'react';
import { api } from '../../lib/api';
import { saveAlert } from '../../lib/alerts';
import { formatPrice } from '../../lib/prices';
import { parseAlertOverview, filterAlerts, type AlertOverview, type AlertFilter } from '../../lib/alert-overview';
export default function AlertsPage(){
 const [data,setData]=useState<AlertOverview|null>(null),[error,setError]=useState(''),[filter,setFilter]=useState<AlertFilter>('all'),[kind,setKind]=useState('all'),[revision,setRevision]=useState(0),[busy,setBusy]=useState(false);const pending=useRef(false);
 useEffect(()=>{let active=true;const c=new AbortController();const timer=setTimeout(()=>c.abort(),10000);api('/alerts/overview',parseAlertOverview,c.signal).then(v=>{if(active){setData(v);setError('')}}).catch(()=>{if(active)setError('Could not load alerts. Retry.')}).finally(()=>clearTimeout(timer));return()=>{active=false;c.abort();clearTimeout(timer)}},[revision]);
 return <main><h1>Alerts</h1><p>Based on observed offer price, otherwise sale price. Promotions and EffectivePrice are excluded. Create a new alert on Listing detail to change conditions.</p><button onClick={()=>setRevision(n=>n+1)}>Reload alerts</button>
 {error&&<p role="alert">{error}</p>}{!data&&!error&&<p role="status">Loading alerts…</p>}
 <label>Status <select value={filter} onChange={e=>setFilter(e.target.value as AlertFilter)}>{['all','enabled','disabled','triggered','never'].map(x=><option key={x} value={x}>{x==='never'?'Never triggered':x==='triggered'?'Triggered at least once':x}</option>)}</select></label>
 <label>Type <select value={kind} onChange={e=>setKind(e.target.value)}>{['all','target','drop','historical_low'].map(x=><option key={x} value={x}>{x.replaceAll('_',' ')}</option>)}</select></label>
 {data&&<>{data.alerts_truncated&&<p>First 100 alerts by ID. Filters apply to loaded alerts only.</p>}{!filterAlerts(data.alerts,filter,kind).length&&<p>No alerts match these filters.</p>}
 {filterAlerts(data.alerts,filter,kind).map(row=>{const a=row.alert;return <article className="panel" key={a.id}><h2>{row.product_name||row.product_id} · {row.retailer_name||row.retailer_id}</h2><p>Product ID: {row.product_id} · Retailer ID: {row.retailer_id}</p><Link href={`/listings/${encodeURIComponent(a.listing_id)}`}>Listing {a.listing_id}</Link><p>Alert {a.id}: {a.kind.replaceAll('_',' ')} · {a.currency}</p>
 <p>{a.kind==='target'?`Observed price ≤ ${formatPrice(a.threshold_minor_units!,a.currency)}`:a.kind==='drop'?`Price drop ≥ ${a.drop_basis_points!/100}% from the previous comparable observation`:'Lower than all earlier comparable observations'} · {a.require_in_stock?'Requires in stock':'No stock requirement'}</p>
 <p>{a.enabled?'Enabled':'Disabled'} · Last triggered: {row.last_triggered_at||'Never'}</p><button disabled={busy} onClick={async()=>{if(pending.current)return;pending.current=true;setBusy(true);setError('');try{const saved = await saveAlert(a.listing_id,{enabled:!a.enabled},a.id);setData(current => current ? {...current, alerts: current.alerts.map(row => row.alert.id === saved.id ? {...row, alert: saved} : row)} : current);setRevision(n=>n+1)}catch{setError('Could not change alert state. Retry.')}finally{pending.current=false;setBusy(false)}}}>{a.enabled?'Disable':'Enable'}</button></article>})}
 <h2>Recent triggered events</h2><p>Newest 20 stored events, independent of filters above. Events remain historical evidence even if an observation was later invalidated.</p>{data.events_truncated&&<p>Additional older events are available on Listing detail.</p>}{!data.events.length&&<p>No triggered events.</p>}
 {data.events.map(e=><article className="panel" key={JSON.stringify([e.alert_id,e.observation_id])}><h3>{e.product_name||e.product_id} · {e.retailer_name||e.retailer_id}</h3><p>Product ID: {e.product_id} · Retailer ID: {e.retailer_id}</p><Link href={`/listings/${encodeURIComponent(e.listing_id)}`}>Listing {e.listing_id}</Link><p>{e.kind.replaceAll('_',' ')} · Observed trigger value: {formatPrice(e.minor_units,e.currency)}</p><p>Triggered: {e.triggered_at} · Alert: {e.alert_id}</p></article>)}
 </>}
 </main>;
}
