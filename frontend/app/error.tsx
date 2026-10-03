"use client";
import Link from "next/link";
export default function ErrorPage({ reset }: {
    reset: () => void;
}) { return <main><h1>Could not load this page</h1><p role="alert">Check that the backend and database are running, then try again.</p><button onClick={reset}>Try again</button><p><Link href="/products">Back to Products</Link></p></main>; }
