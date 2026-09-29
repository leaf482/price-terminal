package main

import (
	"context"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"math/big"
	"net/http"
	"time"
)

type historyStore interface {
	GetListing(context.Context, string) (domain.Listing, error)
	PriceHistory(context.Context, string, *time.Time, time.Time) (persistence.History, error)
}

func historyStart(rangeName string, now time.Time) (*time.Time, error) {
	days := map[string]int{"1D": 1, "1W": 7, "1M": 30, "3M": 90, "1Y": 365}
	if rangeName == "ALL" {
		return nil, nil
	}
	n, ok := days[rangeName]
	if !ok {
		return nil, fmt.Errorf("invalid range")
	}
	start := now.Add(-time.Duration(n) * 24 * time.Hour)
	return &start, nil
}

type historyLow struct {
	MinorUnits int64           `json:"minor_units"`
	Currency   domain.Currency `json:"currency"`
}

// Metadata describes observed item prices, irrespective of historical stock.
// Missing points are not filled. Mixed currencies or truncation suppress both
// summaries; change requires priced first/last rows, distinct times and nonzero start.
func historyMetadata(rows []domain.PriceObservation, truncated bool) (*historyLow, *string) {
	if truncated {
		return nil, nil
	}
	var low *historyLow
	for _, o := range rows {
		m, ok := observedBasis(o)
		if !ok {
			continue
		}
		if low != nil && low.Currency != m.Currency {
			return nil, nil
		}
		if low == nil || m.MinorUnits < low.MinorUnits {
			low = &historyLow{m.MinorUnits, m.Currency}
		}
	}
	if len(rows) < 2 {
		return low, nil
	}
	first, last := rows[0], rows[len(rows)-1]
	a, okA := observedBasis(first)
	b, okB := observedBasis(last)
	if !okA || !okB || a.Currency != b.Currency || a.MinorUnits == 0 || !last.ObservedAt().After(first.ObservedAt()) {
		return low, nil
	}
	delta := new(big.Int).Sub(big.NewInt(b.MinorUnits), big.NewInt(a.MinorUnits))
	delta.Mul(delta, big.NewInt(100))
	percent := new(big.Rat).SetFrac(delta, big.NewInt(a.MinorUnits)).FloatString(2)
	return low, &percent
}

func observedBasis(o domain.PriceObservation) (domain.Money, bool) {
	if p, ok := o.OfferPrice(); ok {
		return p, true
	}
	return o.SalePrice()
}

func historyHandler(store historyStore, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := "1M"
		values, provided := r.URL.Query()["range"]
		if provided && len(values) == 1 {
			name = values[0]
		}
		end := now().UTC()
		start, err := historyStart(name, end)
		if err != nil || (provided && len(values) != 1) {
			productError(w, 400, "invalid_range", "range must be 1D, 1W, 1M, 3M, 1Y, or ALL")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		id := r.PathValue("id")
		if _, err := store.GetListing(ctx, id); err != nil {
			catalogReadError(w, err, "listing")
			return
		}
		history, err := store.PriceHistory(ctx, id, start, end)
		if err != nil {
			productError(w, 500, "internal_error", "unable to read history")
			return
		}
		data := make([]*observationJSON, 0, len(history.Observations))
		for _, o := range history.Observations {
			data = append(data, observationResponse(o))
		}
		low, change := historyMetadata(history.Observations, history.Truncated)
		writeJSON(w, 200, map[string]any{"data": map[string]any{"listing_id": id, "range": name, "from": start, "to": end, "observations": data, "limit": persistence.MaxHistory, "truncated": history.Truncated, "price_basis": "offer_price_else_sale_price", "historical_low": low, "period_change_percent": change}})
	}
}
