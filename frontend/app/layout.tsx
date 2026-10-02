import type { Metadata } from "next";
import type { ReactNode } from "react";
import "./globals.css";
import GlobalSearch from './global-search';

export const metadata: Metadata = {
  title: "Product Price Tracker",
  description: "Track product prices over time.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body><GlobalSearch/>{children}</body>
    </html>
  );
}
