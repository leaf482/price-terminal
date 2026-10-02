import Link from "next/link";
import { api } from "../../lib/api";
import { parseDashboard } from "../../lib/dashboard";
import Dashboard from "../dashboard";
export const dynamic = "force-dynamic";
export default async function Home({ searchParams }: { searchParams: Promise<{ include_archived?: string }> }) {
    const includeArchived = (await searchParams).include_archived === 'true';
    const data = await api(`/dashboard?include_archived=${includeArchived}`, parseDashboard);
    return <main><header><p className="eyebrow">PRICE TERMINAL</p><h1>Your tracked products</h1><Link href="/catalog">Manage catalog</Link><p>Observed prices, with their history and context.</p></header>
 <Dashboard data={data} includeArchived={includeArchived}/></main>;
}
