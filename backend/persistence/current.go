package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
)

const MaxCurrentListings = 100

var ErrTooManyListings = errors.New("current prices: product has more than 100 listings")

type CurrentListing struct {
	Listing     domain.Listing
	Observation *domain.PriceObservation // nil means no history; stock-only is non-nil
}

// One statement selects whole rows, including stock-only observations. Reverse
// history order retains nanoseconds and breaks exact ties by greatest bytewise ID.
const currentSelect = `SELECT l.id,l.product_id,l.retailer_id,l.url,l.retailer_product_id,
 o.observed_at,o.observed_at_ns_remainder,o.source,o.stock,o.currency,
 o.msrp,o.retailer_list_price,o.sale_price,o.offer_price,o.msrp_source
 FROM listings l LEFT JOIN LATERAL (
 SELECT * FROM price_observations p WHERE listing_id=l.id
 AND NOT EXISTS (SELECT 1 FROM observation_invalidations i WHERE i.observation_id=p.result_id)
 ORDER BY observed_at DESC,observed_at_ns_remainder DESC,result_id COLLATE "C" DESC LIMIT 1
 ) o ON true `

func (s *Store) CurrentListing(ctx context.Context, id string) (CurrentListing, error) {
	if strings.TrimSpace(id) == "" {
		return CurrentListing{}, fmt.Errorf("current listing: ID required")
	}
	result, err := scanCurrent(s.db.QueryRowContext(ctx, currentSelect+`WHERE l.id=$1`, id))
	if err != nil {
		return CurrentListing{}, fmt.Errorf("current listing: %w", err)
	}
	return result, nil
}

// Reject oversized products instead of calculating a misleading minimum over
// a truncated subset. Empty results do not establish whether the product exists.
func (s *Store) CurrentProduct(ctx context.Context, id string) ([]CurrentListing, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("current product: ID required")
	}
	rows, err := s.db.QueryContext(ctx, currentSelect+`WHERE l.product_id=$1 ORDER BY l.id COLLATE "C" LIMIT $2`, id, MaxCurrentListings+1)
	if err != nil {
		return nil, fmt.Errorf("current product: %w", err)
	}
	defer rows.Close()
	results := make([]CurrentListing, 0)
	for rows.Next() {
		result, err := scanCurrent(rows)
		if err != nil {
			return nil, fmt.Errorf("current product: %w", err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("current product: %w", err)
	}
	if len(results) > MaxCurrentListings {
		return nil, ErrTooManyListings
	}
	return results, nil
}

func scanCurrent(row interface{ Scan(...any) error }) (CurrentListing, error) {
	var result CurrentListing
	l := &result.Listing
	var observed sql.NullTime
	var remainder sql.NullInt64
	var source, stock, currency, msrpSource sql.NullString
	var amounts [4]sql.NullInt64
	if err := row.Scan(&l.ID, &l.ProductID, &l.RetailerID, &l.URL, &l.RetailerProductID,
		&observed, &remainder, &source, &stock, &currency, &amounts[0], &amounts[1], &amounts[2], &amounts[3], &msrpSource); err != nil {
		return result, err
	}
	if err := l.Validate(); err != nil {
		return result, err
	}
	if !observed.Valid {
		return result, nil
	}
	input := domain.PriceObservationInput{ListingID: l.ID, ObservedAt: observed.Time.Add(time.Duration(remainder.Int64)), Source: source.String, Stock: domain.StockState(stock.String), MSRPSource: msrpSource.String}
	fields := []**domain.Money{&input.MSRP, &input.RetailerListPrice, &input.SalePrice, &input.OfferPrice}
	for i, amount := range amounts {
		if amount.Valid {
			*fields[i] = &domain.Money{MinorUnits: amount.Int64, Currency: domain.Currency(currency.String)}
		}
	}
	observation, err := domain.NewPriceObservation(input)
	if err != nil {
		return result, err
	}
	result.Observation = &observation
	return result, nil
}
