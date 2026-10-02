export type CollectionAttempt = {
    id: string; listing_id: string; trigger: 'manual' | 'scheduled';
    started_at: string; finished_at: string;
    outcome: 'success' | 'provider_error' | 'persistence_error' | 'cancelled' | 'unavailable';
    observation_id?: string; error_summary?: string;
};
export function parseAttempts(value: unknown): CollectionAttempt[] {
    if (!Array.isArray(value) || value.length > 50) throw new Error('Invalid collection attempts');
    for (const a of value) {
        if (!a || typeof a !== 'object' || !['id', 'listing_id', 'started_at', 'finished_at'].every(k => typeof a[k] === 'string') ||
            !['manual', 'scheduled'].includes(a.trigger) || !['success', 'provider_error', 'persistence_error', 'cancelled', 'unavailable'].includes(a.outcome) ||
            (a.observation_id !== undefined && typeof a.observation_id !== 'string') || (a.error_summary !== undefined && typeof a.error_summary !== 'string')) throw new Error('Invalid collection attempt');
    }
    return value;
}
