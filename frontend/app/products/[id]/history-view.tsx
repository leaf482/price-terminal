"use client";
import { useEffect, useState } from "react";
import { api, parseHistory, type History } from "../../../lib/api";
import { formatPrice, historyPoints, historySegments } from "../../../lib/prices";
export default function HistoryView({ listings }: {
    listings: {
        id: string;
        retailer: string;
    }[];
}) {
    const [id, setID] = useState(listings[0]?.id || ""), [range, setRange] = useState("1M");
    if (!listings.length)
        return null;
    return <section className="history"><h2>Price history</h2><div className="controls"><label>Listing <select value={id} onChange={e => setID(e.target.value)}>{listings.map(l => <option key={l.id} value={l.id}>{l.retailer} / {l.id}</option>)}</select></label><label>Range <select value={range} onChange={e => setRange(e.target.value)}>{["1D", "1W", "1M", "3M", "1Y", "ALL"].map(r => <option key={r}>{r}</option>)}</select></label></div><HistoryResult key={`${id}:${range}`} id={id} range={range}/></section>;
}
function HistoryResult({ id, range }: {
    id: string;
    range: string;
}) {
    const [data, setData] = useState<History | null>(null), [error, setError] = useState(false);
    useEffect(() => {
        let active = true;
        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), 10000);
        api(`/listings/${encodeURIComponent(id)}/history?range=${range}`, parseHistory, controller.signal)
            .then(history => {
                if (!active) return;
                setData(history);
                setError(false);
            })
            .catch(() => {
                // A timeout is still an error while this effect is active.
                if (active) setError(true);
            })
            .finally(() => clearTimeout(timer));
        return () => {
            active = false;
            controller.abort();
            clearTimeout(timer);
        };
    }, [id, range]);
    if (error)
        return <p role="alert">History could not be loaded. Select another range or reload to retry.</p>;
    if (!data)
        return <p role="status">Loading history…</p>;
    if (!data.observations.length)
        return <p>No observations in this range.</p>;
    const points = historyPoints(data.observations), priced = points.filter(p => p.price), currencies = [...new Set(priced.map(p => p.price!.currency))];
    return <div className="panel"><p>Basis: offer price, otherwise sale price. Reference prices are excluded.</p>{data.truncated && <p role="status">Showing the newest 1,000 observations only. Narrow the range; period summaries are unavailable.</p>}<div className="controls"><p>Historical low: <strong>{data.historical_low ? formatPrice(data.historical_low.minor_units, data.historical_low.currency) : "Unavailable"}</strong></p><p>Period change: <strong>{data.period_change_percent !== null ? `${data.period_change_percent}%` : "Unavailable"}</strong></p></div>
 {!priced.length ? <p>Stock-only or reference-only history: no observed offer/sale prices to plot.</p> : currencies.map(currency => { const selected = priced.filter(p => p.price!.currency === currency), amounts = selected.map(p => p.price!.amount), min = Math.min(...amounts), max = Math.max(...amounts), start = points[0].time, end = points[points.length - 1].time; return <figure key={currency}><figcaption>{currency} · observed offer / sale prices</figcaption><svg viewBox="0 0 800 240" role="img" aria-label={`${currency} observed prices; unobserved intervals are left blank`}><line x1="80" y1="200" x2="780" y2="200" stroke="currentColor"/><text x="0" y="25">{formatPrice(max, currency)}</text><text x="0" y="195">{formatPrice(min, currency)}</text>{historySegments(data.observations).filter(segment => segment[0].price!.currency === currency).map((segment, i) => <polyline key={`line-${i}`} fill="none" stroke="#2563eb" strokeWidth="1.5" points={segment.map(point => `${start === end ? 430 : 80 + (point.time - start) / (end - start) * 700},${min === max ? 110 : 190 - (point.price!.amount - min) / (max - min) * 160}`).join(" ")}/>)}{selected.map((point, i) => { const x = start === end ? 430 : 80 + (point.time - start) / (end - start) * 700, y = min === max ? 110 : 190 - (point.price!.amount - min) / (max - min) * 160; return <g key={i}><line x1={x - 3} x2={x + 3} y1={y} y2={y} stroke="#2563eb" strokeWidth="2"/><circle cx={x} cy={y} r="3" fill="#2563eb"><title>{point.timestamp} · {point.price!.label}: {formatPrice(point.price!.amount, currency)}</title></circle></g>; })}</svg><div className="axis"><span>{points[0].timestamp}</span><span>{points[points.length - 1].timestamp}</span></div></figure>; })}
 <p className="muted">Each mark is an actual observation. Lines only guide the eye between adjacent prices within 15 minutes. Missing prices, changes in price basis, and longer gaps break the line; no data points are filled or interpolated. Currencies are plotted separately.</p><details><summary>Observation values ({points.length})</summary><ol>{points.map((p, i) => <li key={i}>{p.timestamp} · {p.price ? `${formatPrice(p.price.amount, p.price.currency)} (${p.price.label})` : "Price unavailable"} · {data.observations[i].stock.replaceAll("_", " ")}</li>)}</ol></details></div>;
}
