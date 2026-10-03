import { jsonRequest } from './json-request.ts';
export async function setProductArchived(id: string, archived: boolean): Promise<void> {
    const response = await jsonRequest(`/api/products/${encodeURIComponent(id)}/archive`, 'PATCH', { archived });
    if (!response.ok) throw new Error(response.status === 404 ? 'Product not found.' : 'Could not change archive state. Try again.');
}
