import { jsonRequest } from './json-request.ts';
export async function setTracking(id: string, enabled: boolean) {
    const response = await jsonRequest(`/api/listings/${encodeURIComponent(id)}/tracking`, 'PATCH', { tracking_enabled: enabled });
    if (!response.ok) throw new Error(`Unable to change tracking (${response.status}).`);
}
