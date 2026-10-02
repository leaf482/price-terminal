import { parsePriceChanges, type PriceChange } from './price-changes.ts';
import { parseAlertOverview, type OverviewEvent } from './alert-overview.ts';
export const countLabels = { active_products: 'Active Products', archived_products: 'Archived Products', listings: 'Listings', tracking_enabled: 'Tracking enabled', tracking_disabled: 'Tracking disabled', collection_errors: 'Listings with collection errors', enabled_alerts: 'Enabled alerts', recently_triggered_alerts: 'Alerts triggered in last 7 days' };
export type HomeFailure = { id: string; listing_id: string; product_id: string; product_name: string; retailer_id: string; retailer_name: string; started_at: string; outcome: string; error_summary: string };
export type HomeOverview = { counts: Record<keyof typeof countLabels,number> | null; price_changes: PriceChange[] | null; alert_events: OverviewEvent[] | null; collection_failures: HomeFailure[] | null; errors: string[] };
export function parseHome(x: unknown): HomeOverview {
    if (!x || typeof x !== 'object') throw new Error('Invalid overview');
    const v=x as HomeOverview;
    if (!Array.isArray(v.errors) || !v.errors.every(e => typeof e === 'string')) throw new Error('Invalid overview errors');
    if (v.counts !== null && (!v.counts || !Object.keys(countLabels).every(k => Number.isSafeInteger(v.counts![k as keyof typeof countLabels]) && v.counts![k as keyof typeof countLabels] >= 0))) throw new Error('Invalid counts');
    for (const items of [v.price_changes,v.alert_events,v.collection_failures]) if (items !== null && (!Array.isArray(items) || items.length > 5)) throw new Error('Invalid overview bounds');
    if (v.price_changes) parsePriceChanges({changes:v.price_changes,truncated:false});
    if (v.alert_events) parseAlertOverview({alerts:[],events:v.alert_events,alerts_truncated:false,events_truncated:false});
    if (v.collection_failures) for (const row of v.collection_failures) if (!row || !['id','listing_id','product_id','product_name','retailer_id','retailer_name','started_at','outcome','error_summary'].every(k => typeof row[k as keyof HomeFailure] === 'string')) throw new Error('Invalid failure');
    return v;
}
