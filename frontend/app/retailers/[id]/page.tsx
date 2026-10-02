import Link from 'next/link';
import { api } from '../../../lib/api';
import { parseRetailerOverview } from '../../../lib/retailer-overview';
import Overview from './overview';
export default async function RetailerPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = await params;
    const data = await api(`/retailers/${encodeURIComponent(id)}/overview`, parseRetailerOverview);
    return <main><Link href="/catalog">← Manage catalog</Link><h1>{data.retailer.name || data.retailer.id}</h1><p>Retailer ID: {data.retailer.id}</p><Overview data={data}/></main>;
}
