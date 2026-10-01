export async function saveMetadata(kind: 'products' | 'retailers', id: string, form: FormData): Promise<void> {
    const fields = kind === 'products' ? ['name', 'brand', 'model'] : ['name'];
    const body = Object.fromEntries(fields.map(field => [field, String(form.get(field) ?? '')]));
    const response = await fetch(`/api/${kind}/${encodeURIComponent(id)}`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body), signal: AbortSignal.timeout(10000) });
    if (response.ok) return;
    if (response.status === 404) throw new Error('Record not found. Reload the catalog.');
    if (response.status === 409) throw new Error('Update conflicts with the existing catalog. Reload and retry.');
    if (response.status === 400) throw new Error('Invalid metadata. All descriptive fields must be text.');
    throw new Error('Could not save metadata. Try again.');
}
