export async function setProductArchived(id: string, archived: boolean): Promise<void> {
    const response = await fetch(`/api/products/${encodeURIComponent(id)}/archive`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ archived }), signal: AbortSignal.timeout(10000) });
    if (!response.ok) throw new Error(response.status === 404 ? 'Product not found.' : 'Could not change archive state. Try again.');
}
