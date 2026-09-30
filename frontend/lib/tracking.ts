export async function setTracking(id: string, enabled: boolean) {
    const response = await fetch(`/api/listings/${encodeURIComponent(id)}/tracking`, {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ tracking_enabled: enabled }), signal: AbortSignal.timeout(10000),
    });
    if (!response.ok) throw new Error(`Unable to change tracking (${response.status}).`);
}
