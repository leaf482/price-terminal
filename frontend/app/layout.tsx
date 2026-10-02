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
      <body><nav aria-label="Main navigation"><Link href="/">Products</Link> · <Link href="/catalog">Catalog</Link> · <Link href="/alerts">Alerts</Link></nav><GlobalSearch/>{children}</body>
    </html>
  );
}
