package persistence

import (
	"context"
	"time"
)

type HomeCounts struct {
	ActiveProducts          int64 `json:"active_products"`
	ArchivedProducts        int64 `json:"archived_products"`
	Listings                int64 `json:"listings"`
	TrackingEnabled         int64 `json:"tracking_enabled"`
	TrackingDisabled        int64 `json:"tracking_disabled"`
	CollectionErrors        int64 `json:"collection_errors"`
	EnabledAlerts           int64 `json:"enabled_alerts"`
	RecentlyTriggeredAlerts int64 `json:"recently_triggered_alerts"`
}

func (s *Store) HomeCounts(ctx context.Context, now time.Time) (HomeCounts, error) {
	var c HomeCounts
	err := s.db.QueryRowContext(ctx, `SELECT
 (SELECT count(*) FROM products WHERE NOT archived),(SELECT count(*) FROM products WHERE archived),
 (SELECT count(*) FROM listings),(SELECT count(*) FROM listings WHERE tracking_enabled),(SELECT count(*) FROM listings WHERE NOT tracking_enabled),
 (SELECT count(*) FROM listings l JOIN LATERAL (SELECT outcome FROM collection_attempts a WHERE a.listing_id=l.id ORDER BY started_at DESC,id COLLATE "C" DESC LIMIT 1) a ON true WHERE l.tracking_enabled AND a.outcome<>'success'),
 (SELECT count(*) FROM price_alerts WHERE enabled),
 (SELECT count(DISTINCT alert_id) FROM price_alert_events WHERE triggered_at >= $1 AND triggered_at <= $2)`, now.Add(-7*24*time.Hour), now).Scan(&c.ActiveProducts, &c.ArchivedProducts, &c.Listings, &c.TrackingEnabled, &c.TrackingDisabled, &c.CollectionErrors, &c.EnabledAlerts, &c.RecentlyTriggeredAlerts)
	return c, err
}

type HomeFailure struct {
	ID        string `json:"id"`
	ListingID string `json:"listing_id"`
	AlertContext
	StartedAt    time.Time `json:"started_at"`
	Outcome      string    `json:"outcome"`
	ErrorSummary string    `json:"error_summary"`
}

func (s *Store) RecentCollectionFailures(ctx context.Context) ([]HomeFailure, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,a.listing_id,p.id,p.name,r.id,r.name,a.started_at,a.outcome,a.error_summary
 FROM (SELECT * FROM collection_attempts WHERE outcome<>'success' ORDER BY started_at DESC,id COLLATE "C" DESC LIMIT 5) a
 JOIN listings l ON l.id=a.listing_id JOIN products p ON p.id=l.product_id JOIN retailers r ON r.id=l.retailer_id ORDER BY a.started_at DESC,a.id COLLATE "C" DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HomeFailure{}
	for rows.Next() {
		var v HomeFailure
		if err := rows.Scan(&v.ID, &v.ListingID, &v.ProductID, &v.ProductName, &v.RetailerID, &v.RetailerName, &v.StartedAt, &v.Outcome, &v.ErrorSummary); err != nil {
			return nil, err
		}
		v.StartedAt = v.StartedAt.UTC()
		out = append(out, v)
	}
	return out, rows.Err()
}
