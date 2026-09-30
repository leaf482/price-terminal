package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
)

const MaxAlerts = 100

var ErrAlertLimit = errors.New("at most 100 alerts per listing")

const alertColumns = `id,listing_id,kind,currency,threshold_minor_units,drop_basis_points,enabled,require_in_stock`

func scanAlert(row interface{ Scan(...any) error }) (domain.PriceAlert, error) {
	var a domain.PriceAlert
	var threshold, drop sql.NullInt64
	if err := row.Scan(&a.ID, &a.ListingID, &a.Kind, &a.Currency, &threshold, &drop, &a.Enabled, &a.RequireInStock); err != nil {
		return a, err
	}
	if threshold.Valid {
		a.Threshold = &threshold.Int64
	}
	if drop.Valid {
		a.DropBasisPoints = &drop.Int64
	}
	return a, a.Validate()
}
func (s *Store) InsertAlert(ctx context.Context, a domain.PriceAlert) error {
	if err := a.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM listings WHERE id=$1 FOR UPDATE`, a.ListingID).Scan(&id); err != nil {
		return fmt.Errorf("alert listing: %w", err)
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM price_alerts WHERE listing_id=$1`, a.ListingID).Scan(&count); err != nil {
		return err
	}
	if count >= MaxAlerts {
		return ErrAlertLimit
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO price_alerts (`+alertColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, a.ID, a.ListingID, a.Kind, a.Currency, a.Threshold, a.DropBasisPoints, a.Enabled, a.RequireInStock); err != nil {
		return fmt.Errorf("insert alert: %w", err)
	}
	return tx.Commit()
}
func (s *Store) ListAlerts(ctx context.Context, listing string) ([]domain.PriceAlert, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+alertColumns+` FROM price_alerts WHERE listing_id=$1 ORDER BY id COLLATE "C" LIMIT $2`, listing, MaxAlerts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.PriceAlert, 0)
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
func (s *Store) SetAlertEnabled(ctx context.Context, listing, id string, enabled bool) (domain.PriceAlert, error) {
	return scanAlert(s.db.QueryRowContext(ctx, `UPDATE price_alerts SET enabled=$3 WHERE listing_id=$1 AND id=$2 RETURNING `+alertColumns, listing, id, enabled))
}

type AlertEvent struct {
	AlertID              string          `json:"alert_id"`
	ListingID            string          `json:"listing_id"`
	ObservationID        string          `json:"observation_id"`
	Kind                 string          `json:"kind"`
	TriggeredAt          time.Time       `json:"triggered_at"`
	ObservedAt           time.Time       `json:"observed_at"`
	MinorUnits           int64           `json:"minor_units"`
	Currency             domain.Currency `json:"currency"`
	PriceBasis           string          `json:"price_basis"`
	ComparisonMinorUnits *int64          `json:"comparison_minor_units"`
}

func (s *Store) ListAlertEvents(ctx context.Context, listing string) ([]AlertEvent, bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT e.alert_id,e.listing_id,e.observation_id,e.kind,e.triggered_at,o.observed_at,o.observed_at_ns_remainder,e.minor_units,e.currency,e.price_basis,e.comparison_minor_units FROM price_alert_events e JOIN price_observations o ON o.result_id=e.observation_id WHERE e.listing_id=$1 ORDER BY e.triggered_at DESC,e.alert_id COLLATE "C",e.observation_id COLLATE "C" LIMIT 101`, listing)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	result := make([]AlertEvent, 0)
	for rows.Next() {
		var e AlertEvent
		var ns int
		var comparison sql.NullInt64
		if err := rows.Scan(&e.AlertID, &e.ListingID, &e.ObservationID, &e.Kind, &e.TriggeredAt, &e.ObservedAt, &ns, &e.MinorUnits, &e.Currency, &e.PriceBasis, &comparison); err != nil {
			return nil, false, err
		}
		e.ObservedAt = e.ObservedAt.UTC().Add(time.Duration(ns))
		e.TriggeredAt = e.TriggeredAt.UTC()
		if comparison.Valid {
			e.ComparisonMinorUnits = &comparison.Int64
		}
		result = append(result, e)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(result) > 100
	if more {
		result = result[:100]
	}
	return result, more, nil
}

// EvaluateAlerts uses a separate transaction after observation persistence.
// Earlier means (observed time, nanosecond remainder, bytewise result ID),
// including deterministic equal-time ties. No promotion inputs are consulted.
func (s *Store) EvaluateAlerts(ctx context.Context, resultID string) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialize against invalidation before taking comparison snapshots. At READ
	// COMMITTED, a waiter sees invalidations committed by the preceding holder.
	var listing string
	if err = tx.QueryRowContext(ctx, `SELECT l.id FROM listings l JOIN price_observations o ON o.listing_id=l.id WHERE o.result_id=$1 FOR UPDATE OF l`, resultID).Scan(&listing); err != nil {
		return err
	}
	var invalid bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM observation_invalidations WHERE observation_id=$1)`, resultID).Scan(&invalid); err != nil {
		return err
	}
	if invalid {
		return nil
	}
	current, err := scanCurrent(tx.QueryRowContext(ctx, `SELECT l.id,l.product_id,l.retailer_id,l.url,l.retailer_product_id,NOT l.tracking_enabled,o.observed_at,o.observed_at_ns_remainder,o.source,o.stock,o.currency,o.msrp,o.retailer_list_price,o.sale_price,o.offer_price,o.msrp_source FROM price_observations o JOIN listings l ON l.id=o.listing_id WHERE o.result_id=$1`, resultID))
	if err != nil {
		return err
	}
	o := *current.Observation
	money, ok := domain.AlertPrice(o)
	now := time.Now().UTC()
	if !ok || o.ObservedAt().After(now) {
		return nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+alertColumns+` FROM price_alerts WHERE listing_id=$1 AND enabled ORDER BY id COLLATE "C" LIMIT $2 FOR SHARE`, o.ListingID(), MaxAlerts)
	if err != nil {
		return err
	}
	alerts := make([]domain.PriceAlert, 0)
	for rows.Next() {
		a, e := scanAlert(rows)
		if e != nil {
			rows.Close()
			return e
		}
		alerts = append(alerts, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, a := range alerts {
		if a.Currency != money.Currency || (a.RequireInStock && o.Stock() != domain.StockInStock) {
			continue
		}
		var prior *domain.Money
		if a.Kind != "target" {
			query := `SELECT COALESCE(offer_price,sale_price) FROM price_observations p WHERE listing_id=$1 AND currency=$2 AND COALESCE(offer_price,sale_price) IS NOT NULL AND NOT EXISTS (SELECT 1 FROM observation_invalidations i WHERE i.observation_id=p.result_id) AND (NOT $3 OR stock='in_stock') AND (observed_at,observed_at_ns_remainder,result_id COLLATE "C")<($4,$5,$6 COLLATE "C")`
			if a.Kind == "drop" {
				query += ` ORDER BY observed_at DESC,observed_at_ns_remainder DESC,result_id COLLATE "C" DESC LIMIT 1`
			} else {
				query += ` ORDER BY COALESCE(offer_price,sale_price) LIMIT 1`
			}
			var amount int64
			e := tx.QueryRowContext(ctx, query, o.ListingID(), a.Currency, a.RequireInStock, o.ObservedAt().Truncate(time.Microsecond), o.ObservedAt().Nanosecond()%1000, resultID).Scan(&amount)
			if e != nil && !errors.Is(e, sql.ErrNoRows) {
				return e
			}
			if e == nil {
				prior = &domain.Money{MinorUnits: amount, Currency: a.Currency}
			}
		}
		hit, e := a.Matches(o, prior, prior)
		if e != nil {
			return e
		}
		if !hit {
			continue
		}
		basis := "offer_price"
		if _, ok := o.OfferPrice(); !ok {
			basis = "sale_price"
		}
		var comparison any
		if prior != nil {
			comparison = prior.MinorUnits
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO price_alert_events (alert_id,listing_id,observation_id,kind,triggered_at,minor_units,currency,price_basis,comparison_minor_units) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (alert_id,observation_id) DO NOTHING`, a.ID, a.ListingID, resultID, a.Kind, now, money.MinorUnits, money.Currency, basis, comparison); err != nil {
			return err
		}
	}
	return tx.Commit()
}
