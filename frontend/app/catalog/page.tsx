import Link from 'next/link';
import CatalogManager from './manager';
export default function CatalogPage() { return <main><Link href="/">← Products and prices</Link><h1>Manage catalog</h1><p>Create Products and Retailers, then link them with Listings. No edit or delete operations.</p><CatalogManager /></main>; }
