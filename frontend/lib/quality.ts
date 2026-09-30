import { parseObservation, type Observation } from './api.ts';
export type Audit = {
    id: string;
    listing_id: string;
    observation: Observation;
    valid: boolean;
    invalidation_reason: string;
    invalidated_at: string | null;
};
export type AuditList = {
    observations: Audit[];
    truncated: boolean;
};
export function parseAudit(value: unknown): Audit {
    if (!value || typeof value !== 'object')
        throw new Error('Invalid audit record');
    const a = value as Audit;
    if (typeof a.id !== 'string' || typeof a.listing_id !== 'string' || typeof a.valid !== 'boolean' || typeof a.invalidation_reason !== 'string' || (a.valid ? a.invalidated_at !== null : typeof a.invalidated_at !== 'string'))
        throw new Error('Invalid audit state');
    parseObservation(a.observation);
    return a;
}
export function parseAuditList(value: unknown): AuditList { if (!value || typeof value !== 'object')
    throw new Error('Invalid audit list'); const v = value as AuditList; if (!Array.isArray(v.observations) || typeof v.truncated !== 'boolean')
    throw new Error('Invalid audit list'); v.observations.forEach(parseAudit); return v; }
export async function invalidateObservation(id: string, reason: string): Promise<void> {
    if (!reason.trim() || [...reason].length > 1000)
        throw new Error('Provide a reason of 1–1000 characters.');
    const r = await fetch(`/api/observations/${encodeURIComponent(id)}/invalidate`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reason }), signal: AbortSignal.timeout(10000) });
    if (!r.ok)
        throw new Error(r.status === 404 ? 'Observation not found.' : 'Could not invalidate observation. Check the reason and backend.');
}
