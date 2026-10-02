'use client';
import { useEffect, useState } from 'react';
import { api } from '../../../lib/api';
import { parseAttempts, type CollectionAttempt } from '../../../lib/collection-attempts';

export default function AttemptHistory({ listingID }: { listingID: string }) {
    const [items, setItems] = useState<CollectionAttempt[] | null>(null);
    const [error, setError] = useState(false);
    useEffect(() => {
        let active = true;
        const controller = new AbortController();
        api(`/listings/${encodeURIComponent(listingID)}/collection-attempts`, parseAttempts, controller.signal)
            .then(v => { if (active) { setItems(v); setError(false); } })
            .catch(() => { if (active) setError(true); });
        return () => { active = false; controller.abort(); };
    }, [listingID]);
    return <section><h2>Collection attempt history</h2>
        <p>Most recent 50 attempts, newest start first. Operational history is separate from observations and alert events.</p>
        {error ? <p role="alert">Could not load collection attempts.</p> : items === null ? <p>Loading collection attempts…</p> : items.length === 0 ? <p>No recorded collection attempts.</p> :
            <ul>{items.map(a => <li key={a.id}>
                <strong>{a.trigger} · {a.outcome.replaceAll('_', ' ')}</strong>
                <p>Started: {a.started_at} · Finished: {a.finished_at}</p>
                {a.error_summary && <p>Error: {a.error_summary.replaceAll('_', ' ')}</p>}
                {a.observation_id && <p>Observation reference: {a.observation_id}</p>}
            </li>)}</ul>}
    </section>;
}
