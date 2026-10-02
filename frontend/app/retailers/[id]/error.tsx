"use client";
export default function ErrorPage({ reset }: { reset: () => void }) { return <main><p role="alert">Could not load this Retailer. Check its ID and backend availability.</p><button onClick={reset}>Retry</button></main>; }
