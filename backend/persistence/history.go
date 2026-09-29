package persistence

import (
	"context"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"slices"
	"strings"
	"time"
)

const MaxHistory = 1000

type History struct {
	Observations []domain.PriceObservation
	Truncated    bool
}

// PriceHistory selects the newest bounded window, returned oldest first. Bounds
// are inclusive UTC instants; nil from means all history up to the supplied time.
func (s *Store) PriceHistory(ctx context.Context, id string, from *time.Time, to time.Time) (History, error) {
	result := History{Observations: make([]domain.PriceObservation, 0)}
	if strings.TrimSpace(id) == "" || to.IsZero() || (from != nil && from.After(to)) {
		return result, fmt.Errorf("invalid history bounds")
	}
	lower := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	if from != nil {
		lower = *from
	}
	rows, err := s.db.QueryContext(ctx, `SELECT l.id,l.product_id,l.retailer_id,l.url,l.retailer_product_id,
 o.observed_at,o.observed_at_ns_remainder,o.source,o.stock,o.currency,
 o.msrp,o.retailer_list_price,o.sale_price,o.offer_price,o.msrp_source
 FROM price_observations o JOIN listings l ON l.id=o.listing_id
 WHERE o.listing_id=$1 AND (o.observed_at,o.observed_at_ns_remainder)>=($2,$3)
 AND (o.observed_at,o.observed_at_ns_remainder)<=($4,$5)
 ORDER BY o.observed_at DESC,o.observed_at_ns_remainder DESC,o.result_id COLLATE "C" DESC LIMIT $6`, id, lower.Truncate(time.Microsecond), lower.Nanosecond()%1000, to.Truncate(time.Microsecond), to.Nanosecond()%1000, MaxHistory+1)
	if err != nil {
		return result, fmt.Errorf("read history: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		value, err := scanCurrent(rows)
		if err != nil {
			return result, fmt.Errorf("read history: %w", err)
		}
		result.Observations = append(result.Observations, *value.Observation)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("read history: %w", err)
	}
	if len(result.Observations) > MaxHistory {
		result.Truncated = true
		result.Observations = result.Observations[:MaxHistory]
	}
	slices.Reverse(result.Observations)
	return result, nil
}
