export type BulkResult = { listing_id: string; outcome: 'success' | 'failure' | 'unavailable' | 'cancelled'; error_summary?: string; observation_id?: string };
export async function refreshSelected(ids: string[]): Promise<BulkResult[]> {
    if (!ids.length || ids.length > 20) throw new Error('Select 1–20 Listings.');
    // The server bounds each attempt. Do not apply the ordinary short read
    // timeout to a sequential batch; a transport error can have partial writes.
    const response = await fetch('/api/collection/refresh', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ listing_ids: ids }) });
    if (!response.ok) throw new Error('Could not complete batch request. Reload health before retrying; some attempts may have completed.');
    const { data } = await response.json();
    if (!Array.isArray(data) || data.length > 20 || data.some(r => !r || typeof r.listing_id !== 'string' || !['success','failure','unavailable','cancelled'].includes(r.outcome) || (r.error_summary !== undefined && typeof r.error_summary !== 'string') || (r.observation_id !== undefined && typeof r.observation_id !== 'string'))) throw new Error('Invalid batch response. Reload health before retrying.');
    return data;
}
