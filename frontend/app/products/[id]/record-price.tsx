"use client";
import { useRef, useState } from 'react';
import { observationBody, recordObservation } from '../../../lib/observations';

export default function RecordPrice({ listingID }: { listingID: string }) {
    const pending = useRef(false), requestID = useRef('');
    const [busy, setBusy] = useState(false), [error, setError] = useState('');
    const [time, setTime] = useState('');
    return <details onToggle={e => { if (e.currentTarget.open && !time) setTime(new Date().toISOString()); }}><summary>Record price</summary>
        <p>Manual observed facts only. Blank prices stay missing; 0 is explicit zero. Promotions and EffectivePrice are separate.</p>
        <form onSubmit={async e => {
            e.preventDefault(); if (pending.current) return;
            const form = new FormData(e.currentTarget);
            pending.current = true; setBusy(true); setError('');
            try {
                // Retain this ID on an uncertain response; retries cannot append duplicates.
                requestID.current ||= crypto.randomUUID();
                await recordObservation(listingID, observationBody(form, requestID.current));
                // Reload all server and client reads, including backdated history and events.
                window.location.reload();
            } catch (err) { setError(err instanceof Error ? err.message : 'Unable to record observation.'); }
            finally { pending.current = false; setBusy(false); }
        }}>
            <fieldset disabled={busy}>
                <label>Observed time (ISO 8601 with timezone) <input name="observed_at" required value={time} onChange={e => setTime(e.target.value)} /></label>
                <label>Source evidence <input name="source" required placeholder="Listing URL / store receipt / notes" /></label>
                <label>Stock <select name="stock" defaultValue="unknown"><option value="unknown">Unknown</option><option value="in_stock">In stock</option><option value="out_of_stock">Out of stock</option></select></label>
                <label>Currency (only used for prices) <select name="currency" defaultValue=""><option value="">No currency</option><option>USD</option><option>JPY</option></select></label>
                {['offer_price', 'sale_price', 'retailer_list_price', 'msrp'].map(field => <label key={field}>{field === 'offer_price' ? 'Current / offer price' : field.replaceAll('_', ' ')} <input name={field} inputMode="decimal" placeholder="Missing" /></label>)}
                <label>Explicit MSRP evidence <input name="msrp_source" /></label>
                <button type="submit">{busy ? 'Recording…' : 'Record observation'}</button>
            </fieldset>
            {error && <p role="alert">{error}</p>}
        </form>
    </details>;
}
