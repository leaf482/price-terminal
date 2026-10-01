"use client";
import { useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { saveMetadata } from '../../lib/metadata';

export default function MetadataEditor({ kind, record, onSaved }: {
    kind: 'products' | 'retailers'; record: { id: string; name: string; brand?: string; model?: string }; onSaved?: () => void;
}) {
    const router = useRouter(), pending = useRef(false);
    const [busy, setBusy] = useState(false), [message, setMessage] = useState(''), [error, setError] = useState('');
    const fields = kind === 'products' ? ['name', 'brand', 'model'] as const : ['name'] as const;
    return <details><summary>Edit {kind === 'products' ? 'Product metadata' : 'Retailer name'}</summary>
        <p>ID (read-only): {record.id}. Blank descriptive fields are allowed.</p>
        <form onSubmit={async e => {
            e.preventDefault(); if (pending.current) return;
            const form = new FormData(e.currentTarget);
            pending.current = true; setBusy(true); setError(''); setMessage('');
            try { await saveMetadata(kind, record.id, form); setMessage('Metadata saved.'); onSaved?.(); router.refresh(); }
            catch (e) { setError(e instanceof Error ? e.message : 'Could not save metadata.'); }
            finally { pending.current = false; setBusy(false); }
        }}>
            {fields.map(field => <p key={field}><label>{field}<input name={field} defaultValue={record[field] ?? ''} disabled={busy}/></label></p>)}
            <button disabled={busy}>{busy ? 'Saving…' : 'Save metadata'}</button>
            {message && <p role="status">{message}</p>}{error && <p role="alert">{error}</p>}
        </form></details>;
}
