"use client";
import { useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import type { Current } from '../../../lib/api';
import { postCatalog } from '../../../lib/catalog';
export default function CollectionControls({ id, status, trackingEnabled = true }: {
    id: string;
    status: Current['collection'];
	trackingEnabled?: boolean;
}) {
    const router = useRouter(), pending = useRef(false);
    const [busy, setBusy] = useState(false), [message, setMessage] = useState(''), [error, setError] = useState('');
    return <section><h4>Collection status</h4><p>Consecutive failed attempts: {status.consecutive_failures ?? 0} (resets on success or restart)</p><p>Status: {status.state} {status.error && `(${status.error})`}</p><p>Last attempted: {status.last_attempted_at || 'Never'}</p><p>Last successful: {status.last_successful_at || 'Never'}</p><p>Process-local status resets when the backend restarts. Observation time above is source time, not the latest collection attempt.</p>
 {status.state === 'inactive' && <p>No provider configured. Creating a Listing does not enable collection; the backend COLLECTOR_CONFIG must include it (currently Fake fixtures only).</p>}
 <button disabled={!trackingEnabled || busy || status.state === 'collecting' || status.state === 'inactive' || status.state === 'disabled'} onClick={async () => { if (pending.current || !trackingEnabled || status.state === 'disabled')
        return; pending.current = true; setBusy(true); setMessage(''); setError(''); try {
        await postCatalog(`/listings/${encodeURIComponent(id)}/collect`);
        setMessage('Collection succeeded. Source observation time is preserved.');
    }
    catch (e) {
        setError(e instanceof Error ? e.message : 'Collection failed.');
    }
    finally {
        pending.current = false;
        setBusy(false);
        router.refresh();
    } }}>{busy ? 'Collecting…' : 'Refresh price'}</button> <button onClick={() => router.refresh()}>Reload status</button>{message && <p role="status">{message}</p>}{error && <p role="alert">{error}</p>}</section>;
}
