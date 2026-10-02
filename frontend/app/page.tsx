import Link from 'next/link';
import { api } from '../lib/api';
import { parseHome } from '../lib/home';
import { HomeView } from './home-view';
export const dynamic = 'force-dynamic';
export default async function Home() {
    let data;
    try { data = await api('/overview',parseHome); }
    catch { return <main><h1>Tracker overview</h1><p role="alert">Overview unavailable. Check the backend and reload.</p><Link href="/products">Browse Products</Link> · <Link href="/alerts">Alerts</Link> · <Link href="/collection">Collection</Link> · <Link href="/price-changes">Price Changes</Link></main>; }
    return <HomeView data={data}/>;
}
