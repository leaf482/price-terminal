package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const MaxDashboardProducts = 20

// CurrentProducts uses one query, capped at 101 Listings per requested Product.
// The extra row detects overflow without returning a misleading partial summary.
// currentSelect and scanCurrent preserve exactly the single-Product semantics.
func (s *Store) CurrentProducts(ctx context.Context, ids []string) (map[string][]CurrentListing, error) {
	if len(ids) > MaxDashboardProducts {
		return nil, fmt.Errorf("current products: at most %d product IDs", MaxDashboardProducts)
	}
	result := make(map[string][]CurrentListing, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("current products: product ID required")
		}
		result[id] = []CurrentListing{}
	}
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT snapshot.*
 FROM (SELECT DISTINCT unnest($1::text[]) AS product_id) requested
 CROSS JOIN LATERAL (`+currentSelect+`
 WHERE l.product_id=requested.product_id
 ORDER BY l.id COLLATE "C" LIMIT $2) snapshot
 ORDER BY snapshot.product_id COLLATE "C", snapshot.id COLLATE "C"`, ids, MaxCurrentListings+1)
	if err != nil {
		return nil, fmt.Errorf("current products: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		value, err := scanCurrent(rows)
		if err != nil {
			return nil, fmt.Errorf("current products: %w", err)
		}
		id := value.Listing.ProductID
		if len(result[id]) == MaxCurrentListings {
			return nil, ErrTooManyListings
		}
		result[id] = append(result[id], value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("current products: %w", err)
	}
	return result, nil
}

// ProductsWithRecentAlerts reads only a bounded set of product identities.
// Events remain historical evidence even if their observation is invalidated.
func (s *Store) ProductsWithRecentAlerts(ctx context.Context, ids []string, since, until time.Time) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(ids) > MaxProductListLimit {
		return nil, fmt.Errorf("too many products for alert summary")
	}
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT l.product_id FROM listings l
 JOIN price_alert_events e ON e.listing_id=l.id
 WHERE l.product_id=ANY($1) AND e.triggered_at >= $2 AND e.triggered_at <= $3`, ids, since, until)
	if err != nil {
		return nil, fmt.Errorf("dashboard alerts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
}
