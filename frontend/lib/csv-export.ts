export async function downloadObservationCSV(listingID: string): Promise<void> {
    const response = await fetch(`/api/listings/${encodeURIComponent(listingID)}/observations/export`, { signal: AbortSignal.timeout(20000), cache: 'no-store' });
    if (!response.ok) {
        const payload = await response.json().catch(() => null);
        throw new Error(payload?.error?.message || `CSV export failed (${response.status}).`);
    }
    const url = URL.createObjectURL(await response.blob());
    const link = document.createElement('a');
    link.href = url; link.download = 'listing-observations.csv';
    document.body.appendChild(link);
    try { link.click(); } finally { link.remove(); setTimeout(() => URL.revokeObjectURL(url), 1000); }
}
