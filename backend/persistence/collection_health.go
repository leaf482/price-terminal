package persistence

import (
	"context"
	"database/sql"
	"time"
)

const CollectionHealthLimit = 100

type CollectionHealthListing struct {
	ListingID       string     `json:"listing_id"`
	ProductID       string     `json:"product_id"`
	ProductName     string     `json:"product_name"`
	RetailerID      string     `json:"retailer_id"`
	RetailerName    string     `json:"retailer_name"`
	TrackingEnabled bool       `json:"tracking_enabled"`
	ObservedAt      *time.Time `json:"observed_at"`
	AttemptedAt     *time.Time `json:"attempted_at"`
	SuccessfulAt    *time.Time `json:"successful_at"`
	Outcome         string     `json:"outcome"`
	ErrorSummary    string     `json:"error_summary"`
}
type CollectionHealthCounts struct {
	Total    int `json:"total"`
	Healthy  int `json:"healthy"`
	Error    int `json:"error"`
	Never    int `json:"never"`
	Disabled int `json:"disabled"`
}
type CollectionHealth struct {
	Listings  []CollectionHealthListing `json:"listings"`
	Counts    CollectionHealthCounts    `json:"counts"`
	Truncated bool                      `json:"truncated"`
}

// One statement, independent of Listing count. Only timestamps are read from
// observations. Lateral reads use existing Listing/time indexes, not API calls.
func (s *Store) CollectionHealth(ctx context.Context) (CollectionHealth, error) {
	out := CollectionHealth{Listings: []CollectionHealthListing{}}
	rows, err := s.db.QueryContext(ctx, `SELECT l.id,p.id,p.name,r.id,r.name,l.tracking_enabled,
 o.observed_at,o.observed_at_ns_remainder,a.started_at,success.finished_at,COALESCE(a.outcome,''),COALESCE(a.error_summary,'')
 FROM (SELECT * FROM listings ORDER BY id COLLATE "C" LIMIT $1) l
 JOIN products p ON p.id=l.product_id JOIN retailers r ON r.id=l.retailer_id
 LEFT JOIN LATERAL (SELECT observed_at,observed_at_ns_remainder FROM price_observations o
 WHERE o.listing_id=l.id AND NOT EXISTS (SELECT 1 FROM observation_invalidations i WHERE i.observation_id=o.result_id)
 ORDER BY observed_at DESC,observed_at_ns_remainder DESC,result_id COLLATE "C" DESC LIMIT 1) o ON true
 LEFT JOIN LATERAL (SELECT started_at,outcome,error_summary FROM collection_attempts a WHERE a.listing_id=l.id
 ORDER BY started_at DESC,id COLLATE "C" DESC LIMIT 1) a ON true
 LEFT JOIN LATERAL (SELECT max(finished_at) AS finished_at FROM collection_attempts a WHERE a.listing_id=l.id AND outcome='success') success ON true
 ORDER BY l.id COLLATE "C"`, CollectionHealthLimit+1)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v CollectionHealthListing
		var observed, attempted, successful sql.NullTime
		var ns sql.NullInt64
		if err := rows.Scan(&v.ListingID, &v.ProductID, &v.ProductName, &v.RetailerID, &v.RetailerName, &v.TrackingEnabled, &observed, &ns, &attempted, &successful, &v.Outcome, &v.ErrorSummary); err != nil {
			return out, err
		}
		if observed.Valid {
			at := observed.Time.UTC().Add(time.Duration(ns.Int64))
			v.ObservedAt = &at
		}
		if attempted.Valid {
			at := attempted.Time.UTC()
			v.AttemptedAt = &at
		}
		if successful.Valid {
			at := successful.Time.UTC()
			v.SuccessfulAt = &at
		}
		out.Listings = append(out.Listings, v)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Listings) > CollectionHealthLimit {
		out.Truncated = true
		out.Listings = out.Listings[:CollectionHealthLimit]
	}
	out.Counts = collectionHealthCounts(out.Listings)
	return out, nil
}

// Counts partition the shown set. Healthy means enabled with latest recorded
// attempt successful; it makes no claim about observation freshness.
func collectionHealthCounts(items []CollectionHealthListing) CollectionHealthCounts {
	c := CollectionHealthCounts{Total: len(items)}
	for _, v := range items {
		switch {
		case !v.TrackingEnabled:
			c.Disabled++
		case v.AttemptedAt == nil:
			c.Never++
		case v.Outcome == "success":
			c.Healthy++
		default:
			c.Error++
		}
	}
	return c
}
