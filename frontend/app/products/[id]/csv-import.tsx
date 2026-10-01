"use client";
import { useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { csvHeader, importCSV } from '../../../lib/csv-import';

export default function CSVImport({ listingID }: { listingID: string }) {
    const router = useRouter(), pending = useRef(false), importID = useRef('');
    const [file, setFile] = useState<File | null>(null), [busy, setBusy] = useState(false);
    const [message, setMessage] = useState(''), [error, setError] = useState('');
    return <details><summary>Import CSV observations</summary>
        <p>Exact header (up to 500 rows / 1 MiB):</p><code>{csvHeader}</code>
        <p>Integer minor units, USD/JPY, RFC3339 timestamps with timezone. Blank prices are missing; 0 is explicit zero. Stock: unknown, in_stock or out_of_stock. MSRP requires evidence. All rows succeed or none do. No historical alert events are generated. Import remains available when tracking is disabled.</p>
        <form onSubmit={async e => {
            e.preventDefault(); if (pending.current) return;
            if (!file) { setError('Select a CSV file.'); return; }
            pending.current = true; setBusy(true); setError(''); setMessage('');
            try {
                importID.current ||= crypto.randomUUID();
                const count = await importCSV(listingID, file, importID.current);
                setMessage(`Imported ${count} rows. No retroactive alerts evaluated.`);
                setFile(null); router.refresh();
            } catch (e) { setError(e instanceof Error ? e.message : 'CSV import failed.'); }
            finally { pending.current = false; setBusy(false); }
        }}>
            <label>CSV file <input type="file" accept=".csv,text/csv" disabled={busy} onChange={e => { setFile(e.target.files?.[0] ?? null); importID.current = ''; setMessage(''); setError(''); }} /></label>
            <button disabled={busy || !file}>{busy ? 'Importing…' : 'Import observations'}</button>
        </form>
        {message && <p role="status">{message}</p>}
        {error && <pre role="alert" style={{ whiteSpace: 'pre-wrap' }}>{error}</pre>}
    </details>;
}
