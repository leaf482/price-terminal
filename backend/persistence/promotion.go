package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"strings"
	"time"
)

const promotionColumns = `id,listing_id,source,observed_at,observed_ns,starts_at,starts_ns,ends_at,ends_ns,kind,amount,currency,basis_points,requirement,stacking,terms`
const MaxPromotions = 100

func (s *Store) InsertPromotion(ctx context.Context, p domain.Promotion) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("insert promotion: %w", err)
	}
	split := func(t *time.Time) (any, any) {
		if t == nil {
			return nil, nil
		}
		return t.Truncate(time.Microsecond), t.Nanosecond() % 1000
	}
	start, startNS := split(p.StartsAt)
	end, endNS := split(p.EndsAt)
	var amount, currency any
	if p.Amount != nil {
		amount = p.Amount.MinorUnits
		currency = string(p.Amount.Currency)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO promotions (`+promotionColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, p.ID, p.ListingID, p.Source, p.ObservedAt.Truncate(time.Microsecond), p.ObservedAt.Nanosecond()%1000, start, startNS, end, endNS, p.Kind, amount, currency, p.BasisPoints, p.Requirement, p.Stacking, p.Terms)
	if err != nil {
		return fmt.Errorf("insert promotion: %w", err)
	}
	return nil
}
func scanPromotion(row interface{ Scan(...any) error }) (domain.Promotion, error) {
	var p domain.Promotion
	var observedNS int
	var start, end sql.NullTime
	var startNS, endNS, amount, bps sql.NullInt64
	var currency sql.NullString
	err := row.Scan(&p.ID, &p.ListingID, &p.Source, &p.ObservedAt, &observedNS, &start, &startNS, &end, &endNS, &p.Kind, &amount, &currency, &bps, &p.Requirement, &p.Stacking, &p.Terms)
	if err != nil {
		return p, err
	}
	p.ObservedAt = p.ObservedAt.UTC().Add(time.Duration(observedNS))
	if start.Valid {
		v := start.Time.UTC().Add(time.Duration(startNS.Int64))
		p.StartsAt = &v
	}
	if end.Valid {
		v := end.Time.UTC().Add(time.Duration(endNS.Int64))
		p.EndsAt = &v
	}
	if amount.Valid {
		p.Amount = &domain.Money{MinorUnits: amount.Int64, Currency: domain.Currency(currency.String)}
	}
	if bps.Valid {
		p.BasisPoints = &bps.Int64
	}
	return p, p.Validate()
}
func (s *Store) GetPromotion(ctx context.Context, listingID, id string) (domain.Promotion, error) {
	p, err := scanPromotion(s.db.QueryRowContext(ctx, `SELECT `+promotionColumns+` FROM promotions WHERE listing_id=$1 AND id=$2`, listingID, id))
	if err != nil {
		return p, fmt.Errorf("get promotion: %w", err)
	}
	return p, nil
}

// Relevant means observed by now and not explicitly expired. Upcoming evidence
// remains visible with its start time. Missing bounds never imply verified validity.
func (s *Store) RelevantPromotions(ctx context.Context, listingID string, at time.Time) ([]domain.Promotion, bool, error) {
	if strings.TrimSpace(listingID) == "" || at.IsZero() {
		return nil, false, fmt.Errorf("invalid promotion query")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+promotionColumns+` FROM promotions WHERE listing_id=$1 AND (observed_at,observed_ns)<=($2,$3) AND (ends_at IS NULL OR (ends_at,ends_ns)>($2,$3)) ORDER BY observed_at DESC,observed_ns DESC,id COLLATE "C" DESC LIMIT $4`, listingID, at.Truncate(time.Microsecond), at.Nanosecond()%1000, MaxPromotions+1)
	if err != nil {
		return nil, false, fmt.Errorf("list promotions: %w", err)
	}
	defer rows.Close()
	result := make([]domain.Promotion, 0)
	for rows.Next() {
		p, err := scanPromotion(rows)
		if err != nil {
			return nil, false, err
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(result) > MaxPromotions
	if more {
		result = result[:MaxPromotions]
	}
	return result, more, nil
}
