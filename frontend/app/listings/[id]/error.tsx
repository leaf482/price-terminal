"use client";
export default function ErrorPage({ reset }: { reset: () => void }) { return <main><p role="alert">Could not load Listing detail. Check backend availability and retry.</p><button onClick={reset}>Retry</button></main>; }
