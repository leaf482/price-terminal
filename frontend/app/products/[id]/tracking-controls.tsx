"use client";
import { useRef, useState } from 'react';
import { setTracking } from '../../../lib/tracking';

export default function TrackingControls({ id, enabled }: { id: string; enabled: boolean }) {
    const pending = useRef(false);
    const [busy, setBusy] = useState(false), [error, setError] = useState('');
    return <section><h4>Tracking {enabled ? 'enabled' : 'disabled'}</h4>
        <p>Disabling stops provider collection. History, prices, promotions and alerts remain available. Explicit Record price entries are still allowed.</p>
        <button disabled={busy} onClick={async () => {
            if (pending.current) return;
            pending.current = true; setBusy(true); setError('');
            try { await setTracking(id, !enabled); window.location.reload(); }
            catch (e) { setError(e instanceof Error ? e.message : 'Unable to change tracking.'); }
            finally { pending.current = false; setBusy(false); }
        }}>{busy ? 'Saving…' : enabled ? 'Disable tracking' : 'Enable tracking'}</button>
        {error && <p role="alert">{error}</p>}
    </section>;
}
