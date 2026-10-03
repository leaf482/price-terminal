package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
)

// InsertPriceObservation appends a snapshot. resultID identifies one collected
// result globally: callers must reuse it on retries and assign a new ID for each
// independent collection. A duplicate returns a wrapped PostgreSQL uniqueness
// error, even if its payload differs; it never overwrites or silently accepts it.
func (s *Store) InsertPriceObservation(ctx context.Context, resultID string, observation domain.PriceObservation) error {
	return insertPriceObservation(ctx, s.db, resultID, observation)
}

// Collection writes serialize with tracking changes. Explicit manual writes do
// not use this guard. A committed disable prevents subsequent collection inserts.
func (s *Store) InsertCollectedObservation(ctx context.Context, resultID string, observation domain.PriceObservation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var enabled bool
	if err := tx.QueryRowContext(ctx, `SELECT tracking_enabled FROM listings WHERE id=$1 FOR SHARE`, observation.ListingID()).Scan(&enabled); err != nil {
		return err
	}
	if !enabled {
		return domain.ErrTrackingDisabled
	}
	if err := insertPriceObservation(ctx, tx, resultID, observation); err != nil {
		return err
	}
	return tx.Commit()
}

func insertPriceObservation(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, resultID string, observation domain.PriceObservation) error {
	if strings.TrimSpace(resultID) == "" {
		return fmt.Errorf("insert observation: result ID is required")
	}
	if err := observation.Validate(); err != nil {
		return fmt.Errorf("insert observation: %w", err)
	}
	var currency any
	if value, present := observation.Currency(); present {
		currency = string(value)
	}
	_, err := db.ExecContext(ctx, `INSERT INTO price_observations
		(result_id, listing_id, observed_at, observed_at_ns_remainder, source, stock,
		 currency, msrp, retailer_list_price, sale_price, offer_price, msrp_source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		resultID, observation.ListingID(), observation.ObservedAt().Truncate(time.Microsecond),
		observation.ObservedAt().Nanosecond()%1000, observation.Source(), string(observation.Stock()), currency,
		priceAmount(observation.MSRP()), priceAmount(observation.RetailerListPrice()),
		priceAmount(observation.SalePrice()), priceAmount(observation.OfferPrice()), observation.MSRPSource())
	if err != nil {
		return fmt.Errorf("insert observation: %w", err)
	}
	return nil
}

func priceAmount(money domain.Money, present bool) any {
	if !present {
		return nil
	}
	return money.MinorUnits
}

// ListPriceObservations returns one listing's history oldest first. Exact time
// ties use result ID in bytewise C collation, independent of insertion order.
// No observations returns an empty slice. Bounded/time-range queries are deferred
// to the historical API task; this method performs no updates or deletions.
func (s *Store) ListPriceObservations(ctx context.Context, listingID string) ([]domain.PriceObservation, error) {
	if strings.TrimSpace(listingID) == "" {
		return nil, fmt.Errorf("list observations: listing ID is required")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT listing_id, observed_at, observed_at_ns_remainder,
		source, stock, currency, msrp, retailer_list_price, sale_price, offer_price, msrp_source
		FROM price_observations WHERE listing_id = $1
		ORDER BY observed_at, observed_at_ns_remainder, result_id COLLATE "C"`, listingID)
	if err != nil {
		return nil, fmt.Errorf("list observations: %w", err)
	}
	defer rows.Close()
	observations := make([]domain.PriceObservation, 0)
	for rows.Next() {
		var input domain.PriceObservationInput
		var remainder int
		var currency sql.NullString
		var amounts [4]sql.NullInt64
		if err := rows.Scan(&input.ListingID, &input.ObservedAt, &remainder, &input.Source, &input.Stock,
			&currency, &amounts[0], &amounts[1], &amounts[2], &amounts[3], &input.MSRPSource); err != nil {
			return nil, fmt.Errorf("list observations: %w", err)
		}
		input.ObservedAt = input.ObservedAt.Add(time.Duration(remainder))
		observation, err := observationFromAmounts(input, currency, amounts)
		if err != nil {
			return nil, fmt.Errorf("list observations: %w", err)
		}
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list observations: %w", err)
	}
	return observations, nil
}
