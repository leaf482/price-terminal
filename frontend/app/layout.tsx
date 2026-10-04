import type { Metadata } from "next";
import type { ReactNode } from "react";
import "./globals.css";
import GlobalSearch from './global-search';
import Link from 'next/link';

export const metadata: Metadata = {
  title: "Product Price Tracker",
  description: "Track product prices over time.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body><a className="skip-link" href="#page-content">Skip to content</a><nav className="primary-nav" aria-label="Main navigation"><Link href="/">Home</Link><Link href="/products">Products</Link><a href="#catalog-search">Search</a><Link href="/alerts">Alerts</Link><span className="nav-secondary"><Link href="/catalog">Catalog</Link><Link href="/collection">Collection</Link><Link href="/price-changes">Price Changes</Link></span></nav><GlobalSearch/><div id="page-content" tabIndex={-1}>{children}</div></body>
    </html>
  );
}
