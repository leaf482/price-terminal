package persistence

import (
	"context"
	"fmt"
	"time"
)

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
