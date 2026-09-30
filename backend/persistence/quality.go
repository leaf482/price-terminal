package persistence

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/leaf482/price-terminal/backend/domain"
)

var ErrInvalidInvalidation = errors.New("observation ID and a reason of 1..1000 characters are required")

type ObservationAudit struct {
	ID            string
	Observation   domain.PriceObservation
	Reason        string
	InvalidatedAt *time.Time
}

const auditSelect = `SELECT o.result_id,i.reason,i.invalidated_at,l.id,l.product_id,l.retailer_id,l.url,l.retailer_product_id,NOT l.tracking_enabled,o.observed_at,o.observed_at_ns_remainder,o.source,o.stock,o.currency,o.msrp,o.retailer_list_price,o.sale_price,o.offer_price,o.msrp_source FROM price_observations o JOIN listings l ON l.id=o.listing_id LEFT JOIN observation_invalidations i ON i.observation_id=o.result_id `

// Adapt the leading audit columns to the existing immutable-fact scanner.
type auditScanner struct {
	row    interface{ Scan(...any) error }
	id     *string
	reason *sql.NullString
	at     *sql.NullTime
}

func (s auditScanner) Scan(dest ...any) error {
	return s.row.Scan(append([]any{s.id, s.reason, s.at}, dest...)...)
}
func scanAudit(row interface{ Scan(...any) error }) (ObservationAudit, error) {
	var a ObservationAudit
	var reason sql.NullString
	var at sql.NullTime
	value, err := scanCurrent(auditScanner{row, &a.ID, &reason, &at})
	if err != nil {
		return a, err
	}
	a.Observation = *value.Observation
	a.Reason = reason.String
	if at.Valid {
		v := at.Time.UTC()
		a.InvalidatedAt = &v
	}
	return a, nil
}
func (s *Store) GetObservationAudit(ctx context.Context, id string) (ObservationAudit, error) {
	return scanAudit(s.db.QueryRowContext(ctx, auditSelect+`WHERE o.result_id=$1`, id))
}
func (s *Store) ListObservationAudit(ctx context.Context, listing string) ([]ObservationAudit, bool, error) {
	rows, err := s.db.QueryContext(ctx, auditSelect+`WHERE o.listing_id=$1 ORDER BY o.observed_at DESC,o.observed_at_ns_remainder DESC,o.result_id COLLATE "C" DESC LIMIT 101`, listing)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	values := make([]ObservationAudit, 0)
	for rows.Next() {
		a, err := scanAudit(rows)
		if err != nil {
			return nil, false, err
		}
		values = append(values, a)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(values) > 100
	if more {
		values = values[:100]
	}
	return values, more, nil
}

// Invalidation is append-only and first-write-wins, including its reason/time.
// The Listing lock serializes invalidation with alert evaluation for all of its
// observations, including observations used as earlier comparison evidence.
func (s *Store) InvalidateObservation(ctx context.Context, id, reason string) (ObservationAudit, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(reason) == "" || utf8.RuneCountInString(reason) > 1000 {
		return ObservationAudit{}, ErrInvalidInvalidation
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ObservationAudit{}, err
	}
	defer tx.Rollback()
	var listing string
	if err = tx.QueryRowContext(ctx, `SELECT l.id FROM listings l JOIN price_observations o ON o.listing_id=l.id WHERE o.result_id=$1 FOR UPDATE OF l`, id).Scan(&listing); err != nil {
		return ObservationAudit{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO observation_invalidations(observation_id,reason) VALUES ($1,$2) ON CONFLICT (observation_id) DO NOTHING`, id, reason); err != nil {
		return ObservationAudit{}, err
	}
	if err = tx.Commit(); err != nil {
		return ObservationAudit{}, err
	}
	return s.GetObservationAudit(ctx, id)
}
