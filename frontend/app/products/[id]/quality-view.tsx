"use client";
import { useEffect, useRef, useState } from 'react';
import { api } from '../../../lib/api';
import { formatPrice } from '../../../lib/prices';
import { parseAuditList, parseAudit, invalidateObservation, type Audit, type AuditList } from '../../../lib/quality';
export default function QualityView({ listingID, table = false }: {
    listingID: string;
    table?: boolean;
}) {
    const [data, setData] = useState<AuditList | null>(null), [error, setError] = useState(''), [lookup, setLookup] = useState(''), [found, setFound] = useState<Audit | null>(null), [busy, setBusy] = useState(false), [revision, setRevision] = useState(0);
    const pending = useRef(false);
    useEffect(() => { let active = true; const c = new AbortController(); const timer = setTimeout(() => c.abort(), 10000); api(`/listings/${encodeURIComponent(listingID)}/observations`, parseAuditList, c.signal).then(v => { if (active) {
        setData(v);
        setError('');
    } }).catch(() => { if (active)
        setError('Could not read observation audit.'); }).finally(() => clearTimeout(timer)); return () => { active = false; c.abort(); clearTimeout(timer); }; }, [listingID, revision]);
    async function invalidate(id: string, reason: string) { if (pending.current)
        return; pending.current = true; setBusy(true); setError(''); try {
        await invalidateObservation(id, reason);
        window.location.reload();
    }
    catch (e) {
        setError(e instanceof Error ? e.message : 'Invalidation failed.');
    }
    finally {
        pending.current = false;
        setBusy(false);
    } }
    return <section><h4>Observation audit / data quality</h4><p>Invalidation excludes a record from current prices, history and future alert evaluation. Original facts and existing alert events remain. There is no undo; the first reason is retained.</p><button onClick={() => setRevision(n => n + 1)}>Reload audit</button>
 <form onSubmit={async (e) => { e.preventDefault(); if (pending.current)
        return; pending.current = true; setBusy(true); setError(''); setFound(null); try {
        const a = await api(`/observations/${encodeURIComponent(lookup)}`, parseAudit);
        if (a.listing_id !== listingID)
            throw new Error('Observation belongs to another Listing.');
        setFound(a);
    }
    catch {
        setError('Observation not found for this Listing, or the backend is unavailable.');
    }
    finally {
        pending.current = false;
        setBusy(false);
    } }}><label>Inspect an exact observation ID <input required value={lookup} onChange={e => setLookup(e.target.value)}/></label> <button disabled={busy}>Inspect</button></form>
 {error && <p role="alert">{error}</p>}{!data && !error && <p>Loading audit…</p>}{found && <AuditRecord record={found} busy={busy} onInvalidate={invalidate}/>}{data?.truncated && <p>Newest 100 observations shown. Inspect older records by exact ID or use Export CSV for a larger audit export (up to 10,000 rows).</p>}{data?.observations.length === 0 && <p>No observations.</p>}{data && table ? <AuditTable records={data.observations} busy={busy} onInvalidate={invalidate}/> : data?.observations.filter(a => a.id !== found?.id).map(a => <AuditRecord key={a.id} record={a} busy={busy} onInvalidate={invalidate}/>)}</section>;
}
function price(record: Audit, amount: number | undefined) { return amount === undefined ? 'Missing' : formatPrice(amount, record.observation.currency!); }
export function AuditTable({ records, busy, onInvalidate }: { records: Audit[]; busy: boolean; onInvalidate: (id: string, reason: string) => Promise<void> }) {
    return <div style={{ overflowX: 'auto' }}><table><caption>Observation history audit — newest first, including invalidated facts. Charts and current prices exclude invalidated observations.</caption>
        <thead><tr>{['Observed time','Offer price','Sale price','Retailer list price','MSRP','Currency','Stock','Source / provenance','Validity','Detail'].map(label => <th scope="col" key={label}>{label}</th>)}</tr></thead>
        <tbody>{records.map(a => { const o=a.observation; return <tr key={a.id}>
            <td>{o.observed_at}</td><td>{price(a,o.offer_price)}</td><td>{price(a,o.sale_price)}</td><td>{price(a,o.retailer_list_price)}</td><td>{price(a,o.msrp)}</td><td>{o.currency || 'Missing'}</td><td>{o.stock.replaceAll('_',' ')}</td><td style={{whiteSpace:'pre-wrap'}}>{o.source}</td><td>{a.valid?'Valid':'Invalidated'}</td><td><AuditRecord record={a} busy={busy} onInvalidate={onInvalidate}/></td>
        </tr>; })}</tbody>
    </table></div>;
}
export function AuditRecord({ record: a, busy, onInvalidate }: {
    record: Audit;
    busy: boolean;
    onInvalidate: (id: string, reason: string) => Promise<void>;
}) { const o=a.observation; return <details><summary>{a.id} · {o.observed_at} · {a.valid ? 'Valid' : 'Invalidated'}</summary><dl>
 <dt>Observation ID</dt><dd>{a.id}</dd><dt>Listing ID</dt><dd>{a.listing_id}</dd><dt>Observed timestamp</dt><dd>{o.observed_at}</dd>
 <dt>Offer price</dt><dd>{price(a,o.offer_price)}</dd><dt>Sale price</dt><dd>{price(a,o.sale_price)}</dd><dt>Retailer list price</dt><dd>{price(a,o.retailer_list_price)}</dd><dt>MSRP</dt><dd>{price(a,o.msrp)}</dd><dt>MSRP source / evidence</dt><dd style={{whiteSpace:'pre-wrap'}}>{o.msrp_source || 'Missing'}</dd>
 <dt>Currency</dt><dd>{o.currency || 'Missing'}</dd><dt>Stock</dt><dd>{o.stock.replaceAll('_',' ')}</dd><dt>Source / provenance</dt><dd style={{whiteSpace:'pre-wrap'}}>{o.source}</dd>
 </dl>{a.valid ? <form onSubmit={e => { e.preventDefault(); const reason = String(new FormData(e.currentTarget).get('reason') || ''); void onInvalidate(a.id, reason); }}><label>Reason <input name="reason" required maxLength={1000}/></label> <button disabled={busy}>Invalidate observation</button></form> : <p style={{whiteSpace:'pre-wrap'}}>Invalidated: {a.invalidated_at} · Reason: {a.invalidation_reason}</p>}</details>; }
