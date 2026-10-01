export const csvHeader = 'observed_at,currency,offer_price_minor,sale_price_minor,list_price_minor,msrp_minor,msrp_source,stock';
export async function importCSV(listingID: string, file: Blob, importID: string): Promise<number> {
    if (file.size > 1048576) throw new Error('CSV must be at most 1 MiB.');
    const response = await fetch(`/api/listings/${encodeURIComponent(listingID)}/observations/import`, {
        method: 'POST', headers: { 'Content-Type': 'text/csv', 'X-Import-ID': importID }, body: file, signal: AbortSignal.timeout(20000),
    });
    const payload = await response.json().catch(() => null);
    if (!response.ok) {
        const rows = payload?.error?.rows;
        if (Array.isArray(rows)) throw new Error(rows.map(row => `Row ${row.row}: ${row.message}`).join('\n'));
        throw new Error(payload?.error?.message || `CSV import failed (${response.status}).`);
    }
    if (!Number.isSafeInteger(payload?.data?.imported) || payload.data.imported < 1) throw new Error('Invalid import response; check history before retrying.');
    return payload.data.imported;
}
