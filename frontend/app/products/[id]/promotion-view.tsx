"use client";
import { useEffect, useState } from "react";
import { api } from "../../../lib/api";
import { formatPrice } from "../../../lib/prices";
import { parsePromotions, parseEffective, type PromotionList, type Effective } from "../../../lib/promotions";

export default function PromotionView({ listingID }: { listingID: string }) {
    const [data, setData] = useState<PromotionList | null>(null), [error, setError] = useState(false);
    const [ids, setIDs] = useState<string[]>([]), [member, setMember] = useState("unknown"), [eligible, setEligible] = useState("unknown"), [query, setQuery] = useState("");
    useEffect(() => {
        let active = true;
        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), 10000);
        api(`/listings/${encodeURIComponent(listingID)}/promotions`, parsePromotions, controller.signal)
            .then(value => { if (active) { setData(value); setError(false); } })
            .catch(() => { if (active) setError(true); }).finally(() => clearTimeout(timer));
        return () => { active = false; controller.abort(); clearTimeout(timer); };
    }, [listingID]);
    return <section><h4>Promotion evidence — separate from observed prices</h4>
        {error ? <p role="alert">Could not load promotion evidence.</p> : !data ? <p role="status">Loading promotions…</p> : !data.promotions.length ? <p>No relevant promotion evidence.</p> : <>
            {data.truncated && <p>Showing only the newest 100 relevant evidence records.</p>}
            <p>Select up to one immediate discount and one cashback offer. Evidence records are not automatically combined or deduplicated into offers.</p>
            {data.promotions.map(p => <div key={p.id}><label><input type="checkbox" checked={ids.includes(p.id)} disabled={!ids.includes(p.id) && ids.length === 2} onChange={e => { setIDs(e.target.checked ? [...ids, p.id] : ids.filter(id => id !== p.id)); setQuery(""); }}/>{p.kind} · {p.amount ? formatPrice(p.amount.minor_units, p.amount.currency) : `${p.basis_points} basis points (100 = 1%)`} · {p.id}</label>
                <p>{p.terms}</p><p className="muted">Requirement: {p.requirement} · Stacking: {p.stacking}<br/>Observed: {p.observed_at} · Source: {p.source}<br/>Starts: {p.starts_at || "not specified"} · Ends: {p.ends_at || "not specified"}</p></div>)}
            <label>Membership assumption <select value={member} onChange={e => { setMember(e.target.value); setQuery(""); }}>{["unknown", "yes", "no"].map(v => <option key={v}>{v}</option>)}</select></label><br/>
            <label>All other documented conditions satisfied <select value={eligible} onChange={e => { setEligible(e.target.value); setQuery(""); }}>{["unknown", "yes", "no"].map(v => <option key={v}>{v}</option>)}</select></label>
            <p><button disabled={!ids.length} onClick={() => { const q = new URLSearchParams({ scenario: "selected", member, eligible }); ids.forEach(id => q.append("promotion_id", id)); setQuery(q.toString()); }}>Calculate selected scenario</button></p>
            {query && <EffectiveResult key={query} listingID={listingID} query={query}/>}
        </>}
    </section>;
}
function EffectiveResult({ listingID, query }: { listingID: string; query: string }) {
    const [data, setData] = useState<Effective | null>(null), [error, setError] = useState(false);
    useEffect(() => {
        let active = true;
        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), 10000);
        api(`/listings/${encodeURIComponent(listingID)}/effective-price?${query}`, parseEffective, controller.signal)
            .then(value => { if (active) { setData(value); setError(false); } })
            .catch(() => { if (active) setError(true); }).finally(() => clearTimeout(timer));
        return () => { active = false; controller.abort(); clearTimeout(timer); };
    }, [listingID, query]);
    if (error) return <p role="alert">Could not calculate this scenario.</p>;
    if (!data) return <p role="status">Calculating scenario…</p>;
    return <EffectiveSummary data={data}/>;
}
export function EffectiveSummary({ data }: { data: Effective }) {
    return <div className="panel"><h4>Derived EffectivePrice — {data.status}</h4><p>{data.reason.replaceAll("_", " ")}</p>
        {data.base && <p>Base observed price: {formatPrice(data.base.minor_units, data.base.currency)}</p>}
        <p className="muted">Observed: {data.observation?.observed_at || "unavailable"} · Stock: {data.observation?.stock || "unknown"}<br/>Calculated: {data.calculated_at}</p>
        {data.status === "available" ? <>
            <p>Selected immediate discount: {formatPrice(data.immediate_discount!.minor_units, data.immediate_discount!.currency)}</p>
            <p>Derived immediate payable: {formatPrice(data.immediate_payable!.minor_units, data.immediate_payable!.currency)}</p>
            <p>Possible cashback: {formatPrice(data.potential_cashback!.minor_units, data.potential_cashback!.currency)}</p>
            <p>Potential net after cashback: {formatPrice(data.potential_net!.minor_units, data.potential_net!.currency)}</p>
            <p>Scenario estimate only. Conditional savings and cashback are not guaranteed.</p>
        </> : <p>Derived totals are withheld until the selected evidence and eligibility support calculation.</p>}
        <ul>{data.assumptions.map(text => <li key={text}>{text}</li>)}</ul><p>Excluded: {data.exclusions.join(", ")}</p>
    </div>;
}
