package domain

import (
	"fmt"
	"math/big"
)

type PriceAlert struct {
	ID              string   `json:"id"`
	ListingID       string   `json:"listing_id"`
	Kind            string   `json:"kind"` // target, drop, historical_low
	Currency        Currency `json:"currency"`
	Threshold       *int64   `json:"threshold_minor_units,omitempty"`
	DropBasisPoints *int64   `json:"drop_basis_points,omitempty"`
	Enabled         bool     `json:"enabled"`
	RequireInStock  bool     `json:"require_in_stock"`
}

func (a PriceAlert) Validate() error {
	if err := requireText("alert ID", a.ID); err != nil {
		return err
	}
	if err := requireText("listing ID", a.ListingID); err != nil {
		return err
	}
	if err := a.Currency.Validate(); err != nil {
		return err
	}
	switch a.Kind {
	case "target":
		if a.Threshold == nil || *a.Threshold < 0 || a.DropBasisPoints != nil {
			return fmt.Errorf("target requires a nonnegative threshold only")
		}
	case "drop":
		if a.Threshold != nil || a.DropBasisPoints == nil || *a.DropBasisPoints < 1 || *a.DropBasisPoints > 10000 {
			return fmt.Errorf("drop requires 1..10000 basis points only")
		}
	case "historical_low":
		if a.Threshold != nil || a.DropBasisPoints != nil {
			return fmt.Errorf("historical low has no numeric threshold")
		}
	default:
		return fmt.Errorf("invalid alert kind")
	}
	return nil
}

// AlertPrice uses only observed offer/sale values, never promotions or reference prices.
func AlertPrice(o PriceObservation) (Money, bool) {
	if m, ok := o.OfferPrice(); ok {
		return m, true
	}
	return o.SalePrice()
}

// Matches compares caller-supplied earlier comparable values; nil means none.
// Percentage comparisons use exact cross multiplication, without rounding.
func (a PriceAlert) Matches(o PriceObservation, previous, low *Money) (bool, error) {
	if err := a.Validate(); err != nil {
		return false, err
	}
	if err := o.Validate(); err != nil {
		return false, err
	}
	m, ok := AlertPrice(o)
	if !a.Enabled || a.ListingID != o.ListingID() || !ok || m.Currency != a.Currency || (a.RequireInStock && o.Stock() != StockInStock) {
		return false, nil
	}
	comparable := func(p *Money) bool { return p != nil && p.Validate() == nil && p.Currency == m.Currency }
	switch a.Kind {
	case "target":
		return m.MinorUnits <= *a.Threshold, nil
	case "drop":
		if !comparable(previous) || previous.MinorUnits == 0 || m.MinorUnits >= previous.MinorUnits {
			return false, nil
		}
		change := new(big.Int).Mul(big.NewInt(previous.MinorUnits-m.MinorUnits), big.NewInt(10000))
		required := new(big.Int).Mul(big.NewInt(previous.MinorUnits), big.NewInt(*a.DropBasisPoints))
		return change.Cmp(required) >= 0, nil
	case "historical_low":
		return comparable(low) && m.MinorUnits < low.MinorUnits, nil
	}
	return false, nil
}
