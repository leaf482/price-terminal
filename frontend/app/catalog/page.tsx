import Link from 'next/link';
import CatalogManager from './manager';
export default function CatalogPage() { return <main><Link href="/">← Products and prices</Link><h1>Manage catalog</h1><p>Create Products and Retailers, then link them with Listings. Product descriptions and Retailer names can be edited. Listing source identity is read-only.</p><CatalogManager /></main>; }
