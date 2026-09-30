import { parseObservation, type Currency, type Observation } from "./api.ts";
export type Money = { minor_units: number; currency: Currency };
export type Promotion = {
    id: string; listing_id: string; source: string; observed_at: string;
    starts_at?: string; ends_at?: string; kind: "fixed" | "percentage" | "cashback" | "membership";
    amount?: Money; basis_points?: number; requirement: "none" | "membership" | "other" | "unknown";
    stacking: "allowed" | "disallowed" | "unknown"; terms: string;
};
export type PromotionList = { promotions: Promotion[]; truncated: boolean };
export type Effective = {
    status: "available" | "conditional" | "unavailable"; reason: string;
    base: Money | null; immediate_discount: Money | null; immediate_payable: Money | null;
    potential_cashback: Money | null; potential_net: Money | null;
    observation: Observation | null; calculated_at: string;
    assumptions: string[]; exclusions: string[];
};
function object(value: unknown): Record<string, unknown> {
    if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("Invalid promotion response");
    return value as Record<string, unknown>;
}
function text(value: unknown) { if (typeof value !== "string") throw new Error("Invalid promotion text"); }
function money(value: unknown) {
    const m = object(value);
    if (!Number.isSafeInteger(m.minor_units) || (m.minor_units as number) < 0 || !["USD", "JPY"].includes(String(m.currency))) throw new Error("Invalid exact promotion money");
}
export function parsePromotions(value: unknown): PromotionList {
    const v = object(value);
    if (!Array.isArray(v.promotions) || typeof v.truncated !== "boolean") throw new Error("Invalid promotion list");
    for (const entry of v.promotions) {
        const p = object(entry);
        for (const key of ["id", "listing_id", "source", "observed_at", "terms"]) text(p[key]);
        for (const key of ["starts_at", "ends_at"]) if (p[key] !== undefined) text(p[key]);
        if (!["fixed", "percentage", "cashback", "membership"].includes(String(p.kind)) || !["none", "membership", "other", "unknown"].includes(String(p.requirement)) || !["allowed", "disallowed", "unknown"].includes(String(p.stacking))) throw new Error("Invalid promotion conditions");
        if (p.kind === "percentage") {
            if (!Number.isSafeInteger(p.basis_points) || (p.basis_points as number) < 1 || (p.basis_points as number) > 10000 || p.amount !== undefined) throw new Error("Invalid percentage");
        } else { money(p.amount); if (p.basis_points !== undefined) throw new Error("Unexpected percentage"); }
    }
    return v as PromotionList;
}
export function parseEffective(value: unknown): Effective {
    const v = object(value);
    if (!["available", "conditional", "unavailable"].includes(String(v.status))) throw new Error("Invalid result status");
    text(v.reason); text(v.calculated_at);
    if (v.observation !== null) parseObservation(v.observation);
    for (const key of ["base", "immediate_discount", "immediate_payable", "potential_cashback", "potential_net"]) {
        if (v[key] !== null) money(v[key]);
        if (key !== "base" && ((v.status === "available") === (v[key] === null))) throw new Error("Contradictory derived total");
    }
    for (const key of ["assumptions", "exclusions"]) { if (!Array.isArray(v[key])) throw new Error("Missing explanation"); (v[key] as unknown[]).forEach(text); }
    return v as Effective;
}
