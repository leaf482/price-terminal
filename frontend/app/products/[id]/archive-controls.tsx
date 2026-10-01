"use client";
import { useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { setProductArchived } from '../../../lib/archive';
export default function ArchiveControls({ id, archived }: { id: string; archived: boolean }) {
    const router = useRouter(), pending = useRef(false);
    const [busy, setBusy] = useState(false), [error, setError] = useState('');
    return <section><p>Product: {archived ? 'Archived' : 'Active'}</p>
        <p>Archiving hides this Product from the default dashboard. Listing tracking and collection continue unchanged.</p>
        <button disabled={busy} onClick={async () => {
            if (pending.current) return;
            pending.current = true; setBusy(true); setError('');
            try { await setProductArchived(id, !archived); router.refresh(); }
            catch (e) { setError(e instanceof Error ? e.message : 'Could not change archive state.'); }
            finally { pending.current = false; setBusy(false); }
        }}>{busy ? 'Saving…' : archived ? 'Unarchive' : 'Archive'}</button>
        {error && <p role="alert">{error}</p>}
    </section>;
}
