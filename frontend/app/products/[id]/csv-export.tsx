"use client";
import { useRef, useState } from 'react';
import { downloadObservationCSV } from '../../../lib/csv-export';

export default function CSVExport({ listingID }: { listingID: string }) {
    const pending = useRef(false);
    const [busy, setBusy] = useState(false), [error, setError] = useState('');
    return <div><button disabled={busy} onClick={async () => {
        if (pending.current) return;
        pending.current = true; setBusy(true); setError('');
        try { await downloadObservationCSV(listingID); }
        catch (e) { setError(e instanceof Error ? e.message : 'CSV export failed.'); }
        finally { pending.current = false; setBusy(false); }
    }}>{busy ? 'Exporting…' : 'Export CSV'}</button>
        {error && <p role="alert">{error}</p>}
    </div>;
}
