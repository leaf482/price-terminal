package persistence

import (
	"context"
	"math/big"
	"strconv"
	"time"
)

const PriceChangeLimit = 100

type ChangePoint struct {
	ID         string    `json:"id"`
	ObservedAt time.Time `json:"observed_at"`
	MinorUnits string    `json:"minor_units"` // decimal string preserves all int64 values in JS
}
type PriceChange struct {
	ListingID    string      `json:"listing_id"`
	ProductID    string      `json:"product_id"`
	ProductName  string      `json:"product_name"`
	RetailerID   string      `json:"retailer_id"`
	RetailerName string      `json:"retailer_name"`
	Previous     ChangePoint `json:"previous"`
	Current      ChangePoint `json:"current"`
	Currency     string      `json:"currency"`
	ChangeMinor  string      `json:"change_minor"`
	Percentage   *string     `json:"percentage"`
	Direction    string      `json:"direction"`
}
type PriceChanges struct {
	Changes   []PriceChange `json:"changes"`
	Truncated bool          `json:"truncated"`
}

func changeValues(previous, current int64) (string, *string, string) {
	delta := new(big.Int).Sub(big.NewInt(current), big.NewInt(previous))
	direction := "unchanged"
	if delta.Sign() < 0 {
		direction = "decreased"
	} else if delta.Sign() > 0 {
		direction = "increased"
	}
	var percent *string
	if previous != 0 {
		n := new(big.Int).Mul(new(big.Int).Set(delta), big.NewInt(100))
		v := new(big.Rat).SetFrac(n, big.NewInt(previous)).FloatString(2)
		percent = &v
	}
	return delta.String(), percent, direction
}

// Comparable means valid executable prices in the same Listing and currency.
// Missing/stock-only points and other currencies are skipped, not interpolated.
// Windowing happens before LIMIT so predecessors outside the feed remain usable.
func (s *Store) RecentPriceChanges(ctx context.Context) (PriceChanges, error) {
	out := PriceChanges{Changes: []PriceChange{}}
	rows, err := s.db.QueryContext(ctx, `WITH points AS (
 SELECT o.result_id,o.listing_id,o.observed_at,o.observed_at_ns_remainder,o.currency,COALESCE(o.offer_price,o.sale_price) AS amount
 FROM price_observations o WHERE COALESCE(o.offer_price,o.sale_price) IS NOT NULL
 AND NOT EXISTS (SELECT 1 FROM observation_invalidations i WHERE i.observation_id=o.result_id)
 ), pairs AS (
 SELECT *,lag(result_id) OVER w AS previous_id,lag(observed_at) OVER w AS previous_at,
 lag(observed_at_ns_remainder) OVER w AS previous_ns,lag(amount) OVER w AS previous_amount
 FROM points WINDOW w AS (PARTITION BY listing_id,currency ORDER BY observed_at,observed_at_ns_remainder,result_id COLLATE "C")
 ) SELECT c.listing_id,p.id,p.name,r.id,r.name,c.previous_id,c.previous_at,c.previous_ns,c.previous_amount,
 c.result_id,c.observed_at,c.observed_at_ns_remainder,c.amount,c.currency
 FROM pairs c JOIN listings l ON l.id=c.listing_id JOIN products p ON p.id=l.product_id JOIN retailers r ON r.id=l.retailer_id
 WHERE c.previous_id IS NOT NULL ORDER BY c.observed_at DESC,c.observed_at_ns_remainder DESC,c.result_id COLLATE "C" DESC LIMIT $1`, PriceChangeLimit+1)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var c PriceChange
		var previous, current int64
		var pn, cn int
		if err := rows.Scan(&c.ListingID, &c.ProductID, &c.ProductName, &c.RetailerID, &c.RetailerName, &c.Previous.ID, &c.Previous.ObservedAt, &pn, &previous, &c.Current.ID, &c.Current.ObservedAt, &cn, &current, &c.Currency); err != nil {
			return out, err
		}
		c.Previous.ObservedAt = c.Previous.ObservedAt.UTC().Add(time.Duration(pn))
		c.Current.ObservedAt = c.Current.ObservedAt.UTC().Add(time.Duration(cn))
		c.Previous.MinorUnits = strconv.FormatInt(previous, 10)
		c.Current.MinorUnits = strconv.FormatInt(current, 10)
		c.ChangeMinor, c.Percentage, c.Direction = changeValues(previous, current)
		out.Changes = append(out.Changes, c)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Changes) > PriceChangeLimit {
		out.Truncated = true
		out.Changes = out.Changes[:PriceChangeLimit]
	}
	return out, nil
}
